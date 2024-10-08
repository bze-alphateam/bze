// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: burner/raffle.proto

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

type Raffle struct {
	Pot         string `protobuf:"bytes,1,opt,name=pot,proto3" json:"pot,omitempty"`
	Duration    uint64 `protobuf:"varint,2,opt,name=duration,proto3" json:"duration,omitempty"`
	Chances     uint64 `protobuf:"varint,3,opt,name=chances,proto3" json:"chances,omitempty"`
	Ratio       string `protobuf:"bytes,4,opt,name=ratio,proto3" json:"ratio,omitempty"`
	EndAt       uint64 `protobuf:"varint,5,opt,name=end_at,json=endAt,proto3" json:"end_at,omitempty"`
	Winners     uint64 `protobuf:"varint,6,opt,name=winners,proto3" json:"winners,omitempty"`
	TicketPrice string `protobuf:"bytes,7,opt,name=ticket_price,json=ticketPrice,proto3" json:"ticket_price,omitempty"`
	Denom       string `protobuf:"bytes,8,opt,name=denom,proto3" json:"denom,omitempty"`
	TotalWon    string `protobuf:"bytes,9,opt,name=total_won,json=totalWon,proto3" json:"total_won,omitempty"`
}

func (m *Raffle) Reset()         { *m = Raffle{} }
func (m *Raffle) String() string { return proto.CompactTextString(m) }
func (*Raffle) ProtoMessage()    {}
func (*Raffle) Descriptor() ([]byte, []int) {
	return fileDescriptor_c0e1cbf55ea8852f, []int{0}
}
func (m *Raffle) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Raffle) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Raffle.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Raffle) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Raffle.Merge(m, src)
}
func (m *Raffle) XXX_Size() int {
	return m.Size()
}
func (m *Raffle) XXX_DiscardUnknown() {
	xxx_messageInfo_Raffle.DiscardUnknown(m)
}

var xxx_messageInfo_Raffle proto.InternalMessageInfo

func (m *Raffle) GetPot() string {
	if m != nil {
		return m.Pot
	}
	return ""
}

func (m *Raffle) GetDuration() uint64 {
	if m != nil {
		return m.Duration
	}
	return 0
}

func (m *Raffle) GetChances() uint64 {
	if m != nil {
		return m.Chances
	}
	return 0
}

func (m *Raffle) GetRatio() string {
	if m != nil {
		return m.Ratio
	}
	return ""
}

func (m *Raffle) GetEndAt() uint64 {
	if m != nil {
		return m.EndAt
	}
	return 0
}

func (m *Raffle) GetWinners() uint64 {
	if m != nil {
		return m.Winners
	}
	return 0
}

func (m *Raffle) GetTicketPrice() string {
	if m != nil {
		return m.TicketPrice
	}
	return ""
}

func (m *Raffle) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func (m *Raffle) GetTotalWon() string {
	if m != nil {
		return m.TotalWon
	}
	return ""
}

type RaffleDeleteHook struct {
	Denom string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	EndAt uint64 `protobuf:"varint,2,opt,name=end_at,json=endAt,proto3" json:"end_at,omitempty"`
}

func (m *RaffleDeleteHook) Reset()         { *m = RaffleDeleteHook{} }
func (m *RaffleDeleteHook) String() string { return proto.CompactTextString(m) }
func (*RaffleDeleteHook) ProtoMessage()    {}
func (*RaffleDeleteHook) Descriptor() ([]byte, []int) {
	return fileDescriptor_c0e1cbf55ea8852f, []int{1}
}
func (m *RaffleDeleteHook) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaffleDeleteHook) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaffleDeleteHook.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RaffleDeleteHook) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaffleDeleteHook.Merge(m, src)
}
func (m *RaffleDeleteHook) XXX_Size() int {
	return m.Size()
}
func (m *RaffleDeleteHook) XXX_DiscardUnknown() {
	xxx_messageInfo_RaffleDeleteHook.DiscardUnknown(m)
}

var xxx_messageInfo_RaffleDeleteHook proto.InternalMessageInfo

func (m *RaffleDeleteHook) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func (m *RaffleDeleteHook) GetEndAt() uint64 {
	if m != nil {
		return m.EndAt
	}
	return 0
}

type RaffleWinner struct {
	Index  string `protobuf:"bytes,1,opt,name=index,proto3" json:"index,omitempty"`
	Denom  string `protobuf:"bytes,2,opt,name=denom,proto3" json:"denom,omitempty"`
	Amount string `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Winner string `protobuf:"bytes,4,opt,name=winner,proto3" json:"winner,omitempty"`
}

func (m *RaffleWinner) Reset()         { *m = RaffleWinner{} }
func (m *RaffleWinner) String() string { return proto.CompactTextString(m) }
func (*RaffleWinner) ProtoMessage()    {}
func (*RaffleWinner) Descriptor() ([]byte, []int) {
	return fileDescriptor_c0e1cbf55ea8852f, []int{2}
}
func (m *RaffleWinner) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaffleWinner) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaffleWinner.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RaffleWinner) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaffleWinner.Merge(m, src)
}
func (m *RaffleWinner) XXX_Size() int {
	return m.Size()
}
func (m *RaffleWinner) XXX_DiscardUnknown() {
	xxx_messageInfo_RaffleWinner.DiscardUnknown(m)
}

var xxx_messageInfo_RaffleWinner proto.InternalMessageInfo

func (m *RaffleWinner) GetIndex() string {
	if m != nil {
		return m.Index
	}
	return ""
}

func (m *RaffleWinner) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func (m *RaffleWinner) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

func (m *RaffleWinner) GetWinner() string {
	if m != nil {
		return m.Winner
	}
	return ""
}

type RaffleParticipant struct {
	Index       uint64 `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Denom       string `protobuf:"bytes,2,opt,name=denom,proto3" json:"denom,omitempty"`
	Participant string `protobuf:"bytes,3,opt,name=participant,proto3" json:"participant,omitempty"`
	ExecuteAt   int64  `protobuf:"varint,4,opt,name=execute_at,json=executeAt,proto3" json:"execute_at,omitempty"`
}

func (m *RaffleParticipant) Reset()         { *m = RaffleParticipant{} }
func (m *RaffleParticipant) String() string { return proto.CompactTextString(m) }
func (*RaffleParticipant) ProtoMessage()    {}
func (*RaffleParticipant) Descriptor() ([]byte, []int) {
	return fileDescriptor_c0e1cbf55ea8852f, []int{3}
}
func (m *RaffleParticipant) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RaffleParticipant) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RaffleParticipant.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RaffleParticipant) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaffleParticipant.Merge(m, src)
}
func (m *RaffleParticipant) XXX_Size() int {
	return m.Size()
}
func (m *RaffleParticipant) XXX_DiscardUnknown() {
	xxx_messageInfo_RaffleParticipant.DiscardUnknown(m)
}

var xxx_messageInfo_RaffleParticipant proto.InternalMessageInfo

func (m *RaffleParticipant) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *RaffleParticipant) GetDenom() string {
	if m != nil {
		return m.Denom
	}
	return ""
}

func (m *RaffleParticipant) GetParticipant() string {
	if m != nil {
		return m.Participant
	}
	return ""
}

func (m *RaffleParticipant) GetExecuteAt() int64 {
	if m != nil {
		return m.ExecuteAt
	}
	return 0
}

func init() {
	proto.RegisterType((*Raffle)(nil), "bze.burner.v1.Raffle")
	proto.RegisterType((*RaffleDeleteHook)(nil), "bze.burner.v1.RaffleDeleteHook")
	proto.RegisterType((*RaffleWinner)(nil), "bze.burner.v1.RaffleWinner")
	proto.RegisterType((*RaffleParticipant)(nil), "bze.burner.v1.RaffleParticipant")
}

func init() { proto.RegisterFile("burner/raffle.proto", fileDescriptor_c0e1cbf55ea8852f) }

var fileDescriptor_c0e1cbf55ea8852f = []byte{
	// 405 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x92, 0xcf, 0x8e, 0xd3, 0x30,
	0x10, 0xc6, 0xeb, 0xfe, 0xc9, 0x36, 0xb3, 0x8b, 0xb4, 0x98, 0x3f, 0xb2, 0x40, 0x44, 0xa5, 0xa7,
	0x95, 0x10, 0x8d, 0x10, 0x0f, 0x80, 0x16, 0x81, 0xc4, 0x71, 0x95, 0xcb, 0x4a, 0x5c, 0x2a, 0x27,
	0x99, 0xa5, 0x66, 0x53, 0x3b, 0x72, 0x27, 0x6c, 0xd9, 0x13, 0x8f, 0xc0, 0x63, 0x71, 0xdc, 0x23,
	0x47, 0xd4, 0x9e, 0x79, 0x07, 0x64, 0x3b, 0x25, 0xe1, 0xc2, 0xcd, 0xdf, 0x37, 0x9e, 0xdf, 0x37,
	0x63, 0x19, 0x1e, 0xe4, 0x8d, 0xd5, 0x68, 0x53, 0x2b, 0xaf, 0xae, 0x2a, 0x5c, 0xd4, 0xd6, 0x90,
	0xe1, 0xf7, 0xf2, 0x5b, 0x5c, 0x84, 0xc2, 0xe2, 0xcb, 0xab, 0xf9, 0x6f, 0x06, 0x51, 0xe6, 0xeb,
	0xfc, 0x14, 0x46, 0xb5, 0x21, 0xc1, 0x66, 0xec, 0x2c, 0xce, 0xdc, 0x91, 0x3f, 0x81, 0x69, 0xd9,
	0x58, 0x49, 0xca, 0x68, 0x31, 0x9c, 0xb1, 0xb3, 0x71, 0xf6, 0x57, 0x73, 0x01, 0x47, 0xc5, 0x4a,
	0xea, 0x02, 0x37, 0x62, 0xe4, 0x4b, 0x07, 0xc9, 0x1f, 0xc2, 0xc4, 0xdf, 0x11, 0x63, 0x4f, 0x0a,
	0x82, 0x3f, 0x82, 0x08, 0x75, 0xb9, 0x94, 0x24, 0x26, 0xfe, 0xfa, 0x04, 0x75, 0x79, 0x4e, 0x0e,
	0x73, 0xa3, 0xb4, 0x46, 0xbb, 0x11, 0x51, 0xc0, 0xb4, 0x92, 0x3f, 0x87, 0x13, 0x52, 0xc5, 0x35,
	0xd2, 0xb2, 0xb6, 0xaa, 0x40, 0x71, 0xe4, 0x69, 0xc7, 0xc1, 0xbb, 0x70, 0x96, 0x4b, 0x2a, 0x51,
	0x9b, 0xb5, 0x98, 0x86, 0x24, 0x2f, 0xf8, 0x53, 0x88, 0xc9, 0x90, 0xac, 0x96, 0x37, 0x46, 0x8b,
	0xd8, 0x57, 0xa6, 0xde, 0xb8, 0x34, 0x7a, 0xfe, 0x06, 0x4e, 0xc3, 0xba, 0xef, 0xb0, 0x42, 0xc2,
	0x0f, 0xc6, 0x5c, 0x77, 0x18, 0xd6, 0xc7, 0x74, 0x03, 0x0f, 0x7b, 0x03, 0xcf, 0x3f, 0xc3, 0x49,
	0x00, 0x5c, 0xfa, 0x39, 0x5d, 0xb3, 0xd2, 0x25, 0x6e, 0x0f, 0xcd, 0x5e, 0x74, 0xc8, 0x61, 0x1f,
	0xf9, 0x18, 0x22, 0xb9, 0x36, 0x8d, 0x26, 0xff, 0x64, 0x71, 0xd6, 0x2a, 0xe7, 0x87, 0xad, 0xdb,
	0x27, 0x6b, 0xd5, 0xfc, 0x1b, 0x83, 0xfb, 0x21, 0xec, 0x42, 0x5a, 0x52, 0x85, 0xaa, 0xa5, 0xa6,
	0x7f, 0x13, 0xc7, 0xff, 0x4f, 0x9c, 0xc1, 0x71, 0xdd, 0xb5, 0xb6, 0xb1, 0x7d, 0x8b, 0x3f, 0x03,
	0xc0, 0x2d, 0x16, 0x0d, 0xa1, 0x5b, 0xd5, 0xe5, 0x8f, 0xb2, 0xb8, 0x75, 0xce, 0xe9, 0xed, 0xfb,
	0x1f, 0xbb, 0x84, 0xdd, 0xed, 0x12, 0xf6, 0x6b, 0x97, 0xb0, 0xef, 0xfb, 0x64, 0x70, 0xb7, 0x4f,
	0x06, 0x3f, 0xf7, 0xc9, 0xe0, 0xe3, 0x8b, 0x4f, 0x8a, 0x56, 0x4d, 0xbe, 0x28, 0xcc, 0x3a, 0xcd,
	0x6f, 0xf1, 0xa5, 0xac, 0xea, 0x95, 0x24, 0x94, 0x5e, 0xa5, 0xdb, 0xb4, 0xfd, 0x7c, 0xf4, 0xb5,
	0xc6, 0x4d, 0x1e, 0xf9, 0xcf, 0xf7, 0xfa, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x66, 0x3c, 0xe4,
	0xd8, 0x93, 0x02, 0x00, 0x00,
}

func (m *Raffle) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Raffle) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Raffle) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.TotalWon) > 0 {
		i -= len(m.TotalWon)
		copy(dAtA[i:], m.TotalWon)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.TotalWon)))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.TicketPrice) > 0 {
		i -= len(m.TicketPrice)
		copy(dAtA[i:], m.TicketPrice)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.TicketPrice)))
		i--
		dAtA[i] = 0x3a
	}
	if m.Winners != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.Winners))
		i--
		dAtA[i] = 0x30
	}
	if m.EndAt != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.EndAt))
		i--
		dAtA[i] = 0x28
	}
	if len(m.Ratio) > 0 {
		i -= len(m.Ratio)
		copy(dAtA[i:], m.Ratio)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Ratio)))
		i--
		dAtA[i] = 0x22
	}
	if m.Chances != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.Chances))
		i--
		dAtA[i] = 0x18
	}
	if m.Duration != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.Duration))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Pot) > 0 {
		i -= len(m.Pot)
		copy(dAtA[i:], m.Pot)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Pot)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *RaffleDeleteHook) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaffleDeleteHook) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RaffleDeleteHook) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.EndAt != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.EndAt))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *RaffleWinner) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaffleWinner) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RaffleWinner) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Winner) > 0 {
		i -= len(m.Winner)
		copy(dAtA[i:], m.Winner)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Winner)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Amount) > 0 {
		i -= len(m.Amount)
		copy(dAtA[i:], m.Amount)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Amount)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Index) > 0 {
		i -= len(m.Index)
		copy(dAtA[i:], m.Index)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Index)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *RaffleParticipant) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RaffleParticipant) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RaffleParticipant) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ExecuteAt != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.ExecuteAt))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Participant) > 0 {
		i -= len(m.Participant)
		copy(dAtA[i:], m.Participant)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Participant)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Denom) > 0 {
		i -= len(m.Denom)
		copy(dAtA[i:], m.Denom)
		i = encodeVarintRaffle(dAtA, i, uint64(len(m.Denom)))
		i--
		dAtA[i] = 0x12
	}
	if m.Index != 0 {
		i = encodeVarintRaffle(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintRaffle(dAtA []byte, offset int, v uint64) int {
	offset -= sovRaffle(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Raffle) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Pot)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	if m.Duration != 0 {
		n += 1 + sovRaffle(uint64(m.Duration))
	}
	if m.Chances != 0 {
		n += 1 + sovRaffle(uint64(m.Chances))
	}
	l = len(m.Ratio)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	if m.EndAt != 0 {
		n += 1 + sovRaffle(uint64(m.EndAt))
	}
	if m.Winners != 0 {
		n += 1 + sovRaffle(uint64(m.Winners))
	}
	l = len(m.TicketPrice)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.TotalWon)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	return n
}

func (m *RaffleDeleteHook) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	if m.EndAt != 0 {
		n += 1 + sovRaffle(uint64(m.EndAt))
	}
	return n
}

func (m *RaffleWinner) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Index)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.Amount)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.Winner)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	return n
}

func (m *RaffleParticipant) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Index != 0 {
		n += 1 + sovRaffle(uint64(m.Index))
	}
	l = len(m.Denom)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	l = len(m.Participant)
	if l > 0 {
		n += 1 + l + sovRaffle(uint64(l))
	}
	if m.ExecuteAt != 0 {
		n += 1 + sovRaffle(uint64(m.ExecuteAt))
	}
	return n
}

func sovRaffle(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozRaffle(x uint64) (n int) {
	return sovRaffle(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Raffle) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaffle
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
			return fmt.Errorf("proto: Raffle: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Raffle: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pot", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Pot = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Duration", wireType)
			}
			m.Duration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Duration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Chances", wireType)
			}
			m.Chances = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Chances |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Ratio", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Ratio = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndAt", wireType)
			}
			m.EndAt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.EndAt |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Winners", wireType)
			}
			m.Winners = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Winners |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TicketPrice", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TicketPrice = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalWon", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TotalWon = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaffle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRaffle
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
func (m *RaffleDeleteHook) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaffle
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
			return fmt.Errorf("proto: RaffleDeleteHook: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaffleDeleteHook: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndAt", wireType)
			}
			m.EndAt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.EndAt |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipRaffle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRaffle
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
func (m *RaffleWinner) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaffle
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
			return fmt.Errorf("proto: RaffleWinner: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaffleWinner: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Index = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Amount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Winner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Winner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaffle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRaffle
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
func (m *RaffleParticipant) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaffle
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
			return fmt.Errorf("proto: RaffleParticipant: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaffleParticipant: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Denom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Participant", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
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
				return ErrInvalidLengthRaffle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthRaffle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Participant = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ExecuteAt", wireType)
			}
			m.ExecuteAt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaffle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ExecuteAt |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipRaffle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthRaffle
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
func skipRaffle(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRaffle
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
					return 0, ErrIntOverflowRaffle
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
					return 0, ErrIntOverflowRaffle
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
				return 0, ErrInvalidLengthRaffle
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupRaffle
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthRaffle
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthRaffle        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRaffle          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupRaffle = fmt.Errorf("proto: unexpected end of group")
)
