package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// SettlementKeeper defines the expected settlement module keeper interface.
type SettlementKeeper interface {
	CreateEscrow(ctx sdk.Context, creator, provider string, amount sdk.Coin, timeoutSeconds uint64) (string, error)
	ReleaseEscrow(ctx sdk.Context, escrowID string, releaser string) error
	RefundEscrow(ctx sdk.Context, escrowID string, refunder string) error
}
