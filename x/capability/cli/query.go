package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/oasyce/chain/x/capability/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capability query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdGetCapability(),
		CmdListCapabilities(),
		CmdCapabilitiesByProvider(),
		CmdEarnings(),
		CmdGetInvocation(),
		CmdCapabilityParams(),
	)
	return cmd
}

func CmdGetCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [capability-id]",
		Short: "Query a capability by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Capability(cmd.Context(), &types.QueryCapabilityRequest{
				CapabilityId: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdListCapabilities() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Capabilities(cmd.Context(), &types.QueryCapabilitiesRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdCapabilitiesByProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-provider [provider-address]",
		Short: "Query all capabilities for a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.CapabilitiesByProvider(cmd.Context(), &types.QueryCapabilitiesByProviderRequest{
				Provider: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetInvocation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invocation [invocation-id]",
		Short: "Query an invocation by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Invocation(cmd.Context(), &types.QueryInvocationRequest{
				InvocationId: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdCapabilityParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query capability module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.CapabilityParams(cmd.Context(), &types.QueryCapabilityParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdEarnings() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "earnings [provider-address]",
		Short: "Query provider earnings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Earnings(cmd.Context(), &types.QueryEarningsRequest{
				Provider: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
