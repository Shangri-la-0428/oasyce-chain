package types

import (
	"reflect"
	"testing"

	proto "github.com/cosmos/gogoproto/proto"
)

func TestMsgPulseDimensionsEntryResolvesToProtoMessage(t *testing.T) {
	typ := proto.MessageType("oasyce.sigil.v1.MsgPulse.DimensionsEntry")
	if typ == nil {
		t.Fatalf("MsgPulse.DimensionsEntry type not registered")
	}
	if !typ.Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
		t.Fatalf("registered type does not implement proto.Message: %v", typ)
	}
}
