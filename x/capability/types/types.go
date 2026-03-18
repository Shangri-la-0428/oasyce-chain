package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InvocationStatus represents the status of a capability invocation.
type InvocationStatus uint32

const (
	StatusPending InvocationStatus = iota
	StatusSuccess
	StatusFailed
)

func (s InvocationStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusSuccess:
		return "SUCCESS"
	case StatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// Capability represents a registered AI capability endpoint.
type Capability struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	EndpointURL  string   `json:"endpoint_url"`
	PricePerCall sdk.Coin `json:"price_per_call"`
	Tags         []string `json:"tags"`
	RateLimit    uint64   `json:"rate_limit"`
	TotalCalls   uint64   `json:"total_calls"`
	TotalEarned  math.Int `json:"total_earned"`
	AvgLatencyMs uint64   `json:"avg_latency_ms"`
	SuccessRate  uint64   `json:"success_rate"` // basis points 0-10000
	IsActive     bool     `json:"is_active"`
	CreatedAt    int64    `json:"created_at"`
}

// Invocation represents a single invocation of a capability.
type Invocation struct {
	ID           string           `json:"id"`
	CapabilityID string           `json:"capability_id"`
	Consumer     string           `json:"consumer"`
	Provider     string           `json:"provider"`
	InputHash    string           `json:"input_hash"`
	OutputHash   string           `json:"output_hash"`
	Status       InvocationStatus `json:"status"`
	Amount       sdk.Coin         `json:"amount"`
	EscrowID     string           `json:"escrow_id"`
	Timestamp    int64            `json:"timestamp"`
}

// Params holds the module parameters.
type Params struct {
	MinProviderStake sdk.Coin `json:"min_provider_stake"`
	MaxRateLimit     uint64   `json:"max_rate_limit"`
	ProtocolFeeRate  uint64   `json:"protocol_fee_rate"` // basis points
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		MinProviderStake: sdk.NewInt64Coin("uoas", 0),
		MaxRateLimit:     1000,
		ProtocolFeeRate:  500, // 5%
	}
}

// GenesisState defines the capability module's genesis state.
type GenesisState struct {
	Params       Params       `json:"params"`
	Capabilities []Capability `json:"capabilities"`
	Invocations  []Invocation `json:"invocations"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Capabilities: []Capability{},
		Invocations:  []Invocation{},
	}
}
