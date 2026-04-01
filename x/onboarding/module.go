package onboarding

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

	"github.com/oasyce/chain/x/onboarding/cli"
	"github.com/oasyce/chain/x/onboarding/keeper"
	"github.com/oasyce/chain/x/onboarding/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

type AppModuleBasic struct {
	cdc codec.Codec
}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (b AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesisState()
	return cdc.MustMarshalJSON(gs)
}

func (b AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal onboarding genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModuleBasic) GetTxCmd() *cobra.Command   { return cli.GetTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         k,
	}
}

func (am AppModule) Name() string { return types.ModuleName }

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set onboarding params: %v", err))
	}

	for _, reg := range gs.Registrations {
		if err := am.keeper.SetRegistration(ctx, reg); err != nil {
			panic(fmt.Sprintf("failed to set registration %s: %v", reg.Address, err))
		}
		am.keeper.RebuildDeadlineIndex(ctx, reg)
	}

	// Derive total_registrations counter from registration count.
	am.keeper.SetTotalRegistrations(ctx, uint64(len(gs.Registrations)))

	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var registrations []types.Registration
	am.keeper.IterateAllRegistrations(ctx, func(reg types.Registration) bool {
		registrations = append(registrations, reg)
		return false
	})
	if registrations == nil {
		registrations = []types.Registration{}
	}

	gs := types.GenesisState{
		Registrations: registrations,
		Params:        am.keeper.GetParams(ctx),
	}

	return cdc.MustMarshalJSON(&gs)
}

// EndBlock marks expired PoW debts as DEFAULTED.
func (am AppModule) EndBlock(ctx sdk.Context) []abci.ValidatorUpdate {
	am.keeper.ExpireDebts(ctx)
	return nil
}

func (am AppModule) ConsensusVersion() uint64 { return 3 }

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}
