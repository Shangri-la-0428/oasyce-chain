package settlement

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

	"github.com/oasyce/chain/x/settlement/keeper"
	"github.com/oasyce/chain/x/settlement/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

// AppModuleBasic defines the basic application module for settlement.
type AppModuleBasic struct{}

// Name returns the module name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the module's types on the amino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers the module's interface types.
func (AppModuleBasic) RegisterInterfaces(_ codectypes.InterfaceRegistry) {}

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
		return fmt.Errorf("failed to unmarshal settlement genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

// RegisterGRPCGatewayRoutes registers the module's gRPC gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// GetTxCmd returns the root tx command for the settlement module.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns the root query command for the settlement module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

// AppModule implements the settlement module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new settlement AppModule.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

// RegisterInvariants registers module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers module gRPC services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Wire up msg and query servers once protobuf codegen is available.
	_ = keeper.NewMsgServer(am.keeper)
	_ = keeper.NewQueryServer(am.keeper)
}

// InitGenesis initializes the module's state from genesis.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal settlement genesis: %v", err))
	}

	// Set params.
	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set settlement params: %v", err))
	}

	// Restore escrows.
	for _, escrow := range gs.Escrows {
		if err := am.keeper.SetEscrow(ctx, escrow); err != nil {
			panic(fmt.Sprintf("failed to set escrow %s: %v", escrow.ID, err))
		}
	}

	// Restore bonding curve states.
	for _, bcs := range gs.BondingCurveStates {
		if err := am.keeper.SetBondingCurveState(ctx, bcs); err != nil {
			panic(fmt.Sprintf("failed to set bonding curve %s: %v", bcs.AssetID, err))
		}
	}

	return nil
}

// ExportGenesis exports the module's current state as genesis.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	var escrows []types.Escrow
	am.keeper.IterateAllEscrows(ctx, func(e types.Escrow) bool {
		escrows = append(escrows, e)
		return false
	})

	var bcs []types.BondingCurveState
	am.keeper.IterateAllBondingCurves(ctx, func(s types.BondingCurveState) bool {
		bcs = append(bcs, s)
		return false
	})

	gs := types.GenesisState{
		Escrows:            escrows,
		BondingCurveStates: bcs,
		Params:             am.keeper.GetParams(ctx),
	}

	bz, _ := json.Marshal(gs)
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock is called at the beginning of every block.
func (am AppModule) BeginBlock(_ sdk.Context) error { return nil }

// EndBlock is called at the end of every block. It expires stale escrows.
func (am AppModule) EndBlock(ctx sdk.Context) error {
	return am.keeper.ExpireStaleEscrows(ctx)
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}
