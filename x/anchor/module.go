package anchor

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

	"github.com/oasyce/chain/x/anchor/cli"
	"github.com/oasyce/chain/x/anchor/keeper"
	"github.com/oasyce/chain/x/anchor/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

// AppModuleBasic defines the basic application module for anchor.
type AppModuleBasic struct{}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's types on the amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns the module's default genesis state as raw JSON.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis validates the module's genesis state as raw JSON.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal anchor genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

// RegisterGRPCGatewayRoutes registers the module's gRPC gateway routes.
// Note: The anchor module does not register gRPC-gateway routes because
// query request/response types use raw bytes fields that don't map cleanly
// to REST path parameters. Use gRPC or CLI instead.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd returns the root tx command for the anchor module.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return cli.GetTxCmd() }

// GetQueryCmd returns the root query command for the anchor module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

// AppModule implements the anchor module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new anchor AppModule.
func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

// RegisterInvariants registers module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers module gRPC services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

// InitGenesis initializes the module's state from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	// Restore anchor records.
	for _, record := range gs.Anchors {
		if err := am.keeper.SetAnchor(ctx, record); err != nil {
			panic(fmt.Sprintf("failed to set anchor record for trace_id %x: %v", record.TraceId, err))
		}
	}

	return nil
}

// ExportGenesis exports the module's current state as genesis.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var anchors []types.AnchorRecord
	am.keeper.IterateAllAnchors(ctx, func(record types.AnchorRecord) bool {
		anchors = append(anchors, record)
		return false
	})

	gs := types.GenesisState{
		Anchors: anchors,
	}

	return cdc.MustMarshalJSON(&gs)
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
