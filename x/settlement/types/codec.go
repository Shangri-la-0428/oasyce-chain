package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers the settlement module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateEscrow{}, "oasyce/settlement/MsgCreateEscrow", nil)
	cdc.RegisterConcrete(&MsgReleaseEscrow{}, "oasyce/settlement/MsgReleaseEscrow", nil)
	cdc.RegisterConcrete(&MsgRefundEscrow{}, "oasyce/settlement/MsgRefundEscrow", nil)
}
