package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
)

// QueryServer implements the datarights query service.
type QueryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the datarights QueryServer.
func NewQueryServer(keeper Keeper) QueryServer {
	return QueryServer{Keeper: keeper}
}

// QueryAsset returns a single data asset by ID.
func (q QueryServer) QueryAsset(ctx context.Context, assetID string) (*QueryAssetResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	asset, found := q.Keeper.GetAsset(sdkCtx, assetID)
	if !found {
		return nil, types.ErrAssetNotFound.Wrapf("asset %s not found", assetID)
	}
	return &QueryAssetResponse{Asset: asset}, nil
}

// QueryAssets returns all data assets.
func (q QueryServer) QueryAssets(ctx context.Context) (*QueryAssetsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	assets := q.Keeper.ListAssets(sdkCtx)
	return &QueryAssetsResponse{Assets: assets}, nil
}

// QueryShareHolders returns all shareholders for an asset.
func (q QueryServer) QueryShareHolders(ctx context.Context, assetID string) (*QueryShareHoldersResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	holders := q.Keeper.GetShareHolders(sdkCtx, assetID)
	return &QueryShareHoldersResponse{ShareHolders: holders}, nil
}

// QueryDispute returns a single dispute by ID.
func (q QueryServer) QueryDispute(ctx context.Context, disputeID string) (*QueryDisputeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	dispute, found := q.Keeper.GetDispute(sdkCtx, disputeID)
	if !found {
		return nil, types.ErrDisputeNotFound.Wrapf("dispute %s not found", disputeID)
	}
	return &QueryDisputeResponse{Dispute: dispute}, nil
}

// Response types for queries.

// QueryAssetResponse is the response for QueryAsset.
type QueryAssetResponse struct {
	Asset types.DataAsset `json:"asset"`
}

// QueryAssetsResponse is the response for QueryAssets.
type QueryAssetsResponse struct {
	Assets []types.DataAsset `json:"assets"`
}

// QueryShareHoldersResponse is the response for QueryShareHolders.
type QueryShareHoldersResponse struct {
	ShareHolders []types.ShareHolder `json:"share_holders"`
}

// QueryDisputeResponse is the response for QueryDispute.
type QueryDisputeResponse struct {
	Dispute types.Dispute `json:"dispute"`
}
