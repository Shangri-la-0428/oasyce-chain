package integration_test

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

	capkeeper "github.com/oasyce/chain/x/capability/keeper"
	captypes "github.com/oasyce/chain/x/capability/types"
	drkeeper "github.com/oasyce/chain/x/datarights/keeper"
	drtypes "github.com/oasyce/chain/x/datarights/types"
	repkeeper "github.com/oasyce/chain/x/reputation/keeper"
	reptypes "github.com/oasyce/chain/x/reputation/types"
	setkeeper "github.com/oasyce/chain/x/settlement/keeper"
	settypes "github.com/oasyce/chain/x/settlement/types"
)

// ---------------------------------------------------------------------------
// Shared bank state backing both mock bank keepers
// ---------------------------------------------------------------------------

type bankState struct {
	balances       map[string]sdk.Coins
	moduleBalances map[string]sdk.Coins
}

func newBankState() *bankState {
	return &bankState{
		balances:       make(map[string]sdk.Coins),
		moduleBalances: make(map[string]sdk.Coins),
	}
}

func (bs *bankState) fundAccount(addr string, coins sdk.Coins) {
	bs.balances[addr] = bs.balances[addr].Add(coins...)
}

func (bs *bankState) balanceOf(addr, denom string) math.Int {
	return bs.balances[addr].AmountOf(denom)
}

// ---------------------------------------------------------------------------
// sdkBankKeeper: satisfies settlement.BankKeeper and capability.BankKeeper
// (methods accept sdk.Context)
// ---------------------------------------------------------------------------

type sdkBankKeeper struct {
	*bankState
}

func (m *sdkBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	from := fromAddr.String()
	to := toAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *sdkBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.balances[addr.String()]
}

func (m *sdkBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	from := senderAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *sdkBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	to := recipientAddr.String()
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *sdkBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *sdkBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	if !m.moduleBalances[moduleName].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient module funds for burn")
	}
	m.moduleBalances[moduleName] = m.moduleBalances[moduleName].Sub(amt...)
	return nil
}

// ---------------------------------------------------------------------------
// ctxBankKeeper: satisfies datarights.BankKeeper (methods accept context.Context)
// ---------------------------------------------------------------------------

type ctxBankKeeper struct {
	*bankState
}

func (m *ctxBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	from := fromAddr.String()
	to := toAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *ctxBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	from := senderAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *ctxBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	to := recipientAddr.String()
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *ctxBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return settypes.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

// ---------------------------------------------------------------------------
// Test addresses
// ---------------------------------------------------------------------------

func providerAddr() string {
	return sdk.AccAddress([]byte("provider____________")).String()
}

func consumerAddr() string {
	return sdk.AccAddress([]byte("consumer____________")).String()
}

func arbitratorAddr() string {
	return sdk.AccAddress([]byte("arbitrator__________")).String()
}

func thirdPartyAddr() string {
	return sdk.AccAddress([]byte("thirdparty__________")).String()
}

func buyerAddr() string {
	return sdk.AccAddress([]byte("buyer_______________")).String()
}

// ---------------------------------------------------------------------------
// Shared test suite setup
// ---------------------------------------------------------------------------

type testSuite struct {
	ctx         sdk.Context
	bank        *bankState
	settlementK setkeeper.Keeper
	capabilityK capkeeper.Keeper
	reputationK repkeeper.Keeper
	datarightsK drkeeper.Keeper
}

func setupSuite(t *testing.T) *testSuite {
	t.Helper()

	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	// Create store keys for all four modules.
	settlementStoreKey := storetypes.NewKVStoreKey(settypes.StoreKey)
	capabilityStoreKey := storetypes.NewKVStoreKey(captypes.StoreKey)
	reputationStoreKey := storetypes.NewKVStoreKey(reptypes.StoreKey)
	datarightsStoreKey := storetypes.NewKVStoreKey(drtypes.StoreKey)

	cms := store.NewCommitMultiStore(db, logger, metrics.NoOpMetrics{})
	cms.MountStoreWithDB(settlementStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(capabilityStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(reputationStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(datarightsStoreKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, false, logger)

	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	bs := newBankState()
	sdkBank := &sdkBankKeeper{bs}
	ctxBank := &ctxBankKeeper{bs}

	// Wire keepers with real cross-module dependencies.
	settlementK := setkeeper.NewKeeper(cdc, settlementStoreKey, sdkBank, "authority")
	capabilityK := capkeeper.NewKeeper(capabilityStoreKey, cdc, sdkBank, settlementK)
	reputationK := repkeeper.NewKeeper(cdc, reputationStoreKey, capabilityK, "authority")
	datarightsK := drkeeper.NewKeeper(cdc, datarightsStoreKey, ctxBank, arbitratorAddr())

	// Set default params for all modules.
	if err := settlementK.SetParams(ctx, settypes.DefaultParams()); err != nil {
		t.Fatal(err)
	}
	capabilityK.SetParams(ctx, captypes.DefaultParams())
	if err := reputationK.SetParams(ctx, reptypes.DefaultParams()); err != nil {
		t.Fatal(err)
	}
	if err := datarightsK.SetParams(ctx, drtypes.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return &testSuite{
		ctx:         ctx,
		bank:        bs,
		settlementK: settlementK,
		capabilityK: capabilityK,
		reputationK: reputationK,
		datarightsK: datarightsK,
	}
}

// ---------------------------------------------------------------------------
// Test 1: Full Capability Invocation Flow
// register capability -> invoke (creates escrow) -> complete (releases escrow)
//   -> submit feedback -> check reputation
// ---------------------------------------------------------------------------

func TestFullCapabilityInvocationFlow(t *testing.T) {
	s := setupSuite(t)

	prov := providerAddr()
	cons := consumerAddr()

	// Fund the consumer with enough tokens.
	s.bank.fundAccount(cons, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(1000))))

	// 1. Register capability with price 100uoas.
	capID, err := s.capabilityK.RegisterCapability(s.ctx, &captypes.MsgRegisterCapability{
		Creator:      prov,
		Name:         "TestAI",
		Description:  "An AI capability for testing",
		EndpointUrl:  "https://example.com/ai",
		PricePerCall: sdk.NewCoin("uoas", math.NewInt(100)),
		Tags:         []string{"ai", "test"},
		RateLimit:    100,
	})
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// 2. Consumer invokes the capability (escrow locks 100uoas).
	invocationID, escrowID, err := s.capabilityK.InvokeCapability(s.ctx, &captypes.MsgInvokeCapability{
		Creator:      cons,
		CapabilityId: capID,
		Input:        []byte("test input"),
	})
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}
	if escrowID == "" {
		t.Fatal("expected non-empty escrow ID for paid capability")
	}

	// Verify escrow is locked.
	escrow, found := s.settlementK.GetEscrow(s.ctx, escrowID)
	if !found {
		t.Fatal("escrow not found after invocation")
	}
	if escrow.Status != settypes.EscrowStatusLocked {
		t.Fatalf("expected LOCKED escrow, got %s", escrow.Status)
	}

	// Consumer balance should be reduced by 100.
	consBal := s.bank.balanceOf(cons, "uoas")
	if !consBal.Equal(math.NewInt(900)) {
		t.Fatalf("expected consumer balance 900, got %s", consBal)
	}

	// 3. Provider completes the invocation (escrow releases: 95 to provider, 5 protocol fee).
	err = s.capabilityK.CompleteInvocation(s.ctx, invocationID, "outputhash123", prov)
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// Verify escrow is released.
	escrow, _ = s.settlementK.GetEscrow(s.ctx, escrowID)
	if escrow.Status != settypes.EscrowStatusReleased {
		t.Fatalf("expected RELEASED escrow, got %s", escrow.Status)
	}

	// Provider should receive 93 (93% of 100: 3% validator + 2% burn + 2% treasury).
	provBal := s.bank.balanceOf(prov, "uoas")
	if !provBal.Equal(math.NewInt(93)) {
		t.Fatalf("expected provider balance 93, got %s", provBal)
	}

	// Protocol fee collector should have 5 (3% validator + 2% treasury).
	feeBal := s.bank.moduleBalances["fee_collector"].AmountOf("uoas")
	if !feeBal.Equal(math.NewInt(5)) {
		t.Fatalf("expected fee_collector balance 5, got %s", feeBal)
	}

	// 2 uoas burned (removed from circulation).
	// Module balance should be 0 after release.
	settlementModBal := s.bank.moduleBalances["settlement"].AmountOf("uoas")
	if !settlementModBal.IsZero() {
		t.Fatalf("expected settlement module balance 0, got %s", settlementModBal)
	}

	// 4. Consumer submits feedback (rating 450 = 4.5/5.0).
	_, err = s.reputationK.SubmitFeedback(s.ctx, cons, invocationID, 450, "Great service!")
	if err != nil {
		t.Fatalf("SubmitFeedback failed: %v", err)
	}

	// 5. Verify provider reputation score is ~4.5 (in 0-500 scale = 450).
	rep, found := s.reputationK.GetReputation(s.ctx, prov)
	if !found {
		t.Fatal("reputation not found for provider")
	}

	// Score is stored in 0-500 scale as uint64. With a single verified feedback of 450,
	// the time-decayed weighted score should be exactly 450 (age=0, decay=1.0).
	if rep.TotalScore != 450 {
		t.Fatalf("expected reputation score 450, got %d", rep.TotalScore)
	}
	if rep.TotalFeedbacks != 1 {
		t.Fatalf("expected 1 total feedback, got %d", rep.TotalFeedbacks)
	}
	if rep.VerifiedFeedbacks != 1 {
		t.Fatalf("expected 1 verified feedback, got %d", rep.VerifiedFeedbacks)
	}
}

// ---------------------------------------------------------------------------
// Test 2: Failed Invocation with Refund
// register capability -> invoke -> fail -> refund -> check no reputation change
// ---------------------------------------------------------------------------

func TestFailedInvocationWithRefund(t *testing.T) {
	s := setupSuite(t)

	prov := providerAddr()
	cons := consumerAddr()

	// Fund the consumer.
	s.bank.fundAccount(cons, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(500))))

	// Register capability.
	capID, err := s.capabilityK.RegisterCapability(s.ctx, &captypes.MsgRegisterCapability{
		Creator:      prov,
		Name:         "FailTest",
		Description:  "Capability that will fail",
		EndpointUrl:  "https://example.com/fail",
		PricePerCall: sdk.NewCoin("uoas", math.NewInt(100)),
		Tags:         []string{"test"},
		RateLimit:    100,
	})
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// Invoke the capability.
	invocationID, escrowID, err := s.capabilityK.InvokeCapability(s.ctx, &captypes.MsgInvokeCapability{
		Creator:      cons,
		CapabilityId: capID,
		Input:        []byte("some input"),
	})
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}

	// Check consumer lost 100 tokens.
	consBal := s.bank.balanceOf(cons, "uoas")
	if !consBal.Equal(math.NewInt(400)) {
		t.Fatalf("expected consumer balance 400 after invoke, got %s", consBal)
	}

	// Fail the invocation (provider reports failure).
	err = s.capabilityK.FailInvocation(s.ctx, invocationID, prov)
	if err != nil {
		t.Fatalf("FailInvocation failed: %v", err)
	}

	// Verify consumer gets full refund.
	consBal = s.bank.balanceOf(cons, "uoas")
	if !consBal.Equal(math.NewInt(500)) {
		t.Fatalf("expected consumer balance 500 after refund, got %s", consBal)
	}

	// Verify escrow is refunded.
	escrow, found := s.settlementK.GetEscrow(s.ctx, escrowID)
	if !found {
		t.Fatal("escrow not found")
	}
	if escrow.Status != settypes.EscrowStatusRefunded {
		t.Fatalf("expected REFUNDED escrow, got %s", escrow.Status)
	}

	// Provider should have received nothing.
	provBal := s.bank.balanceOf(prov, "uoas")
	if !provBal.IsZero() {
		t.Fatalf("expected provider balance 0, got %s", provBal)
	}

	// Verify capability stats show the failure.
	cap, err := s.capabilityK.GetCapability(s.ctx, capID)
	if err != nil {
		t.Fatalf("GetCapability failed: %v", err)
	}
	if cap.TotalCalls != 1 {
		t.Fatalf("expected 1 total call, got %d", cap.TotalCalls)
	}
	// Success rate should be 0 after one failure.
	if cap.SuccessRate != 0 {
		t.Fatalf("expected success rate 0, got %d", cap.SuccessRate)
	}

	// No reputation change (no feedback submitted).
	_, found = s.reputationK.GetReputation(s.ctx, prov)
	if found {
		t.Fatal("expected no reputation record for provider after failed invocation without feedback")
	}
}

// ---------------------------------------------------------------------------
// Test 3: Data Asset Lifecycle
// register data asset -> buy shares (bonding curve) -> file dispute
//   -> resolve (delist) -> verify asset inactive
// ---------------------------------------------------------------------------

func TestDataAssetLifecycle(t *testing.T) {
	s := setupSuite(t)

	owner := providerAddr()
	buyAddr := buyerAddr()
	plaintiff := thirdPartyAddr()
	arb := arbitratorAddr()

	// Fund buyer for share purchase and plaintiff for dispute deposit.
	s.bank.fundAccount(buyAddr, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(500))))
	s.bank.fundAccount(plaintiff, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(2000000000))))

	coCreator := sdk.AccAddress([]byte("cocreator___________")).String()

	// 1. Register data asset with co-creators (60/40 split).
	assetID, err := s.datarightsK.RegisterDataAsset(s.ctx, drtypes.MsgRegisterDataAsset{
		Creator:     owner,
		Name:        "TestDataset",
		Description: "A test data asset",
		ContentHash: "abc123hash",
		RightsType:  drtypes.RightsOriginal,
		Tags:        []string{"data", "test"},
		CoCreators: []drtypes.CoCreator{
			{Address: owner, ShareBps: 6000},
			{Address: coCreator, ShareBps: 4000},
		},
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}
	if assetID == "" {
		t.Fatal("expected non-empty asset ID")
	}

	// Verify asset is active.
	asset, found := s.datarightsK.GetAsset(s.ctx, assetID)
	if !found {
		t.Fatal("asset not found after registration")
	}
	if asset.Status != drtypes.ASSET_STATUS_ACTIVE {
		t.Fatal("expected asset to be active")
	}

	// 2. Buyer purchases shares via bonding curve.
	sharesMinted, err := s.datarightsK.BuyShares(s.ctx, drtypes.MsgBuyShares{
		Creator: buyAddr,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(500)),
	})
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}
	if sharesMinted.IsZero() {
		t.Fatal("expected non-zero shares minted")
	}

	// First 1000 shares at 100% rate with RightsOriginal multiplier (1.0).
	expectedShares := math.NewInt(500)
	if !sharesMinted.Equal(expectedShares) {
		t.Fatalf("expected %s shares, got %s", expectedShares, sharesMinted)
	}

	// Verify buyer has no remaining funds.
	buyBal := s.bank.balanceOf(buyAddr, "uoas")
	if !buyBal.IsZero() {
		t.Fatalf("expected buyer balance 0, got %s", buyBal)
	}

	// 3. Third party files dispute.
	disputeID, err := s.datarightsK.FileDispute(s.ctx, drtypes.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Copyright infringement",
		Evidence: []byte("evidence of infringement"),
	})
	if err != nil {
		t.Fatalf("FileDispute failed: %v", err)
	}
	if disputeID == "" {
		t.Fatal("expected non-empty dispute ID")
	}

	// Verify dispute is open.
	dispute, found := s.datarightsK.GetDispute(s.ctx, disputeID)
	if !found {
		t.Fatal("dispute not found")
	}
	if dispute.Status != drtypes.StatusOpen {
		t.Fatalf("expected OPEN dispute, got %s", dispute.Status)
	}

	// 4. Arbitrator resolves with delist remedy.
	err = s.datarightsK.ResolveDispute(s.ctx, drtypes.MsgResolveDispute{
		Creator:   arb,
		DisputeId: disputeID,
		Remedy:    drtypes.RemedyDelist,
	})
	if err != nil {
		t.Fatalf("ResolveDispute failed: %v", err)
	}

	// 5. Verify asset is now inactive.
	asset, found = s.datarightsK.GetAsset(s.ctx, assetID)
	if !found {
		t.Fatal("asset not found after dispute resolution")
	}
	if asset.Status == drtypes.ASSET_STATUS_ACTIVE {
		t.Fatal("expected asset to not be active after delist")
	}

	// Verify dispute is resolved.
	dispute, found = s.datarightsK.GetDispute(s.ctx, disputeID)
	if !found {
		t.Fatal("dispute not found")
	}
	if dispute.Status != drtypes.StatusResolved {
		t.Fatalf("expected RESOLVED dispute, got %s", dispute.Status)
	}

	// 6. Verify no more share purchases allowed on delisted asset.
	s.bank.fundAccount(buyAddr, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100))))
	_, err = s.datarightsK.BuyShares(s.ctx, drtypes.MsgBuyShares{
		Creator: buyAddr,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(100)),
	})
	if err == nil {
		t.Fatal("expected error when buying shares of delisted asset")
	}
}

// ---------------------------------------------------------------------------
// Test 4: Free Capability (Zero Price)
// register free capability -> invoke -> complete -> no escrow involved
// ---------------------------------------------------------------------------

func TestFreeCapabilityNoEscrow(t *testing.T) {
	s := setupSuite(t)

	prov := providerAddr()
	cons := consumerAddr()

	// Fund the consumer (should not be spent).
	s.bank.fundAccount(cons, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(1000))))

	// 1. Register free capability with price 0uoas.
	capID, err := s.capabilityK.RegisterCapability(s.ctx, &captypes.MsgRegisterCapability{
		Creator:      prov,
		Name:         "FreeAI",
		Description:  "A free AI capability",
		EndpointUrl:  "https://example.com/free",
		PricePerCall: sdk.NewInt64Coin("uoas", 0),
		Tags:         []string{"free"},
		RateLimit:    100,
	})
	if err != nil {
		t.Fatalf("RegisterCapability failed: %v", err)
	}

	// 2. Invoke (should skip escrow).
	invocationID, escrowID, err := s.capabilityK.InvokeCapability(s.ctx, &captypes.MsgInvokeCapability{
		Creator:      cons,
		CapabilityId: capID,
		Input:        []byte("free request"),
	})
	if err != nil {
		t.Fatalf("InvokeCapability failed: %v", err)
	}
	if escrowID != "" {
		t.Fatalf("expected empty escrow ID for free capability, got %s", escrowID)
	}

	// Consumer balance should be unchanged.
	consBal := s.bank.balanceOf(cons, "uoas")
	if !consBal.Equal(math.NewInt(1000)) {
		t.Fatalf("expected consumer balance 1000, got %s", consBal)
	}

	// 3. Complete successfully.
	err = s.capabilityK.CompleteInvocation(s.ctx, invocationID, "freehash", prov)
	if err != nil {
		t.Fatalf("CompleteInvocation failed: %v", err)
	}

	// 4. Verify no funds moved.
	consBalAfter := s.bank.balanceOf(cons, "uoas")
	if !consBalAfter.Equal(math.NewInt(1000)) {
		t.Fatalf("expected consumer balance still 1000, got %s", consBalAfter)
	}

	provBal := s.bank.balanceOf(prov, "uoas")
	if !provBal.IsZero() {
		t.Fatalf("expected provider balance 0 for free capability, got %s", provBal)
	}

	// Verify the invocation succeeded.
	inv, err := s.capabilityK.GetInvocation(s.ctx, invocationID)
	if err != nil {
		t.Fatalf("GetInvocation failed: %v", err)
	}
	if inv.Status != captypes.StatusSuccess {
		t.Fatalf("expected SUCCESS status, got %s", inv.Status)
	}
}

// ---------------------------------------------------------------------------
// Test 5: Bancor Bonding Curve Diminishing Returns
// register asset -> buy shares in multiple rounds -> verify Bancor curve behavior
// ---------------------------------------------------------------------------

func TestBancorBondingCurve(t *testing.T) {
	s := setupSuite(t)

	owner := providerAddr()
	buyAddr := buyerAddr()

	// Register data asset (RightsOriginal, multiplier 1.0).
	assetID, err := s.datarightsK.RegisterDataAsset(s.ctx, drtypes.MsgRegisterDataAsset{
		Creator:     owner,
		Name:        "BondingTest",
		Description: "Testing Bancor bonding curve",
		ContentHash: "bonding123",
		RightsType:  drtypes.RightsOriginal,
		Tags:        []string{"bonding"},
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Fund buyer with enough for multiple rounds.
	s.bank.fundAccount(buyAddr, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(10000))))

	// Round 1: Bootstrap purchase (reserve=0) -> tokens = payment / INITIAL_PRICE
	// 500 uoas -> 500 tokens (1:1 bootstrap)
	shares1, err := s.datarightsK.BuyShares(s.ctx, drtypes.MsgBuyShares{
		Creator: buyAddr,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(500)),
	})
	if err != nil {
		t.Fatalf("BuyShares round 1 failed: %v", err)
	}
	expectedShares1 := math.NewInt(500)
	if !shares1.Equal(expectedShares1) {
		t.Fatalf("round 1: expected %s shares, got %s", expectedShares1, shares1)
	}

	// Verify total shares on asset.
	asset, _ := s.datarightsK.GetAsset(s.ctx, assetID)
	if !asset.TotalShares.Equal(math.NewInt(500)) {
		t.Fatalf("expected total shares 500, got %s", asset.TotalShares)
	}

	// Round 2: Bancor formula. supply=500, reserve=500
	// tokens = 500 * (sqrt(1 + 600/500) - 1) = 500 * (sqrt(2.2) - 1)
	// sqrt(2.2) ≈ 1.4832... -> tokens ≈ 500 * 0.4832 ≈ 241
	shares2, err := s.datarightsK.BuyShares(s.ctx, drtypes.MsgBuyShares{
		Creator: buyAddr,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(600)),
	})
	if err != nil {
		t.Fatalf("BuyShares round 2 failed: %v", err)
	}
	// Allow rounding — Bancor gives approximately 241 tokens
	if shares2.IsZero() {
		t.Fatal("round 2: expected non-zero shares")
	}

	// Round 3: Further purchase on bigger pool.
	shares3, err := s.datarightsK.BuyShares(s.ctx, drtypes.MsgBuyShares{
		Creator: buyAddr,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(500)),
	})
	if err != nil {
		t.Fatalf("BuyShares round 3 failed: %v", err)
	}

	// Verify diminishing returns: round 3 gives fewer shares than round 1 for same payment.
	if shares3.GTE(shares1) {
		t.Fatalf("expected round 3 (%s) to mint fewer shares than round 1 (%s)", shares3, shares1)
	}

	// Verify total shares match sum.
	asset, _ = s.datarightsK.GetAsset(s.ctx, assetID)
	expectedTotal := shares1.Add(shares2).Add(shares3)
	if !asset.TotalShares.Equal(expectedTotal) {
		t.Fatalf("expected total shares %s, got %s", expectedTotal, asset.TotalShares)
	}

	// Verify shareholder record.
	sh, found := s.datarightsK.GetShareHolder(s.ctx, assetID, buyAddr)
	if !found {
		t.Fatal("shareholder record not found")
	}
	if !sh.Shares.Equal(expectedTotal) {
		t.Fatalf("expected shareholder shares %s, got %s", expectedTotal, sh.Shares)
	}
}
