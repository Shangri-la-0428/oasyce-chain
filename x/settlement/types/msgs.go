package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure proto-generated message types implement sdk.Msg.
var (
	_ sdk.Msg = &MsgCreateEscrow{}
	_ sdk.Msg = &MsgReleaseEscrow{}
	_ sdk.Msg = &MsgRefundEscrow{}
)

// ValidateBasic performs stateless validation for MsgCreateEscrow.
func (msg *MsgCreateEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return ErrInsufficientFunds.Wrap("amount must be positive and valid")
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgReleaseEscrow.
func (msg *MsgReleaseEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.EscrowId == "" {
		return ErrEscrowNotFound.Wrap("escrow_id cannot be empty")
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgRefundEscrow.
func (msg *MsgRefundEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.EscrowId == "" {
		return ErrEscrowNotFound.Wrap("escrow_id cannot be empty")
	}
	return nil
}
