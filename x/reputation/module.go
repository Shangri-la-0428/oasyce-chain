package reputation

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/oasyce/chain/x/reputation/keeper"
	"github.com/oasyce/chain/x/reputation/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

// AppModuleBasic defines the basic application module for reputation.
type AppModuleBasic struct{}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's types on the amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (AppModuleBasic) RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// Will register proper protobuf interfaces once codegen is ready.
}

// DefaultGenesis returns the module's default genesis state as raw JSON.
func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesisState()
	bz, _ := json.Marshal(gs)
	return bz
}

// ValidateGenesis validates the module's genesis state as raw JSON.
func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal reputation genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

// RegisterGRPCGatewayRoutes registers the module's gRPC gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd returns the root tx command for the reputation module.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns the root query command for the reputation module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

// AppModule implements the reputation module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new reputation AppModule.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

// Name returns the module name.
func (am AppModule) Name() string { return types.ModuleName }

// RegisterInvariants registers module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers module gRPC services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Once protobuf codegen is available, register MsgServer and QueryServer here:
	// types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	// types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
	_ = cfg
}

// InitGenesis initializes the module's state from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal reputation genesis: %v", err))
	}

	// Set params.
	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set reputation params: %v", err))
	}

	// Restore scores.
	for _, score := range gs.Scores {
		if err := am.keeper.SetReputation(ctx, score); err != nil {
			panic(fmt.Sprintf("failed to set reputation score for %s: %v", score.Address, err))
		}
	}

	// Restore feedbacks.
	for _, fb := range gs.Feedbacks {
		if err := am.keeper.SetFeedback(ctx, fb); err != nil {
			panic(fmt.Sprintf("failed to set feedback %s: %v", fb.ID, err))
		}
	}

	// Restore reports.
	for _, report := range gs.Reports {
		if err := am.keeper.SetReport(ctx, report); err != nil {
			panic(fmt.Sprintf("failed to set report %s: %v", report.ID, err))
		}
	}

	return nil
}

// ExportGenesis exports the module's current state as genesis.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	var scores []types.ReputationScore
	am.keeper.IterateAllScores(ctx, func(s types.ReputationScore) bool {
		scores = append(scores, s)
		return false
	})

	var feedbacks []types.Feedback
	am.keeper.IterateAllFeedbacks(ctx, func(fb types.Feedback) bool {
		feedbacks = append(feedbacks, fb)
		return false
	})

	gs := types.GenesisState{
		Scores:    scores,
		Feedbacks: feedbacks,
		Reports:   []types.MisbehaviorReport{},
		Params:    am.keeper.GetParams(ctx),
	}

	bz, _ := json.Marshal(gs)
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock is called at the beginning of every block.
func (am AppModule) BeginBlock(_ sdk.Context) error { return nil }

// EndBlock is called at the end of every block.
func (am AppModule) EndBlock(_ sdk.Context) error { return nil }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}
