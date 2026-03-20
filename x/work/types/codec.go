package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterExecutor{}, "oasyce/work/MsgRegisterExecutor", nil)
	cdc.RegisterConcrete(&MsgUpdateExecutor{}, "oasyce/work/MsgUpdateExecutor", nil)
	cdc.RegisterConcrete(&MsgSubmitTask{}, "oasyce/work/MsgSubmitTask", nil)
	cdc.RegisterConcrete(&MsgCommitResult{}, "oasyce/work/MsgCommitResult", nil)
	cdc.RegisterConcrete(&MsgRevealResult{}, "oasyce/work/MsgRevealResult", nil)
	cdc.RegisterConcrete(&MsgDisputeResult{}, "oasyce/work/MsgDisputeResult", nil)
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterExecutor{},
		&MsgUpdateExecutor{},
		&MsgSubmitTask{},
		&MsgCommitResult{},
		&MsgRevealResult{},
		&MsgDisputeResult{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
