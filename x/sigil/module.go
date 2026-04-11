package sigil

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

	"github.com/oasyce/chain/x/sigil/cli"
	"github.com/oasyce/chain/x/sigil/keeper"
	"github.com/oasyce/chain/x/sigil/types"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// ---------------------------------------------------------------------------
// AppModuleBasic
// ---------------------------------------------------------------------------

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
		return fmt.Errorf("failed to unmarshal sigil genesis state: %w", err)
	}
	return types.ValidateGenesis(gs)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModuleBasic) GetTxCmd() *cobra.Command { return cli.GetTxCmd() }

func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// ---------------------------------------------------------------------------
// AppModule
// ---------------------------------------------------------------------------

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{}, keeper: k}
}

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
	if err := cfg.RegisterMigration(types.ModuleName, 1, am.keeper.Migrate1to2); err != nil {
		panic(err)
	}
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	// Restore params.
	if err := am.keeper.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set sigil params: %v", err))
	}

	// Restore sigils.
	var activeCount uint64
	for _, s := range gs.Sigils {
		if err := am.keeper.SetSigil(ctx, s); err != nil {
			panic(fmt.Sprintf("failed to set sigil %s: %v", s.SigilId, err))
		}
		if types.SigilStatus(s.Status) == types.SigilStatusActive {
			activeCount++
		}
		// Restore lineage edges.
		for _, parentID := range s.Lineage {
			am.keeper.SetLineage(ctx, parentID, s.SigilId)
		}
	}
	am.keeper.SetActiveCount(ctx, activeCount)

	// Restore bonds.
	for _, b := range gs.Bonds {
		if err := am.keeper.SetBond(ctx, b); err != nil {
			panic(fmt.Sprintf("failed to set bond %s: %v", b.BondId, err))
		}
	}

	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var sigils []types.Sigil
	am.keeper.IterateAllSigils(ctx, func(s types.Sigil) bool {
		sigils = append(sigils, s)
		return false
	})

	var bonds []types.Bond
	am.keeper.IterateAllBonds(ctx, func(b types.Bond) bool {
		bonds = append(bonds, b)
		return false
	})

	gs := types.GenesisState{
		Sigils: sigils,
		Bonds:  bonds,
		Params: am.keeper.GetParams(ctx),
	}

	return cdc.MustMarshalJSON(&gs)
}

func (AppModule) ConsensusVersion() uint64 { return 2 }

func (am AppModule) BeginBlock(ctx sdk.Context) error {
	return am.keeper.BeginBlocker(ctx)
}

func (am AppModule) EndBlock(_ sdk.Context) error { return nil }

func (am AppModule) IsOnePerModuleType() {}
func (am AppModule) IsAppModule()        {}
