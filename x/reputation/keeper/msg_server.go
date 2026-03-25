package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/reputation/types"
)

var _ types.MsgServer = msgServer{}

// msgServer implements the reputation MsgServer interface.
type msgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the reputation MsgServer interface.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// SubmitFeedback handles MsgSubmitFeedback.
func (m msgServer) SubmitFeedback(goCtx context.Context, msg *types.MsgSubmitFeedback) (*types.MsgSubmitFeedbackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	feedbackID, err := m.Keeper.SubmitFeedback(ctx, msg.Creator, msg.InvocationId, msg.Rating, msg.Comment)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitFeedbackResponse{FeedbackId: feedbackID}, nil
}

// ReportMisbehavior handles MsgReportMisbehavior.
func (m msgServer) ReportMisbehavior(goCtx context.Context, msg *types.MsgReportMisbehavior) (*types.MsgReportMisbehaviorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reportID, err := m.Keeper.ReportMisbehavior(ctx, msg.Creator, msg.Target, msg.EvidenceType, msg.Evidence)
	if err != nil {
		return nil, err
	}

	return &types.MsgReportMisbehaviorResponse{ReportId: reportID}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter updates.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if m.Keeper.Authority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.Authority(), msg.Authority)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"reputation_params_updated",
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
