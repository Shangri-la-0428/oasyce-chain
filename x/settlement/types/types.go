package types

import (
	"fmt"

	"cosmossdk.io/math"
	proto "github.com/cosmos/gogoproto/proto"
)

// String returns a text representation of Params.
// This is needed because the proto-generated genesis.pb.go omits it.
func (m *Params) String() string { return proto.CompactTextString(m) }

// Convenience aliases for proto-generated EscrowStatus enum values.
var (
	EscrowStatusUnspecified = ESCROW_STATUS_UNSPECIFIED
	EscrowStatusLocked      = ESCROW_STATUS_LOCKED
	EscrowStatusReleased    = ESCROW_STATUS_RELEASED
	EscrowStatusRefunded    = ESCROW_STATUS_REFUNDED
	EscrowStatusExpired     = ESCROW_STATUS_EXPIRED
)

// IsEscrowTerminal returns true if the escrow status is terminal (no further transitions).
func IsEscrowTerminal(status EscrowStatus) bool {
	return status == ESCROW_STATUS_RELEASED || status == ESCROW_STATUS_REFUNDED || status == ESCROW_STATUS_EXPIRED
}

// Protocol-level constants for bonding curve math.
// Fee split: 90% provider, 5% protocol, 2% burn, 3% treasury.
var (
	// ReserveRatio is the Bancor connector weight (CW = 0.5).
	ReserveRatio = math.LegacyNewDecWithPrec(5, 1) // 0.5

	// InitialPrice is the bootstrap price in uoas per token (1 uoas = 1 token).
	InitialPrice = math.LegacyOneDec()

	// ReserveSolvencyCap is the maximum fraction of reserve payable on sell.
	ReserveSolvencyCap = math.LegacyNewDecWithPrec(95, 2) // 0.95

	// BurnRate is the fraction of settlement burned permanently (2%).
	BurnRate = math.LegacyNewDecWithPrec(2, 2) // 0.02

	// TreasuryRate is the fraction of settlement sent to protocol treasury (3%).
	TreasuryRate = math.LegacyNewDecWithPrec(3, 2) // 0.03
)

// DefaultParams returns the default settlement module parameters.
func DefaultParams() Params {
	return Params{
		EscrowTimeoutSeconds: 300, // 5 minutes
		ProtocolFeeRate:      math.LegacyNewDecWithPrec(5, 2), // 5% = 0.05 (protocol fee)
	}
}

// Validate checks that Params fields are sane.
func (p Params) Validate() error {
	if p.EscrowTimeoutSeconds <= 0 {
		return fmt.Errorf("escrow_timeout_seconds must be positive, got %d", p.EscrowTimeoutSeconds)
	}
	if p.ProtocolFeeRate.IsNegative() || p.ProtocolFeeRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("protocol_fee_rate must be in [0, 1], got %s", p.ProtocolFeeRate)
	}
	return nil
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Escrows:            []Escrow{},
		BondingCurveStates: []BondingCurveState{},
		Params:             DefaultParams(),
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if gs.Params.EscrowTimeoutSeconds == 0 {
		return ErrInvalidParams.Wrap("escrow_timeout_seconds must be > 0")
	}
	if gs.Params.ProtocolFeeRate.IsNegative() {
		return ErrInvalidParams.Wrap("protocol_fee_rate must be >= 0")
	}
	if gs.Params.ProtocolFeeRate.GT(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrap("protocol_fee_rate must be <= 1")
	}
	return nil
}
