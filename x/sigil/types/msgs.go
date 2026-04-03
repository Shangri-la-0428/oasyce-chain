package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure message types implement sdk.Msg.
var (
	_ sdk.Msg = &MsgGenesis{}
	_ sdk.Msg = &MsgDissolve{}
	_ sdk.Msg = &MsgBond{}
	_ sdk.Msg = &MsgUnbond{}
	_ sdk.Msg = &MsgFork{}
	_ sdk.Msg = &MsgMerge{}
	_ sdk.Msg = &MsgUpdateParams{}
)

func (msg *MsgGenesis) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if len(msg.PublicKey) == 0 {
		return ErrInvalidPublicKey.Wrap("public_key cannot be empty")
	}
	if len(msg.PublicKey) > 64 {
		return ErrInvalidPublicKey.Wrapf("public_key too long: %d bytes (max 64)", len(msg.PublicKey))
	}
	for _, parent := range msg.Lineage {
		if parent == "" {
			return ErrInvalidLineage.Wrap("lineage contains empty sigil_id")
		}
	}
	return nil
}

func (msg *MsgDissolve) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if msg.SigilId == "" {
		return ErrInvalidSigilID.Wrap("sigil_id cannot be empty")
	}
	return nil
}

func (msg *MsgBond) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if msg.SigilA == "" || msg.SigilB == "" {
		return ErrInvalidSigilID.Wrap("sigil_a and sigil_b cannot be empty")
	}
	if msg.SigilA == msg.SigilB {
		return ErrSelfBond.Wrap("cannot bond a sigil with itself")
	}
	return nil
}

func (msg *MsgUnbond) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if msg.BondId == "" {
		return ErrInvalidBondID.Wrap("bond_id cannot be empty")
	}
	return nil
}

func (msg *MsgFork) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if msg.ParentSigilId == "" {
		return ErrInvalidSigilID.Wrap("parent_sigil_id cannot be empty")
	}
	if len(msg.PublicKey) == 0 {
		return ErrInvalidPublicKey.Wrap("public_key cannot be empty")
	}
	if msg.ForkMode != int32(ForkModeSymmetric) && msg.ForkMode != int32(ForkModeAsymmetric) {
		return ErrInvalidForkMode.Wrapf("unknown fork_mode: %d", msg.ForkMode)
	}
	return nil
}

func (msg *MsgMerge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer: %s", err)
	}
	if msg.SigilA == "" || msg.SigilB == "" {
		return ErrInvalidSigilID.Wrap("sigil_a and sigil_b cannot be empty")
	}
	if msg.SigilA == msg.SigilB {
		return ErrInvalidSigilID.Wrap("cannot merge a sigil with itself")
	}
	if msg.MergeMode != int32(MergeModeSymmetric) && msg.MergeMode != int32(MergeModeAbsorption) {
		return ErrInvalidMergeMode.Wrapf("unknown merge_mode: %d", msg.MergeMode)
	}
	return nil
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrInvalidAddress.Wrapf("invalid authority: %s", err)
	}
	return msg.Params.Validate()
}
