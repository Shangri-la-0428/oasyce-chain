package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/oasyce/chain/x/onboarding/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Onboarding transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdSelfRegister(),
		CmdRepayDebt(),
		CmdUpdateParams(),
	)
	return cmd
}

func CmdSelfRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [nonce]",
		Short: "Register as a new user with a PoW nonce",
		Long: `Register as a new user by providing a proof-of-work nonce.
The nonce must satisfy: sha256(address || nonce_le_bytes) has N leading zero bits.
Use the 'oasyced pow' command or client SDK to find a valid nonce.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			nonce, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid nonce: %s", args[0])
			}

			msg := &types.MsgSelfRegister{
				Creator: clientCtx.GetFromAddress().String(),
				Nonce:   nonce,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRepayDebt() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repay [amount]",
		Short: "Repay airdrop debt (amount in uoas)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, ok := math.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid amount: %s", args[0])
			}

			msg := &types.MsgRepayDebt{
				Creator: clientCtx.GetFromAddress().String(),
				Amount:  amount,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateParams creates a CLI command for governance-gated parameter update.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [params-json-file]",
		Short: "Update module parameters (governance only)",
		Long:  "Submit a transaction to update onboarding module parameters. The params-json-file should contain the full Params JSON.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bz, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			var params types.Params
			if err := json.Unmarshal(bz, &params); err != nil {
				return err
			}

			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    params,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
