package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/oasyce/chain/x/reputation/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Reputation transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdSubmitFeedback(),
		CmdReportMisbehavior(),
	)
	return cmd
}

func CmdSubmitFeedback() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-feedback [invocation-id] [rating]",
		Short: "Submit feedback for an invocation. Rating: 0-500 (0.0-5.0)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			rating, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid rating: %w", err)
			}
			if rating > 500 {
				return fmt.Errorf("rating must be 0-500")
			}

			comment, _ := cmd.Flags().GetString("comment")

			msg := &types.MsgSubmitFeedback{
				Creator:      clientCtx.GetFromAddress().String(),
				InvocationId: args[0],
				Rating:       uint32(rating),
				Comment:      comment,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("comment", "", "Optional feedback comment")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdReportMisbehavior() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report [target-address] [evidence-type]",
		Short: "Report misbehavior",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			evidence, _ := cmd.Flags().GetString("evidence")

			msg := &types.MsgReportMisbehavior{
				Creator:      clientCtx.GetFromAddress().String(),
				Target:       args[0],
				EvidenceType: args[1],
				Evidence:     []byte(evidence),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("evidence", "", "Evidence data")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
