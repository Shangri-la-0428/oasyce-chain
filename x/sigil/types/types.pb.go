// Hand-written protobuf types for x/sigil module.
// Follows the same pattern as x/anchor/types/types.pb.go.

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

const _ = proto.GoGoProtoPackageIsVersion3

// ---------------------------------------------------------------------------
// Core types
// ---------------------------------------------------------------------------

// Sigil represents an on-chain identity record for a causal feedback loop.
type Sigil struct {
	SigilId          string      `protobuf:"bytes,1,opt,name=sigil_id,json=sigilId,proto3" json:"sigil_id,omitempty"`
	Creator          string      `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty"`
	PublicKey        []byte      `protobuf:"bytes,3,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	Status           SigilStatus `protobuf:"varint,4,opt,name=status,proto3,casttype=SigilStatus" json:"status,omitempty"`
	CreationHeight   int64       `protobuf:"varint,5,opt,name=creation_height,json=creationHeight,proto3" json:"creation_height,omitempty"`
	LastActiveHeight int64       `protobuf:"varint,6,opt,name=last_active_height,json=lastActiveHeight,proto3" json:"last_active_height,omitempty"`
	StateRoot        []byte      `protobuf:"bytes,7,opt,name=state_root,json=stateRoot,proto3" json:"state_root,omitempty"`
	Lineage          []string    `protobuf:"bytes,8,rep,name=lineage,proto3" json:"lineage,omitempty"`
	Metadata         string            `protobuf:"bytes,9,opt,name=metadata,proto3" json:"metadata,omitempty"`
	DimensionPulses  map[string]int64  `protobuf:"bytes,10,rep,name=dimension_pulses,json=dimensionPulses,proto3" json:"dimension_pulses,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (m *Sigil) Reset()         { *m = Sigil{} }
func (m *Sigil) String() string { return proto.CompactTextString(m) }
func (*Sigil) ProtoMessage()    {}

func (m *Sigil) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Sigil) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Sigil) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.DimensionPulses) > 0 {
		for k := range m.DimensionPulses {
			v := m.DimensionPulses[k]
			baseI := i
			if v != 0 {
				i = encodeVarint(dAtA, i, uint64(v))
				i--
				dAtA[i] = 0x10 // map value: field 2, wire type 0
			}
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarint(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0x0a // map key: field 1, wire type 2
			i = encodeVarint(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x52 // field 10, wire type 2
		}
	}
	if len(m.Metadata) > 0 {
		i -= len(m.Metadata)
		copy(dAtA[i:], m.Metadata)
		i = encodeVarint(dAtA, i, uint64(len(m.Metadata)))
		i--
		dAtA[i] = 0x4a // field 9, type 2
	}
	if len(m.Lineage) > 0 {
		for iNdEx := len(m.Lineage) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Lineage[iNdEx])
			copy(dAtA[i:], m.Lineage[iNdEx])
			i = encodeVarint(dAtA, i, uint64(len(m.Lineage[iNdEx])))
			i--
			dAtA[i] = 0x42 // field 8, type 2
		}
	}
	if len(m.StateRoot) > 0 {
		i -= len(m.StateRoot)
		copy(dAtA[i:], m.StateRoot)
		i = encodeVarint(dAtA, i, uint64(len(m.StateRoot)))
		i--
		dAtA[i] = 0x3a // field 7, type 2
	}
	if m.LastActiveHeight != 0 {
		i = encodeVarint(dAtA, i, uint64(m.LastActiveHeight))
		i--
		dAtA[i] = 0x30 // field 6, type 0
	}
	if m.CreationHeight != 0 {
		i = encodeVarint(dAtA, i, uint64(m.CreationHeight))
		i--
		dAtA[i] = 0x28 // field 5, type 0
	}
	if m.Status != 0 {
		i = encodeVarint(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x20 // field 4, type 0
	}
	if len(m.PublicKey) > 0 {
		i -= len(m.PublicKey)
		copy(dAtA[i:], m.PublicKey)
		i = encodeVarint(dAtA, i, uint64(len(m.PublicKey)))
		i--
		dAtA[i] = 0x1a // field 3, type 2
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarint(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x12 // field 2, type 2
	}
	if len(m.SigilId) > 0 {
		i -= len(m.SigilId)
		copy(dAtA[i:], m.SigilId)
		i = encodeVarint(dAtA, i, uint64(len(m.SigilId)))
		i--
		dAtA[i] = 0x0a // field 1, type 2
	}
	return len(dAtA) - i, nil
}

func (m *Sigil) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.SigilId)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	l = len(m.PublicKey)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovSize(uint64(m.Status))
	}
	if m.CreationHeight != 0 {
		n += 1 + sovSize(uint64(m.CreationHeight))
	}
	if m.LastActiveHeight != 0 {
		n += 1 + sovSize(uint64(m.LastActiveHeight))
	}
	l = len(m.StateRoot)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	if len(m.Lineage) > 0 {
		for _, s := range m.Lineage {
			l = len(s)
			n += 1 + l + sovSize(uint64(l))
		}
	}
	l = len(m.Metadata)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	if len(m.DimensionPulses) > 0 {
		for k, v := range m.DimensionPulses {
			mapEntrySize := 1 + len(k) + sovSize(uint64(len(k)))
			if v != 0 {
				mapEntrySize += 1 + sovSize(uint64(v))
			}
			n += 1 + mapEntrySize + sovSize(uint64(mapEntrySize))
		}
	}
	return n
}

func (m *Sigil) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
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
		switch fieldNum {
		case 1: // sigil_id
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SigilId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SigilId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2: // creator
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3: // public_key
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PublicKey", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PublicKey = append(m.PublicKey[:0], dAtA[iNdEx:postIndex]...)
			iNdEx = postIndex
		case 4: // status
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= SigilStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5: // creation_height
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreationHeight", wireType)
			}
			m.CreationHeight = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreationHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6: // last_active_height
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastActiveHeight", wireType)
			}
			m.LastActiveHeight = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LastActiveHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7: // state_root
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StateRoot", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StateRoot = append(m.StateRoot[:0], dAtA[iNdEx:postIndex]...)
			iNdEx = postIndex
		case 8: // lineage
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Lineage", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Lineage = append(m.Lineage, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 9: // metadata
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Metadata", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Metadata = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10: // dimension_pulses (map<string, int64>)
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DimensionPulses", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return fmt.Errorf("proto: negative length found during unmarshaling")
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.DimensionPulses == nil {
				m.DimensionPulses = make(map[string]int64)
			}
			var mapKey string
			var mapValue int64
			for iNdEx < postIndex {
				var entryWire uint64
				for shift := uint(0); ; shift += 7 {
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					entryWire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				entryFieldNum := int32(entryWire >> 3)
				switch entryFieldNum {
				case 1: // key (string)
					var stringLen uint64
					for shift := uint(0); ; shift += 7 {
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
					postStringIndex := iNdEx + int(stringLen)
					if postStringIndex > postIndex {
						return io.ErrUnexpectedEOF
					}
					mapKey = string(dAtA[iNdEx:postStringIndex])
					iNdEx = postStringIndex
				case 2: // value (int64)
					for shift := uint(0); ; shift += 7 {
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapValue |= int64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
				default:
					iNdEx = postIndex
				}
			}
			m.DimensionPulses[mapKey] = mapValue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return fmt.Errorf("proto: negative skip")
			}
			if iNdEx+skippy > l {
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

// Bond represents an on-chain bond record between two Sigils.
type Bond struct {
	BondId         string `protobuf:"bytes,1,opt,name=bond_id,json=bondId,proto3" json:"bond_id,omitempty"`
	SigilA         string `protobuf:"bytes,2,opt,name=sigil_a,json=sigilA,proto3" json:"sigil_a,omitempty"`
	SigilB         string `protobuf:"bytes,3,opt,name=sigil_b,json=sigilB,proto3" json:"sigil_b,omitempty"`
	TermsHash      []byte `protobuf:"bytes,4,opt,name=terms_hash,json=termsHash,proto3" json:"terms_hash,omitempty"`
	CreationHeight int64  `protobuf:"varint,5,opt,name=creation_height,json=creationHeight,proto3" json:"creation_height,omitempty"`
	Scope          string `protobuf:"bytes,6,opt,name=scope,proto3" json:"scope,omitempty"`
}

func (m *Bond) Reset()         { *m = Bond{} }
func (m *Bond) String() string { return proto.CompactTextString(m) }
func (*Bond) ProtoMessage()    {}

func (m *Bond) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Bond) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Bond) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if len(m.Scope) > 0 {
		i -= len(m.Scope)
		copy(dAtA[i:], m.Scope)
		i = encodeVarint(dAtA, i, uint64(len(m.Scope)))
		i--
		dAtA[i] = 0x32 // field 6
	}
	if m.CreationHeight != 0 {
		i = encodeVarint(dAtA, i, uint64(m.CreationHeight))
		i--
		dAtA[i] = 0x28 // field 5
	}
	if len(m.TermsHash) > 0 {
		i -= len(m.TermsHash)
		copy(dAtA[i:], m.TermsHash)
		i = encodeVarint(dAtA, i, uint64(len(m.TermsHash)))
		i--
		dAtA[i] = 0x22 // field 4
	}
	if len(m.SigilB) > 0 {
		i -= len(m.SigilB)
		copy(dAtA[i:], m.SigilB)
		i = encodeVarint(dAtA, i, uint64(len(m.SigilB)))
		i--
		dAtA[i] = 0x1a // field 3
	}
	if len(m.SigilA) > 0 {
		i -= len(m.SigilA)
		copy(dAtA[i:], m.SigilA)
		i = encodeVarint(dAtA, i, uint64(len(m.SigilA)))
		i--
		dAtA[i] = 0x12 // field 2
	}
	if len(m.BondId) > 0 {
		i -= len(m.BondId)
		copy(dAtA[i:], m.BondId)
		i = encodeVarint(dAtA, i, uint64(len(m.BondId)))
		i--
		dAtA[i] = 0x0a // field 1
	}
	return len(dAtA) - i, nil
}

func (m *Bond) Size() (n int) {
	if m == nil {
		return 0
	}
	l := len(m.BondId)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	l = len(m.SigilA)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	l = len(m.SigilB)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	l = len(m.TermsHash)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	if m.CreationHeight != 0 {
		n += 1 + sovSize(uint64(m.CreationHeight))
	}
	l = len(m.Scope)
	if l > 0 {
		n += 1 + l + sovSize(uint64(l))
	}
	return n
}

func (m *Bond) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
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
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BondId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BondId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SigilA", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SigilA = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SigilB", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SigilB = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TermsHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TermsHash = append(m.TermsHash[:0], dAtA[iNdEx:postIndex]...)
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreationHeight", wireType)
			}
			m.CreationHeight = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreationHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Scope", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
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
			postIndex := iNdEx + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Scope = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if iNdEx+skippy > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	return nil
}

// Params stores module parameters.
type Params struct {
	DormantThreshold  int64 `protobuf:"varint,1,opt,name=dormant_threshold,json=dormantThreshold,proto3" json:"dormant_threshold,omitempty"`
	DissolveThreshold int64 `protobuf:"varint,2,opt,name=dissolve_threshold,json=dissolveThreshold,proto3" json:"dissolve_threshold,omitempty"`
	SubmitWindow      int64 `protobuf:"varint,3,opt,name=submit_window,json=submitWindow,proto3" json:"submit_window,omitempty"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.SubmitWindow != 0 {
		i = encodeVarint(dAtA, i, uint64(m.SubmitWindow))
		i--
		dAtA[i] = 0x18
	}
	if m.DissolveThreshold != 0 {
		i = encodeVarint(dAtA, i, uint64(m.DissolveThreshold))
		i--
		dAtA[i] = 0x10
	}
	if m.DormantThreshold != 0 {
		i = encodeVarint(dAtA, i, uint64(m.DormantThreshold))
		i--
		dAtA[i] = 0x08
	}
	return len(dAtA) - i, nil
}

func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	if m.DormantThreshold != 0 {
		n += 1 + sovSize(uint64(m.DormantThreshold))
	}
	if m.DissolveThreshold != 0 {
		n += 1 + sovSize(uint64(m.DissolveThreshold))
	}
	if m.SubmitWindow != 0 {
		n += 1 + sovSize(uint64(m.SubmitWindow))
	}
	return n
}

func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
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
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DormantThreshold", wireType)
			}
			m.DormantThreshold = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DormantThreshold |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DissolveThreshold", wireType)
			}
			m.DissolveThreshold = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DissolveThreshold |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SubmitWindow", wireType)
			}
			m.SubmitWindow = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SubmitWindow |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if iNdEx+skippy > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	return nil
}

// GenesisState defines the sigil module genesis state.
type GenesisState struct {
	Sigils []Sigil `protobuf:"bytes,1,rep,name=sigils,proto3" json:"sigils"`
	Bonds  []Bond  `protobuf:"bytes,2,rep,name=bonds,proto3" json:"bonds"`
	Params Params  `protobuf:"bytes,3,opt,name=params,proto3" json:"params"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	{
		size := m.Params.Size()
		i -= size
		if _, err := m.Params.MarshalToSizedBuffer(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarint(dAtA, i, uint64(size))
		i--
		dAtA[i] = 0x1a // field 3
	}
	if len(m.Bonds) > 0 {
		for iNdEx := len(m.Bonds) - 1; iNdEx >= 0; iNdEx-- {
			size := m.Bonds[iNdEx].Size()
			i -= size
			if _, err := m.Bonds[iNdEx].MarshalToSizedBuffer(dAtA[i:]); err != nil {
				return 0, err
			}
			i = encodeVarint(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x12 // field 2
		}
	}
	if len(m.Sigils) > 0 {
		for iNdEx := len(m.Sigils) - 1; iNdEx >= 0; iNdEx-- {
			size := m.Sigils[iNdEx].Size()
			i -= size
			if _, err := m.Sigils[iNdEx].MarshalToSizedBuffer(dAtA[i:]); err != nil {
				return 0, err
			}
			i = encodeVarint(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x0a // field 1
		}
	}
	return len(dAtA) - i, nil
}

func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	if len(m.Sigils) > 0 {
		for _, e := range m.Sigils {
			l := e.Size()
			n += 1 + l + sovSize(uint64(l))
		}
	}
	if len(m.Bonds) > 0 {
		for _, e := range m.Bonds {
			l := e.Size()
			n += 1 + l + sovSize(uint64(l))
		}
	}
	l := m.Params.Size()
	n += 1 + l + sovSize(uint64(l))
	return n
}

func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
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
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sigils", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sigils = append(m.Sigils, Sigil{})
			if err := m.Sigils[len(m.Sigils)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Bonds", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Bonds = append(m.Bonds, Bond{})
			if err := m.Bonds[len(m.Bonds)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if iNdEx+skippy > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers — shared by all types in this file
// ---------------------------------------------------------------------------

func encodeVarint(dAtA []byte, offset int, v uint64) int {
	offset -= sovSize(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func sovSize(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func skip(dAtA []byte) (int, error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, fmt.Errorf("proto: negative length")
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, fmt.Errorf("proto: unexpected end group")
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, fmt.Errorf("proto: negative position after skip")
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

func (*Sigil) Descriptor() ([]byte, []int)        { return fileDescriptor_sigil_types, []int{0} }
func (*Bond) Descriptor() ([]byte, []int)         { return fileDescriptor_sigil_types, []int{1} }
func (*Params) Descriptor() ([]byte, []int)       { return fileDescriptor_sigil_types, []int{2} }
func (*GenesisState) Descriptor() ([]byte, []int) { return fileDescriptor_sigil_types, []int{3} }

func init() {
	proto.RegisterType((*Sigil)(nil), "oasyce.sigil.v1.Sigil")
	proto.RegisterMapType((map[string]int64)(nil), "oasyce.sigil.v1.Sigil.DimensionPulsesEntry")
	proto.RegisterType((*Bond)(nil), "oasyce.sigil.v1.Bond")
	proto.RegisterType((*Params)(nil), "oasyce.sigil.v1.Params")
	proto.RegisterType((*GenesisState)(nil), "oasyce.sigil.v1.GenesisState")
}

func init() { proto.RegisterFile("oasyce/sigil/v1/types.proto", fileDescriptor_sigil_types) }

var fileDescriptor_sigil_types = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x8c, 0x53, 0xdd, 0x6e, 0xd3, 0x30,
	0x14, 0x56, 0x9a, 0x35, 0x6d, 0xcf, 0x0a, 0xdb, 0x2c, 0xd8, 0x0c, 0x68, 0x52, 0x54, 0x84, 0x56,
	0x09, 0x48, 0xb4, 0xf1, 0x04, 0xeb, 0x0d, 0x9b, 0xb8, 0x41, 0x1e, 0x12, 0x12, 0x37, 0x91, 0x93,
	0x58, 0x8d, 0x45, 0x12, 0x47, 0xb1, 0xdb, 0xd1, 0x47, 0xe0, 0x1d, 0x90, 0x78, 0x04, 0x5e, 0x11,
	0xf9, 0xd8, 0xa1, 0x88, 0xdd, 0x70, 0xf9, 0xfd, 0x28, 0x39, 0xdf, 0xe7, 0x73, 0xe0, 0x85, 0xe2,
	0x7a, 0x57, 0x88, 0x54, 0xcb, 0xb5, 0xac, 0xd3, 0xed, 0x65, 0x6a, 0x76, 0x9d, 0xd0, 0x49, 0xd7,
	0x2b, 0xa3, 0xc8, 0x91, 0x13, 0x13, 0x14, 0x93, 0xed, 0xe5, 0xe2, 0xe7, 0x08, 0xc6, 0x77, 0x16,
	0x90, 0x67, 0x30, 0x45, 0x36, 0x93, 0x25, 0x0d, 0xe2, 0x60, 0x39, 0x63, 0x13, 0xc4, 0xb7, 0x25,
	0xa1, 0x30, 0x29, 0x7a, 0xc1, 0x8d, 0xea, 0xe9, 0xc8, 0x29, 0x1e, 0x92, 0x73, 0x80, 0x6e, 0x93,
	0xd7, 0xb2, 0xc8, 0xbe, 0x8a, 0x1d, 0x0d, 0xe3, 0x60, 0x39, 0x67, 0x33, 0xc7, 0x7c, 0x10, 0x3b,
	0x72, 0x0a, 0x91, 0x36, 0xdc, 0x6c, 0x34, 0x3d, 0x88, 0x83, 0xe5, 0x98, 0x79, 0x44, 0x2e, 0xe0,
	0x08, 0xbf, 0x20, 0x55, 0x9b, 0x55, 0x42, 0xae, 0x2b, 0x43, 0xc7, 0x71, 0xb0, 0x0c, 0xd9, 0xe3,
	0x81, 0xbe, 0x41, 0x96, 0xbc, 0x01, 0x52, 0x73, 0x6d, 0x32, 0x5e, 0x18, 0xb9, 0x15, 0x83, 0x37,
	0x42, 0xef, 0xb1, 0x55, 0xae, 0x51, 0xf0, 0xee, 0x73, 0x00, 0xfb, 0x03, 0x91, 0xf5, 0x4a, 0x19,
	0x3a, 0x71, 0xd3, 0x20, 0xc3, 0x94, 0x32, 0x36, 0x46, 0x2d, 0x5b, 0xc1, 0xd7, 0x82, 0x4e, 0xe3,
	0xd0, 0xc6, 0xf0, 0x90, 0x3c, 0x87, 0x69, 0x23, 0x0c, 0x2f, 0xb9, 0xe1, 0x74, 0x86, 0x09, 0xff,
	0xe0, 0xc5, 0xaf, 0x00, 0x0e, 0x56, 0xaa, 0x2d, 0xc9, 0x19, 0x4c, 0x72, 0xd5, 0x96, 0xfb, 0x7e,
	0x22, 0x0b, 0x6f, 0x51, 0x70, 0xcd, 0x71, 0x5f, 0x4f, 0x84, 0xf0, 0x7a, 0x2f, 0xe4, 0x58, 0xcd,
	0x20, 0xac, 0xec, 0xa0, 0x46, 0xf4, 0x8d, 0xce, 0x2a, 0xae, 0x2b, 0xec, 0x66, 0xce, 0x66, 0xc8,
	0xdc, 0x70, 0x5d, 0xfd, 0x7f, 0x3d, 0x4f, 0x60, 0xac, 0x0b, 0xd5, 0x09, 0x6c, 0x64, 0xc6, 0x1c,
	0x58, 0x7c, 0x0f, 0x20, 0xfa, 0xc8, 0x7b, 0xde, 0x68, 0xf2, 0x1a, 0x4e, 0x4a, 0xd5, 0x37, 0xbc,
	0x35, 0x99, 0xa9, 0x7a, 0xa1, 0x2b, 0x55, 0xbb, 0xe9, 0x43, 0x76, 0xec, 0x85, 0x4f, 0x03, 0x4f,
	0xde, 0x02, 0x29, 0xa5, 0xd6, 0xaa, 0xde, 0x8a, 0xbf, 0xdc, 0x23, 0x74, 0x9f, 0x0c, 0xca, 0xde,
	0xfe, 0x12, 0x1e, 0xe9, 0x4d, 0xde, 0x48, 0x93, 0xdd, 0xcb, 0xb6, 0x54, 0xf7, 0x98, 0x31, 0x64,
	0x73, 0x47, 0x7e, 0x46, 0x6e, 0xf1, 0x23, 0x80, 0xf9, 0x7b, 0xd1, 0x0a, 0x2d, 0xf5, 0x9d, 0x7d,
	0x08, 0x92, 0x80, 0x2b, 0x41, 0xd3, 0x20, 0x0e, 0x97, 0x87, 0x57, 0xa7, 0xc9, 0x3f, 0x2b, 0x99,
	0xe0, 0x3a, 0xfa, 0xaa, 0x6c, 0x82, 0xb1, 0xad, 0x59, 0xd3, 0x11, 0xda, 0x9f, 0x3e, 0xb0, 0xdb,
	0xb7, 0x61, 0xce, 0x43, 0x52, 0x88, 0x3a, 0x0c, 0x8e, 0xb3, 0x1c, 0x5e, 0x9d, 0x3d, 0x70, 0xbb,
	0x5e, 0x98, 0xb7, 0xad, 0x2e, 0xbe, 0xbc, 0x5a, 0x4b, 0x53, 0x6d, 0xf2, 0xa4, 0x50, 0x4d, 0xea,
	0x2f, 0xa7, 0xa8, 0xb8, 0x6c, 0xd3, 0x6f, 0xfe, 0x82, 0xf0, 0x7c, 0xf2, 0x08, 0xef, 0xe7, 0xdd,
	0xef, 0x00, 0x00, 0x00, 0xff, 0xff, 0x7b, 0x2f, 0x83, 0xb4, 0x5e, 0x03, 0x00, 0x00,
}
