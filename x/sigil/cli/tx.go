package cli

import (
	"encoding/hex"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/oasyce/chain/x/sigil/types"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Sigil lifecycle transactions",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdGenesis(),
		CmdDissolve(),
		CmdBond(),
		CmdUnbond(),
		CmdFork(),
		CmdMerge(),
		CmdPulse(),
	)

	return txCmd
}

func CmdGenesis() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis [public-key-hex]",
		Short: "Create a new Sigil",
		Long:  "Create a new Sigil from a public key. Optional flags: --lineage, --state-root, --metadata.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubKey, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}

			lineageStr, _ := cmd.Flags().GetString("lineage")
			var lineage []string
			if lineageStr != "" {
				lineage = strings.Split(lineageStr, ",")
			}

			stateRootHex, _ := cmd.Flags().GetString("state-root")
			var stateRoot []byte
			if stateRootHex != "" {
				stateRoot, err = hex.DecodeString(stateRootHex)
				if err != nil {
					return err
				}
			}

			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgGenesis{
				Signer:    clientCtx.GetFromAddress().String(),
				PublicKey: pubKey,
				Lineage:   lineage,
				StateRoot: stateRoot,
				Metadata:  metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("lineage", "", "Comma-separated parent Sigil IDs")
	cmd.Flags().String("state-root", "", "Initial state root (hex)")
	cmd.Flags().String("metadata", "", "Metadata string")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdDissolve() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dissolve [sigil-id]",
		Short: "Dissolve a Sigil permanently",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDissolve{
				Signer:  clientCtx.GetFromAddress().String(),
				SigilId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdBond() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bond [sigil-a] [sigil-b]",
		Short: "Create a bond between two Sigils",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			termsHashHex, _ := cmd.Flags().GetString("terms-hash")
			var termsHash []byte
			if termsHashHex != "" {
				termsHash, err = hex.DecodeString(termsHashHex)
				if err != nil {
					return err
				}
			}

			scope, _ := cmd.Flags().GetString("scope")

			msg := &types.MsgBond{
				Signer:    clientCtx.GetFromAddress().String(),
				SigilA:    args[0],
				SigilB:    args[1],
				TermsHash: termsHash,
				Scope:     scope,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("terms-hash", "", "Bond terms hash (hex)")
	cmd.Flags().String("scope", "", "Bond scope")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdUnbond() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbond [bond-id]",
		Short: "Remove a bond",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnbond{
				Signer: clientCtx.GetFromAddress().String(),
				BondId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFork() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork [parent-sigil-id] [child-public-key-hex]",
		Short: "Fork a new Sigil from an existing parent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubKey, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			forkMode, _ := cmd.Flags().GetInt32("fork-mode")
			mutation, _ := cmd.Flags().GetString("mutation")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgFork{
				Signer:        clientCtx.GetFromAddress().String(),
				ParentSigilId: args[0],
				PublicKey:     pubKey,
				ForkMode:      forkMode,
				Mutation:      mutation,
				Metadata:      metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Int32("fork-mode", 0, "Fork mode (0=symmetric, 1=asymmetric)")
	cmd.Flags().String("mutation", "", "Mutation parameter")
	cmd.Flags().String("metadata", "", "Metadata string")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdMerge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge [sigil-a] [sigil-b]",
		Short: "Merge two Sigils",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			mergeMode, _ := cmd.Flags().GetInt32("merge-mode")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgMerge{
				Signer:    clientCtx.GetFromAddress().String(),
				SigilA:    args[0],
				SigilB:    args[1],
				MergeMode: mergeMode,
				Metadata:  metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Int32("merge-mode", 0, "Merge mode (0=symmetric, 1=absorption)")
	cmd.Flags().String("metadata", "", "Metadata string")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdPulse() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pulse [sigil-id]",
		Short: "Send multi-dimensional heartbeat pulse for a Sigil",
		Long:  "Record a multi-dimensional heartbeat for a Sigil. Dimensions keep the Sigil alive on-chain.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			dimsStr, _ := cmd.Flags().GetString("dimensions")
			dims := make(map[string]int64)
			if dimsStr != "" {
				for _, d := range strings.Split(dimsStr, ",") {
					d = strings.TrimSpace(d)
					if d != "" {
						dims[d] = 1
					}
				}
			}
			if len(dims) == 0 {
				dims["chain"] = 1
			}

			msg := &types.MsgPulse{
				Signer:     clientCtx.GetFromAddress().String(),
				SigilId:    args[0],
				Dimensions: dims,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("dimensions", "chain", "Comma-separated dimension names (e.g. thronglets,psyche)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
