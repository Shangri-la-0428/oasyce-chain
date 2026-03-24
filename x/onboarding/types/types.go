package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// String implements the proto.Message interface for Params.
func (m *Params) String() string { return proto.CompactTextString(m) }

// DefaultParams returns the default onboarding module parameters.
func DefaultParams() Params {
	return Params{
		AirdropAmount:         sdk.NewCoin("uoas", math.NewInt(20000000)), // 20 OAS
		PowDifficulty:         16,                                         // 16 leading zero bits (~65536 attempts avg)
		RepaymentDeadlineDays: 90,                                         // 90 days
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Registrations: []Registration{},
		Params:        DefaultParams(),
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if !gs.Params.AirdropAmount.IsValid() || gs.Params.AirdropAmount.IsZero() {
		return ErrInvalidParams.Wrap("airdrop_amount must be positive")
	}
	if gs.Params.PowDifficulty == 0 {
		return ErrInvalidParams.Wrap("pow_difficulty must be > 0")
	}
	if gs.Params.RepaymentDeadlineDays == 0 {
		return ErrInvalidParams.Wrap("repayment_deadline_days must be > 0")
	}
	return nil
}
