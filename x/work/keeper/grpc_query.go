package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) Task(goCtx context.Context, req *types.QueryTaskRequest) (*types.QueryTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	task, found := q.GetTask(ctx, req.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound.Wrapf("task %d not found", req.TaskId)
	}
	return &types.QueryTaskResponse{Task: task}, nil
}

func (q queryServer) TasksByStatus(goCtx context.Context, req *types.QueryTasksByStatusRequest) (*types.QueryTasksByStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var tasks []types.Task
	status := types.TaskStatus(req.Status)
	q.IterateTasksByStatus(ctx, status, func(task types.Task) bool {
		tasks = append(tasks, task)
		return len(tasks) >= 100 // hard cap
	})

	return &types.QueryTasksByStatusResponse{Tasks: tasks}, nil
}

func (q queryServer) TasksByCreator(goCtx context.Context, req *types.QueryTasksByCreatorRequest) (*types.QueryTasksByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var tasks []types.Task
	q.IterateTasksByCreator(ctx, req.Creator, func(task types.Task) bool {
		tasks = append(tasks, task)
		return len(tasks) >= 100
	})

	return &types.QueryTasksByCreatorResponse{Tasks: tasks}, nil
}

func (q queryServer) TasksByExecutor(goCtx context.Context, req *types.QueryTasksByExecutorRequest) (*types.QueryTasksByExecutorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var tasks []types.Task
	q.IterateTasksByExecutor(ctx, req.Executor, func(task types.Task) bool {
		tasks = append(tasks, task)
		return len(tasks) >= 100
	})

	return &types.QueryTasksByExecutorResponse{Tasks: tasks}, nil
}

func (q queryServer) ExecutorProfile(goCtx context.Context, req *types.QueryExecutorProfileRequest) (*types.QueryExecutorProfileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := q.GetExecutorProfile(ctx, req.Address)
	if !found {
		return nil, types.ErrExecutorNotFound.Wrapf("executor %s not found", req.Address)
	}

	return &types.QueryExecutorProfileResponse{Profile: profile}, nil
}

func (q queryServer) Executors(goCtx context.Context, req *types.QueryExecutorsRequest) (*types.QueryExecutorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var executors []types.ExecutorProfile
	q.IterateExecutorProfiles(ctx, func(p types.ExecutorProfile) bool {
		executors = append(executors, p)
		return len(executors) >= 100
	})

	return &types.QueryExecutorsResponse{Executors: executors}, nil
}

func (q queryServer) WorkParams(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

func (q queryServer) EpochStats(goCtx context.Context, req *types.QueryEpochStatsRequest) (*types.QueryEpochStatsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	stats, found := q.GetEpochStats(ctx, req.Epoch)
	if !found {
		return &types.QueryEpochStatsResponse{Stats: types.EpochStats{Epoch: req.Epoch}}, nil
	}
	return &types.QueryEpochStatsResponse{Stats: stats}, nil
}
