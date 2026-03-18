package keeper

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/reputation/types"
)

// QueryServer implements the reputation module's query service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer returns a new QueryServer instance.
func NewQueryServer(k Keeper) QueryServer {
	return QueryServer{keeper: k}
}

// QueryReputation returns the reputation score for a given address.
func (qs QueryServer) QueryReputation(ctx sdk.Context, address string) (*types.ReputationScore, error) {
	score, found := qs.keeper.GetReputation(ctx, address)
	if !found {
		return nil, types.ErrInvalidAddress.Wrapf("no reputation found for %s", address)
	}
	return &score, nil
}

// QueryFeedback returns all feedbacks for a given target address.
func (qs QueryServer) QueryFeedback(ctx sdk.Context, target string) ([]types.Feedback, error) {
	feedbacks := qs.keeper.GetFeedbacksByTarget(ctx, target)
	return feedbacks, nil
}

// QueryLeaderboard returns the top N reputation scores.
func (qs QueryServer) QueryLeaderboard(ctx sdk.Context, limit uint64) ([]types.ReputationScore, error) {
	if limit == 0 {
		limit = 100
	}

	var allScores []types.ReputationScore
	qs.keeper.IterateAllScores(ctx, func(score types.ReputationScore) bool {
		allScores = append(allScores, score)
		return false
	})

	// Sort by TotalScore descending.
	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].TotalScore.GT(allScores[j].TotalScore)
	})

	if uint64(len(allScores)) > limit {
		allScores = allScores[:limit]
	}

	return allScores, nil
}
