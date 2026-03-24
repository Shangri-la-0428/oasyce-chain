package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/onboarding/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Registration(ctx context.Context, req *types.QueryRegistrationRequest) (*types.QueryRegistrationResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	reg, found := q.Keeper.GetRegistration(sdkCtx, req.Address)
	if !found {
		return nil, types.ErrRegistrationNotFound.Wrapf("registration for %s not found", req.Address)
	}
	return &types.QueryRegistrationResponse{Registration: reg}, nil
}

func (q queryServer) Debt(ctx context.Context, req *types.QueryDebtRequest) (*types.QueryDebtResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	reg, found := q.Keeper.GetRegistration(sdkCtx, req.Address)
	if !found {
		return nil, types.ErrRegistrationNotFound.Wrapf("registration for %s not found", req.Address)
	}
	return &types.QueryDebtResponse{Registration: reg}, nil
}

func (q queryServer) OnboardingParams(ctx context.Context, _ *types.QueryOnboardingParamsRequest) (*types.QueryOnboardingParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := q.Keeper.GetParams(sdkCtx)
	return &types.QueryOnboardingParamsResponse{Params: params}, nil
}
