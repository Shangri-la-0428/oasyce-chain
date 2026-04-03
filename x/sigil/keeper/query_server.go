package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Sigil(goCtx context.Context, req *types.QuerySigilRequest) (*types.QuerySigilResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.SigilId == "" {
		return nil, types.ErrInvalidSigilID.Wrap("sigil_id cannot be empty")
	}

	s, found := q.Keeper.GetSigil(ctx, req.SigilId)
	if !found {
		return nil, types.ErrSigilNotFound.Wrapf("sigil %s not found", req.SigilId)
	}

	return &types.QuerySigilResponse{Sigil: s}, nil
}

func (q queryServer) Bond(goCtx context.Context, req *types.QueryBondRequest) (*types.QueryBondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.BondId == "" {
		return nil, types.ErrInvalidBondID.Wrap("bond_id cannot be empty")
	}

	b, found := q.Keeper.GetBond(ctx, req.BondId)
	if !found {
		return nil, types.ErrBondNotFound.Wrapf("bond %s not found", req.BondId)
	}

	return &types.QueryBondResponse{Bond: b}, nil
}

func (q queryServer) BondsBySigil(goCtx context.Context, req *types.QueryBondsBySigilRequest) (*types.QueryBondsBySigilResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.SigilId == "" {
		return nil, types.ErrInvalidSigilID.Wrap("sigil_id cannot be empty")
	}

	bonds := q.Keeper.GetBondsBySigil(ctx, req.SigilId)
	return &types.QueryBondsBySigilResponse{Bonds: bonds}, nil
}

func (q queryServer) Lineage(goCtx context.Context, req *types.QueryLineageRequest) (*types.QueryLineageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.SigilId == "" {
		return nil, types.ErrInvalidSigilID.Wrap("sigil_id cannot be empty")
	}

	children := q.Keeper.GetChildren(ctx, req.SigilId)
	return &types.QueryLineageResponse{Children: children}, nil
}

func (q queryServer) ActiveCount(goCtx context.Context, _ *types.QueryActiveCountRequest) (*types.QueryActiveCountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	count := q.Keeper.GetActiveCount(ctx)
	return &types.QueryActiveCountResponse{Count: count}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: p}, nil
}
