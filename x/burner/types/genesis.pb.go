// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: burner/genesis.proto

package types

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
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

// GenesisState defines the burner module's genesis state.
type GenesisState struct {
	Params                 Params              `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	BurnedCoinsList        []BurnedCoins       `protobuf:"bytes,2,rep,name=burned_coins_list,json=burnedCoinsList,proto3" json:"burned_coins_list,omitempty"`
	RaffleList             []Raffle            `protobuf:"bytes,3,rep,name=raffle_list,json=raffleList,proto3" json:"raffle_list,omitempty"`
	RaffleWinnersList      []RaffleWinner      `protobuf:"bytes,4,rep,name=raffle_winners_list,json=raffleWinnersList,proto3" json:"raffle_winners_list,omitempty"`
	RaffleParticipantsList []RaffleParticipant `protobuf:"bytes,5,rep,name=raffle_participants_list,json=raffleParticipantsList,proto3" json:"raffle_participants_list,omitempty"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_62cceffcaad9705b, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

func (m *GenesisState) GetBurnedCoinsList() []BurnedCoins {
	if m != nil {
		return m.BurnedCoinsList
	}
	return nil
}

func (m *GenesisState) GetRaffleList() []Raffle {
	if m != nil {
		return m.RaffleList
	}
	return nil
}

func (m *GenesisState) GetRaffleWinnersList() []RaffleWinner {
	if m != nil {
		return m.RaffleWinnersList
	}
	return nil
}

func (m *GenesisState) GetRaffleParticipantsList() []RaffleParticipant {
	if m != nil {
		return m.RaffleParticipantsList
	}
	return nil
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "bze.burner.v1.GenesisState")
}

func init() { proto.RegisterFile("burner/genesis.proto", fileDescriptor_62cceffcaad9705b) }

var fileDescriptor_62cceffcaad9705b = []byte{
	// 384 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xcf, 0x4e, 0xea, 0x40,
	0x14, 0xc6, 0xdb, 0x0b, 0x97, 0xc5, 0x70, 0x6f, 0x6e, 0x28, 0x70, 0x83, 0x25, 0x14, 0x82, 0x31,
	0x21, 0xfe, 0x69, 0x23, 0xbc, 0xc1, 0x18, 0xe3, 0xc6, 0x05, 0xc1, 0x18, 0x13, 0x37, 0x64, 0x8a,
	0x43, 0x99, 0x48, 0x3b, 0x93, 0xe9, 0xa0, 0xc2, 0xda, 0xb8, 0xf6, 0xb1, 0x58, 0xb2, 0x74, 0x45,
	0x0c, 0xec, 0x7c, 0x0a, 0xc3, 0xcc, 0x24, 0xd4, 0x8a, 0xab, 0x76, 0xce, 0xf9, 0xbe, 0xef, 0x77,
	0x4e, 0x72, 0x40, 0xc9, 0x9f, 0xf0, 0x08, 0x73, 0x2f, 0xc0, 0x11, 0x8e, 0x49, 0xec, 0x32, 0x4e,
	0x05, 0xb5, 0xfe, 0xfa, 0x33, 0xec, 0xaa, 0x8e, 0xfb, 0x70, 0x6a, 0x97, 0x02, 0x1a, 0x50, 0xd9,
	0xf1, 0x36, 0x7f, 0x4a, 0x64, 0x17, 0xb5, 0x95, 0x21, 0x8e, 0x42, 0xed, 0xb4, 0xf7, 0x74, 0x51,
	0x7e, 0xee, 0xfa, 0x03, 0x4a, 0xa2, 0x38, 0xa5, 0xe7, 0x68, 0x38, 0x1c, 0x63, 0x55, 0x6c, 0x3e,
	0x67, 0xc1, 0x9f, 0x0b, 0xc5, 0xbe, 0x12, 0x48, 0x60, 0xab, 0x03, 0x72, 0x2a, 0xb0, 0x62, 0x36,
	0xcc, 0x56, 0xbe, 0x5d, 0x76, 0xbf, 0xcc, 0xe2, 0x76, 0x65, 0x13, 0x66, 0xe7, 0xcb, 0xba, 0xd1,
	0xd3, 0x52, 0xeb, 0x1e, 0x14, 0x92, 0xc0, 0xfe, 0x98, 0xc4, 0xa2, 0xf2, 0xab, 0x91, 0x69, 0xe5,
	0xdb, 0x76, 0xca, 0x0f, 0xa5, 0xee, 0x6c, 0x23, 0x83, 0xfb, 0x9b, 0x90, 0x8f, 0x65, 0xbd, 0xfa,
	0xcd, 0x7c, 0x4c, 0x43, 0x22, 0x70, 0xc8, 0xc4, 0xb4, 0xf7, 0xcf, 0xdf, 0x3a, 0x2e, 0x49, 0x2c,
	0xac, 0x6b, 0x90, 0x57, 0x2b, 0x28, 0x4c, 0x46, 0x62, 0xd2, 0x63, 0xf6, 0xa4, 0x02, 0xd6, 0x34,
	0xa1, 0x9c, 0x70, 0x24, 0xb2, 0x81, 0x2a, 0xcb, 0x58, 0x0e, 0x8a, 0x5a, 0xf4, 0x48, 0xa2, 0x08,
	0x73, 0xbd, 0x45, 0x56, 0xc6, 0x57, 0x77, 0xc6, 0xdf, 0x48, 0x21, 0x3c, 0xd0, 0x90, 0xda, 0x0e,
	0x7f, 0x02, 0x56, 0xe0, 0x09, 0x93, 0x5a, 0xe5, 0xc5, 0x04, 0x15, 0x6d, 0x62, 0x88, 0x0b, 0x32,
	0x20, 0x0c, 0x45, 0x42, 0x93, 0x7f, 0x4b, 0x72, 0x63, 0x27, 0xb9, 0xbb, 0x55, 0xc3, 0x43, 0x8d,
	0x6f, 0xfe, 0x94, 0x94, 0x98, 0xe1, 0x3f, 0x4f, 0xdb, 0xe5, 0x20, 0xf0, 0x7c, 0xbe, 0x72, 0xcc,
	0xc5, 0xca, 0x31, 0xdf, 0x57, 0x8e, 0xf9, 0xba, 0x76, 0x8c, 0xc5, 0xda, 0x31, 0xde, 0xd6, 0x8e,
	0x71, 0x7b, 0x14, 0x10, 0x31, 0x9a, 0xf8, 0xee, 0x80, 0x86, 0x9e, 0x3f, 0xc3, 0x27, 0x68, 0xcc,
	0x46, 0x48, 0x60, 0x24, 0x5f, 0xde, 0x93, 0xa7, 0x8f, 0x4a, 0x4c, 0x19, 0x8e, 0xfd, 0x9c, 0x3c,
	0xaa, 0xce, 0x67, 0x00, 0x00, 0x00, 0xff, 0xff, 0x41, 0x9f, 0x3e, 0x85, 0xd6, 0x02, 0x00, 0x00,
}

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
	_ = i
	var l int
	_ = l
	if len(m.RaffleParticipantsList) > 0 {
		for iNdEx := len(m.RaffleParticipantsList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.RaffleParticipantsList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.RaffleWinnersList) > 0 {
		for iNdEx := len(m.RaffleWinnersList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.RaffleWinnersList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.RaffleList) > 0 {
		for iNdEx := len(m.RaffleList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.RaffleList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.BurnedCoinsList) > 0 {
		for iNdEx := len(m.BurnedCoinsList) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.BurnedCoinsList[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	if len(m.BurnedCoinsList) > 0 {
		for _, e := range m.BurnedCoinsList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.RaffleList) > 0 {
		for _, e := range m.RaffleList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.RaffleWinnersList) > 0 {
		for _, e := range m.RaffleWinnersList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.RaffleParticipantsList) > 0 {
		for _, e := range m.RaffleParticipantsList {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BurnedCoinsList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BurnedCoinsList = append(m.BurnedCoinsList, BurnedCoins{})
			if err := m.BurnedCoinsList[len(m.BurnedCoinsList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RaffleList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RaffleList = append(m.RaffleList, Raffle{})
			if err := m.RaffleList[len(m.RaffleList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RaffleWinnersList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RaffleWinnersList = append(m.RaffleWinnersList, RaffleWinner{})
			if err := m.RaffleWinnersList[len(m.RaffleWinnersList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RaffleParticipantsList", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RaffleParticipantsList = append(m.RaffleParticipantsList, RaffleParticipant{})
			if err := m.RaffleParticipantsList[len(m.RaffleParticipantsList)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
