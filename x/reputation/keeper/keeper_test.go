package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	captypes "github.com/oasyce/chain/x/capability/types"
	"github.com/oasyce/chain/x/reputation/keeper"
	"github.com/oasyce/chain/x/reputation/types"
)

// --- Mock keepers ---

type mockCapabilityKeeper struct {
	invocations map[string]captypes.Invocation
}

func newMockCapabilityKeeper() *mockCapabilityKeeper {
	return &mockCapabilityKeeper{
		invocations: make(map[string]captypes.Invocation),
	}
}

func (m *mockCapabilityKeeper) GetInvocation(_ sdk.Context, id string) (captypes.Invocation, error) {
	inv, ok := m.invocations[id]
	if !ok {
		return captypes.Invocation{}, captypes.ErrInvocationNotFound.Wrapf("id: %s", id)
	}
	return inv, nil
}

// setupKeeper creates a test keeper with an in-memory store.
func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockCapabilityKeeper) {
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

	mockCap := newMockCapabilityKeeper()
	k := keeper.NewKeeper(cdc, storeKey, mockCap, "authority")

	return k, ctx, mockCap
}

const (
	consumer = "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
	provider = "cosmos1p8s0p6gqc6c9gt77lgr2qqujz49huhu6a80smx"
	provider2 = "cosmos1z7tu6kj5wfcfydxrzfs59s4rylakm7mjc0zydx"
)

// TestSubmitFeedbackAndScoreUpdate tests that submitting feedback updates the score.
func TestSubmitFeedbackAndScoreUpdate(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	// Register a mock invocation.
	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}

	// Submit feedback with rating 400 (= 4.0/5.0).
	feedbackID, err := k.SubmitFeedback(ctx, consumer, "INV_001", 400, "great service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if feedbackID == "" {
		t.Fatal("expected non-empty feedback ID")
	}

	// Check reputation score.
	score, found := k.GetReputation(ctx, provider)
	if !found {
		t.Fatal("expected reputation score to exist")
	}
	if score.TotalFeedbacks != 1 {
		t.Fatalf("expected 1 feedback, got %d", score.TotalFeedbacks)
	}
	if score.VerifiedFeedbacks != 1 {
		t.Fatalf("expected 1 verified feedback, got %d", score.VerifiedFeedbacks)
	}

	// Score should be approximately 400 (since single feedback, no decay, verified).
	// The normalized rating is 400/500 = 0.8, then scaled back to 400.
	scoreFloat := score.TotalScore.MustFloat64()
	if scoreFloat < 399.0 || scoreFloat > 401.0 {
		t.Fatalf("expected score ~400, got %f", scoreFloat)
	}
}

// TestVerifiedVsUnverifiedWeighting tests that verified feedback is weighted more.
func TestVerifiedVsUnverifiedWeighting(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	// Create two invocations with different consumers.
	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}
	mockCap.invocations["INV_002"] = captypes.Invocation{
		ID:       "INV_002",
		Consumer: provider2, // provider2 is the consumer here
		Provider: provider,
	}

	// Consumer submits verified feedback (rating=500, perfect score).
	_, err := k.SubmitFeedback(ctx, consumer, "INV_001", 500, "perfect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// provider2 submits feedback on INV_002 -- but provider2 IS the consumer, so this is also verified.
	// We need an unverified feedback: someone who is NOT the consumer.
	// Let's create INV_003 where consumer is someone else, and provider2 submits unverified feedback.
	mockCap.invocations["INV_003"] = captypes.Invocation{
		ID:       "INV_003",
		Consumer: "cosmos1other00000000000000000000000000000000000",
		Provider: provider,
	}

	// provider2 submits unverified feedback (not the consumer) with rating=0.
	_, err = k.SubmitFeedback(ctx, provider2, "INV_003", 0, "terrible")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score, found := k.GetReputation(ctx, provider)
	if !found {
		t.Fatal("expected reputation score to exist")
	}

	// With verified weight 1.0 and unverified weight 0.1:
	// Verified: weight=1.0, score=500/500=1.0
	// Unverified: weight=0.1, score=0/500=0.0
	// Weighted avg = (1.0*1.0 + 0.1*0.0) / (1.0 + 0.1) = 1.0 / 1.1 ~= 0.9091
	// Scaled: 0.9091 * 500 ~= 454.5
	scoreFloat := score.TotalScore.MustFloat64()
	if scoreFloat < 450.0 || scoreFloat > 460.0 {
		t.Fatalf("expected score ~454.5 (verified heavily outweighs unverified), got %f", scoreFloat)
	}

	if score.TotalFeedbacks != 2 {
		t.Fatalf("expected 2 feedbacks, got %d", score.TotalFeedbacks)
	}
	if score.VerifiedFeedbacks != 1 {
		t.Fatalf("expected 1 verified feedback, got %d", score.VerifiedFeedbacks)
	}
}

// TestDuplicatePrevention tests that duplicate feedback for the same invocation is rejected.
func TestDuplicatePrevention(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}

	// First feedback should succeed.
	_, err := k.SubmitFeedback(ctx, consumer, "INV_001", 400, "good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second feedback for same invocation from same user should fail.
	_, err = k.SubmitFeedback(ctx, consumer, "INV_001", 300, "changed my mind")
	if err == nil {
		t.Fatal("expected duplicate feedback error")
	}
}

// TestTimeDecay tests that older feedbacks contribute less to the score.
func TestTimeDecay(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}
	mockCap.invocations["INV_002"] = captypes.Invocation{
		ID:       "INV_002",
		Consumer: provider2,
		Provider: provider,
	}

	// Submit first feedback at t=0 with rating 0 (bad).
	_, err := k.SubmitFeedback(ctx, consumer, "INV_001", 0, "bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Advance time by 90 days.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(90 * 24 * time.Hour))

	// Submit second feedback at t=90 days with rating 500 (perfect).
	_, err = k.SubmitFeedback(futureCtx, provider2, "INV_002", 500, "amazing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score, found := k.GetReputation(futureCtx, provider)
	if !found {
		t.Fatal("expected reputation score to exist")
	}

	// The old feedback (90 days) should be heavily decayed.
	// decay = exp(-0.693 * 90 / 30) = exp(-2.079) ~= 0.125
	// Old: weight=0.125*1.0=0.125, score=0
	// New: weight=1.0*1.0=1.0, score=1.0
	// Weighted avg = (0.125*0.0 + 1.0*1.0) / (0.125 + 1.0) = 1.0 / 1.125 ~= 0.8889
	// Scaled: 0.8889 * 500 ~= 444.4
	scoreFloat := score.TotalScore.MustFloat64()
	if scoreFloat < 440.0 || scoreFloat > 450.0 {
		t.Fatalf("expected score ~444 (old bad feedback decayed), got %f", scoreFloat)
	}
}

// TestSelfFeedbackPrevention tests that self-feedback is rejected.
func TestSelfFeedbackPrevention(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: provider,  // provider is also the consumer
		Provider: provider,
	}

	_, err := k.SubmitFeedback(ctx, provider, "INV_001", 500, "I am great")
	if err == nil {
		t.Fatal("expected self-feedback error")
	}
}

// TestCooldown tests that feedback cooldown is enforced.
func TestCooldown(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}
	mockCap.invocations["INV_002"] = captypes.Invocation{
		ID:       "INV_002",
		Consumer: consumer,
		Provider: provider,
	}

	// First feedback succeeds.
	_, err := k.SubmitFeedback(ctx, consumer, "INV_001", 400, "good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Immediate second feedback to same provider should fail (cooldown).
	_, err = k.SubmitFeedback(ctx, consumer, "INV_002", 300, "meh")
	if err == nil {
		t.Fatal("expected cooldown error")
	}

	// Advance time past cooldown (default 60s).
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(61 * time.Second))
	_, err = k.SubmitFeedback(futureCtx, consumer, "INV_002", 300, "meh")
	if err != nil {
		t.Fatalf("unexpected error after cooldown: %v", err)
	}
}

// TestInvalidRating tests that ratings above max are rejected.
func TestInvalidRating(t *testing.T) {
	k, ctx, mockCap := setupKeeper(t)

	mockCap.invocations["INV_001"] = captypes.Invocation{
		ID:       "INV_001",
		Consumer: consumer,
		Provider: provider,
	}

	_, err := k.SubmitFeedback(ctx, consumer, "INV_001", 501, "too high")
	if err == nil {
		t.Fatal("expected invalid rating error")
	}
}

// TestReportMisbehavior tests misbehavior reporting.
func TestReportMisbehavior(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	reportID, err := k.ReportMisbehavior(ctx, consumer, provider, "spam", []byte("evidence data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reportID == "" {
		t.Fatal("expected non-empty report ID")
	}
}
