package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the datarights module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterDataAsset{}, "oasyce/datarights/MsgRegisterDataAsset", nil)
	cdc.RegisterConcrete(&MsgBuyShares{}, "oasyce/datarights/MsgBuyShares", nil)
	cdc.RegisterConcrete(&MsgFileDispute{}, "oasyce/datarights/MsgFileDispute", nil)
	cdc.RegisterConcrete(&MsgResolveDispute{}, "oasyce/datarights/MsgResolveDispute", nil)
}

// RegisterInterfaces registers the module's interface types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterDataAsset{},
		&MsgBuyShares{},
		&MsgFileDispute{},
		&MsgResolveDispute{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
