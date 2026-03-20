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
