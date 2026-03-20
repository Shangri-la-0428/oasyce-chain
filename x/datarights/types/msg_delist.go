package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgDelistAsset allows an asset owner to voluntarily delist their asset.
type MsgDelistAsset struct {
	// Creator is the address of the asset owner.
	Creator string `json:"creator"`
	// AssetId is the ID of the data asset to delist.
	AssetId string `json:"asset_id"`
}

// MsgDelistAssetResponse is the response for MsgDelistAsset.
type MsgDelistAssetResponse struct{}

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
