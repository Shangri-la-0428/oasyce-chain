package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure our message types implement sdk.Msg.
var (
	_ sdk.Msg = &MsgSubmitFeedback{}
	_ sdk.Msg = &MsgReportMisbehavior{}
)

// ---------------------------------------------------------------------------
// MsgSubmitFeedback
// ---------------------------------------------------------------------------

// MsgSubmitFeedback submits a rating for a completed invocation.
type MsgSubmitFeedback struct {
	Creator      string `json:"creator"`
	InvocationID string `json:"invocation_id"`
	Rating       uint32 `json:"rating"` // 0-500
	Comment      string `json:"comment"`
}

func NewMsgSubmitFeedback(creator, invocationID string, rating uint32, comment string) *MsgSubmitFeedback {
	return &MsgSubmitFeedback{
		Creator:      creator,
		InvocationID: invocationID,
		Rating:       rating,
		Comment:      comment,
	}
}

func (msg *MsgSubmitFeedback) ProtoMessage()  {}
func (msg *MsgSubmitFeedback) Reset()         { *msg = MsgSubmitFeedback{} }
func (msg *MsgSubmitFeedback) String() string { return "MsgSubmitFeedback" }

func (msg *MsgSubmitFeedback) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.InvocationID == "" {
		return ErrInvocationNotFound.Wrap("invocation_id cannot be empty")
	}
	if msg.Rating > 500 {
		return ErrInvalidRating.Wrapf("rating must be 0-500, got %d", msg.Rating)
	}
	return nil
}

func (msg *MsgSubmitFeedback) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// ---------------------------------------------------------------------------
// MsgReportMisbehavior
// ---------------------------------------------------------------------------

// MsgReportMisbehavior reports bad behavior by a participant.
type MsgReportMisbehavior struct {
	Creator      string `json:"creator"`
	Target       string `json:"target"`
	EvidenceType string `json:"evidence_type"`
	Evidence     []byte `json:"evidence"`
}

func NewMsgReportMisbehavior(creator, target, evidenceType string, evidence []byte) *MsgReportMisbehavior {
	return &MsgReportMisbehavior{
		Creator:      creator,
		Target:       target,
		EvidenceType: evidenceType,
		Evidence:     evidence,
	}
}

func (msg *MsgReportMisbehavior) ProtoMessage()  {}
func (msg *MsgReportMisbehavior) Reset()         { *msg = MsgReportMisbehavior{} }
func (msg *MsgReportMisbehavior) String() string { return "MsgReportMisbehavior" }

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

func (msg *MsgReportMisbehavior) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// MsgSubmitFeedbackResponse is the response for MsgSubmitFeedback.
type MsgSubmitFeedbackResponse struct {
	FeedbackID string `json:"feedback_id"`
}

// MsgReportMisbehaviorResponse is the response for MsgReportMisbehavior.
type MsgReportMisbehaviorResponse struct {
	ReportID string `json:"report_id"`
}
