package cmd

import (
	"errors"
	"io"
	"os"

	"cosmossdk.io/log"

	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/oasyce/chain/app"
)

// NewRootCmd creates the root command for the oasyced daemon.
func NewRootCmd() *cobra.Command {
	// Address prefixes are already set in app.init().

	// Create the encoding config.
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	// Register module interfaces.
	authtypes.RegisterInterfaces(interfaceRegistry)

	initClientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithLegacyAmino(legacyAmino).
		WithTxConfig(txConfig).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("OASYCE")

	rootCmd := &cobra.Command{
		Use:   "oasyced",
		Short: "Oasyce Chain Daemon",
		Long:  "oasyced is the daemon for the Oasyce cosmos-sdk blockchain.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}
			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()

			cmtCfg := cmtcfg.DefaultConfig()
			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, cmtCfg)
		},
	}

	initRootCmd(rootCmd, txConfig)
	return rootCmd
}

func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = "0uoas"
	return serverconfig.DefaultConfigTemplate, srvCfg
}

func initRootCmd(rootCmd *cobra.Command, txCfg client.TxConfig) {
	addrCodec := addresscodec.NewBech32Codec("oasyce")
	valCodec := addresscodec.NewBech32Codec("oasycevaloper")

	genesisCmd := &cobra.Command{
		Use:                        "genesis",
		Short:                      "Application's genesis-related subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
	}
	genesisCmd.AddCommand(
		genutilcli.AddGenesisAccountCmd(app.DefaultNodeHome, addrCodec),
		genutilcli.GenTxCmd(app.ModuleBasics, txCfg, banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, valCodec),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome, genutiltypes.DefaultMessageValidator, valCodec),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),
	)

	rootCmd.AddCommand(
		OasyceInitCmd(),
		genesisCmd,
		debug.Cmd(),
		keys.Commands(),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)
}

func newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	return app.NewOasyceApp(
		logger,
		db,
		traceStore,
		true,
		appOpts,
	)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	_ = app.NewOasyceApp(logger, db, traceStore, false, appOpts)
	_ = height
	_ = forZeroHeight
	_ = jailAllowedAddrs
	_ = modulesToExport
	return servertypes.ExportedApp{}, errors.New("export not implemented yet")
}

func addModuleInitFlags(startCmd *cobra.Command) {
	// Standard flags are already added by the server module.
	// Add custom module flags here if needed.
	_ = startCmd
}
