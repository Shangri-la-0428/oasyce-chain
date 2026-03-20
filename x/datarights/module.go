package datarights

import (
	"context"
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

	"github.com/oasyce/chain/x/datarights/cli"
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
type AppModuleBasic struct{
	cdc codec.Codec
}

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
func (b AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesisState()
	return cdc.MustMarshalJSON(gs)
}

// ValidateGenesis validates the module's genesis state as raw JSON.
func (b AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal datarights genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

// RegisterGRPCGatewayRoutes registers the module's gRPC gateway routes.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the datarights module.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return cli.GetTxCmd() }

// GetQueryCmd returns the root query command for the datarights module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

// AppModule implements the datarights module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new datarights AppModule.
func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         k,
	}
}

// Name returns the module name.
func (am AppModule) Name() string { return types.ModuleName }

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

	// Set params.
	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set datarights params: %v", err))
	}

	// Restore data assets.
	for _, asset := range gs.DataAssets {
		if err := am.keeper.SetAsset(ctx, asset); err != nil {
			panic(fmt.Sprintf("failed to set data asset %s: %v", asset.Id, err))
		}
	}

	// Restore shareholders.
	for _, sh := range gs.Shareholders {
		if err := am.keeper.SetShareHolder(ctx, sh); err != nil {
			panic(fmt.Sprintf("failed to set shareholder %s/%s: %v", sh.AssetId, sh.Address, err))
		}
	}

	// Restore disputes.
	for _, dispute := range gs.Disputes {
		if err := am.keeper.SetDispute(ctx, dispute); err != nil {
			panic(fmt.Sprintf("failed to set dispute %s: %v", dispute.Id, err))
		}
	}

	return nil
}

// ExportGenesis exports the module's current state as genesis.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var assets []types.DataAsset
	am.keeper.IterateAllAssets(ctx, func(a types.DataAsset) bool {
		assets = append(assets, a)
		return false
	})
	if assets == nil {
		assets = []types.DataAsset{}
	}

	var shareholders []types.ShareHolder
	am.keeper.IterateAllShareHolders(ctx, func(sh types.ShareHolder) bool {
		shareholders = append(shareholders, sh)
		return false
	})
	if shareholders == nil {
		shareholders = []types.ShareHolder{}
	}

	var disputes []types.Dispute
	am.keeper.IterateAllDisputes(ctx, func(d types.Dispute) bool {
		disputes = append(disputes, d)
		return false
	})
	if disputes == nil {
		disputes = []types.Dispute{}
	}

	gs := types.GenesisState{
		DataAssets:   assets,
		Shareholders: shareholders,
		Disputes:     disputes,
		Params:       am.keeper.GetParams(ctx),
	}

	return cdc.MustMarshalJSON(&gs)
}

// ConsensusVersion returns the module's consensus version.
func (am AppModule) ConsensusVersion() uint64 { return 1 }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}
