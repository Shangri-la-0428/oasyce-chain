package cli

import (
	"encoding/json"
	"fmt"
	"os"
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
		CmdSellShares(),
		CmdDelistAsset(),
		CmdFileDispute(),
		CmdResolveDispute(),
		CmdInitiateShutdown(),
		CmdClaimSettlement(),
		CmdCreateMigrationPath(),
		CmdDisableMigration(),
		CmdMigrate(),
		CmdUpdateParams(),
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

			parentAssetId, _ := cmd.Flags().GetString("parent")

			msg := &types.MsgRegisterDataAsset{
				Creator:       clientCtx.GetFromAddress().String(),
				Name:          name,
				ContentHash:   contentHash,
				RightsType:    rightsType,
				Description:   description,
				Tags:          tags,
				ParentAssetId: parentAssetId,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Asset description")
	cmd.Flags().String("rights-type", "original", "Rights type: original|co_creation|licensed|collection")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("parent", "", "Parent asset ID (for versioned assets)")
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

// CmdSellShares creates a SellShares transaction.
func CmdSellShares() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell-shares [asset-id] [shares]",
		Short: "Sell shares back to the bonding curve",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			shares, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid shares amount: %s", args[1])
			}

			msg := &types.MsgSellShares{
				Creator: clientCtx.GetFromAddress().String(),
				AssetId: args[0],
				Shares:  shares,
			}

			minOutStr, _ := cmd.Flags().GetString("min-payout-out")
			if minOutStr != "" {
				minOut, ok := math.NewIntFromString(minOutStr)
				if ok {
					msg.MinPayoutOut = &minOut
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("min-payout-out", "", "Minimum payout expected (slippage protection)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdDelistAsset creates a DelistAsset transaction.
func CmdDelistAsset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delist [asset-id]",
		Short: "Delist a data asset (owner only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDelistAsset{
				Creator: clientCtx.GetFromAddress().String(),
				AssetId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

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

// CmdInitiateShutdown creates an InitiateShutdown transaction.
func CmdInitiateShutdown() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate-shutdown [asset-id]",
		Short: "Initiate graceful shutdown of a data asset (owner only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgInitiateShutdown{
				Creator: clientCtx.GetFromAddress().String(),
				AssetId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdClaimSettlement creates a ClaimSettlement transaction.
func CmdClaimSettlement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-settlement [asset-id]",
		Short: "Claim pro-rata settlement payout after shutdown cooldown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimSettlement{
				Creator: clientCtx.GetFromAddress().String(),
				AssetId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCreateMigrationPath creates a CreateMigrationPath transaction.
func CmdCreateMigrationPath() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-migration [source-asset-id] [target-asset-id] [exchange-rate-bps]",
		Short: "Create a migration path from source to target asset",
		Long:  "Create a migration path. exchange-rate-bps: 10000 = 1:1 ratio",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			rateBps, ok := math.NewIntFromString(args[2])
			if !ok || !rateBps.IsPositive() {
				return fmt.Errorf("invalid exchange-rate-bps: %s", args[2])
			}

			maxShares := math.ZeroInt()
			maxStr, _ := cmd.Flags().GetString("max-migrated-shares")
			if maxStr != "" {
				ms, ok := math.NewIntFromString(maxStr)
				if ok {
					maxShares = ms
				}
			}

			msg := &types.MsgCreateMigrationPath{
				Creator:           clientCtx.GetFromAddress().String(),
				SourceAssetId:     args[0],
				TargetAssetId:     args[1],
				ExchangeRateBps:   uint32(rateBps.Int64()),
				MaxMigratedShares: maxShares,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("max-migrated-shares", "", "Maximum shares allowed to migrate (0 = unlimited)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdDisableMigration creates a DisableMigration transaction.
func CmdDisableMigration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-migration [source-asset-id] [target-asset-id]",
		Short: "Disable a migration path (target asset owner only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDisableMigration{
				Creator:       clientCtx.GetFromAddress().String(),
				SourceAssetId: args[0],
				TargetAssetId: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdMigrate creates a Migrate transaction.
func CmdMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [source-asset-id] [target-asset-id] [shares]",
		Short: "Migrate shares from source to target asset",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			shares, ok := math.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid shares amount: %s", args[2])
			}

			msg := &types.MsgMigrate{
				Creator:       clientCtx.GetFromAddress().String(),
				SourceAssetId: args[0],
				TargetAssetId: args[1],
				Shares:        shares,
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
		Long:  "Submit a transaction to update datarights module parameters. The params-json-file should contain the full Params JSON.",
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
