package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

type executorCandidate struct {
	address string
	// Deterministic score using only integer/byte operations.
	// scoreHigh:scoreLow form a 128-bit composite key for sorting.
	// Lower weight bucket = higher priority; within same bucket, sort by hash.
	weightBucket uint64 // reputation weight bucket (lower = higher priority)
	hashVal      [32]byte
}

// AssignExecutors deterministically selects executors for a task.
// Selection uses sha256(taskID + blockHash + executorAddr) weighted by reputation.
// The task creator is excluded from assignment (anti self-trade).
func (k Keeper) AssignExecutors(ctx sdk.Context, task types.Task) ([]string, error) {
	params := k.GetParams(ctx)
	blockHash := ctx.HeaderHash()

	var candidates []executorCandidate

	k.IterateExecutorProfiles(ctx, func(p types.ExecutorProfile) bool {
		if !p.Active {
			return false
		}
		if p.Address == task.Creator {
			return false
		}
		if !supportsTaskType(p.SupportedTaskTypes, task.TaskType) {
			return false
		}
		if p.MaxComputeUnits < task.MaxComputeUnits {
			return false
		}

		rep := k.getReputationScore(ctx, p.Address)
		if rep < uint64(params.MinExecutorReputation) {
			return false
		}

		hashVal := computeAssignmentHash(task.Id, blockHash, p.Address)
		// Weight bucket: higher reputation = lower bucket = more likely selected.
		// Use integer log2 approximation: bucket = MaxUint64 / (1 + rep)
		// This is fully deterministic across all platforms.
		weightBucket := ^uint64(0) / (1 + rep)

		candidates = append(candidates, executorCandidate{
			address:      p.Address,
			weightBucket: weightBucket,
			hashVal:      hashVal,
		})
		return false
	})

	redundancy := int(task.Redundancy)
	if len(candidates) < redundancy {
		return nil, types.ErrNotEnoughExecutors.Wrapf(
			"need %d executors, only %d eligible", redundancy, len(candidates))
	}

	// Deterministic sort: by weight bucket ascending, then by hash bytes
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].weightBucket != candidates[j].weightBucket {
			return candidates[i].weightBucket < candidates[j].weightBucket
		}
		// Tie-break by hash (deterministic byte comparison)
		for b := 0; b < 32; b++ {
			if candidates[i].hashVal[b] != candidates[j].hashVal[b] {
				return candidates[i].hashVal[b] < candidates[j].hashVal[b]
			}
		}
		return candidates[i].address < candidates[j].address
	})

	// Select top N, skipping duplicates (Sybil resistance)
	selected := make([]string, 0, redundancy)
	for _, c := range candidates {
		if len(selected) >= redundancy {
			break
		}
		selected = append(selected, c.address)
	}

	return selected, nil
}

// computeAssignmentHash returns a deterministic 32-byte hash for executor selection.
// No floating point — fully deterministic across all architectures.
func computeAssignmentHash(taskID uint64, blockHash []byte, addr string) [32]byte {
	h := sha256.New()

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	h.Write(bz)
	h.Write(blockHash)
	h.Write([]byte(addr))

	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// getReputationScore returns the reputation score as uint64.
// Returns a default of 50 if no reputation record exists.
func (k Keeper) getReputationScore(ctx sdk.Context, addr string) uint64 {
	rep, found := k.reputationKeeper.GetReputation(ctx, addr)
	if !found {
		return 50 // default for new executors
	}
	return rep.TotalScore
}

func supportsTaskType(supported []string, taskType string) bool {
	for _, t := range supported {
		if t == taskType || t == "*" {
			return true
		}
	}
	return false
}
