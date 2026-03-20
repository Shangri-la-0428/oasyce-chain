package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capability transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdRegisterCapability(),
		CmdInvokeCapability(),
	)
	return cmd
}

func CmdRegisterCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [name] [endpoint-url] [price]",
		Short: "Register a capability. Price format: 1000uoas",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			price, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			tagsStr, _ := cmd.Flags().GetString("tags")
			rateLimit, _ := cmd.Flags().GetUint64("rate-limit")

			var tags []string
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
			}

			msg := &types.MsgRegisterCapability{
				Creator:      clientCtx.GetFromAddress().String(),
				Name:         args[0],
				EndpointUrl:  args[1],
				PricePerCall: price,
				Description:  description,
				Tags:         tags,
				RateLimit:    rateLimit,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Capability description")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().Uint64("rate-limit", 100, "Max calls per block")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdInvokeCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke [capability-id]",
		Short: "Invoke a capability",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			input, _ := cmd.Flags().GetString("input")

			msg := &types.MsgInvokeCapability{
				Creator:      clientCtx.GetFromAddress().String(),
				CapabilityId: args[0],
				Input:        []byte(input),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("input", "", "Input data (JSON string)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

