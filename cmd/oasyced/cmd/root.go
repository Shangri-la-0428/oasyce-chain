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
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	txsigning "cosmossdk.io/x/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"

	distcli "github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"

	"github.com/oasyce/chain/app"
	capabilitycli "github.com/oasyce/chain/x/capability/cli"
	datarightscli "github.com/oasyce/chain/x/datarights/cli"
	reputationcli "github.com/oasyce/chain/x/reputation/cli"
	settlementcli "github.com/oasyce/chain/x/settlement/cli"
	workcli "github.com/oasyce/chain/x/work/cli"
	onboardingcli "github.com/oasyce/chain/x/onboarding/cli"
	anchorcli "github.com/oasyce/chain/x/anchor/cli"
	delegatecli "github.com/oasyce/chain/x/delegate/cli"
	sigilcli "github.com/oasyce/chain/x/sigil/cli"
)

// NewRootCmd creates the root command for the oasyced daemon.
func NewRootCmd() *cobra.Command {
	// Address prefixes are already set in app.init().

	// Create the encoding config with proper address codecs.
	addrCodec := addresscodec.NewBech32Codec("oasyce")
	valAddrCodec := addresscodec.NewBech32Codec("oasycevaloper")
	consAddrCodec := addresscodec.NewBech32Codec("oasycevalcons")

	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: txsigning.Options{
			AddressCodec:          addrCodec,
			ValidatorAddressCodec: valAddrCodec,
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	// Register all module interfaces (crypto keys, staking msgs, etc.).
	app.ModuleBasics.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)

	_ = consAddrCodec // available for future use

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

	// Query commands
	queryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
	}
	queryCmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)
	app.ModuleBasics.AddQueryCommands(queryCmd)
	queryCmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	// Tx commands
	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
	}
	txCmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)
	// Note: ModuleBasics.AddTxCommands panics because distr/staking modules
	// need AddressCodec which isn't set in BasicManager. Add custom module
	// tx commands individually.
	txCmd.AddCommand(
		bankcli.NewSendTxCmd(addrCodec),
		stakingcli.NewTxCmd(valCodec, addrCodec),
		distcli.NewTxCmd(valCodec, addrCodec),
		govcli.NewTxCmd(nil),
		datarightscli.GetTxCmd(),
		settlementcli.GetTxCmd(),
		capabilitycli.GetTxCmd(),
		reputationcli.GetTxCmd(),
		workcli.GetTxCmd(),
		onboardingcli.GetTxCmd(),
		anchorcli.GetTxCmd(),
		delegatecli.GetTxCmd(),
		sigilcli.GetTxCmd(),
	)
	txCmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	rootCmd.AddCommand(
		OasyceInitCmd(),
		genesisCmd,
		queryCmd,
		txCmd,
		debug.Cmd(),
		keys.Commands(),
		UtilCmd(),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)
}

func newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	return app.NewOasyceApp(
		logger,
		db,
		traceStore,
		true,
		appOpts,
		baseappOptions...,
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
