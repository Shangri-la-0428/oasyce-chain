package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
	settlementtypes "github.com/oasyce/chain/x/settlement/types"
)

// Keeper manages the datarights module's state.
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
	authority  string // module authority address (arbitrator)
}

// NewKeeper creates a new datarights Keeper.
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

// Authority returns the module authority address.
func (k Keeper) Authority() string {
	return k.authority
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

// GetParams returns the datarights module parameters.
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

// SetParams sets the datarights module parameters.
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
// DataAsset CRUD
// ---------------------------------------------------------------------------

// GetAsset retrieves a data asset by ID.
func (k Keeper) GetAsset(ctx sdk.Context, assetID string) (types.DataAsset, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DataAssetKey(assetID))
	if bz == nil {
		return types.DataAsset{}, false
	}
	var asset types.DataAsset
	if err := k.cdc.Unmarshal(bz, &asset); err != nil {
		return types.DataAsset{}, false
	}
	return asset, true
}

// SetAsset persists a data asset to the store.
func (k Keeper) SetAsset(ctx sdk.Context, asset types.DataAsset) error {
	bz, err := k.cdc.Marshal(&asset)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.DataAssetKey(asset.Id), bz)
	return nil
}

// setAssetOwnerIndex creates a secondary index entry for owner -> asset.
func (k Keeper) setAssetOwnerIndex(ctx sdk.Context, owner, assetID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.AssetByOwnerKey(owner, assetID), []byte(assetID))
}

// ListAssets returns all data assets.
func (k Keeper) ListAssets(ctx sdk.Context) []types.DataAsset {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DataAssetKeyPrefix)
	defer iter.Close()

	var assets []types.DataAsset
	for ; iter.Valid(); iter.Next() {
		var asset types.DataAsset
		if err := k.cdc.Unmarshal(iter.Value(), &asset); err != nil {
			continue
		}
		assets = append(assets, asset)
	}
	return assets
}

// ListAssetsByOwner returns all data assets owned by the given address.
func (k Keeper) ListAssetsByOwner(ctx sdk.Context, owner string) []types.DataAsset {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AssetByOwnerIteratorPrefix(owner)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var assets []types.DataAsset
	for ; iter.Valid(); iter.Next() {
		assetID := string(iter.Value())
		asset, found := k.GetAsset(ctx, assetID)
		if found {
			assets = append(assets, asset)
		}
	}
	return assets
}

// generateAssetID creates a unique asset ID from content hash.
func (k Keeper) generateAssetID(ctx sdk.Context, contentHash string) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AssetCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.AssetCounterKey, newBz)

	// Deterministic ID from content hash + counter.
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", contentHash, counter)))
	return fmt.Sprintf("DATA_%s", hex.EncodeToString(h[:8]))
}

// ---------------------------------------------------------------------------
// ShareHolder CRUD
// ---------------------------------------------------------------------------

// GetShareHolder retrieves a shareholder record.
func (k Keeper) GetShareHolder(ctx sdk.Context, assetID, address string) (types.ShareHolder, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ShareHolderKey(assetID, address))
	if bz == nil {
		return types.ShareHolder{}, false
	}
	var sh types.ShareHolder
	if err := k.cdc.Unmarshal(bz, &sh); err != nil {
		return types.ShareHolder{}, false
	}
	return sh, true
}

// SetShareHolder persists a shareholder record.
func (k Keeper) SetShareHolder(ctx sdk.Context, sh types.ShareHolder) error {
	bz, err := k.cdc.Marshal(&sh)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ShareHolderKey(sh.AssetId, sh.Address), bz)
	// Secondary index.
	store.Set(types.ShareHolderByAssetKey(sh.AssetId, sh.Address), []byte(sh.Address))
	return nil
}

// GetShareHolders returns all shareholders for an asset.
func (k Keeper) GetShareHolders(ctx sdk.Context, assetID string) []types.ShareHolder {
	store := ctx.KVStore(k.storeKey)
	prefix := types.ShareHolderByAssetIteratorPrefix(assetID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var holders []types.ShareHolder
	for ; iter.Valid(); iter.Next() {
		address := string(iter.Value())
		sh, found := k.GetShareHolder(ctx, assetID, address)
		if found {
			holders = append(holders, sh)
		}
	}
	return holders
}

// IterateAllShareHolders iterates over all shareholder records and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllShareHolders(ctx sdk.Context, cb func(sh types.ShareHolder) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ShareHolderKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var sh types.ShareHolder
		if err := k.cdc.Unmarshal(iter.Value(), &sh); err != nil {
			continue
		}
		if cb(sh) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Dispute CRUD
// ---------------------------------------------------------------------------

// GetDispute retrieves a dispute by ID.
func (k Keeper) GetDispute(ctx sdk.Context, disputeID string) (types.Dispute, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DisputeKey(disputeID))
	if bz == nil {
		return types.Dispute{}, false
	}
	var dispute types.Dispute
	if err := k.cdc.Unmarshal(bz, &dispute); err != nil {
		return types.Dispute{}, false
	}
	return dispute, true
}

// SetDispute persists a dispute to the store.
func (k Keeper) SetDispute(ctx sdk.Context, dispute types.Dispute) error {
	bz, err := k.cdc.Marshal(&dispute)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.DisputeKey(dispute.Id), bz)
	return nil
}

// IterateAllDisputes iterates over all disputes and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllDisputes(ctx sdk.Context, cb func(d types.Dispute) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DisputeKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var d types.Dispute
		if err := k.cdc.Unmarshal(iter.Value(), &d); err != nil {
			continue
		}
		if cb(d) {
			break
		}
	}
}

// generateDisputeID creates a unique deterministic dispute ID.
// Uses counter + block hash for determinism across validators.
func (k Keeper) generateDisputeID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DisputeCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.DisputeCounterKey, newBz)

	// Deterministic: hash counter + block header for uniqueness.
	h := sha256.Sum256(append(newBz, ctx.HeaderHash()...))
	return fmt.Sprintf("DSP_%s", hex.EncodeToString(h[:8]))
}

// ---------------------------------------------------------------------------
// Asset Reserve (bonding curve backing)
// ---------------------------------------------------------------------------

// GetAssetReserve retrieves the bonding curve reserve for a data asset.
func (k Keeper) GetAssetReserve(ctx sdk.Context, assetID string) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AssetReserveKey(assetID))
	if bz == nil {
		return math.ZeroInt()
	}
	var reserve math.Int
	if err := reserve.Unmarshal(bz); err != nil {
		return math.ZeroInt()
	}
	return reserve
}

// GetAssetReserveDenom retrieves the bonding curve reserve denomination for a data asset.
// Returns "uoas" as fallback for pre-existing assets without stored denom.
func (k Keeper) GetAssetReserveDenom(ctx sdk.Context, assetID string) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AssetReserveDenomKey(assetID))
	if bz == nil {
		return "uoas"
	}
	return string(bz)
}

// SetAssetReserveDenom stores the bonding curve reserve denomination for a data asset.
func (k Keeper) SetAssetReserveDenom(ctx sdk.Context, assetID string, denom string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.AssetReserveDenomKey(assetID), []byte(denom))
}

// SetAssetReserve stores the bonding curve reserve for a data asset.
func (k Keeper) SetAssetReserve(ctx sdk.Context, assetID string, reserve math.Int) error {
	bz, err := reserve.Marshal()
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.AssetReserveKey(assetID), bz)
	return nil
}

// ---------------------------------------------------------------------------
// Content Hash Verification (Phase 7)
// ---------------------------------------------------------------------------

// VerifyContentHash verifies that the provided content matches the stored
// content hash for the given asset.
func (k Keeper) VerifyContentHash(ctx sdk.Context, assetID string, content []byte) error {
	asset, found := k.GetAsset(ctx, assetID)
	if !found {
		return types.ErrAssetNotFound
	}
	hash := sha256.Sum256(content)
	if hex.EncodeToString(hash[:]) != asset.ContentHash {
		return types.ErrContentHashMismatch
	}
	return nil
}

// ---------------------------------------------------------------------------
// Business Logic
// ---------------------------------------------------------------------------

// RegisterDataAsset validates and registers a new data asset.
func (k Keeper) RegisterDataAsset(ctx context.Context, msg types.MsgRegisterDataAsset) (string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}

	// Validate rights type.
	if msg.RightsType < types.RIGHTS_TYPE_ORIGINAL || msg.RightsType > types.RIGHTS_TYPE_COLLECTION {
		return "", types.ErrInvalidRightsType.Wrapf("invalid rights_type: %d", msg.RightsType)
	}

	// Validate co-creators if present.
	if len(msg.CoCreators) > 0 {
		params := k.GetParams(sdkCtx)
		if uint32(len(msg.CoCreators)) > params.MaxCoCreators {
			return "", types.ErrInvalidCoCreators.Wrapf("too many co-creators: %d > %d", len(msg.CoCreators), params.MaxCoCreators)
		}
		if err := types.ValidateCoCreators(msg.CoCreators); err != nil {
			return "", err
		}
	}

	// Versioning: validate parent and compute version.
	var parentAssetId string
	var version uint32 = 1
	if msg.ParentAssetId != "" {
		parent, found := k.GetAsset(sdkCtx, msg.ParentAssetId)
		if !found {
			return "", types.ErrAssetNotFound.Wrapf("parent asset %s not found", msg.ParentAssetId)
		}
		if parent.Status != types.ASSET_STATUS_ACTIVE && parent.Status != types.ASSET_STATUS_SHUTTING_DOWN {
			return "", types.ErrAssetDelisted.Wrap("parent asset must be active or shutting down")
		}
		parentAssetId = msg.ParentAssetId
		version = parent.Version + 1
	}

	assetID := k.generateAssetID(sdkCtx, msg.ContentHash)

	// Generate fingerprint from content hash.
	h := sha256.Sum256([]byte(msg.ContentHash))
	fingerprint := hex.EncodeToString(h[:16])

	asset := types.DataAsset{
		Id:            assetID,
		Owner:         msg.Creator,
		Name:          msg.Name,
		Description:   msg.Description,
		ContentHash:   msg.ContentHash,
		Fingerprint:   fingerprint,
		RightsType:    msg.RightsType,
		Tags:          msg.Tags,
		CoCreators:    msg.CoCreators,
		TotalShares:   math.ZeroInt(),
		CreatedAt:     sdkCtx.BlockTime(),
		Status:        types.ASSET_STATUS_ACTIVE,
		Version:       version,
		ParentAssetId: parentAssetId,
	}

	if err := k.SetAsset(sdkCtx, asset); err != nil {
		return "", err
	}
	k.setAssetOwnerIndex(sdkCtx, msg.Creator, assetID)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"data_asset_registered",
		sdk.NewAttribute("asset_id", assetID),
		sdk.NewAttribute("owner", msg.Creator),
		sdk.NewAttribute("name", msg.Name),
		sdk.NewAttribute("rights_type", msg.RightsType.String()),
	))

	return assetID, nil
}

// BuyShares purchases shares of a data asset via the Bancor bonding curve.
// Formula: tokens = supply × (sqrt(1 + payment/reserve) − 1)
// Bootstrap: tokens = payment / INITIAL_PRICE when reserve is zero.
// A rights type multiplier is applied to the minted tokens.
func (k Keeper) BuyShares(ctx context.Context, msg types.MsgBuyShares) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	buyerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return math.Int{}, types.ErrInvalidAddress.Wrapf("invalid buyer: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetId)
	if !found {
		return math.Int{}, types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetId)
	}
	if asset.Status != types.ASSET_STATUS_ACTIVE {
		return math.Int{}, types.ErrAssetDelisted.Wrapf("asset %s is not active (status: %s)", msg.AssetId, asset.Status)
	}

	paymentAmount := msg.Amount.Amount
	if paymentAmount.IsZero() || paymentAmount.IsNegative() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment must be positive")
	}

	// Get current bonding curve state.
	supply := asset.TotalShares
	reserve := k.GetAssetReserve(sdkCtx, msg.AssetId)

	// Bancor calculation.
	supplyDec := math.LegacyNewDecFromInt(supply)
	reserveDec := math.LegacyNewDecFromInt(reserve)
	paymentDec := math.LegacyNewDecFromInt(paymentAmount)

	var baseTokensDec math.LegacyDec
	if reserveDec.IsZero() || supplyDec.IsZero() {
		// Bootstrap: tokens = payment / INITIAL_PRICE
		baseTokensDec = paymentDec.Quo(settlementtypes.InitialPrice)
	} else {
		// ratio = 1 + payment/reserve
		ratio := math.LegacyOneDec().Add(paymentDec.Quo(reserveDec))
		sqrtRatio, sqrtErr := ratio.ApproxSqrt()
		if sqrtErr != nil {
			return math.Int{}, fmt.Errorf("bancor sqrt failed: %w", sqrtErr)
		}
		// tokens = supply × (sqrtRatio − 1)
		baseTokensDec = supplyDec.Mul(sqrtRatio.Sub(math.LegacyOneDec()))
	}

	// Apply rights type multiplier.
	multiplier := types.RightsTypeMultiplier(asset.RightsType)
	sharesMinted := multiplier.Mul(baseTokensDec).TruncateInt()

	if sharesMinted.IsZero() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment too small to mint shares")
	}

	// Front-running protection: reject if minted shares below caller's minimum.
	if msg.MinSharesOut != nil && !msg.MinSharesOut.IsZero() && sharesMinted.LT(*msg.MinSharesOut) {
		return math.Int{}, types.ErrSlippageExceeded.Wrapf(
			"slippage: would mint %s shares, minimum requested %s", sharesMinted, msg.MinSharesOut)
	}

	// Transfer payment from buyer to module.
	coins := sdk.NewCoins(msg.Amount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, buyerAddr, types.ModuleName, coins); err != nil {
		return math.Int{}, types.ErrInsufficientFunds.Wrapf("failed to collect payment: %s", err)
	}

	// Update asset total shares.
	asset.TotalShares = asset.TotalShares.Add(sharesMinted)
	if err := k.SetAsset(sdkCtx, asset); err != nil {
		return math.Int{}, err
	}

	// Update reserve and store denom (set on first buy, verify on subsequent).
	storedDenom := k.GetAssetReserveDenom(sdkCtx, msg.AssetId)
	if storedDenom == "uoas" && reserve.IsZero() {
		// First buy — store the actual denom.
		k.SetAssetReserveDenom(sdkCtx, msg.AssetId, msg.Amount.Denom)
	}
	newReserve := reserve.Add(paymentAmount)
	if err := k.SetAssetReserve(sdkCtx, msg.AssetId, newReserve); err != nil {
		return math.Int{}, err
	}

	// Update or create shareholder record.
	sh, found := k.GetShareHolder(sdkCtx, msg.AssetId, msg.Creator)
	if !found {
		sh = types.ShareHolder{
			Address:     msg.Creator,
			AssetId:     msg.AssetId,
			Shares:      math.ZeroInt(),
			PurchasedAt: sdkCtx.BlockTime(),
		}
	}
	sh.Shares = sh.Shares.Add(sharesMinted)
	sh.PurchasedAt = sdkCtx.BlockTime()
	if err := k.SetShareHolder(sdkCtx, sh); err != nil {
		return math.Int{}, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shares_bought",
		sdk.NewAttribute("asset_id", msg.AssetId),
		sdk.NewAttribute("buyer", msg.Creator),
		sdk.NewAttribute("payment", msg.Amount.String()),
		sdk.NewAttribute("shares_minted", sharesMinted.String()),
		sdk.NewAttribute("new_supply", asset.TotalShares.String()),
		sdk.NewAttribute("new_reserve", newReserve.String()),
	))

	return sharesMinted, nil
}

// SellShares sells tokens back to the bonding curve using the inverse Bancor formula.
// Formula: payout = reserve × (1 − (1 − tokens/supply)²)
// Capped at 95% of reserve (RESERVE_SOLVENCY_CAP).
// Protocol fee (5%) is deducted from the gross payout.
func (k Keeper) SellShares(ctx context.Context, msg types.MsgSellShares) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sellerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return math.Int{}, types.ErrInvalidAddress.Wrapf("invalid seller: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetId)
	if !found {
		return math.Int{}, types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetId)
	}

	// Block selling if asset is settled. During shutdown, allow selling only within cooldown.
	if asset.Status == types.ASSET_STATUS_SETTLED {
		return math.Int{}, types.ErrAssetSettled.Wrapf("asset %s is settled, use ClaimSettlement", msg.AssetId)
	}
	if asset.Status == types.ASSET_STATUS_SHUTTING_DOWN {
		params := k.GetParams(sdkCtx)
		cooldownEnd := asset.ShutdownInitiatedAt.Add(time.Duration(params.ShutdownCooldownSeconds) * time.Second)
		if sdkCtx.BlockTime().After(cooldownEnd) || sdkCtx.BlockTime().Equal(cooldownEnd) {
			return math.Int{}, types.ErrAssetSettled.Wrapf("cooldown elapsed for asset %s, use ClaimSettlement", msg.AssetId)
		}
	}

	tokensToSell := msg.Shares
	if tokensToSell.IsZero() || tokensToSell.IsNegative() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("shares must be positive")
	}

	// Verify seller has enough shares.
	sh, found := k.GetShareHolder(sdkCtx, msg.AssetId, msg.Creator)
	if !found || sh.Shares.LT(tokensToSell) {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("insufficient shares to sell")
	}

	supply := asset.TotalShares
	reserve := k.GetAssetReserve(sdkCtx, msg.AssetId)

	if supply.IsZero() || reserve.IsZero() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("pool has no liquidity")
	}

	// Inverse Bancor: payout = reserve × (1 − (1 − tokens/supply)^(1/CW))
	// CW=0.5 → 1/CW=2, so (1 − tokens/supply)²
	supplyDec := math.LegacyNewDecFromInt(supply)
	reserveDec := math.LegacyNewDecFromInt(reserve)
	tokensDec := math.LegacyNewDecFromInt(tokensToSell)
	solvencyCap := settlementtypes.ReserveSolvencyCap
	maxPayout := reserveDec.Mul(solvencyCap)

	var grossPayout math.LegacyDec
	if tokensDec.GTE(supplyDec) {
		grossPayout = maxPayout
	} else {
		ratio := math.LegacyOneDec().Sub(tokensDec.Quo(supplyDec))
		ratioSquared := ratio.Mul(ratio)
		grossPayout = reserveDec.Mul(math.LegacyOneDec().Sub(ratioSquared))
		if grossPayout.GT(maxPayout) {
			grossPayout = maxPayout
		}
	}

	// Deduct protocol fee (5%).
	protocolFeeDec := grossPayout.Mul(settlementtypes.DefaultParams().ProtocolFeeRate)
	feeAmount := protocolFeeDec.TruncateInt()
	// Guard against fee truncation to 0 on small sells — minimum 1 uoas fee.
	if feeAmount.IsZero() && grossPayout.IsPositive() {
		feeAmount = math.OneInt()
	}
	netPayout := grossPayout.Sub(math.LegacyNewDecFromInt(feeAmount))
	payoutAmount := netPayout.TruncateInt()

	if payoutAmount.IsZero() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("sell amount too small")
	}

	// Front-running protection: reject if payout below caller's minimum.
	if msg.MinPayoutOut != nil && !msg.MinPayoutOut.IsZero() && payoutAmount.LT(*msg.MinPayoutOut) {
		return math.Int{}, types.ErrSlippageExceeded.Wrapf(
			"slippage: would pay out %s, minimum requested %s", payoutAmount, msg.MinPayoutOut)
	}

	// Resolve reserve denomination for coin creation.
	reserveDenom := k.GetAssetReserveDenom(sdkCtx, msg.AssetId)

	// Send payout from module to seller.
	if payoutAmount.IsPositive() {
		payoutCoin := sdk.NewCoin(reserveDenom, payoutAmount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sellerAddr, sdk.NewCoins(payoutCoin)); err != nil {
			return math.Int{}, fmt.Errorf("failed to send payout: %w", err)
		}
	}

	// Send protocol fee to fee_collector.
	if feeAmount.IsPositive() {
		feeCoin := sdk.NewCoin(reserveDenom, feeAmount)
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "fee_collector", sdk.NewCoins(feeCoin)); err != nil {
			return math.Int{}, fmt.Errorf("failed to send protocol fee: %w", err)
		}
	}

	// Update asset total shares.
	asset.TotalShares = asset.TotalShares.Sub(tokensToSell)
	if err := k.SetAsset(sdkCtx, asset); err != nil {
		return math.Int{}, err
	}

	// Update reserve.
	reserveReduction := payoutAmount.Add(feeAmount)
	newReserve := reserve.Sub(reserveReduction)
	if newReserve.IsNegative() {
		newReserve = math.ZeroInt()
	}
	if err := k.SetAssetReserve(sdkCtx, msg.AssetId, newReserve); err != nil {
		return math.Int{}, err
	}

	// Update shareholder record.
	sh.Shares = sh.Shares.Sub(tokensToSell)
	if sh.Shares.IsZero() {
		// Remove shareholder record entirely.
		store := sdkCtx.KVStore(k.storeKey)
		store.Delete(types.ShareHolderKey(msg.AssetId, msg.Creator))
		store.Delete(types.ShareHolderByAssetKey(msg.AssetId, msg.Creator))
	} else {
		if err := k.SetShareHolder(sdkCtx, sh); err != nil {
			return math.Int{}, err
		}
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shares_sold",
		sdk.NewAttribute("asset_id", msg.AssetId),
		sdk.NewAttribute("seller", msg.Creator),
		sdk.NewAttribute("shares_sold", tokensToSell.String()),
		sdk.NewAttribute("payout", payoutAmount.String()),
		sdk.NewAttribute("protocol_fee", feeAmount.String()),
		sdk.NewAttribute("new_supply", asset.TotalShares.String()),
		sdk.NewAttribute("new_reserve", newReserve.String()),
	))

	return payoutAmount, nil
}

// DelistAsset allows an asset owner to voluntarily delist their own asset.
// Deprecated: now redirects to InitiateShutdown for graceful shutdown.
func (k Keeper) DelistAsset(ctx context.Context, msg types.MsgDelistAsset) error {
	return k.InitiateShutdown(ctx, types.MsgInitiateShutdown(msg))
}

// InitiateShutdown begins graceful shutdown of a data asset.
// Only the owner can initiate. Sets status to SHUTTING_DOWN and records timestamp.
func (k Keeper) InitiateShutdown(ctx context.Context, msg types.MsgInitiateShutdown) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetId)
	if !found {
		return types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetId)
	}
	if asset.Owner != msg.Creator {
		return types.ErrUnauthorized.Wrapf("only the owner can initiate shutdown")
	}
	if asset.Status != types.ASSET_STATUS_ACTIVE {
		return types.ErrAssetDelisted.Wrapf("asset %s is not active (status: %s)", msg.AssetId, asset.Status)
	}

	asset.Status = types.ASSET_STATUS_SHUTTING_DOWN
	asset.ShutdownInitiatedAt = sdkCtx.BlockTime()
	if err := k.SetAsset(sdkCtx, asset); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"asset_shutdown_initiated",
		sdk.NewAttribute("asset_id", msg.AssetId),
		sdk.NewAttribute("owner", msg.Creator),
	))

	return nil
}

// ClaimSettlement claims pro-rata reserve payout after shutdown cooldown.
// Any shareholder can call this after the cooldown period has elapsed.
// No protocol fee is charged on settlement claims.
func (k Keeper) ClaimSettlement(ctx context.Context, msg types.MsgClaimSettlement) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	claimerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return math.Int{}, types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetId)
	if !found {
		return math.Int{}, types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetId)
	}
	if asset.Status != types.ASSET_STATUS_SHUTTING_DOWN && asset.Status != types.ASSET_STATUS_SETTLED {
		return math.Int{}, types.ErrAssetNotShuttingDown.Wrapf("asset %s is active, cannot claim settlement", msg.AssetId)
	}

	// Check cooldown has elapsed.
	params := k.GetParams(sdkCtx)
	cooldownEnd := asset.ShutdownInitiatedAt.Add(time.Duration(params.ShutdownCooldownSeconds) * time.Second)
	if sdkCtx.BlockTime().Before(cooldownEnd) {
		return math.Int{}, types.ErrCooldownNotElapsed.Wrapf("cooldown ends at %s, current time %s", cooldownEnd, sdkCtx.BlockTime())
	}

	// Get claimer's shares.
	sh, found := k.GetShareHolder(sdkCtx, msg.AssetId, msg.Creator)
	if !found || sh.Shares.IsZero() {
		return math.Int{}, types.ErrNoSharesHeld.Wrapf("no shares held for asset %s", msg.AssetId)
	}

	// Pro-rata payout: payout = reserve * (claimer_shares / total_shares)
	reserve := k.GetAssetReserve(sdkCtx, msg.AssetId)
	if reserve.IsZero() || asset.TotalShares.IsZero() {
		// No reserve left — just burn shares, zero payout.
		asset.TotalShares = asset.TotalShares.Sub(sh.Shares)
		if asset.TotalShares.IsZero() || asset.TotalShares.IsNegative() {
			asset.Status = types.ASSET_STATUS_SETTLED
			asset.TotalShares = math.ZeroInt()
		}
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return math.Int{}, err
		}
		store := sdkCtx.KVStore(k.storeKey)
		store.Delete(types.ShareHolderKey(msg.AssetId, msg.Creator))
		store.Delete(types.ShareHolderByAssetKey(msg.AssetId, msg.Creator))
		return math.ZeroInt(), nil
	}

	reserveDec := math.LegacyNewDecFromInt(reserve)
	sharesDec := math.LegacyNewDecFromInt(sh.Shares)
	totalDec := math.LegacyNewDecFromInt(asset.TotalShares)

	payout := reserveDec.Mul(sharesDec).Quo(totalDec).TruncateInt()
	if payout.IsZero() {
		payout = math.ZeroInt()
	}

	// Send payout from module to claimer (no protocol fee on settlement).
	reserveDenom := k.GetAssetReserveDenom(sdkCtx, msg.AssetId)
	if payout.IsPositive() {
		payoutCoin := sdk.NewCoin(reserveDenom, payout)
		if sendErr := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimerAddr, sdk.NewCoins(payoutCoin)); sendErr != nil {
			return math.Int{}, fmt.Errorf("failed to send settlement: %w", sendErr)
		}
	}

	// Update reserve.
	newReserve := reserve.Sub(payout)
	if newReserve.IsNegative() {
		newReserve = math.ZeroInt()
	}
	if err := k.SetAssetReserve(sdkCtx, msg.AssetId, newReserve); err != nil {
		return math.Int{}, err
	}

	// Update asset total shares.
	asset.TotalShares = asset.TotalShares.Sub(sh.Shares)
	if asset.TotalShares.IsZero() || asset.TotalShares.IsNegative() {
		asset.Status = types.ASSET_STATUS_SETTLED
		asset.TotalShares = math.ZeroInt()
	}
	if err := k.SetAsset(sdkCtx, asset); err != nil {
		return math.Int{}, err
	}

	// Remove shareholder record.
	store := sdkCtx.KVStore(k.storeKey)
	store.Delete(types.ShareHolderKey(msg.AssetId, msg.Creator))
	store.Delete(types.ShareHolderByAssetKey(msg.AssetId, msg.Creator))

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"settlement_claimed",
		sdk.NewAttribute("asset_id", msg.AssetId),
		sdk.NewAttribute("claimer", msg.Creator),
		sdk.NewAttribute("payout", payout.String()),
		sdk.NewAttribute("shares_burned", sh.Shares.String()),
	))

	return payout, nil
}

// FileDispute creates a new dispute against a data asset.
func (k Keeper) FileDispute(ctx context.Context, msg types.MsgFileDispute) (string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	plaintiffAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid plaintiff: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetId)
	if !found {
		return "", types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetId)
	}
	if asset.Status != types.ASSET_STATUS_ACTIVE {
		return "", types.ErrAssetDelisted.Wrapf("asset %s is not active", msg.AssetId)
	}

	// Require dispute deposit.
	params := k.GetParams(sdkCtx)
	deposit := sdk.NewCoins(params.DisputeDeposit)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, plaintiffAddr, types.ModuleName, deposit); err != nil {
		return "", types.ErrInsufficientFunds.Wrapf("failed to collect dispute deposit: %s", err)
	}

	// Compute evidence hash.
	evidenceHash := ""
	if len(msg.Evidence) > 0 {
		h := sha256.Sum256(msg.Evidence)
		evidenceHash = hex.EncodeToString(h[:])
	}

	disputeID := k.generateDisputeID(sdkCtx)
	// Default to delist if no remedy specified.
	requestedRemedy := msg.RequestedRemedy
	if requestedRemedy == types.DISPUTE_REMEDY_UNSPECIFIED {
		requestedRemedy = types.DISPUTE_REMEDY_DELIST
	}

	dispute := types.Dispute{
		Id:              disputeID,
		AssetId:         msg.AssetId,
		Plaintiff:       msg.Creator,
		Reason:          msg.Reason,
		EvidenceHash:    evidenceHash,
		Status:          types.DISPUTE_STATUS_OPEN,
		Remedy:          types.DISPUTE_REMEDY_UNSPECIFIED,
		Arbitrator:      "",
		ResolvedAt:      time.Time{},
		RequestedRemedy: requestedRemedy,
	}

	if err := k.SetDispute(sdkCtx, dispute); err != nil {
		return "", err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_filed",
		sdk.NewAttribute("dispute_id", disputeID),
		sdk.NewAttribute("asset_id", msg.AssetId),
		sdk.NewAttribute("plaintiff", msg.Creator),
		sdk.NewAttribute("reason", msg.Reason),
	))

	return disputeID, nil
}

// ResolveDispute resolves an open dispute by applying the chosen remedy.
func (k Keeper) ResolveDispute(ctx context.Context, msg types.MsgResolveDispute) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Only the authority (arbitrator) can resolve disputes.
	if msg.Creator != k.authority {
		return types.ErrNotArbitrator.Wrapf("caller %s is not the arbitrator %s", msg.Creator, k.authority)
	}

	dispute, found := k.GetDispute(sdkCtx, msg.DisputeId)
	if !found {
		return types.ErrDisputeNotFound.Wrapf("dispute %s not found", msg.DisputeId)
	}
	if dispute.Status != types.DISPUTE_STATUS_OPEN {
		return types.ErrDisputeNotOpen.Wrapf("dispute %s is %s, not OPEN", msg.DisputeId, dispute.Status)
	}

	// Execute remedy.
	switch msg.Remedy {
	case types.DISPUTE_REMEDY_DELIST:
		asset, found := k.GetAsset(sdkCtx, dispute.AssetId)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetId)
		}
		asset.Status = types.ASSET_STATUS_SHUTTING_DOWN
		asset.ShutdownInitiatedAt = sdkCtx.BlockTime()
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	case types.DISPUTE_REMEDY_TRANSFER:
		// Details should contain the new owner address.
		if len(msg.Details) == 0 {
			return types.ErrInvalidParams.Wrap("transfer remedy requires new_owner in details")
		}
		newOwner := string(msg.Details)
		if _, err := sdk.AccAddressFromBech32(newOwner); err != nil {
			return types.ErrInvalidAddress.Wrapf("invalid new owner: %s", err)
		}
		asset, found := k.GetAsset(sdkCtx, dispute.AssetId)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetId)
		}
		asset.Owner = newOwner
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}
		k.setAssetOwnerIndex(sdkCtx, newOwner, asset.Id)

	case types.DISPUTE_REMEDY_RIGHTS_CORRECTION:
		// Details should contain the new rights type as a single byte.
		if len(msg.Details) == 0 {
			return types.ErrInvalidParams.Wrap("rights_correction remedy requires new rights_type in details")
		}
		newRightsType := types.RightsType(msg.Details[0])
		if newRightsType < types.RIGHTS_TYPE_ORIGINAL || newRightsType > types.RIGHTS_TYPE_COLLECTION {
			return types.ErrInvalidRightsType.Wrapf("invalid new rights_type: %d", newRightsType)
		}
		asset, found := k.GetAsset(sdkCtx, dispute.AssetId)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetId)
		}
		asset.RightsType = newRightsType
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	case types.DISPUTE_REMEDY_SHARE_ADJUSTMENT:
		// Details contain JSON-encoded co-creator adjustments.
		if len(msg.Details) == 0 {
			return types.ErrInvalidParams.Wrap("share_adjustment remedy requires co-creator details")
		}
		var newCoCreators []types.CoCreator
		if err := json.Unmarshal(msg.Details, &newCoCreators); err != nil {
			return types.ErrInvalidParams.Wrapf("failed to parse co-creator details: %s", err)
		}
		if err := types.ValidateCoCreators(newCoCreators); err != nil {
			return err
		}
		asset, found := k.GetAsset(sdkCtx, dispute.AssetId)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetId)
		}
		asset.CoCreators = newCoCreators
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	default:
		return types.ErrInvalidParams.Wrapf("unknown remedy: %d", msg.Remedy)
	}

	// Return dispute deposit to plaintiff (dispute was resolved — remedy applied).
	plaintiffAddr, pErr := sdk.AccAddressFromBech32(dispute.Plaintiff)
	if pErr == nil {
		params := k.GetParams(sdkCtx)
		deposit := sdk.NewCoins(params.DisputeDeposit)
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, plaintiffAddr, deposit)
	}

	// Update dispute.
	dispute.Status = types.DISPUTE_STATUS_RESOLVED
	dispute.Remedy = msg.Remedy
	dispute.Arbitrator = msg.Creator
	dispute.ResolvedAt = sdkCtx.BlockTime()
	if err := k.SetDispute(sdkCtx, dispute); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_resolved",
		sdk.NewAttribute("dispute_id", msg.DisputeId),
		sdk.NewAttribute("remedy", msg.Remedy.String()),
		sdk.NewAttribute("arbitrator", msg.Creator),
	))

	return nil
}

// deleteShareHolder removes a shareholder record and its secondary index.
func (k Keeper) deleteShareHolder(ctx sdk.Context, assetID, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ShareHolderKey(assetID, address))
	store.Delete(types.ShareHolderByAssetKey(assetID, address))
}

// ---------------------------------------------------------------------------
// Migration Path CRUD
// ---------------------------------------------------------------------------

// GetMigrationPath retrieves a migration path by source and target asset IDs.
func (k Keeper) GetMigrationPath(ctx sdk.Context, sourceAssetID, targetAssetID string) (types.MigrationPath, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.MigrationPathKey(sourceAssetID, targetAssetID))
	if bz == nil {
		return types.MigrationPath{}, false
	}
	var mp types.MigrationPath
	if err := k.cdc.Unmarshal(bz, &mp); err != nil {
		return types.MigrationPath{}, false
	}
	return mp, true
}

// SetMigrationPath persists a migration path.
func (k Keeper) SetMigrationPath(ctx sdk.Context, mp types.MigrationPath) error {
	bz, err := k.cdc.Marshal(&mp)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.MigrationPathKey(mp.SourceAssetId, mp.TargetAssetId), bz)
	return nil
}

// IterateAllMigrationPaths iterates over all migration paths.
func (k Keeper) IterateAllMigrationPaths(ctx sdk.Context, cb func(mp types.MigrationPath) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.MigrationPathKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var mp types.MigrationPath
		if err := k.cdc.Unmarshal(iter.Value(), &mp); err != nil {
			continue
		}
		if cb(mp) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Migration Business Logic
// ---------------------------------------------------------------------------

// CreateMigrationPath creates a migration path from source to target asset.
// Only the target asset owner can create a migration path.
// The target asset must reference the source as its parent (or be a legitimate successor).
func (k Keeper) CreateMigrationPath(ctx context.Context, msg types.MsgCreateMigrationPath) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Verify target asset exists and caller is its owner.
	targetAsset, found := k.GetAsset(sdkCtx, msg.TargetAssetId)
	if !found {
		return types.ErrAssetNotFound.Wrapf("target asset %s not found", msg.TargetAssetId)
	}
	if targetAsset.Owner != msg.Creator {
		return types.ErrUnauthorized.Wrap("only target asset owner can create migration path")
	}
	if targetAsset.Status != types.ASSET_STATUS_ACTIVE {
		return types.ErrAssetDelisted.Wrap("target asset must be active")
	}

	// Verify source asset exists.
	sourceAsset, found := k.GetAsset(sdkCtx, msg.SourceAssetId)
	if !found {
		return types.ErrAssetNotFound.Wrapf("source asset %s not found", msg.SourceAssetId)
	}

	// Target must declare source as parent (version chain integrity).
	if targetAsset.ParentAssetId != sourceAsset.Id {
		return types.ErrInvalidVersion.Wrap("target asset must reference source as parent_asset_id")
	}

	// Check no existing migration path for this pair.
	if _, exists := k.GetMigrationPath(sdkCtx, msg.SourceAssetId, msg.TargetAssetId); exists {
		return types.ErrMigrationExists.Wrap("migration path already exists for this pair")
	}

	mp := types.MigrationPath{
		SourceAssetId:    msg.SourceAssetId,
		TargetAssetId:    msg.TargetAssetId,
		ExchangeRateBps:  msg.ExchangeRateBps,
		MaxMigratedShares: msg.MaxMigratedShares,
		TotalMigrated:    math.ZeroInt(),
		Enabled:          true,
		CreatedAt:        sdkCtx.BlockTime(),
	}

	if err := k.SetMigrationPath(sdkCtx, mp); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"migration_path_created",
		sdk.NewAttribute("source_asset_id", msg.SourceAssetId),
		sdk.NewAttribute("target_asset_id", msg.TargetAssetId),
		sdk.NewAttribute("exchange_rate_bps", fmt.Sprintf("%d", msg.ExchangeRateBps)),
	))

	return nil
}

// DisableMigration disables an active migration path.
// Only the target asset owner can disable it.
func (k Keeper) DisableMigration(ctx context.Context, msg types.MsgDisableMigration) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	mp, found := k.GetMigrationPath(sdkCtx, msg.SourceAssetId, msg.TargetAssetId)
	if !found {
		return types.ErrMigrationNotFound.Wrapf("migration path %s->%s not found", msg.SourceAssetId, msg.TargetAssetId)
	}

	// Verify caller is target asset owner.
	targetAsset, found := k.GetAsset(sdkCtx, msg.TargetAssetId)
	if !found {
		return types.ErrAssetNotFound.Wrapf("target asset %s not found", msg.TargetAssetId)
	}
	if targetAsset.Owner != msg.Creator {
		return types.ErrUnauthorized.Wrap("only target asset owner can disable migration")
	}

	mp.Enabled = false
	if err := k.SetMigrationPath(sdkCtx, mp); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"migration_disabled",
		sdk.NewAttribute("source_asset_id", msg.SourceAssetId),
		sdk.NewAttribute("target_asset_id", msg.TargetAssetId),
	))

	return nil
}

// Migrate converts shares from source asset to target asset via migration path.
// Shares are burned from source and minted directly to target (no bonding curve).
// This is a dilutive operation — migrated shares have no reserve backing.
func (k Keeper) Migrate(ctx context.Context, msg types.MsgMigrate) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	mp, found := k.GetMigrationPath(sdkCtx, msg.SourceAssetId, msg.TargetAssetId)
	if !found {
		return math.Int{}, types.ErrMigrationNotFound.Wrapf("migration path %s->%s not found", msg.SourceAssetId, msg.TargetAssetId)
	}
	if !mp.Enabled {
		return math.Int{}, types.ErrMigrationDisabled.Wrap("migration path is disabled")
	}

	// Verify source shareholder holds enough shares.
	sh, found := k.GetShareHolder(sdkCtx, msg.SourceAssetId, msg.Creator)
	if !found || sh.Shares.LT(msg.Shares) {
		return math.Int{}, types.ErrNoSharesHeld.Wrap("insufficient shares in source asset")
	}

	// Calculate target shares: source_shares * exchange_rate_bps / 10000
	targetShares := msg.Shares.Mul(math.NewInt(int64(mp.ExchangeRateBps))).Quo(math.NewInt(10000))
	if targetShares.IsZero() {
		return math.Int{}, types.ErrInvalidParams.Wrap("migration would yield zero shares")
	}

	// Check migration cap.
	if mp.MaxMigratedShares.IsPositive() {
		newTotal := mp.TotalMigrated.Add(msg.Shares)
		if newTotal.GT(mp.MaxMigratedShares) {
			return math.Int{}, types.ErrMigrationCapExceeded.Wrapf(
				"would exceed cap: %s + %s > %s",
				mp.TotalMigrated, msg.Shares, mp.MaxMigratedShares,
			)
		}
	}

	// Burn shares from source.
	sh.Shares = sh.Shares.Sub(msg.Shares)
	if sh.Shares.IsZero() {
		k.deleteShareHolder(sdkCtx, msg.SourceAssetId, msg.Creator)
	} else {
		if err := k.SetShareHolder(sdkCtx, sh); err != nil {
			return math.Int{}, err
		}
	}

	// Update source asset total_shares.
	sourceAsset, _ := k.GetAsset(sdkCtx, msg.SourceAssetId)
	sourceAsset.TotalShares = sourceAsset.TotalShares.Sub(msg.Shares)
	if err := k.SetAsset(sdkCtx, sourceAsset); err != nil {
		return math.Int{}, err
	}

	// Mint shares to target.
	targetSh, found := k.GetShareHolder(sdkCtx, msg.TargetAssetId, msg.Creator)
	if !found {
		targetSh = types.ShareHolder{
			Address:     msg.Creator,
			AssetId:     msg.TargetAssetId,
			Shares:      math.ZeroInt(),
			PurchasedAt: sdkCtx.BlockTime(),
		}
	}
	targetSh.Shares = targetSh.Shares.Add(targetShares)
	if err := k.SetShareHolder(sdkCtx, targetSh); err != nil {
		return math.Int{}, err
	}

	// Update target asset total_shares.
	targetAsset, _ := k.GetAsset(sdkCtx, msg.TargetAssetId)
	targetAsset.TotalShares = targetAsset.TotalShares.Add(targetShares)
	if err := k.SetAsset(sdkCtx, targetAsset); err != nil {
		return math.Int{}, err
	}

	// Update migration path total_migrated.
	mp.TotalMigrated = mp.TotalMigrated.Add(msg.Shares)
	if err := k.SetMigrationPath(sdkCtx, mp); err != nil {
		return math.Int{}, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shares_migrated",
		sdk.NewAttribute("source_asset_id", msg.SourceAssetId),
		sdk.NewAttribute("target_asset_id", msg.TargetAssetId),
		sdk.NewAttribute("migrator", msg.Creator),
		sdk.NewAttribute("source_shares_burned", msg.Shares.String()),
		sdk.NewAttribute("target_shares_minted", targetShares.String()),
	))

	return targetShares, nil
}

// IterateAllAssets iterates over all data assets and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllAssets(ctx sdk.Context, cb func(asset types.DataAsset) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DataAssetKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var asset types.DataAsset
		if err := k.cdc.Unmarshal(iter.Value(), &asset); err != nil {
			continue
		}
		if cb(asset) {
			break
		}
	}
}
