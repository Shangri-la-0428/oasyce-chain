package app

import (
	"encoding/json"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/oasyce/chain/x/capability/types"
	reputationtypes "github.com/oasyce/chain/x/reputation/types"
	settlementtypes "github.com/oasyce/chain/x/settlement/types"
)

// DefaultOasyceGenesis returns a fully configured default genesis state for the
// Oasyce blockchain. It starts from OasyceDefaultGenesis (which patches the
// standard SDK module defaults to use "uoas") and then layers on the custom
// Oasyce module parameters.
//
// cdc is the JSON codec used for proto JSON encoding of SDK module genesis
// state. It is forwarded to OasyceDefaultGenesis.
func DefaultOasyceGenesis(cdc codec.JSONCodec) map[string]json.RawMessage {
	genesis := OasyceDefaultGenesis(cdc)

	// --- Settlement module ---
	patchSettlementGenesis(genesis)

	// --- Capability module ---
	patchCapabilityGenesis(genesis)

	// --- Reputation module ---
	patchReputationGenesis(genesis)

	return genesis
}

// patchSettlementGenesis configures settlement module defaults.
// escrow_timeout = 3600s (1 hour), protocol_fee_rate = 0.05 (5%).
func patchSettlementGenesis(genesis map[string]json.RawMessage) {
	gs := settlementtypes.GenesisState{
		Escrows:            []settlementtypes.Escrow{},
		BondingCurveStates: []settlementtypes.BondingCurveState{},
		Params: settlementtypes.Params{
			EscrowTimeoutSeconds: 3600, // 1 hour
			ProtocolFeeRate:      math.LegacyNewDecWithPrec(5, 2), // 5%
		},
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	genesis[settlementtypes.ModuleName] = bz
}

// patchCapabilityGenesis configures capability module defaults.
// min_provider_stake = 0 (no minimum for testnet).
func patchCapabilityGenesis(genesis map[string]json.RawMessage) {
	gs := capabilitytypes.GenesisState{
		Params: capabilitytypes.Params{
			MinProviderStake: sdk.NewCoin("uoas", math.NewInt(0)),
			MaxRateLimit:     1000,
			ProtocolFeeRate:  500, // 5% in basis points
		},
		Capabilities: []capabilitytypes.Capability{},
		Invocations:  []capabilitytypes.Invocation{},
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	genesis[capabilitytypes.ModuleName] = bz
}

// patchReputationGenesis configures reputation module defaults.
// min_rating = 0, max_rating = 500, feedback_cooldown = 60s.
func patchReputationGenesis(genesis map[string]json.RawMessage) {
	gs := reputationtypes.GenesisState{
		Params: reputationtypes.Params{
			MinRating:               0,
			MaxRating:               500,
			FeedbackCooldownSeconds: 60,
			VerifiedWeight:          math.LegacyNewDec(1),
			UnverifiedWeight:        math.LegacyNewDecWithPrec(1, 1), // 0.1
		},
		ReputationScores: []reputationtypes.ReputationScore{},
		Feedbacks: []reputationtypes.Feedback{},
		Reports:   []reputationtypes.MisbehaviorReport{},
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	genesis[reputationtypes.ModuleName] = bz
}
