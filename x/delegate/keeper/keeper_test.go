package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/keeper"
	"github.com/oasyce/chain/x/delegate/types"
)

// ---------------------------------------------------------------------------
// Mock bank keeper with trackable balances
// ---------------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdk.Coins // addr -> coins
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{balances: make(map[string]sdk.Coins)}
}

func (m *mockBankKeeper) SetBalance(addr sdk.AccAddress, coin sdk.Coin) {
	m.balances[addr.String()] = sdk.NewCoins(coin)
}

func (m *mockBankKeeper) SubBalance(addr sdk.AccAddress, coin sdk.Coin) {
	cur := m.balances[addr.String()]
	m.balances[addr.String()] = cur.Sub(coin)
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins, ok := m.balances[addr.String()]
	if !ok {
		return sdk.NewCoin(denom, math.ZeroInt())
	}
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	coins, ok := m.balances[addr.String()]
	if !ok {
		return sdk.Coins{}
	}
	return coins
}

// ---------------------------------------------------------------------------
// Mock message router (no-op, ExecDelegate tests bypass router for unit tests)
// ---------------------------------------------------------------------------

type mockRouter struct{}

func (mockRouter) Handler(_ sdk.Msg) baseapp.MsgServiceHandler       { return nil }
func (mockRouter) HandlerByTypeURL(_ string) baseapp.MsgServiceHandler { return nil }

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NoOpMetrics{})
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, false, logger)

	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	bank := newMockBankKeeper()
	router := mockRouter{}

	k := keeper.NewKeeper(cdc, storeKey, bank, router, "authority")
	return k, ctx, bank
}

// Helper: create a standard policy for tests.
func setTestPolicy(t *testing.T, k keeper.Keeper, ctx sdk.Context, principal string) {
	t.Helper()
	policy := types.DelegatePolicy{
		Principal:           principal,
		PerTxLimit:          sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:         sdk.NewCoin("uoas", math.NewInt(10000000)),
		WindowSeconds:       86400,
		AllowedMsgs:         []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentMode:      types.ENROLLMENT_MODE_TOKEN,
		EnrollmentTokenHash: keeper.HashToken("my-secret"),
		CreatedAtSeconds:    ctx.BlockTime().Unix(),
	}
	if err := k.SetPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Tests: Policy CRUD
// ---------------------------------------------------------------------------

func TestSetAndGetPolicy(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"

	// No policy yet.
	_, found := k.GetPolicy(ctx, principal)
	if found {
		t.Fatal("expected no policy initially")
	}

	setTestPolicy(t, k, ctx, principal)

	policy, found := k.GetPolicy(ctx, principal)
	if !found {
		t.Fatal("expected policy to exist")
	}
	if policy.Principal != principal {
		t.Fatalf("principal mismatch: got %s", policy.Principal)
	}
	if !policy.PerTxLimit.Amount.Equal(math.NewInt(1000000)) {
		t.Fatalf("per_tx_limit mismatch: got %s", policy.PerTxLimit.Amount)
	}
	if len(policy.AllowedMsgs) != 1 {
		t.Fatalf("expected 1 allowed msg, got %d", len(policy.AllowedMsgs))
	}
}

func TestDeletePolicy(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"

	setTestPolicy(t, k, ctx, principal)
	k.DeletePolicy(ctx, principal)

	_, found := k.GetPolicy(ctx, principal)
	if found {
		t.Fatal("policy should be deleted")
	}
}

func TestPolicyExpiration(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"

	policy := types.DelegatePolicy{
		Principal:         principal,
		PerTxLimit:        sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:       sdk.NewCoin("uoas", math.NewInt(10000000)),
		WindowSeconds:     86400,
		ExpirationSeconds: 3600, // 1 hour
		CreatedAtSeconds:  ctx.BlockTime().Unix(),
	}
	if err := k.SetPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}

	// Not expired yet.
	if k.IsPolicyExpired(ctx, policy) {
		t.Fatal("policy should not be expired yet")
	}

	// Advance time past expiration.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Hour))
	if !k.IsPolicyExpired(futureCtx, policy) {
		t.Fatal("policy should be expired after 2 hours")
	}
}

func TestPolicyNoExpiry(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	policy := types.DelegatePolicy{
		Principal:         "oasyce1principal",
		ExpirationSeconds: 0, // no expiry
		CreatedAtSeconds:  ctx.BlockTime().Unix(),
	}

	// Even far in the future, should not expire.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(365 * 24 * time.Hour))
	if k.IsPolicyExpired(futureCtx, policy) {
		t.Fatal("policy with 0 expiration should never expire")
	}
}

// ---------------------------------------------------------------------------
// Tests: Delegate CRUD
// ---------------------------------------------------------------------------

func TestEnrollAndGetDelegate(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"
	delegate := "oasyce1delegate1"

	setTestPolicy(t, k, ctx, principal)

	// Not enrolled yet.
	_, found := k.GetDelegate(ctx, delegate)
	if found {
		t.Fatal("delegate should not exist yet")
	}

	rec := types.DelegateRecord{
		Delegate:          delegate,
		Principal:         principal,
		Label:             "agent-1",
		EnrolledAtSeconds: ctx.BlockTime().Unix(),
	}
	if err := k.SetDelegate(ctx, rec); err != nil {
		t.Fatal(err)
	}

	got, found := k.GetDelegate(ctx, delegate)
	if !found {
		t.Fatal("delegate should exist")
	}
	if got.Principal != principal {
		t.Fatalf("principal mismatch: got %s", got.Principal)
	}
	if got.Label != "agent-1" {
		t.Fatalf("label mismatch: got %s", got.Label)
	}
}

func TestDeleteDelegate(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"
	delegate := "oasyce1delegate1"

	rec := types.DelegateRecord{
		Delegate:  delegate,
		Principal: principal,
	}
	if err := k.SetDelegate(ctx, rec); err != nil {
		t.Fatal(err)
	}

	k.DeleteDelegate(ctx, principal, delegate)

	_, found := k.GetDelegate(ctx, delegate)
	if found {
		t.Fatal("delegate should be deleted")
	}
}

func TestListDelegates(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"

	// Enroll 3 delegates.
	for i, addr := range []string{"oasyce1d1", "oasyce1d2", "oasyce1d3"} {
		rec := types.DelegateRecord{
			Delegate:  addr,
			Principal: principal,
			Label:     string(rune('A' + i)),
		}
		if err := k.SetDelegate(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}

	records := k.ListDelegates(ctx, principal)
	if len(records) != 3 {
		t.Fatalf("expected 3 delegates, got %d", len(records))
	}

	// Another principal should have 0.
	records2 := k.ListDelegates(ctx, "oasyce1other")
	if len(records2) != 0 {
		t.Fatalf("expected 0 delegates for other principal, got %d", len(records2))
	}
}

// ---------------------------------------------------------------------------
// Tests: Token verification
// ---------------------------------------------------------------------------

func TestTokenVerification(t *testing.T) {
	token := "my-secret-token"
	hash := keeper.HashToken(token)

	if !keeper.VerifyToken(token, hash) {
		t.Fatal("correct token should verify")
	}
	if keeper.VerifyToken("wrong-token", hash) {
		t.Fatal("wrong token should not verify")
	}
	if keeper.VerifyToken(token, []byte("short")) {
		t.Fatal("short hash should not verify")
	}
}

// ---------------------------------------------------------------------------
// Tests: MsgServer — SetPolicy + Enroll + Revoke
// ---------------------------------------------------------------------------

func TestMsgSetPolicy(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, err := ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(500000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(5000000)),
		WindowSeconds:   3600,
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "agent-token-123",
	})
	if err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	policy, found := k.GetPolicy(ctx, "oasyce1principal")
	if !found {
		t.Fatal("policy not stored")
	}
	if !policy.PerTxLimit.Amount.Equal(math.NewInt(500000)) {
		t.Fatalf("per_tx_limit wrong: %s", policy.PerTxLimit.Amount)
	}
	// Token should be stored as hash, not plaintext.
	if keeper.VerifyToken("agent-token-123", policy.EnrollmentTokenHash) == false {
		t.Fatal("stored hash should verify against original token")
	}
}

func TestMsgEnrollSuccess(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	// Set policy first.
	_, err := ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Enroll with correct token.
	_, err = ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "secret",
		Label:     "macbook-agent",
	})
	if err != nil {
		t.Fatalf("Enroll failed: %v", err)
	}

	rec, found := k.GetDelegate(ctx, "oasyce1agent1")
	if !found {
		t.Fatal("delegate not stored")
	}
	if rec.Label != "macbook-agent" {
		t.Fatalf("label mismatch: %s", rec.Label)
	}
}

func TestMsgEnrollWrongToken(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "correct-token",
	})

	_, err := ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "wrong-token",
	})
	if err == nil {
		t.Fatal("enroll with wrong token should fail")
	}
}

func TestMsgEnrollNoPolicyFails(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, err := ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1nonexistent",
		Token:     "anything",
	})
	if err == nil {
		t.Fatal("enroll without policy should fail")
	}
}

func TestMsgEnrollDuplicateFails(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "tok",
	})

	_, _ = ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "tok",
	})

	// Second enroll should fail.
	_, err := ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "tok",
	})
	if err == nil {
		t.Fatal("duplicate enroll should fail")
	}
}

func TestMsgEnrollExpiredPolicyFails(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:         "oasyce1principal",
		PerTxLimit:        sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:       sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:       []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken:   "tok",
		ExpirationSeconds: 60, // 1 minute
	})

	// Advance time past expiration.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Minute))
	_, err := ms.Enroll(sdk.WrapSDKContext(futureCtx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "tok",
	})
	if err == nil {
		t.Fatal("enroll on expired policy should fail")
	}
}

func TestMsgRevoke(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "tok",
	})

	_, _ = ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "tok",
	})

	// Revoke.
	_, err := ms.Revoke(sdk.WrapSDKContext(ctx), &types.MsgRevoke{
		Principal: "oasyce1principal",
		Delegate:  "oasyce1agent1",
	})
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	_, found := k.GetDelegate(ctx, "oasyce1agent1")
	if found {
		t.Fatal("delegate should be removed after revoke")
	}
}

func TestMsgRevokeWrongPrincipalFails(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "tok",
	})

	_, _ = ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
		Delegate:  "oasyce1agent1",
		Principal: "oasyce1principal",
		Token:     "tok",
	})

	// Different principal tries to revoke — should fail.
	_, err := ms.Revoke(sdk.WrapSDKContext(ctx), &types.MsgRevoke{
		Principal: "oasyce1attacker",
		Delegate:  "oasyce1agent1",
	})
	if err == nil {
		t.Fatal("revoke by wrong principal should fail")
	}
}

// ---------------------------------------------------------------------------
// Tests: SpendWindow
// ---------------------------------------------------------------------------

func TestSpendWindowResets(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	principal := "oasyce1principal"

	// First access — creates fresh window.
	w := k.GetOrResetWindow(ctx, principal, 86400)
	if !w.Spent.Amount.IsZero() {
		t.Fatal("fresh window should have zero spend")
	}

	// Simulate spending.
	w.Spent = sdk.NewCoin("uoas", math.NewInt(500000))
	if err := k.SetSpendWindow(ctx, w); err != nil {
		t.Fatal(err)
	}

	// Within window — should keep spend.
	w2 := k.GetOrResetWindow(ctx, principal, 86400)
	if !w2.Spent.Amount.Equal(math.NewInt(500000)) {
		t.Fatalf("expected 500000 spent, got %s", w2.Spent.Amount)
	}

	// Advance past window — should reset.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(25 * time.Hour))
	w3 := k.GetOrResetWindow(futureCtx, principal, 86400)
	if !w3.Spent.Amount.IsZero() {
		t.Fatal("expired window should reset to zero spend")
	}
}

// ---------------------------------------------------------------------------
// Tests: IterateAll
// ---------------------------------------------------------------------------

func TestIterateAllDelegates(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for _, addr := range []string{"oasyce1d1", "oasyce1d2", "oasyce1d3"} {
		rec := types.DelegateRecord{Delegate: addr, Principal: "oasyce1p"}
		if err := k.SetDelegate(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}

	var count int
	k.IterateAllDelegates(ctx, func(rec types.DelegateRecord) bool {
		count++
		return false
	})
	if count != 3 {
		t.Fatalf("expected 3 delegates, iterated %d", count)
	}
}

func TestIterateAllPolicies(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for _, p := range []string{"oasyce1p1", "oasyce1p2"} {
		setTestPolicy(t, k, ctx, p)
	}

	var count int
	k.IterateAllPolicies(ctx, func(policy types.DelegatePolicy) bool {
		count++
		return false
	})
	if count != 2 {
		t.Fatalf("expected 2 policies, iterated %d", count)
	}
}

// ---------------------------------------------------------------------------
// Tests: MsgServer default window seconds
// ---------------------------------------------------------------------------

func TestMsgSetPolicyDefaultWindow(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, err := ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		WindowSeconds:   0, // should default to 86400
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "tok",
	})
	if err != nil {
		t.Fatal(err)
	}

	policy, _ := k.GetPolicy(ctx, "oasyce1principal")
	if policy.WindowSeconds != 86400 {
		t.Fatalf("expected default 86400 window, got %d", policy.WindowSeconds)
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple delegates sharing budget
// ---------------------------------------------------------------------------

func TestMultipleDelegatesSamePrincipal(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	_, _ = ms.SetPolicy(sdk.WrapSDKContext(ctx), &types.MsgSetPolicy{
		Principal:       "oasyce1principal",
		PerTxLimit:      sdk.NewCoin("uoas", math.NewInt(1000000)),
		WindowLimit:     sdk.NewCoin("uoas", math.NewInt(10000000)),
		AllowedMsgs:     []string{"/oasyce.datarights.v1.MsgBuyShares"},
		EnrollmentToken: "shared-secret",
	})

	// Enroll 3 agents with the same token.
	for _, agent := range []string{"oasyce1agent1", "oasyce1agent2", "oasyce1agent3"} {
		_, err := ms.Enroll(sdk.WrapSDKContext(ctx), &types.MsgEnroll{
			Delegate:  agent,
			Principal: "oasyce1principal",
			Token:     "shared-secret",
		})
		if err != nil {
			t.Fatalf("enroll %s failed: %v", agent, err)
		}
	}

	// All 3 should be listed.
	records := k.ListDelegates(ctx, "oasyce1principal")
	if len(records) != 3 {
		t.Fatalf("expected 3 delegates, got %d", len(records))
	}

	// All 3 should share the same spend window.
	w := k.GetOrResetWindow(ctx, "oasyce1principal", 86400)
	if !w.Spent.Amount.IsZero() {
		t.Fatal("initial window should be zero")
	}
}
