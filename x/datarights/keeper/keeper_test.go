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

func (m *mockSettlementKeeper) BuyShares(_ context.Context, _ string, _ string, _ math.Int) (math.Int, error) {
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

func TestBuySharesWithDiminishingReturns(t *testing.T) {
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

	// First purchase: total shares = 0 (< 1000), rate = 100%, multiplier = 1.0
	buyMsg := types.MsgBuyShares{
		Creator: buyer,
		AssetID: assetID,
		Amount:  payment,
	}
	shares1, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares (1st) failed: %v", err)
	}
	// Expected: 100000 * 10000 / 10000 * 1.0 = 100000
	expectedShares1 := math.NewInt(100000)
	if !shares1.Equal(expectedShares1) {
		t.Fatalf("expected shares1 %s, got %s", expectedShares1, shares1)
	}

	// Verify asset total shares updated.
	asset, _ := k.GetAsset(ctx, assetID)
	if !asset.TotalShares.Equal(expectedShares1) {
		t.Fatalf("expected total shares %s, got %s", expectedShares1, asset.TotalShares)
	}

	// Second purchase: total shares = 100000 (> 10000), rate = 40%, multiplier = 1.0
	shares2, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares (2nd) failed: %v", err)
	}
	// Expected: 100000 * 4000 / 10000 * 1.0 = 40000
	expectedShares2 := math.NewInt(40000)
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
		AssetID: assetID,
		Amount:  payment,
	}
	shares, err := k.BuyShares(ctx, buyMsg)
	if err != nil {
		t.Fatalf("BuyShares failed: %v", err)
	}
	// Rate = 100% (total < 1000), multiplier = 0.3
	// Expected: 100000 * 10000 / 10000 * 0.3 = 30000
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
		AssetID: assetID,
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
		AssetID:  assetID,
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
		DisputeID: disputeID,
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
		AssetID:  assetID,
		Reason:   "Test reason",
		Evidence: nil,
	}
	disputeID, _ := k.FileDispute(ctx, disputeMsg)

	// Non-arbitrator tries to resolve.
	resolveMsg := types.MsgResolveDispute{
		Creator:   notArbitrator,
		DisputeID: disputeID,
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

	_, _ = k.BuyShares(ctx, types.MsgBuyShares{Creator: buyer1, AssetID: assetID, Amount: sdk.NewCoin("uoas", math.NewInt(50000))})
	_, _ = k.BuyShares(ctx, types.MsgBuyShares{Creator: buyer2, AssetID: assetID, Amount: sdk.NewCoin("uoas", math.NewInt(50000))})

	holders := k.GetShareHolders(ctx, assetID)
	if len(holders) != 2 {
		t.Fatalf("expected 2 shareholders, got %d", len(holders))
	}
}
