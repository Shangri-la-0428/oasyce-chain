package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the settlement module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	bz, err := k.cdc.Marshal(&params)
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
	if err := k.cdc.Unmarshal(bz, &escrow); err != nil {
		return types.Escrow{}, false
	}
	return escrow, true
}

// SetEscrow persists an escrow to the store.
func (k Keeper) SetEscrow(ctx sdk.Context, escrow types.Escrow) error {
	bz, err := k.cdc.Marshal(&escrow)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.EscrowKey(escrow.Id), bz)
	return nil
}

// setEscrowIndex creates a secondary index entry for creator -> escrow.
func (k Keeper) setEscrowIndex(ctx sdk.Context, creator, escrowID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.EscrowByCreatorKey(creator, escrowID), []byte(escrowID))
}

// deleteEscrowIndex removes a secondary index entry for creator -> escrow.
func (k Keeper) deleteEscrowIndex(ctx sdk.Context, creator, escrowID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.EscrowByCreatorKey(creator, escrowID))
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

// generateEscrowID creates a unique deterministic escrow ID.
// Uses counter + block hash for determinism across validators.
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

	// Deterministic: hash counter + block header for uniqueness.
	h := sha256.Sum256(append(newBz, ctx.HeaderHash()...))
	return fmt.Sprintf("ESC_%s", hex.EncodeToString(h[:8]))
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
		Id:        escrowID,
		Creator:   creator,
		Provider:  provider,
		Amount:    amount,
		Status:    types.ESCROW_STATUS_LOCKED,
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
// Fee split: 93% creator/provider, 3% validator, 2% burn, 2% treasury.
//
// This method signature matches the SettlementKeeper interface expected by the
// capability module.
func (k Keeper) ReleaseEscrow(ctx sdk.Context, escrowID string, releaser string) error {
	escrow, found := k.GetEscrow(ctx, escrowID)
	if !found {
		return types.ErrEscrowNotFound.Wrapf("escrow %s not found", escrowID)
	}
	if escrow.Status != types.ESCROW_STATUS_LOCKED {
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

	// Calculate fee split per spec (90/5/2/3):
	//   creator/provider = 90% (remaining after fees)
	//   protocol_fee     = 5% (to fee_collector for validator rewards)
	//   burn             = 2% (permanently destroyed)
	//   treasury         = 3% (to fee_collector for protocol treasury)
	totalAmount := escrow.Amount.Amount
	protocolFeeAmt := totalAmount.Mul(math.NewInt(500)).Quo(math.NewInt(10000))  // 5%
	burnAmount := totalAmount.Mul(math.NewInt(200)).Quo(math.NewInt(10000))      // 2%
	treasuryAmount := totalAmount.Mul(math.NewInt(300)).Quo(math.NewInt(10000))  // 3%
	providerAmount := totalAmount.Sub(protocolFeeAmt).Sub(burnAmount).Sub(treasuryAmount) // 90%

	// Send provider/creator share from module to provider.
	if providerAmount.IsPositive() {
		providerCoin := sdk.NewCoin(escrow.Amount.Denom, providerAmount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, providerAddr, sdk.NewCoins(providerCoin),
		); err != nil {
			return fmt.Errorf("failed to send to provider: %w", err)
		}
	}

	// Send protocol fee + treasury to fee collector (distributed to validators).
	protocolFee := protocolFeeAmt.Add(treasuryAmount) // combined for fee_collector
	if protocolFee.IsPositive() {
		protocolCoin := sdk.NewCoin(escrow.Amount.Denom, protocolFee)
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx, types.ModuleName, "fee_collector", sdk.NewCoins(protocolCoin),
		); err != nil {
			return fmt.Errorf("failed to send protocol fee: %w", err)
		}
	}

	// Burn 2% — permanently removed from supply.
	if burnAmount.IsPositive() {
		burnCoin := sdk.NewCoin(escrow.Amount.Denom, burnAmount)
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
			return fmt.Errorf("failed to burn tokens: %w", err)
		}
	}

	escrow.Status = types.ESCROW_STATUS_RELEASED
	if err := k.SetEscrow(ctx, escrow); err != nil {
		return err
	}

	// Delete secondary index since the escrow is terminal.
	k.deleteEscrowIndex(ctx, escrow.Creator, escrow.Id)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"escrow_released",
		sdk.NewAttribute("escrow_id", escrowID),
		sdk.NewAttribute("provider_amount", sdk.NewCoin(escrow.Amount.Denom, providerAmount).String()),
		sdk.NewAttribute("protocol_fee", sdk.NewCoin(escrow.Amount.Denom, protocolFeeAmt).String()),
		sdk.NewAttribute("burn_amount", sdk.NewCoin(escrow.Amount.Denom, burnAmount).String()),
		sdk.NewAttribute("treasury_amount", sdk.NewCoin(escrow.Amount.Denom, treasuryAmount).String()),
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
	if escrow.Status != types.ESCROW_STATUS_LOCKED {
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

	escrow.Status = types.ESCROW_STATUS_REFUNDED
	if err := k.SetEscrow(ctx, escrow); err != nil {
		return err
	}

	// Delete secondary index since the escrow is terminal.
	k.deleteEscrowIndex(ctx, escrow.Creator, escrow.Id)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"escrow_refunded",
		sdk.NewAttribute("escrow_id", escrowID),
		sdk.NewAttribute("creator", escrow.Creator),
		sdk.NewAttribute("amount", escrow.Amount.String()),
	))

	return nil
}

// ExpireStaleEscrows iterates all escrows and auto-refunds those that have expired.
// This is intended to be called from EndBlock. Errors are collected and returned.
func (k Keeper) ExpireStaleEscrows(ctx sdk.Context) error {
	now := ctx.BlockTime()
	store := ctx.KVStore(k.storeKey)

	iter := storetypes.KVStorePrefixIterator(store, types.EscrowKeyPrefix)
	defer iter.Close()

	var errs []error

	for ; iter.Valid(); iter.Next() {
		var escrow types.Escrow
		if err := k.cdc.Unmarshal(iter.Value(), &escrow); err != nil {
			errs = append(errs, fmt.Errorf("failed to unmarshal escrow: %w", err))
			continue
		}
		if escrow.Status != types.ESCROW_STATUS_LOCKED {
			continue
		}
		if now.Before(escrow.ExpiresAt) {
			continue
		}

		// Escrow has expired -- refund the creator.
		creatorAddr, err := sdk.AccAddressFromBech32(escrow.Creator)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid creator address for escrow %s: %w", escrow.Id, err))
			continue
		}
		coins := sdk.NewCoins(escrow.Amount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, coins); err != nil {
			errs = append(errs, fmt.Errorf("failed to refund expired escrow %s: %w", escrow.Id, err))
			continue
		}

		escrow.Status = types.ESCROW_STATUS_EXPIRED
		bz, err := k.cdc.Marshal(&escrow)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to marshal escrow %s: %w", escrow.Id, err))
			continue
		}
		store.Set(types.EscrowKey(escrow.Id), bz)

		// Delete secondary index since the escrow is terminal.
		k.deleteEscrowIndex(ctx, escrow.Creator, escrow.Id)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"escrow_expired",
			sdk.NewAttribute("escrow_id", escrow.Id),
			sdk.NewAttribute("creator", escrow.Creator),
			sdk.NewAttribute("amount", escrow.Amount.String()),
		))
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors during escrow expiry: %v", len(errs), errs)
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
		if err := k.cdc.Unmarshal(iter.Value(), &escrow); err != nil {
			continue
		}
		if cb(escrow) {
			break
		}
	}
}
