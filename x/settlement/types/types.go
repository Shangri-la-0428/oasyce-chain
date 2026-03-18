package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EscrowStatus represents the state of an escrow.
type EscrowStatus uint32

const (
	EscrowStatusUnspecified EscrowStatus = 0
	EscrowStatusLocked      EscrowStatus = 1
	EscrowStatusReleased    EscrowStatus = 2
	EscrowStatusRefunded    EscrowStatus = 3
	EscrowStatusExpired     EscrowStatus = 4
)

func (s EscrowStatus) String() string {
	switch s {
	case EscrowStatusLocked:
		return "LOCKED"
	case EscrowStatusReleased:
		return "RELEASED"
	case EscrowStatusRefunded:
		return "REFUNDED"
	case EscrowStatusExpired:
		return "EXPIRED"
	default:
		return "UNSPECIFIED"
	}
}

// IsTerminal returns true if the escrow status is terminal (no further transitions).
func (s EscrowStatus) IsTerminal() bool {
	return s == EscrowStatusReleased || s == EscrowStatusRefunded || s == EscrowStatusExpired
}

// Escrow represents a locked-fund escrow for a capability invocation or data purchase.
type Escrow struct {
	ID        string       `json:"id"`
	Creator   string       `json:"creator"`
	Provider  string       `json:"provider"`
	Amount    sdk.Coin     `json:"amount"`
	Status    EscrowStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// BondingCurveState holds the bonding curve parameters for an asset.
type BondingCurveState struct {
	AssetID     string        `json:"asset_id"`
	TotalShares math.Int      `json:"total_shares"`
	Reserve     math.Int      `json:"reserve"`
	PriceFactor math.LegacyDec `json:"price_factor"`
	BuyerCount  uint32        `json:"buyer_count"`
}

// Params defines the parameters for the settlement module.
type Params struct {
	EscrowTimeoutSeconds uint64         `json:"escrow_timeout_seconds"`
	ProtocolFeeRate      math.LegacyDec `json:"protocol_fee_rate"`
}

// DefaultParams returns the default settlement module parameters.
func DefaultParams() Params {
	return Params{
		EscrowTimeoutSeconds: 300, // 5 minutes
		ProtocolFeeRate:      math.LegacyNewDecWithPrec(5, 2), // 5% = 0.05
	}
}

// GenesisState defines the settlement module's genesis state.
type GenesisState struct {
	Escrows            []Escrow            `json:"escrows"`
	BondingCurveStates []BondingCurveState `json:"bonding_curve_states"`
	Params             Params              `json:"params"`
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
