package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the bank module interface needed by the halving module.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

// Keeper manages block reward halving logic.
type Keeper struct {
	bankKeeper BankKeeper
}

// NewKeeper creates a new halving Keeper.
func NewKeeper(bk BankKeeper) Keeper {
	return Keeper{bankKeeper: bk}
}

// Halving schedule constants (block heights).
const (
	HalvingInterval1 int64 = 10_000_000
	HalvingInterval2 int64 = 20_000_000
	HalvingInterval3 int64 = 30_000_000
)

// Block rewards in uoas (1 OAS = 1,000,000 uoas).
var (
	Reward0 = math.NewInt(4_000_000) // 4 OAS/block for blocks 0–10M
	Reward1 = math.NewInt(2_000_000) // 2 OAS/block for blocks 10M–20M
	Reward2 = math.NewInt(1_000_000) // 1 OAS/block for blocks 20M–30M
	Reward3 = math.NewInt(500_000)   // 0.5 OAS/block for blocks 30M+
)

// BlockReward returns the per-block reward in uoas for a given block height.
func BlockReward(height int64) math.Int {
	switch {
	case height <= HalvingInterval1:
		return Reward0
	case height <= HalvingInterval2:
		return Reward1
	case height <= HalvingInterval3:
		return Reward2
	default:
		return Reward3
	}
}
