// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: burner/burned_coins.proto

package types

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type BurnedCoins struct {
	Burned string `protobuf:"bytes,1,opt,name=burned,proto3" json:"burned,omitempty"`
	Height string `protobuf:"bytes,2,opt,name=height,proto3" json:"height,omitempty"`
}

func (m *BurnedCoins) Reset()         { *m = BurnedCoins{} }
func (m *BurnedCoins) String() string { return proto.CompactTextString(m) }
func (*BurnedCoins) ProtoMessage()    {}
func (*BurnedCoins) Descriptor() ([]byte, []int) {
	return fileDescriptor_930c7119c82a796d, []int{0}
}
func (m *BurnedCoins) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BurnedCoins) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BurnedCoins.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BurnedCoins) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BurnedCoins.Merge(m, src)
}
func (m *BurnedCoins) XXX_Size() int {
	return m.Size()
}
func (m *BurnedCoins) XXX_DiscardUnknown() {
	xxx_messageInfo_BurnedCoins.DiscardUnknown(m)
}

var xxx_messageInfo_BurnedCoins proto.InternalMessageInfo

func (m *BurnedCoins) GetBurned() string {
	if m != nil {
		return m.Burned
	}
	return ""
}

func (m *BurnedCoins) GetHeight() string {
	if m != nil {
		return m.Height
	}
	return ""
}

func init() {
	proto.RegisterType((*BurnedCoins)(nil), "bze.burner.v1.BurnedCoins")
}

func init() { proto.RegisterFile("burner/burned_coins.proto", fileDescriptor_930c7119c82a796d) }

var fileDescriptor_930c7119c82a796d = []byte{
	// 167 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4c, 0x2a, 0x2d, 0xca,
	0x4b, 0x2d, 0xd2, 0x07, 0x53, 0x29, 0xf1, 0xc9, 0xf9, 0x99, 0x79, 0xc5, 0x7a, 0x05, 0x45, 0xf9,
	0x25, 0xf9, 0x42, 0xbc, 0x49, 0x55, 0xa9, 0x7a, 0x10, 0x69, 0xbd, 0x32, 0x43, 0x25, 0x5b, 0x2e,
	0x6e, 0x27, 0xb0, 0x22, 0x67, 0x90, 0x1a, 0x21, 0x31, 0x2e, 0x36, 0x88, 0x1e, 0x09, 0x46, 0x05,
	0x46, 0x0d, 0xce, 0x20, 0x28, 0x0f, 0x24, 0x9e, 0x91, 0x9a, 0x99, 0x9e, 0x51, 0x22, 0xc1, 0x04,
	0x11, 0x87, 0xf0, 0x9c, 0x5c, 0x4f, 0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23,
	0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18, 0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21, 0x4a,
	0x3b, 0x3d, 0xb3, 0x24, 0xa3, 0x34, 0x49, 0x2f, 0x39, 0x3f, 0x57, 0x3f, 0xa9, 0x2a, 0x55, 0x37,
	0x31, 0xa7, 0x20, 0x23, 0xb1, 0x24, 0x35, 0x11, 0xcc, 0xd3, 0xaf, 0xd0, 0x87, 0xba, 0xb0, 0xa4,
	0xb2, 0x20, 0xb5, 0x38, 0x89, 0x0d, 0xec, 0x36, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x78,
	0xb1, 0x9d, 0xe8, 0xb8, 0x00, 0x00, 0x00,
}

func (m *BurnedCoins) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BurnedCoins) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BurnedCoins) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Height) > 0 {
		i -= len(m.Height)
		copy(dAtA[i:], m.Height)
		i = encodeVarintBurnedCoins(dAtA, i, uint64(len(m.Height)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Burned) > 0 {
		i -= len(m.Burned)
		copy(dAtA[i:], m.Burned)
		i = encodeVarintBurnedCoins(dAtA, i, uint64(len(m.Burned)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintBurnedCoins(dAtA []byte, offset int, v uint64) int {
	offset -= sovBurnedCoins(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *BurnedCoins) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Burned)
	if l > 0 {
		n += 1 + l + sovBurnedCoins(uint64(l))
	}
	l = len(m.Height)
	if l > 0 {
		n += 1 + l + sovBurnedCoins(uint64(l))
	}
	return n
}

func sovBurnedCoins(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBurnedCoins(x uint64) (n int) {
	return sovBurnedCoins(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *BurnedCoins) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBurnedCoins
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
			return fmt.Errorf("proto: BurnedCoins: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BurnedCoins: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Burned", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBurnedCoins
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
				return ErrInvalidLengthBurnedCoins
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBurnedCoins
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Burned = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBurnedCoins
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
				return ErrInvalidLengthBurnedCoins
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBurnedCoins
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Height = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBurnedCoins(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBurnedCoins
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
func skipBurnedCoins(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBurnedCoins
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowBurnedCoins
				}
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
				if shift >= 64 {
					return 0, ErrIntOverflowBurnedCoins
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthBurnedCoins
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBurnedCoins
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBurnedCoins
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBurnedCoins        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBurnedCoins          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBurnedCoins = fmt.Errorf("proto: unexpected end of group")
)