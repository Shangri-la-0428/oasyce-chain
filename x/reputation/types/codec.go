package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers the reputation module's concrete types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitFeedback{}, "oasyce/reputation/MsgSubmitFeedback", nil)
	cdc.RegisterConcrete(&MsgReportMisbehavior{}, "oasyce/reputation/MsgReportMisbehavior", nil)
}
