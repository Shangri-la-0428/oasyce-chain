package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/oasyce/chain/x/datarights/types"
)

// GetQueryCmd returns the query commands for the datarights module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Datarights query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdGetDataAsset(),
		CmdListDataAssets(),
		CmdShares(),
		CmdAccessLevel(),
		CmdGetDispute(),
		CmdListDisputes(),
		CmdGetMigrationPath(),
		CmdListMigrationPaths(),
		CmdListAssetChildren(),
		CmdDatarightsParams(),
	)
	return cmd
}

func CmdGetDataAsset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset [asset-id]",
		Short: "Query a data asset by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DataAsset(cmd.Context(), &types.QueryDataAssetRequest{
				AssetId: args[0],
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

func CmdListDataAssets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all data assets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DataAssets(cmd.Context(), &types.QueryDataAssetsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetMigrationPath() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration-path [source-asset-id] [target-asset-id]",
		Short: "Query a migration path between two assets",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.MigrationPath(cmd.Context(), &types.QueryMigrationPathRequest{
				SourceAssetId: args[0],
				TargetAssetId: args[1],
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

func CmdListMigrationPaths() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration-paths [source-asset-id]",
		Short: "List all migration paths from a source asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.MigrationPaths(cmd.Context(), &types.QueryMigrationPathsRequest{
				SourceAssetId: args[0],
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

func CmdListAssetChildren() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "children [parent-asset-id]",
		Short: "List all child/fork assets of a parent asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AssetChildren(cmd.Context(), &types.QueryAssetChildrenRequest{
				ParentAssetId: args[0],
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

func CmdShares() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shares [asset-id]",
		Short: "Query all shareholders of a data asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Shares(cmd.Context(), &types.QuerySharesRequest{
				AssetId: args[0],
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

func CmdAccessLevel() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-level [asset-id] [address]",
		Short: "Query access level for an address on a data asset",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AccessLevel(cmd.Context(), &types.QueryAccessLevelRequest{
				AssetId: args[0],
				Address: args[1],
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

func CmdDatarightsParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query datarights module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DatarightsParams(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdListDisputes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disputes",
		Short: "List all disputes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Disputes(cmd.Context(), &types.QueryDisputesRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [dispute-id]",
		Short: "Query a dispute by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Dispute(cmd.Context(), &types.QueryDisputeRequest{
				DisputeId: args[0],
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
