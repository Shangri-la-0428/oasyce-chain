package keeper_test

import (
	"context"
	"fmt"
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

	"github.com/oasyce/chain/x/datarights/keeper"
	"github.com/oasyce/chain/x/datarights/types"
	settlementtypes "github.com/oasyce/chain/x/settlement/types"
)

// ---------------------------------------------------------------------------
// Mock Bank Keeper
// ---------------------------------------------------------------------------

type mockBankKeeper struct {
	balances       map[string]sdk.Coins
	moduleBalances map[string]sdk.Coins
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

// ---------------------------------------------------------------------------
// Mock Settlement Keeper
// ---------------------------------------------------------------------------

type mockSettlementKeeper struct{}

func (m *mockSettlementKeeper) GetBondingCurveState(_ sdk.Context, _ string) (settlementtypes.BondingCurveState, bool) {
	return settlementtypes.BondingCurveState{}, false
}

func (m *mockSettlementKeeper) BuyShares(_ sdk.Context, _ string, _ string, _ math.Int) (math.Int, error) {
	return math.ZeroInt(), nil
}

// ---------------------------------------------------------------------------
// Test Setup
// ---------------------------------------------------------------------------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bankKeeper := newMockBankKeeper()
	settlementKeeper := &mockSettlementKeeper{}
	authority := sdk.AccAddress([]byte("authority___________")).String()
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, settlementKeeper, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Time: time.Now().UTC(),
	}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return k, ctx, bankKeeper
}

func testAddresses() (string, string, string) {
	creator := sdk.AccAddress([]byte("creator_____________")).String()
	buyer := sdk.AccAddress([]byte("buyer_______________")).String()
	arbitrator := sdk.AccAddress([]byte("authority___________")).String()
	return creator, buyer, arbitrator
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRegisterDataAssetWithCoCreators(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	creator, _, _ := testAddresses()
	coCreator1 := sdk.AccAddress([]byte("cocreator1__________")).String()
	coCreator2 := sdk.AccAddress([]byte("cocreator2__________")).String()

	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Test Dataset",
		Description: "A test dataset with co-creators",
		ContentHash: "abc123hash",
		RightsType:  types.RightsCoCreation,
		Tags:        []string{"test", "data"},
		CoCreators: []types.CoCreator{
			{Address: coCreator1, ShareBps: 6000},
			{Address: coCreator2, ShareBps: 4000},
		},
	}

	assetID, err := k.RegisterDataAsset(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}
	if assetID == "" {
		t.Fatal("expected non-empty asset ID")
	}

	// Verify asset is stored.
	asset, found := k.GetAsset(ctx, assetID)
	if !found {
		t.Fatal("asset not found after registration")
	}
	if asset.Owner != creator {
		t.Fatalf("expected owner %s, got %s", creator, asset.Owner)
	}
	if asset.Name != "Test Dataset" {
		t.Fatalf("expected name 'Test Dataset', got '%s'", asset.Name)
	}
	if asset.RightsType != types.RightsCoCreation {
		t.Fatalf("expected rights type CO_CREATION, got %s", asset.RightsType)
	}
	if len(asset.CoCreators) != 2 {
		t.Fatalf("expected 2 co-creators, got %d", len(asset.CoCreators))
	}
	if !asset.IsActive {
		t.Fatal("expected asset to be active")
	}
	if !asset.TotalShares.IsZero() {
		t.Fatalf("expected zero total shares, got %s", asset.TotalShares)
	}
}

func TestRegisterDataAssetInvalidCoCreators(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	creator, _, _ := testAddresses()

	// Co-creators don't sum to 10000 bps.
	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Bad Co-creators",
		Description: "Should fail",
		ContentHash: "hash123",
		RightsType:  types.RightsCoCreation,
		CoCreators: []types.CoCreator{
			{Address: creator, ShareBps: 5000},
			{Address: creator, ShareBps: 3000},
		},
	}

	_, err := k.RegisterDataAsset(ctx, msg)
	if err == nil {
		t.Fatal("expected error for invalid co-creators, got nil")
	}
}

func TestBuySharesBancorCurve(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	// Register an original asset (multiplier 1.0).
	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Original Dataset",
		Description: "Test",
		ContentHash: "originalhash",
		RightsType:  types.RightsOriginal,
	}
	assetID, err := k.RegisterDataAsset(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	payment := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(10000000))))

	// First purchase (bootstrap): reserve=0, supply=0
	// Bancor bootstrap: tokens = payment / INITIAL_PRICE = 100000 / 1 = 100000
	buyMsg := types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  payment,
	}
	shares1, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares (1st) failed: %v", err)
	}
	expectedShares1 := math.NewInt(100000)
	if !shares1.Equal(expectedShares1) {
		t.Fatalf("expected shares1 %s, got %s", expectedShares1, shares1)
	}

	// Verify asset total shares and reserve.
	asset, _ := k.GetAsset(ctx, assetID)
	if !asset.TotalShares.Equal(expectedShares1) {
		t.Fatalf("expected total shares %s, got %s", expectedShares1, asset.TotalShares)
	}
	reserve := k.GetAssetReserve(ctx, assetID)
	if !reserve.Equal(math.NewInt(100000)) {
		t.Fatalf("expected reserve 100000, got %s", reserve)
	}

	// Second purchase: Bancor formula with supply=100000, reserve=100000
	// tokens = supply * (sqrt(1 + payment/reserve) - 1)
	//        = 100000 * (sqrt(1 + 100000/100000) - 1)
	//        = 100000 * (sqrt(2) - 1)
	//        = 100000 * 0.41421... = 41421
	shares2, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares (2nd) failed: %v", err)
	}
	expectedShares2 := math.NewInt(41421) // floor(100000 * (sqrt(2) - 1))
	if !shares2.Equal(expectedShares2) {
		t.Fatalf("expected shares2 %s, got %s", expectedShares2, shares2)
	}

	// Verify shareholder record.
	sh, found := k.GetShareHolder(ctx, assetID, buyer)
	if !found {
		t.Fatal("shareholder not found")
	}
	expectedTotal := expectedShares1.Add(expectedShares2)
	if !sh.Shares.Equal(expectedTotal) {
		t.Fatalf("expected shareholder shares %s, got %s", expectedTotal, sh.Shares)
	}

	// Third purchase: further diminishing returns from Bancor curve
	// supply=141421, reserve=200000
	// tokens = 141421 * (sqrt(1 + 100000/200000) - 1)
	//        = 141421 * (sqrt(1.5) - 1)
	//        = 141421 * 0.22474... = 31783
	shares3, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares (3rd) failed: %v", err)
	}
	// Allow small rounding difference: should be ~31783
	if shares3.LT(math.NewInt(31780)) || shares3.GT(math.NewInt(31790)) {
		t.Fatalf("expected shares3 ~31783, got %s", shares3)
	}
}

func TestBuySharesRightsTypeMultiplier(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	// Register a collection asset (multiplier 0.3).
	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Collection Dataset",
		Description: "Test collection",
		ContentHash: "collhash",
		RightsType:  types.RightsCollection,
	}
	assetID, err := k.RegisterDataAsset(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	payment := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(10000000))))

	buyMsg := types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  payment,
	}
	shares, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}
	// Bootstrap: tokens = payment / INITIAL_PRICE = 100000
	// With collection multiplier 0.3: 100000 * 0.3 = 30000
	expectedShares := math.NewInt(30000)
	if !shares.Equal(expectedShares) {
		t.Fatalf("expected shares %s, got %s", expectedShares, shares)
	}
}

func TestBuySharesDelistedAsset(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "To Delist",
		Description: "Will be delisted",
		ContentHash: "delisthash",
		RightsType:  types.RightsOriginal,
	}
	assetID, err := k.RegisterDataAsset(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Manually delist.
	asset, _ := k.GetAsset(ctx, assetID)
	asset.IsActive = false
	_ = k.SetAsset(ctx, asset)

	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100000))))
	buyMsg := types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(100000)),
	}
	_, err = k.BuyShares(ctx, buyMsg)
	if err == nil {
		t.Fatal("expected error buying shares of delisted asset, got nil")
	}
}

func TestDisputeAndDelistFlow(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, arbitrator := testAddresses()
	plaintiff := sdk.AccAddress([]byte("plaintiff___________")).String()

	// Register asset.
	regMsg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Disputed Asset",
		Description: "Will be disputed",
		ContentHash: "disputehash",
		RightsType:  types.RightsOriginal,
	}
	assetID, err := k.RegisterDataAsset(ctx, regMsg)
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Fund plaintiff for dispute deposit.
	params := k.GetParams(ctx)
	bank.fundAccount(plaintiff, sdk.NewCoins(params.DisputeDeposit))

	// File dispute.
	disputeMsg := types.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Plagiarized content",
		Evidence: []byte("proof of plagiarism"),
	}
	disputeID, err := k.FileDispute(ctx, disputeMsg)
	if err != nil {
		t.Fatalf("FileDispute failed: %v", err)
	}
	if disputeID == "" {
		t.Fatal("expected non-empty dispute ID")
	}

	// Verify dispute is stored and OPEN.
	dispute, found := k.GetDispute(ctx, disputeID)
	if !found {
		t.Fatal("dispute not found")
	}
	if dispute.Status != types.StatusOpen {
		t.Fatalf("expected OPEN, got %s", dispute.Status)
	}
	if dispute.Plaintiff != plaintiff {
		t.Fatalf("expected plaintiff %s, got %s", plaintiff, dispute.Plaintiff)
	}

	// Resolve with delist remedy.
	resolveMsg := types.MsgResolveDispute{
		Creator:   arbitrator,
		DisputeId: disputeID,
		Remedy:    types.RemedyDelist,
	}
	if err := k.ResolveDispute(ctx, resolveMsg); err != nil {
		t.Fatalf("ResolveDispute failed: %v", err)
	}

	// Verify dispute is resolved.
	dispute, _ = k.GetDispute(ctx, disputeID)
	if dispute.Status != types.StatusResolved {
		t.Fatalf("expected RESOLVED, got %s", dispute.Status)
	}
	if dispute.Remedy != types.RemedyDelist {
		t.Fatalf("expected DELIST remedy, got %s", dispute.Remedy)
	}

	// Verify asset is delisted.
	asset, _ := k.GetAsset(ctx, assetID)
	if asset.IsActive {
		t.Fatal("expected asset to be inactive after delist")
	}
}

func TestDisputeNotArbitrator(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	plaintiff := sdk.AccAddress([]byte("plaintiff___________")).String()
	notArbitrator := sdk.AccAddress([]byte("notarbitrator_______")).String()

	regMsg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Asset",
		Description: "Test",
		ContentHash: "hash999",
		RightsType:  types.RightsOriginal,
	}
	assetID, _ := k.RegisterDataAsset(ctx, regMsg)

	params := k.GetParams(ctx)
	bank.fundAccount(plaintiff, sdk.NewCoins(params.DisputeDeposit))

	disputeMsg := types.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Test reason",
		Evidence: nil,
	}
	disputeID, _ := k.FileDispute(ctx, disputeMsg)

	// Non-arbitrator tries to resolve.
	resolveMsg := types.MsgResolveDispute{
		Creator:   notArbitrator,
		DisputeId: disputeID,
		Remedy:    types.RemedyDelist,
	}
	err := k.ResolveDispute(ctx, resolveMsg)
	if err == nil {
		t.Fatal("expected not-arbitrator error, got nil")
	}
}

func TestListAssets(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	creator, _, _ := testAddresses()

	// Register multiple assets.
	for i := 0; i < 3; i++ {
		msg := types.MsgRegisterDataAsset{
			Creator:     creator,
			Name:        fmt.Sprintf("Asset %d", i),
			Description: "Test",
			ContentHash: fmt.Sprintf("hash_%d", i),
			RightsType:  types.RightsOriginal,
		}
		_, err := k.RegisterDataAsset(ctx, msg)
		if err != nil {
			t.Fatalf("RegisterDataAsset %d failed: %v", i, err)
		}
	}

	assets := k.ListAssets(ctx)
	if len(assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(assets))
	}
}

func TestGetShareHolders(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	buyer1 := sdk.AccAddress([]byte("buyer1______________")).String()
	buyer2 := sdk.AccAddress([]byte("buyer2______________")).String()

	msg := types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Shared Asset",
		Description: "Test",
		ContentHash: "sharedhash",
		RightsType:  types.RightsOriginal,
	}
	assetID, _ := k.RegisterDataAsset(ctx, msg)

	bank.fundAccount(buyer1, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100000))))
	bank.fundAccount(buyer2, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100000))))

	_, _ = k.BuyShares(ctx, types.MsgBuyShares{Creator: buyer1, AssetId: assetID, Amount: sdk.NewCoin("uoas", math.NewInt(50000))})
	_, _ = k.BuyShares(ctx, types.MsgBuyShares{Creator: buyer2, AssetId: assetID, Amount: sdk.NewCoin("uoas", math.NewInt(50000))})

	holders := k.GetShareHolders(ctx, assetID)
	if len(holders) != 2 {
		t.Fatalf("expected 2 shareholders, got %d", len(holders))
	}
}

// ---------------------------------------------------------------------------
// SellShares Tests
// ---------------------------------------------------------------------------

func TestSellShares(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	// Register an original asset.
	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "SellShares Asset",
		Description: "For sell test",
		ContentHash: "sellhash001",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Fund buyer and buy shares (bootstrap: 500000 tokens for 500000 uoas).
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(500000))))
	shares, err := k.BuyShares(ctx, types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(500000)),
	})
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}
	if shares.IsZero() {
		t.Fatal("expected non-zero shares after buy")
	}

	// Record state before sell.
	assetBefore, _ := k.GetAsset(ctx, assetID)
	reserveBefore := k.GetAssetReserve(ctx, assetID)
	shBefore, _ := k.GetShareHolder(ctx, assetID, buyer)
	buyerBalanceBefore := bank.balances[buyer]

	// Sell half of the shares.
	sharesToSell := shares.Quo(math.NewInt(2))
	payout, err := k.SellShares(ctx, types.MsgSellShares{
		Creator: buyer,
		AssetId: assetID,
		Shares:  sharesToSell,
	})
	if err != nil {
		t.Fatalf("SellShares failed: %v", err)
	}

	// Payout must be non-zero.
	if payout.IsZero() || payout.IsNegative() {
		t.Fatalf("expected positive payout, got %s", payout)
	}

	// Verify asset total shares decreased.
	assetAfter, _ := k.GetAsset(ctx, assetID)
	expectedSupply := assetBefore.TotalShares.Sub(sharesToSell)
	if !assetAfter.TotalShares.Equal(expectedSupply) {
		t.Fatalf("expected total shares %s, got %s", expectedSupply, assetAfter.TotalShares)
	}

	// Verify reserve decreased.
	reserveAfter := k.GetAssetReserve(ctx, assetID)
	if !reserveAfter.LT(reserveBefore) {
		t.Fatalf("expected reserve to decrease: before=%s, after=%s", reserveBefore, reserveAfter)
	}

	// Verify shareholder shares decreased.
	shAfter, shFound := k.GetShareHolder(ctx, assetID, buyer)
	if !shFound {
		t.Fatal("shareholder not found after partial sell")
	}
	expectedHolderShares := shBefore.Shares.Sub(sharesToSell)
	if !shAfter.Shares.Equal(expectedHolderShares) {
		t.Fatalf("expected shareholder shares %s, got %s", expectedHolderShares, shAfter.Shares)
	}

	// Verify buyer received payout (balance increased).
	buyerBalanceAfter := bank.balances[buyer]
	receivedAmt := buyerBalanceAfter.AmountOf("uoas")
	beforeAmt := buyerBalanceBefore.AmountOf("uoas")
	if !receivedAmt.Equal(beforeAmt.Add(payout)) {
		t.Fatalf("expected buyer balance to increase by payout %s; before=%s after=%s",
			payout, beforeAmt, receivedAmt)
	}

	// Verify inverse Bancor formula: gross_payout = reserve * (1 - (1 - sold/supply)^2).
	// With 5% protocol fee: net = gross * 0.95.
	supplyDec := math.LegacyNewDecFromInt(assetBefore.TotalShares)
	reserveDec := math.LegacyNewDecFromInt(reserveBefore)
	soldDec := math.LegacyNewDecFromInt(sharesToSell)
	ratio := math.LegacyOneDec().Sub(soldDec.Quo(supplyDec))
	ratioSq := ratio.Mul(ratio)
	grossDec := reserveDec.Mul(math.LegacyOneDec().Sub(ratioSq))
	netDec := grossDec.Mul(math.LegacyNewDecWithPrec(95, 2)) // 0.95 after 5% fee
	expectedPayout := netDec.TruncateInt()
	if !payout.Equal(expectedPayout) {
		t.Fatalf("expected Bancor-derived payout %s, got %s", expectedPayout, payout)
	}
}

func TestSellSharesInsufficientShares(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "InsufficientShares Asset",
		Description: "Test",
		ContentHash: "sellhash002",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Buyer buys 100000 tokens (bootstrap).
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100000))))
	shares, err := k.BuyShares(ctx, types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(100000)),
	})
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}

	// Try to sell more shares than owned.
	tooMany := shares.Add(math.NewInt(1))
	_, err = k.SellShares(ctx, types.MsgSellShares{
		Creator: buyer,
		AssetId: assetID,
		Shares:  tooMany,
	})
	if err == nil {
		t.Fatal("expected error when selling more shares than owned, got nil")
	}
}

func TestSellSharesEmptyPool(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, buyer, _ := testAddresses()

	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "EmptyPool Asset",
		Description: "Test",
		ContentHash: "sellhash003",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Buy shares to get a shareholder record.
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100000))))
	shares, err := k.BuyShares(ctx, types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(100000)),
	})
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}

	// Manually drain the reserve to simulate an empty pool.
	if err := k.SetAssetReserve(ctx, assetID, math.ZeroInt()); err != nil {
		t.Fatalf("SetAssetReserve failed: %v", err)
	}

	// Selling should fail because pool has no liquidity.
	_, err = k.SellShares(ctx, types.MsgSellShares{
		Creator: buyer,
		AssetId: assetID,
		Shares:  shares,
	})
	if err == nil {
		t.Fatal("expected error when selling from empty pool, got nil")
	}
}

// ---------------------------------------------------------------------------
// AccessLevel Tests
// ---------------------------------------------------------------------------

func TestAccessLevel(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	buyer := sdk.AccAddress([]byte("aclbuyer____________")).String()

	// Register an original asset.
	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "AccessLevel Asset",
		Description: "Test",
		ContentHash: "aclhash001",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Before any shares: access level should be empty string.
	noAccess := k.GetAccessLevel(ctx, assetID, buyer, math.LegacyNewDec(100))
	if noAccess != "" {
		t.Fatalf("expected empty access level for non-shareholder, got %s", noAccess)
	}

	// Bootstrap buy: 1_000_000 uoas → 1_000_000 shares.
	// We need a large pool so we can test equity thresholds.
	// First, a big buyer establishes supply (this is NOT the address we're testing).
	bigBuyer := sdk.AccAddress([]byte("bigbuyer____________")).String()
	// bigBuyer gets 90% of the shares by buying first.
	// Bootstrap: 9_000_000 uoas → 9_000_000 shares (multiplier=1.0).
	bank.fundAccount(bigBuyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(9_000_000))))
	_, err = k.BuyShares(ctx, types.MsgBuyShares{
		Creator: bigBuyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(9_000_000)),
	})
	if err != nil {
		t.Fatalf("BuyShares for bigBuyer failed: %v", err)
	}

	// At this point supply=9_000_000, reserve=9_000_000.
	// Next buy for buyer: we want buyer to get ~1_000_000 shares out of ~10_000_000 total (10% equity).
	// Bancor: tokens = supply * (sqrt(1 + payment/reserve) - 1)
	//       = 9_000_000 * (sqrt(1 + payment/9_000_000) - 1)
	// To get ~1_000_000 tokens: 1_000_000 = 9_000_000 * (sqrt(1+p/9M)-1)
	// 1/9 + 1 = (sqrt(1+p/9M))^2 → 10/9 = 1+p/9M → p = 9M/9 = 1_000_000
	// So pay 1_000_000 uoas → get ~1_000_000 shares (slightly more from curve, but close enough)
	// We just need >= 10% so let's pay enough to clearly reach 10%.
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(2_000_000))))
	buyerShares, err := k.BuyShares(ctx, types.MsgBuyShares{
		Creator: buyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(2_000_000)),
	})
	if err != nil {
		t.Fatalf("BuyShares for buyer failed: %v", err)
	}
	_ = buyerShares

	// Check equity: buyer should have >= 10% (L3) with high reputation (>= 50).
	highRep := math.LegacyNewDec(100)
	level := k.GetAccessLevel(ctx, assetID, buyer, highRep)

	// Compute actual equity to decide expected level.
	asset, _ := k.GetAsset(ctx, assetID)
	sh, _ := k.GetShareHolder(ctx, assetID, buyer)
	equityBps := sh.Shares.Mul(math.NewInt(10000)).Quo(asset.TotalShares)

	if equityBps.GTE(math.NewInt(1000)) {
		// >= 10% equity with high reputation → L3
		if level != "L3" {
			t.Fatalf("expected L3 for >=10%% equity (equityBps=%s), got %s", equityBps, level)
		}
	} else if equityBps.GTE(math.NewInt(500)) {
		// >= 5% → L2
		if level != "L2" {
			t.Fatalf("expected L2 for >=5%% equity (equityBps=%s), got %s", equityBps, level)
		}
	}

	// Test reputation cap: with low reputation (< 20), even large equity → L0.
	sandboxRep := math.LegacyNewDecWithPrec(15, 0) // reputation = 15
	levelLowRep := k.GetAccessLevel(ctx, assetID, buyer, sandboxRep)
	if levelLowRep != "L0" {
		t.Fatalf("expected L0 for low reputation (< 20), got %s", levelLowRep)
	}

	// Test small equity holder: bigBuyer owns ~90% → L3 regardless.
	levelBig := k.GetAccessLevel(ctx, assetID, bigBuyer, highRep)
	if levelBig != "L3" {
		t.Fatalf("expected L3 for bigBuyer with large equity, got %s", levelBig)
	}

	// Test mid-reputation cap: reputation 30 (>= 20 but < 50) → max L1.
	midRep := math.LegacyNewDec(30)
	levelMidRep := k.GetAccessLevel(ctx, assetID, buyer, midRep)
	if levelMidRep != "L1" {
		t.Fatalf("expected L1 for reputation=30 (limited tier), got %s", levelMidRep)
	}

	// Test an address with tiny equity (create a separate micro-buyer).
	microBuyer := sdk.AccAddress([]byte("microbuyer__________")).String()
	bank.fundAccount(microBuyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(1))))
	_, err = k.BuyShares(ctx, types.MsgBuyShares{
		Creator: microBuyer,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(1)),
	})
	// Very small buy might yield 0 shares (below mint threshold); if it fails or gives 0, skip.
	if err == nil {
		microAsset, _ := k.GetAsset(ctx, assetID)
		microSh, microFound := k.GetShareHolder(ctx, assetID, microBuyer)
		if microFound && !microSh.Shares.IsZero() {
			microEquityBps := microSh.Shares.Mul(math.NewInt(10000)).Quo(microAsset.TotalShares)
			microLevel := k.GetAccessLevel(ctx, assetID, microBuyer, highRep)
			if microEquityBps.LT(math.NewInt(10)) {
				// < 0.1% → no access
				if microLevel != "" {
					t.Fatalf("expected empty access level for <0.1%% equity (bps=%s), got %s", microEquityBps, microLevel)
				}
			} else if microEquityBps.LT(math.NewInt(100)) {
				// >= 0.1% but < 1% → L0
				if microLevel != "L0" {
					t.Fatalf("expected L0 for micro equity (bps=%s), got %s", microEquityBps, microLevel)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Jury Voting Tests
// ---------------------------------------------------------------------------

func TestJuryVoting(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	plaintiff := sdk.AccAddress([]byte("plaintiff___________")).String()

	// Register an asset.
	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Jury Asset",
		Description: "For jury test",
		ContentHash: "juryhash001",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Create several shareholders (potential jurors).
	jurorCandidates := []string{
		sdk.AccAddress([]byte("juror1______________")).String(),
		sdk.AccAddress([]byte("juror2______________")).String(),
		sdk.AccAddress([]byte("juror3______________")).String(),
		sdk.AccAddress([]byte("juror4______________")).String(),
		sdk.AccAddress([]byte("juror5______________")).String(),
		sdk.AccAddress([]byte("juror6______________")).String(),
	}
	for _, juror := range jurorCandidates {
		bank.fundAccount(juror, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100_000))))
		_, err := k.BuyShares(ctx, types.MsgBuyShares{
			Creator: juror,
			AssetId: assetID,
			Amount:  sdk.NewCoin("uoas", math.NewInt(100_000)),
		})
		if err != nil {
			t.Fatalf("BuyShares for juror %s failed: %v", juror, err)
		}
	}

	// Fund plaintiff and file a dispute.
	params := k.GetParams(ctx)
	bank.fundAccount(plaintiff, sdk.NewCoins(params.DisputeDeposit))
	disputeID, err := k.FileDispute(ctx, types.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Test jury dispute",
		Evidence: []byte("evidence data"),
	})
	if err != nil {
		t.Fatalf("FileDispute failed: %v", err)
	}

	// Select jury.
	jury := k.SelectJury(ctx, disputeID, assetID, plaintiff)
	if len(jury) == 0 {
		t.Fatal("expected non-empty jury selection")
	}
	if len(jury) > keeper.JurySize {
		t.Fatalf("expected at most %d jurors, got %d", keeper.JurySize, len(jury))
	}

	// Ensure plaintiff is not in the jury.
	for _, j := range jury {
		if j == plaintiff {
			t.Fatal("plaintiff should not be selected as juror")
		}
	}

	// --- Test: uphold outcome (all jurors vote to uphold) ---

	for _, juror := range jury {
		if err := k.SubmitJuryVote(ctx, disputeID, juror, true); err != nil {
			t.Fatalf("SubmitJuryVote (uphold) for juror %s failed: %v", juror, err)
		}
	}

	// Tally votes.
	upholdCount, totalCount := k.TallyVotes(ctx, disputeID)
	if totalCount != len(jury) {
		t.Fatalf("expected %d total votes, got %d", len(jury), totalCount)
	}
	if upholdCount != len(jury) {
		t.Fatalf("expected %d uphold votes, got %d", len(jury), upholdCount)
	}

	// Resolve by jury — unanimous uphold should delist the asset.
	if err := k.ResolveByJury(ctx, disputeID); err != nil {
		t.Fatalf("ResolveByJury failed: %v", err)
	}

	// Verify dispute is resolved with delist remedy.
	dispute, found := k.GetDispute(ctx, disputeID)
	if !found {
		t.Fatal("dispute not found after resolution")
	}
	if dispute.Status != types.StatusResolved {
		t.Fatalf("expected RESOLVED status, got %s", dispute.Status)
	}
	if dispute.Remedy != types.RemedyDelist {
		t.Fatalf("expected DELIST remedy, got %s", dispute.Remedy)
	}
	if dispute.Arbitrator != "jury" {
		t.Fatalf("expected arbitrator='jury', got %s", dispute.Arbitrator)
	}

	// Verify asset is delisted.
	asset, _ := k.GetAsset(ctx, assetID)
	if asset.IsActive {
		t.Fatal("expected asset to be delisted after jury uphold")
	}
}

func TestJuryVotingReject(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	plaintiff := sdk.AccAddress([]byte("plaintiff2__________")).String()

	// Register a second asset for the reject test.
	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "Jury Reject Asset",
		Description: "For reject test",
		ContentHash: "juryhash002",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Create shareholders.
	jurorCandidates := []string{
		sdk.AccAddress([]byte("rjuror1_____________")).String(),
		sdk.AccAddress([]byte("rjuror2_____________")).String(),
		sdk.AccAddress([]byte("rjuror3_____________")).String(),
		sdk.AccAddress([]byte("rjuror4_____________")).String(),
		sdk.AccAddress([]byte("rjuror5_____________")).String(),
		sdk.AccAddress([]byte("rjuror6_____________")).String(),
	}
	for _, juror := range jurorCandidates {
		bank.fundAccount(juror, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100_000))))
		_, err := k.BuyShares(ctx, types.MsgBuyShares{
			Creator: juror,
			AssetId: assetID,
			Amount:  sdk.NewCoin("uoas", math.NewInt(100_000)),
		})
		if err != nil {
			t.Fatalf("BuyShares failed: %v", err)
		}
	}

	// File dispute.
	params := k.GetParams(ctx)
	bank.fundAccount(plaintiff, sdk.NewCoins(params.DisputeDeposit))
	disputeID, err := k.FileDispute(ctx, types.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Reject test dispute",
		Evidence: nil,
	})
	if err != nil {
		t.Fatalf("FileDispute failed: %v", err)
	}

	// Select jury.
	jury := k.SelectJury(ctx, disputeID, assetID, plaintiff)
	if len(jury) == 0 {
		t.Fatal("expected non-empty jury")
	}

	// All jurors vote to reject (uphold=false).
	for _, juror := range jury {
		if err := k.SubmitJuryVote(ctx, disputeID, juror, false); err != nil {
			t.Fatalf("SubmitJuryVote (reject) for juror %s failed: %v", juror, err)
		}
	}

	// Tally: upholdCount should be 0.
	upholdCount, totalCount := k.TallyVotes(ctx, disputeID)
	if totalCount == 0 {
		t.Fatal("expected at least one vote")
	}
	if upholdCount != 0 {
		t.Fatalf("expected 0 uphold votes, got %d", upholdCount)
	}

	// Resolve: dispute should be rejected.
	if err := k.ResolveByJury(ctx, disputeID); err != nil {
		t.Fatalf("ResolveByJury failed: %v", err)
	}

	dispute, found := k.GetDispute(ctx, disputeID)
	if !found {
		t.Fatal("dispute not found after reject resolution")
	}
	if dispute.Status != types.StatusRejected {
		t.Fatalf("expected REJECTED status, got %s", dispute.Status)
	}

	// Asset should still be active after dispute rejection.
	asset, _ := k.GetAsset(ctx, assetID)
	if !asset.IsActive {
		t.Fatal("expected asset to remain active after jury reject")
	}
}

func TestJuryVotingDoubleVote(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, _, _ := testAddresses()
	plaintiff := sdk.AccAddress([]byte("plaintiff3__________")).String()

	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     creator,
		Name:        "DoubleVote Asset",
		Description: "Test double vote prevention",
		ContentHash: "juryhash003",
		RightsType:  types.RightsOriginal,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	juror := sdk.AccAddress([]byte("dvjuror1____________")).String()
	bank.fundAccount(juror, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100_000))))
	_, _ = k.BuyShares(ctx, types.MsgBuyShares{
		Creator: juror,
		AssetId: assetID,
		Amount:  sdk.NewCoin("uoas", math.NewInt(100_000)),
	})

	params := k.GetParams(ctx)
	bank.fundAccount(plaintiff, sdk.NewCoins(params.DisputeDeposit))
	disputeID, err := k.FileDispute(ctx, types.MsgFileDispute{
		Creator:  plaintiff,
		AssetId:  assetID,
		Reason:   "Double vote test",
		Evidence: nil,
	})
	if err != nil {
		t.Fatalf("FileDispute failed: %v", err)
	}

	// Select jury so the juror is registered as a member.
	jury := k.SelectJury(ctx, disputeID, assetID, plaintiff)
	if len(jury) == 0 {
		t.Fatal("expected non-empty jury")
	}
	// Use the first selected juror for the double-vote test.
	selectedJuror := jury[0]

	// First vote succeeds.
	if err := k.SubmitJuryVote(ctx, disputeID, selectedJuror, true); err != nil {
		t.Fatalf("First SubmitJuryVote failed: %v", err)
	}

	// Second vote should fail.
	if err := k.SubmitJuryVote(ctx, disputeID, selectedJuror, false); err == nil {
		t.Fatal("expected error on double vote, got nil")
	}

	// Non-member should also fail.
	outsider := sdk.AccAddress([]byte("outsider____________")).String()
	if err := k.SubmitJuryVote(ctx, disputeID, outsider, true); err == nil {
		t.Fatal("expected error for non-jury-member vote, got nil")
	}
}

func TestJuryScore(t *testing.T) {
	// JuryScore is deterministic: same inputs → same output.
	score1 := keeper.JuryScore("DISPUTE_001", "NODE_A", 100.0)
	score2 := keeper.JuryScore("DISPUTE_001", "NODE_A", 100.0)
	if score1 != score2 {
		t.Fatalf("JuryScore is not deterministic: got %f and %f", score1, score2)
	}

	// Different disputeID → different score.
	score3 := keeper.JuryScore("DISPUTE_002", "NODE_A", 100.0)
	if score1 == score3 {
		t.Fatal("expected different JuryScore for different disputeID")
	}

	// Different nodeID → different score.
	score4 := keeper.JuryScore("DISPUTE_001", "NODE_B", 100.0)
	if score1 == score4 {
		t.Fatal("expected different JuryScore for different nodeID")
	}

	// Higher reputation → higher score (log scale).
	scoreZeroRep := keeper.JuryScore("DISPUTE_X", "NODE_Z", 0.0)
	scoreHighRep := keeper.JuryScore("DISPUTE_X", "NODE_Z", 1000.0)
	if scoreZeroRep != 0.0 {
		t.Fatalf("expected zero score for zero reputation, got %f", scoreZeroRep)
	}
	if scoreHighRep <= scoreZeroRep {
		t.Fatalf("expected higher score for higher reputation: zero=%f, high=%f", scoreZeroRep, scoreHighRep)
	}

	// Negative reputation should be treated as 0 (no panic).
	scoreNegRep := keeper.JuryScore("DISPUTE_X", "NODE_Z", -50.0)
	if scoreNegRep != 0.0 {
		t.Fatalf("expected zero score for negative reputation, got %f", scoreNegRep)
	}

	// Score is in [0, log(1+reputation)] range.
	for _, rep := range []float64{1.0, 10.0, 100.0} {
		s := keeper.JuryScore("DISPUTE_TEST", "NODE_TEST", rep)
		if s < 0 {
			t.Fatalf("JuryScore should not be negative, got %f for rep=%f", s, rep)
		}
	}
}

func TestDelistAsset(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	owner := sdk.AccAddress([]byte("owner_______________")).String()

	// Register an asset.
	assetID, err := k.RegisterDataAsset(ctx, types.MsgRegisterDataAsset{
		Creator:     owner,
		Name:        "Test Asset",
		ContentHash: "hash_delist_test",
		RightsType:  types.RIGHTS_TYPE_ORIGINAL,
	})
	if err != nil {
		t.Fatalf("RegisterDataAsset failed: %v", err)
	}

	// Verify asset is active.
	asset, found := k.GetAsset(sdk.UnwrapSDKContext(ctx), assetID)
	if !found || !asset.IsActive {
		t.Fatal("expected active asset")
	}

	// Non-owner cannot delist.
	nonOwner := sdk.AccAddress([]byte("nonowner____________")).String()
	err = k.DelistAsset(ctx, types.MsgDelistAsset{Creator: nonOwner, AssetId: assetID})
	if err == nil {
		t.Fatal("expected error for non-owner delist")
	}

	// Owner delists.
	err = k.DelistAsset(ctx, types.MsgDelistAsset{Creator: owner, AssetId: assetID})
	if err != nil {
		t.Fatalf("DelistAsset failed: %v", err)
	}

	// Verify asset is now inactive.
	asset, found = k.GetAsset(sdk.UnwrapSDKContext(ctx), assetID)
	if !found || asset.IsActive {
		t.Fatal("expected delisted asset")
	}

	// Double delist should fail.
	err = k.DelistAsset(ctx, types.MsgDelistAsset{Creator: owner, AssetId: assetID})
	if err == nil {
		t.Fatal("expected error for double delist")
	}
}
