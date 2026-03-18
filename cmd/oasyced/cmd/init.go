package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math/unsafe"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/oasyce/chain/app"
)

const (
	flagOverwrite = "overwrite"
	flagRecover   = "recover"
)

type oasyceInitInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

// OasyceInitCmd returns a command that initializes all files needed for the
// Oasyce chain node. It produces a genesis.json with "uoas" as the bond denom
// and Oasyce-specific module parameters pre-configured.
func OasyceInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long:  "Initialize validator and node configuration files with Oasyce-specific defaults.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			switch {
			case chainID != "":
			case clientCtx.ChainID != "":
				chainID = clientCtx.ChainID
			default:
				chainID = fmt.Sprintf("test-chain-%v", unsafe.Str(6))
			}

			// Handle recovery from mnemonic.
			var mnemonic string
			recover, _ := cmd.Flags().GetBool(flagRecover)
			if recover {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				value, err := input.GetString("Enter your bip39 mnemonic", inBuf)
				if err != nil {
					return err
				}
				mnemonic = value
				if !bip39.IsMnemonicValid(mnemonic) {
					return errors.New("invalid mnemonic")
				}
			}

			initHeight, _ := cmd.Flags().GetInt64(flags.FlagInitHeight)
			if initHeight < 1 {
				initHeight = 1
			}

			nodeID, _, err := genutil.InitializeNodeValidatorFilesFromMnemonic(config, mnemonic)
			if err != nil {
				return err
			}

			config.Moniker = args[0]

			genFile := config.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(flagOverwrite)

			_, err = os.Stat(genFile)
			if !overwrite && !os.IsNotExist(err) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}

			// Use the Oasyce-specific default genesis with all denoms = "uoas"
			// and custom module parameters pre-configured.
			appGenState := app.DefaultOasyceGenesis(cdc)

			appState, err := json.MarshalIndent(appGenState, "", " ")
			if err != nil {
				return errorsmod.Wrap(err, "failed to marshal default genesis state")
			}

			appGenesis := &genutiltypes.AppGenesis{}
			if _, err := os.Stat(genFile); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				appGenesis, err = genutiltypes.AppGenesisFromFile(genFile)
				if err != nil {
					return errorsmod.Wrap(err, "failed to read genesis doc from file")
				}
			}

			appGenesis.AppName = version.AppName
			appGenesis.AppVersion = version.Version
			appGenesis.ChainID = chainID
			appGenesis.AppState = appState
			appGenesis.InitialHeight = initHeight
			appGenesis.Consensus = &genutiltypes.ConsensusGenesis{
				Validators: nil,
			}

			if err = genutil.ExportGenesisFile(appGenesis, genFile); err != nil {
				return errorsmod.Wrap(err, "failed to export genesis file")
			}

			info := oasyceInitInfo{
				Moniker:    config.Moniker,
				ChainID:    chainID,
				NodeID:     nodeID,
				AppMessage: appState,
			}
			out, err := json.MarshalIndent(info, "", " ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(os.Stderr, "%s\n", out)
			if err != nil {
				return err
			}

			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(flagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().Bool(flagRecover, false, "provide seed phrase to recover existing key instead of creating")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().Int64(flags.FlagInitHeight, 1, "specify the initial block height at genesis")

	return cmd
}
