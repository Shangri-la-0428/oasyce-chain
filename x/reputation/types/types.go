package types

import (
	"fmt"

	"cosmossdk.io/math"
	proto "github.com/cosmos/gogoproto/proto"
)

// String returns a text representation of Params.
// This is needed because the proto-generated genesis.pb.go omits it.
func (m *Params) String() string { return proto.CompactTextString(m) }

// DefaultParams returns the default reputation module parameters.
func DefaultParams() Params {
	return Params{
		MinRating:               0,
		MaxRating:               500,
		FeedbackCooldownSeconds: 3600,
		VerifiedWeight:          math.LegacyNewDec(1),             // 1.0
		UnverifiedWeight:        math.LegacyNewDecWithPrec(1, 1), // 0.1
	}
}

// Validate checks that Params fields are sane.
func (p Params) Validate() error {
	if p.MaxRating <= p.MinRating {
		return fmt.Errorf("max_rating (%d) must be greater than min_rating (%d)", p.MaxRating, p.MinRating)
	}
	// FeedbackCooldownSeconds is uint64, always >= 0; no check needed.
	if p.VerifiedWeight.IsNegative() {
		return fmt.Errorf("verified_weight must be non-negative")
	}
	if p.UnverifiedWeight.IsNegative() {
		return fmt.Errorf("unverified_weight must be non-negative")
	}
	return nil
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		ReputationScores: []ReputationScore{},
		Feedbacks:        []Feedback{},
		Reports:          []MisbehaviorReport{},
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if gs.Params.MaxRating == 0 {
		return ErrInvalidRating.Wrap("max_rating must be > 0")
	}
	if gs.Params.MinRating > gs.Params.MaxRating {
		return ErrInvalidRating.Wrap("min_rating must be <= max_rating")
	}
	if gs.Params.VerifiedWeight.IsNegative() {
		return ErrInvalidRating.Wrap("verified_weight must be >= 0")
	}
	if gs.Params.UnverifiedWeight.IsNegative() {
		return ErrInvalidRating.Wrap("unverified_weight must be >= 0")
	}
	return nil
}
