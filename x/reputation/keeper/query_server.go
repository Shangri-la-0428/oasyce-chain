package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/reputation/types"
)

var _ types.QueryServer = queryServer{}

// queryServer implements the reputation QueryServer interface.
type queryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the reputation QueryServer.
func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Reputation returns the reputation score for a given address.
func (q queryServer) Reputation(goCtx context.Context, req *types.QueryReputationRequest) (*types.QueryReputationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	score, found := q.Keeper.GetReputation(ctx, req.Address)
	if !found {
		// Return default zero reputation for new addresses.
		score = types.ReputationScore{
			Address:     req.Address,
			LastUpdated: ctx.BlockTime(),
		}
	}
	return &types.QueryReputationResponse{Reputation: score}, nil
}

// Feedback returns all feedbacks for a given invocation.
func (q queryServer) Feedback(goCtx context.Context, req *types.QueryFeedbackRequest) (*types.QueryFeedbackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	feedbacks := q.Keeper.GetFeedbacksByTarget(ctx, req.InvocationId)
	return &types.QueryFeedbackResponse{Feedbacks: feedbacks}, nil
}

// ReputationParams returns the module parameters.
func (q queryServer) ReputationParams(goCtx context.Context, _ *types.QueryReputationParamsRequest) (*types.QueryReputationParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryReputationParamsResponse{Params: params}, nil
}

// Leaderboard returns the top reputation scores.
func (q queryServer) Leaderboard(goCtx context.Context, req *types.QueryLeaderboardRequest) (*types.QueryLeaderboardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var allScores []types.ReputationScore
	q.Keeper.IterateAllScores(ctx, func(score types.ReputationScore) bool {
		allScores = append(allScores, score)
		return false
	})

	// Sort by TotalScore descending.
	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].TotalScore > allScores[j].TotalScore
	})

	// Apply pagination limit if set via the page request, otherwise default to 100.
	limit := uint64(100)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if uint64(len(allScores)) > limit {
		allScores = allScores[:limit]
	}

	return &types.QueryLeaderboardResponse{Scores: allScores}, nil
}
