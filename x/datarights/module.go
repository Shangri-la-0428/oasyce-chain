package datarights

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/oasyce/chain/x/datarights/keeper"
	"github.com/oasyce/chain/x/datarights/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

// AppModuleBasic defines the basic application module for datarights.
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
		return fmt.Errorf("failed to unmarshal datarights genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

// RegisterGRPCGatewayRoutes registers the module's gRPC gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

// AppModule implements the datarights module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new datarights AppModule.
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
		panic(fmt.Sprintf("failed to unmarshal datarights genesis: %v", err))
	}

	// Set params.
	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set datarights params: %v", err))
	}

	// Restore data assets.
	for _, asset := range gs.DataAssets {
		if err := am.keeper.SetAsset(ctx, asset); err != nil {
			panic(fmt.Sprintf("failed to set data asset %s: %v", asset.ID, err))
		}
	}

	// Restore shareholders.
	for _, sh := range gs.ShareHolders {
		if err := am.keeper.SetShareHolder(ctx, sh); err != nil {
			panic(fmt.Sprintf("failed to set shareholder %s/%s: %v", sh.AssetID, sh.Address, err))
		}
	}

	// Restore disputes.
	for _, dispute := range gs.Disputes {
		if err := am.keeper.SetDispute(ctx, dispute); err != nil {
			panic(fmt.Sprintf("failed to set dispute %s: %v", dispute.ID, err))
		}
	}

	return nil
}

// ExportGenesis exports the module's current state as genesis.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	var assets []types.DataAsset
	am.keeper.IterateAllAssets(ctx, func(a types.DataAsset) bool {
		assets = append(assets, a)
		return false
	})

	gs := types.GenesisState{
		DataAssets:   assets,
		ShareHolders: []types.ShareHolder{},
		Disputes:     []types.Dispute{},
		Params:       am.keeper.GetParams(ctx),
	}

	bz, _ := json.Marshal(gs)
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (am AppModule) ConsensusVersion() uint64 { return 1 }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}
