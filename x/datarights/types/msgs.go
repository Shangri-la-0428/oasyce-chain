package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic for MsgRegisterDataAsset.
func (msg MsgRegisterDataAsset) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.Name == "" {
		return ErrInvalidParams.Wrap("name must not be empty")
	}
	if msg.ContentHash == "" {
		return ErrInvalidParams.Wrap("content_hash must not be empty")
	}
	if msg.RightsType < RIGHTS_TYPE_ORIGINAL || msg.RightsType > RIGHTS_TYPE_COLLECTION {
		return ErrInvalidRightsType.Wrapf("invalid rights_type: %d", msg.RightsType)
	}
	if len(msg.CoCreators) > 0 {
		if err := ValidateCoCreators(msg.CoCreators); err != nil {
			return err
		}
	}
	return nil
}

// ValidateBasic for MsgBuyShares.
func (msg MsgBuyShares) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return ErrInsufficientFunds.Wrap("amount must be positive")
	}
	return nil
}

// ValidateBasic for MsgFileDispute.
func (msg MsgFileDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if msg.Reason == "" {
		return ErrInvalidParams.Wrap("reason must not be empty")
	}
	return nil
}

// ValidateBasic for MsgResolveDispute.
func (msg MsgResolveDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.DisputeId == "" {
		return ErrInvalidParams.Wrap("dispute_id must not be empty")
	}
	if msg.Remedy < DISPUTE_REMEDY_DELIST || msg.Remedy > DISPUTE_REMEDY_SHARE_ADJUSTMENT {
		return ErrInvalidParams.Wrapf("invalid remedy: %d", msg.Remedy)
	}
	return nil
}
