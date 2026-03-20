package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterCodec registers the reputation module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitFeedback{}, "oasyce/reputation/MsgSubmitFeedback", nil)
	cdc.RegisterConcrete(&MsgReportMisbehavior{}, "oasyce/reputation/MsgReportMisbehavior", nil)
}

// RegisterInterfaces registers the reputation module's interface types with the interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitFeedback{},
		&MsgReportMisbehavior{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
