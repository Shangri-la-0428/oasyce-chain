package keeper

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

	"github.com/oasyce/chain/x/sigil/types"
)

func setupKeeperInternal(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Time: time.Now()}, false, log.NewNopLogger())
	k := NewKeeper(cdc, storeKey, "authority")
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	return k, ctx
}

func TestMigrate1to2_RebuildsLivenessIndexesUsingMaxPulseHeight(t *testing.T) {
	k, ctx := setupKeeperInternal(t)
	store := ctx.KVStore(k.storeKey)

	active := types.Sigil{
		SigilId:          "SIG_active_migration",
		Creator:          "oasyce1creator",
		Status:           types.SigilStatusActive,
		LastActiveHeight: 10,
		DimensionPulses: map[string]int64{
			"thronglets": 200,
		},
	}
	dormant := types.Sigil{
		SigilId:          "SIG_dormant_migration",
		Creator:          "oasyce1creator",
		Status:           types.SigilStatusDormant,
		LastActiveHeight: 15,
		DimensionPulses: map[string]int64{
			"thronglets": 250,
		},
	}

	for _, sigil := range []types.Sigil{active, dormant} {
		bz, err := k.cdc.Marshal(&sigil)
		require.NoError(t, err)
		store.Set(types.SigilKey(sigil.SigilId), bz)
		store.Set(types.SigilByStatusKey(types.SigilStatus(sigil.Status), sigil.SigilId), []byte(sigil.SigilId))
		store.Set(types.LivenessIndexKey(sigil.LastActiveHeight, sigil.SigilId), []byte(sigil.SigilId))
	}

	require.NoError(t, k.Migrate1to2(ctx))

	// Active sigil ends up in the active liveness bucket at MaxPulseHeight.
	require.Nil(t, store.Get(types.LivenessIndexKey(active.LastActiveHeight, active.SigilId)))
	require.Equal(t, []byte(active.SigilId), store.Get(types.LivenessIndexKey(200, active.SigilId)))
	require.Nil(t, store.Get(types.DormantLivenessIndexKey(200, active.SigilId)))

	// Dormant sigil ends up in the dormant liveness bucket at MaxPulseHeight,
	// and is removed from every active-bucket key that ever referenced it.
	require.Nil(t, store.Get(types.LivenessIndexKey(dormant.LastActiveHeight, dormant.SigilId)))
	require.Nil(t, store.Get(types.LivenessIndexKey(250, dormant.SigilId)))
	require.Equal(t, []byte(dormant.SigilId), store.Get(types.DormantLivenessIndexKey(250, dormant.SigilId)))
}
