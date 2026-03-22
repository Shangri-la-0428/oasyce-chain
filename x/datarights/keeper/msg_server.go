package keeper

import (
	"context"

	"github.com/oasyce/chain/x/datarights/types"
)

var _ types.MsgServer = msgServer{}

// msgServer implements the datarights MsgServer interface.
type msgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the datarights MsgServer.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RegisterDataAsset handles MsgRegisterDataAsset.
func (m msgServer) RegisterDataAsset(ctx context.Context, msg *types.MsgRegisterDataAsset) (*types.MsgRegisterDataAssetResponse, error) {
	assetID, err := m.Keeper.RegisterDataAsset(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgRegisterDataAssetResponse{AssetId: assetID}, nil
}

// BuyShares handles MsgBuyShares.
func (m msgServer) BuyShares(ctx context.Context, msg *types.MsgBuyShares) (*types.MsgBuySharesResponse, error) {
	sharesMinted, err := m.Keeper.BuyShares(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgBuySharesResponse{SharesPurchased: sharesMinted}, nil
}

// FileDispute handles MsgFileDispute.
func (m msgServer) FileDispute(ctx context.Context, msg *types.MsgFileDispute) (*types.MsgFileDisputeResponse, error) {
	disputeID, err := m.Keeper.FileDispute(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgFileDisputeResponse{DisputeId: disputeID}, nil
}

// ResolveDispute handles MsgResolveDispute.
func (m msgServer) ResolveDispute(ctx context.Context, msg *types.MsgResolveDispute) (*types.MsgResolveDisputeResponse, error) {
	if err := m.Keeper.ResolveDispute(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgResolveDisputeResponse{}, nil
}

// DelistAsset handles MsgDelistAsset.
func (m msgServer) DelistAsset(ctx context.Context, msg *types.MsgDelistAsset) (*types.MsgDelistAssetResponse, error) {
	if err := m.Keeper.DelistAsset(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgDelistAssetResponse{}, nil
}

// SellShares handles MsgSellShares.
func (m msgServer) SellShares(ctx context.Context, msg *types.MsgSellShares) (*types.MsgSellSharesResponse, error) {
	payout, err := m.Keeper.SellShares(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSellSharesResponse{Payout: payout}, nil
}

// InitiateShutdown handles MsgInitiateShutdown.
func (m msgServer) InitiateShutdown(ctx context.Context, msg *types.MsgInitiateShutdown) (*types.MsgInitiateShutdownResponse, error) {
	if err := m.Keeper.InitiateShutdown(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgInitiateShutdownResponse{}, nil
}

// ClaimSettlement handles MsgClaimSettlement.
func (m msgServer) ClaimSettlement(ctx context.Context, msg *types.MsgClaimSettlement) (*types.MsgClaimSettlementResponse, error) {
	payout, err := m.Keeper.ClaimSettlement(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgClaimSettlementResponse{Payout: payout}, nil
}

// CreateMigrationPath handles MsgCreateMigrationPath.
func (m msgServer) CreateMigrationPath(ctx context.Context, msg *types.MsgCreateMigrationPath) (*types.MsgCreateMigrationPathResponse, error) {
	if err := m.Keeper.CreateMigrationPath(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgCreateMigrationPathResponse{}, nil
}

// DisableMigration handles MsgDisableMigration.
func (m msgServer) DisableMigration(ctx context.Context, msg *types.MsgDisableMigration) (*types.MsgDisableMigrationResponse, error) {
	if err := m.Keeper.DisableMigration(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgDisableMigrationResponse{}, nil
}

// Migrate handles MsgMigrate.
func (m msgServer) Migrate(ctx context.Context, msg *types.MsgMigrate) (*types.MsgMigrateResponse, error) {
	sharesReceived, err := m.Keeper.Migrate(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgMigrateResponse{SharesReceived: sharesReceived}, nil
}
