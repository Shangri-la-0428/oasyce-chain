package keeper

import (
	"encoding/binary"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

// ---- Task CRUD ----

func (k Keeper) GetTask(ctx sdk.Context, taskID uint64) (types.Task, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TaskKey(taskID))
	if bz == nil {
		return types.Task{}, false
	}
	var task types.Task
	if err := k.cdc.Unmarshal(bz, &task); err != nil {
		return types.Task{}, false
	}
	return task, true
}

func (k Keeper) SetTask(ctx sdk.Context, task types.Task) error {
	bz, err := k.cdc.Marshal(&task)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.TaskKey(task.Id), bz)
	return nil
}

// setTaskWithIndexes persists a task and maintains all secondary indexes.
func (k Keeper) setTaskWithIndexes(ctx sdk.Context, task types.Task, oldStatus types.TaskStatus) error {
	store := ctx.KVStore(k.storeKey)

	// Remove old status index if status changed
	if oldStatus != task.Status {
		store.Delete(types.StatusIndexKey(oldStatus, task.Id))
	}

	// Set primary record
	if err := k.SetTask(ctx, task); err != nil {
		return err
	}

	// Status index
	store.Set(types.StatusIndexKey(task.Status, task.Id), []byte{})

	// Creator index (set once)
	store.Set(types.CreatorIndexKey(task.Creator, task.Id), []byte{})

	// Executor indexes
	for _, exec := range task.AssignedExecutors {
		store.Set(types.ExecutorIndexKey(exec, task.Id), []byte{})
	}

	return nil
}

// RebuildTaskIndexes rebuilds all secondary indexes for a task (used during InitGenesis).
func (k Keeper) RebuildTaskIndexes(ctx sdk.Context, task types.Task) {
	store := ctx.KVStore(k.storeKey)

	// Status index
	store.Set(types.StatusIndexKey(task.Status, task.Id), []byte{})

	// Creator index
	store.Set(types.CreatorIndexKey(task.Creator, task.Id), []byte{})

	// Executor indexes
	for _, exec := range task.AssignedExecutors {
		store.Set(types.ExecutorIndexKey(exec, task.Id), []byte{})
	}

	// Expiry index — needed by BeginBlocker to find timed-out tasks
	if !types.IsTerminalStatus(task.Status) && task.TimeoutHeight > 0 {
		store.Set(types.ExpiryIndexKey(task.TimeoutHeight, task.Id), []byte{})
	}

	// Reveal expiry index — needed by BeginBlocker for reveal timeout
	if task.Status == types.TASK_STATUS_REVEALING && task.RevealTimeoutHeight > 0 {
		store.Set(types.RevealExpiryIndexKey(task.RevealTimeoutHeight, task.Id), []byte{})
	}
}

// ---- Iteration ----

func (k Keeper) IterateTasksByStatus(ctx sdk.Context, status types.TaskStatus, cb func(types.Task) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.StatusIndexIteratorPrefix(status))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Key: prefix(1) + status(1) + taskID(8)
		key := iter.Key()
		if len(key) < 10 {
			continue
		}
		taskID := binary.BigEndian.Uint64(key[2:10])
		task, found := k.GetTask(ctx, taskID)
		if !found {
			continue
		}
		if cb(task) {
			break
		}
	}
}

func (k Keeper) IterateTasksByCreator(ctx sdk.Context, creator string, cb func(types.Task) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.CreatorIndexIteratorPrefix(creator))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// Last 8 bytes are taskID
		if len(key) < 8 {
			continue
		}
		taskID := binary.BigEndian.Uint64(key[len(key)-8:])
		task, found := k.GetTask(ctx, taskID)
		if !found {
			continue
		}
		if cb(task) {
			break
		}
	}
}

func (k Keeper) IterateTasksByExecutor(ctx sdk.Context, executor string, cb func(types.Task) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ExecutorIndexIteratorPrefix(executor))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 8 {
			continue
		}
		taskID := binary.BigEndian.Uint64(key[len(key)-8:])
		task, found := k.GetTask(ctx, taskID)
		if !found {
			continue
		}
		if cb(task) {
			break
		}
	}
}

func (k Keeper) IterateAllTasks(ctx sdk.Context, cb func(types.Task) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.TaskKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var task types.Task
		if err := k.cdc.Unmarshal(iter.Value(), &task); err != nil {
			continue
		}
		if cb(task) {
			break
		}
	}
}
