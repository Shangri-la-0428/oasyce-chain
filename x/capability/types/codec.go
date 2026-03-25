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
	cdc.RegisterConcrete(&MsgCompleteInvocation{}, "oasyce/capability/MsgCompleteInvocation", nil)
	cdc.RegisterConcrete(&MsgFailInvocation{}, "oasyce/capability/MsgFailInvocation", nil)
	cdc.RegisterConcrete(&MsgClaimInvocation{}, "oasyce/capability/MsgClaimInvocation", nil)
	cdc.RegisterConcrete(&MsgDisputeInvocation{}, "oasyce/capability/MsgDisputeInvocation", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "oasyce/capability/MsgUpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types with the InterfaceRegistry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterCapability{},
		&MsgInvokeCapability{},
		&MsgUpdateCapability{},
		&MsgDeactivateCapability{},
		&MsgCompleteInvocation{},
		&MsgFailInvocation{},
		&MsgClaimInvocation{},
		&MsgDisputeInvocation{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
