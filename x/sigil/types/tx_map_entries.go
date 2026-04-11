package types

import proto "github.com/cosmos/gogoproto/proto"

// MsgPulse_DimensionsEntry exists only so gogoproto's dynamic type lookup
// can resolve the map entry as a proto.Message during tx parsing.
type MsgPulse_DimensionsEntry struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value int64  `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *MsgPulse_DimensionsEntry) Reset()         { *m = MsgPulse_DimensionsEntry{} }
func (m *MsgPulse_DimensionsEntry) String() string { return proto.CompactTextString(m) }
func (*MsgPulse_DimensionsEntry) ProtoMessage()    {}
func (*MsgPulse_DimensionsEntry) Descriptor() ([]byte, []int) {
	return fileDescriptor_sigil_tx, []int{14, 0}
}

func init() {
	proto.RegisterType((*MsgPulse_DimensionsEntry)(nil), "oasyce.sigil.v1.MsgPulse.DimensionsEntry")
}
