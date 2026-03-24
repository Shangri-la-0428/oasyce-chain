package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (msg MsgSelfRegister) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	return nil
}

func (msg MsgRepayDebt) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if !msg.Amount.IsPositive() {
		return ErrInvalidParams.Wrap("amount must be positive")
	}
	return nil
}
