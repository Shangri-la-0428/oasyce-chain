package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// DatarightsKeeper defines the read-only interface settlement needs from datarights.
type DatarightsKeeper interface {
	GetAssetReserve(ctx sdk.Context, assetID string) math.Int
	GetAssetReserveDenom(ctx sdk.Context, assetID string) string
	GetAssetTotalShares(ctx sdk.Context, assetID string) (math.Int, bool)
}
