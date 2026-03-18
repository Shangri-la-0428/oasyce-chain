package keeper

import (
	"context"

	"github.com/oasyce/chain/x/datarights/types"
)

// MsgServer implements the datarights message service.
type MsgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the datarights MsgServer.
func NewMsgServer(keeper Keeper) MsgServer {
	return MsgServer{Keeper: keeper}
}

// RegisterDataAsset handles MsgRegisterDataAsset.
func (m MsgServer) RegisterDataAsset(ctx context.Context, msg *types.MsgRegisterDataAsset) (*types.MsgRegisterDataAssetResponse, error) {
	assetID, err := m.Keeper.RegisterDataAsset(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgRegisterDataAssetResponse{AssetID: assetID}, nil
}

// BuyShares handles MsgBuyShares.
func (m MsgServer) BuyShares(ctx context.Context, msg *types.MsgBuyShares) (*types.MsgBuySharesResponse, error) {
	sharesMinted, err := m.Keeper.BuyShares(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgBuySharesResponse{SharesPurchased: sharesMinted.String()}, nil
}

// FileDispute handles MsgFileDispute.
func (m MsgServer) FileDispute(ctx context.Context, msg *types.MsgFileDispute) (*types.MsgFileDisputeResponse, error) {
	disputeID, err := m.Keeper.FileDispute(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgFileDisputeResponse{DisputeID: disputeID}, nil
}

// ResolveDispute handles MsgResolveDispute.
func (m MsgServer) ResolveDispute(ctx context.Context, msg *types.MsgResolveDispute) (*types.MsgResolveDisputeResponse, error) {
	if err := m.Keeper.ResolveDispute(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgResolveDisputeResponse{}, nil
}
