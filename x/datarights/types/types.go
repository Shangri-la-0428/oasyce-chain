package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RightsType enumerates the possible rights types for a data asset.
type RightsType uint32

const (
	RightsTypeUnspecified RightsType = 0
	RightsOriginal        RightsType = 1
	RightsCoCreation      RightsType = 2
	RightsLicensed        RightsType = 3
	RightsCollection      RightsType = 4
)

func (r RightsType) String() string {
	switch r {
	case RightsOriginal:
		return "ORIGINAL"
	case RightsCoCreation:
		return "CO_CREATION"
	case RightsLicensed:
		return "LICENSED"
	case RightsCollection:
		return "COLLECTION"
	default:
		return "UNSPECIFIED"
	}
}

// Multiplier returns the bonding curve pricing multiplier for the rights type.
// Per spec section 5.3.
func (r RightsType) Multiplier() math.LegacyDec {
	switch r {
	case RightsOriginal:
		return math.LegacyNewDecWithPrec(10, 1) // 1.0
	case RightsCoCreation:
		return math.LegacyNewDecWithPrec(9, 1) // 0.9
	case RightsLicensed:
		return math.LegacyNewDecWithPrec(7, 1) // 0.7
	case RightsCollection:
		return math.LegacyNewDecWithPrec(3, 1) // 0.3
	default:
		return math.LegacyNewDecWithPrec(10, 1) // default to 1.0
	}
}

// DisputeStatus enumerates the possible states of a dispute.
type DisputeStatus uint32

const (
	DisputeStatusUnspecified DisputeStatus = 0
	StatusOpen               DisputeStatus = 1
	StatusResolved           DisputeStatus = 2
	StatusRejected           DisputeStatus = 3
)

func (s DisputeStatus) String() string {
	switch s {
	case StatusOpen:
		return "OPEN"
	case StatusResolved:
		return "RESOLVED"
	case StatusRejected:
		return "REJECTED"
	default:
		return "UNSPECIFIED"
	}
}

// DisputeRemedy enumerates the possible remedies for a dispute.
type DisputeRemedy uint32

const (
	RemedyNone             DisputeRemedy = 0
	RemedyDelist           DisputeRemedy = 1
	RemedyTransfer         DisputeRemedy = 2
	RemedyRightsCorrection DisputeRemedy = 3
	RemedyShareAdjustment  DisputeRemedy = 4
)

func (r DisputeRemedy) String() string {
	switch r {
	case RemedyDelist:
		return "DELIST"
	case RemedyTransfer:
		return "TRANSFER"
	case RemedyRightsCorrection:
		return "RIGHTS_CORRECTION"
	case RemedyShareAdjustment:
		return "SHARE_ADJUSTMENT"
	default:
		return "NONE"
	}
}

// CoCreator represents a co-creator of a data asset with their share allocation.
type CoCreator struct {
	Address  string `json:"address"`
	ShareBps uint32 `json:"share_bps"` // basis points, all co-creators must sum to 10000
}

// DataAsset represents a registered data asset on-chain.
type DataAsset struct {
	ID          string     `json:"id"`
	Owner       string     `json:"owner"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ContentHash string     `json:"content_hash"`
	Fingerprint string     `json:"fingerprint"`
	RightsType  RightsType `json:"rights_type"`
	Tags        []string   `json:"tags"`
	CoCreators  []CoCreator `json:"co_creators"`
	TotalShares math.Int   `json:"total_shares"`
	CreatedAt   int64      `json:"created_at"`
	IsActive    bool       `json:"is_active"`
}

// ShareHolder represents an account holding shares of a data asset.
type ShareHolder struct {
	Address     string   `json:"address"`
	AssetID     string   `json:"asset_id"`
	Shares      math.Int `json:"shares"`
	PurchasedAt int64    `json:"purchased_at"`
}

// Dispute represents a filed dispute against a data asset.
type Dispute struct {
	ID           string        `json:"id"`
	AssetID      string        `json:"asset_id"`
	Plaintiff    string        `json:"plaintiff"`
	Reason       string        `json:"reason"`
	EvidenceHash string        `json:"evidence_hash"`
	Status       DisputeStatus `json:"status"`
	Remedy       DisputeRemedy `json:"remedy"`
	Arbitrator   string        `json:"arbitrator"`
	ResolvedAt   int64         `json:"resolved_at"`
}

// Params defines the parameters for the datarights module.
type Params struct {
	MaxCoCreators     uint32   `json:"max_co_creators"`
	DisputeDeposit    sdk.Coin `json:"dispute_deposit"`
	DisputeTimeoutDays uint32  `json:"dispute_timeout_days"`
}

// DefaultParams returns the default datarights module parameters.
func DefaultParams() Params {
	return Params{
		MaxCoCreators:      10,
		DisputeDeposit:     sdk.NewCoin("uoas", math.NewInt(1000000000)), // 100 OAS
		DisputeTimeoutDays: 30,
	}
}

// GenesisState defines the datarights module's genesis state.
type GenesisState struct {
	DataAssets   []DataAsset   `json:"data_assets"`
	ShareHolders []ShareHolder `json:"share_holders"`
	Disputes     []Dispute     `json:"disputes"`
	Params       Params        `json:"params"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		DataAssets:   []DataAsset{},
		ShareHolders: []ShareHolder{},
		Disputes:     []Dispute{},
		Params:       DefaultParams(),
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

// Placeholder for time utility.
var _ = time.Now
