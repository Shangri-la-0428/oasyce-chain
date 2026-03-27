package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/anchor/types"
)

var _ types.MsgServer = msgServer{}

// msgServer implements the anchor MsgServer interface.
type msgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the anchor MsgServer interface.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// AnchorTrace handles MsgAnchorTrace.
func (m msgServer) AnchorTrace(goCtx context.Context, msg *types.MsgAnchorTrace) (*types.MsgAnchorTraceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	anchored, err := m.Keeper.AnchorTrace(ctx, msg)
	if err != nil {
		return nil, err
	}

	if !anchored {
		return nil, types.ErrDuplicateAnchor.Wrapf("trace_id %x already anchored", msg.TraceId)
	}

	return &types.MsgAnchorTraceResponse{}, nil
}

// AnchorBatch handles MsgAnchorBatch.
func (m msgServer) AnchorBatch(goCtx context.Context, msg *types.MsgAnchorBatch) (*types.MsgAnchorBatchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var anchored, skipped uint32

	for _, anchor := range msg.Anchors {
		// Override signer to match batch signer for consistency.
		anchor.Signer = msg.Signer

		ok, err := m.Keeper.AnchorTrace(ctx, anchor)
		if err != nil {
			// If it's a signer mismatch or validation error, skip this anchor.
			skipped++
			continue
		}
		if ok {
			anchored++
		} else {
			skipped++ // duplicate
		}
	}

	return &types.MsgAnchorBatchResponse{
		Anchored: anchored,
		Skipped:  skipped,
	}, nil
}
