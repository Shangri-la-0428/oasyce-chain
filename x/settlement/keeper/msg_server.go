package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

var _ types.MsgServer = msgServer{}

// msgServer implements the settlement MsgServer interface.
type msgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the settlement MsgServer interface.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// CreateEscrow handles MsgCreateEscrow.
func (m msgServer) CreateEscrow(goCtx context.Context, msg *types.MsgCreateEscrow) (*types.MsgCreateEscrowResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// The proto MsgCreateEscrow does not carry a provider field.
	// The provider is determined by the capability/asset being purchased.
	// For now, pass the creator as a placeholder; real provider resolution
	// happens in the capability module via the keeper interface.
	escrowID, err := m.Keeper.CreateEscrow(ctx, msg.Creator, msg.Creator, msg.Amount, 0)
	if err != nil {
		return nil, err
	}
	return &types.MsgCreateEscrowResponse{EscrowId: escrowID}, nil
}

// ReleaseEscrow handles MsgReleaseEscrow.
func (m msgServer) ReleaseEscrow(goCtx context.Context, msg *types.MsgReleaseEscrow) (*types.MsgReleaseEscrowResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.ReleaseEscrow(ctx, msg.EscrowId, msg.Creator); err != nil {
		return nil, err
	}
	return &types.MsgReleaseEscrowResponse{}, nil
}

// RefundEscrow handles MsgRefundEscrow.
func (m msgServer) RefundEscrow(goCtx context.Context, msg *types.MsgRefundEscrow) (*types.MsgRefundEscrowResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.RefundEscrow(ctx, msg.EscrowId, msg.Creator); err != nil {
		return nil, err
	}
	return &types.MsgRefundEscrowResponse{}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter updates.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if m.Keeper.Authority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.Authority(), msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"settlement_params_updated",
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
