package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	settlementtypes "github.com/oasyce/chain/x/settlement/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

// SettlementKeeper defines the expected settlement module keeper interface.
type SettlementKeeper interface {
	GetBondingCurveState(ctx sdk.Context, assetID string) (settlementtypes.BondingCurveState, bool)
	BuyShares(ctx context.Context, assetID string, buyer string, paymentAmount math.Int) (math.Int, error)
}
