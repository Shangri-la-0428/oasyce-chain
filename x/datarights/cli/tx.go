package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
)

// GetTxCmd returns the transaction commands for the datarights module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Datarights transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdRegisterDataAsset(),
		CmdBuyShares(),
		CmdFileDispute(),
		CmdResolveDispute(),
	)
	return cmd
}

// CmdRegisterDataAsset creates a RegisterDataAsset transaction.
func CmdRegisterDataAsset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [name] [content-hash]",
		Short: "Register a new data asset",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			contentHash := args[1]
			description, _ := cmd.Flags().GetString("description")
			rightsTypeStr, _ := cmd.Flags().GetString("rights-type")
			tagsStr, _ := cmd.Flags().GetString("tags")

			var rightsType types.RightsType
			switch strings.ToLower(rightsTypeStr) {
			case "original":
				rightsType = types.RIGHTS_TYPE_ORIGINAL
			case "co_creation", "cocreation":
				rightsType = types.RIGHTS_TYPE_CO_CREATION
			case "licensed":
				rightsType = types.RIGHTS_TYPE_LICENSED
			case "collection":
				rightsType = types.RIGHTS_TYPE_COLLECTION
			default:
				rightsType = types.RIGHTS_TYPE_ORIGINAL
			}

			var tags []string
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
			}

			msg := &types.MsgRegisterDataAsset{
				Creator:     clientCtx.GetFromAddress().String(),
				Name:        name,
				ContentHash: contentHash,
				RightsType:  rightsType,
				Description: description,
				Tags:        tags,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Asset description")
	cmd.Flags().String("rights-type", "original", "Rights type: original|co_creation|licensed|collection")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdBuyShares creates a BuyShares transaction.
func CmdBuyShares() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy-shares [asset-id] [amount]",
		Short: "Buy shares of a data asset",
		Long:  "Buy shares of a data asset. Amount format: 1000uoas",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return fmt.Errorf("invalid amount: %w", err)
			}

			msg := &types.MsgBuyShares{
				Creator: clientCtx.GetFromAddress().String(),
				AssetId: args[0],
				Amount:  coin,
			}

			minOutStr, _ := cmd.Flags().GetString("min-shares-out")
			if minOutStr != "" {
				minOut, ok := math.NewIntFromString(minOutStr)
				if ok {
					msg.MinSharesOut = &minOut
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("min-shares-out", "", "Minimum shares expected (slippage protection)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdFileDispute creates a FileDispute transaction.
func CmdFileDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file-dispute [asset-id] [reason]",
		Short: "File a dispute against a data asset",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			remedyStr, _ := cmd.Flags().GetString("remedy")
			var requestedRemedy types.DisputeRemedy
			switch strings.ToLower(remedyStr) {
			case "delist":
				requestedRemedy = types.DISPUTE_REMEDY_DELIST
			case "transfer":
				requestedRemedy = types.DISPUTE_REMEDY_TRANSFER
			case "rights_correction":
				requestedRemedy = types.DISPUTE_REMEDY_RIGHTS_CORRECTION
			case "share_adjustment":
				requestedRemedy = types.DISPUTE_REMEDY_SHARE_ADJUSTMENT
			default:
				requestedRemedy = types.DISPUTE_REMEDY_DELIST
			}

			msg := &types.MsgFileDispute{
				Creator:         clientCtx.GetFromAddress().String(),
				AssetId:         args[0],
				Reason:          args[1],
				RequestedRemedy: requestedRemedy,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("remedy", "delist", "Requested remedy: delist|transfer|rights_correction|share_adjustment")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdResolveDispute creates a ResolveDispute transaction.
func CmdResolveDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-dispute [dispute-id] [remedy]",
		Short: "Resolve a dispute",
		Long:  "Resolve a dispute. Remedy: delist|transfer|rights_correction|share_adjustment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var remedy types.DisputeRemedy
			switch strings.ToLower(args[1]) {
			case "delist":
				remedy = types.DISPUTE_REMEDY_DELIST
			case "transfer":
				remedy = types.DISPUTE_REMEDY_TRANSFER
			case "rights_correction":
				remedy = types.DISPUTE_REMEDY_RIGHTS_CORRECTION
			case "share_adjustment":
				remedy = types.DISPUTE_REMEDY_SHARE_ADJUSTMENT
			default:
				return fmt.Errorf("unknown remedy: %s", args[1])
			}

			msg := &types.MsgResolveDispute{
				Creator:   clientCtx.GetFromAddress().String(),
				DisputeId: args[0],
				Remedy:    remedy,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
