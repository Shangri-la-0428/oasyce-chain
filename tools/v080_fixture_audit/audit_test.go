package main

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	sigilkeeper "github.com/oasyce/chain/x/sigil/keeper"
	sigiltypes "github.com/oasyce/chain/x/sigil/types"
)

func setupAuditKeeper(t *testing.T) (sigilkeeper.Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(sigiltypes.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 500, Time: time.Now()}, false, log.NewNopLogger())
	k := sigilkeeper.NewKeeper(cdc, storeKey, "authority")
	require.NoError(t, k.SetParams(ctx, sigiltypes.DefaultParams()))
	return k, ctx, storeKey
}

func TestCollectAudit_ValidState(t *testing.T) {
	k, ctx, storeKey := setupAuditKeeper(t)

	require.NoError(t, k.SetSigil(ctx, sigiltypes.Sigil{
		SigilId:          "SIG_active",
		Creator:          "oasyce1active",
		Status:           sigiltypes.SigilStatusActive,
		LastActiveHeight: 20,
		DimensionPulses:  map[string]int64{"anchor": 200},
	}))
	require.NoError(t, k.SetSigil(ctx, sigiltypes.Sigil{
		SigilId:          "SIG_dormant",
		Creator:          "oasyce1dormant",
		Status:           sigiltypes.SigilStatusDormant,
		LastActiveHeight: 30,
		DimensionPulses:  map[string]int64{"anchor": 300},
	}))
	require.NoError(t, k.SetSigil(ctx, sigiltypes.Sigil{
		SigilId:          "SIG_dissolved",
		Creator:          "oasyce1dissolved",
		Status:           sigiltypes.SigilStatusDissolved,
		LastActiveHeight: 40,
	}))
	k.SetActiveCount(ctx, 1)

	report := collectAudit(ctx, k, storeKey, 2)
	require.Empty(t, report.InvariantErrors)
	require.Len(t, report.ActiveBucket, 1)
	require.Len(t, report.DormantBucket, 1)
	require.Empty(t, report.OrphanIndexEntries)
}

func TestCollectAudit_FlagsOrphanAndDualBucket(t *testing.T) {
	k, ctx, storeKey := setupAuditKeeper(t)
	store := ctx.KVStore(storeKey)

	require.NoError(t, k.SetSigil(ctx, sigiltypes.Sigil{
		SigilId:          "SIG_dual",
		Creator:          "oasyce1dual",
		Status:           sigiltypes.SigilStatusActive,
		LastActiveHeight: 10,
		DimensionPulses:  map[string]int64{"anchor": 120},
	}))
	k.SetActiveCount(ctx, 1)
	store.Set(sigiltypes.DormantLivenessIndexKey(120, "SIG_dual"), []byte("SIG_dual"))
	store.Set(sigiltypes.LivenessIndexKey(99, "SIG_missing"), []byte("SIG_missing"))

	report := collectAudit(ctx, k, storeKey, 1)
	require.NotEmpty(t, report.InvariantErrors)
	require.Len(t, report.OrphanIndexEntries, 1)
	require.Contains(t, report.InvariantErrors, "orphan liveness index entries present")
	require.Contains(t, report.InvariantErrors, "sigil SIG_dual appears in both active and dormant liveness buckets")
}
