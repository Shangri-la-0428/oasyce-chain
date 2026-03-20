package keeper

import (
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
	if err := k.cdc.Unmarshal(bz, &state); err != nil {
		return types.BondingCurveState{}, false
	}
	return state, true
}

// SetBondingCurveState persists a bonding curve state.
func (k Keeper) SetBondingCurveState(ctx sdk.Context, state types.BondingCurveState) error {
	bz, err := k.cdc.Marshal(&state)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.BondingCurveKey(state.AssetId), bz)
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
		if err := k.cdc.Unmarshal(iter.Value(), &state); err != nil {
			continue
		}
		if cb(state) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Bancor Bonding Curve Pricing
// ---------------------------------------------------------------------------

// SpotPrice calculates the current spot price using the Bancor formula.
// Formula: price = reserve / (supply × CW)
// When supply is zero, returns the initial bootstrap price (1 uoas per token).
func (k Keeper) SpotPrice(ctx sdk.Context, assetID string) (math.LegacyDec, error) {
	state, found := k.GetBondingCurveState(ctx, assetID)
	if !found {
		return math.LegacyDec{}, types.ErrBondingCurveNotFound.Wrapf("asset %s", assetID)
	}

	if state.TotalShares.IsZero() || state.Reserve.IsZero() {
		return types.InitialPrice, nil
	}

	reserveDec := math.LegacyNewDecFromInt(state.Reserve)
	supplyDec := math.LegacyNewDecFromInt(state.TotalShares)

	// price = reserve / (supply × CW)
	return reserveDec.Quo(supplyDec.Mul(types.ReserveRatio)), nil
}

// GetPrice returns the integer spot price (backwards compatible).
func (k Keeper) GetPrice(ctx sdk.Context, assetID string) (math.Int, error) {
	price, err := k.SpotPrice(ctx, assetID)
	if err != nil {
		return math.Int{}, err
	}
	result := price.TruncateInt()
	if result.IsZero() {
		result = math.OneInt() // Floor: at least 1 uoas
	}
	return result, nil
}

// BancorBuy calculates tokens minted for a given payment using Bancor formula.
// Formula: tokens = supply × (sqrt(1 + payment/reserve) − 1)
// Bootstrap (reserve=0): tokens = payment / INITIAL_PRICE
// CW=0.5, so exponentiation becomes square root.
func BancorBuy(supply, reserve, payment math.LegacyDec) (math.LegacyDec, error) {
	if payment.IsNegative() || payment.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	// Bootstrap: first purchase seeds the pool at INITIAL_PRICE.
	if reserve.IsZero() || supply.IsZero() {
		return payment.Quo(types.InitialPrice), nil
	}

	// ratio = 1 + payment/reserve
	ratio := math.LegacyOneDec().Add(payment.Quo(reserve))

	// sqrt(ratio) — CW=0.5 means we take the square root
	sqrtRatio, err := ratio.ApproxSqrt()
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("sqrt failed: %w", err)
	}

	// tokens = supply × (sqrtRatio − 1)
	tokens := supply.Mul(sqrtRatio.Sub(math.LegacyOneDec()))
	return tokens, nil
}

// BancorSell calculates the gross payout for selling tokens back.
// Formula: payout = reserve × (1 − (1 − tokens/supply)^(1/CW))
// CW=0.5 → 1/CW=2, so (1−tokens/supply)^2
// Capped at RESERVE_SOLVENCY_CAP × reserve.
func BancorSell(supply, reserve, tokens math.LegacyDec) (math.LegacyDec, error) {
	if tokens.IsNegative() || tokens.IsZero() || supply.IsZero() || reserve.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	maxPayout := reserve.Mul(types.ReserveSolvencyCap)

	// Can't sell entire supply — cap at solvency limit.
	if tokens.GTE(supply) {
		return maxPayout, nil
	}

	// ratio = 1 − tokens/supply
	ratio := math.LegacyOneDec().Sub(tokens.Quo(supply))

	// (ratio)^2 — since 1/CW = 2
	ratioSquared := ratio.Mul(ratio)

	// payout = reserve × (1 − ratio²)
	payout := reserve.Mul(math.LegacyOneDec().Sub(ratioSquared))

	// Cap at solvency limit.
	if payout.GT(maxPayout) {
		payout = maxPayout
	}

	return payout, nil
}

// BuyShares calculates and mints shares for a buyer on a bonding curve
// using the Bancor continuous token model.
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
			AssetId:     assetID,
			TotalShares: math.ZeroInt(),
			Reserve:     math.ZeroInt(),
			PriceFactor: math.LegacyNewDec(1),
			BuyerCount:  0,
		}
	}

	// Bancor calculation using LegacyDec.
	supplyDec := math.LegacyNewDecFromInt(state.TotalShares)
	reserveDec := math.LegacyNewDecFromInt(state.Reserve)
	paymentDec := math.LegacyNewDecFromInt(paymentAmount)

	tokensDec, err := BancorBuy(supplyDec, reserveDec, paymentDec)
	if err != nil {
		return math.Int{}, fmt.Errorf("bancor buy calculation failed: %w", err)
	}

	sharesMinted := tokensDec.TruncateInt()
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
		sdk.NewAttribute("new_supply", state.TotalShares.String()),
		sdk.NewAttribute("new_reserve", state.Reserve.String()),
	))

	return sharesMinted, nil
}
