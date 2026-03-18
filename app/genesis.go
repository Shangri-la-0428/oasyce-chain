package app

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	oasyceparams "github.com/oasyce/chain/app/params"
)

// OasyceDefaultGenesis returns the default genesis state for the application,
// with all denominations set to "uoas" and Oasyce-specific parameters applied.
// cdc is the JSON codec used by the module manager for proto JSON encoding.
func OasyceDefaultGenesis(cdc codec.JSONCodec) map[string]json.RawMessage {
	genesis := ModuleBasics.DefaultGenesis(cdc)

	// Patch standard SDK modules to use "uoas" and Oasyce parameters.
	patchStakingGenesis(cdc, genesis)
	patchMintGenesis(cdc, genesis)
	patchGovGenesis(cdc, genesis)
	patchCrisisGenesis(cdc, genesis)
	patchSlashingGenesis(cdc, genesis)

	return genesis
}

// patchStakingGenesis sets bond_denom to "uoas" and configures staking params.
func patchStakingGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var gs stakingtypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[stakingtypes.ModuleName], &gs)

	gs.Params.BondDenom = oasyceparams.BondDenom
	gs.Params.UnbondingTime = 21 * 24 * time.Hour // 21 days
	gs.Params.MaxValidators = 100

	genesis[stakingtypes.ModuleName] = cdc.MustMarshalJSON(&gs)
}

// patchMintGenesis sets mint_denom to "uoas" and configures inflation.
func patchMintGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var gs minttypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[minttypes.ModuleName], &gs)

	gs.Params.MintDenom = oasyceparams.BondDenom
	gs.Params.InflationMax = math.LegacyNewDecWithPrec(5, 2)    // 0.05
	gs.Params.InflationMin = math.LegacyNewDecWithPrec(5, 2)    // 0.05
	gs.Params.InflationRateChange = math.LegacyZeroDec()
	gs.Params.GoalBonded = math.LegacyNewDecWithPrec(67, 2)     // 0.67
	// blocks_per_year based on 5s block time: 365.25*24*3600/5 = 6,311,520
	gs.Params.BlocksPerYear = 6_311_520
	gs.Minter.Inflation = math.LegacyNewDecWithPrec(5, 2) // 0.05

	genesis[minttypes.ModuleName] = cdc.MustMarshalJSON(&gs)
}

// patchGovGenesis sets min_deposit denom to "uoas" and configures governance params.
func patchGovGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var gs govtypesv1.GenesisState
	cdc.MustUnmarshalJSON(genesis["gov"], &gs)

	// min_deposit: 1000 OAS = 1000 * 10^8 uoas = 100_000_000_000
	gs.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(oasyceparams.BondDenom, math.NewInt(100_000_000_000)))
	// expedited_min_deposit must be > min_deposit
	gs.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(oasyceparams.BondDenom, math.NewInt(500_000_000_000)))
	// Voting period: 7 days
	votingPeriod := 7 * 24 * time.Hour
	gs.Params.VotingPeriod = &votingPeriod
	// Expedited voting: 1 day
	expeditedVotingPeriod := 24 * time.Hour
	gs.Params.ExpeditedVotingPeriod = &expeditedVotingPeriod
	// Quorum: 0.4 (40%)
	gs.Params.Quorum = "0.400000000000000000"
	// Threshold: 0.667
	gs.Params.Threshold = "0.667000000000000000"
	// Expedited threshold must be > regular threshold
	gs.Params.ExpeditedThreshold = "0.750000000000000000"

	genesis["gov"] = cdc.MustMarshalJSON(&gs)
}

// patchCrisisGenesis sets constant_fee denom to "uoas".
func patchCrisisGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var gs crisistypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[crisistypes.ModuleName], &gs)

	gs.ConstantFee = sdk.NewCoin(oasyceparams.BondDenom, gs.ConstantFee.Amount)

	genesis[crisistypes.ModuleName] = cdc.MustMarshalJSON(&gs)
}

// patchSlashingGenesis configures slashing parameters.
func patchSlashingGenesis(cdc codec.JSONCodec, genesis map[string]json.RawMessage) {
	var gs slashingtypes.GenesisState
	cdc.MustUnmarshalJSON(genesis[slashingtypes.ModuleName], &gs)

	gs.Params.SignedBlocksWindow = 100
	gs.Params.MinSignedPerWindow = math.LegacyNewDecWithPrec(5, 1)      // 0.5
	gs.Params.SlashFractionDowntime = math.LegacyNewDecWithPrec(1, 2)   // 0.01
	gs.Params.SlashFractionDoubleSign = math.LegacyNewDecWithPrec(5, 2) // 0.05

	genesis[slashingtypes.ModuleName] = cdc.MustMarshalJSON(&gs)
}
