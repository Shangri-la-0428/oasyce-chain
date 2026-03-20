package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

// ---- Commitment CRUD ----

func (k Keeper) GetCommitment(ctx sdk.Context, taskID uint64, executor string) (types.Commitment, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CommitmentKey(taskID, executor))
	if bz == nil {
		return types.Commitment{}, false
	}
	var c types.Commitment
	if err := k.cdc.Unmarshal(bz, &c); err != nil {
		return types.Commitment{}, false
	}
	return c, true
}

func (k Keeper) SetCommitment(ctx sdk.Context, c types.Commitment) error {
	bz, err := k.cdc.Marshal(&c)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CommitmentKey(c.TaskId, c.Executor), bz)
	return nil
}

func (k Keeper) CountCommitments(ctx sdk.Context, taskID uint64) int {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.CommitmentIteratorPrefix(taskID))
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

// ---- Result CRUD ----

func (k Keeper) GetResult(ctx sdk.Context, taskID uint64, executor string) (types.Result, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ResultKey(taskID, executor))
	if bz == nil {
		return types.Result{}, false
	}
	var r types.Result
	if err := k.cdc.Unmarshal(bz, &r); err != nil {
		return types.Result{}, false
	}
	return r, true
}

func (k Keeper) SetResult(ctx sdk.Context, r types.Result) error {
	bz, err := k.cdc.Marshal(&r)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ResultKey(r.TaskId, r.Executor), bz)
	return nil
}

func (k Keeper) GetAllResults(ctx sdk.Context, taskID uint64) []types.Result {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ResultIteratorPrefix(taskID))
	defer iter.Close()

	var results []types.Result
	for ; iter.Valid(); iter.Next() {
		var r types.Result
		if err := k.cdc.Unmarshal(iter.Value(), &r); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results
}

func (k Keeper) CountResults(ctx sdk.Context, taskID uint64) int {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ResultIteratorPrefix(taskID))
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

// ---- Executor Profile CRUD ----

func (k Keeper) GetExecutorProfile(ctx sdk.Context, addr string) (types.ExecutorProfile, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ExecutorProfileKey(addr))
	if bz == nil {
		return types.ExecutorProfile{}, false
	}
	var p types.ExecutorProfile
	if err := k.cdc.Unmarshal(bz, &p); err != nil {
		return types.ExecutorProfile{}, false
	}
	return p, true
}

func (k Keeper) SetExecutorProfile(ctx sdk.Context, p types.ExecutorProfile) error {
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ExecutorProfileKey(p.Address), bz)
	return nil
}

func (k Keeper) IterateExecutorProfiles(ctx sdk.Context, cb func(types.ExecutorProfile) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ExecutorProfilePrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var p types.ExecutorProfile
		if err := k.cdc.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		if cb(p) {
			break
		}
	}
}

// ---- Epoch Stats ----

func (k Keeper) GetEpochStats(ctx sdk.Context, epoch uint64) (types.EpochStats, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EpochStatsKey(epoch))
	if bz == nil {
		return types.EpochStats{}, false
	}
	var s types.EpochStats
	if err := k.cdc.Unmarshal(bz, &s); err != nil {
		return types.EpochStats{}, false
	}
	return s, true
}

func (k Keeper) SetEpochStats(ctx sdk.Context, s types.EpochStats) error {
	bz, err := k.cdc.Marshal(&s)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.EpochStatsKey(s.Epoch), bz)
	return nil
}
