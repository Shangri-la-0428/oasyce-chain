package keeper_test

import (
	"context"
	"crypto/sha256"
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

	reputationtypes "github.com/oasyce/chain/x/reputation/types"
	"github.com/oasyce/chain/x/work/keeper"
	"github.com/oasyce/chain/x/work/types"
)

// ---- Mock Bank Keeper ----

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

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	from := fromAddr.String()
	to := toAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return types.ErrInvalidBounty.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	from := senderAddr.String()
	if !m.balances[from].IsAllGTE(amt) {
		return types.ErrInvalidBounty.Wrap("mock: insufficient funds")
	}
	m.balances[from] = m.balances[from].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return types.ErrInvalidBounty.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	to := recipientAddr.String()
	m.balances[to] = m.balances[to].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	if !m.moduleBalances[senderModule].IsAllGTE(amt) {
		return types.ErrInvalidBounty.Wrap("mock: insufficient module funds")
	}
	m.moduleBalances[senderModule] = m.moduleBalances[senderModule].Sub(amt...)
	m.moduleBalances[recipientModule] = m.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	if !m.moduleBalances[moduleName].IsAllGTE(amt) {
		return types.ErrInvalidBounty.Wrap("mock: insufficient module funds for burn")
	}
	m.moduleBalances[moduleName] = m.moduleBalances[moduleName].Sub(amt...)
	return nil
}

// ---- Mock Reputation Keeper ----

type mockReputationKeeper struct {
	scores map[string]uint64
}

func newMockReputationKeeper() *mockReputationKeeper {
	return &mockReputationKeeper{scores: make(map[string]uint64)}
}

func (m *mockReputationKeeper) setScore(addr string, score uint64) {
	m.scores[addr] = score
}

func (m *mockReputationKeeper) GetReputation(_ sdk.Context, address string) (reputationtypes.ReputationScore, bool) {
	score, found := m.scores[address]
	if !found {
		return reputationtypes.ReputationScore{}, false
	}
	return reputationtypes.ReputationScore{
		Address:    address,
		TotalScore: score,
	}, true
}

// ---- Test Setup ----

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockReputationKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NoOpMetrics{})
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(cms, cmtproto.Header{
		Time:   time.Now(),
		Height: 100,
	}, false, logger)

	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	bankKeeper := newMockBankKeeper()
	repKeeper := newMockReputationKeeper()

	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, repKeeper, "authority")

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatal(err)
	}

	return k, ctx, bankKeeper, repKeeper
}

func testAddresses(n int) []string {
	names := []string{
		"creator_____________",
		"executor1___________",
		"executor2___________",
		"executor3___________",
		"executor4___________",
		"challenger__________",
	}
	addrs := make([]string, n)
	for i := 0; i < n && i < len(names); i++ {
		addrs[i] = sdk.AccAddress([]byte(names[i])).String()
	}
	return addrs
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestExecutorRegistration(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	addrs := testAddresses(2)
	executor := addrs[1]

	// Register
	profile := types.ExecutorProfile{
		Address:            executor,
		SupportedTaskTypes: []string{"inference", "embedding"},
		MaxComputeUnits:    1000,
		Active:             true,
	}
	if err := k.SetExecutorProfile(ctx, profile); err != nil {
		t.Fatalf("SetExecutorProfile failed: %v", err)
	}

	// Retrieve
	got, found := k.GetExecutorProfile(ctx, executor)
	if !found {
		t.Fatal("executor profile not found")
	}
	if got.Address != executor {
		t.Errorf("address mismatch: got %s, want %s", got.Address, executor)
	}
	if len(got.SupportedTaskTypes) != 2 {
		t.Errorf("task types count: got %d, want 2", len(got.SupportedTaskTypes))
	}
	if !got.Active {
		t.Error("executor should be active")
	}
}

func TestTaskCRUD(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	addrs := testAddresses(1)
	creator := addrs[0]

	task := types.Task{
		Id:              1,
		Creator:         creator,
		TaskType:        "inference",
		InputHash:       sha256Hash([]byte("test input")),
		InputUri:        "ipfs://Qm123",
		MaxComputeUnits: 100,
		Bounty:          sdk.NewCoin("uoas", math.NewInt(10000)),
		Deposit:         sdk.NewCoin("uoas", math.NewInt(1000)),
		Redundancy:      3,
		TimeoutHeight:   200,
		Status:          types.TASK_STATUS_SUBMITTED,
		SubmitHeight:    100,
	}

	if err := k.SetTask(ctx, task); err != nil {
		t.Fatalf("SetTask failed: %v", err)
	}

	got, found := k.GetTask(ctx, 1)
	if !found {
		t.Fatal("task not found")
	}
	if got.Creator != creator {
		t.Errorf("creator mismatch: got %s, want %s", got.Creator, creator)
	}
	if got.Status != types.TASK_STATUS_SUBMITTED {
		t.Errorf("status mismatch: got %d, want SUBMITTED", got.Status)
	}

	// Not found
	_, found = k.GetTask(ctx, 999)
	if found {
		t.Error("should not find non-existent task")
	}
}

func TestCommitRevealFlow(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	addrs := testAddresses(2)
	executor := addrs[1]

	taskID := uint64(1)
	outputHash := sha256Hash([]byte("result data"))
	salt := []byte("random_salt_123")

	// Commit
	commitHash := keeper.ComputeCommitHash(outputHash, salt, executor, false)
	commitment := types.Commitment{
		Executor:     executor,
		TaskId:       taskID,
		CommitHash:   commitHash,
		CommitHeight: 101,
	}
	if err := k.SetCommitment(ctx, commitment); err != nil {
		t.Fatalf("SetCommitment failed: %v", err)
	}

	// Verify commitment exists
	got, found := k.GetCommitment(ctx, taskID, executor)
	if !found {
		t.Fatal("commitment not found")
	}

	// Verify reveal matches
	if !keeper.VerifyCommitment(got, outputHash, salt, executor, false) {
		t.Error("VerifyCommitment should return true for correct reveal")
	}

	// Wrong output should fail
	wrongHash := sha256Hash([]byte("wrong data"))
	if keeper.VerifyCommitment(got, wrongHash, salt, executor, false) {
		t.Error("VerifyCommitment should return false for wrong output")
	}

	// Wrong salt should fail
	if keeper.VerifyCommitment(got, outputHash, []byte("wrong_salt"), executor, false) {
		t.Error("VerifyCommitment should return false for wrong salt")
	}
}

func TestParamsRoundtrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.DefaultRedundancy != 3 {
		t.Errorf("default_redundancy: got %d, want 3", params.DefaultRedundancy)
	}
	if !params.ExecutorShare.Equal(math.LegacyNewDecWithPrec(90, 2)) {
		t.Errorf("executor_share: got %s, want 0.90", params.ExecutorShare)
	}

	// Verify shares sum to 1
	sum := params.ExecutorShare.Add(params.ProtocolShare).Add(params.BurnShare).Add(params.SubmitterRebate)
	if !sum.Equal(math.LegacyOneDec()) {
		t.Errorf("shares sum: got %s, want 1.0", sum)
	}
}

func TestGenesisValidation(t *testing.T) {
	// Valid genesis
	gs := *types.DefaultGenesisState()
	if err := types.ValidateGenesis(gs); err != nil {
		t.Fatalf("default genesis should be valid: %v", err)
	}

	// Invalid: shares don't sum to 1
	bad := gs
	bad.Params.ExecutorShare = math.LegacyNewDecWithPrec(50, 2) // 0.50 instead of 0.90
	if err := types.ValidateGenesis(bad); err == nil {
		t.Error("should reject genesis where shares != 1.0")
	}

	// Invalid: zero redundancy
	bad2 := gs
	bad2.Params.DefaultRedundancy = 0
	if err := types.ValidateGenesis(bad2); err == nil {
		t.Error("should reject genesis with zero redundancy")
	}
}

func TestTaskCounter(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Counter starts at 0
	if got := k.GetTaskCounter(ctx); got != 0 {
		t.Errorf("initial counter: got %d, want 0", got)
	}

	// Set and retrieve
	k.SetTaskCounter(ctx, 42)
	if got := k.GetTaskCounter(ctx); got != 42 {
		t.Errorf("counter after set: got %d, want 42", got)
	}
}

func TestMsgValidation(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("valid_address_______")).String()

	// Valid MsgSubmitTask
	msg := &types.MsgSubmitTask{
		Creator:         validAddr,
		TaskType:        "inference",
		InputHash:       sha256Hash([]byte("input")),
		Bounty:          sdk.NewCoin("uoas", math.NewInt(1000)),
		MaxComputeUnits: 100,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("valid MsgSubmitTask rejected: %v", err)
	}

	// Invalid: empty task type
	bad := *msg
	bad.TaskType = ""
	if err := bad.ValidateBasic(); err == nil {
		t.Error("should reject empty task_type")
	}

	// Invalid: wrong hash length
	bad2 := *msg
	bad2.InputHash = []byte("too short")
	if err := bad2.ValidateBasic(); err == nil {
		t.Error("should reject non-32-byte input_hash")
	}

	// Valid MsgRegisterExecutor
	regMsg := &types.MsgRegisterExecutor{
		Executor:           validAddr,
		SupportedTaskTypes: []string{"inference"},
		MaxComputeUnits:    1000,
	}
	if err := regMsg.ValidateBasic(); err != nil {
		t.Errorf("valid MsgRegisterExecutor rejected: %v", err)
	}

	// Invalid: no task types
	badReg := *regMsg
	badReg.SupportedTaskTypes = nil
	if err := badReg.ValidateBasic(); err == nil {
		t.Error("should reject executor with no task types")
	}
}

func TestEpochStats(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Initially empty
	_, found := k.GetEpochStats(ctx, 0)
	if found {
		t.Error("epoch stats should not exist initially")
	}

	// Set and retrieve
	stats := types.EpochStats{
		Epoch:          0,
		TasksSubmitted: 10,
		TasksSettled:   8,
		TasksExpired:   2,
		TotalBounties:  math.NewInt(100000),
		TotalBurned:    math.NewInt(2000),
	}
	if err := k.SetEpochStats(ctx, stats); err != nil {
		t.Fatalf("SetEpochStats failed: %v", err)
	}

	got, found := k.GetEpochStats(ctx, 0)
	if !found {
		t.Fatal("epoch stats not found")
	}
	if got.TasksSettled != 8 {
		t.Errorf("tasks_settled: got %d, want 8", got.TasksSettled)
	}
	if !got.TotalBurned.Equal(math.NewInt(2000)) {
		t.Errorf("total_burned: got %s, want 2000", got.TotalBurned)
	}
}

func TestExecutorAssignment(t *testing.T) {
	k, ctx, _, repKeeper := setupKeeper(t)
	addrs := testAddresses(5)
	creator := addrs[0]

	// Register 3 executors
	for i := 1; i <= 3; i++ {
		profile := types.ExecutorProfile{
			Address:            addrs[i],
			SupportedTaskTypes: []string{"inference"},
			MaxComputeUnits:    1000,
			Active:             true,
		}
		k.SetExecutorProfile(ctx, profile)
		repKeeper.setScore(addrs[i], 100)
	}

	task := types.Task{
		Id:              1,
		Creator:         creator,
		TaskType:        "inference",
		MaxComputeUnits: 100,
		Redundancy:      3,
	}

	// Need to set header hash for deterministic assignment
	ctx = ctx.WithHeaderHash(sha256Hash([]byte("block_hash")))

	executors, err := k.AssignExecutors(ctx, task)
	if err != nil {
		t.Fatalf("AssignExecutors failed: %v", err)
	}
	if len(executors) != 3 {
		t.Errorf("assigned %d executors, want 3", len(executors))
	}

	// Creator should not be assigned
	for _, e := range executors {
		if e == creator {
			t.Error("creator should not be assigned as executor")
		}
	}

	// Deterministic: same inputs should give same output
	executors2, _ := k.AssignExecutors(ctx, task)
	for i := range executors {
		if executors[i] != executors2[i] {
			t.Error("assignment should be deterministic")
		}
	}
}

func TestAssignmentFailsWithoutEnoughExecutors(t *testing.T) {
	k, ctx, _, repKeeper := setupKeeper(t)
	addrs := testAddresses(2)

	// Only 1 executor but need 3
	profile := types.ExecutorProfile{
		Address:            addrs[1],
		SupportedTaskTypes: []string{"inference"},
		MaxComputeUnits:    1000,
		Active:             true,
	}
	k.SetExecutorProfile(ctx, profile)
	repKeeper.setScore(addrs[1], 100)

	task := types.Task{
		Id:              1,
		Creator:         addrs[0],
		TaskType:        "inference",
		MaxComputeUnits: 100,
		Redundancy:      3,
	}

	ctx = ctx.WithHeaderHash(sha256Hash([]byte("block_hash")))
	_, err := k.AssignExecutors(ctx, task)
	if err == nil {
		t.Error("should fail when not enough executors")
	}
}

func TestSettlement(t *testing.T) {
	k, ctx, bank, _ := setupKeeper(t)
	addrs := testAddresses(5)
	creator := addrs[0]
	exec1, exec2, exec3 := addrs[1], addrs[2], addrs[3]

	bounty := sdk.NewCoin("uoas", math.NewInt(100000))

	// Fund module account (simulating escrow)
	deposit := sdk.NewCoin("uoas", math.NewInt(10000))
	bank.moduleBalances[types.ModuleName] = sdk.NewCoins(bounty.Add(deposit))

	task := types.Task{
		Id:                1,
		Creator:           creator,
		TaskType:          "inference",
		Bounty:            bounty,
		Deposit:           deposit,
		Redundancy:        3,
		Status:            types.TASK_STATUS_REVEALING,
		AssignedExecutors: []string{exec1, exec2, exec3},
	}
	k.SetTask(ctx, task)

	// All 3 executors submit same output_hash (consensus)
	sameHash := sha256Hash([]byte("correct output"))
	for _, exec := range []string{exec1, exec2, exec3} {
		// Register executor profile
		k.SetExecutorProfile(ctx, types.ExecutorProfile{
			Address: exec,
			Active:  true,
		})
		// Submit result
		k.SetResult(ctx, types.Result{
			Executor:         exec,
			TaskId:           1,
			OutputHash:       sameHash,
			ComputeUnitsUsed: 100,
		})
	}

	err := k.SettleTask(ctx, task)
	if err != nil {
		t.Fatalf("SettleTask failed: %v", err)
	}

	// Verify task is settled
	settled, _ := k.GetTask(ctx, 1)
	if settled.Status != types.TASK_STATUS_SETTLED {
		t.Errorf("task status: got %d, want SETTLED", settled.Status)
	}

	// Verify executor profiles updated
	for _, exec := range []string{exec1, exec2, exec3} {
		p, _ := k.GetExecutorProfile(ctx, exec)
		if p.TasksCompleted != 1 {
			t.Errorf("executor %s tasks_completed: got %d, want 1", exec, p.TasksCompleted)
		}
	}

	// Verify creator got deposit + rebate back
	creatorBalance := bank.balances[creator]
	if creatorBalance.IsZero() {
		t.Error("creator should have received deposit + rebate")
	}
}

func TestSettlementWithMinority(t *testing.T) {
	k, ctx, bank, _ := setupKeeper(t)
	addrs := testAddresses(5)
	exec1, exec2, exec3 := addrs[1], addrs[2], addrs[3]

	bounty := sdk.NewCoin("uoas", math.NewInt(100000))
	deposit := sdk.NewCoin("uoas", math.NewInt(10000))
	bank.moduleBalances[types.ModuleName] = sdk.NewCoins(bounty.Add(deposit))

	task := types.Task{
		Id:                1,
		Creator:           addrs[0],
		Bounty:            bounty,
		Deposit:           deposit,
		Redundancy:        3,
		Status:            types.TASK_STATUS_REVEALING,
		AssignedExecutors: []string{exec1, exec2, exec3},
	}
	k.SetTask(ctx, task)

	majorityHash := sha256Hash([]byte("correct"))
	minorityHash := sha256Hash([]byte("wrong"))

	// exec1 and exec2 agree, exec3 disagrees
	for _, exec := range []string{exec1, exec2, exec3} {
		k.SetExecutorProfile(ctx, types.ExecutorProfile{Address: exec, Active: true})
	}
	k.SetResult(ctx, types.Result{Executor: exec1, TaskId: 1, OutputHash: majorityHash, ComputeUnitsUsed: 100})
	k.SetResult(ctx, types.Result{Executor: exec2, TaskId: 1, OutputHash: majorityHash, ComputeUnitsUsed: 100})
	k.SetResult(ctx, types.Result{Executor: exec3, TaskId: 1, OutputHash: minorityHash, ComputeUnitsUsed: 100})

	err := k.SettleTask(ctx, task)
	if err != nil {
		t.Fatalf("SettleTask failed: %v", err)
	}

	// Majority executors should have completed, minority should have failed
	p1, _ := k.GetExecutorProfile(ctx, exec1)
	if p1.TasksCompleted != 1 {
		t.Errorf("exec1 tasks_completed: got %d, want 1", p1.TasksCompleted)
	}

	p3, _ := k.GetExecutorProfile(ctx, exec3)
	if p3.TasksFailed != 2 {
		t.Errorf("exec3 tasks_failed: got %d, want 2 (2x penalty for wrong result)", p3.TasksFailed)
	}
}

func TestTerminalStatus(t *testing.T) {
	tests := []struct {
		status   types.TaskStatus
		terminal bool
	}{
		{types.TASK_STATUS_SUBMITTED, false},
		{types.TASK_STATUS_ASSIGNED, false},
		{types.TASK_STATUS_COMMITTED, false},
		{types.TASK_STATUS_REVEALING, false},
		{types.TASK_STATUS_SETTLED, true},
		{types.TASK_STATUS_EXPIRED, true},
		{types.TASK_STATUS_DISPUTED, true},
	}

	for _, tt := range tests {
		if got := types.IsTerminalStatus(tt.status); got != tt.terminal {
			t.Errorf("IsTerminalStatus(%s): got %v, want %v", tt.status, got, tt.terminal)
		}
	}
}
