package sigil_test

import (
	"encoding/json"
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

	"github.com/oasyce/chain/x/sigil"
	"github.com/oasyce/chain/x/sigil/keeper"
	"github.com/oasyce/chain/x/sigil/types"
)

func setupSigilModule(t *testing.T) (sigil.AppModule, keeper.Keeper, codec.Codec, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Time: time.Now()}, false, log.NewNopLogger())
	k := keeper.NewKeeper(cdc, storeKey, "authority")
	am := sigil.NewAppModule(cdc, k)
	return am, k, cdc, ctx
}

func TestInitGenesis_RebuildsActiveIndexFromDimensionPulses(t *testing.T) {
	am, k, cdc, ctx := setupSigilModule(t)

	gs := types.GenesisState{
		Sigils: []types.Sigil{
			{
				SigilId:          "SIG_genesis_pulse",
				Creator:          "oasyce1creator",
				PublicKey:        []byte("genesis-pulse-pubkey"),
				Status:           types.SigilStatusActive,
				CreationHeight:   1,
				LastActiveHeight: 10,
				DimensionPulses: map[string]int64{
					"thronglets": 200,
				},
			},
		},
		Bonds: []types.Bond{},
		Params: types.Params{
			DormantThreshold:  100,
			DissolveThreshold: 200,
			SubmitWindow:      10,
		},
	}
	bz, err := json.Marshal(gs)
	require.NoError(t, err)

	am.InitGenesis(ctx, cdc, bz)

	ctx = ctx.WithBlockHeight(250)
	require.NoError(t, k.BeginBlocker(ctx))

	sigilState, found := k.GetSigil(ctx, "SIG_genesis_pulse")
	require.True(t, found)
	require.Equal(t, types.SigilStatusActive, types.SigilStatus(sigilState.Status))

	ctx = ctx.WithBlockHeight(320)
	require.NoError(t, k.BeginBlocker(ctx))

	sigilState, found = k.GetSigil(ctx, "SIG_genesis_pulse")
	require.True(t, found)
	require.Equal(t, types.SigilStatusDormant, types.SigilStatus(sigilState.Status))
}
