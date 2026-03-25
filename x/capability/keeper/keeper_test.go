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
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/keeper"
	"github.com/oasyce/chain/x/capability/types"
)

// --- Mock keepers ---

type mockBankKeeper struct{}

func (m mockBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m mockBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(10000000000)))
}

type mockSettlementKeeper struct {
	escrowCounter int
	released      map[string]bool
	refunded      map[string]bool
}

func newMockSettlementKeeper() *mockSettlementKeeper {
	return &mockSettlementKeeper{
		released: make(map[string]bool),
		refunded: make(map[string]bool),
	}
}

func (m *mockSettlementKeeper) CreateEscrow(_ sdk.Context, _, _ string, _ sdk.Coin, _ uint64) (string, error) {
	m.escrowCounter++
	id := "ESCROW_" + string(rune('0'+m.escrowCounter))
	return id, nil
}

func (m *mockSettlementKeeper) ReleaseEscrow(_ sdk.Context, escrowID string, _ string) error {
	m.released[escrowID] = true
	return nil
}

func (m *mockSettlementKeeper) RefundEscrow(_ sdk.Context, escrowID string, _ string) error {
	m.refunded[escrowID] = true
	return nil
}

// setupKeeper creates a test keeper with an in-memory store.
func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockSettlementKeeper) {
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

	bank := mockBankKeeper{}
	settlement := newMockSettlementKeeper()

	k := keeper.NewKeeper(storeKey, cdc, bank, settlement, "authority")
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return k, ctx, settlement
}

func TestRegisterAndGetCapability(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	msg := &types.MsgRegisterCapability{
		Creator:      "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3",
		Name:         "Translation API",
		Description:  "Translate text between languages",
		EndpointUrl:  "https://api.example.com/translate",
		PricePerCall: sdk.NewInt64Coin("uoas", 50000000),
		Tags:         []string{"nlp", "translation"},
		RateLimit:    60,
	}

	capID, err := k.RegisterCapability(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	cap, err := k.GetCapability(ctx, capID)
	if err != nil {
		t.Fatalf("GetCapability failed: %v", err)
	}

	if cap.Name != "Translation API" {
		t.Errorf("expected name 'Translation API', got '%s'", cap.Name)
	}
	if cap.Provider != msg.Creator {
		t.Errorf("expected provider %s, got %s", msg.Creator, cap.Provider)
	}
	if !cap.IsActive {
		t.Error("expected capability to be active")
	}
	if cap.TotalCalls != 0 {
		t.Errorf("expected 0 total calls, got %d", cap.TotalCalls)
	}
}

func TestInvokeAndCompleteFlow(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "a]b4c5d6e7f8a1b2c3d4e5f6a7b8c9d0e1f2a3b4" // 40 chars

	// Register capability.
	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Test API",
		Description:  "A test capability",
		EndpointUrl:  "https://api.test.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100000),
		Tags:         []string{"test"},
		RateLimit:    100,
	}
	capID, err := k.RegisterCapability(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// Set block height for challenge window tracking.
	ctx = ctx.WithBlockHeight(10)

	// Invoke capability.
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{"query": "hello"}`),
	}
	invID, escrowID, err := k.InvokeCapability(ctx, invokeMsg)
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}

	if escrowID == "" {
		t.Error("expected non-empty escrow ID")
	}

	// Verify invocation is pending.
	inv, err := k.GetInvocation(ctx, invID)
	if err != nil {
		t.Fatalf("GetInvocation failed: %v", err)
	}
	if inv.Status != types.StatusPending {
		t.Errorf("expected status PENDING, got %s", inv.Status)
	}

	// Complete the invocation — should mark COMPLETED, not SUCCESS.
	err = k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Verify invocation is COMPLETED (not SUCCESS yet).
	inv, _ = k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusCompleted {
		t.Errorf("expected status COMPLETED, got %s", inv.Status)
	}
	if inv.CompletedHeight != 10 {
		t.Errorf("expected completed_height 10, got %d", inv.CompletedHeight)
	}

	// Escrow should NOT be released yet.
	if settlement.released[escrowID] {
		t.Error("escrow should NOT be released during challenge window")
	}

	// Provider tries to claim too early — should fail.
	err = k.ClaimInvocation(ctx, invID, provider)
	if err == nil {
		t.Error("expected challenge window error when claiming too early")
	}

	// Advance past challenge window (block 10 + 100 = 110).
	ctx = ctx.WithBlockHeight(110)
	err = k.ClaimInvocation(ctx, invID, provider)
	if err != nil {
		t.Fatalf("ClaimInvocation failed: %v", err)
	}

	// Verify invocation is now SUCCESS.
	inv, _ = k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusSuccess {
		t.Errorf("expected status SUCCESS after claim, got %s", inv.Status)
	}

	// Verify escrow was released.
	if !settlement.released[escrowID] {
		t.Error("expected escrow to be released after claim")
	}

	// Verify capability stats updated.
	cap, _ := k.GetCapability(ctx, capID)
	if cap.TotalCalls != 1 {
		t.Errorf("expected 1 total call, got %d", cap.TotalCalls)
	}
	if !cap.TotalEarned.Equal(math.NewInt(100000)) {
		t.Errorf("expected total earned 100000, got %s", cap.TotalEarned)
	}
}

func TestDisputeInvocationFlow(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Dispute Test API",
		Description:  "Will be disputed",
		EndpointUrl:  "https://api.dispute.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 500000),
		Tags:         []string{"test"},
		RateLimit:    10,
	}
	capID, _ := k.RegisterCapability(ctx, regMsg)

	ctx = ctx.WithBlockHeight(50)
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{"data":"test"}`),
	}
	invID, escrowID, _ := k.InvokeCapability(ctx, invokeMsg)

	// Provider completes at block 50.
	_ = k.CompleteInvocation(ctx, invID, outputHash, provider, "")

	// Consumer disputes at block 80 (within 100-block window).
	ctx = ctx.WithBlockHeight(80)
	err := k.DisputeInvocation(ctx, invID, consumer, "output was garbage")
	if err != nil {
		t.Fatalf("DisputeInvocation failed: %v", err)
	}

	// Verify status is DISPUTED.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusDisputed {
		t.Errorf("expected DISPUTED, got %s", inv.Status)
	}

	// Verify escrow was refunded.
	if !settlement.refunded[escrowID] {
		t.Error("expected escrow to be refunded on dispute")
	}

	// Verify success rate dropped.
	cap, _ := k.GetCapability(ctx, capID)
	if cap.SuccessRate != 0 {
		t.Errorf("expected success rate 0 after dispute, got %d", cap.SuccessRate)
	}
}

func TestDisputeAfterWindowExpired(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Late Dispute",
		Description:  "Too late to dispute",
		EndpointUrl:  "https://api.late.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100000),
		Tags:         []string{},
		RateLimit:    10,
	}
	capID, _ := k.RegisterCapability(ctx, regMsg)

	ctx = ctx.WithBlockHeight(10)
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{}`),
	}
	invID, _, _ := k.InvokeCapability(ctx, invokeMsg)
	_ = k.CompleteInvocation(ctx, invID, outputHash, provider, "")

	// Try to dispute at block 200 (window ended at 110).
	ctx = ctx.WithBlockHeight(200)
	err := k.DisputeInvocation(ctx, invID, consumer, "too late")
	if err == nil {
		t.Error("expected challenge window expired error")
	}
}

func TestCompleteRejectsShortOutputHash(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Hash Test",
		Description:  "Test hash validation",
		EndpointUrl:  "https://api.hash.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100000),
		Tags:         []string{},
		RateLimit:    10,
	}
	capID, _ := k.RegisterCapability(ctx, regMsg)

	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{}`),
	}
	invID, _, _ := k.InvokeCapability(ctx, invokeMsg)

	// Short hash should be rejected.
	err := k.CompleteInvocation(ctx, invID, "tooshort", provider, "")
	if err == nil {
		t.Error("expected error for short output hash")
	}

	// Empty hash should be rejected.
	err = k.CompleteInvocation(ctx, invID, "", provider, "")
	if err == nil {
		t.Error("expected error for empty output hash")
	}
}

func TestInvokeAndFailRefundFlow(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	// Register capability.
	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Fail API",
		Description:  "Will fail",
		EndpointUrl:  "https://api.fail.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 200000),
		Tags:         []string{"test"},
		RateLimit:    10,
	}
	capID, err := k.RegisterCapability(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// Invoke.
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{"data": "test"}`),
	}
	invID, escrowID, err := k.InvokeCapability(ctx, invokeMsg)
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}

	// Fail the invocation.
	err = k.FailInvocation(ctx, invID, provider)
	if err != nil {
		t.Fatalf("FailInvocation failed: %v", err)
	}

	// Verify invocation is failed.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusFailed {
		t.Errorf("expected status FAILED, got %s", inv.Status)
	}

	// Verify escrow was refunded.
	if !settlement.refunded[escrowID] {
		t.Error("expected escrow to be refunded")
	}

	// Verify capability stats: success rate should drop.
	cap, _ := k.GetCapability(ctx, capID)
	if cap.TotalCalls != 1 {
		t.Errorf("expected 1 total call, got %d", cap.TotalCalls)
	}
	if cap.SuccessRate != 0 {
		t.Errorf("expected success rate 0 after failure, got %d", cap.SuccessRate)
	}
}

func TestDeactivateCapability(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"

	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Deactivate Me",
		Description:  "Will be deactivated",
		EndpointUrl:  "https://api.deactivate.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100),
		Tags:         []string{"temp"},
		RateLimit:    10,
	}
	capID, err := k.RegisterCapability(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// Deactivate.
	deactivateMsg := &types.MsgDeactivateCapability{
		Creator:      provider,
		CapabilityId: capID,
	}
	err = k.DeactivateCapability(ctx, deactivateMsg)
	if err != nil {
		t.Fatalf("DeactivateCapability failed: %v", err)
	}

	cap, _ := k.GetCapability(ctx, capID)
	if cap.IsActive {
		t.Error("expected capability to be inactive after deactivation")
	}

	// Try to invoke deactivated capability.
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde",
		CapabilityId: capID,
		Input:        []byte(`{}`),
	}
	_, _, err = k.InvokeCapability(ctx, invokeMsg)
	if err == nil {
		t.Error("expected error when invoking inactive capability")
	}
}

func TestDeactivateUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	other := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Auth Test",
		Description:  "Test auth",
		EndpointUrl:  "https://api.auth.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100),
		Tags:         []string{},
		RateLimit:    10,
	}
	capID, _ := k.RegisterCapability(ctx, regMsg)

	// Try to deactivate as non-owner.
	deactivateMsg := &types.MsgDeactivateCapability{
		Creator:      other,
		CapabilityId: capID,
	}
	err := k.DeactivateCapability(ctx, deactivateMsg)
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

func TestListByProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"

	for i := 0; i < 3; i++ {
		msg := &types.MsgRegisterCapability{
			Creator:      provider,
			Name:         "API " + string(rune('A'+i)),
			Description:  "desc",
			EndpointUrl:  "https://api.test.com",
			PricePerCall: sdk.NewInt64Coin("uoas", 100),
			Tags:         []string{"test"},
			RateLimit:    10,
		}
		if _, err := k.RegisterCapability(ctx, msg); err != nil {
			t.Fatalf("RegisterCapability failed: %v", err)
		}
	}

	caps := k.ListByProvider(ctx, provider)
	if len(caps) != 3 {
		t.Errorf("expected 3 capabilities, got %d", len(caps))
	}
}

func TestRateLimitExceeded(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set max rate limit to 50.
	params := types.DefaultParams()
	params.MaxRateLimit = 50
	if err := k.SetParams(ctx, params); err != nil {
		t.Fatal(err)
	}

	msg := &types.MsgRegisterCapability{
		Creator:      "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3",
		Name:         "Rate Test",
		Description:  "Test rate limit",
		EndpointUrl:  "https://api.rate.com",
		PricePerCall: sdk.NewInt64Coin("uoas", 100),
		Tags:         []string{},
		RateLimit:    100, // Exceeds max of 50
	}
	_, err := k.RegisterCapability(ctx, msg)
	if err == nil {
		t.Error("expected rate limit exceeded error")
	}
}

func TestListCapabilitiesByTag(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"

	// Register 3 capabilities with different tags.
	caps := []struct {
		name string
		tags []string
	}{
		{"Translation API", []string{"nlp", "translation"}},
		{"Image Classifier", []string{"vision", "ml"}},
		{"Summarizer", []string{"nlp", "summarization"}},
	}

	for _, c := range caps {
		msg := &types.MsgRegisterCapability{
			Creator:      provider,
			Name:         c.name,
			Description:  "Test capability",
			EndpointUrl:  "https://api.test.com/" + c.name,
			PricePerCall: sdk.NewInt64Coin("uoas", 100000),
			Tags:         c.tags,
			RateLimit:    60,
		}
		if _, err := k.RegisterCapability(ctx, msg); err != nil {
			t.Fatalf("RegisterCapability(%s) failed: %v", c.name, err)
		}
	}

	// Filter by "nlp" — should return 2 (Translation API + Summarizer).
	nlpCaps := k.ListCapabilities(ctx, "nlp")
	if len(nlpCaps) != 2 {
		t.Fatalf("ListCapabilities(nlp): expected 2, got %d", len(nlpCaps))
	}

	// Filter by "vision" — should return 1 (Image Classifier).
	visionCaps := k.ListCapabilities(ctx, "vision")
	if len(visionCaps) != 1 {
		t.Fatalf("ListCapabilities(vision): expected 1, got %d", len(visionCaps))
	}
	if visionCaps[0].Name != "Image Classifier" {
		t.Fatalf("expected 'Image Classifier', got '%s'", visionCaps[0].Name)
	}

	// Empty tag — should return all 3.
	allCaps := k.ListCapabilities(ctx, "")
	if len(allCaps) != 3 {
		t.Fatalf("ListCapabilities(''): expected 3, got %d", len(allCaps))
	}

	// Non-existent tag — should return 0.
	noneCaps := k.ListCapabilities(ctx, "nonexistent")
	if len(noneCaps) != 0 {
		t.Fatalf("ListCapabilities(nonexistent): expected 0, got %d", len(noneCaps))
	}
}

// ---------------------------------------------------------------------------
// Challenge-window invocation lifecycle: authorization & state-machine tests
// ---------------------------------------------------------------------------

// helper: register a capability + invoke it, returning IDs and escrowID.
func registerAndInvoke(t *testing.T, k keeper.Keeper, ctx sdk.Context, provider, consumer string) (capID, invID, escrowID string) {
	t.Helper()
	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Lifecycle Test",
		Description:  "Used by lifecycle tests",
		EndpointUrl:  "https://api.lifecycle.test",
		PricePerCall: sdk.NewInt64Coin("uoas", 100000),
		Tags:         []string{"lifecycle"},
		RateLimit:    100,
	}
	capID, err := k.RegisterCapability(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{"lifecycle":"test"}`),
	}
	invID, escrowID, err = k.InvokeCapability(ctx, invokeMsg)
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}
	return capID, invID, escrowID
}

func TestCompleteInvocationUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "cccccccccccccccccccccccccccccccccccccccc"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Consumer (non-provider) tries to complete — should fail.
	err := k.CompleteInvocation(ctx, invID, outputHash, consumer, "")
	if err == nil {
		t.Fatal("expected ErrUnauthorized when non-provider completes invocation")
	}

	// Third party tries to complete — should also fail.
	thirdParty := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5xxxxxx"
	err = k.CompleteInvocation(ctx, invID, outputHash, thirdParty, "")
	if err == nil {
		t.Fatal("expected ErrUnauthorized when third-party completes invocation")
	}

	// Verify invocation still PENDING.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusPending {
		t.Errorf("expected PENDING after unauthorized complete attempt, got %s", inv.Status)
	}
}

func TestCompleteInvocationAlreadyCompleted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "dddddddddddddddddddddddddddddddddddddddd"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// First completion succeeds.
	err := k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err != nil {
		t.Fatalf("first CompleteInvocation failed: %v", err)
	}

	// Second completion should fail (status is COMPLETED, not PENDING).
	err = k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err == nil {
		t.Fatal("expected error when completing an already-COMPLETED invocation")
	}
}

func TestCompleteInvocationEmptyHash(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Empty string should be rejected.
	err := k.CompleteInvocation(ctx, invID, "", provider, "")
	if err == nil {
		t.Fatal("expected error for empty output_hash")
	}

	// Invocation should remain PENDING.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusPending {
		t.Errorf("expected PENDING after empty-hash rejection, got %s", inv.Status)
	}
}

func TestClaimInvocationUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	thirdParty := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5xxxxxx"
	outputHash := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(10)
	_ = k.CompleteInvocation(ctx, invID, outputHash, provider, "")

	// Advance past challenge window.
	ctx = ctx.WithBlockHeight(110)

	// Consumer tries to claim — should fail.
	err := k.ClaimInvocation(ctx, invID, consumer)
	if err == nil {
		t.Fatal("expected ErrUnauthorized when consumer tries to claim")
	}

	// Third party tries to claim — should fail.
	err = k.ClaimInvocation(ctx, invID, thirdParty)
	if err == nil {
		t.Fatal("expected ErrUnauthorized when third-party tries to claim")
	}

	// Verify invocation still COMPLETED (not SUCCESS).
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusCompleted {
		t.Errorf("expected COMPLETED after unauthorized claim attempts, got %s", inv.Status)
	}
}

func TestClaimInvocationNotCompleted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Invocation is PENDING — provider tries to claim directly.
	err := k.ClaimInvocation(ctx, invID, provider)
	if err == nil {
		t.Fatal("expected ErrInvalidStatus when claiming a PENDING invocation")
	}

	// Verify still PENDING.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusPending {
		t.Errorf("expected PENDING, got %s", inv.Status)
	}
}

func TestDisputeInvocationUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	thirdParty := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5xxxxxx"
	outputHash := "ffffffffffffffffffffffffffffffffffffffff"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(20)
	_ = k.CompleteInvocation(ctx, invID, outputHash, provider, "")

	// Third party tries to dispute — should fail.
	err := k.DisputeInvocation(ctx, invID, thirdParty, "I am not involved")
	if err == nil {
		t.Fatal("expected ErrUnauthorized when third-party disputes")
	}

	// Provider tries to dispute — should also fail (only consumer can).
	err = k.DisputeInvocation(ctx, invID, provider, "provider cannot dispute own work")
	if err == nil {
		t.Fatal("expected ErrUnauthorized when provider disputes own invocation")
	}

	// Verify still COMPLETED.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusCompleted {
		t.Errorf("expected COMPLETED after unauthorized dispute attempts, got %s", inv.Status)
	}
}

func TestDisputeInvocationNotCompleted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Invocation is PENDING — consumer tries to dispute directly.
	err := k.DisputeInvocation(ctx, invID, consumer, "not yet completed")
	if err == nil {
		t.Fatal("expected ErrInvalidStatus when disputing a PENDING invocation")
	}

	// Verify still PENDING.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusPending {
		t.Errorf("expected PENDING, got %s", inv.Status)
	}
}

func TestFailInvocationUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	thirdParty := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5xxxxxx"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Third party (not provider, not consumer) tries to fail.
	err := k.FailInvocation(ctx, invID, thirdParty)
	if err == nil {
		t.Fatal("expected ErrUnauthorized when third-party fails invocation")
	}

	// Verify still PENDING.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusPending {
		t.Errorf("expected PENDING after unauthorized fail attempt, got %s", inv.Status)
	}
}

func TestFailInvocationAlreadyCompleted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "1111111111111111111111111111111111111111"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	// Complete the invocation first.
	err := k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Now try to fail it — should reject (status is COMPLETED, not PENDING).
	err = k.FailInvocation(ctx, invID, provider)
	if err == nil {
		t.Fatal("expected ErrInvalidStatus when failing a COMPLETED invocation")
	}

	// Verify still COMPLETED.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", inv.Status)
	}
}

func TestChallengeWindowBoundary(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "2222222222222222222222222222222222222222"

	_, invID, escrowID := registerAndInvoke(t, k, ctx, provider, consumer)

	// Complete at block N=50.
	ctx = ctx.WithBlockHeight(50)
	err := k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// ChallengeWindow = 100, so window ends at block 50+100 = 150.
	// Claim at N+99 = 149 — should FAIL (block 149 < 150).
	ctx = ctx.WithBlockHeight(149)
	err = k.ClaimInvocation(ctx, invID, provider)
	if err == nil {
		t.Fatal("expected challenge window error at block N+99 (149)")
	}

	// Escrow should NOT be released yet.
	if settlement.released[escrowID] {
		t.Error("escrow should NOT be released before challenge window ends")
	}

	// Claim at exactly N+100 = 150 — should SUCCEED.
	ctx = ctx.WithBlockHeight(150)
	err = k.ClaimInvocation(ctx, invID, provider)
	if err != nil {
		t.Fatalf("ClaimInvocation at block N+100 (150) should succeed: %v", err)
	}

	// Verify invocation is SUCCESS.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusSuccess {
		t.Errorf("expected SUCCESS at block N+100, got %s", inv.Status)
	}

	// Verify escrow was released.
	if !settlement.released[escrowID] {
		t.Error("expected escrow to be released at block N+100")
	}
}

func TestDisputeAtWindowBoundary(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"
	outputHash := "3333333333333333333333333333333333333333"

	capID, invID, escrowID := registerAndInvoke(t, k, ctx, provider, consumer)

	// Complete at block N=50.
	ctx = ctx.WithBlockHeight(50)
	err := k.CompleteInvocation(ctx, invID, outputHash, provider, "")
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Dispute at N+99 = 149 — last valid block (149 < 150, inside window).
	ctx = ctx.WithBlockHeight(149)
	err = k.DisputeInvocation(ctx, invID, consumer, "bad output at boundary")
	if err != nil {
		t.Fatalf("DisputeInvocation at block N+99 (149) should succeed: %v", err)
	}

	// Verify status is DISPUTED.
	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusDisputed {
		t.Errorf("expected DISPUTED, got %s", inv.Status)
	}

	// Verify escrow was refunded.
	if !settlement.refunded[escrowID] {
		t.Error("expected escrow to be refunded on boundary dispute")
	}

	// Escrow should NOT be released.
	if settlement.released[escrowID] {
		t.Error("escrow should NOT be released after dispute")
	}

	// Verify success rate dropped.
	cap, _ := k.GetCapability(ctx, capID)
	if cap.SuccessRate != 0 {
		t.Errorf("expected success rate 0 after boundary dispute, got %d", cap.SuccessRate)
	}
}

// ---------------------------------------------------------------------------
// Non-existent invocation ID → ErrInvocationNotFound
// ---------------------------------------------------------------------------

func TestCompleteInvocationNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.CompleteInvocation(ctx, "INV_DOES_NOT_EXIST", "aaaaaaaabbbbbbbbccccccccdddddddd", "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3", "")
	if err == nil {
		t.Fatal("expected ErrInvocationNotFound")
	}
}

func TestClaimInvocationNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.ClaimInvocation(ctx, "INV_DOES_NOT_EXIST", "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3")
	if err == nil {
		t.Fatal("expected ErrInvocationNotFound")
	}
}

func TestDisputeInvocationNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.DisputeInvocation(ctx, "INV_DOES_NOT_EXIST", "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde", "bad output")
	if err == nil {
		t.Fatal("expected ErrInvocationNotFound")
	}
}

func TestFailInvocationNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.FailInvocation(ctx, "INV_DOES_NOT_EXIST", "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3")
	if err == nil {
		t.Fatal("expected ErrInvocationNotFound")
	}
}

// ---------------------------------------------------------------------------
// Consumer can also call FailInvocation (code: provider OR consumer allowed)
// ---------------------------------------------------------------------------

func TestFailInvocationByConsumer(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, escrowID := registerAndInvoke(t, k, ctx, provider, consumer)

	// Consumer (not provider) fails — should succeed.
	err := k.FailInvocation(ctx, invID, consumer)
	if err != nil {
		t.Fatalf("consumer FailInvocation should succeed: %v", err)
	}

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusFailed {
		t.Errorf("expected FAILED, got %s", inv.Status)
	}
	if !settlement.refunded[escrowID] {
		t.Error("expected escrow refunded when consumer fails invocation")
	}
}

// ---------------------------------------------------------------------------
// Free capability (price=0): complete → dispute / fail  (no escrow path)
// ---------------------------------------------------------------------------

func registerFreeAndInvoke(t *testing.T, k keeper.Keeper, ctx sdk.Context, provider, consumer string) (capID, invID string) {
	t.Helper()
	regMsg := &types.MsgRegisterCapability{
		Creator:      provider,
		Name:         "Free API",
		Description:  "Zero price capability",
		EndpointUrl:  "https://api.free.test",
		PricePerCall: sdk.NewInt64Coin("uoas", 0),
		Tags:         []string{"free"},
		RateLimit:    100,
	}
	capID, err := k.RegisterCapability(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}
	invokeMsg := &types.MsgInvokeCapability{
		Creator:      consumer,
		CapabilityId: capID,
		Input:        []byte(`{}`),
	}
	invID, escrowID, err := k.InvokeCapability(ctx, invokeMsg)
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}
	if escrowID != "" {
		t.Fatalf("expected empty escrow for free capability, got %s", escrowID)
	}
	return capID, invID
}

func TestFreeCapabilityDisputeNoEscrow(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID := registerFreeAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(10)
	if err := k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, ""); err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Dispute within window — no escrow to refund, should still succeed.
	err := k.DisputeInvocation(ctx, invID, consumer, "bad output")
	if err != nil {
		t.Fatalf("DisputeInvocation on free capability should succeed: %v", err)
	}

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusDisputed {
		t.Errorf("expected DISPUTED, got %s", inv.Status)
	}
	// No escrow should have been touched.
	if len(settlement.refunded) != 0 || len(settlement.released) != 0 {
		t.Error("no escrow operations expected for free capability")
	}
}

func TestFreeCapabilityFailNoEscrow(t *testing.T) {
	k, ctx, settlement := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID := registerFreeAndInvoke(t, k, ctx, provider, consumer)

	// Fail a free capability invocation — no escrow to refund.
	err := k.FailInvocation(ctx, invID, provider)
	if err != nil {
		t.Fatalf("FailInvocation on free capability should succeed: %v", err)
	}

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.Status != types.StatusFailed {
		t.Errorf("expected FAILED, got %s", inv.Status)
	}
	if len(settlement.refunded) != 0 || len(settlement.released) != 0 {
		t.Error("no escrow operations expected for free capability")
	}
}

// ---------------------------------------------------------------------------
// Double operations: claim SUCCESS, dispute DISPUTED, fail FAILED
// ---------------------------------------------------------------------------

func TestDoubleClaimRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(10)
	_ = k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, "")
	ctx = ctx.WithBlockHeight(110)
	_ = k.ClaimInvocation(ctx, invID, provider) // first claim succeeds

	// Second claim should fail — status is SUCCESS, not COMPLETED.
	err := k.ClaimInvocation(ctx, invID, provider)
	if err == nil {
		t.Fatal("expected error when double-claiming a SUCCESS invocation")
	}
}

func TestDoubleDisputeRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(10)
	_ = k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, "")
	_ = k.DisputeInvocation(ctx, invID, consumer, "first dispute") // succeeds

	// Second dispute should fail — status is DISPUTED, not COMPLETED.
	err := k.DisputeInvocation(ctx, invID, consumer, "second dispute")
	if err == nil {
		t.Fatal("expected error when double-disputing a DISPUTED invocation")
	}
}

func TestDoubleFailRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	_ = k.FailInvocation(ctx, invID, provider) // first fail succeeds

	// Second fail should fail — status is FAILED, not PENDING.
	err := k.FailInvocation(ctx, invID, provider)
	if err == nil {
		t.Fatal("expected error when double-failing a FAILED invocation")
	}
}

// ---------------------------------------------------------------------------
// CompletedHeight is stored correctly
// ---------------------------------------------------------------------------

func TestCompletedHeightStored(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5abcde"

	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	ctx = ctx.WithBlockHeight(42)
	_ = k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, "")

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.CompletedHeight != 42 {
		t.Errorf("expected CompletedHeight=42, got %d", inv.CompletedHeight)
	}
}

// TestUsageReportStored verifies that usage_report is persisted on-chain when completing an invocation.
func TestUsageReportStored(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1consumer000000000000000000000000"
	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	usageJSON := `{"prompt_tokens":150,"completion_tokens":80,"total_tokens":230,"model":"codex-2025"}`
	err := k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, usageJSON)
	if err != nil {
		t.Fatalf("CompleteInvocation with usage_report failed: %v", err)
	}

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.UsageReport != usageJSON {
		t.Errorf("expected usage_report=%q, got %q", usageJSON, inv.UsageReport)
	}
}

// TestUsageReportEmptyIsOk verifies that empty usage_report is accepted (optional field).
func TestUsageReportEmptyIsOk(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	provider := "oasyce1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5z5tp3"
	consumer := "oasyce1consumer000000000000000000000000"
	_, invID, _ := registerAndInvoke(t, k, ctx, provider, consumer)

	err := k.CompleteInvocation(ctx, invID, "aaaaaaaabbbbbbbbccccccccdddddddd", provider, "")
	if err != nil {
		t.Fatalf("CompleteInvocation without usage_report failed: %v", err)
	}

	inv, _ := k.GetInvocation(ctx, invID)
	if inv.UsageReport != "" {
		t.Errorf("expected empty usage_report, got %q", inv.UsageReport)
	}
}
