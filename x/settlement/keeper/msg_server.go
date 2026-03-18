package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

// MsgServer implements the settlement message service.
type MsgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the settlement MsgServer interface.
func NewMsgServer(keeper Keeper) MsgServer {
	return MsgServer{Keeper: keeper}
}

// CreateEscrow handles MsgCreateEscrow.
func (m MsgServer) CreateEscrow(ctx sdk.Context, msg *types.MsgCreateEscrow) (*MsgCreateEscrowResponse, error) {
	escrowID, err := m.Keeper.CreateEscrow(ctx, msg.Creator, msg.Provider, msg.Amount, 0)
	if err != nil {
		return nil, err
	}
	return &MsgCreateEscrowResponse{EscrowID: escrowID}, nil
}

// ReleaseEscrow handles MsgReleaseEscrow.
func (m MsgServer) ReleaseEscrow(ctx sdk.Context, msg *types.MsgReleaseEscrow) (*MsgReleaseEscrowResponse, error) {
	if err := m.Keeper.ReleaseEscrow(ctx, msg.EscrowID, msg.Creator); err != nil {
		return nil, err
	}
	return &MsgReleaseEscrowResponse{}, nil
}

// RefundEscrow handles MsgRefundEscrow.
func (m MsgServer) RefundEscrow(ctx sdk.Context, msg *types.MsgRefundEscrow) (*MsgRefundEscrowResponse, error) {
	if err := m.Keeper.RefundEscrow(ctx, msg.EscrowID, msg.Creator); err != nil {
		return nil, err
	}
	return &MsgRefundEscrowResponse{}, nil
}

// Response types (plain Go, replacing protobuf-generated types).

// MsgCreateEscrowResponse is the response for CreateEscrow.
type MsgCreateEscrowResponse struct {
	EscrowID string `json:"escrow_id"`
}

// MsgReleaseEscrowResponse is the response for ReleaseEscrow.
type MsgReleaseEscrowResponse struct{}

// MsgRefundEscrowResponse is the response for RefundEscrow.
type MsgRefundEscrowResponse struct{}
