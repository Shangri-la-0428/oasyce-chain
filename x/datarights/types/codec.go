package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers the datarights module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterDataAsset{}, "oasyce/datarights/MsgRegisterDataAsset", nil)
	cdc.RegisterConcrete(&MsgBuyShares{}, "oasyce/datarights/MsgBuyShares", nil)
	cdc.RegisterConcrete(&MsgFileDispute{}, "oasyce/datarights/MsgFileDispute", nil)
	cdc.RegisterConcrete(&MsgResolveDispute{}, "oasyce/datarights/MsgResolveDispute", nil)
}
