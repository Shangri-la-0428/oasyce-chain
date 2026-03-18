package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// --- Query request/response types ---

type QueryCapabilityRequest struct {
	CapabilityID string `json:"capability_id"`
}

type QueryCapabilityResponse struct {
	Capability Capability `json:"capability"`
}

type QueryCapabilitiesRequest struct {
	Tag string `json:"tag"`
}

type QueryCapabilitiesResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

type QueryCapabilitiesByProviderRequest struct {
	Provider string `json:"provider"`
}

type QueryCapabilitiesByProviderResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

type QueryEarningsRequest struct {
	Provider string `json:"provider"`
}

type QueryEarningsResponse struct {
	TotalEarned sdk.Coins `json:"total_earned"`
	TotalCalls  uint64    `json:"total_calls"`
}
