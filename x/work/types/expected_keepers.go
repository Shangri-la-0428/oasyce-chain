package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	reputationtypes "github.com/oasyce/chain/x/reputation/types"
)

// BankKeeper defines the expected bank module interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// ReputationKeeper defines the expected reputation module interface.
type ReputationKeeper interface {
	GetReputation(ctx sdk.Context, address string) (reputationtypes.ReputationScore, bool)
}
