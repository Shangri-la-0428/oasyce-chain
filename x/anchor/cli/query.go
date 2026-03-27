package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/oasyce/chain/x/anchor/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Anchor query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryAnchor(),
		CmdIsAnchored(),
		CmdAnchorsByCapability(),
		CmdAnchorsByNode(),
	)
	return cmd
}

// CmdQueryAnchor queries an anchor record by trace_id.
func CmdQueryAnchor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [trace-id-hex]",
		Short: "Query an anchor record by trace_id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			traceID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid trace_id hex: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Anchor(cmd.Context(), &types.QueryAnchorRequest{
				TraceId: traceID,
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

// CmdIsAnchored checks whether a trace_id has been anchored.
func CmdIsAnchored() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-anchored [trace-id-hex]",
		Short: "Check if a trace_id has been anchored",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			traceID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid trace_id hex: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.IsAnchored(cmd.Context(), &types.QueryIsAnchoredRequest{
				TraceId: traceID,
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

// CmdAnchorsByCapability queries anchors by capability.
func CmdAnchorsByCapability() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-capability [capability]",
		Short: "Query anchors by capability",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AnchorsByCapability(cmd.Context(), &types.QueryAnchorsByCapabilityRequest{
				Capability: args[0],
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

// CmdAnchorsByNode queries anchors by node public key.
func CmdAnchorsByNode() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-node [node-pubkey-hex]",
		Short: "Query anchors by node public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			nodePubkey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid node_pubkey hex: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AnchorsByNode(cmd.Context(), &types.QueryAnchorsByNodeRequest{
				NodePubkey: nodePubkey,
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
