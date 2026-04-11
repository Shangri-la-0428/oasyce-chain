package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

// MaxPulseHeight returns the most recent activity height across all dimensions.
// This is max(LastActiveHeight, max(DimensionPulses values)).
func MaxPulseHeight(s types.Sigil) int64 {
	h := s.LastActiveHeight
	for _, v := range s.DimensionPulses {
		if v > h {
			h = v
		}
	}
	return h
}

func (k Keeper) effectiveActivityHeight(s types.Sigil) int64 {
	return MaxPulseHeight(s)
}

func (k Keeper) clearSigilIndexes(ctx sdk.Context, s types.Sigil) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SigilByStatusKey(types.SigilStatus(s.Status), s.SigilId))
	h := k.effectiveActivityHeight(s)
	switch types.SigilStatus(s.Status) {
	case types.SigilStatusActive:
		store.Delete(types.LivenessIndexKey(h, s.SigilId))
	case types.SigilStatusDormant:
		store.Delete(types.DormantLivenessIndexKey(h, s.SigilId))
	}
}

func (k Keeper) writeSigilIndexes(ctx sdk.Context, s types.Sigil) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SigilByStatusKey(types.SigilStatus(s.Status), s.SigilId), []byte(s.SigilId))
	h := k.effectiveActivityHeight(s)
	switch types.SigilStatus(s.Status) {
	case types.SigilStatusActive:
		store.Set(types.LivenessIndexKey(h, s.SigilId), []byte(s.SigilId))
	case types.SigilStatusDormant:
		// Dormant liveness index is write-once: MsgPulse rejects non-active
		// sigils, so h is frozen from dormancy onset until dissolve.
		store.Set(types.DormantLivenessIndexKey(h, s.SigilId), []byte(s.SigilId))
	}
}

func (k Keeper) IterateSigilIDsByStatus(ctx sdk.Context, status types.SigilStatus, cb func(sigilID string) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SigilByStatusIteratorPrefix(status))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if cb(string(iter.Value())) {
			break
		}
	}
}

// IterateStaleDormantSigils range-scans the dormant liveness index and emits
// sigil IDs whose frozen effective activity height <= maxHeight.
func (k Keeper) IterateStaleDormantSigils(ctx sdk.Context, maxHeight int64, cb func(sigilID string) bool) {
	store := ctx.KVStore(k.storeKey)
	endKey := types.DormantLivenessIndexIteratorPrefix(maxHeight)
	iter := store.Iterator(types.DormantLivenessIndexPrefix, endKey)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if cb(string(iter.Value())) {
			break
		}
	}
}

func (k Keeper) ClearLivenessIndex(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	for _, prefix := range [][]byte{types.LivenessIndexPrefix, types.DormantLivenessIndexPrefix} {
		iter := storetypes.KVStorePrefixIterator(store, prefix)
		var keys [][]byte
		for ; iter.Valid(); iter.Next() {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			keys = append(keys, key)
		}
		iter.Close()
		for _, key := range keys {
			store.Delete(key)
		}
	}
}

// RebuildActiveLivenessIndex wipes both liveness buckets and rewrites them
// from the primary sigil store using the current effective activity height
// for every active and dormant sigil. Name kept for backward compat — the
// function now rebuilds the dormant bucket too.
func (k Keeper) RebuildActiveLivenessIndex(ctx sdk.Context) {
	k.ClearLivenessIndex(ctx)
	store := ctx.KVStore(k.storeKey)
	k.IterateAllSigils(ctx, func(s types.Sigil) bool {
		h := k.effectiveActivityHeight(s)
		switch types.SigilStatus(s.Status) {
		case types.SigilStatusActive:
			store.Set(types.LivenessIndexKey(h, s.SigilId), []byte(s.SigilId))
		case types.SigilStatusDormant:
			store.Set(types.DormantLivenessIndexKey(h, s.SigilId), []byte(s.SigilId))
		}
		return false
	})
}

func (k Keeper) Migrate1to2(ctx sdk.Context) error {
	k.RebuildActiveLivenessIndex(ctx)
	return nil
}
