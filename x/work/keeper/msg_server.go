package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RegisterExecutor registers a new compute executor.
func (m msgServer) RegisterExecutor(goCtx context.Context, msg *types.MsgRegisterExecutor) (*types.MsgRegisterExecutorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, found := m.GetExecutorProfile(ctx, msg.Executor); found {
		return nil, types.ErrExecutorExists.Wrapf("executor %s already registered", msg.Executor)
	}

	profile := types.ExecutorProfile{
		Address:            msg.Executor,
		SupportedTaskTypes: msg.SupportedTaskTypes,
		MaxComputeUnits:    msg.MaxComputeUnits,
		Active:             true,
	}

	if err := m.SetExecutorProfile(ctx, profile); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"executor_registered",
		sdk.NewAttribute("executor", msg.Executor),
	))

	return &types.MsgRegisterExecutorResponse{}, nil
}

// UpdateExecutor updates an executor's profile.
func (m msgServer) UpdateExecutor(goCtx context.Context, msg *types.MsgUpdateExecutor) (*types.MsgUpdateExecutorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := m.GetExecutorProfile(ctx, msg.Executor)
	if !found {
		return nil, types.ErrExecutorNotFound.Wrapf("executor %s not registered", msg.Executor)
	}

	if len(msg.SupportedTaskTypes) > 0 {
		profile.SupportedTaskTypes = msg.SupportedTaskTypes
	}
	if msg.MaxComputeUnits > 0 {
		profile.MaxComputeUnits = msg.MaxComputeUnits
	}
	profile.Active = msg.Active

	if err := m.SetExecutorProfile(ctx, profile); err != nil {
		return nil, err
	}

	return &types.MsgUpdateExecutorResponse{}, nil
}

// SubmitTask creates a new compute task and escrows bounty + deposit.
func (m msgServer) SubmitTask(goCtx context.Context, msg *types.MsgSubmitTask) (*types.MsgSubmitTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)

	// Validate bounty minimum
	if msg.Bounty.Amount.LT(params.MinBounty) {
		return nil, types.ErrInvalidBounty.Wrapf("bounty %s below minimum %s", msg.Bounty.Amount, params.MinBounty)
	}

	// Apply defaults
	redundancy := msg.Redundancy
	if redundancy == 0 {
		redundancy = params.DefaultRedundancy
	}
	timeoutBlocks := msg.TimeoutBlocks
	if timeoutBlocks == 0 {
		timeoutBlocks = params.MinTimeoutBlocks
	}
	if timeoutBlocks < params.MinTimeoutBlocks || timeoutBlocks > params.MaxTimeoutBlocks {
		return nil, types.ErrInvalidTimeout.Wrapf("timeout %d not in [%d, %d]",
			timeoutBlocks, params.MinTimeoutBlocks, params.MaxTimeoutBlocks)
	}

	// Calculate deposit
	deposit := sdk.NewCoin(msg.Bounty.Denom,
		params.DepositRate.MulInt(msg.Bounty.Amount).TruncateInt())

	// Transfer bounty + deposit to module account
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}

	totalEscrow := sdk.NewCoins(msg.Bounty.Add(deposit))
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, totalEscrow); err != nil {
		return nil, err
	}

	// Create task
	taskID := m.nextTaskID(ctx)
	task := types.Task{
		Id:              taskID,
		Creator:         msg.Creator,
		TaskType:        msg.TaskType,
		InputHash:       msg.InputHash,
		InputUri:        msg.InputUri,
		MaxComputeUnits: msg.MaxComputeUnits,
		Bounty:          msg.Bounty,
		Deposit:         deposit,
		Redundancy:      redundancy,
		TimeoutHeight:   timeoutBlocks, // relative — converted to absolute during assignment
		Status:          types.TASK_STATUS_SUBMITTED,
		SubmitHeight:    uint64(ctx.BlockHeight()),
	}

	if err := m.setTaskWithIndexes(ctx, task, types.TASK_STATUS_UNSPECIFIED); err != nil {
		return nil, err
	}

	// Update epoch stats
	height := uint64(ctx.BlockHeight())
	epoch := height / 1000
	stats, found := m.GetEpochStats(ctx, epoch)
	if !found {
		stats = types.EpochStats{
			Epoch:         epoch,
			TotalBounties: math.ZeroInt(),
			TotalBurned:   math.ZeroInt(),
		}
	}
	stats.TasksSubmitted++
	_ = m.SetEpochStats(ctx, stats)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"task_submitted",
		sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskID)),
		sdk.NewAttribute("creator", msg.Creator),
		sdk.NewAttribute("task_type", msg.TaskType),
		sdk.NewAttribute("bounty", msg.Bounty.String()),
	))

	return &types.MsgSubmitTaskResponse{TaskId: taskID}, nil
}

// CommitResult submits a sealed result hash (commit phase).
func (m msgServer) CommitResult(goCtx context.Context, msg *types.MsgCommitResult) (*types.MsgCommitResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound.Wrapf("task %d", msg.TaskId)
	}

	if task.Status != types.TASK_STATUS_ASSIGNED && task.Status != types.TASK_STATUS_COMMITTED {
		return nil, types.ErrInvalidStatus.Wrapf("task status %s, expected ASSIGNED or COMMITTED", task.Status)
	}

	// Verify executor is assigned
	if !isAssigned(task.AssignedExecutors, msg.Executor) {
		return nil, types.ErrUnauthorized.Wrapf("%s is not assigned to task %d", msg.Executor, msg.TaskId)
	}

	// Prevent double commit
	if _, exists := m.GetCommitment(ctx, msg.TaskId, msg.Executor); exists {
		return nil, types.ErrCommitmentExists.Wrapf("executor %s already committed for task %d", msg.Executor, msg.TaskId)
	}

	commitment := types.Commitment{
		Executor:     msg.Executor,
		TaskId:       msg.TaskId,
		CommitHash:   msg.CommitHash,
		CommitHeight: uint64(ctx.BlockHeight()),
	}

	if err := m.SetCommitment(ctx, commitment); err != nil {
		return nil, err
	}

	// Check if all executors have committed
	commitCount := m.CountCommitments(ctx, msg.TaskId)
	if commitCount >= int(task.Redundancy) {
		// All committed — transition to REVEALING
		if task.Status != types.TASK_STATUS_COMMITTED {
			oldStatus := task.Status
			task.Status = types.TASK_STATUS_COMMITTED
			_ = m.setTaskWithIndexes(ctx, task, oldStatus)
		}
		if err := m.TransitionToReveal(ctx, task); err != nil {
			return nil, err
		}
	} else if task.Status == types.TASK_STATUS_ASSIGNED {
		// First commit — mark as COMMITTED
		oldStatus := task.Status
		task.Status = types.TASK_STATUS_COMMITTED
		_ = m.setTaskWithIndexes(ctx, task, oldStatus)
	}

	return &types.MsgCommitResultResponse{}, nil
}

// RevealResult reveals the actual result, verified against prior commitment.
func (m msgServer) RevealResult(goCtx context.Context, msg *types.MsgRevealResult) (*types.MsgRevealResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound.Wrapf("task %d", msg.TaskId)
	}

	if task.Status != types.TASK_STATUS_REVEALING {
		return nil, types.ErrInvalidStatus.Wrapf("task status %s, expected REVEALING", task.Status)
	}

	// Verify executor is assigned
	if !isAssigned(task.AssignedExecutors, msg.Executor) {
		return nil, types.ErrUnauthorized.Wrapf("%s is not assigned to task %d", msg.Executor, msg.TaskId)
	}

	// Prevent double reveal
	if _, exists := m.GetResult(ctx, msg.TaskId, msg.Executor); exists {
		return nil, types.ErrResultExists.Wrapf("executor %s already revealed for task %d", msg.Executor, msg.TaskId)
	}

	// Verify commitment
	commitment, found := m.GetCommitment(ctx, msg.TaskId, msg.Executor)
	if !found {
		return nil, types.ErrCommitmentNotFound.Wrapf("executor %s has no commitment for task %d", msg.Executor, msg.TaskId)
	}

	if !VerifyCommitment(commitment, msg.OutputHash, msg.Salt, msg.Executor, msg.Unavailable) {
		return nil, types.ErrRevealMismatch.Wrapf("reveal does not match commitment for executor %s", msg.Executor)
	}

	result := types.Result{
		Executor:         msg.Executor,
		TaskId:           msg.TaskId,
		OutputHash:       msg.OutputHash,
		OutputUri:        msg.OutputUri,
		ComputeUnitsUsed: msg.ComputeUnitsUsed,
		Salt:             msg.Salt,
		Unavailable:      msg.Unavailable,
	}

	if err := m.SetResult(ctx, result); err != nil {
		return nil, err
	}

	// Check if all executors have revealed
	revealCount := m.CountResults(ctx, msg.TaskId)
	if revealCount >= int(task.Redundancy) {
		// All revealed — settle immediately
		if err := m.SettleTask(ctx, task); err != nil {
			return nil, err
		}
	}

	return &types.MsgRevealResultResponse{}, nil
}

// DisputeResult challenges a settled task.
func (m msgServer) DisputeResult(goCtx context.Context, msg *types.MsgDisputeResult) (*types.MsgDisputeResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound.Wrapf("task %d", msg.TaskId)
	}

	if task.Status != types.TASK_STATUS_SETTLED {
		return nil, types.ErrInvalidStatus.Wrapf("can only dispute SETTLED tasks, got %s", task.Status)
	}

	// Validate bond minimum
	params := m.GetParams(ctx)
	minBond := params.DisputeBondRate.MulInt(task.Bounty.Amount).TruncateInt()
	if msg.Bond.Amount.LT(minBond) {
		return nil, types.ErrDisputeBondTooLow.Wrapf("bond %s below minimum %s%s",
			msg.Bond.Amount, minBond, task.Bounty.Denom)
	}

	// Escrow dispute bond
	challengerAddr, err := sdk.AccAddressFromBech32(msg.Challenger)
	if err != nil {
		return nil, types.ErrInvalidAddress.Wrapf("invalid challenger: %s", err)
	}

	bondCoins := sdk.NewCoins(msg.Bond)
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, challengerAddr, types.ModuleName, bondCoins); err != nil {
		return nil, err
	}

	// Mark as disputed
	oldStatus := task.Status
	task.Status = types.TASK_STATUS_DISPUTED
	if err := m.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"task_disputed",
		sdk.NewAttribute("task_id", fmt.Sprintf("%d", msg.TaskId)),
		sdk.NewAttribute("challenger", msg.Challenger),
		sdk.NewAttribute("reason", msg.Reason),
		sdk.NewAttribute("bond", msg.Bond.String()),
	))

	return &types.MsgDisputeResultResponse{}, nil
}

// ---- helpers ----

func isAssigned(executors []string, addr string) bool {
	for _, e := range executors {
		if e == addr {
			return true
		}
	}
	return false
}

// ensure math import is used
var _ = math.ZeroInt
