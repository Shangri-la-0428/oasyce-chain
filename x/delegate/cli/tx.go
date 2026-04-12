package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Delegate module transactions",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdSetPolicy(),
		CmdEnroll(),
		CmdRevoke(),
	)
	return cmd
}

// CmdSetPolicy — the ONE command the user runs. Everything else is automatic.
//
//	oasyced tx delegate set-policy \
//	  --token "my-secret" \
//	  --per-tx 1000000uoas \
//	  --daily 10000000uoas \
//	  --allow "/oasyce.datarights.v1.MsgBuyShares,/oasyce.datarights.v1.MsgSellShares,/oasyce.capability.v1.MsgInvokeCapability"
func CmdSetPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-policy",
		Short: "Set delegation policy (one command, all rules)",
		Long: `Set a delegation policy for your account. Agents enroll with the token and
operate within the spending limits. One command, zero maintenance.

Example:
  oasyced tx delegate set-policy \
    --token "my-agent-secret" \
    --per-tx 1000000uoas \
    --daily 10000000uoas \
    --allow "/oasyce.datarights.v1.MsgBuyShares,/oasyce.capability.v1.MsgInvokeCapability" \
    --from mykey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			token, _ := cmd.Flags().GetString("token")
			perTxStr, _ := cmd.Flags().GetString("per-tx")
			dailyStr, _ := cmd.Flags().GetString("daily")
			allowStr, _ := cmd.Flags().GetString("allow")
			windowSecs, _ := cmd.Flags().GetUint64("window")
			expireSecs, _ := cmd.Flags().GetUint64("expire")
			maxMsgsPerExec, _ := cmd.Flags().GetInt32("max-msgs-per-exec")

			perTx, err := sdk.ParseCoinNormalized(perTxStr)
			if err != nil {
				return err
			}
			daily, err := sdk.ParseCoinNormalized(dailyStr)
			if err != nil {
				return err
			}

			allowedMsgs := strings.Split(allowStr, ",")
			for i := range allowedMsgs {
				allowedMsgs[i] = strings.TrimSpace(allowedMsgs[i])
			}

			if windowSecs == 0 {
				windowSecs = 86400
			}

			msg := &types.MsgSetPolicy{
				Principal:         clientCtx.GetFromAddress().String(),
				PerTxLimit:        perTx,
				WindowLimit:       daily,
				WindowSeconds:     windowSecs,
				AllowedMsgs:       allowedMsgs,
				EnrollmentToken:   token,
				ExpirationSeconds: expireSecs,
				MaxMsgsPerExec:    maxMsgsPerExec,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("token", "", "Shared enrollment token (required)")
	cmd.Flags().String("per-tx", "1000000uoas", "Max spend per transaction (default 1 OAS)")
	cmd.Flags().String("daily", "10000000uoas", "Daily spend cap for all delegates combined (default 10 OAS)")
	cmd.Flags().String("allow", "", "Comma-separated list of allowed msg type URLs (required)")
	cmd.Flags().Uint64("window", 86400, "Budget window in seconds (default 86400 = 1 day)")
	cmd.Flags().Uint64("expire", 0, "Policy expiration in seconds (0 = no expiry)")
	cmd.Flags().Int32("max-msgs-per-exec", 0, "Max inner messages per execution (0 = server default 16)")
	_ = cmd.MarkFlagRequired("token")
	_ = cmd.MarkFlagRequired("allow")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdEnroll — agent runs this automatically on first start.
func CmdEnroll() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enroll [principal-address]",
		Short: "Enroll as delegate under a principal (agent auto-runs this)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			token, _ := cmd.Flags().GetString("token")
			label, _ := cmd.Flags().GetString("label")

			msg := &types.MsgEnroll{
				Delegate:  clientCtx.GetFromAddress().String(),
				Principal: args[0],
				Token:     token,
				Label:     label,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("token", "", "Enrollment token from principal")
	cmd.Flags().String("label", "", "Optional label (e.g. 'macbook-agent-1')")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdRevoke — principal removes a delegate.
func CmdRevoke() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke [delegate-address]",
		Short: "Remove a delegate from your policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRevoke{
				Principal: clientCtx.GetFromAddress().String(),
				Delegate:  args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
