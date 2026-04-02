package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/oasyce/chain/x/delegate/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Delegate module queries",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryPolicy(),
		CmdQueryDelegates(),
		CmdQuerySpend(),
		CmdQueryPrincipal(),
	)
	return cmd
}

func CmdQueryPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy [principal-address]",
		Short: "Query a principal's delegation policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			qc := types.NewQueryClient(clientCtx)
			res, err := qc.Policy(cmd.Context(), &types.QueryPolicyRequest{Principal: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryDelegates() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegates [principal-address]",
		Short: "List all delegates enrolled under a principal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			qc := types.NewQueryClient(clientCtx)
			res, err := qc.Delegates(cmd.Context(), &types.QueryDelegatesRequest{Principal: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQuerySpend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spend [principal-address]",
		Short: "Query current spending window for a principal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			qc := types.NewQueryClient(clientCtx)
			res, err := qc.Spend(cmd.Context(), &types.QuerySpendRequest{Principal: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryPrincipal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "principal [delegate-address]",
		Short: "Query which principal a delegate belongs to",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			qc := types.NewQueryClient(clientCtx)
			res, err := qc.Principal(cmd.Context(), &types.QueryPrincipalRequest{Delegate: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
