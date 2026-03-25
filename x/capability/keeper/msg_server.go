package keeper

import (
	"context"
	"fmt"

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

// CompleteInvocation handles MsgCompleteInvocation.
func (m msgServer) CompleteInvocation(goCtx context.Context, msg *types.MsgCompleteInvocation) (*types.MsgCompleteInvocationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.CompleteInvocation(ctx, msg.InvocationId, msg.OutputHash, msg.Creator, msg.UsageReport); err != nil {
		return nil, err
	}

	return &types.MsgCompleteInvocationResponse{}, nil
}

// FailInvocation handles MsgFailInvocation.
func (m msgServer) FailInvocation(goCtx context.Context, msg *types.MsgFailInvocation) (*types.MsgFailInvocationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.FailInvocation(ctx, msg.InvocationId, msg.Creator); err != nil {
		return nil, err
	}

	return &types.MsgFailInvocationResponse{}, nil
}

// ClaimInvocation handles MsgClaimInvocation.
func (m msgServer) ClaimInvocation(goCtx context.Context, msg *types.MsgClaimInvocation) (*types.MsgClaimInvocationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.ClaimInvocation(ctx, msg.InvocationId, msg.Creator); err != nil {
		return nil, err
	}

	return &types.MsgClaimInvocationResponse{}, nil
}

// DisputeInvocation handles MsgDisputeInvocation.
func (m msgServer) DisputeInvocation(goCtx context.Context, msg *types.MsgDisputeInvocation) (*types.MsgDisputeInvocationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := m.keeper.DisputeInvocation(ctx, msg.InvocationId, msg.Creator, msg.Reason); err != nil {
		return nil, err
	}

	return &types.MsgDisputeInvocationResponse{}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter updates.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if m.keeper.Authority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.keeper.Authority(), msg.Authority)
	}
	if err := m.keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("capability_params_updated", sdk.NewAttribute("authority", msg.Authority)))
	return &types.MsgUpdateParamsResponse{}, nil
}
