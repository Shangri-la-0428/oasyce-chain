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

	"github.com/oasyce/chain/x/settlement/keeper"
	"github.com/oasyce/chain/x/settlement/types"
)

// mockBankKeeper is a simple mock for the bank keeper.
type mockBankKeeper struct {
	balances       map[string]sdk.Coins // addr -> coins
	moduleBalances map[string]sdk.Coins // module -> coins
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       make(map[string]sdk.Coins),
		moduleBalances: make(map[string]sdk.Coins),
	}
}

func (m *mockBankKeeper) fundAccount(addr string, coins sdk.Coins) {
	m.balances[addr] = m.balances[addr].Add(coins...)
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	from := fromAddr.String()
	to := toAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	from := senderAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	to := recipientAddr.String()
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	if !m.moduleBalances[moduleName].IsAllGTE(amt) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient module funds for burn")
	}
	m.moduleBalances[moduleName] = m.moduleBalances[moduleName].Sub(amt...)
	return nil
}

// setupKeeper creates a test keeper with an in-memory store.
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

	bankKeeper := newMockBankKeeper()
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, "authority")

	// Set default params.
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return k, ctx, bankKeeper
}

// testAddresses returns deterministic test addresses.
func testAddresses() (string, string) {
	creator := sdk.AccAddress([]byte("creator_____________")).String()
	provider := sdk.AccAddress([]byte("provider____________")).String()
	return creator, provider
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateEscrowAndRelease(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(1000000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	// Create escrow.
	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}
	if escrowID == "" {
		t.Fatal("expected non-empty escrow ID")
	}

	// Verify escrow is stored and LOCKED.
	escrow, found := k.GetEscrow(ctx, escrowID)
	if !found {
		t.Fatal("escrow not found after creation")
	}
	if escrow.Status != types.EscrowStatusLocked {
		t.Fatalf("expected LOCKED, got %s", escrow.Status)
	}
	if escrow.Creator != creator {
		t.Fatalf("expected creator %s, got %s", creator, escrow.Creator)
	}

	// Verify funds moved to module.
	creatorBal := bank.balances[creator]
	if !creatorBal.IsZero() {
		t.Fatalf("expected creator balance 0, got %s", creatorBal)
	}
	moduleBal := bank.moduleBalances[types.ModuleName]
	if !moduleBal.Equal(sdk.NewCoins(amount)) {
		t.Fatalf("expected module balance %s, got %s", amount, moduleBal)
	}

	// Release escrow.
	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("ReleaseEscrow failed: %v", err)
	}

	// Verify escrow is RELEASED.
	escrow, _ = k.GetEscrow(ctx, escrowID)
	if escrow.Status != types.EscrowStatusReleased {
		t.Fatalf("expected RELEASED, got %s", escrow.Status)
	}

	// Verify fee split: provider gets 90%, protocol 5%, burn 2%, treasury 3%.
	providerBal := bank.balances[provider]
	expectedProvider := sdk.NewCoin("uoas", math.NewInt(900000)) // 90% of 1000000
	if !providerBal.Equal(sdk.NewCoins(expectedProvider)) {
		t.Fatalf("expected provider balance %s, got %s", expectedProvider, providerBal)
	}

	feeCollectorBal := bank.moduleBalances["fee_collector"]
	expectedFee := sdk.NewCoin("uoas", math.NewInt(80000)) // 5% + 3% = 8% of 1000000
	if !feeCollectorBal.Equal(sdk.NewCoins(expectedFee)) {
		t.Fatalf("expected fee_collector balance %s, got %s", expectedFee, feeCollectorBal)
	}

	// Verify 2% was burned (module balance should be reduced by the full amount).
	// Total module received 1000000, sent 900000 to provider, 80000 to fee_collector, burned 20000.
	// Module balance should be 0.
	moduleBal = bank.moduleBalances[types.ModuleName]
	if !moduleBal.IsZero() {
		t.Fatalf("expected module balance 0 after release, got %s", moduleBal)
	}
}

func TestCreateEscrowAndRefund(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(500000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	// Refund escrow.
	if err := k.RefundEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("RefundEscrow failed: %v", err)
	}

	escrow, _ := k.GetEscrow(ctx, escrowID)
	if escrow.Status != types.EscrowStatusRefunded {
		t.Fatalf("expected REFUNDED, got %s", escrow.Status)
	}

	// Creator should have full balance back.
	creatorBal := bank.balances[creator]
	if !creatorBal.Equal(sdk.NewCoins(amount)) {
		t.Fatalf("expected creator balance %s, got %s", amount, creatorBal)
	}
}

func TestEscrowExpiry(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	// Advance block time past the escrow expiry.
	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Minute))

	if err := k.ExpireStaleEscrows(futureCtx); err != nil {
		t.Fatalf("ExpireStaleEscrows failed: %v", err)
	}

	escrow, _ := k.GetEscrow(futureCtx, escrowID)
	if escrow.Status != types.EscrowStatusExpired {
		t.Fatalf("expected EXPIRED, got %s", escrow.Status)
	}

	// Creator should have full balance back.
	creatorBal := bank.balances[creator]
	if !creatorBal.Equal(sdk.NewCoins(amount)) {
		t.Fatalf("expected creator balance %s, got %s", amount, creatorBal)
	}
}

func TestBondingCurvePricing(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	buyer1 := sdk.AccAddress([]byte("buyer1______________")).String()
	buyer2 := sdk.AccAddress([]byte("buyer2______________")).String()

	assetID := "DATA_001"
	payment := math.NewInt(100000)

	// Fund buyers.
	bank.fundAccount(buyer1, sdk.NewCoins(sdk.NewCoin("uoas", payment)))
	bank.fundAccount(buyer2, sdk.NewCoins(sdk.NewCoin("uoas", payment)))

	// First buyer (bootstrap): tokens = payment / INITIAL_PRICE = 100000
	shares1, err := k.BuyShares(ctx, assetID, buyer1, payment)
	if err != nil {
		t.Fatalf("BuyShares buyer1 failed: %v", err)
	}
	expectedShares1 := math.NewInt(100000) // bootstrap: 1:1
	if !shares1.Equal(expectedShares1) {
		t.Fatalf("expected shares1 %s, got %s", expectedShares1, shares1)
	}

	// Second buyer: Bancor formula with supply=100000, reserve=100000
	// tokens = 100000 * (sqrt(1 + 100000/100000) - 1) = 100000 * (sqrt(2)-1) ≈ 41421
	shares2, err := k.BuyShares(ctx, assetID, buyer2, payment)
	if err != nil {
		t.Fatalf("BuyShares buyer2 failed: %v", err)
	}
	expectedShares2 := math.NewInt(41421)
	if !shares2.Equal(expectedShares2) {
		t.Fatalf("expected shares2 %s, got %s", expectedShares2, shares2)
	}

	// Verify bonding curve state.
	state, found := k.GetBondingCurveState(ctx, assetID)
	if !found {
		t.Fatal("bonding curve state not found")
	}
	expectedTotalShares := shares1.Add(shares2)
	if !state.TotalShares.Equal(expectedTotalShares) {
		t.Fatalf("expected total shares %s, got %s", expectedTotalShares, state.TotalShares)
	}
	expectedReserve := payment.Mul(math.NewInt(2))
	if !state.Reserve.Equal(expectedReserve) {
		t.Fatalf("expected reserve %s, got %s", expectedReserve, state.Reserve)
	}

	// Get price — should be higher after purchases.
	// spot_price = reserve / (supply * CW) = 200000 / (141421 * 0.5) ≈ 2.83
	price, err := k.GetPrice(ctx, assetID)
	if err != nil {
		t.Fatalf("GetPrice failed: %v", err)
	}
	if price.IsZero() {
		t.Fatal("expected non-zero price")
	}
	// Price should be > 1 (initial price) since there have been purchases.
	if price.LT(math.NewInt(2)) {
		t.Fatalf("expected price > 1 after purchases, got %s", price)
	}
}

func TestUnauthorizedRelease(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	// Try to release as a random third party (should fail).
	thirdParty := sdk.AccAddress([]byte("thirdparty__________")).String()
	err = k.ReleaseEscrow(ctx, escrowID, thirdParty)
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestDoubleRelease(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("first release failed: %v", err)
	}

	// Second release should fail.
	err = k.ReleaseEscrow(ctx, escrowID, creator)
	if err == nil {
		t.Fatal("expected error on double release, got nil")
	}
}

func TestGetEscrowsByCreator(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	bank.fundAccount(creator, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(1000000))))

	// Create multiple escrows.
	for i := 0; i < 3; i++ {
		_, err := k.CreateEscrow(ctx, creator, provider, sdk.NewCoin("uoas", math.NewInt(100000)), 0)
		if err != nil {
			t.Fatalf("CreateEscrow %d failed: %v", i, err)
		}
	}

	escrows := k.GetEscrowsByCreator(ctx, creator)
	if len(escrows) != 3 {
		t.Fatalf("expected 3 escrows, got %d", len(escrows))
	}
}
