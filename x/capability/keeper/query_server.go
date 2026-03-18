package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

// QueryServer implements the capability module's Query service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer returns a new QueryServer instance.
func NewQueryServer(keeper Keeper) QueryServer {
	return QueryServer{keeper: keeper}
}

// QueryCapability returns a single capability by ID.
func (q QueryServer) QueryCapability(goCtx context.Context, req *types.QueryCapabilityRequest) (*types.QueryCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cap, err := q.keeper.GetCapability(ctx, req.CapabilityID)
	if err != nil {
		return nil, err
	}
	return &types.QueryCapabilityResponse{Capability: cap}, nil
}

// QueryCapabilities returns all capabilities, optionally filtered by tag.
func (q QueryServer) QueryCapabilities(goCtx context.Context, req *types.QueryCapabilitiesRequest) (*types.QueryCapabilitiesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListCapabilities(ctx, req.Tag)
	return &types.QueryCapabilitiesResponse{Capabilities: caps}, nil
}

// QueryCapabilitiesByProvider returns all capabilities for a given provider.
func (q QueryServer) QueryCapabilitiesByProvider(goCtx context.Context, req *types.QueryCapabilitiesByProviderRequest) (*types.QueryCapabilitiesByProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListByProvider(ctx, req.Provider)
	return &types.QueryCapabilitiesByProviderResponse{Capabilities: caps}, nil
}

// QueryEarnings returns the total earnings for a provider across all capabilities.
func (q QueryServer) QueryEarnings(goCtx context.Context, req *types.QueryEarningsRequest) (*types.QueryEarningsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	caps := q.keeper.ListByProvider(ctx, req.Provider)

	totalEarned := math.ZeroInt()
	var totalCalls uint64
	for _, cap := range caps {
		totalEarned = totalEarned.Add(cap.TotalEarned)
		totalCalls += cap.TotalCalls
	}

	// Return the total earned as a single coin (uoas).
	var coins sdk.Coins
	if totalEarned.IsPositive() {
		coins = sdk.NewCoins(sdk.NewCoin("uoas", totalEarned))
	}

	return &types.QueryEarningsResponse{
		TotalEarned: coins,
		TotalCalls:  totalCalls,
	}, nil
}
