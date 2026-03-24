package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/oasyce/chain/x/halving/types"
)

// BeginBlocker mints the correct block reward based on current height
// and sends it to fee_collector for distribution to validators.
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	reward := BlockReward(ctx.BlockHeight())
	if reward.IsZero() {
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin("uoas", reward))

	// Mint to halving module account.
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return err
	}

	// Transfer to fee_collector → distribution module → validators + delegators.
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, coins)
}
