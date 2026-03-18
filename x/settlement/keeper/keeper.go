package keeper

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

// Keeper manages the settlement module's state.
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
	authority  string // module authority address for governance
}

// NewKeeper creates a new settlement Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
	}
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

// GetParams returns the settlement module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the settlement module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// Escrow CRUD
// ---------------------------------------------------------------------------

// GetEscrow retrieves an escrow by ID.
func (k Keeper) GetEscrow(ctx sdk.Context, escrowID string) (types.Escrow, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EscrowKey(escrowID))
	if bz == nil {
		return types.Escrow{}, false
	}
	var escrow types.Escrow
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return types.Escrow{}, false
	}
	return escrow, true
}

// SetEscrow persists an escrow to the store.
func (k Keeper) SetEscrow(ctx sdk.Context, escrow types.Escrow) error {
	bz, err := json.Marshal(escrow)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.EscrowKey(escrow.ID), bz)
	return nil
}

// setEscrowIndex creates a secondary index entry for creator -> escrow.
func (k Keeper) setEscrowIndex(ctx sdk.Context, creator, escrowID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.EscrowByCreatorKey(creator, escrowID), []byte(escrowID))
}

// GetEscrowsByCreator returns all escrows created by a given address.
func (k Keeper) GetEscrowsByCreator(ctx sdk.Context, creator string) []types.Escrow {
	store := ctx.KVStore(k.storeKey)
	prefix := types.EscrowByCreatorIteratorPrefix(creator)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var escrows []types.Escrow
	for ; iter.Valid(); iter.Next() {
		escrowID := string(iter.Value())
		escrow, found := k.GetEscrow(ctx, escrowID)
		if found {
			escrows = append(escrows, escrow)
		}
	}
	return escrows
}

// generateEscrowID creates a unique escrow ID.
func (k Keeper) generateEscrowID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EscrowCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.EscrowCounterKey, newBz)

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("ESC_%s%s", hex.EncodeToString(newBz), hex.EncodeToString(randBytes))
}

// ---------------------------------------------------------------------------
// Escrow Business Logic
// ---------------------------------------------------------------------------

// CreateEscrow validates and creates a new escrow, locking funds from the creator
// into the settlement module account. The timeoutSeconds parameter overrides the
// default if non-zero.
//
// This method signature matches the SettlementKeeper interface expected by the
// capability module.
func (k Keeper) CreateEscrow(ctx sdk.Context, creator, provider string, amount sdk.Coin, timeoutSeconds uint64) (string, error) {
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(provider); err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid provider: %s", err)
	}
	if !amount.IsValid() || amount.IsZero() {
		return "", types.ErrInsufficientFunds.Wrap("amount must be positive")
	}

	// Send coins from creator to module account.
	coins := sdk.NewCoins(amount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, coins); err != nil {
		return "", types.ErrInsufficientFunds.Wrapf("failed to lock funds: %s", err)
	}

	params := k.GetParams(ctx)
	timeout := params.EscrowTimeoutSeconds
	if timeoutSeconds > 0 {
		timeout = timeoutSeconds
	}

	now := ctx.BlockTime()
	expiresAt := now.Add(time.Duration(timeout) * time.Second)
	escrowID := k.generateEscrowID(ctx)

	escrow := types.Escrow{
		ID:        escrowID,
		Creator:   creator,
		Provider:  provider,
		Amount:    amount,
		Status:    types.EscrowStatusLocked,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := k.SetEscrow(ctx, escrow); err != nil {
		return "", err
	}
	k.setEscrowIndex(ctx, creator, escrowID)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"escrow_created",
		sdk.NewAttribute("escrow_id", escrowID),
		sdk.NewAttribute("creator", creator),
		sdk.NewAttribute("provider", provider),
		sdk.NewAttribute("amount", amount.String()),
	))

	return escrowID, nil
}

// ReleaseEscrow releases escrowed funds to the provider.
// Fee split: provider gets 95%, protocol treasury (fee collector) gets 5%.
//
// This method signature matches the SettlementKeeper interface expected by the
// capability module.
func (k Keeper) ReleaseEscrow(ctx sdk.Context, escrowID string, releaser string) error {
	escrow, found := k.GetEscrow(ctx, escrowID)
	if !found {
		return types.ErrEscrowNotFound.Wrapf("escrow %s not found", escrowID)
	}
	if escrow.Status != types.EscrowStatusLocked {
		return types.ErrInvalidStatus.Wrapf("escrow %s is %s, not LOCKED", escrowID, escrow.Status)
	}

	// Either the creator or the provider can authorize release.
	if escrow.Creator != releaser && escrow.Provider != releaser {
		return types.ErrUnauthorized.Wrapf("only the escrow creator or provider can release")
	}

	providerAddr, err := sdk.AccAddressFromBech32(escrow.Provider)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid provider: %s", err)
	}

	// Calculate fee split per spec: protocol_fee = amount * 500 / 10000 (5%).
	totalAmount := escrow.Amount.Amount
	protocolFee := totalAmount.Mul(math.NewInt(500)).Quo(math.NewInt(10000))
	providerAmount := totalAmount.Sub(protocolFee)

	providerCoin := sdk.NewCoin(escrow.Amount.Denom, providerAmount)
	protocolCoin := sdk.NewCoin(escrow.Amount.Denom, protocolFee)

	// Send provider share from module to provider.
	if providerAmount.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, providerAddr, sdk.NewCoins(providerCoin),
		); err != nil {
			return fmt.Errorf("failed to send to provider: %w", err)
		}
	}

	// Send protocol fee to fee collector (protocol treasury).
	if protocolFee.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx, types.ModuleName, "fee_collector", sdk.NewCoins(protocolCoin),
		); err != nil {
			return fmt.Errorf("failed to send protocol fee: %w", err)
		}
	}

	escrow.Status = types.EscrowStatusReleased
	if err := k.SetEscrow(ctx, escrow); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"escrow_released",
		sdk.NewAttribute("escrow_id", escrowID),
		sdk.NewAttribute("provider_amount", providerCoin.String()),
		sdk.NewAttribute("protocol_fee", protocolCoin.String()),
	))

	return nil
}

// RefundEscrow refunds escrowed funds back to the consumer (creator).
//
// This method signature matches the SettlementKeeper interface expected by the
// capability module.
func (k Keeper) RefundEscrow(ctx sdk.Context, escrowID string, refunder string) error {
	escrow, found := k.GetEscrow(ctx, escrowID)
	if !found {
		return types.ErrEscrowNotFound.Wrapf("escrow %s not found", escrowID)
	}
	if escrow.Status != types.EscrowStatusLocked {
		return types.ErrInvalidStatus.Wrapf("escrow %s is %s, not LOCKED", escrowID, escrow.Status)
	}

	// Either the creator or the provider can authorize refund.
	if escrow.Creator != refunder && escrow.Provider != refunder {
		return types.ErrUnauthorized.Wrapf("only the escrow creator or provider can refund")
	}

	creatorAddr, err := sdk.AccAddressFromBech32(escrow.Creator)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}

	// Full refund to consumer.
	coins := sdk.NewCoins(escrow.Amount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, coins); err != nil {
		return fmt.Errorf("failed to refund: %w", err)
	}

	escrow.Status = types.EscrowStatusRefunded
	if err := k.SetEscrow(ctx, escrow); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"escrow_refunded",
		sdk.NewAttribute("escrow_id", escrowID),
		sdk.NewAttribute("creator", escrow.Creator),
		sdk.NewAttribute("amount", escrow.Amount.String()),
	))

	return nil
}

// ExpireStaleEscrows iterates all escrows and auto-refunds those that have expired.
// This is intended to be called from EndBlock.
func (k Keeper) ExpireStaleEscrows(ctx sdk.Context) error {
	now := ctx.BlockTime()
	store := ctx.KVStore(k.storeKey)

	iter := storetypes.KVStorePrefixIterator(store, types.EscrowKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var escrow types.Escrow
		if err := json.Unmarshal(iter.Value(), &escrow); err != nil {
			continue
		}
		if escrow.Status != types.EscrowStatusLocked {
			continue
		}
		if now.Before(escrow.ExpiresAt) {
			continue
		}

		// Escrow has expired -- refund the creator.
		creatorAddr, err := sdk.AccAddressFromBech32(escrow.Creator)
		if err != nil {
			continue
		}
		coins := sdk.NewCoins(escrow.Amount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, coins); err != nil {
			// Log and continue; do not halt the chain.
			continue
		}

		escrow.Status = types.EscrowStatusExpired
		bz, err := json.Marshal(escrow)
		if err != nil {
			continue
		}
		store.Set(types.EscrowKey(escrow.ID), bz)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"escrow_expired",
			sdk.NewAttribute("escrow_id", escrow.ID),
			sdk.NewAttribute("creator", escrow.Creator),
			sdk.NewAttribute("amount", escrow.Amount.String()),
		))
	}

	return nil
}

// IterateAllEscrows iterates over all escrows and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllEscrows(ctx sdk.Context, cb func(escrow types.Escrow) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.EscrowKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var escrow types.Escrow
		if err := json.Unmarshal(iter.Value(), &escrow); err != nil {
			continue
		}
		if cb(escrow) {
			break
		}
	}
}
