package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgRegisterDataAsset registers a new data asset on-chain.
type MsgRegisterDataAsset struct {
	Creator     string      `json:"creator"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ContentHash string      `json:"content_hash"`
	RightsType  RightsType  `json:"rights_type"`
	Tags        []string    `json:"tags"`
	CoCreators  []CoCreator `json:"co_creators"`
}

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
	if msg.RightsType < RightsOriginal || msg.RightsType > RightsCollection {
		return ErrInvalidRightsType.Wrapf("invalid rights_type: %d", msg.RightsType)
	}
	if len(msg.CoCreators) > 0 {
		if err := ValidateCoCreators(msg.CoCreators); err != nil {
			return err
		}
	}
	return nil
}

func (msg MsgRegisterDataAsset) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// MsgRegisterDataAssetResponse is the response for MsgRegisterDataAsset.
type MsgRegisterDataAssetResponse struct {
	AssetID string `json:"asset_id"`
}

// MsgBuyShares purchases shares of a data asset via bonding curve.
type MsgBuyShares struct {
	Creator string   `json:"creator"`
	AssetID string   `json:"asset_id"`
	Amount  sdk.Coin `json:"amount"`
}

func (msg MsgBuyShares) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetID == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return ErrInsufficientFunds.Wrap("amount must be positive")
	}
	return nil
}

func (msg MsgBuyShares) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// MsgBuySharesResponse is the response for MsgBuyShares.
type MsgBuySharesResponse struct {
	SharesPurchased string `json:"shares_purchased"`
}

// MsgFileDispute files a dispute against a data asset.
type MsgFileDispute struct {
	Creator  string `json:"creator"`
	AssetID  string `json:"asset_id"`
	Reason   string `json:"reason"`
	Evidence []byte `json:"evidence"`
}

func (msg MsgFileDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.AssetID == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if msg.Reason == "" {
		return ErrInvalidParams.Wrap("reason must not be empty")
	}
	return nil
}

func (msg MsgFileDispute) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// MsgFileDisputeResponse is the response for MsgFileDispute.
type MsgFileDisputeResponse struct {
	DisputeID string `json:"dispute_id"`
}

// MsgResolveDispute resolves an open dispute.
type MsgResolveDispute struct {
	Creator   string        `json:"creator"` // arbitrator
	DisputeID string        `json:"dispute_id"`
	Remedy    DisputeRemedy `json:"remedy"`
	Details   []byte        `json:"details"`
}

func (msg MsgResolveDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if msg.DisputeID == "" {
		return ErrInvalidParams.Wrap("dispute_id must not be empty")
	}
	if msg.Remedy < RemedyDelist || msg.Remedy > RemedyShareAdjustment {
		return ErrInvalidParams.Wrapf("invalid remedy: %d", msg.Remedy)
	}
	return nil
}

func (msg MsgResolveDispute) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// MsgResolveDisputeResponse is the response for MsgResolveDispute.
type MsgResolveDisputeResponse struct{}
