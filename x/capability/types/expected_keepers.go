package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

// SettlementKeeper defines the expected settlement module keeper interface.
type SettlementKeeper interface {
	CreateEscrow(ctx sdk.Context, creator, provider string, amount sdk.Coin, timeoutSeconds uint64) (string, error)
	ReleaseEscrow(ctx sdk.Context, escrowID string, releaser string) error
	RefundEscrow(ctx sdk.Context, escrowID string, refunder string) error
}
