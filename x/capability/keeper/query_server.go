package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

var _ types.QueryServer = queryServer{}

// queryServer implements the proto-generated types.QueryServer interface.
type queryServer struct {
	keeper Keeper
}

// NewQueryServer returns a new QueryServer instance.
func NewQueryServer(keeper Keeper) types.QueryServer {
	return queryServer{keeper: keeper}
}

// Capability queries a single capability by ID.
func (q queryServer) Capability(goCtx context.Context, req *types.QueryCapabilityRequest) (*types.QueryCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cap, err := q.keeper.GetCapability(ctx, req.CapabilityId)
	if err != nil {
		return nil, err
	}
	return &types.QueryCapabilityResponse{Capability: cap}, nil
}

// Capabilities queries all capabilities, optionally filtered by tag.
func (q queryServer) Capabilities(goCtx context.Context, req *types.QueryCapabilitiesRequest) (*types.QueryCapabilitiesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListCapabilities(ctx, req.Tag)
	return &types.QueryCapabilitiesResponse{Capabilities: caps}, nil
}

// CapabilitiesByProvider queries all capabilities for a given provider.
func (q queryServer) CapabilitiesByProvider(goCtx context.Context, req *types.QueryCapabilitiesByProviderRequest) (*types.QueryCapabilitiesByProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListByProvider(ctx, req.Provider)
	return &types.QueryCapabilitiesByProviderResponse{Capabilities: caps}, nil
}

// Invocation queries a single invocation by ID.
func (q queryServer) Invocation(goCtx context.Context, req *types.QueryInvocationRequest) (*types.QueryInvocationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	inv, err := q.keeper.GetInvocation(ctx, req.InvocationId)
	if err != nil {
		return nil, err
	}
	return &types.QueryInvocationResponse{Invocation: inv}, nil
}

// CapabilityParams returns the module parameters.
func (q queryServer) CapabilityParams(goCtx context.Context, _ *types.QueryCapabilityParamsRequest) (*types.QueryCapabilityParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.keeper.GetParams(ctx)
	return &types.QueryCapabilityParamsResponse{Params: params}, nil
}

// Earnings queries the total earnings for a provider across all capabilities.
func (q queryServer) Earnings(goCtx context.Context, req *types.QueryEarningsRequest) (*types.QueryEarningsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListByProvider(ctx, req.Provider)

	// Aggregate earnings per denomination from each capability's PricePerCall.
	denomTotals := make(map[string]math.Int)
	var totalCalls uint64
	for _, cap := range caps {
		totalCalls += cap.TotalCalls
		if cap.TotalEarned.IsPositive() {
			denom := cap.PricePerCall.Denom
			if denom == "" {
				denom = "uoas" // fallback for capabilities without explicit denom
			}
			if existing, ok := denomTotals[denom]; ok {
				denomTotals[denom] = existing.Add(cap.TotalEarned)
			} else {
				denomTotals[denom] = cap.TotalEarned
			}
		}
	}

	var coins []sdk.Coin
	for denom, amount := range denomTotals {
		coins = append(coins, sdk.NewCoin(denom, amount))
	}
	if len(coins) > 0 {
		coins = sdk.NewCoins(coins...) // sort and deduplicate
	}

	return &types.QueryEarningsResponse{
		TotalEarned: coins,
		TotalCalls:  totalCalls,
	}, nil
}
