package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

// ---------------------------------------------------------------------------
// BondingCurveState CRUD
// ---------------------------------------------------------------------------

// GetBondingCurveState retrieves the bonding curve state for an asset.
func (k Keeper) GetBondingCurveState(ctx sdk.Context, assetID string) (types.BondingCurveState, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.BondingCurveKey(assetID))
	if bz == nil {
		return types.BondingCurveState{}, false
	}
	var state types.BondingCurveState
	if err := json.Unmarshal(bz, &state); err != nil {
		return types.BondingCurveState{}, false
	}
	return state, true
}

// SetBondingCurveState persists a bonding curve state.
func (k Keeper) SetBondingCurveState(ctx sdk.Context, state types.BondingCurveState) error {
	bz, err := json.Marshal(state)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.BondingCurveKey(state.AssetID), bz)
	return nil
}

// IterateAllBondingCurves iterates over all bonding curve states and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllBondingCurves(ctx sdk.Context, cb func(state types.BondingCurveState) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.BondingCurvePrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var state types.BondingCurveState
		if err := json.Unmarshal(iter.Value(), &state); err != nil {
			continue
		}
		if cb(state) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Bonding Curve Pricing
// ---------------------------------------------------------------------------

// GetPrice calculates the current price per share for an asset.
// Formula: price = reserve * priceFactor / totalShares
// If totalShares is zero, returns a base price of 1 uoas.
func (k Keeper) GetPrice(ctx sdk.Context, assetID string) (math.Int, error) {
	state, found := k.GetBondingCurveState(ctx, assetID)
	if !found {
		return math.Int{}, types.ErrBondingCurveNotFound.Wrapf("asset %s", assetID)
	}

	if state.TotalShares.IsZero() {
		// Base price when no shares exist: 1 unit.
		return math.NewInt(1), nil
	}

	// price = reserve * priceFactor / totalShares (truncated to integer).
	reserveDec := math.LegacyNewDecFromInt(state.Reserve)
	totalSharesDec := math.LegacyNewDecFromInt(state.TotalShares)
	price := reserveDec.Mul(state.PriceFactor).Quo(totalSharesDec).TruncateInt()

	if price.IsZero() {
		price = math.NewInt(1) // Floor price: at least 1 unit.
	}

	return price, nil
}

// shareRateBps returns the diminishing share rate in basis points based on buyer index.
// Per spec section 13:
//
//	1st buyer (index 0): 100% = 10000 bps
//	2nd buyer (index 1):  80% =  8000 bps
//	3rd buyer (index 2):  60% =  6000 bps
//	4th+ buyer:           40% =  4000 bps
func shareRateBps(buyerIndex uint32) math.Int {
	switch {
	case buyerIndex == 0:
		return math.NewInt(10000)
	case buyerIndex == 1:
		return math.NewInt(8000)
	case buyerIndex == 2:
		return math.NewInt(6000)
	default:
		return math.NewInt(4000)
	}
}

// BuyShares calculates and mints shares for a buyer on a bonding curve.
// The payment amount is added to the reserve, and shares are minted with
// diminishing returns based on the buyer index.
func (k Keeper) BuyShares(ctx sdk.Context, assetID string, buyer string, paymentAmount math.Int) (math.Int, error) {
	buyerAddr, err := sdk.AccAddressFromBech32(buyer)
	if err != nil {
		return math.Int{}, types.ErrInvalidAddress.Wrapf("invalid buyer: %s", err)
	}

	if paymentAmount.IsZero() || paymentAmount.IsNegative() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment must be positive")
	}

	state, found := k.GetBondingCurveState(ctx, assetID)
	if !found {
		// Initialize a new bonding curve for this asset.
		state = types.BondingCurveState{
			AssetID:     assetID,
			TotalShares: math.ZeroInt(),
			Reserve:     math.ZeroInt(),
			PriceFactor: math.LegacyNewDec(1), // default factor = 1.0
			BuyerCount:  0,
		}
	}

	// Calculate shares with diminishing returns.
	rate := shareRateBps(state.BuyerCount)
	sharesMinted := paymentAmount.Mul(rate).Quo(math.NewInt(10000))

	if sharesMinted.IsZero() {
		return math.Int{}, types.ErrInsufficientFunds.Wrap("payment too small to mint shares")
	}

	// Transfer payment from buyer to module.
	coins := sdk.NewCoins(sdk.NewCoin("uoas", paymentAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, buyerAddr, types.ModuleName, coins); err != nil {
		return math.Int{}, types.ErrInsufficientFunds.Wrapf("failed to collect payment: %s", err)
	}

	// Update state.
	state.TotalShares = state.TotalShares.Add(sharesMinted)
	state.Reserve = state.Reserve.Add(paymentAmount)
	state.BuyerCount++

	if err := k.SetBondingCurveState(ctx, state); err != nil {
		return math.Int{}, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"shares_bought",
		sdk.NewAttribute("asset_id", assetID),
		sdk.NewAttribute("buyer", buyer),
		sdk.NewAttribute("payment", paymentAmount.String()),
		sdk.NewAttribute("shares_minted", sharesMinted.String()),
		sdk.NewAttribute("buyer_index", fmt.Sprintf("%d", state.BuyerCount-1)),
	))

	return sharesMinted, nil
}
