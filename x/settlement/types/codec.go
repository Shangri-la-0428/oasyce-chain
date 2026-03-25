package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the settlement module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateEscrow{}, "oasyce/settlement/MsgCreateEscrow", nil)
	cdc.RegisterConcrete(&MsgReleaseEscrow{}, "oasyce/settlement/MsgReleaseEscrow", nil)
	cdc.RegisterConcrete(&MsgRefundEscrow{}, "oasyce/settlement/MsgRefundEscrow", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "oasyce/settlement/MsgUpdateParams", nil)
}

// RegisterInterfaces registers the settlement module's interface types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateEscrow{},
		&MsgReleaseEscrow{},
		&MsgRefundEscrow{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
