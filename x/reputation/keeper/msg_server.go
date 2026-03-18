package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/reputation/types"
)

// MsgServer implements the reputation module's Msg service.
type MsgServer struct {
	keeper Keeper
}

// NewMsgServer returns a new MsgServer instance.
func NewMsgServer(k Keeper) MsgServer {
	return MsgServer{keeper: k}
}

// SubmitFeedback handles MsgSubmitFeedback.
func (ms MsgServer) SubmitFeedback(goCtx context.Context, msg *types.MsgSubmitFeedback) (*types.MsgSubmitFeedbackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	feedbackID, err := ms.keeper.SubmitFeedback(ctx, msg.Creator, msg.InvocationID, msg.Rating, msg.Comment)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitFeedbackResponse{FeedbackID: feedbackID}, nil
}

// ReportMisbehavior handles MsgReportMisbehavior.
func (ms MsgServer) ReportMisbehavior(goCtx context.Context, msg *types.MsgReportMisbehavior) (*types.MsgReportMisbehaviorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reportID, err := ms.keeper.ReportMisbehavior(ctx, msg.Creator, msg.Target, msg.EvidenceType, msg.Evidence)
	if err != nil {
		return nil, err
	}

	return &types.MsgReportMisbehaviorResponse{ReportID: reportID}, nil
}

