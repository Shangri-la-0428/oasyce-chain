package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/oasyce/chain/x/anchor/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Anchor transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdAnchorTrace(),
	)
	return cmd
}

// CmdAnchorTrace creates a CLI command for anchoring a single trace.
func CmdAnchorTrace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anchor-trace [trace-id-hex] [node-pubkey-hex] [capability] [outcome] [timestamp-ms] [signature-hex]",
		Short: "Anchor a trace record on-chain",
		Long: `Anchor a trace record on-chain. All byte fields are hex-encoded.

Arguments:
  trace-id-hex     - Hex-encoded trace ID (up to 64 bytes)
  node-pubkey-hex  - Hex-encoded ed25519 public key (32 bytes)
  capability       - Capability identifier string
  outcome          - Numeric outcome code
  timestamp-ms     - Trace creation time in unix milliseconds
  signature-hex    - Hex-encoded ed25519 signature (64 bytes)`,
		Args: cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			traceID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid trace_id hex: %w", err)
			}

			nodePubkey, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid node_pubkey hex: %w", err)
			}

			capability := args[2]

			outcome, err := strconv.ParseUint(args[3], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid outcome: %w", err)
			}

			timestamp, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp: %w", err)
			}

			signature, err := hex.DecodeString(args[5])
			if err != nil {
				return fmt.Errorf("invalid trace_signature hex: %w", err)
			}

			sigilID, _ := cmd.Flags().GetString("sigil-id")

			msg := &types.MsgAnchorTrace{
				Signer:         clientCtx.GetFromAddress().String(),
				TraceId:        traceID,
				NodePubkey:     nodePubkey,
				Capability:     capability,
				Outcome:        uint32(outcome),
				Timestamp:      timestamp,
				TraceSignature: signature,
				SigilId:        sigilID,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("sigil-id", "", "Optional Sigil ID to associate with this trace")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
