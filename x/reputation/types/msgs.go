package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure proto-generated message types implement sdk.Msg.
var (
	_ sdk.Msg = &MsgSubmitFeedback{}
	_ sdk.Msg = &MsgReportMisbehavior{}
)

// ValidateBasic performs stateless validation for MsgSubmitFeedback.
func (msg *MsgSubmitFeedback) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.InvocationId == "" {
		return ErrInvocationNotFound.Wrap("invocation_id cannot be empty")
	}
	if msg.Rating > 500 {
		return ErrInvalidRating.Wrapf("rating must be 0-500, got %d", msg.Rating)
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgReportMisbehavior.
func (msg *MsgReportMisbehavior) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Target); err != nil {
		return ErrInvalidAddress.Wrapf("invalid target address: %s", err)
	}
	if msg.EvidenceType == "" {
		return ErrInvalidAddress.Wrap("evidence_type cannot be empty")
	}
	if len(msg.Evidence) == 0 {
		return ErrInvalidAddress.Wrap("evidence cannot be empty")
	}
	return nil
}
