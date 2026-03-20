package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Settlement transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdCreateEscrow(),
		CmdReleaseEscrow(),
		CmdRefundEscrow(),
	)
	return cmd
}

func CmdCreateEscrow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-escrow [amount]",
		Short: "Create an escrow. Amount format: 1000uoas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			capId, _ := cmd.Flags().GetString("capability-id")
			assetId, _ := cmd.Flags().GetString("asset-id")

			msg := &types.MsgCreateEscrow{
				Creator:      clientCtx.GetFromAddress().String(),
				Amount:       coin,
				CapabilityId: capId,
				AssetId:      assetId,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("capability-id", "", "Capability ID (for capability escrows)")
	cmd.Flags().String("asset-id", "", "Asset ID (for data asset escrows)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdReleaseEscrow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release-escrow [escrow-id]",
		Short: "Release an escrow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgReleaseEscrow{
				Creator:  clientCtx.GetFromAddress().String(),
				EscrowId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRefundEscrow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund-escrow [escrow-id]",
		Short: "Refund an escrow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRefundEscrow{
				Creator:  clientCtx.GetFromAddress().String(),
				EscrowId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
