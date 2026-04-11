package keeper_test

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

	"github.com/oasyce/chain/x/sigil/keeper"
	"github.com/oasyce/chain/x/sigil/types"
)

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	k, ctx, _ := setupKeeperWithStoreKey(t)
	return k, ctx
}

func setupKeeperWithStoreKey(t *testing.T) (keeper.Keeper, sdk.Context, *storetypes.KVStoreKey) {
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

	// Set default params.
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	return k, ctx, storeKey
}

func TestSigilCRUD(t *testing.T) {
	k, ctx := setupKeeper(t)

	sigil := types.Sigil{
		SigilId:          "SIG_test1",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("pubkey1234567890123456"),
		Status:           types.SigilStatusActive,
		CreationHeight:   100,
		LastActiveHeight: 100,
		StateRoot:        []byte("root"),
		Lineage:          nil,
		Metadata:         "test sigil",
	}

	// Set.
	err := k.SetSigil(ctx, sigil)
	require.NoError(t, err)

	// Get.
	got, found := k.GetSigil(ctx, "SIG_test1")
	require.True(t, found)
	require.Equal(t, sigil.SigilId, got.SigilId)
	require.Equal(t, sigil.Creator, got.Creator)
	require.Equal(t, sigil.Status, got.Status)

	// Not found.
	_, found = k.GetSigil(ctx, "SIG_nonexistent")
	require.False(t, found)
}

func TestBondCRUD(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create two sigils.
	require.NoError(t, k.SetSigil(ctx, types.Sigil{SigilId: "SIG_a", Creator: "creator", Status: types.SigilStatusActive}))
	require.NoError(t, k.SetSigil(ctx, types.Sigil{SigilId: "SIG_b", Creator: "creator", Status: types.SigilStatusActive}))

	bond := types.Bond{
		BondId:         "BOND_test1",
		SigilA:         "SIG_a",
		SigilB:         "SIG_b",
		CreationHeight: 100,
		Scope:          "test",
	}

	// Set.
	err := k.SetBond(ctx, bond)
	require.NoError(t, err)

	// Get.
	got, found := k.GetBond(ctx, "BOND_test1")
	require.True(t, found)
	require.Equal(t, bond.BondId, got.BondId)
	require.Equal(t, bond.SigilA, got.SigilA)

	// GetBondsBySigil.
	bondsA := k.GetBondsBySigil(ctx, "SIG_a")
	require.Len(t, bondsA, 1)
	require.Equal(t, "BOND_test1", bondsA[0].BondId)

	bondsB := k.GetBondsBySigil(ctx, "SIG_b")
	require.Len(t, bondsB, 1)

	// Delete.
	k.DeleteBond(ctx, bond)
	_, found = k.GetBond(ctx, "BOND_test1")
	require.False(t, found)
	require.Len(t, k.GetBondsBySigil(ctx, "SIG_a"), 0)
}

func TestLineage(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetLineage(ctx, "SIG_parent", "SIG_child1")
	k.SetLineage(ctx, "SIG_parent", "SIG_child2")

	children := k.GetChildren(ctx, "SIG_parent")
	require.Len(t, children, 2)
	require.Contains(t, children, "SIG_child1")
	require.Contains(t, children, "SIG_child2")

	// No children for a sigil without lineage.
	noChildren := k.GetChildren(ctx, "SIG_alone")
	require.Len(t, noChildren, 0)
}

func TestActiveCount(t *testing.T) {
	k, ctx := setupKeeper(t)

	require.Equal(t, uint64(0), k.GetActiveCount(ctx))

	k.IncrementActiveCount(ctx)
	k.IncrementActiveCount(ctx)
	require.Equal(t, uint64(2), k.GetActiveCount(ctx))

	k.DecrementActiveCount(ctx)
	require.Equal(t, uint64(1), k.GetActiveCount(ctx))

	// Won't go below 0.
	k.DecrementActiveCount(ctx)
	k.DecrementActiveCount(ctx)
	require.Equal(t, uint64(0), k.GetActiveCount(ctx))
}

func TestParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := k.GetParams(ctx)
	require.Equal(t, types.DefaultDormantThreshold, params.DormantThreshold)
	require.Equal(t, types.DefaultDissolveThreshold, params.DissolveThreshold)
	require.Equal(t, types.DefaultSubmitWindow, params.SubmitWindow)

	// Update.
	newParams := types.Params{
		DormantThreshold:  50000,
		DissolveThreshold: 500000,
		SubmitWindow:      200,
	}
	require.NoError(t, k.SetParams(ctx, newParams))

	got := k.GetParams(ctx)
	require.Equal(t, int64(50000), got.DormantThreshold)
	require.Equal(t, int64(500000), got.DissolveThreshold)
	require.Equal(t, int64(200), got.SubmitWindow)
}

func TestDeriveSigilID(t *testing.T) {
	id := types.DeriveSigilID([]byte("test-pubkey"))
	require.Contains(t, id, "SIG_")
	require.Len(t, id, 4+32) // "SIG_" + 16 bytes hex

	// Deterministic.
	id2 := types.DeriveSigilID([]byte("test-pubkey"))
	require.Equal(t, id, id2)

	// Different keys produce different IDs.
	id3 := types.DeriveSigilID([]byte("other-pubkey"))
	require.NotEqual(t, id, id3)
}

func TestDeriveBondID(t *testing.T) {
	id := types.DeriveBondID("SIG_a", "SIG_b")
	require.Contains(t, id, "BOND_")

	// Order-independent.
	id2 := types.DeriveBondID("SIG_b", "SIG_a")
	require.Equal(t, id, id2)
}

func TestMsgGenesisFlow(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServer(k)

	// Create a sigil.
	resp, err := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags",
		PublicKey: []byte("pubkey-32-bytes-for-testing-xxxxx"),
		Metadata:  "first sigil",
	})
	require.NoError(t, err)
	require.Contains(t, resp.SigilId, "SIG_")

	// Verify stored.
	sigil, found := k.GetSigil(ctx, resp.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusActive, sigil.Status)
	require.Equal(t, uint64(1), k.GetActiveCount(ctx))

	// Duplicate fails.
	_, err = srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags",
		PublicKey: []byte("pubkey-32-bytes-for-testing-xxxxx"),
	})
	require.ErrorContains(t, err, "already exists")
}

func TestMsgDissolveFlow(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServer(k)

	creator := "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags"

	resp, err := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-for-dissolve-test-xxxxxxx"),
	})
	require.NoError(t, err)

	// Dissolve.
	_, err = srv.Dissolve(ctx, &types.MsgDissolve{
		Signer:  creator,
		SigilId: resp.SigilId,
	})
	require.NoError(t, err)

	sigil, _ := k.GetSigil(ctx, resp.SigilId)
	require.Equal(t, types.SigilStatusDissolved, sigil.Status)
	require.Equal(t, uint64(0), k.GetActiveCount(ctx))

	// Double dissolve fails.
	_, err = srv.Dissolve(ctx, &types.MsgDissolve{
		Signer:  creator,
		SigilId: resp.SigilId,
	})
	require.ErrorContains(t, err, "already dissolved")
}

func TestMsgBondUnbondFlow(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServer(k)

	creator := "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags"

	respA, _ := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-bond-test-a-xxxxxxxxxxxxx"),
	})
	respB, _ := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-bond-test-b-xxxxxxxxxxxxx"),
	})

	// Bond.
	bondResp, err := srv.Bond(ctx, &types.MsgBond{
		Signer: creator,
		SigilA: respA.SigilId,
		SigilB: respB.SigilId,
		Scope:  "shared-memory",
	})
	require.NoError(t, err)
	require.Contains(t, bondResp.BondId, "BOND_")

	// Verify bond exists.
	bond, found := k.GetBond(ctx, bondResp.BondId)
	require.True(t, found)
	require.Equal(t, "shared-memory", bond.Scope)

	// Duplicate bond fails.
	_, err = srv.Bond(ctx, &types.MsgBond{
		Signer: creator,
		SigilA: respA.SigilId,
		SigilB: respB.SigilId,
	})
	require.ErrorContains(t, err, "already exists")

	// Unbond.
	_, err = srv.Unbond(ctx, &types.MsgUnbond{
		Signer: creator,
		BondId: bondResp.BondId,
	})
	require.NoError(t, err)

	_, found = k.GetBond(ctx, bondResp.BondId)
	require.False(t, found)
}

func TestMsgForkFlow(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServer(k)

	creator := "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags"

	parentResp, _ := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-fork-parent-xxxxxxxxxxxxx"),
		StateRoot: []byte("parent-state"),
	})

	// Fork.
	forkResp, err := srv.Fork(ctx, &types.MsgFork{
		Signer:        creator,
		ParentSigilId: parentResp.SigilId,
		PublicKey:     []byte("pubkey-fork-child-xxxxxxxxxxxxxx"),
		ForkMode:      0,
		Metadata:      "forked child",
	})
	require.NoError(t, err)
	require.Contains(t, forkResp.ChildSigilId, "SIG_")

	// Child inherits parent state root (Lamarckian).
	child, _ := k.GetSigil(ctx, forkResp.ChildSigilId)
	require.Equal(t, []byte("parent-state"), child.StateRoot)
	require.Equal(t, []string{parentResp.SigilId}, child.Lineage)

	// Lineage edge recorded.
	children := k.GetChildren(ctx, parentResp.SigilId)
	require.Contains(t, children, forkResp.ChildSigilId)

	require.Equal(t, uint64(2), k.GetActiveCount(ctx))
}

func TestMsgMergeFlow(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServer(k)

	creator := "oasyce1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq07gags"

	respA, _ := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-merge-test-a-xxxxxxxxxxxx"),
	})
	respB, _ := srv.Genesis(ctx, &types.MsgGenesis{
		Signer:    creator,
		PublicKey: []byte("pubkey-merge-test-b-xxxxxxxxxxxx"),
	})
	require.Equal(t, uint64(2), k.GetActiveCount(ctx))

	// Merge (absorption: A absorbs B).
	mergeResp, err := srv.Merge(ctx, &types.MsgMerge{
		Signer:    creator,
		SigilA:    respA.SigilId,
		SigilB:    respB.SigilId,
		MergeMode: 1, // Absorption
	})
	require.NoError(t, err)
	require.Equal(t, respA.SigilId, mergeResp.MergedSigilId)

	// B is dissolved.
	sigB, _ := k.GetSigil(ctx, respB.SigilId)
	require.Equal(t, types.SigilStatusDissolved, sigB.Status)

	// A is still active.
	sigA, _ := k.GetSigil(ctx, respA.SigilId)
	require.Equal(t, types.SigilStatusActive, sigA.Status)

	require.Equal(t, uint64(1), k.GetActiveCount(ctx))
}

func TestBeginBlockerDormancy(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set short thresholds for testing.
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  10,
		DissolveThreshold: 20,
		SubmitWindow:      100,
	}))

	// Create sigil at height 0.
	sigil := types.Sigil{
		SigilId:          "SIG_dormant_test",
		Creator:          "creator",
		Status:           types.SigilStatusActive,
		LastActiveHeight: 0,
	}
	require.NoError(t, k.SetSigil(ctx, sigil))
	k.IncrementActiveCount(ctx)

	// At height 5: too early for dormancy.
	ctx = ctx.WithBlockHeight(5)
	require.NoError(t, k.BeginBlocker(ctx))
	got, _ := k.GetSigil(ctx, "SIG_dormant_test")
	require.Equal(t, types.SigilStatusActive, got.Status)

	// At height 11: dormancy kicks in.
	ctx = ctx.WithBlockHeight(11)
	require.NoError(t, k.BeginBlocker(ctx))
	got, _ = k.GetSigil(ctx, "SIG_dormant_test")
	require.Equal(t, types.SigilStatusDormant, got.Status)
	require.Equal(t, uint64(0), k.GetActiveCount(ctx))

	// At height 21: dissolution.
	ctx = ctx.WithBlockHeight(21)
	require.NoError(t, k.BeginBlocker(ctx))
	got, _ = k.GetSigil(ctx, "SIG_dormant_test")
	require.Equal(t, types.SigilStatusDissolved, got.Status)
}

func TestValidateGenesis(t *testing.T) {
	// Valid genesis.
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		Sigils: []types.Sigil{
			{SigilId: "SIG_1", Creator: "c1"},
			{SigilId: "SIG_2", Creator: "c2"},
		},
		Bonds: []types.Bond{
			{BondId: "BOND_1", SigilA: "SIG_1", SigilB: "SIG_2"},
		},
	}
	require.NoError(t, types.ValidateGenesis(gs))

	// Duplicate sigil.
	bad := gs
	bad.Sigils = append(bad.Sigils, types.Sigil{SigilId: "SIG_1", Creator: "c3"})
	require.Error(t, types.ValidateGenesis(bad))

	// Bond references unknown sigil.
	bad2 := types.GenesisState{
		Params: types.DefaultParams(),
		Sigils: []types.Sigil{{SigilId: "SIG_1", Creator: "c1"}},
		Bonds:  []types.Bond{{BondId: "B1", SigilA: "SIG_1", SigilB: "SIG_UNKNOWN"}},
	}
	require.Error(t, types.ValidateGenesis(bad2))
}

func TestRegisterSigilEmitsGenesisEvent(t *testing.T) {
	k, ctx := setupKeeper(t)

	ctx = ctx.WithBlockHeight(42)
	pubkey := []byte("register-sigil-test-key1")
	sigilID, err := k.RegisterSigil(ctx, "oasyce1creator", pubkey, `{"source":"onboarding"}`)
	require.NoError(t, err)
	require.NotEmpty(t, sigilID)

	// Verify sigil_genesis event was emitted.
	events := ctx.EventManager().Events()
	found := false
	for _, e := range events {
		if e.Type == "sigil_genesis" {
			found = true
			attrs := make(map[string]string)
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, sigilID, attrs["sigil_id"])
			require.Equal(t, "oasyce1creator", attrs["creator"])
			require.Equal(t, "42", attrs["height"])
			break
		}
	}
	require.True(t, found, "expected sigil_genesis event from RegisterSigil")

	// Idempotent: second call returns same ID, no additional event.
	eventCountBefore := len(ctx.EventManager().Events())
	sigilID2, err := k.RegisterSigil(ctx, "oasyce1creator", pubkey, `{"source":"onboarding"}`)
	require.NoError(t, err)
	require.Equal(t, sigilID, sigilID2)
	require.Equal(t, eventCountBefore, len(ctx.EventManager().Events()))
}

func TestMsgPulse_Happy(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(100)
	ms := keeper.NewMsgServer(k)

	// Create a sigil first.
	_, err := ms.Genesis(sdk.WrapSDKContext(ctx), &types.MsgGenesis{
		Signer:    "oasyce1creator",
		PublicKey: []byte("pubkey1234567890123456"),
	})
	require.NoError(t, err)

	// Find the sigil ID.
	var sigilID string
	k.IterateAllSigils(ctx, func(s types.Sigil) bool {
		sigilID = s.SigilId
		return true
	})
	require.NotEmpty(t, sigilID)

	// Pulse with two dimensions.
	ctx = ctx.WithBlockHeight(200)
	_, err = ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1creator",
		SigilId:    sigilID,
		Dimensions: map[string]int64{"thronglets": 1, "psyche": 1},
	})
	require.NoError(t, err)

	// Verify dimensions and LastActiveHeight updated.
	sigil, found := k.GetSigil(ctx, sigilID)
	require.True(t, found)
	require.Equal(t, int64(200), sigil.LastActiveHeight)
	require.Len(t, sigil.DimensionPulses, 2)
	require.Equal(t, int64(200), sigil.DimensionPulses["thronglets"])
	require.Equal(t, int64(200), sigil.DimensionPulses["psyche"])
}

func TestMsgPulse_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, err := ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1creator",
		SigilId:    "SIG_nonexistent",
		Dimensions: map[string]int64{"chain": 1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestMsgPulse_NotOwner(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(100)
	ms := keeper.NewMsgServer(k)

	_, err := ms.Genesis(sdk.WrapSDKContext(ctx), &types.MsgGenesis{
		Signer:    "oasyce1creator",
		PublicKey: []byte("pubkey1234567890123456"),
	})
	require.NoError(t, err)

	var sigilID string
	k.IterateAllSigils(ctx, func(s types.Sigil) bool {
		sigilID = s.SigilId
		return true
	})

	_, err = ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1wrong",
		SigilId:    sigilID,
		Dimensions: map[string]int64{"chain": 1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not sigil owner")
}

func TestMsgPulse_RejectsDormantAndDissolved(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(100)
	ms := keeper.NewMsgServer(k)

	_, err := ms.Genesis(sdk.WrapSDKContext(ctx), &types.MsgGenesis{
		Signer:    "oasyce1creator",
		PublicKey: []byte("pubkey1234567890123456"),
	})
	require.NoError(t, err)

	var sigilID string
	var sigil types.Sigil
	k.IterateAllSigils(ctx, func(s types.Sigil) bool {
		sigilID = s.SigilId
		sigil = s
		return true
	})

	sigil.Status = types.SigilStatusDormant
	require.NoError(t, k.SetSigil(ctx, sigil))
	_, err = ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1creator",
		SigilId:    sigilID,
		Dimensions: map[string]int64{"chain": 1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")

	sigil.Status = types.SigilStatusDissolved
	require.NoError(t, k.SetSigil(ctx, sigil))
	_, err = ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1creator",
		SigilId:    sigilID,
		Dimensions: map[string]int64{"chain": 1},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")
}

func TestBeginBlocker_PulseKeepsSigilAlive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	// Create sigil at height 1.
	ctx = ctx.WithBlockHeight(1)
	_, err := ms.Genesis(sdk.WrapSDKContext(ctx), &types.MsgGenesis{
		Signer:    "oasyce1creator",
		PublicKey: []byte("pubkey1234567890123456"),
	})
	require.NoError(t, err)

	var sigilID string
	k.IterateAllSigils(ctx, func(s types.Sigil) bool {
		sigilID = s.SigilId
		return true
	})

	// Pulse at height 50000 (well before dormant threshold of 100000).
	ctx = ctx.WithBlockHeight(50000)
	_, err = ms.Pulse(sdk.WrapSDKContext(ctx), &types.MsgPulse{
		Signer:     "oasyce1creator",
		SigilId:    sigilID,
		Dimensions: map[string]int64{"thronglets": 1},
	})
	require.NoError(t, err)

	// Run BeginBlocker at height 120000 (which is 70000 blocks after pulse — still under DormantThreshold of 100000).
	ctx = ctx.WithBlockHeight(120000)
	err = k.BeginBlocker(ctx)
	require.NoError(t, err)

	sigil, found := k.GetSigil(ctx, sigilID)
	require.True(t, found)
	require.Equal(t, types.SigilStatusActive, types.SigilStatus(sigil.Status), "sigil should still be active after pulse")
}

func TestBeginBlocker_UsesEffectiveActivityHeight(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	sigil := types.Sigil{
		SigilId:          "SIG_effective_height",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("effective-height-pubkey"),
		Status:           types.SigilStatusActive,
		CreationHeight:   1,
		LastActiveHeight: 10,
		DimensionPulses: map[string]int64{
			"thronglets": 200,
		},
	}
	require.NoError(t, k.SetSigil(ctx, sigil))
	k.SetActiveCount(ctx, 1)

	ctx = ctx.WithBlockHeight(250)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found := k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusActive, types.SigilStatus(got.Status))

	ctx = ctx.WithBlockHeight(320)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found = k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDormant, types.SigilStatus(got.Status))
}

func TestBeginBlocker_UsesMaxOfLastActiveAndPulse(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	sigil := types.Sigil{
		SigilId:          "SIG_last_active_wins",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("last-active-wins-pubkey"),
		Status:           types.SigilStatusActive,
		CreationHeight:   1,
		LastActiveHeight: 300,
		DimensionPulses: map[string]int64{
			"thronglets": 50,
		},
	}
	require.NoError(t, k.SetSigil(ctx, sigil))
	k.SetActiveCount(ctx, 1)

	ctx = ctx.WithBlockHeight(350)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found := k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusActive, types.SigilStatus(got.Status))
}

func TestBeginBlocker_DormantDissolveUsesEffectiveActivityHeight(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	sigil := types.Sigil{
		SigilId:          "SIG_dormant_effective_height",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("dormant-effective-height-pubkey"),
		Status:           types.SigilStatusDormant,
		CreationHeight:   1,
		LastActiveHeight: 10,
		DimensionPulses: map[string]int64{
			"thronglets": 200,
		},
	}
	require.NoError(t, k.SetSigil(ctx, sigil))

	ctx = ctx.WithBlockHeight(350)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found := k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDormant, types.SigilStatus(got.Status))

	ctx = ctx.WithBlockHeight(450)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found = k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDissolved, types.SigilStatus(got.Status))
}

func TestBeginBlocker_DormantTransitionPopulatesDormantIndex(t *testing.T) {
	// Phase 2 (active → dormant) must write the sigil into the dormant
	// liveness bucket at its frozen MaxPulseHeight, so Phase 1 can
	// range-scan rather than iterate every dormant sigil.
	k, ctx, storeKey := setupKeeperWithStoreKey(t)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	sigil := types.Sigil{
		SigilId:          "SIG_phase2_bucket",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("phase2-bucket-pubkey"),
		Status:           types.SigilStatusActive,
		CreationHeight:   1,
		LastActiveHeight: 10,
		DimensionPulses: map[string]int64{
			"thronglets": 50,
		},
	}
	require.NoError(t, k.SetSigil(ctx, sigil))
	k.SetActiveCount(ctx, 1)

	kvStore := ctx.KVStore(storeKey)
	require.NotNil(t, kvStore.Get(types.LivenessIndexKey(50, sigil.SigilId)))
	require.Nil(t, kvStore.Get(types.DormantLivenessIndexKey(50, sigil.SigilId)))

	ctx = ctx.WithBlockHeight(250)
	require.NoError(t, k.BeginBlocker(ctx))

	got, found := k.GetSigil(ctx, sigil.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDormant, types.SigilStatus(got.Status))

	// Active bucket cleared, dormant bucket populated at the frozen height.
	require.Nil(t, kvStore.Get(types.LivenessIndexKey(50, sigil.SigilId)))
	require.Equal(t, []byte(sigil.SigilId), kvStore.Get(types.DormantLivenessIndexKey(50, sigil.SigilId)))
}

func TestBeginBlocker_DormantPhase1IgnoresUnexpiredEntries(t *testing.T) {
	// Range-scan must only touch dormant sigils whose frozen effective
	// height <= dissolveThreshold. Fresher dormant entries stay put.
	k, ctx, storeKey := setupKeeperWithStoreKey(t)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	stale := types.Sigil{
		SigilId:          "SIG_stale_dormant",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("stale-dormant-pubkey"),
		Status:           types.SigilStatusDormant,
		CreationHeight:   1,
		LastActiveHeight: 30,
	}
	fresh := types.Sigil{
		SigilId:          "SIG_fresh_dormant",
		Creator:          "oasyce1creator",
		PublicKey:        []byte("fresh-dormant-pubkey"),
		Status:           types.SigilStatusDormant,
		CreationHeight:   1,
		LastActiveHeight: 220,
	}
	require.NoError(t, k.SetSigil(ctx, stale))
	require.NoError(t, k.SetSigil(ctx, fresh))

	ctx = ctx.WithBlockHeight(300)
	require.NoError(t, k.BeginBlocker(ctx))

	gotStale, found := k.GetSigil(ctx, stale.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDissolved, types.SigilStatus(gotStale.Status))

	gotFresh, found := k.GetSigil(ctx, fresh.SigilId)
	require.True(t, found)
	require.Equal(t, types.SigilStatusDormant, types.SigilStatus(gotFresh.Status))

	kvStore := ctx.KVStore(storeKey)
	require.Nil(t, kvStore.Get(types.DormantLivenessIndexKey(30, stale.SigilId)))
	require.Equal(t, []byte(fresh.SigilId), kvStore.Get(types.DormantLivenessIndexKey(220, fresh.SigilId)))
}

func TestDimensionPulses_MarshalRoundtrip(t *testing.T) {
	original := types.Sigil{
		SigilId:          "SIG_test",
		Creator:          "oasyce1creator",
		Status:           types.SigilStatusActive,
		LastActiveHeight: 42,
		DimensionPulses: map[string]int64{
			"thronglets": 100,
			"psyche":     200,
			"chain":      300,
		},
	}

	bz, err := original.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	var decoded types.Sigil
	err = decoded.Unmarshal(bz)
	require.NoError(t, err)

	require.Equal(t, original.SigilId, decoded.SigilId)
	require.Equal(t, original.Creator, decoded.Creator)
	require.Equal(t, original.LastActiveHeight, decoded.LastActiveHeight)
	require.Len(t, decoded.DimensionPulses, 3)
	require.Equal(t, int64(100), decoded.DimensionPulses["thronglets"])
	require.Equal(t, int64(200), decoded.DimensionPulses["psyche"])
	require.Equal(t, int64(300), decoded.DimensionPulses["chain"])
}

func TestMaxPulseHeight(t *testing.T) {
	s := types.Sigil{LastActiveHeight: 100}
	require.Equal(t, int64(100), keeper.MaxPulseHeight(s))

	s.DimensionPulses = map[string]int64{"a": 50, "b": 200}
	require.Equal(t, int64(200), keeper.MaxPulseHeight(s))

	s.LastActiveHeight = 300
	require.Equal(t, int64(300), keeper.MaxPulseHeight(s))
}
