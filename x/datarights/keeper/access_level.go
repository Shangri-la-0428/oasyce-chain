package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Access level constants — equity thresholds for tiered access.
// Matches formulas.py EQUITY_ACCESS_THRESHOLDS.
var equityAccessThresholds = []struct {
	ThresholdBps int64  // Basis points (10000 = 100%)
	Level        string // Access level name
}{
	{1000, "L3"}, // >= 10% → Deliver
	{500, "L2"},  // >= 5%  → Compute
	{100, "L1"},  // >= 1%  → Sample
	{10, "L0"},   // >= 0.1% → Query
}

// Reputation thresholds that cap access level.
const (
	reputationSandbox = 20 // R < 20 → L0 only
	reputationLimited = 50 // R 20-49 → L0+L1; R >= 50 → all
)

// levelIndex maps level names to numeric indices.
var levelIndex = map[string]int{
	"L0": 0,
	"L1": 1,
	"L2": 2,
	"L3": 3,
}

// GetAccessLevel determines the access level for an address on a data asset
// based on their equity percentage and reputation score.
// Returns "" if the address has insufficient equity.
func (k Keeper) GetAccessLevel(ctx sdk.Context, assetID string, address string, reputation math.LegacyDec) string {
	asset, found := k.GetAsset(ctx, assetID)
	if !found || asset.TotalShares.IsZero() {
		return ""
	}

	// Get shareholder's shares.
	sh, found := k.GetShareHolder(ctx, assetID, address)
	if !found || sh.Shares.IsZero() {
		return ""
	}

	// Calculate equity in basis points: (shares * 10000) / totalShares
	sharesBps := sh.Shares.Mul(math.NewInt(10000)).Quo(asset.TotalShares)

	// Find highest qualifying level from equity.
	equityLevel := ""
	for _, threshold := range equityAccessThresholds {
		if sharesBps.GTE(math.NewInt(threshold.ThresholdBps)) {
			equityLevel = threshold.Level
			break
		}
	}

	if equityLevel == "" {
		return ""
	}

	// Cap by reputation.
	repInt := reputation.TruncateInt().Int64()
	var maxIdx int
	if repInt < reputationSandbox {
		maxIdx = 0
	} else if repInt < reputationLimited {
		maxIdx = 1
	} else {
		maxIdx = 3
	}

	equityIdx := levelIndex[equityLevel]
	if equityIdx > maxIdx {
		equityIdx = maxIdx
	}

	levels := []string{"L0", "L1", "L2", "L3"}
	return levels[equityIdx]
}
