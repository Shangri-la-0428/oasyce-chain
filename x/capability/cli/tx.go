package cli

import (
	"encoding/json"
	"os"
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
		CmdUpdateCapability(),
		CmdDeactivateCapability(),
		CmdCompleteInvocation(),
		CmdFailInvocation(),
		CmdClaimInvocation(),
		CmdDisputeInvocation(),
		CmdUpdateParams(),
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

func CmdUpdateCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [capability-id]",
		Short: "Update a capability's description, price, or tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			priceStr, _ := cmd.Flags().GetString("price")
			tagsStr, _ := cmd.Flags().GetString("tags")

			var price *sdk.Coin
			if priceStr != "" {
				p, err := sdk.ParseCoinNormalized(priceStr)
				if err != nil {
					return err
				}
				price = &p
			}

			_ = tagsStr // tags update not supported in proto yet

			msg := &types.MsgUpdateCapability{
				Creator:      clientCtx.GetFromAddress().String(),
				CapabilityId: args[0],
				Description:  description,
			}
			if price != nil {
				msg.PricePerCall = price
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("price", "", "New price (e.g. 200000uoas)")
	cmd.Flags().String("tags", "", "New comma-separated tags")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDeactivateCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate [capability-id]",
		Short: "Deactivate a capability (only owner can deactivate)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeactivateCapability{
				Creator:      clientCtx.GetFromAddress().String(),
				CapabilityId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdCompleteInvocation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete-invocation [invocation-id] [output-hash]",
		Short: "Submit output hash for a pending invocation (starts challenge window)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			usageReport, _ := cmd.Flags().GetString("usage-report")

			msg := &types.MsgCompleteInvocation{
				Creator:      clientCtx.GetFromAddress().String(),
				InvocationId: args[0],
				OutputHash:   args[1],
				UsageReport:  usageReport,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("usage-report", "", "Optional JSON usage metadata (e.g. token counts)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFailInvocation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fail-invocation [invocation-id]",
		Short: "Report a failed invocation and refund escrow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFailInvocation{
				Creator:      clientCtx.GetFromAddress().String(),
				InvocationId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdClaimInvocation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-invocation [invocation-id]",
		Short: "Claim payment for a completed invocation after challenge window",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimInvocation{
				Creator:      clientCtx.GetFromAddress().String(),
				InvocationId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDisputeInvocation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute-invocation [invocation-id] [reason]",
		Short: "Dispute a completed invocation within challenge window",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDisputeInvocation{
				Creator:      clientCtx.GetFromAddress().String(),
				InvocationId: args[0],
				Reason:       args[1],
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
		Long:  "Submit a transaction to update capability module parameters. The params-json-file should contain the full Params JSON.",
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

