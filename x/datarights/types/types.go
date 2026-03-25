package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// Convenience constants mapping old names to proto enum values.
var (
	// RightsType aliases.
	RightsOriginal   = RIGHTS_TYPE_ORIGINAL
	RightsCoCreation = RIGHTS_TYPE_CO_CREATION
	RightsLicensed   = RIGHTS_TYPE_LICENSED
	RightsCollection = RIGHTS_TYPE_COLLECTION

	// DisputeStatus aliases.
	StatusOpen     = DISPUTE_STATUS_OPEN
	StatusResolved = DISPUTE_STATUS_RESOLVED
	StatusRejected = DISPUTE_STATUS_REJECTED

	// AssetStatus aliases.
	StatusActive       = ASSET_STATUS_ACTIVE
	StatusShuttingDown = ASSET_STATUS_SHUTTING_DOWN
	StatusSettled      = ASSET_STATUS_SETTLED

	// DisputeRemedy aliases.
	RemedyNone             = DISPUTE_REMEDY_UNSPECIFIED
	RemedyDelist           = DISPUTE_REMEDY_DELIST
	RemedyTransfer         = DISPUTE_REMEDY_TRANSFER
	RemedyRightsCorrection = DISPUTE_REMEDY_RIGHTS_CORRECTION
	RemedyShareAdjustment  = DISPUTE_REMEDY_SHARE_ADJUSTMENT
)

// RightsTypeMultiplier returns the bonding curve pricing multiplier for the rights type.
// Per spec section 5.3.
func RightsTypeMultiplier(r RightsType) math.LegacyDec {
	switch r {
	case RIGHTS_TYPE_ORIGINAL:
		return math.LegacyNewDecWithPrec(10, 1) // 1.0
	case RIGHTS_TYPE_CO_CREATION:
		return math.LegacyNewDecWithPrec(9, 1) // 0.9
	case RIGHTS_TYPE_LICENSED:
		return math.LegacyNewDecWithPrec(7, 1) // 0.7
	case RIGHTS_TYPE_COLLECTION:
		return math.LegacyNewDecWithPrec(3, 1) // 0.3
	default:
		return math.LegacyNewDecWithPrec(10, 1) // default to 1.0
	}
}

// String implements the proto.Message interface for Params.
// The generated genesis.pb.go omits this method.
func (m *Params) String() string { return proto.CompactTextString(m) }

// DefaultParams returns the default datarights module parameters.
func DefaultParams() Params {
	return Params{
		MaxCoCreators:           10,
		DisputeDeposit:          sdk.NewCoin("uoas", math.NewInt(10000000)), // 10 OAS
		DisputeTimeoutDays:      30,
		ShutdownCooldownSeconds: 604800, // 7 days
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		DataAssets:     []DataAsset{},
		Shareholders:   []ShareHolder{},
		Disputes:       []Dispute{},
		Params:         DefaultParams(),
		MigrationPaths: []MigrationPath{},
		AssetReserves:  []AssetReserve{},
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	if gs.Params.MaxCoCreators == 0 {
		return ErrInvalidCoCreators.Wrap("max_co_creators must be > 0")
	}
	if !gs.Params.DisputeDeposit.IsValid() || gs.Params.DisputeDeposit.IsZero() {
		return ErrInvalidParams.Wrap("dispute_deposit must be positive")
	}
	if gs.Params.DisputeTimeoutDays == 0 {
		return ErrInvalidParams.Wrap("dispute_timeout_days must be > 0")
	}
	return nil
}

// ValidateCoCreators checks that co-creators are valid and shares sum to 10000 bps.
func ValidateCoCreators(coCreators []CoCreator) error {
	if len(coCreators) == 0 {
		return nil
	}
	var totalBps uint32
	for _, cc := range coCreators {
		if cc.Address == "" {
			return ErrInvalidCoCreators.Wrap("co-creator address must not be empty")
		}
		if cc.ShareBps == 0 {
			return ErrInvalidCoCreators.Wrap("co-creator share_bps must be > 0")
		}
		totalBps += cc.ShareBps
	}
	if totalBps != 10000 {
		return ErrInvalidCoCreators.Wrapf("co-creator shares must sum to 10000 bps, got %d", totalBps)
	}
	return nil
}
