package keeper

import (
	"encoding/binary"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

// BeginBlocker handles timeout expiration for tasks.
// Scans expiry indexes up to current block height, with per-block processing cap.
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	maxPerBlock := int(params.MaxTasksPerBlock)

	// 1. Expire timed-out ASSIGNED tasks (no commits received in time)
	processed := k.expireByIndex(ctx, types.ExpiryIndexPrefix, currentHeight, maxPerBlock)

	// 2. Expire timed-out REVEALING tasks (not all reveals received in time)
	remaining := maxPerBlock - processed
	if remaining > 0 {
		k.expireByIndex(ctx, types.RevealExpiryIndexPrefix, currentHeight, remaining)
	}

	return nil
}

func (k Keeper) expireByIndex(ctx sdk.Context, prefix []byte, currentHeight uint64, limit int) int {
	store := ctx.KVStore(k.storeKey)

	// Build end key: prefix + (currentHeight+1) — scans all entries <= currentHeight
	endBz := make([]byte, 8)
	binary.BigEndian.PutUint64(endBz, currentHeight+1)
	endKey := append(prefix, endBz...)

	iter := store.Iterator(prefix, endKey)
	defer iter.Close()

	processed := 0
	for ; iter.Valid() && processed < limit; iter.Next() {
		key := iter.Key()
		// Key format: prefix(1) + height(8) + taskID(8)
		if len(key) < 17 {
			continue
		}
		taskID := binary.BigEndian.Uint64(key[9:17])

		task, found := k.GetTask(ctx, taskID)
		if !found {
			store.Delete(key)
			continue
		}

		// Only expire non-terminal tasks
		if types.IsTerminalStatus(task.Status) {
			store.Delete(key)
			continue
		}

		// Try to settle if we have results, otherwise expire
		resultCount := k.CountResults(ctx, taskID)
		if resultCount > 0 && task.Status == types.TASK_STATUS_REVEALING {
			// Settle with whatever results we have
			_ = k.SettleTask(ctx, task)
		} else {
			_ = k.expireTaskTimeout(ctx, task)
		}

		// Clean up the expiry index entry
		store.Delete(key)
		processed++
	}

	return processed
}

func (k Keeper) expireTaskTimeout(ctx sdk.Context, task types.Task) error {
	// Refund bounty + deposit to creator
	creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
	if err != nil {
		return err
	}

	refundCoins := sdk.NewCoins(task.Bounty)
	if task.Deposit.IsPositive() {
		refundCoins = refundCoins.Add(task.Deposit)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, refundCoins); err != nil {
		return err
	}

	oldStatus := task.Status
	task.Status = types.TASK_STATUS_EXPIRED
	if err := k.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
		return err
	}

	// Update epoch stats
	height := uint64(ctx.BlockHeight())
	epoch := height / 1000
	stats, found := k.GetEpochStats(ctx, epoch)
	if !found {
		stats = types.EpochStats{
			Epoch:         epoch,
			TotalBounties: math.ZeroInt(),
			TotalBurned:   math.ZeroInt(),
		}
	}
	stats.TasksExpired++
	_ = k.SetEpochStats(ctx, stats)

	return nil
}

// EndBlocker handles task assignment for newly submitted tasks.
// Assigns executors to all SUBMITTED tasks using current block hash for determinism.
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	maxPerBlock := int(params.MaxTasksPerBlock)

	var tasksToAssign []types.Task
	k.IterateTasksByStatus(ctx, types.TASK_STATUS_SUBMITTED, func(task types.Task) bool {
		tasksToAssign = append(tasksToAssign, task)
		return len(tasksToAssign) >= maxPerBlock
	})

	for _, task := range tasksToAssign {
		executors, err := k.AssignExecutors(ctx, task)
		if err != nil {
			// Not enough executors — leave as SUBMITTED for next block
			continue
		}

		oldStatus := task.Status
		task.Status = types.TASK_STATUS_ASSIGNED
		task.AssignedExecutors = executors
		task.AssignHeight = uint64(ctx.BlockHeight())
		task.TimeoutHeight = uint64(ctx.BlockHeight()) + task.TimeoutHeight // convert relative to absolute

		if err := k.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
			continue
		}

		// Set expiry index for timeout
		store := ctx.KVStore(k.storeKey)
		store.Set(types.ExpiryIndexKey(task.TimeoutHeight, task.Id), []byte{})

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"task_assigned",
			sdk.NewAttribute("task_id", string(sdk.Uint64ToBigEndian(task.Id))),
			sdk.NewAttribute("executors", joinAddresses(executors)),
		))
	}

	return nil
}

func joinAddresses(addrs []string) string {
	result := ""
	for i, a := range addrs {
		if i > 0 {
			result += ","
		}
		result += a
	}
	return result
}

// TransitionToReveal moves a task from COMMITTED to REVEALING phase.
func (k Keeper) TransitionToReveal(ctx sdk.Context, task types.Task) error {
	params := k.GetParams(ctx)

	oldStatus := task.Status
	task.Status = types.TASK_STATUS_REVEALING
	task.RevealTimeoutHeight = uint64(ctx.BlockHeight()) + uint64(params.RevealBlocks)

	if err := k.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
		return err
	}

	// Set reveal expiry index
	store := ctx.KVStore(k.storeKey)
	store.Set(types.RevealExpiryIndexKey(task.RevealTimeoutHeight, task.Id), []byte{})

	// Remove the old expiry index (commit timeout)
	store.Delete(types.ExpiryIndexKey(task.TimeoutHeight, task.Id))

	return nil
}
