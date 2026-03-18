package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

// MsgServer implements the capability module's Msg service.
type MsgServer struct {
	keeper Keeper
}

// NewMsgServer returns a new MsgServer instance.
func NewMsgServer(keeper Keeper) MsgServer {
	return MsgServer{keeper: keeper}
}

// RegisterCapability handles MsgRegisterCapability.
func (m MsgServer) RegisterCapability(goCtx context.Context, msg *types.MsgRegisterCapability) (*types.MsgRegisterCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	capID, err := m.keeper.RegisterCapability(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterCapabilityResponse{CapabilityID: capID}, nil
}

// InvokeCapability handles MsgInvokeCapability.
func (m MsgServer) InvokeCapability(goCtx context.Context, msg *types.MsgInvokeCapability) (*types.MsgInvokeCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	invID, escrowID, err := m.keeper.InvokeCapability(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgInvokeCapabilityResponse{InvocationID: invID, EscrowID: escrowID}, nil
}

// UpdateCapability handles MsgUpdateCapability.
func (m MsgServer) UpdateCapability(goCtx context.Context, msg *types.MsgUpdateCapability) (*types.MsgUpdateCapabilityResponse, error) {
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
func (m MsgServer) DeactivateCapability(goCtx context.Context, msg *types.MsgDeactivateCapability) (*types.MsgDeactivateCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.DeactivateCapability(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDeactivateCapabilityResponse{}, nil
}

