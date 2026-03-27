package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the anchor module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAnchorTrace{}, "oasyce/anchor/MsgAnchorTrace", nil)
	cdc.RegisterConcrete(&MsgAnchorBatch{}, "oasyce/anchor/MsgAnchorBatch", nil)
}

// RegisterInterfaces registers the anchor module's interface types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAnchorTrace{},
		&MsgAnchorBatch{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
