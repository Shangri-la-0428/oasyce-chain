package keeper

import (
	"context"

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
func (q queryServer) BondingCurvePrice(goCtx context.Context, req *types.QueryBondingCurvePriceRequest) (*types.QueryBondingCurvePriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	price, err := q.Keeper.GetPrice(ctx, req.AssetId)
	if err != nil {
		return nil, err
	}
	state, found := q.Keeper.GetBondingCurveState(ctx, req.AssetId)
	if !found {
		return nil, types.ErrBondingCurveNotFound.Wrapf("asset %s", req.AssetId)
	}
	return &types.QueryBondingCurvePriceResponse{
		CurrentPrice: sdk.NewCoin("uoas", price),
		State:        state,
	}, nil
}
