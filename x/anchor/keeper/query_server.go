package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/anchor/types"
)

var _ types.QueryServer = queryServer{}

// queryServer implements the anchor QueryServer interface.
type queryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the anchor QueryServer.
func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Anchor returns an anchor record by trace_id.
func (q queryServer) Anchor(goCtx context.Context, req *types.QueryAnchorRequest) (*types.QueryAnchorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if len(req.TraceId) == 0 {
		return nil, types.ErrInvalidTraceID.Wrap("trace_id cannot be empty")
	}

	record, found := q.Keeper.GetAnchor(ctx, req.TraceId)
	if !found {
		return nil, types.ErrAnchorNotFound.Wrapf("trace_id %s not found", hex.EncodeToString(req.TraceId))
	}

	return &types.QueryAnchorResponse{Anchor: record}, nil
}

// IsAnchored checks whether a trace_id has been anchored.
func (q queryServer) IsAnchored(goCtx context.Context, req *types.QueryIsAnchoredRequest) (*types.QueryIsAnchoredResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if len(req.TraceId) == 0 {
		return nil, types.ErrInvalidTraceID.Wrap("trace_id cannot be empty")
	}

	anchored := q.Keeper.IsAnchored(ctx, req.TraceId)
	return &types.QueryIsAnchoredResponse{Anchored: anchored}, nil
}

// AnchorsByCapability returns anchors by capability with pagination.
func (q queryServer) AnchorsByCapability(goCtx context.Context, req *types.QueryAnchorsByCapabilityRequest) (*types.QueryAnchorsByCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.Capability == "" {
		return nil, types.ErrInvalidCapability.Wrap("capability cannot be empty")
	}

	// Apply pagination limit; default to 100.
	limit := uint64(100)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}

	anchors := q.Keeper.GetAnchorsByCapability(ctx, req.Capability, limit)

	return &types.QueryAnchorsByCapabilityResponse{
		Anchors: anchors,
	}, nil
}

// AnchorsByNode returns anchors by node public key with pagination.
func (q queryServer) AnchorsByNode(goCtx context.Context, req *types.QueryAnchorsByNodeRequest) (*types.QueryAnchorsByNodeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if len(req.NodePubkey) == 0 {
		return nil, types.ErrInvalidPubkey.Wrap("node_pubkey cannot be empty")
	}

	// Apply pagination limit; default to 100.
	limit := uint64(100)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}

	anchors := q.Keeper.GetAnchorsByNode(ctx, req.NodePubkey, limit)

	return &types.QueryAnchorsByNodeResponse{
		Anchors: anchors,
	}, nil
}

// AnchorsBySigil returns anchors by sigil ID with pagination.
func (q queryServer) AnchorsBySigil(goCtx context.Context, req *types.QueryAnchorsBySigilRequest) (*types.QueryAnchorsBySigilResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.SigilId == "" {
		return nil, types.ErrInvalidTraceID.Wrap("sigil_id cannot be empty")
	}

	limit := uint64(100)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}

	anchors := q.Keeper.GetAnchorsBySigil(ctx, req.SigilId, limit)

	return &types.QueryAnchorsBySigilResponse{
		Anchors: anchors,
	}, nil
}
