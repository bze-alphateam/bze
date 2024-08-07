// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: tradebin/queue_message.proto

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

type QueueMessage struct {
	MessageId   string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	MarketId    string `protobuf:"bytes,2,opt,name=market_id,json=marketId,proto3" json:"market_id,omitempty"`
	MessageType string `protobuf:"bytes,3,opt,name=message_type,json=messageType,proto3" json:"message_type,omitempty"`
	Amount      string `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`
	Price       string `protobuf:"bytes,5,opt,name=price,proto3" json:"price,omitempty"`
	CreatedAt   int64  `protobuf:"varint,6,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	OrderId     string `protobuf:"bytes,7,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	OrderType   string `protobuf:"bytes,8,opt,name=order_type,json=orderType,proto3" json:"order_type,omitempty"`
	Owner       string `protobuf:"bytes,9,opt,name=owner,proto3" json:"owner,omitempty"`
}

func (m *QueueMessage) Reset()         { *m = QueueMessage{} }
func (m *QueueMessage) String() string { return proto.CompactTextString(m) }
func (*QueueMessage) ProtoMessage()    {}
func (*QueueMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_ea2e5e8e6d6aeac7, []int{0}
}
func (m *QueueMessage) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueueMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueueMessage.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueueMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueueMessage.Merge(m, src)
}
func (m *QueueMessage) XXX_Size() int {
	return m.Size()
}
func (m *QueueMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_QueueMessage.DiscardUnknown(m)
}

var xxx_messageInfo_QueueMessage proto.InternalMessageInfo

func (m *QueueMessage) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

func (m *QueueMessage) GetMarketId() string {
	if m != nil {
		return m.MarketId
	}
	return ""
}

func (m *QueueMessage) GetMessageType() string {
	if m != nil {
		return m.MessageType
	}
	return ""
}

func (m *QueueMessage) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

func (m *QueueMessage) GetPrice() string {
	if m != nil {
		return m.Price
	}
	return ""
}

func (m *QueueMessage) GetCreatedAt() int64 {
	if m != nil {
		return m.CreatedAt
	}
	return 0
}

func (m *QueueMessage) GetOrderId() string {
	if m != nil {
		return m.OrderId
	}
	return ""
}

func (m *QueueMessage) GetOrderType() string {
	if m != nil {
		return m.OrderType
	}
	return ""
}

func (m *QueueMessage) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func init() {
	proto.RegisterType((*QueueMessage)(nil), "bze.tradebin.v1.QueueMessage")
}

func init() { proto.RegisterFile("tradebin/queue_message.proto", fileDescriptor_ea2e5e8e6d6aeac7) }

var fileDescriptor_ea2e5e8e6d6aeac7 = []byte{
	// 297 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x3c, 0x90, 0xbf, 0x4e, 0xc3, 0x30,
	0x18, 0xc4, 0xeb, 0x96, 0xfe, 0x89, 0xa9, 0x84, 0x64, 0x55, 0xc8, 0x08, 0xb0, 0x0a, 0x53, 0x97,
	0x26, 0x42, 0x3c, 0x01, 0x2c, 0xa8, 0x03, 0x03, 0x15, 0x13, 0x4b, 0xe5, 0xd4, 0x9f, 0xda, 0x08,
	0x12, 0x07, 0xc7, 0x01, 0xda, 0x47, 0x60, 0xe2, 0xb1, 0x18, 0x3b, 0x32, 0xa2, 0xe4, 0x45, 0x90,
	0x3f, 0xa7, 0x8c, 0xf7, 0xbb, 0x3b, 0xf9, 0xfc, 0xd1, 0x33, 0x6b, 0xa4, 0x82, 0x38, 0xc9, 0xa2,
	0xd7, 0x12, 0x4a, 0x58, 0xa4, 0x50, 0x14, 0x72, 0x05, 0x61, 0x6e, 0xb4, 0xd5, 0xec, 0x28, 0xde,
	0x42, 0xb8, 0x4f, 0x84, 0x6f, 0x57, 0x97, 0x9f, 0x6d, 0x3a, 0x7c, 0x70, 0xc1, 0x7b, 0x9f, 0x63,
	0xe7, 0x94, 0x36, 0x95, 0x45, 0xa2, 0x38, 0x19, 0x93, 0x49, 0x30, 0x0f, 0x1a, 0x32, 0x53, 0xec,
	0x94, 0x06, 0xa9, 0x34, 0xcf, 0x60, 0x9d, 0xdb, 0x46, 0x77, 0xe0, 0xc1, 0x4c, 0xb1, 0x0b, 0x3a,
	0xdc, 0x77, 0xed, 0x26, 0x07, 0xde, 0x41, 0xff, 0xb0, 0x61, 0x8f, 0x9b, 0x1c, 0xd8, 0x31, 0xed,
	0xc9, 0x54, 0x97, 0x99, 0xe5, 0x07, 0x68, 0x36, 0x8a, 0x8d, 0x68, 0x37, 0x37, 0xc9, 0x12, 0x78,
	0x17, 0xb1, 0x17, 0x6e, 0xcc, 0xd2, 0x80, 0xb4, 0xa0, 0x16, 0xd2, 0xf2, 0xde, 0x98, 0x4c, 0x3a,
	0xf3, 0xa0, 0x21, 0x37, 0x96, 0x9d, 0xd0, 0x81, 0x36, 0x0a, 0x8c, 0xdb, 0xd2, 0xc7, 0x5e, 0x1f,
	0xf5, 0x4c, 0xb9, 0xa6, 0xb7, 0x70, 0xc8, 0xc0, 0x7f, 0x03, 0x09, 0xce, 0x18, 0xd1, 0xae, 0x7e,
	0xcf, 0xc0, 0xf0, 0xc0, 0x3f, 0x87, 0xe2, 0xf6, 0xee, 0xbb, 0x12, 0x64, 0x57, 0x09, 0xf2, 0x5b,
	0x09, 0xf2, 0x55, 0x8b, 0xd6, 0xae, 0x16, 0xad, 0x9f, 0x5a, 0xb4, 0x9e, 0xa6, 0xab, 0xc4, 0xae,
	0xcb, 0x38, 0x5c, 0xea, 0x34, 0x8a, 0xb7, 0x30, 0x95, 0x2f, 0xf9, 0x5a, 0x5a, 0x90, 0xa8, 0xa2,
	0x8f, 0xe8, 0xff, 0xe8, 0xee, 0xb9, 0x22, 0xee, 0xe1, 0xb5, 0xaf, 0xff, 0x02, 0x00, 0x00, 0xff,
	0xff, 0x6a, 0xea, 0x7a, 0x40, 0x8d, 0x01, 0x00, 0x00,
}

func (m *QueueMessage) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueueMessage) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueueMessage) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Owner) > 0 {
		i -= len(m.Owner)
		copy(dAtA[i:], m.Owner)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.Owner)))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.OrderType) > 0 {
		i -= len(m.OrderType)
		copy(dAtA[i:], m.OrderType)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.OrderType)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.OrderId) > 0 {
		i -= len(m.OrderId)
		copy(dAtA[i:], m.OrderId)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.OrderId)))
		i--
		dAtA[i] = 0x3a
	}
	if m.CreatedAt != 0 {
		i = encodeVarintQueueMessage(dAtA, i, uint64(m.CreatedAt))
		i--
		dAtA[i] = 0x30
	}
	if len(m.Price) > 0 {
		i -= len(m.Price)
		copy(dAtA[i:], m.Price)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.Price)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Amount) > 0 {
		i -= len(m.Amount)
		copy(dAtA[i:], m.Amount)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.Amount)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.MessageType) > 0 {
		i -= len(m.MessageType)
		copy(dAtA[i:], m.MessageType)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.MessageType)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.MarketId) > 0 {
		i -= len(m.MarketId)
		copy(dAtA[i:], m.MarketId)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.MarketId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.MessageId) > 0 {
		i -= len(m.MessageId)
		copy(dAtA[i:], m.MessageId)
		i = encodeVarintQueueMessage(dAtA, i, uint64(len(m.MessageId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintQueueMessage(dAtA []byte, offset int, v uint64) int {
	offset -= sovQueueMessage(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *QueueMessage) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.MessageId)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.MarketId)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.MessageType)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.Amount)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.Price)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	if m.CreatedAt != 0 {
		n += 1 + sovQueueMessage(uint64(m.CreatedAt))
	}
	l = len(m.OrderId)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.OrderType)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	l = len(m.Owner)
	if l > 0 {
		n += 1 + l + sovQueueMessage(uint64(l))
	}
	return n
}

func sovQueueMessage(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQueueMessage(x uint64) (n int) {
	return sovQueueMessage(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *QueueMessage) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQueueMessage
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
			return fmt.Errorf("proto: QueueMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueueMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MessageId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MessageId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MarketId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MarketId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MessageType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MessageType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Amount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Price", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Price = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedAt", wireType)
			}
			m.CreatedAt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreatedAt |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OrderId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OrderId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OrderType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OrderType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Owner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueueMessage
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
				return ErrInvalidLengthQueueMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueueMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Owner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQueueMessage(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQueueMessage
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
func skipQueueMessage(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQueueMessage
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
					return 0, ErrIntOverflowQueueMessage
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
					return 0, ErrIntOverflowQueueMessage
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
				return 0, ErrInvalidLengthQueueMessage
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQueueMessage
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQueueMessage
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQueueMessage        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQueueMessage          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQueueMessage = fmt.Errorf("proto: unexpected end of group")
)
