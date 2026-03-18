package keeper

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
)

// Keeper manages the datarights module's state.
type Keeper struct {
	cdc              codec.BinaryCodec
	storeKey         storetypes.StoreKey
	bankKeeper       types.BankKeeper
	settlementKeeper types.SettlementKeeper
	authority        string // module authority address (arbitrator)
}

// NewKeeper creates a new datarights Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	settlementKeeper types.SettlementKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		bankKeeper:       bankKeeper,
		settlementKeeper: settlementKeeper,
		authority:        authority,
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
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the datarights module parameters.
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
	if err := json.Unmarshal(bz, &asset); err != nil {
		return types.DataAsset{}, false
	}
	return asset, true
}

// SetAsset persists a data asset to the store.
func (k Keeper) SetAsset(ctx sdk.Context, asset types.DataAsset) error {
	bz, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.DataAssetKey(asset.ID), bz)
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
		if err := json.Unmarshal(iter.Value(), &asset); err != nil {
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
	if err := json.Unmarshal(bz, &sh); err != nil {
		return types.ShareHolder{}, false
	}
	return sh, true
}

// SetShareHolder persists a shareholder record.
func (k Keeper) SetShareHolder(ctx sdk.Context, sh types.ShareHolder) error {
	bz, err := json.Marshal(sh)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ShareHolderKey(sh.AssetID, sh.Address), bz)
	// Secondary index.
	store.Set(types.ShareHolderByAssetKey(sh.AssetID, sh.Address), []byte(sh.Address))
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
	if err := json.Unmarshal(bz, &dispute); err != nil {
		return types.Dispute{}, false
	}
	return dispute, true
}

// SetDispute persists a dispute to the store.
func (k Keeper) SetDispute(ctx sdk.Context, dispute types.Dispute) error {
	bz, err := json.Marshal(dispute)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.DisputeKey(dispute.ID), bz)
	return nil
}

// generateDisputeID creates a unique dispute ID.
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

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("DSP_%s%s", hex.EncodeToString(newBz), hex.EncodeToString(randBytes))
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
	if msg.RightsType < types.RightsOriginal || msg.RightsType > types.RightsCollection {
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

	assetID := k.generateAssetID(sdkCtx, msg.ContentHash)

	// Generate fingerprint from content hash.
	h := sha256.Sum256([]byte(msg.ContentHash))
	fingerprint := hex.EncodeToString(h[:16])

	asset := types.DataAsset{
		ID:          assetID,
		Owner:       msg.Creator,
		Name:        msg.Name,
		Description: msg.Description,
		ContentHash: msg.ContentHash,
		Fingerprint: fingerprint,
		RightsType:  msg.RightsType,
		Tags:        msg.Tags,
		CoCreators:  msg.CoCreators,
		TotalShares: math.ZeroInt(),
		CreatedAt:   sdkCtx.BlockTime().Unix(),
		IsActive:    true,
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

// BuyShares purchases shares of a data asset via the bonding curve with
// diminishing returns based on total shares outstanding (per spec section 13).
func (k Keeper) BuyShares(ctx context.Context, msg types.MsgBuyShares) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	buyerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return math.Int{}, types.ErrInvalidAddress.Wrapf("invalid buyer: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetID)
	if !found {
		return math.Int{}, types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetID)
	}
	if !asset.IsActive {
		return math.Int{}, types.ErrAssetDelisted.Wrapf("asset %s is delisted", msg.AssetID)
	}

	paymentAmount := msg.Amount.Amount
	if paymentAmount.IsZero() || paymentAmount.IsNegative() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment must be positive")
	}

	// Calculate diminishing returns based on total shares outstanding.
	rateBps := shareRateBps(asset.TotalShares)

	// Apply rights type multiplier.
	multiplier := asset.RightsType.Multiplier()
	baseShares := paymentAmount.Mul(rateBps).Quo(math.NewInt(10000))
	sharesMinted := multiplier.MulInt(baseShares).TruncateInt()

	if sharesMinted.IsZero() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment too small to mint shares")
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

	// Update or create shareholder record.
	sh, found := k.GetShareHolder(sdkCtx, msg.AssetID, msg.Creator)
	if !found {
		sh = types.ShareHolder{
			Address:     msg.Creator,
			AssetID:     msg.AssetID,
			Shares:      math.ZeroInt(),
			PurchasedAt: sdkCtx.BlockTime().Unix(),
		}
	}
	sh.Shares = sh.Shares.Add(sharesMinted)
	sh.PurchasedAt = sdkCtx.BlockTime().Unix()
	if err := k.SetShareHolder(sdkCtx, sh); err != nil {
		return math.Int{}, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shares_bought",
		sdk.NewAttribute("asset_id", msg.AssetID),
		sdk.NewAttribute("buyer", msg.Creator),
		sdk.NewAttribute("payment", msg.Amount.String()),
		sdk.NewAttribute("shares_minted", sharesMinted.String()),
	))

	return sharesMinted, nil
}

// shareRateBps returns the diminishing share rate in basis points based on
// total shares outstanding. Per spec section 13:
//
//	First 1000 shares:  100% = 10000 bps
//	1001-5000:           80% =  8000 bps
//	5001-10000:          60% =  6000 bps
//	10001+:              40% =  4000 bps
func shareRateBps(totalShares math.Int) math.Int {
	switch {
	case totalShares.LT(math.NewInt(1000)):
		return math.NewInt(10000)
	case totalShares.LT(math.NewInt(5000)):
		return math.NewInt(8000)
	case totalShares.LT(math.NewInt(10000)):
		return math.NewInt(6000)
	default:
		return math.NewInt(4000)
	}
}

// FileDispute creates a new dispute against a data asset.
func (k Keeper) FileDispute(ctx context.Context, msg types.MsgFileDispute) (string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	plaintiffAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid plaintiff: %s", err)
	}

	asset, found := k.GetAsset(sdkCtx, msg.AssetID)
	if !found {
		return "", types.ErrAssetNotFound.Wrapf("asset %s not found", msg.AssetID)
	}
	if !asset.IsActive {
		return "", types.ErrAssetDelisted.Wrapf("asset %s is already delisted", msg.AssetID)
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
	dispute := types.Dispute{
		ID:           disputeID,
		AssetID:      msg.AssetID,
		Plaintiff:    msg.Creator,
		Reason:       msg.Reason,
		EvidenceHash: evidenceHash,
		Status:       types.StatusOpen,
		Remedy:       types.RemedyNone,
		Arbitrator:   "",
		ResolvedAt:   0,
	}

	if err := k.SetDispute(sdkCtx, dispute); err != nil {
		return "", err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_filed",
		sdk.NewAttribute("dispute_id", disputeID),
		sdk.NewAttribute("asset_id", msg.AssetID),
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

	dispute, found := k.GetDispute(sdkCtx, msg.DisputeID)
	if !found {
		return types.ErrDisputeNotFound.Wrapf("dispute %s not found", msg.DisputeID)
	}
	if dispute.Status != types.StatusOpen {
		return types.ErrDisputeNotOpen.Wrapf("dispute %s is %s, not OPEN", msg.DisputeID, dispute.Status)
	}

	// Execute remedy.
	switch msg.Remedy {
	case types.RemedyDelist:
		asset, found := k.GetAsset(sdkCtx, dispute.AssetID)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetID)
		}
		asset.IsActive = false
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	case types.RemedyTransfer:
		// Details should contain the new owner address.
		if len(msg.Details) == 0 {
			return types.ErrInvalidParams.Wrap("transfer remedy requires new_owner in details")
		}
		newOwner := string(msg.Details)
		if _, err := sdk.AccAddressFromBech32(newOwner); err != nil {
			return types.ErrInvalidAddress.Wrapf("invalid new owner: %s", err)
		}
		asset, found := k.GetAsset(sdkCtx, dispute.AssetID)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetID)
		}
		asset.Owner = newOwner
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}
		k.setAssetOwnerIndex(sdkCtx, newOwner, asset.ID)

	case types.RemedyRightsCorrection:
		// Details should contain the new rights type as a single byte.
		if len(msg.Details) == 0 {
			return types.ErrInvalidParams.Wrap("rights_correction remedy requires new rights_type in details")
		}
		newRightsType := types.RightsType(msg.Details[0])
		if newRightsType < types.RightsOriginal || newRightsType > types.RightsCollection {
			return types.ErrInvalidRightsType.Wrapf("invalid new rights_type: %d", newRightsType)
		}
		asset, found := k.GetAsset(sdkCtx, dispute.AssetID)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetID)
		}
		asset.RightsType = newRightsType
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	case types.RemedyShareAdjustment:
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
		asset, found := k.GetAsset(sdkCtx, dispute.AssetID)
		if !found {
			return types.ErrAssetNotFound.Wrapf("asset %s not found", dispute.AssetID)
		}
		asset.CoCreators = newCoCreators
		if err := k.SetAsset(sdkCtx, asset); err != nil {
			return err
		}

	default:
		return types.ErrInvalidParams.Wrapf("unknown remedy: %d", msg.Remedy)
	}

	// Update dispute.
	dispute.Status = types.StatusResolved
	dispute.Remedy = msg.Remedy
	dispute.Arbitrator = msg.Creator
	dispute.ResolvedAt = sdkCtx.BlockTime().Unix()
	if err := k.SetDispute(sdkCtx, dispute); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_resolved",
		sdk.NewAttribute("dispute_id", msg.DisputeID),
		sdk.NewAttribute("remedy", msg.Remedy.String()),
		sdk.NewAttribute("arbitrator", msg.Creator),
	))

	return nil
}

// IterateAllAssets iterates over all data assets and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllAssets(ctx sdk.Context, cb func(asset types.DataAsset) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DataAssetKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var asset types.DataAsset
		if err := json.Unmarshal(iter.Value(), &asset); err != nil {
			continue
		}
		if cb(asset) {
			break
		}
	}
}
