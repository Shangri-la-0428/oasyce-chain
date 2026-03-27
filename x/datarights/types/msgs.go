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

// ValidateBasic for MsgInitiateShutdown.
func (msg MsgInitiateShutdown) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	return nil
}

// ValidateBasic for MsgClaimSettlement.
func (msg MsgClaimSettlement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	return nil
}

// ValidateBasic for MsgCreateMigrationPath.
func (msg MsgCreateMigrationPath) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.SourceAssetId == "" {
		return ErrInvalidParams.Wrap("source_asset_id must not be empty")
	}
	if msg.TargetAssetId == "" {
		return ErrInvalidParams.Wrap("target_asset_id must not be empty")
	}
	if msg.SourceAssetId == msg.TargetAssetId {
		return ErrInvalidParams.Wrap("source and target asset must be different")
	}
	if msg.ExchangeRateBps == 0 {
		return ErrInvalidParams.Wrap("exchange_rate_bps must be > 0")
	}
	return nil
}

// ValidateBasic for MsgDisableMigration.
func (msg MsgDisableMigration) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.SourceAssetId == "" {
		return ErrInvalidParams.Wrap("source_asset_id must not be empty")
	}
	if msg.TargetAssetId == "" {
		return ErrInvalidParams.Wrap("target_asset_id must not be empty")
	}
	return nil
}

// ValidateBasic for MsgMigrate.
func (msg MsgMigrate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.SourceAssetId == "" {
		return ErrInvalidParams.Wrap("source_asset_id must not be empty")
	}
	if msg.TargetAssetId == "" {
		return ErrInvalidParams.Wrap("target_asset_id must not be empty")
	}
	if !msg.Shares.IsPositive() {
		return ErrInvalidParams.Wrap("shares must be positive")
	}
	return nil
}

// ValidateBasic for MsgUpdateServiceUrl.
func (msg MsgUpdateServiceUrl) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	// service_url can be empty (to clear it)
	return nil
}

// GetSigners returns the expected signers for MsgUpdateServiceUrl.
// Required because Descriptor() returns nil — the SDK cannot extract the signer
// from the proto annotation without a valid file descriptor reference.
func (msg *MsgUpdateServiceUrl) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}
