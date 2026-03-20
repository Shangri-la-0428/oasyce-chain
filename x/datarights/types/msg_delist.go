package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic performs basic validation on MsgDelistAsset.
func (m *MsgDelistAsset) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if m.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	return nil
}

// GetSigners returns the expected signers for MsgDelistAsset.
func (m *MsgDelistAsset) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}
