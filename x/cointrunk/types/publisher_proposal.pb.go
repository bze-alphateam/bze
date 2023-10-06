// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: bze/cointrunk/v1/publisher_proposal.proto

package types

import (
	fmt "fmt"
	proto "github.com/cosmos/gogoproto/proto"
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

type PublisherProposal struct {
	Title       string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Name        string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Address     string `protobuf:"bytes,4,opt,name=address,proto3" json:"address,omitempty"`
	Active      bool   `protobuf:"varint,5,opt,name=active,proto3" json:"active,omitempty"`
}

func (m *PublisherProposal) Reset()         { *m = PublisherProposal{} }
func (m *PublisherProposal) String() string { return proto.CompactTextString(m) }
func (*PublisherProposal) ProtoMessage()    {}
func (*PublisherProposal) Descriptor() ([]byte, []int) {
	return fileDescriptor_af17600cea674632, []int{0}
}
func (m *PublisherProposal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PublisherProposal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PublisherProposal.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PublisherProposal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PublisherProposal.Merge(m, src)
}
func (m *PublisherProposal) XXX_Size() int {
	return m.Size()
}
func (m *PublisherProposal) XXX_DiscardUnknown() {
	xxx_messageInfo_PublisherProposal.DiscardUnknown(m)
}

var xxx_messageInfo_PublisherProposal proto.InternalMessageInfo

func (m *PublisherProposal) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *PublisherProposal) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *PublisherProposal) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *PublisherProposal) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *PublisherProposal) GetActive() bool {
	if m != nil {
		return m.Active
	}
	return false
}

func init() {
	proto.RegisterType((*PublisherProposal)(nil), "bze.cointrunk.v1.PublisherProposal")
}

func init() {
	proto.RegisterFile("bze/cointrunk/v1/publisher_proposal.proto", fileDescriptor_af17600cea674632)
}

var fileDescriptor_af17600cea674632 = []byte{
	// 241 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x90, 0xb1, 0x4e, 0xc3, 0x30,
	0x14, 0x45, 0x63, 0x68, 0x0b, 0x98, 0x05, 0x2c, 0x84, 0x3c, 0x59, 0x11, 0x53, 0x19, 0xb0, 0x55,
	0xf1, 0x07, 0x4c, 0x8c, 0x55, 0x47, 0x16, 0x64, 0x27, 0x4f, 0xc4, 0x22, 0x8d, 0x2d, 0xfb, 0x25,
	0x82, 0x7c, 0x05, 0xfc, 0x15, 0x63, 0x47, 0x46, 0x94, 0xfc, 0x08, 0x92, 0x69, 0x51, 0xb6, 0x77,
	0xee, 0x3d, 0xcb, 0xbb, 0xf4, 0xd6, 0xf4, 0xa0, 0x0a, 0x67, 0x1b, 0x0c, 0x6d, 0xf3, 0xaa, 0xba,
	0x95, 0xf2, 0xad, 0xa9, 0x6d, 0xac, 0x20, 0x3c, 0xfb, 0xe0, 0xbc, 0x8b, 0xba, 0x96, 0x3e, 0x38,
	0x74, 0xec, 0xc2, 0xf4, 0x20, 0xff, 0x55, 0xd9, 0xad, 0x6e, 0x3e, 0x09, 0xbd, 0x5c, 0x1f, 0xf4,
	0xf5, 0xde, 0x66, 0x57, 0x74, 0x8e, 0x16, 0x6b, 0xe0, 0x24, 0x27, 0xcb, 0xb3, 0xcd, 0x1f, 0xb0,
	0x9c, 0x9e, 0x97, 0x10, 0x8b, 0x60, 0x3d, 0x5a, 0xd7, 0xf0, 0xa3, 0xd4, 0x4d, 0x23, 0xc6, 0xe8,
	0xac, 0xd1, 0x5b, 0xe0, 0xc7, 0xa9, 0x4a, 0x37, 0xe3, 0xf4, 0x44, 0x97, 0x65, 0x80, 0x18, 0xf9,
	0x2c, 0xc5, 0x07, 0x64, 0xd7, 0x74, 0xa1, 0x0b, 0xb4, 0x1d, 0xf0, 0x79, 0x4e, 0x96, 0xa7, 0x9b,
	0x3d, 0x3d, 0x3c, 0x7e, 0x0d, 0x82, 0xec, 0x06, 0x41, 0x7e, 0x06, 0x41, 0x3e, 0x46, 0x91, 0xed,
	0x46, 0x91, 0x7d, 0x8f, 0x22, 0x7b, 0x92, 0x2f, 0x16, 0xab, 0xd6, 0xc8, 0xc2, 0x6d, 0x95, 0xe9,
	0xe1, 0x4e, 0xd7, 0xbe, 0xd2, 0x08, 0x3a, 0x91, 0x7a, 0x9b, 0xac, 0x80, 0xef, 0x1e, 0xa2, 0x59,
	0xa4, 0xb7, 0xef, 0x7f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x73, 0x89, 0x9f, 0xd6, 0x23, 0x01, 0x00,
	0x00,
}

func (m *PublisherProposal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PublisherProposal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PublisherProposal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Active {
		i--
		if m.Active {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintPublisherProposal(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintPublisherProposal(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintPublisherProposal(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintPublisherProposal(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintPublisherProposal(dAtA []byte, offset int, v uint64) int {
	offset -= sovPublisherProposal(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *PublisherProposal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovPublisherProposal(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovPublisherProposal(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovPublisherProposal(uint64(l))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovPublisherProposal(uint64(l))
	}
	if m.Active {
		n += 2
	}
	return n
}

func sovPublisherProposal(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPublisherProposal(x uint64) (n int) {
	return sovPublisherProposal(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *PublisherProposal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPublisherProposal
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
			return fmt.Errorf("proto: PublisherProposal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PublisherProposal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPublisherProposal
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
				return ErrInvalidLengthPublisherProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPublisherProposal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPublisherProposal
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
				return ErrInvalidLengthPublisherProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPublisherProposal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPublisherProposal
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
				return ErrInvalidLengthPublisherProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPublisherProposal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPublisherProposal
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
				return ErrInvalidLengthPublisherProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPublisherProposal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Active", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPublisherProposal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Active = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipPublisherProposal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthPublisherProposal
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
func skipPublisherProposal(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPublisherProposal
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
					return 0, ErrIntOverflowPublisherProposal
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
					return 0, ErrIntOverflowPublisherProposal
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
				return 0, ErrInvalidLengthPublisherProposal
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPublisherProposal
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPublisherProposal
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPublisherProposal        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPublisherProposal          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPublisherProposal = fmt.Errorf("proto: unexpected end of group")
)
