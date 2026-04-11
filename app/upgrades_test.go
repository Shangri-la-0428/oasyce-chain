package app

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sigilkeeper "github.com/oasyce/chain/x/sigil/keeper"
	sigiltypes "github.com/oasyce/chain/x/sigil/types"
)

type testAppOptions map[string]interface{}

func (o testAppOptions) Get(key string) interface{} {
	return o[key]
}

func newTestApp(t *testing.T) *OasyceApp {
	t.Helper()

	oldHome := DefaultNodeHome
	DefaultNodeHome = t.TempDir()
	t.Cleanup(func() {
		DefaultNodeHome = oldHome
	})

	return NewOasyceApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, testAppOptions{})
}

func TestUpgradePlanNames(t *testing.T) {
	require.Equal(t, "v0.6.0", UpgradeV060)
	require.Equal(t, "v0.8.0", UpgradeV080)
}

func TestUpgradeHandlerRegistration(t *testing.T) {
	app := &OasyceApp{}
	require.NotNil(t, app.upgradeHandlerV060())
	require.NotNil(t, app.upgradeHandlerV080())
}

func TestStoreUpgradesForPlan(t *testing.T) {
	require.NotNil(t, storeUpgradesForPlan(UpgradeV053))
	require.NotNil(t, storeUpgradesForPlan(UpgradeV070))
	require.Nil(t, storeUpgradesForPlan(UpgradeV060))
	require.Nil(t, storeUpgradesForPlan(UpgradeV080))
	require.Nil(t, storeUpgradesForPlan("unknown"))
}

func TestUpgradeV080DryRunMigratesSigilLivenessIndex(t *testing.T) {
	app := newTestApp(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{
		ChainID: "oasyce-upgrade-test-1",
		Height:  7,
		Time:    time.Now(),
	})

	active := sigiltypes.Sigil{
		SigilId:          "SIG_active_upgrade_test",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("active-upgrade-test-pubkey"),
		Status:           sigiltypes.SigilStatusActive,
		CreationHeight:   1,
		LastActiveHeight: 10,
		DimensionPulses: map[string]int64{
			"thronglets": 200,
		},
	}
	dormant := sigiltypes.Sigil{
		SigilId:          "SIG_dormant_upgrade_test",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("dormant-upgrade-test-pubkey"),
		Status:           sigiltypes.SigilStatusDormant,
		CreationHeight:   1,
		LastActiveHeight: 15,
		DimensionPulses: map[string]int64{
			"thronglets": 250,
		},
	}

	require.NoError(t, app.SigilKeeper.SetSigil(ctx, active))
	require.NoError(t, app.SigilKeeper.SetSigil(ctx, dormant))
	app.SigilKeeper.SetActiveCount(ctx, 1)

	// Simulate pre-v2 state: both active and dormant sigils indexed in the
	// single (legacy) liveness bucket by LastActiveHeight. ClearLivenessIndex
	// inside Migrate1to2 wipes both buckets before rebuilding, so it's safe
	// to leave SetSigil's new-format entries in place.
	store := ctx.KVStore(app.keys[sigiltypes.StoreKey])
	app.SigilKeeper.DeleteSigilFromLivenessIndex(ctx, sigilkeeper.MaxPulseHeight(active), active.SigilId)
	store.Set(sigiltypes.LivenessIndexKey(active.LastActiveHeight, active.SigilId), []byte(active.SigilId))
	store.Set(sigiltypes.LivenessIndexKey(dormant.LastActiveHeight, dormant.SigilId), []byte(dormant.SigilId))

	vm := app.ModuleManager.GetVersionMap()
	vm[sigiltypes.ModuleName] = 1
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))

	plan := upgradetypes.Plan{
		Name:   UpgradeV080,
		Height: 10,
		Info:   "sigil v1 -> v2 effective activity height migration",
	}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))

	updatedVM, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), updatedVM[sigiltypes.ModuleName])

	// Active sigil: only present in active bucket at MaxPulseHeight.
	require.Nil(t, store.Get(sigiltypes.LivenessIndexKey(active.LastActiveHeight, active.SigilId)))
	require.Equal(t, []byte(active.SigilId), store.Get(sigiltypes.LivenessIndexKey(200, active.SigilId)))
	require.Nil(t, store.Get(sigiltypes.DormantLivenessIndexKey(200, active.SigilId)))

	// Dormant sigil: only present in dormant bucket at MaxPulseHeight.
	require.Nil(t, store.Get(sigiltypes.LivenessIndexKey(dormant.LastActiveHeight, dormant.SigilId)))
	require.Nil(t, store.Get(sigiltypes.LivenessIndexKey(250, dormant.SigilId)))
	require.Equal(t, []byte(dormant.SigilId), store.Get(sigiltypes.DormantLivenessIndexKey(250, dormant.SigilId)))

	gotActive, found := app.SigilKeeper.GetSigil(ctx, active.SigilId)
	require.True(t, found)
	require.Equal(t, sigiltypes.SigilStatusActive, sigiltypes.SigilStatus(gotActive.Status))

	gotDormant, found := app.SigilKeeper.GetSigil(ctx, dormant.SigilId)
	require.True(t, found)
	require.Equal(t, sigiltypes.SigilStatusDormant, sigiltypes.SigilStatus(gotDormant.Status))
}

func TestModuleManagerVersionMapIncludesSigilV2(t *testing.T) {
	app := newTestApp(t)
	vm := app.ModuleManager.GetVersionMap()
	require.Equal(t, uint64(2), vm[sigiltypes.ModuleName])
	require.NotEmpty(t, vm)
}
