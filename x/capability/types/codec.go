package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the module's types on the given LegacyAmino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterCapability{}, "oasyce/capability/MsgRegisterCapability", nil)
	cdc.RegisterConcrete(&MsgInvokeCapability{}, "oasyce/capability/MsgInvokeCapability", nil)
	cdc.RegisterConcrete(&MsgUpdateCapability{}, "oasyce/capability/MsgUpdateCapability", nil)
	cdc.RegisterConcrete(&MsgDeactivateCapability{}, "oasyce/capability/MsgDeactivateCapability", nil)
}

// RegisterInterfaces registers the module's interface types with the InterfaceRegistry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterCapability{},
		&MsgInvokeCapability{},
		&MsgUpdateCapability{},
		&MsgDeactivateCapability{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
