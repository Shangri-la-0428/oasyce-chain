package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

var _ types.QueryServer = queryServer{}

// queryServer implements the settlement QueryServer interface.
type queryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the settlement QueryServer.
func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Escrow returns a single escrow by ID.
func (q queryServer) Escrow(goCtx context.Context, req *types.QueryEscrowRequest) (*types.QueryEscrowResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	escrow, found := q.Keeper.GetEscrow(ctx, req.EscrowId)
	if !found {
		return nil, types.ErrEscrowNotFound.Wrapf("escrow %s not found", req.EscrowId)
	}
	return &types.QueryEscrowResponse{Escrow: escrow}, nil
}

// EscrowsByCreator returns all escrows created by an address.
func (q queryServer) EscrowsByCreator(goCtx context.Context, req *types.QueryEscrowsByCreatorRequest) (*types.QueryEscrowsByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	escrows := q.Keeper.GetEscrowsByCreator(ctx, req.Creator)
	return &types.QueryEscrowsByCreatorResponse{Escrows: escrows}, nil
}

// BondingCurvePrice returns the current bonding curve price for an asset.
// First checks settlement store; falls back to datarights store (the authoritative source
// for assets traded via datarights.BuyShares).
func (q queryServer) BondingCurvePrice(goCtx context.Context, req *types.QueryBondingCurvePriceRequest) (*types.QueryBondingCurvePriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	state, found := q.Keeper.GetBondingCurveState(ctx, req.AssetId)
	if !found && q.Keeper.datarightsKeeper != nil {
		// Reconstruct from datarights store (authoritative for data asset trades).
		totalShares, assetFound := q.Keeper.datarightsKeeper.GetAssetTotalShares(ctx, req.AssetId)
		if !assetFound {
			return nil, types.ErrBondingCurveNotFound.Wrapf("asset %s", req.AssetId)
		}
		reserve := q.Keeper.datarightsKeeper.GetAssetReserve(ctx, req.AssetId)
		denom := q.Keeper.datarightsKeeper.GetAssetReserveDenom(ctx, req.AssetId)
		state = types.BondingCurveState{
			AssetId:      req.AssetId,
			TotalShares:  totalShares,
			Reserve:      reserve,
			PriceFactor:  math.LegacyNewDec(1),
			ReserveDenom: denom,
		}
		found = true
	}
	if !found {
		return nil, types.ErrBondingCurveNotFound.Wrapf("asset %s", req.AssetId)
	}

	denom := state.ReserveDenom
	if denom == "" {
		denom = "uoas"
	}

	// Compute spot price.
	var price math.Int
	if state.TotalShares.IsZero() || state.Reserve.IsZero() {
		price = types.InitialPrice.TruncateInt()
		if price.IsZero() {
			price = math.OneInt()
		}
	} else {
		reserveDec := math.LegacyNewDecFromInt(state.Reserve)
		supplyDec := math.LegacyNewDecFromInt(state.TotalShares)
		spotPrice := reserveDec.Quo(supplyDec.Mul(types.ReserveRatio))
		price = spotPrice.TruncateInt()
		if price.IsZero() {
			price = math.OneInt()
		}
	}

	return &types.QueryBondingCurvePriceResponse{
		CurrentPrice: sdk.NewCoin(denom, price),
		State:        state,
	}, nil
}

// SettlementParams returns the settlement module parameters.
func (q queryServer) SettlementParams(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
