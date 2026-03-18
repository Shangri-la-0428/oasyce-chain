package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure our message types implement sdk.Msg.
// In Cosmos SDK v0.50, sdk.Msg embeds proto.Message.
// We implement the proto.Message interface manually since we don't have codegen.

var (
	_ sdk.Msg = &MsgCreateEscrow{}
	_ sdk.Msg = &MsgReleaseEscrow{}
	_ sdk.Msg = &MsgRefundEscrow{}
)

// ---------------------------------------------------------------------------
// MsgCreateEscrow
// ---------------------------------------------------------------------------

// MsgCreateEscrow locks funds for a capability invocation or data purchase.
type MsgCreateEscrow struct {
	Creator      string   `json:"creator"`
	Provider     string   `json:"provider"`
	CapabilityID string   `json:"capability_id,omitempty"`
	AssetID      string   `json:"asset_id,omitempty"`
	Amount       sdk.Coin `json:"amount"`
}

func NewMsgCreateEscrow(creator, provider string, amount sdk.Coin) *MsgCreateEscrow {
	return &MsgCreateEscrow{
		Creator:  creator,
		Provider: provider,
		Amount:   amount,
	}
}

func (msg *MsgCreateEscrow) ProtoMessage()             {}
func (msg *MsgCreateEscrow) Reset()                    { *msg = MsgCreateEscrow{} }
func (msg *MsgCreateEscrow) String() string            { return "MsgCreateEscrow" }

func (msg *MsgCreateEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}
	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return ErrInsufficientFunds.Wrap("amount must be positive and valid")
	}
	return nil
}

func (msg *MsgCreateEscrow) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// ---------------------------------------------------------------------------
// MsgReleaseEscrow
// ---------------------------------------------------------------------------

// MsgReleaseEscrow releases escrowed funds to the provider.
type MsgReleaseEscrow struct {
	Creator  string `json:"creator"`
	EscrowID string `json:"escrow_id"`
}

func NewMsgReleaseEscrow(creator, escrowID string) *MsgReleaseEscrow {
	return &MsgReleaseEscrow{
		Creator:  creator,
		EscrowID: escrowID,
	}
}

func (msg *MsgReleaseEscrow) ProtoMessage()             {}
func (msg *MsgReleaseEscrow) Reset()                    { *msg = MsgReleaseEscrow{} }
func (msg *MsgReleaseEscrow) String() string            { return "MsgReleaseEscrow" }

func (msg *MsgReleaseEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.EscrowID == "" {
		return ErrEscrowNotFound.Wrap("escrow_id cannot be empty")
	}
	return nil
}

func (msg *MsgReleaseEscrow) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// ---------------------------------------------------------------------------
// MsgRefundEscrow
// ---------------------------------------------------------------------------

// MsgRefundEscrow refunds escrowed funds to the consumer.
type MsgRefundEscrow struct {
	Creator  string `json:"creator"`
	EscrowID string `json:"escrow_id"`
}

func NewMsgRefundEscrow(creator, escrowID string) *MsgRefundEscrow {
	return &MsgRefundEscrow{
		Creator:  creator,
		EscrowID: escrowID,
	}
}

func (msg *MsgRefundEscrow) ProtoMessage()             {}
func (msg *MsgRefundEscrow) Reset()                    { *msg = MsgRefundEscrow{} }
func (msg *MsgRefundEscrow) String() string            { return "MsgRefundEscrow" }

func (msg *MsgRefundEscrow) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.EscrowID == "" {
		return ErrEscrowNotFound.Wrap("escrow_id cannot be empty")
	}
	return nil
}

func (msg *MsgRefundEscrow) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}
