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

// Earnings queries the total earnings for a provider across all capabilities.
func (q queryServer) Earnings(goCtx context.Context, req *types.QueryEarningsRequest) (*types.QueryEarningsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListByProvider(ctx, req.Provider)

	totalEarned := math.ZeroInt()
	var totalCalls uint64
	for _, cap := range caps {
		totalEarned = totalEarned.Add(cap.TotalEarned)
		totalCalls += cap.TotalCalls
	}

	// Return the total earned as a single coin (uoas).
	var coins []sdk.Coin
	if totalEarned.IsPositive() {
		coins = sdk.NewCoins(sdk.NewCoin("uoas", totalEarned))
	}

	return &types.QueryEarningsResponse{
		TotalEarned: coins,
		TotalCalls:  totalCalls,
	}, nil
}
