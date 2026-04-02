package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSetPolicy{}, "oasyce/delegate/MsgSetPolicy", nil)
	cdc.RegisterConcrete(&MsgEnroll{}, "oasyce/delegate/MsgEnroll", nil)
	cdc.RegisterConcrete(&MsgRevoke{}, "oasyce/delegate/MsgRevoke", nil)
	cdc.RegisterConcrete(&MsgExec{}, "oasyce/delegate/MsgExec", nil)
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetPolicy{},
		&MsgEnroll{},
		&MsgRevoke{},
		&MsgExec{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
