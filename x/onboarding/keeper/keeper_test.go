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

	"github.com/oasyce/chain/x/onboarding/keeper"
	"github.com/oasyce/chain/x/onboarding/types"
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

func (m *mockBankKeeper) MintCoins(_ context.Context, moduleName string, amounts sdk.Coins) error {
	m.moduleBalances[moduleName] = m.moduleBalances[moduleName].Add(amounts...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amounts sdk.Coins) error {
	if !m.moduleBalances[moduleName].IsAllGTE(amounts) {
		return types.ErrInsufficientFunds.Wrap("mock: insufficient module funds for burn")
	}
	m.moduleBalances[moduleName] = m.moduleBalances[moduleName].Sub(amounts...)
	return nil
}

// ---------------------------------------------------------------------------
// Setup
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
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, "authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Time: time.Now().UTC(),
	}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return k, ctx, bankKeeper
}

// findValidNonce brute-forces a valid PoW nonce for testing.
func findValidNonce(address string, difficulty uint32) uint64 {
	var nonce uint64
	for {
		if keeper.VerifyPoW(address, nonce, difficulty) {
			return nonce
		}
		nonce++
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestVerifyPoW(t *testing.T) {
	addr := "oasyce1testaddr"

	// Difficulty 0 should always pass.
	if !keeper.VerifyPoW(addr, 0, 0) {
		t.Fatal("difficulty 0 should always pass")
	}

	// Find a valid nonce for difficulty 8.
	nonce := findValidNonce(addr, 8)
	if !keeper.VerifyPoW(addr, nonce, 8) {
		t.Fatalf("nonce %d should be valid for difficulty 8", nonce)
	}

	// Same nonce should fail for much higher difficulty.
	if keeper.VerifyPoW(addr, nonce, 128) {
		t.Fatal("should not pass difficulty 128 easily")
	}
}

func TestSelfRegisterAndRepay(t *testing.T) {
	k, sdkCtx, bank := setupKeeper(t)
	ctx := sdkCtx
	user := sdk.AccAddress([]byte("user____________________")).String()

	params := k.GetParams(sdkCtx)

	// Halving enforces min difficulty 16 at epoch 0 regardless of params.
	// Find a valid nonce for the effective difficulty.
	nonce := findValidNonce(user, 16)

	// Self-register.
	amount, err := k.SelfRegister(ctx, types.MsgSelfRegister{
		Creator: user,
		Nonce:   nonce,
	})
	if err != nil {
		t.Fatalf("SelfRegister failed: %v", err)
	}
	if !amount.Equal(params.AirdropAmount.Amount) {
		t.Fatalf("expected airdrop %s, got %s", params.AirdropAmount.Amount, amount)
	}

	// Verify user received tokens.
	userBalance := bank.balances[user].AmountOf("uoas")
	if !userBalance.Equal(params.AirdropAmount.Amount) {
		t.Fatalf("user balance: expected %s, got %s", params.AirdropAmount.Amount, userBalance)
	}

	// Verify registration is ACTIVE.
	reg, found := k.GetRegistration(sdkCtx, user)
	if !found {
		t.Fatal("registration not found")
	}
	if reg.Status != types.REGISTRATION_STATUS_ACTIVE {
		t.Fatalf("expected ACTIVE, got %v", reg.Status)
	}

	// Double registration should fail.
	_, err = k.SelfRegister(ctx, types.MsgSelfRegister{Creator: user, Nonce: 99})
	if err == nil {
		t.Fatal("expected error for double registration")
	}

	// Partial repay.
	halfDebt := params.AirdropAmount.Amount.Quo(math.NewInt(2))
	remaining, err := k.RepayDebt(ctx, types.MsgRepayDebt{
		Creator: user,
		Amount:  halfDebt,
	})
	if err != nil {
		t.Fatalf("RepayDebt failed: %v", err)
	}
	expectedRemaining := params.AirdropAmount.Amount.Sub(halfDebt)
	if !remaining.Equal(expectedRemaining) {
		t.Fatalf("remaining debt: expected %s, got %s", expectedRemaining, remaining)
	}

	// Still ACTIVE.
	reg, _ = k.GetRegistration(sdkCtx, user)
	if reg.Status != types.REGISTRATION_STATUS_ACTIVE {
		t.Fatalf("expected ACTIVE after partial repay, got %v", reg.Status)
	}

	// Full repay.
	remaining, err = k.RepayDebt(ctx, types.MsgRepayDebt{
		Creator: user,
		Amount:  expectedRemaining,
	})
	if err != nil {
		t.Fatalf("RepayDebt (full) failed: %v", err)
	}
	if !remaining.IsZero() {
		t.Fatalf("expected zero remaining, got %s", remaining)
	}

	// Verify REPAID status.
	reg, _ = k.GetRegistration(sdkCtx, user)
	if reg.Status != types.REGISTRATION_STATUS_REPAID {
		t.Fatalf("expected REPAID, got %v", reg.Status)
	}
}

func TestSelfRegisterInvalidPoW(t *testing.T) {
	k, sdkCtx, _ := setupKeeper(t)
	ctx := sdkCtx
	user := sdk.AccAddress([]byte("user____________________")).String()

	// Default difficulty is 16 — random nonce will almost certainly fail.
	_, err := k.SelfRegister(ctx, types.MsgSelfRegister{
		Creator: user,
		Nonce:   0,
	})
	if err == nil {
		t.Fatal("expected error for invalid PoW")
	}
}

func TestHalvingEpoch(t *testing.T) {
	tests := []struct {
		totalRegs uint64
		epoch     uint32
	}{
		{0, 0},
		{1, 0},
		{10_000, 0},
		{10_001, 1},
		{50_000, 1},
		{50_001, 2},
		{200_000, 2},
		{200_001, 3},
		{1_000_000, 3},
	}
	for _, tt := range tests {
		got := keeper.HalvingEpoch(tt.totalRegs)
		if got != tt.epoch {
			t.Errorf("HalvingEpoch(%d) = %d, want %d", tt.totalRegs, got, tt.epoch)
		}
	}
}

func TestHalvingAirdrop(t *testing.T) {
	tests := []struct {
		epoch    uint32
		expected int64
	}{
		{0, 20_000_000}, // 20 OAS
		{1, 10_000_000}, // 10 OAS
		{2, 5_000_000},  // 5 OAS
		{3, 2_500_000},  // 2.5 OAS
	}
	for _, tt := range tests {
		got := keeper.HalvingAirdrop(tt.epoch)
		if !got.Equal(math.NewInt(tt.expected)) {
			t.Errorf("HalvingAirdrop(%d) = %s, want %d", tt.epoch, got, tt.expected)
		}
	}
}

func TestHalvingDifficulty(t *testing.T) {
	tests := []struct {
		epoch    uint32
		expected uint32
	}{
		{0, 16},
		{1, 18},
		{2, 20},
		{3, 22},
	}
	for _, tt := range tests {
		got := keeper.HalvingDifficulty(tt.epoch)
		if got != tt.expected {
			t.Errorf("HalvingDifficulty(%d) = %d, want %d", tt.epoch, got, tt.expected)
		}
	}
}

func TestTotalRegistrationsCounter(t *testing.T) {
	k, sdkCtx, _ := setupKeeper(t)

	// Initially zero.
	if got := k.GetTotalRegistrations(sdkCtx); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}

	// Set and get.
	k.SetTotalRegistrations(sdkCtx, 42)
	if got := k.GetTotalRegistrations(sdkCtx); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	// Increment.
	newCount := k.IncrementTotalRegistrations(sdkCtx)
	if newCount != 43 {
		t.Fatalf("expected 43, got %d", newCount)
	}
	if got := k.GetTotalRegistrations(sdkCtx); got != 43 {
		t.Fatalf("expected 43 after increment, got %d", got)
	}
}

func TestSelfRegisterIncrementsCounter(t *testing.T) {
	k, sdkCtx, _ := setupKeeper(t)
	ctx := sdkCtx

	// Register two users, verify counter increments.
	for i, name := range []string{"userAAAAAAAAAAAAAAAAAAAA", "userBBBBBBBBBBBBBBBBBBBB"} {
		user := sdk.AccAddress([]byte(name)).String()
		nonce := findValidNonce(user, 16)
		_, err := k.SelfRegister(ctx, types.MsgSelfRegister{Creator: user, Nonce: nonce})
		if err != nil {
			t.Fatalf("SelfRegister user %d failed: %v", i, err)
		}
		expected := uint64(i + 1)
		if got := k.GetTotalRegistrations(sdkCtx); got != expected {
			t.Fatalf("after user %d: expected counter %d, got %d", i, expected, got)
		}
	}
}

func TestGenesisRoundTrip(t *testing.T) {
	// --- Phase 1: Populate state in keeper A ---
	kA, sdkCtxA, _ := setupKeeper(t)
	ctxA := sdkCtxA

	// Use custom params to verify they survive the round-trip.
	customParams := types.Params{
		AirdropAmount:         sdk.NewCoin("uoas", math.NewInt(15000000)),
		PowDifficulty:         16,
		RepaymentDeadlineDays: 60,
	}
	if err := kA.SetParams(sdkCtxA, customParams); err != nil {
		t.Fatalf("SetParams failed: %v", err)
	}

	// Register 2 users.
	userA := sdk.AccAddress([]byte("genesisUserAAAAAAAAA")).String()
	userB := sdk.AccAddress([]byte("genesisUserBBBBBBBBB")).String()

	nonceA := findValidNonce(userA, 16)
	if _, err := kA.SelfRegister(ctxA, types.MsgSelfRegister{Creator: userA, Nonce: nonceA}); err != nil {
		t.Fatalf("SelfRegister userA failed: %v", err)
	}
	nonceB := findValidNonce(userB, 16)
	if _, err := kA.SelfRegister(ctxA, types.MsgSelfRegister{Creator: userB, Nonce: nonceB}); err != nil {
		t.Fatalf("SelfRegister userB failed: %v", err)
	}

	// Snapshot state from keeper A.
	paramsA := kA.GetParams(sdkCtxA)
	totalRegsA := kA.GetTotalRegistrations(sdkCtxA)

	var registrationsA []types.Registration
	kA.IterateAllRegistrations(sdkCtxA, func(reg types.Registration) bool {
		registrationsA = append(registrationsA, reg)
		return false
	})

	if len(registrationsA) != 2 {
		t.Fatalf("expected 2 registrations, got %d", len(registrationsA))
	}
	if totalRegsA != 2 {
		t.Fatalf("expected total_registrations=2, got %d", totalRegsA)
	}

	// --- Phase 2: Build genesis state (simulates ExportGenesis) ---
	gs := types.GenesisState{
		Registrations: registrationsA,
		Params:        paramsA,
	}

	// --- Phase 3: Import into fresh keeper B (simulates InitGenesis) ---
	kB, sdkCtxB, _ := setupKeeper(t)

	// Apply genesis: set params, set registrations, derive counter.
	if err := kB.SetParams(sdkCtxB, gs.Params); err != nil {
		t.Fatalf("SetParams on kB failed: %v", err)
	}
	for _, reg := range gs.Registrations {
		if err := kB.SetRegistration(sdkCtxB, reg); err != nil {
			t.Fatalf("SetRegistration on kB failed: %v", err)
		}
	}
	kB.SetTotalRegistrations(sdkCtxB, uint64(len(gs.Registrations)))

	// --- Phase 4: Verify everything matches ---
	paramsB := kB.GetParams(sdkCtxB)
	if paramsB.AirdropAmount.Denom != paramsA.AirdropAmount.Denom ||
		!paramsB.AirdropAmount.Amount.Equal(paramsA.AirdropAmount.Amount) {
		t.Fatalf("params airdrop mismatch: A=%s, B=%s", paramsA.AirdropAmount, paramsB.AirdropAmount)
	}
	if paramsB.PowDifficulty != paramsA.PowDifficulty {
		t.Fatalf("params pow_difficulty mismatch: A=%d, B=%d", paramsA.PowDifficulty, paramsB.PowDifficulty)
	}
	if paramsB.RepaymentDeadlineDays != paramsA.RepaymentDeadlineDays {
		t.Fatalf("params repayment_deadline_days mismatch: A=%d, B=%d", paramsA.RepaymentDeadlineDays, paramsB.RepaymentDeadlineDays)
	}

	// Verify registrations.
	for _, origReg := range registrationsA {
		importedReg, found := kB.GetRegistration(sdkCtxB, origReg.Address)
		if !found {
			t.Fatalf("registration for %s not found in kB", origReg.Address)
		}
		if !importedReg.AirdropAmount.Equal(origReg.AirdropAmount) {
			t.Fatalf("airdrop mismatch for %s: A=%s, B=%s", origReg.Address, origReg.AirdropAmount, importedReg.AirdropAmount)
		}
		if importedReg.Status != origReg.Status {
			t.Fatalf("status mismatch for %s: A=%v, B=%v", origReg.Address, origReg.Status, importedReg.Status)
		}
		if importedReg.PowNonce != origReg.PowNonce {
			t.Fatalf("nonce mismatch for %s: A=%d, B=%d", origReg.Address, origReg.PowNonce, importedReg.PowNonce)
		}
	}

	// Verify total_registrations counter.
	totalRegsB := kB.GetTotalRegistrations(sdkCtxB)
	if totalRegsB != totalRegsA {
		t.Fatalf("total_registrations mismatch: A=%d, B=%d", totalRegsA, totalRegsB)
	}

	// Verify count from iteration also matches.
	var countB int
	kB.IterateAllRegistrations(sdkCtxB, func(_ types.Registration) bool {
		countB++
		return false
	})
	if countB != 2 {
		t.Fatalf("expected 2 registrations in kB via iteration, got %d", countB)
	}
}

func TestLeadingZeroBits(t *testing.T) {
	tests := []struct {
		input    []byte
		expected int
	}{
		{[]byte{0x00, 0x00, 0x00, 0xFF}, 24},
		{[]byte{0x00, 0x01}, 15},
		{[]byte{0x80}, 0},
		{[]byte{0x40}, 1},
		{[]byte{0x00}, 8},
		{[]byte{}, 0},
	}

	for _, tt := range tests {
		got := keeper.LeadingZeroBits(tt.input)
		if got != tt.expected {
			t.Errorf("LeadingZeroBits(%x) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
