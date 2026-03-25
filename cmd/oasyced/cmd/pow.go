package cmd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
)

// UtilCmd returns the parent "util" command for chain utilities.
func UtilCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "util",
		Short: "Chain utility commands for agents and operators",
	}
	cmd.AddCommand(
		SolvePowCmd(),
	)
	return cmd
}

// SolvePowCmd returns a command that solves a proof-of-work puzzle for onboarding.
// This enables AI agents to self-register without external PoW solvers.
func SolvePowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "solve-pow [address]",
		Short: "Solve a proof-of-work puzzle for onboarding registration",
		Long: `Brute-force search for a nonce such that sha256(address || nonce_le_bytes)
has at least --difficulty leading zero bits. Used for permissionless self-registration.

Example:
  oasyced util solve-pow oasyce1abc... --difficulty 16
  oasyced util solve-pow oasyce1abc... --difficulty 16 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]

			difficulty, _ := cmd.Flags().GetUint32("difficulty")
			if difficulty == 0 {
				difficulty = 16 // default matches onboarding module default
			}

			outputJSON, _ := cmd.Flags().GetString("output")

			start := time.Now()

			// Randomize start to allow parallel solvers on the same address.
			nonce := rand.Uint64()
			var hash [32]byte
			var attempts uint64

			for {
				data := make([]byte, len(address)+8)
				copy(data, address)
				binary.LittleEndian.PutUint64(data[len(address):], nonce)
				hash = sha256.Sum256(data)

				if leadingZeroBits(hash[:]) >= int(difficulty) {
					break
				}
				nonce++
				attempts++

				// Progress every 1M attempts (only for non-json output).
				if attempts%1_000_000 == 0 && outputJSON != "json" {
					fmt.Fprintf(cmd.ErrOrStderr(), "\r  searching... %d attempts (%.1fs)", attempts, time.Since(start).Seconds())
				}
			}

			elapsed := time.Since(start)

			if outputJSON == "json" {
				result := map[string]interface{}{
					"address":    address,
					"nonce":      nonce,
					"difficulty": difficulty,
					"hash":       hex.EncodeToString(hash[:]),
					"attempts":   attempts,
					"elapsed_ms": elapsed.Milliseconds(),
				}
				bz, _ := json.MarshalIndent(result, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(bz))
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "\r") // clear progress line
				fmt.Fprintf(cmd.OutOrStdout(), "Solved! nonce=%d difficulty=%d attempts=%d time=%v\nhash=%s\n",
					nonce, difficulty, attempts, elapsed.Round(time.Millisecond), hex.EncodeToString(hash[:]))
				fmt.Fprintf(cmd.OutOrStdout(), "\nTo register:\n  oasyced tx onboarding register %d --from <key> --chain-id <chain-id> --yes\n", nonce)
			}

			return nil
		},
	}

	cmd.Flags().Uint32("difficulty", 0, "Number of leading zero bits required (default: 16, or queried from chain)")
	cmd.Flags().String("output", "text", "Output format: text or json")
	return cmd
}

// leadingZeroBits counts leading zero bits in a byte slice.
// Identical to x/onboarding/keeper/keeper.go:LeadingZeroBits.
func leadingZeroBits(b []byte) int {
	total := 0
	for _, v := range b {
		if v == 0 {
			total += 8
		} else {
			total += bits.LeadingZeros8(v)
			break
		}
	}
	return total
}
