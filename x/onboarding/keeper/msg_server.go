package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

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

// UpdateParams handles MsgUpdateParams — governance-gated parameter updates.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if m.Keeper.Authority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.Authority(), msg.Authority)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"onboarding_params_updated",
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
