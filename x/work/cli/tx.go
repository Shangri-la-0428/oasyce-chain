package cli

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Work module transaction commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdRegisterExecutor(),
		CmdUpdateExecutor(),
		CmdSubmitTask(),
		CmdCommitResult(),
		CmdRevealResult(),
		CmdDisputeResult(),
	)

	return txCmd
}

func CmdRegisterExecutor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-executor [task-types] [max-compute-units]",
		Short: "Register as a compute executor",
		Long:  "Register as a compute executor. task-types is a comma-separated list (e.g. inference,training)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			taskTypes := strings.Split(args[0], ",")
			maxCU, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterExecutor{
				Executor:           clientCtx.GetFromAddress().String(),
				SupportedTaskTypes: taskTypes,
				MaxComputeUnits:    maxCU,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUpdateExecutor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-executor [task-types] [max-compute-units] [active]",
		Short: "Update executor profile",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			taskTypes := strings.Split(args[0], ",")
			maxCU, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			active, err := strconv.ParseBool(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateExecutor{
				Executor:           clientCtx.GetFromAddress().String(),
				SupportedTaskTypes: taskTypes,
				MaxComputeUnits:    maxCU,
				Active:             active,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSubmitTask() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-task [task-type] [input-hash-hex] [input-uri] [max-compute-units] [bounty]",
		Short: "Submit a compute task with bounty",
		Long:  "Submit a compute task. bounty format: 1000uoas. Optional flags: --redundancy, --timeout",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			inputHash, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			maxCU, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}

			bounty, err := sdk.ParseCoinNormalized(args[4])
			if err != nil {
				return err
			}

			redundancy, _ := cmd.Flags().GetUint32("redundancy")
			timeout, _ := cmd.Flags().GetUint64("timeout")

			msg := &types.MsgSubmitTask{
				Creator:         clientCtx.GetFromAddress().String(),
				TaskType:        args[0],
				InputHash:       inputHash,
				InputUri:        args[2],
				MaxComputeUnits: maxCU,
				Bounty:          bounty,
				Redundancy:      redundancy,
				TimeoutBlocks:   timeout,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint32("redundancy", 0, "Number of redundant executors (0 = use default)")
	cmd.Flags().Uint64("timeout", 0, "Timeout in blocks (0 = use default)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdCommitResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-result [task-id] [commit-hash-hex]",
		Short: "Submit a sealed result hash (commit phase)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			taskID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			commitHash, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgCommitResult{
				Executor:   clientCtx.GetFromAddress().String(),
				TaskId:     taskID,
				CommitHash: commitHash,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRevealResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-result [task-id] [output-hash-hex] [output-uri] [compute-units] [salt-hex]",
		Short: "Reveal the actual result (reveal phase)",
		Long:  "Reveal the compute result. Use --unavailable flag if input data was not reachable.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			taskID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			unavailable, _ := cmd.Flags().GetBool("unavailable")

			msg := &types.MsgRevealResult{
				Executor:    clientCtx.GetFromAddress().String(),
				TaskId:      taskID,
				Unavailable: unavailable,
			}

			if !unavailable {
				if len(args) < 5 {
					return types.ErrInvalidParams.Wrap("need 5 args: task-id output-hash output-uri compute-units salt")
				}
				outputHash, err := hex.DecodeString(args[1])
				if err != nil {
					return err
				}
				cu, err := strconv.ParseUint(args[3], 10, 64)
				if err != nil {
					return err
				}
				salt, err := hex.DecodeString(args[4])
				if err != nil {
					return err
				}

				msg.OutputHash = outputHash
				msg.OutputUri = args[2]
				msg.ComputeUnitsUsed = cu
				msg.Salt = salt
			} else {
				if len(args) >= 3 {
					salt, err := hex.DecodeString(args[1])
					if err == nil {
						msg.Salt = salt
					}
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Bool("unavailable", false, "Report input data as unavailable")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDisputeResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute-result [task-id] [reason] [bond]",
		Short: "Dispute a settled task's result",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			taskID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			bond, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgDisputeResult{
				Challenger: clientCtx.GetFromAddress().String(),
				TaskId:     taskID,
				Reason:     args[1],
				Bond:       bond,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
