package types

import (
	"cosmossdk.io/math"
	proto "github.com/cosmos/gogoproto/proto"
)

func (m *Params) String() string { return proto.CompactTextString(m) }

func DefaultParams() Params {
	return Params{
		DefaultRedundancy:     3,
		MinTimeoutBlocks:      100,
		MaxTimeoutBlocks:      10000,
		RevealBlocks:          50,
		MinBounty:             math.NewInt(1000000), // 1 OAS minimum bounty (anti-spam)
		ExecutorShare:         math.LegacyNewDecWithPrec(90, 2), // 0.90
		ProtocolShare:         math.LegacyNewDecWithPrec(5, 2),  // 0.05
		BurnShare:             math.LegacyNewDecWithPrec(2, 2),  // 0.02
		SubmitterRebate:       math.LegacyNewDecWithPrec(3, 2),  // 0.03
		DepositRate:           math.LegacyNewDecWithPrec(10, 2), // 0.10
		DisputeBondRate:       math.LegacyNewDecWithPrec(10, 2), // 0.10
		MinExecutorReputation: 50,
		MaxTasksPerBlock:      100,
		ReputationCapPerEpoch: 100,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Tasks:       []Task{},
		Executors:   []ExecutorProfile{},
		TaskCounter: 0,
	}
}

func ValidateGenesis(gs GenesisState) error {
	p := gs.Params
	if p.DefaultRedundancy == 0 {
		return ErrInvalidParams.Wrap("default_redundancy must be > 0")
	}
	if p.MinTimeoutBlocks == 0 {
		return ErrInvalidParams.Wrap("min_timeout_blocks must be > 0")
	}
	if p.RevealBlocks == 0 {
		return ErrInvalidParams.Wrap("reveal_blocks must be > 0")
	}
	sum := p.ExecutorShare.Add(p.ProtocolShare).Add(p.BurnShare).Add(p.SubmitterRebate)
	if !sum.Equal(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrapf("shares must sum to 1.0, got %s", sum.String())
	}
	return nil
}

// IsTerminalStatus returns true if the task is in a final state.
func IsTerminalStatus(status TaskStatus) bool {
	return status == TASK_STATUS_SETTLED ||
		status == TASK_STATUS_EXPIRED ||
		status == TASK_STATUS_DISPUTED
}
