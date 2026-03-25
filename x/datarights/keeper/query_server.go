package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
)

var _ types.QueryServer = queryServer{}

// queryServer implements the datarights QueryServer interface.
type queryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the datarights QueryServer.
func NewQueryServer(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// DataAsset returns a single data asset by ID.
func (q queryServer) DataAsset(ctx context.Context, req *types.QueryDataAssetRequest) (*types.QueryDataAssetResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	asset, found := q.Keeper.GetAsset(sdkCtx, req.AssetId)
	if !found {
		return nil, types.ErrAssetNotFound.Wrapf("asset %s not found", req.AssetId)
	}
	return &types.QueryDataAssetResponse{DataAsset: asset}, nil
}

// DataAssets returns all data assets, optionally filtered by owner or tag.
func (q queryServer) DataAssets(ctx context.Context, req *types.QueryDataAssetsRequest) (*types.QueryDataAssetsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var assets []types.DataAsset
	if req.Owner != "" {
		assets = q.Keeper.ListAssetsByOwner(sdkCtx, req.Owner)
	} else {
		assets = q.Keeper.ListAssets(sdkCtx)
	}

	// Filter by tag if specified.
	if req.Tag != "" {
		var filtered []types.DataAsset
		for _, a := range assets {
			for _, t := range a.Tags {
				if t == req.Tag {
					filtered = append(filtered, a)
					break
				}
			}
		}
		assets = filtered
	}

	if assets == nil {
		assets = []types.DataAsset{}
	}

	return &types.QueryDataAssetsResponse{DataAssets: assets}, nil
}

// Shares returns all shareholders for a data asset.
func (q queryServer) Shares(ctx context.Context, req *types.QuerySharesRequest) (*types.QuerySharesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	holders := q.Keeper.GetShareHolders(sdkCtx, req.AssetId)
	if holders == nil {
		holders = []types.ShareHolder{}
	}
	return &types.QuerySharesResponse{Shareholders: holders}, nil
}

// Dispute returns a single dispute by ID.
func (q queryServer) Dispute(ctx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	dispute, found := q.Keeper.GetDispute(sdkCtx, req.DisputeId)
	if !found {
		return nil, types.ErrDisputeNotFound.Wrapf("dispute %s not found", req.DisputeId)
	}
	return &types.QueryDisputeResponse{Dispute: dispute}, nil
}

// Disputes returns all disputes, optionally filtered by asset ID.
func (q queryServer) Disputes(ctx context.Context, req *types.QueryDisputesRequest) (*types.QueryDisputesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var disputes []types.Dispute
	q.Keeper.IterateAllDisputes(sdkCtx, func(d types.Dispute) bool {
		if req.AssetId == "" || d.AssetId == req.AssetId {
			disputes = append(disputes, d)
		}
		return false
	})

	if disputes == nil {
		disputes = []types.Dispute{}
	}

	return &types.QueryDisputesResponse{Disputes: disputes}, nil
}

// MigrationPath returns a specific migration path by source and target.
func (q queryServer) MigrationPath(ctx context.Context, req *types.QueryMigrationPathRequest) (*types.QueryMigrationPathResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	mp, found := q.Keeper.GetMigrationPath(sdkCtx, req.SourceAssetId, req.TargetAssetId)
	if !found {
		return nil, types.ErrMigrationNotFound.Wrapf("migration path %s -> %s not found", req.SourceAssetId, req.TargetAssetId)
	}
	return &types.QueryMigrationPathResponse{MigrationPath: mp}, nil
}

// MigrationPaths returns all migration paths from a given source asset.
func (q queryServer) MigrationPaths(ctx context.Context, req *types.QueryMigrationPathsRequest) (*types.QueryMigrationPathsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var paths []types.MigrationPath
	q.Keeper.IterateAllMigrationPaths(sdkCtx, func(mp types.MigrationPath) bool {
		if mp.SourceAssetId == req.SourceAssetId {
			paths = append(paths, mp)
		}
		return false
	})
	if paths == nil {
		paths = []types.MigrationPath{}
	}
	return &types.QueryMigrationPathsResponse{MigrationPaths: paths}, nil
}

// AssetChildren returns all assets whose parent_asset_id matches the request.
func (q queryServer) AssetChildren(ctx context.Context, req *types.QueryAssetChildrenRequest) (*types.QueryAssetChildrenResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	allAssets := q.Keeper.ListAssets(sdkCtx)
	var children []types.DataAsset
	for _, a := range allAssets {
		if a.ParentAssetId == req.ParentAssetId {
			children = append(children, a)
		}
	}
	if children == nil {
		children = []types.DataAsset{}
	}
	return &types.QueryAssetChildrenResponse{DataAssets: children}, nil
}

// DatarightsParams returns the datarights module parameters.
func (q queryServer) DatarightsParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := q.Keeper.GetParams(sdkCtx)
	return &types.QueryParamsResponse{Params: params}, nil
}
