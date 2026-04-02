package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic for MsgSetPolicy.
func (msg MsgSetPolicy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Principal); err != nil {
		return ErrInvalidAddress.Wrapf("invalid principal: %s", err)
	}
	if !msg.PerTxLimit.IsValid() || msg.PerTxLimit.IsZero() {
		return ErrInvalidPolicy.Wrap("per_tx_limit must be a positive valid coin")
	}
	if !msg.WindowLimit.IsValid() || msg.WindowLimit.IsZero() {
		return ErrInvalidPolicy.Wrap("window_limit must be a positive valid coin")
	}
	if msg.WindowSeconds == 0 {
		return ErrInvalidPolicy.Wrap("window_seconds must be > 0")
	}
	if len(msg.AllowedMsgs) == 0 {
		return ErrInvalidPolicy.Wrap("allowed_msgs must not be empty")
	}
	if len(msg.EnrollmentToken) == 0 {
		return ErrInvalidPolicy.Wrap("enrollment_token required (used as shared secret for agent enrollment)")
	}
	return nil
}

// ValidateBasic for MsgEnroll.
func (msg MsgEnroll) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return ErrInvalidAddress.Wrapf("invalid delegate: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Principal); err != nil {
		return ErrInvalidAddress.Wrapf("invalid principal: %s", err)
	}
	if msg.Delegate == msg.Principal {
		return ErrSelfDelegate.Wrap("principal cannot enroll as own delegate")
	}
	return nil
}

// ValidateBasic for MsgRevoke.
func (msg MsgRevoke) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Principal); err != nil {
		return ErrInvalidAddress.Wrapf("invalid principal: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return ErrInvalidAddress.Wrapf("invalid delegate: %s", err)
	}
	return nil
}

// ValidateBasic for MsgExec.
func (msg MsgExec) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return ErrInvalidAddress.Wrapf("invalid delegate: %s", err)
	}
	if len(msg.Msgs) == 0 {
		return ErrInvalidPolicy.Wrap("msgs must not be empty")
	}
	return nil
}
