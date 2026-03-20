package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var (
	_ sdk.Msg = &MsgRegisterExecutor{}
	_ sdk.Msg = &MsgUpdateExecutor{}
	_ sdk.Msg = &MsgSubmitTask{}
	_ sdk.Msg = &MsgCommitResult{}
	_ sdk.Msg = &MsgRevealResult{}
	_ sdk.Msg = &MsgDisputeResult{}
)

func (msg *MsgRegisterExecutor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Executor); err != nil {
		return ErrInvalidAddress.Wrapf("invalid executor address: %s", err)
	}
	if len(msg.SupportedTaskTypes) == 0 {
		return ErrInvalidTaskType.Wrap("must declare at least one supported task type")
	}
	if msg.MaxComputeUnits == 0 {
		return ErrInvalidParams.Wrap("max_compute_units must be > 0")
	}
	return nil
}

func (msg *MsgUpdateExecutor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Executor); err != nil {
		return ErrInvalidAddress.Wrapf("invalid executor address: %s", err)
	}
	return nil
}

func (msg *MsgSubmitTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.TaskType == "" {
		return ErrInvalidTaskType.Wrap("task_type cannot be empty")
	}
	if len(msg.InputHash) != 32 {
		return ErrInvalidInputHash
	}
	if !msg.Bounty.IsValid() || msg.Bounty.IsZero() {
		return ErrInvalidBounty.Wrap("bounty must be positive and valid")
	}
	return nil
}

func (msg *MsgCommitResult) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Executor); err != nil {
		return ErrInvalidAddress.Wrapf("invalid executor address: %s", err)
	}
	if len(msg.CommitHash) != 32 {
		return ErrInvalidInputHash.Wrap("commit_hash must be 32 bytes (SHA-256)")
	}
	return nil
}

func (msg *MsgRevealResult) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Executor); err != nil {
		return ErrInvalidAddress.Wrapf("invalid executor address: %s", err)
	}
	if !msg.Unavailable {
		if len(msg.OutputHash) != 32 {
			return ErrInvalidInputHash.Wrap("output_hash must be 32 bytes (SHA-256)")
		}
		if len(msg.Salt) == 0 {
			return ErrInvalidParams.Wrap("salt cannot be empty")
		}
	}
	return nil
}

func (msg *MsgDisputeResult) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return ErrInvalidAddress.Wrapf("invalid challenger address: %s", err)
	}
	if msg.Reason == "" {
		return ErrInvalidParams.Wrap("dispute reason cannot be empty")
	}
	if !msg.Bond.IsValid() || msg.Bond.IsZero() {
		return ErrDisputeBondTooLow.Wrap("bond must be positive")
	}
	return nil
}
