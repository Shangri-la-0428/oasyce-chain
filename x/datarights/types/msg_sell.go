package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic performs basic validation on MsgSellShares.
func (m *MsgSellShares) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if m.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if m.Shares.IsNil() || m.Shares.IsZero() || m.Shares.IsNegative() {
		return ErrInsufficientFunds.Wrap("shares must be positive")
	}
	return nil
}

// GetSigners returns the expected signers for MsgSellShares.
func (m *MsgSellShares) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}
