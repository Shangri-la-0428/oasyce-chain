package keeper

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

// SettleTask processes results for a completed task, distributing rewards.
// Determines majority output_hash (>= 2/3 agreement).
func (k Keeper) SettleTask(ctx sdk.Context, task types.Task) error {
	params := k.GetParams(ctx)
	results := k.GetAllResults(ctx, task.Id)

	// Count unavailable reports
	unavailableCount := 0
	for _, r := range results {
		if r.Unavailable {
			unavailableCount++
		}
	}

	// If >= 2/3 report unavailable, expire the task and penalize submitter deposit
	threshold := (len(task.AssignedExecutors)*2 + 2) / 3
	if unavailableCount >= threshold {
		return k.expireTaskUnavailable(ctx, task)
	}

	// Find majority output_hash among available results
	hashCounts := make(map[string][]types.Result)
	for _, r := range results {
		if r.Unavailable {
			continue
		}
		key := string(r.OutputHash)
		hashCounts[key] = append(hashCounts[key], r)
	}

	var majorityHash string
	var majorityResults []types.Result
	for hash, res := range hashCounts {
		if len(res) > len(majorityResults) {
			majorityHash = hash
			majorityResults = res
		}
	}

	availableCount := len(results) - unavailableCount
	majorityThreshold := (availableCount*2 + 2) / 3

	if len(majorityResults) < majorityThreshold {
		return k.expireTaskNoConsensus(ctx, task)
	}

	// Distribute rewards to majority executors
	bountyAmt := task.Bounty.Amount
	denom := task.Bounty.Denom

	executorTotal := params.ExecutorShare.MulInt(bountyAmt).TruncateInt()
	protocolTotal := params.ProtocolShare.MulInt(bountyAmt).TruncateInt()
	burnTotal := params.BurnShare.MulInt(bountyAmt).TruncateInt()
	rebateTotal := params.SubmitterRebate.MulInt(bountyAmt).TruncateInt()

	// Pay each majority executor their share (equal split)
	perExecutor := executorTotal.Quo(math.NewInt(int64(len(majorityResults))))
	for _, r := range majorityResults {
		execAddr, err := sdk.AccAddressFromBech32(r.Executor)
		if err != nil {
			continue
		}
		coins := sdk.NewCoins(sdk.NewCoin(denom, perExecutor))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, execAddr, coins); err != nil {
			return err
		}
		k.incrementExecutorCompleted(ctx, r.Executor)
	}

	// Protocol fee -> fee_collector
	if protocolTotal.IsPositive() {
		protocolCoins := sdk.NewCoins(sdk.NewCoin(denom, protocolTotal))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "fee_collector", protocolCoins); err != nil {
			return err
		}
	}

	// Burn
	if burnTotal.IsPositive() {
		burnCoins := sdk.NewCoins(sdk.NewCoin(denom, burnTotal))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			return err
		}
	}

	// Rebate to submitter
	if rebateTotal.IsPositive() {
		creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
		if err == nil {
			rebateCoins := sdk.NewCoins(sdk.NewCoin(denom, rebateTotal))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, rebateCoins)
		}
	}

	// Return deposit to submitter
	if task.Deposit.IsPositive() {
		creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
		if err == nil {
			depositCoins := sdk.NewCoins(task.Deposit)
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, depositCoins)
		}
	}

	// Penalize minority executors
	for hash, res := range hashCounts {
		if hash == majorityHash {
			continue
		}
		for _, r := range res {
			k.incrementExecutorFailed(ctx, r.Executor)
		}
	}

	// Update task status
	oldStatus := task.Status
	task.Status = types.TASK_STATUS_SETTLED
	if err := k.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
		return err
	}

	k.recordSettlement(ctx, task.Bounty.Amount, burnTotal)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"task_settled",
		sdk.NewAttribute("task_id", fmt.Sprintf("%d", task.Id)),
		sdk.NewAttribute("majority_executors", fmt.Sprintf("%d", len(majorityResults))),
		sdk.NewAttribute("burned", burnTotal.String()),
	))

	return nil
}

func (k Keeper) expireTaskUnavailable(ctx sdk.Context, task types.Task) error {
	creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
	if err != nil {
		return err
	}

	// Return bounty to creator
	bountyCoins := sdk.NewCoins(task.Bounty)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, bountyCoins); err != nil {
		return err
	}

	// Deposit is forfeited (burned) — penalty for submitting unavailable data
	if task.Deposit.IsPositive() {
		depositCoins := sdk.NewCoins(task.Deposit)
		_ = k.bankKeeper.BurnCoins(ctx, types.ModuleName, depositCoins)
	}

	oldStatus := task.Status
	task.Status = types.TASK_STATUS_EXPIRED
	return k.setTaskWithIndexes(ctx, task, oldStatus)
}

func (k Keeper) expireTaskNoConsensus(ctx sdk.Context, task types.Task) error {
	creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
	if err != nil {
		return err
	}

	// Refund bounty + deposit
	refundCoins := sdk.NewCoins(task.Bounty)
	if task.Deposit.IsPositive() {
		refundCoins = refundCoins.Add(task.Deposit)
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, refundCoins); err != nil {
		return err
	}

	oldStatus := task.Status
	task.Status = types.TASK_STATUS_EXPIRED
	if err := k.setTaskWithIndexes(ctx, task, oldStatus); err != nil {
		return err
	}

	// Mark all executors with failure (lighter than minority penalty)
	results := k.GetAllResults(ctx, task.Id)
	for _, r := range results {
		k.incrementExecutorFailed(ctx, r.Executor)
	}

	return nil
}

func (k Keeper) incrementExecutorCompleted(ctx sdk.Context, addr string) {
	p, found := k.GetExecutorProfile(ctx, addr)
	if !found {
		return
	}
	p.TasksCompleted++
	_ = k.SetExecutorProfile(ctx, p)
}

func (k Keeper) incrementExecutorFailed(ctx sdk.Context, addr string) {
	p, found := k.GetExecutorProfile(ctx, addr)
	if !found {
		return
	}
	p.TasksFailed++
	_ = k.SetExecutorProfile(ctx, p)
}

func (k Keeper) recordSettlement(ctx sdk.Context, bounty, burned math.Int) {
	height := uint64(ctx.BlockHeight())
	epoch := height / 1000 // 1 epoch = 1000 blocks

	stats, found := k.GetEpochStats(ctx, epoch)
	if !found {
		stats = types.EpochStats{
			Epoch:         epoch,
			TotalBounties: math.ZeroInt(),
			TotalBurned:   math.ZeroInt(),
		}
	}
	stats.TasksSettled++
	stats.TotalBounties = stats.TotalBounties.Add(bounty)
	stats.TotalBurned = stats.TotalBurned.Add(burned)
	_ = k.SetEpochStats(ctx, stats)
}

// VerifyCommitment checks that the reveal matches the prior commitment.
// commit_hash = sha256(output_hash + salt + executor_addr + unavailable_flag)
func VerifyCommitment(commitment types.Commitment, outputHash, salt []byte, executor string, unavailable bool) bool {
	expected := ComputeCommitHash(outputHash, salt, executor, unavailable)
	return bytes.Equal(commitment.CommitHash, expected)
}

// ComputeCommitHash generates the commit hash for the commit-reveal scheme.
func ComputeCommitHash(outputHash, salt []byte, executor string, unavailable bool) []byte {
	h := sha256.New()
	h.Write(outputHash)
	h.Write(salt)
	h.Write([]byte(executor))
	if unavailable {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}
