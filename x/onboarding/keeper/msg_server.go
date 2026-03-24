package keeper

import (
	"context"

	"github.com/oasyce/chain/x/onboarding/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m msgServer) SelfRegister(ctx context.Context, msg *types.MsgSelfRegister) (*types.MsgSelfRegisterResponse, error) {
	amount, err := m.Keeper.SelfRegister(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSelfRegisterResponse{AirdropAmount: amount}, nil
}

func (m msgServer) RepayDebt(ctx context.Context, msg *types.MsgRepayDebt) (*types.MsgRepayDebtResponse, error) {
	remaining, err := m.Keeper.RepayDebt(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgRepayDebtResponse{RemainingDebt: remaining}, nil
}
