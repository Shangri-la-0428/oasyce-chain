package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Convenience constants mapping old status names to proto enum values.
const (
	StatusPending   = INVOCATION_STATUS_PENDING
	StatusSuccess   = INVOCATION_STATUS_SUCCESS
	StatusFailed    = INVOCATION_STATUS_FAILED
	StatusCompleted = INVOCATION_STATUS_COMPLETED
	StatusDisputed  = INVOCATION_STATUS_DISPUTED
)

// ChallengeWindow is the number of blocks after completion during which
// the consumer can dispute the invocation output.
const ChallengeWindow int64 = 100

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		MinProviderStake: sdk.NewInt64Coin("uoas", 0),
		MaxRateLimit:     1000,
		ProtocolFeeRate:  500, // 5%
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Capabilities: []Capability{},
		Invocations:  []Invocation{},
	}
}
