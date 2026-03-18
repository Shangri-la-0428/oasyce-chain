package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- MsgRegisterCapability ---

var _ sdk.Msg = &MsgRegisterCapability{}

type MsgRegisterCapability struct {
	Creator      string   `json:"creator"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	EndpointURL  string   `json:"endpoint_url"`
	PricePerCall sdk.Coin `json:"price_per_call"`
	Tags         []string `json:"tags"`
	RateLimit    uint64   `json:"rate_limit"`
}

func (*MsgRegisterCapability) ProtoMessage()             {}
func (*MsgRegisterCapability) Reset()                    {}
func (msg *MsgRegisterCapability) String() string        { return "MsgRegisterCapability" }

func (msg *MsgRegisterCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.Name == "" {
		return ErrInvalidInput.Wrap("name cannot be empty")
	}
	if msg.EndpointURL == "" {
		return ErrInvalidInput.Wrap("endpoint_url cannot be empty")
	}
	if !msg.PricePerCall.IsValid() {
		return ErrInvalidInput.Wrap("invalid price_per_call")
	}
	return nil
}

func (msg *MsgRegisterCapability) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// --- MsgInvokeCapability ---

var _ sdk.Msg = &MsgInvokeCapability{}

type MsgInvokeCapability struct {
	Creator      string `json:"creator"`
	CapabilityID string `json:"capability_id"`
	Input        []byte `json:"input"`
}

func (*MsgInvokeCapability) ProtoMessage()             {}
func (*MsgInvokeCapability) Reset()                    {}
func (msg *MsgInvokeCapability) String() string        { return "MsgInvokeCapability" }

func (msg *MsgInvokeCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityID == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	return nil
}

func (msg *MsgInvokeCapability) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// --- MsgUpdateCapability ---

var _ sdk.Msg = &MsgUpdateCapability{}

type MsgUpdateCapability struct {
	Creator      string    `json:"creator"`
	CapabilityID string    `json:"capability_id"`
	EndpointURL  string    `json:"endpoint_url,omitempty"`
	PricePerCall *sdk.Coin `json:"price_per_call,omitempty"`
	RateLimit    uint64    `json:"rate_limit,omitempty"`
	Description  string    `json:"description,omitempty"`
}

func (*MsgUpdateCapability) ProtoMessage()             {}
func (*MsgUpdateCapability) Reset()                    {}
func (msg *MsgUpdateCapability) String() string        { return "MsgUpdateCapability" }

func (msg *MsgUpdateCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityID == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	if msg.PricePerCall != nil && !msg.PricePerCall.IsValid() {
		return ErrInvalidInput.Wrap("invalid price_per_call")
	}
	return nil
}

func (msg *MsgUpdateCapability) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// --- MsgDeactivateCapability ---

var _ sdk.Msg = &MsgDeactivateCapability{}

type MsgDeactivateCapability struct {
	Creator      string `json:"creator"`
	CapabilityID string `json:"capability_id"`
}

func (*MsgDeactivateCapability) ProtoMessage()             {}
func (*MsgDeactivateCapability) Reset()                    {}
func (msg *MsgDeactivateCapability) String() string        { return "MsgDeactivateCapability" }

func (msg *MsgDeactivateCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityID == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	return nil
}

func (msg *MsgDeactivateCapability) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

// --- Response types ---

type MsgRegisterCapabilityResponse struct {
	CapabilityID string `json:"capability_id"`
}

type MsgInvokeCapabilityResponse struct {
	InvocationID string `json:"invocation_id"`
	EscrowID     string `json:"escrow_id"`
}

type MsgUpdateCapabilityResponse struct{}

type MsgDeactivateCapabilityResponse struct{}
