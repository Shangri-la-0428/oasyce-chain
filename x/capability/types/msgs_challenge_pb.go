package types

// Hand-written protobuf message definitions for challenge window message types.
// These match the proto definitions in proto/oasyce/capability/v1/tx.proto.

import (
	"fmt"
	"io"

	proto "github.com/cosmos/gogoproto/proto"
)

// --- MsgCompleteInvocation ---

type MsgCompleteInvocation struct {
	Creator      string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	InvocationId string `protobuf:"bytes,2,opt,name=invocation_id,json=invocationId,proto3" json:"invocation_id,omitempty"`
	OutputHash   string `protobuf:"bytes,3,opt,name=output_hash,json=outputHash,proto3" json:"output_hash,omitempty"`
	UsageReport  string `protobuf:"bytes,4,opt,name=usage_report,json=usageReport,proto3" json:"usage_report,omitempty"`
}

func (m *MsgCompleteInvocation) Reset()         { *m = MsgCompleteInvocation{} }
func (m *MsgCompleteInvocation) String() string { return proto.CompactTextString(m) }
func (*MsgCompleteInvocation) ProtoMessage()    {}
func (*MsgCompleteInvocation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{10}
}

func (m *MsgCompleteInvocation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgCompleteInvocation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}
func (m *MsgCompleteInvocation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.UsageReport) > 0 {
		i -= len(m.UsageReport)
		copy(dAtA[i:], m.UsageReport)
		i = encodeVarintTx(dAtA, i, uint64(len(m.UsageReport)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.OutputHash) > 0 {
		i -= len(m.OutputHash)
		copy(dAtA[i:], m.OutputHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.OutputHash)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.InvocationId) > 0 {
		i -= len(m.InvocationId)
		copy(dAtA[i:], m.InvocationId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.InvocationId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x0a
	}
	return len(dAtA) - i, nil
}
func (m *MsgCompleteInvocation) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.InvocationId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.OutputHash)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.UsageReport)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}
func (m *MsgCompleteInvocation) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgCompleteInvocation: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgCompleteInvocation: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InvocationId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InvocationId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OutputHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OutputHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UsageReport", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.UsageReport = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

type MsgCompleteInvocationResponse struct{}

func (m *MsgCompleteInvocationResponse) Reset()         { *m = MsgCompleteInvocationResponse{} }
func (m *MsgCompleteInvocationResponse) String() string { return "MsgCompleteInvocationResponse{}" }
func (*MsgCompleteInvocationResponse) ProtoMessage()    {}
func (*MsgCompleteInvocationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{11}
}
func (m *MsgCompleteInvocationResponse) Marshal() (dAtA []byte, err error) { return []byte{}, nil }
func (m *MsgCompleteInvocationResponse) MarshalTo(dAtA []byte) (int, error) { return 0, nil }
func (m *MsgCompleteInvocationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return len(dAtA), nil
}
func (m *MsgCompleteInvocationResponse) Size() int            { return 0 }
func (m *MsgCompleteInvocationResponse) Unmarshal(dAtA []byte) error { return nil }

// --- MsgFailInvocation ---

type MsgFailInvocation struct {
	Creator      string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	InvocationId string `protobuf:"bytes,2,opt,name=invocation_id,json=invocationId,proto3" json:"invocation_id,omitempty"`
}

func (m *MsgFailInvocation) Reset()         { *m = MsgFailInvocation{} }
func (m *MsgFailInvocation) String() string { return proto.CompactTextString(m) }
func (*MsgFailInvocation) ProtoMessage()    {}
func (*MsgFailInvocation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{12}
}

func (m *MsgFailInvocation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgFailInvocation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}
func (m *MsgFailInvocation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.InvocationId) > 0 {
		i -= len(m.InvocationId)
		copy(dAtA[i:], m.InvocationId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.InvocationId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x0a
	}
	return len(dAtA) - i, nil
}
func (m *MsgFailInvocation) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.InvocationId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}
func (m *MsgFailInvocation) Unmarshal(dAtA []byte) error {
	return unmarshalTwoStringFields(dAtA, &m.Creator, &m.InvocationId, "MsgFailInvocation")
}

type MsgFailInvocationResponse struct{}

func (m *MsgFailInvocationResponse) Reset()         { *m = MsgFailInvocationResponse{} }
func (m *MsgFailInvocationResponse) String() string { return "MsgFailInvocationResponse{}" }
func (*MsgFailInvocationResponse) ProtoMessage()    {}
func (*MsgFailInvocationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{13}
}
func (m *MsgFailInvocationResponse) Marshal() (dAtA []byte, err error) { return []byte{}, nil }
func (m *MsgFailInvocationResponse) MarshalTo(dAtA []byte) (int, error) { return 0, nil }
func (m *MsgFailInvocationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return len(dAtA), nil
}
func (m *MsgFailInvocationResponse) Size() int            { return 0 }
func (m *MsgFailInvocationResponse) Unmarshal(dAtA []byte) error { return nil }

// --- MsgClaimInvocation ---

type MsgClaimInvocation struct {
	Creator      string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	InvocationId string `protobuf:"bytes,2,opt,name=invocation_id,json=invocationId,proto3" json:"invocation_id,omitempty"`
}

func (m *MsgClaimInvocation) Reset()         { *m = MsgClaimInvocation{} }
func (m *MsgClaimInvocation) String() string { return proto.CompactTextString(m) }
func (*MsgClaimInvocation) ProtoMessage()    {}
func (*MsgClaimInvocation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{14}
}

func (m *MsgClaimInvocation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgClaimInvocation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}
func (m *MsgClaimInvocation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.InvocationId) > 0 {
		i -= len(m.InvocationId)
		copy(dAtA[i:], m.InvocationId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.InvocationId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x0a
	}
	return len(dAtA) - i, nil
}
func (m *MsgClaimInvocation) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.InvocationId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}
func (m *MsgClaimInvocation) Unmarshal(dAtA []byte) error {
	return unmarshalTwoStringFields(dAtA, &m.Creator, &m.InvocationId, "MsgClaimInvocation")
}

type MsgClaimInvocationResponse struct{}

func (m *MsgClaimInvocationResponse) Reset()         { *m = MsgClaimInvocationResponse{} }
func (m *MsgClaimInvocationResponse) String() string { return "MsgClaimInvocationResponse{}" }
func (*MsgClaimInvocationResponse) ProtoMessage()    {}
func (*MsgClaimInvocationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{15}
}
func (m *MsgClaimInvocationResponse) Marshal() (dAtA []byte, err error) { return []byte{}, nil }
func (m *MsgClaimInvocationResponse) MarshalTo(dAtA []byte) (int, error) { return 0, nil }
func (m *MsgClaimInvocationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return len(dAtA), nil
}
func (m *MsgClaimInvocationResponse) Size() int            { return 0 }
func (m *MsgClaimInvocationResponse) Unmarshal(dAtA []byte) error { return nil }

// --- MsgDisputeInvocation ---

type MsgDisputeInvocation struct {
	Creator      string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	InvocationId string `protobuf:"bytes,2,opt,name=invocation_id,json=invocationId,proto3" json:"invocation_id,omitempty"`
	Reason       string `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
}

func (m *MsgDisputeInvocation) Reset()         { *m = MsgDisputeInvocation{} }
func (m *MsgDisputeInvocation) String() string { return proto.CompactTextString(m) }
func (*MsgDisputeInvocation) ProtoMessage()    {}
func (*MsgDisputeInvocation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{16}
}

func (m *MsgDisputeInvocation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}
func (m *MsgDisputeInvocation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}
func (m *MsgDisputeInvocation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.Reason) > 0 {
		i -= len(m.Reason)
		copy(dAtA[i:], m.Reason)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Reason)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.InvocationId) > 0 {
		i -= len(m.InvocationId)
		copy(dAtA[i:], m.InvocationId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.InvocationId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x0a
	}
	return len(dAtA) - i, nil
}
func (m *MsgDisputeInvocation) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.InvocationId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Reason)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}
func (m *MsgDisputeInvocation) Unmarshal(dAtA []byte) error {
	return unmarshalThreeStringFields(dAtA, &m.Creator, &m.InvocationId, &m.Reason, "MsgDisputeInvocation")
}

type MsgDisputeInvocationResponse struct{}

func (m *MsgDisputeInvocationResponse) Reset()         { *m = MsgDisputeInvocationResponse{} }
func (m *MsgDisputeInvocationResponse) String() string { return "MsgDisputeInvocationResponse{}" }
func (*MsgDisputeInvocationResponse) ProtoMessage()    {}
func (*MsgDisputeInvocationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b895fbb0c07113, []int{17}
}
func (m *MsgDisputeInvocationResponse) Marshal() (dAtA []byte, err error) { return []byte{}, nil }
func (m *MsgDisputeInvocationResponse) MarshalTo(dAtA []byte) (int, error) { return 0, nil }
func (m *MsgDisputeInvocationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	return len(dAtA), nil
}
func (m *MsgDisputeInvocationResponse) Size() int            { return 0 }
func (m *MsgDisputeInvocationResponse) Unmarshal(dAtA []byte) error { return nil }

// --- init ---

func init() {
	proto.RegisterType((*MsgCompleteInvocation)(nil), "oasyce.capability.v1.MsgCompleteInvocation")
	proto.RegisterType((*MsgCompleteInvocationResponse)(nil), "oasyce.capability.v1.MsgCompleteInvocationResponse")
	proto.RegisterType((*MsgFailInvocation)(nil), "oasyce.capability.v1.MsgFailInvocation")
	proto.RegisterType((*MsgFailInvocationResponse)(nil), "oasyce.capability.v1.MsgFailInvocationResponse")
	proto.RegisterType((*MsgClaimInvocation)(nil), "oasyce.capability.v1.MsgClaimInvocation")
	proto.RegisterType((*MsgClaimInvocationResponse)(nil), "oasyce.capability.v1.MsgClaimInvocationResponse")
	proto.RegisterType((*MsgDisputeInvocation)(nil), "oasyce.capability.v1.MsgDisputeInvocation")
	proto.RegisterType((*MsgDisputeInvocationResponse)(nil), "oasyce.capability.v1.MsgDisputeInvocationResponse")
}

// --- Shared unmarshal helpers for messages with only string fields ---

func unmarshalTwoStringFields(dAtA []byte, field1, field2 *string, msgName string) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: %s: wiretype end group for non-group", msgName)
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: %s: illegal tag %d (wire type %d)", msgName, fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			s, newIdx, err := unmarshalString(dAtA, iNdEx, l)
			if err != nil {
				return err
			}
			*field1 = s
			iNdEx = newIdx
		case 2:
			s, newIdx, err := unmarshalString(dAtA, iNdEx, l)
			if err != nil {
				return err
			}
			*field2 = s
			iNdEx = newIdx
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func unmarshalThreeStringFields(dAtA []byte, field1, field2, field3 *string, msgName string) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: %s: wiretype end group for non-group", msgName)
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: %s: illegal tag %d (wire type %d)", msgName, fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			s, newIdx, err := unmarshalString(dAtA, iNdEx, l)
			if err != nil {
				return err
			}
			*field1 = s
			iNdEx = newIdx
		case 2:
			s, newIdx, err := unmarshalString(dAtA, iNdEx, l)
			if err != nil {
				return err
			}
			*field2 = s
			iNdEx = newIdx
		case 3:
			s, newIdx, err := unmarshalString(dAtA, iNdEx, l)
			if err != nil {
				return err
			}
			*field3 = s
			iNdEx = newIdx
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func unmarshalString(dAtA []byte, iNdEx, l int) (string, int, error) {
	var stringLen uint64
	for shift := uint(0); ; shift += 7 {
		if shift >= 64 {
			return "", 0, ErrIntOverflowTx
		}
		if iNdEx >= l {
			return "", 0, io.ErrUnexpectedEOF
		}
		b := dAtA[iNdEx]
		iNdEx++
		stringLen |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
	}
	intStringLen := int(stringLen)
	if intStringLen < 0 {
		return "", 0, ErrInvalidLengthTx
	}
	postIndex := iNdEx + intStringLen
	if postIndex < 0 {
		return "", 0, ErrInvalidLengthTx
	}
	if postIndex > l {
		return "", 0, io.ErrUnexpectedEOF
	}
	return string(dAtA[iNdEx:postIndex]), postIndex, nil
}
