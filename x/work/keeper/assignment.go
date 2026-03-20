package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

type executorCandidate struct {
	address string
	score   float64
}

// AssignExecutors deterministically selects executors for a task.
// Selection uses sha256(taskID + blockHash + executorAddr) weighted by reputation.
// The task creator is excluded from assignment (anti self-trade).
func (k Keeper) AssignExecutors(ctx sdk.Context, task types.Task) ([]string, error) {
	params := k.GetParams(ctx)
	blockHash := ctx.HeaderHash()

	var candidates []executorCandidate

	k.IterateExecutorProfiles(ctx, func(p types.ExecutorProfile) bool {
		// Skip inactive executors
		if !p.Active {
			return false
		}

		// Skip the task creator (prevent self-assignment)
		if p.Address == task.Creator {
			return false
		}

		// Check task type support
		if !supportsTaskType(p.SupportedTaskTypes, task.TaskType) {
			return false
		}

		// Check compute capacity
		if p.MaxComputeUnits < task.MaxComputeUnits {
			return false
		}

		// Check minimum reputation
		rep := k.getReputationWeight(ctx, p.Address)
		if rep < float64(params.MinExecutorReputation) {
			return false
		}

		// Deterministic score: sha256(taskID + blockHash + addr) / weight
		score := k.computeAssignmentScore(task.Id, blockHash, p.Address, rep)
		candidates = append(candidates, executorCandidate{
			address: p.Address,
			score:   score,
		})
		return false
	})

	redundancy := int(task.Redundancy)
	if len(candidates) < redundancy {
		return nil, types.ErrNotEnoughExecutors.Wrapf(
			"need %d executors, only %d eligible", redundancy, len(candidates))
	}

	// Sort by score ascending — lower score = selected
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})

	selected := make([]string, redundancy)
	for i := 0; i < redundancy; i++ {
		selected[i] = candidates[i].address
	}

	return selected, nil
}

// computeAssignmentScore returns a deterministic pseudo-random score.
// Lower score = higher priority for selection.
func (k Keeper) computeAssignmentScore(taskID uint64, blockHash []byte, addr string, reputation float64) float64 {
	h := sha256.New()

	// taskID
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	h.Write(bz)

	// blockHash (deterministic, unknown at submission time)
	h.Write(blockHash)

	// executor address
	h.Write([]byte(addr))

	digest := h.Sum(nil)

	// Convert first 8 bytes to float64 in [0, 1)
	raw := binary.BigEndian.Uint64(digest[:8])
	randomVal := float64(raw) / float64(^uint64(0))

	// Weight: higher reputation = lower score = more likely selected
	weight := math.Log1p(reputation)
	if weight < 1.0 {
		weight = 1.0
	}

	return randomVal / weight
}

// getReputationWeight returns the reputation score as float64.
// Returns a default of 50 if no reputation record exists.
func (k Keeper) getReputationWeight(ctx sdk.Context, addr string) float64 {
	rep, found := k.reputationKeeper.GetReputation(ctx, addr)
	if !found {
		return 50.0 // default for new executors
	}
	return float64(rep.TotalScore)
}

func supportsTaskType(supported []string, taskType string) bool {
	for _, t := range supported {
		if t == taskType || t == "*" {
			return true
		}
	}
	return false
}
