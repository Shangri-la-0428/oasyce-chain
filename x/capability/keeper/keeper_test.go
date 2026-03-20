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

	k := keeper.NewKeeper(storeKey, cdc, bank, settlement)
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

	// Complete the invocation.
	err = k.CompleteInvocation(ctx, invID, "abc123outputhash", provider)
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Verify invocation is success.
	inv, err = k.GetInvocation(ctx, invID)
	if err != nil {
		t.Fatalf("GetInvocation after complete failed: %v", err)
	}
	if inv.Status != types.StatusSuccess {
		t.Errorf("expected status SUCCESS, got %s", inv.Status)
	}
	if inv.OutputHash != "abc123outputhash" {
		t.Errorf("expected output hash 'abc123outputhash', got '%s'", inv.OutputHash)
	}

	// Verify escrow was released.
	if !settlement.released[escrowID] {
		t.Error("expected escrow to be released")
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
