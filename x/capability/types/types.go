package types

import (
	"fmt"

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

// DisputeDepositRate is the percentage of escrow value the consumer
// forfeits to the provider when disputing (basis points, 1000 = 10%).
// This prevents zero-cost disputes where the consumer receives the output
// and then disputes to reclaim the full payment.
const DisputeDepositRate uint64 = 1000

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		MinProviderStake: sdk.NewInt64Coin("uoas", 0),
		MaxRateLimit:     1000,
		ProtocolFeeRate:  500, // 5%
	}
}

// Validate checks that Params fields are sane.
func (p Params) Validate() error {
	if !p.MinProviderStake.IsValid() {
		return fmt.Errorf("invalid min_provider_stake: %s", p.MinProviderStake)
	}
	if p.ProtocolFeeRate > 10000 {
		return fmt.Errorf("protocol_fee_rate %d exceeds 10000 (100%%)", p.ProtocolFeeRate)
	}
	return nil
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Capabilities: []Capability{},
		Invocations:  []Invocation{},
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if gs.Params.MaxRateLimit == 0 {
		return fmt.Errorf("max_rate_limit must be positive")
	}
	if gs.Params.MinProviderStake.IsNil() || gs.Params.MinProviderStake.IsNegative() {
		return fmt.Errorf("min_provider_stake must be non-negative")
	}
	if gs.Params.ProtocolFeeRate > 10000 {
		return fmt.Errorf("protocol_fee_rate must be <= 10000 (100%%)")
	}
	return nil
}
