package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Policy(ctx context.Context, req *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	policy, found := q.Keeper.GetPolicy(sdkCtx, req.Principal)
	if !found {
		return nil, types.ErrPolicyNotFound.Wrapf("no policy for principal %s", req.Principal)
	}
	return &types.QueryPolicyResponse{Policy: policy}, nil
}

func (q queryServer) Delegates(ctx context.Context, req *types.QueryDelegatesRequest) (*types.QueryDelegatesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	delegates := q.Keeper.ListDelegates(sdkCtx, req.Principal)
	if delegates == nil {
		delegates = []types.DelegateRecord{}
	}
	return &types.QueryDelegatesResponse{Delegates: delegates}, nil
}

func (q queryServer) Spend(ctx context.Context, req *types.QuerySpendRequest) (*types.QuerySpendResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	policy, found := q.Keeper.GetPolicy(sdkCtx, req.Principal)
	if !found {
		return nil, types.ErrPolicyNotFound.Wrapf("no policy for principal %s", req.Principal)
	}

	window := q.Keeper.GetOrResetWindow(sdkCtx, req.Principal, policy.WindowSeconds, policy.PerTxLimit.Denom)
	return &types.QuerySpendResponse{Window: window}, nil
}

func (q queryServer) Principal(ctx context.Context, req *types.QueryPrincipalRequest) (*types.QueryPrincipalResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	rec, found := q.Keeper.GetDelegate(sdkCtx, req.Delegate)
	if !found {
		return nil, types.ErrDelegateNotFound.Wrapf("delegate %s not enrolled", req.Delegate)
	}
	return &types.QueryPrincipalResponse{Principal: rec.Principal, Record: rec}, nil
}
