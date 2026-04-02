package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Policies:  []DelegatePolicy{},
		Delegates: []DelegateRecord{},
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	seen := make(map[string]bool)
	for _, p := range gs.Policies {
		if _, err := sdk.AccAddressFromBech32(p.Principal); err != nil {
			return ErrInvalidAddress.Wrapf("invalid principal in genesis: %s", err)
		}
		if seen[p.Principal] {
			return ErrInvalidPolicy.Wrapf("duplicate principal in genesis: %s", p.Principal)
		}
		seen[p.Principal] = true
	}
	for _, d := range gs.Delegates {
		if _, err := sdk.AccAddressFromBech32(d.Delegate); err != nil {
			return ErrInvalidAddress.Wrapf("invalid delegate in genesis: %s", err)
		}
	}
	return nil
}
