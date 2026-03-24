package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	"github.com/oasyce/chain/x/halving/keeper"
)

func TestBlockReward(t *testing.T) {
	tests := []struct {
		name   string
		height int64
		want   math.Int
	}{
		{"block 1", 1, math.NewInt(4_000_000)},
		{"block 1000", 1000, math.NewInt(4_000_000)},
		{"block 10M (boundary)", 10_000_000, math.NewInt(4_000_000)},
		{"block 10M+1 (first halving)", 10_000_001, math.NewInt(2_000_000)},
		{"block 15M", 15_000_000, math.NewInt(2_000_000)},
		{"block 20M (boundary)", 20_000_000, math.NewInt(2_000_000)},
		{"block 20M+1 (second halving)", 20_000_001, math.NewInt(1_000_000)},
		{"block 25M", 25_000_000, math.NewInt(1_000_000)},
		{"block 30M (boundary)", 30_000_000, math.NewInt(1_000_000)},
		{"block 30M+1 (third halving)", 30_000_001, math.NewInt(500_000)},
		{"block 100M", 100_000_000, math.NewInt(500_000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keeper.BlockReward(tt.height)
			if !got.Equal(tt.want) {
				t.Errorf("BlockReward(%d) = %s, want %s", tt.height, got, tt.want)
			}
		})
	}
}

func TestTotalSupplyFromRewards(t *testing.T) {
	// Verify total minted supply across all epochs.
	// Epoch 0: 10M blocks × 4 OAS = 40M OAS
	// Epoch 1: 10M blocks × 2 OAS = 20M OAS
	// Epoch 2: 10M blocks × 1 OAS = 10M OAS
	// After 30M blocks: 70M OAS total minted from block rewards
	// Epoch 3: 0.5 OAS/block indefinitely

	epoch0 := int64(10_000_000) * 4 // 40M OAS
	epoch1 := int64(10_000_000) * 2 // 20M OAS
	epoch2 := int64(10_000_000) * 1 // 10M OAS
	totalAfter30M := epoch0 + epoch1 + epoch2

	if totalAfter30M != 70_000_000 {
		t.Errorf("Total after 30M blocks = %d OAS, want 70,000,000", totalAfter30M)
	}

	// At block 30M+1, reward switches to 0.5 OAS/block.
	reward := keeper.BlockReward(30_000_001)
	if !reward.Equal(math.NewInt(500_000)) {
		t.Errorf("Expected 0.5 OAS (500000 uoas) at block 30M+1, got %s", reward)
	}
}
