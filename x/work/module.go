package work

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/oasyce/chain/x/work/cli"
	"github.com/oasyce/chain/x/work/keeper"
	"github.com/oasyce/chain/x/work/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// ---- AppModuleBasic ----

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal work genesis: %w", err)
	}
	return types.ValidateGenesis(gs)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ---- AppModule ----

type AppModule struct {
	AppModuleBasic
	cdc    codec.Codec
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		cdc:            cdc,
		keeper:         k,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set work params: %v", err))
	}

	// Restore task counter
	if gs.TaskCounter > 0 {
		am.keeper.SetTaskCounter(ctx, gs.TaskCounter)
	}

	// Restore tasks and rebuild secondary indexes.
	for _, task := range gs.Tasks {
		if err := am.keeper.SetTask(ctx, task); err != nil {
			panic(fmt.Sprintf("failed to restore task %d: %v", task.Id, err))
		}
		am.keeper.RebuildTaskIndexes(ctx, task)
	}

	// Restore executor profiles
	for _, exec := range gs.Executors {
		if err := am.keeper.SetExecutorProfile(ctx, exec); err != nil {
			panic(fmt.Sprintf("failed to restore executor %s: %v", exec.Address, err))
		}
	}

	// Restore commitments
	for _, gc := range gs.Commitments {
		hashBz, err := hex.DecodeString(gc.CommitmentHash)
		if err != nil {
			panic(fmt.Sprintf("failed to decode commitment hash for task %d: %v", gc.TaskId, err))
		}
		c := types.Commitment{
			Executor:   gc.Executor,
			TaskId:     gc.TaskId,
			CommitHash: hashBz,
		}
		if err := am.keeper.SetCommitment(ctx, c); err != nil {
			panic(fmt.Sprintf("failed to restore commitment task=%d executor=%s: %v", gc.TaskId, gc.Executor, err))
		}
	}

	// Restore results
	for _, gr := range gs.Results {
		outputBz, err := hex.DecodeString(gr.OutputHash)
		if err != nil {
			panic(fmt.Sprintf("failed to decode output hash for task %d: %v", gr.TaskId, err))
		}
		saltBz, err := hex.DecodeString(gr.Salt)
		if err != nil {
			panic(fmt.Sprintf("failed to decode salt for task %d: %v", gr.TaskId, err))
		}
		r := types.Result{
			Executor:    gr.Executor,
			TaskId:      gr.TaskId,
			OutputHash:  outputBz,
			Salt:        saltBz,
			Unavailable: gr.InputUnavailable,
		}
		if err := am.keeper.SetResult(ctx, r); err != nil {
			panic(fmt.Sprintf("failed to restore result task=%d executor=%s: %v", gr.TaskId, gr.Executor, err))
		}
	}

	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := types.GenesisState{
		Params:      am.keeper.GetParams(ctx),
		TaskCounter: am.keeper.GetTaskCounter(ctx),
	}

	am.keeper.IterateAllTasks(ctx, func(task types.Task) bool {
		gs.Tasks = append(gs.Tasks, task)
		return false
	})

	am.keeper.IterateExecutorProfiles(ctx, func(p types.ExecutorProfile) bool {
		gs.Executors = append(gs.Executors, p)
		return false
	})

	// Export commitments
	am.keeper.IterateAllCommitments(ctx, func(c types.Commitment) bool {
		gs.Commitments = append(gs.Commitments, types.TaskCommitment{
			TaskId:         c.TaskId,
			Executor:       c.Executor,
			CommitmentHash: hex.EncodeToString(c.CommitHash),
		})
		return false
	})

	// Export results
	am.keeper.IterateAllResults(ctx, func(r types.Result) bool {
		gs.Results = append(gs.Results, types.TaskResult{
			TaskId:           r.TaskId,
			Executor:         r.Executor,
			OutputHash:       hex.EncodeToString(r.OutputHash),
			Salt:             hex.EncodeToString(r.Salt),
			InputUnavailable: r.Unavailable,
		})
		return false
	})

	return cdc.MustMarshalJSON(&gs)
}

func (AppModule) ConsensusVersion() uint64 { return 2 }

func (am AppModule) BeginBlock(ctx sdk.Context) error {
	return am.keeper.BeginBlocker(ctx)
}

func (am AppModule) EndBlock(ctx sdk.Context) error {
	return am.keeper.EndBlocker(ctx)
}

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}
