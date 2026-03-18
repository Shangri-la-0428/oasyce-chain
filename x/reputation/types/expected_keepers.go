package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	captypes "github.com/oasyce/chain/x/capability/types"
)

// CapabilityKeeper defines the expected capability module keeper interface.
type CapabilityKeeper interface {
	GetInvocation(ctx sdk.Context, id string) (captypes.Invocation, error)
}
