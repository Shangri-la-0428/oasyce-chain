package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgGenesis{}, "oasyce/sigil/MsgGenesis", nil)
	cdc.RegisterConcrete(&MsgDissolve{}, "oasyce/sigil/MsgDissolve", nil)
	cdc.RegisterConcrete(&MsgBond{}, "oasyce/sigil/MsgBond", nil)
	cdc.RegisterConcrete(&MsgUnbond{}, "oasyce/sigil/MsgUnbond", nil)
	cdc.RegisterConcrete(&MsgFork{}, "oasyce/sigil/MsgFork", nil)
	cdc.RegisterConcrete(&MsgMerge{}, "oasyce/sigil/MsgMerge", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "oasyce/sigil/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgGenesis{},
		&MsgDissolve{},
		&MsgBond{},
		&MsgUnbond{},
		&MsgFork{},
		&MsgMerge{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
