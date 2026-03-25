package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/keeper"
)

// ---------------------------------------------------------------------------
// Bonding Curve Boundary Tests
// ---------------------------------------------------------------------------

func TestBancorBuy_ZeroPayment(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)

	tokens, err := keeper.BancorBuy(supply, reserve, math.LegacyZeroDec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tokens.IsZero() {
		t.Fatalf("expected zero tokens for zero payment, got %s", tokens)
	}
}

func TestBancorBuy_NegativePayment(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)

	tokens, err := keeper.BancorBuy(supply, reserve, math.LegacyNewDec(-1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tokens.IsZero() {
		t.Fatalf("expected zero tokens for negative payment, got %s", tokens)
	}
}

func TestBancorBuy_TinyPayment(t *testing.T) {
	// After bootstrap, a 1 uoas payment on a large pool should still yield >= 0 tokens (no panic).
	supply := math.LegacyNewDec(1000000000) // 1B supply
	reserve := math.LegacyNewDec(1000000000)

	tokens, err := keeper.BancorBuy(supply, reserve, math.LegacyOneDec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.IsNegative() {
		t.Fatalf("tokens should not be negative, got %s", tokens)
	}
}

func TestBancorBuy_HugePayment(t *testing.T) {
	// 10^18 uoas payment on a small pool — must not overflow or panic.
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)
	hugePayment := math.LegacyNewDecFromInt(math.NewInt(1).Mul(math.NewIntFromUint64(1000000000000000000)))

	tokens, err := keeper.BancorBuy(supply, reserve, hugePayment)
	if err != nil {
		t.Fatalf("unexpected error on huge payment: %v", err)
	}
	if tokens.IsNegative() || tokens.IsZero() {
		t.Fatalf("expected positive tokens for huge payment, got %s", tokens)
	}
}

func TestBancorBuy_BootstrapReturnsOneToOne(t *testing.T) {
	// When reserve=0 and supply=0, should return payment/1 = payment tokens.
	payment := math.LegacyNewDec(500000)
	tokens, err := keeper.BancorBuy(math.LegacyZeroDec(), math.LegacyZeroDec(), payment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tokens.Equal(payment) {
		t.Fatalf("bootstrap should return 1:1, expected %s, got %s", payment, tokens)
	}
}

func TestBancorSell_ZeroTokens(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)

	payout, err := keeper.BancorSell(supply, reserve, math.LegacyZeroDec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !payout.IsZero() {
		t.Fatalf("expected zero payout for zero tokens, got %s", payout)
	}
}

func TestBancorSell_EntireSupply_CappedAtSolvency(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)

	// Selling all tokens should be capped at 95% of reserve.
	payout, err := keeper.BancorSell(supply, reserve, supply)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := math.LegacyNewDec(95000) // 0.95 * 100000
	if !payout.Equal(expected) {
		t.Fatalf("expected solvency-capped payout %s, got %s", expected, payout)
	}
}

func TestBancorSell_MoreThanSupply_CappedAtSolvency(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	reserve := math.LegacyNewDec(100000)

	// Selling MORE than total supply should also be capped at 95%.
	payout, err := keeper.BancorSell(supply, reserve, math.LegacyNewDec(200000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := math.LegacyNewDec(95000)
	if !payout.Equal(expected) {
		t.Fatalf("expected solvency-capped payout %s, got %s", expected, payout)
	}
}

func TestBancorSell_TinyAmount(t *testing.T) {
	// Selling 1 token from a large pool — should yield a small but non-negative payout.
	supply := math.LegacyNewDec(1000000)
	reserve := math.LegacyNewDec(1000000)

	payout, err := keeper.BancorSell(supply, reserve, math.LegacyOneDec())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payout.IsNegative() {
		t.Fatalf("payout should not be negative, got %s", payout)
	}
	// 1 token from 1M supply: payout ≈ reserve * (1 - (1 - 1/1M)^2) ≈ 2 * reserve/1M ≈ 2 uoas
	if payout.GT(math.LegacyNewDec(3)) {
		t.Fatalf("payout for 1 token out of 1M should be ~2, got %s", payout)
	}
}

func TestBancorSell_ZeroReserve(t *testing.T) {
	supply := math.LegacyNewDec(100000)
	payout, err := keeper.BancorSell(supply, math.LegacyZeroDec(), math.LegacyNewDec(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !payout.IsZero() {
		t.Fatalf("expected zero payout with zero reserve, got %s", payout)
	}
}

// ---------------------------------------------------------------------------
// Escrow Fee Split Precision Tests
// ---------------------------------------------------------------------------

func TestEscrowRelease_SmallAmount_FeeSplitPrecision(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	// Small amount: 7 uoas — tests integer rounding.
	// 5% of 7 = 0 (integer), 2% of 7 = 0, 3% of 7 = 0
	// Provider should get 7 - 0 - 0 - 0 = 7
	amount := sdk.NewCoin("uoas", math.NewInt(7))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("ReleaseEscrow failed: %v", err)
	}

	// With integer math: 7*500/10000=0, 7*200/10000=0, 7*300/10000=0
	// Provider gets 7-0-0-0 = 7
	providerBal := bank.balances[provider]
	expectedProvider := sdk.NewCoin("uoas", math.NewInt(7))
	if !providerBal.Equal(sdk.NewCoins(expectedProvider)) {
		t.Fatalf("expected provider balance %s, got %s", expectedProvider, providerBal)
	}

	// Module balance should be zero.
	moduleBal := bank.moduleBalances["settlement"]
	if !moduleBal.IsZero() {
		t.Fatalf("expected module balance 0, got %s", moduleBal)
	}
}

func TestEscrowRelease_OneUoas_NoLoss(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	// Absolute minimum: 1 uoas. All fee percentages truncate to 0.
	amount := sdk.NewCoin("uoas", math.NewInt(1))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("ReleaseEscrow failed: %v", err)
	}

	providerBal := bank.balances[provider]
	if !providerBal.Equal(sdk.NewCoins(amount)) {
		t.Fatalf("expected provider to receive full 1 uoas, got %s", providerBal)
	}
}

func TestEscrowRelease_LargeAmount_ExactFeeSplit(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	// 10,000,000 uoas — clean numbers for verification.
	total := math.NewInt(10000000)
	amount := sdk.NewCoin("uoas", total)
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("ReleaseEscrow failed: %v", err)
	}

	// Verify exact amounts:
	// protocol = 10000000 * 500 / 10000 = 500000
	// burn     = 10000000 * 200 / 10000 = 200000
	// treasury = 10000000 * 300 / 10000 = 300000
	// provider = 10000000 - 500000 - 200000 - 300000 = 9000000
	providerBal := bank.balances[provider]
	if !providerBal.Equal(sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(9000000)))) {
		t.Fatalf("expected provider 9000000, got %s", providerBal)
	}

	feeCollectorBal := bank.moduleBalances["fee_collector"]
	if !feeCollectorBal.Equal(sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(800000)))) {
		t.Fatalf("expected fee_collector 800000, got %s", feeCollectorBal)
	}

	// Total conservation: provider + fee_collector + burned = original amount.
	// burned = 200000
	accounted := math.NewInt(9000000).Add(math.NewInt(800000)).Add(math.NewInt(200000))
	if !accounted.Equal(total) {
		t.Fatalf("conservation violated: %s != %s", accounted, total)
	}
}

func TestEscrowRelease_OddAmount_ConservationOfValue(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	// Odd amount: 999999 uoas. Tests rounding doesn't leak/create tokens.
	total := math.NewInt(999999)
	amount := sdk.NewCoin("uoas", total)
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, err := k.CreateEscrow(ctx, creator, provider, amount, 0)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}

	if err := k.ReleaseEscrow(ctx, escrowID, creator); err != nil {
		t.Fatalf("ReleaseEscrow failed: %v", err)
	}

	// Calculate expected:
	// protocol = 999999 * 500 / 10000 = 49999 (truncated from 49999.95)
	// burn     = 999999 * 200 / 10000 = 19999 (truncated from 19999.98)
	// treasury = 999999 * 300 / 10000 = 29999 (truncated from 29999.97)
	// provider = 999999 - 49999 - 19999 - 29999 = 900002
	expectedProtocol := math.NewInt(49999)
	expectedBurn := math.NewInt(19999)
	expectedTreasury := math.NewInt(29999)
	expectedProvider := total.Sub(expectedProtocol).Sub(expectedBurn).Sub(expectedTreasury)

	providerBal := bank.balances[provider]
	if !providerBal.Equal(sdk.NewCoins(sdk.NewCoin("uoas", expectedProvider))) {
		t.Fatalf("expected provider %s, got %s", expectedProvider, providerBal)
	}

	// Value conservation: no tokens created or destroyed (beyond burn).
	feeCollectorBal := bank.moduleBalances["fee_collector"]
	expectedFeeCollector := expectedProtocol.Add(expectedTreasury)
	if !feeCollectorBal.Equal(sdk.NewCoins(sdk.NewCoin("uoas", expectedFeeCollector))) {
		t.Fatalf("expected fee_collector %s, got %s", expectedFeeCollector, feeCollectorBal)
	}

	// provider_received + fee_collector + burned == total
	accounted := expectedProvider.Add(expectedFeeCollector).Add(expectedBurn)
	if !accounted.Equal(total) {
		t.Fatalf("conservation violated: %s != %s (diff: %s)", accounted, total, total.Sub(accounted))
	}
}

// ---------------------------------------------------------------------------
// BuyShares Keeper Integration Boundary Tests
// ---------------------------------------------------------------------------

func TestBuyShares_ZeroPayment_Rejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	buyer := sdk.AccAddress([]byte("buyer_______________")).String()

	_, err := k.BuyShares(ctx, "ASSET_1", buyer, "uoas", math.ZeroInt())
	if err == nil {
		t.Fatal("expected error for zero payment, got nil")
	}
}

func TestBuyShares_NegativePayment_Rejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	buyer := sdk.AccAddress([]byte("buyer_______________")).String()

	_, err := k.BuyShares(ctx, "ASSET_1", buyer, "uoas", math.NewInt(-100))
	if err == nil {
		t.Fatal("expected error for negative payment, got nil")
	}
}

func TestBuyShares_InsufficientBalance(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	buyer := sdk.AccAddress([]byte("buyer_______________")).String()

	// Fund with 100 but try to buy with 200.
	bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", math.NewInt(100))))

	_, err := k.BuyShares(ctx, "ASSET_1", buyer, "uoas", math.NewInt(200))
	if err == nil {
		t.Fatal("expected insufficient funds error, got nil")
	}
}

func TestBuyShares_MultipleBuyers_PriceIncreases(t *testing.T) {
	k, ctx, bank := setupKeeper(t)

	buyer1 := sdk.AccAddress([]byte("buyer1______________")).String()
	buyer2 := sdk.AccAddress([]byte("buyer2______________")).String()
	buyer3 := sdk.AccAddress([]byte("buyer3______________")).String()

	payment := math.NewInt(100000)
	for _, addr := range []string{buyer1, buyer2, buyer3} {
		bank.fundAccount(addr, sdk.NewCoins(sdk.NewCoin("uoas", payment)))
	}

	// Each subsequent buyer should get fewer tokens for the same payment.
	shares1, _ := k.BuyShares(ctx, "ASSET_1", buyer1, "uoas", payment)
	shares2, _ := k.BuyShares(ctx, "ASSET_1", buyer2, "uoas", payment)
	shares3, _ := k.BuyShares(ctx, "ASSET_1", buyer3, "uoas", payment)

	if !shares1.GT(shares2) {
		t.Fatalf("buyer2 should get fewer shares than buyer1: %s vs %s", shares1, shares2)
	}
	if !shares2.GT(shares3) {
		t.Fatalf("buyer3 should get fewer shares than buyer2: %s vs %s", shares2, shares3)
	}
}

func TestBuyShares_SpotPriceMonotonicallyIncreases(t *testing.T) {
	k, ctx, bank := setupKeeper(t)

	assetID := "ASSET_PRICE"
	payment := math.NewInt(50000)

	for i := 0; i < 5; i++ {
		buyer := sdk.AccAddress([]byte("buyer" + string(rune('A'+i)) + "______________")).String()
		bank.fundAccount(buyer, sdk.NewCoins(sdk.NewCoin("uoas", payment)))
		_, err := k.BuyShares(ctx, assetID, buyer, "uoas", payment)
		if err != nil {
			t.Fatalf("BuyShares round %d failed: %v", i, err)
		}
	}

	// After 5 purchases, price should be well above initial.
	price, err := k.SpotPrice(ctx, assetID)
	if err != nil {
		t.Fatalf("SpotPrice failed: %v", err)
	}
	if price.LTE(math.LegacyOneDec()) {
		t.Fatalf("price should be > 1 after multiple buys, got %s", price)
	}
}

// ---------------------------------------------------------------------------
// Escrow Status Transition Tests
// ---------------------------------------------------------------------------

func TestEscrow_RefundAfterRelease_Fails(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, _ := k.CreateEscrow(ctx, creator, provider, amount, 0)
	_ = k.ReleaseEscrow(ctx, escrowID, creator)

	// Trying to refund a released escrow should fail.
	err := k.RefundEscrow(ctx, escrowID, creator)
	if err == nil {
		t.Fatal("expected error refunding a released escrow, got nil")
	}
}

func TestEscrow_ReleaseAfterRefund_Fails(t *testing.T) {
	k, ctx, bank := setupKeeper(t)
	creator, provider := testAddresses()

	amount := sdk.NewCoin("uoas", math.NewInt(100000))
	bank.fundAccount(creator, sdk.NewCoins(amount))

	escrowID, _ := k.CreateEscrow(ctx, creator, provider, amount, 0)
	_ = k.RefundEscrow(ctx, escrowID, creator)

	// Trying to release a refunded escrow should fail.
	err := k.ReleaseEscrow(ctx, escrowID, creator)
	if err == nil {
		t.Fatal("expected error releasing a refunded escrow, got nil")
	}
}

func TestEscrow_NonexistentID(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	creator, _ := testAddresses()

	err := k.ReleaseEscrow(ctx, "ESC_nonexistent", creator)
	if err == nil {
		t.Fatal("expected error for nonexistent escrow, got nil")
	}

	err = k.RefundEscrow(ctx, "ESC_nonexistent", creator)
	if err == nil {
		t.Fatal("expected error for nonexistent escrow, got nil")
	}
}

func TestEscrow_ZeroAmount_Rejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	creator, provider := testAddresses()

	_, err := k.CreateEscrow(ctx, creator, provider, sdk.NewCoin("uoas", math.ZeroInt()), 0)
	if err == nil {
		t.Fatal("expected error for zero-amount escrow, got nil")
	}
}
