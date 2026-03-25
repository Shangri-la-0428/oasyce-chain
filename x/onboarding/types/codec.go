package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSelfRegister{}, "oasyce/onboarding/MsgSelfRegister", nil)
	cdc.RegisterConcrete(&MsgRepayDebt{}, "oasyce/onboarding/MsgRepayDebt", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "oasyce/onboarding/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSelfRegister{},
		&MsgRepayDebt{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
