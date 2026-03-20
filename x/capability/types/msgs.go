package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- MsgRegisterCapability ---

func (msg *MsgRegisterCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.Name == "" {
		return ErrInvalidInput.Wrap("name cannot be empty")
	}
	if msg.EndpointUrl == "" {
		return ErrInvalidInput.Wrap("endpoint_url cannot be empty")
	}
	if !msg.PricePerCall.IsValid() {
		return ErrInvalidInput.Wrap("invalid price_per_call")
	}
	return nil
}

// --- MsgInvokeCapability ---

func (msg *MsgInvokeCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityId == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	return nil
}

// --- MsgUpdateCapability ---

func (msg *MsgUpdateCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityId == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	if msg.PricePerCall != nil && !msg.PricePerCall.IsValid() {
		return ErrInvalidInput.Wrap("invalid price_per_call")
	}
	return nil
}

// --- MsgDeactivateCapability ---

func (msg *MsgDeactivateCapability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return ErrInvalidInput.Wrapf("invalid creator address: %s", err)
	}
	if msg.CapabilityId == "" {
		return ErrInvalidInput.Wrap("capability_id cannot be empty")
	}
	return nil
}
