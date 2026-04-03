package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/oasyce/chain/x/sigil/keeper"
	"github.com/oasyce/chain/x/sigil/types"
)

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
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

	return k, ctx
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
		PublicKey:      []byte("pubkey-fork-child-xxxxxxxxxxxxxx"),
		ForkMode:      0,
		Metadata:       "forked child",
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
