package types

import (
	"cosmossdk.io/math"
)

// ReputationScore tracks the aggregated reputation of an address.
type ReputationScore struct {
	Address           string         `json:"address"`
	TotalScore        math.LegacyDec `json:"total_score"`
	TotalFeedbacks    uint64         `json:"total_feedbacks"`
	VerifiedFeedbacks uint64         `json:"verified_feedbacks"`
	LastUpdated       int64          `json:"last_updated"`
}

// Feedback represents a rating submitted for a completed invocation.
type Feedback struct {
	ID           string `json:"id"`
	InvocationID string `json:"invocation_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	Rating       uint32 `json:"rating"` // 0-500 = 0.0-5.0 scale
	Comment      string `json:"comment"`
	Verified     bool   `json:"verified"`
	Timestamp    int64  `json:"timestamp"`
}

// MisbehaviorReport represents a report of misbehavior for governance review.
type MisbehaviorReport struct {
	ID           string `json:"id"`
	Creator      string `json:"creator"`
	Target       string `json:"target"`
	EvidenceType string `json:"evidence_type"`
	Evidence     []byte `json:"evidence"`
	Timestamp    int64  `json:"timestamp"`
}

// Params defines the parameters for the reputation module.
type Params struct {
	MinRating               uint32         `json:"min_rating"`
	MaxRating               uint32         `json:"max_rating"`
	FeedbackCooldownSeconds uint64         `json:"feedback_cooldown_seconds"`
	VerifiedWeight          math.LegacyDec `json:"verified_weight"`
	UnverifiedWeight        math.LegacyDec `json:"unverified_weight"`
}

// DefaultParams returns the default reputation module parameters.
func DefaultParams() Params {
	return Params{
		MinRating:               0,
		MaxRating:               500,
		FeedbackCooldownSeconds: 60,
		VerifiedWeight:          math.LegacyNewDec(1),   // 1.0
		UnverifiedWeight:        math.LegacyNewDecWithPrec(1, 1), // 0.1
	}
}

// GenesisState defines the reputation module's genesis state.
type GenesisState struct {
	Params    Params            `json:"params"`
	Scores    []ReputationScore `json:"scores"`
	Feedbacks []Feedback        `json:"feedbacks"`
	Reports   []MisbehaviorReport `json:"reports"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Scores:    []ReputationScore{},
		Feedbacks: []Feedback{},
		Reports:   []MisbehaviorReport{},
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
