package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

var _ types.MsgServer = msgServer{}

// msgServer implements the proto-generated types.MsgServer interface.
type msgServer struct {
	keeper Keeper
}

// NewMsgServer returns a new MsgServer instance.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return msgServer{keeper: keeper}
}

// RegisterCapability handles MsgRegisterCapability.
func (m msgServer) RegisterCapability(goCtx context.Context, msg *types.MsgRegisterCapability) (*types.MsgRegisterCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	capID, err := m.keeper.RegisterCapability(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterCapabilityResponse{CapabilityId: capID}, nil
}

// InvokeCapability handles MsgInvokeCapability.
func (m msgServer) InvokeCapability(goCtx context.Context, msg *types.MsgInvokeCapability) (*types.MsgInvokeCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	invID, escrowID, err := m.keeper.InvokeCapability(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgInvokeCapabilityResponse{InvocationId: invID, EscrowId: escrowID}, nil
}

// UpdateCapability handles MsgUpdateCapability.
func (m msgServer) UpdateCapability(goCtx context.Context, msg *types.MsgUpdateCapability) (*types.MsgUpdateCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.UpdateCapability(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgUpdateCapabilityResponse{}, nil
}

// DeactivateCapability handles MsgDeactivateCapability.
func (m msgServer) DeactivateCapability(goCtx context.Context, msg *types.MsgDeactivateCapability) (*types.MsgDeactivateCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.DeactivateCapability(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDeactivateCapabilityResponse{}, nil
}
