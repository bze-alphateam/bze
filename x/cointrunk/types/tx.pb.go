// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: cointrunk/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type MsgAddArticle struct {
	Publisher string `protobuf:"bytes,1,opt,name=publisher,proto3" json:"publisher,omitempty"`
	Title     string `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
	Url       string `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty"`
	Picture   string `protobuf:"bytes,4,opt,name=picture,proto3" json:"picture,omitempty"`
}

func (m *MsgAddArticle) Reset()         { *m = MsgAddArticle{} }
func (m *MsgAddArticle) String() string { return proto.CompactTextString(m) }
func (*MsgAddArticle) ProtoMessage()    {}
func (*MsgAddArticle) Descriptor() ([]byte, []int) {
	return fileDescriptor_776bb1586e9b1fcd, []int{0}
}
func (m *MsgAddArticle) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgAddArticle) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgAddArticle.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgAddArticle) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgAddArticle.Merge(m, src)
}
func (m *MsgAddArticle) XXX_Size() int {
	return m.Size()
}
func (m *MsgAddArticle) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgAddArticle.DiscardUnknown(m)
}

var xxx_messageInfo_MsgAddArticle proto.InternalMessageInfo

func (m *MsgAddArticle) GetPublisher() string {
	if m != nil {
		return m.Publisher
	}
	return ""
}

func (m *MsgAddArticle) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *MsgAddArticle) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

func (m *MsgAddArticle) GetPicture() string {
	if m != nil {
		return m.Picture
	}
	return ""
}

type MsgAddArticleResponse struct {
}

func (m *MsgAddArticleResponse) Reset()         { *m = MsgAddArticleResponse{} }
func (m *MsgAddArticleResponse) String() string { return proto.CompactTextString(m) }
func (*MsgAddArticleResponse) ProtoMessage()    {}
func (*MsgAddArticleResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_776bb1586e9b1fcd, []int{1}
}
func (m *MsgAddArticleResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgAddArticleResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgAddArticleResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgAddArticleResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgAddArticleResponse.Merge(m, src)
}
func (m *MsgAddArticleResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgAddArticleResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgAddArticleResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgAddArticleResponse proto.InternalMessageInfo

type MsgPayPublisherRespect struct {
	Creator string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	Address string `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	Amount  string `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (m *MsgPayPublisherRespect) Reset()         { *m = MsgPayPublisherRespect{} }
func (m *MsgPayPublisherRespect) String() string { return proto.CompactTextString(m) }
func (*MsgPayPublisherRespect) ProtoMessage()    {}
func (*MsgPayPublisherRespect) Descriptor() ([]byte, []int) {
	return fileDescriptor_776bb1586e9b1fcd, []int{2}
}
func (m *MsgPayPublisherRespect) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgPayPublisherRespect) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgPayPublisherRespect.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgPayPublisherRespect) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgPayPublisherRespect.Merge(m, src)
}
func (m *MsgPayPublisherRespect) XXX_Size() int {
	return m.Size()
}
func (m *MsgPayPublisherRespect) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgPayPublisherRespect.DiscardUnknown(m)
}

var xxx_messageInfo_MsgPayPublisherRespect proto.InternalMessageInfo

func (m *MsgPayPublisherRespect) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *MsgPayPublisherRespect) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *MsgPayPublisherRespect) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

type MsgPayPublisherRespectResponse struct {
	RespectPaid        uint64 `protobuf:"varint,1,opt,name=respect_paid,json=respectPaid,proto3" json:"respect_paid,omitempty"`
	PublisherReward    uint64 `protobuf:"varint,2,opt,name=publisher_reward,json=publisherReward,proto3" json:"publisher_reward,omitempty"`
	CommunityPoolFunds uint64 `protobuf:"varint,3,opt,name=community_pool_funds,json=communityPoolFunds,proto3" json:"community_pool_funds,omitempty"`
}

func (m *MsgPayPublisherRespectResponse) Reset()         { *m = MsgPayPublisherRespectResponse{} }
func (m *MsgPayPublisherRespectResponse) String() string { return proto.CompactTextString(m) }
func (*MsgPayPublisherRespectResponse) ProtoMessage()    {}
func (*MsgPayPublisherRespectResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_776bb1586e9b1fcd, []int{3}
}
func (m *MsgPayPublisherRespectResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgPayPublisherRespectResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgPayPublisherRespectResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgPayPublisherRespectResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgPayPublisherRespectResponse.Merge(m, src)
}
func (m *MsgPayPublisherRespectResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgPayPublisherRespectResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgPayPublisherRespectResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgPayPublisherRespectResponse proto.InternalMessageInfo

func (m *MsgPayPublisherRespectResponse) GetRespectPaid() uint64 {
	if m != nil {
		return m.RespectPaid
	}
	return 0
}

func (m *MsgPayPublisherRespectResponse) GetPublisherReward() uint64 {
	if m != nil {
		return m.PublisherReward
	}
	return 0
}

func (m *MsgPayPublisherRespectResponse) GetCommunityPoolFunds() uint64 {
	if m != nil {
		return m.CommunityPoolFunds
	}
	return 0
}

func init() {
	proto.RegisterType((*MsgAddArticle)(nil), "bze.cointrunk.v1.MsgAddArticle")
	proto.RegisterType((*MsgAddArticleResponse)(nil), "bze.cointrunk.v1.MsgAddArticleResponse")
	proto.RegisterType((*MsgPayPublisherRespect)(nil), "bze.cointrunk.v1.MsgPayPublisherRespect")
	proto.RegisterType((*MsgPayPublisherRespectResponse)(nil), "bze.cointrunk.v1.MsgPayPublisherRespectResponse")
}

func init() { proto.RegisterFile("cointrunk/tx.proto", fileDescriptor_776bb1586e9b1fcd) }

var fileDescriptor_776bb1586e9b1fcd = []byte{
	// 401 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x52, 0xbd, 0x8e, 0xd3, 0x40,
	0x10, 0x8e, 0x89, 0x39, 0x74, 0x03, 0x88, 0x68, 0x39, 0x0e, 0xeb, 0x84, 0x0c, 0xb8, 0xe1, 0x28,
	0xb0, 0x03, 0x3c, 0x41, 0x28, 0x10, 0x4d, 0x24, 0xcb, 0x05, 0x05, 0x8d, 0xb5, 0xf6, 0x2e, 0xce,
	0x0a, 0xdb, 0xbb, 0xec, 0x0f, 0x24, 0x79, 0x0a, 0x1e, 0x81, 0xc7, 0xa1, 0x4c, 0x49, 0x89, 0x92,
	0x17, 0x41, 0xde, 0xd8, 0x4e, 0x82, 0x2c, 0x71, 0x9d, 0xbf, 0xef, 0x1b, 0xcf, 0xf7, 0xed, 0xcc,
	0x00, 0xca, 0x39, 0xab, 0xb5, 0x34, 0xf5, 0x97, 0x48, 0x2f, 0x43, 0x21, 0xb9, 0xe6, 0x68, 0x92,
	0xad, 0x69, 0xd8, 0xf3, 0xe1, 0xb7, 0xd7, 0x01, 0x87, 0xfb, 0x73, 0x55, 0xcc, 0x08, 0x99, 0x49,
	0xcd, 0xf2, 0x92, 0xa2, 0x27, 0x70, 0x2e, 0x4c, 0x56, 0x32, 0xb5, 0xa0, 0xd2, 0x73, 0x9e, 0x39,
	0xd7, 0xe7, 0xc9, 0x81, 0x40, 0x17, 0x70, 0x5b, 0x33, 0x5d, 0x52, 0xef, 0x96, 0x55, 0xf6, 0x00,
	0x4d, 0x60, 0x6c, 0x64, 0xe9, 0x8d, 0x2d, 0xd7, 0x7c, 0x22, 0x0f, 0xee, 0x08, 0x96, 0x6b, 0x23,
	0xa9, 0xe7, 0x5a, 0xb6, 0x83, 0xc1, 0x63, 0x78, 0x74, 0x62, 0x98, 0x50, 0x25, 0x78, 0xad, 0x68,
	0x40, 0xe0, 0x72, 0xae, 0x8a, 0x18, 0xaf, 0xe2, 0xce, 0xad, 0x91, 0x68, 0xae, 0x9b, 0x66, 0xb9,
	0xa4, 0x58, 0xf3, 0x2e, 0x50, 0x07, 0x1b, 0x05, 0x13, 0x22, 0xa9, 0x52, 0x6d, 0xa0, 0x0e, 0xa2,
	0x4b, 0x38, 0xc3, 0x15, 0x37, 0xb5, 0x6e, 0x53, 0xb5, 0x28, 0xf8, 0xe9, 0x80, 0x3f, 0x6c, 0xd3,
	0x05, 0x41, 0xcf, 0xe1, 0x9e, 0xdc, 0x53, 0xa9, 0xc0, 0x8c, 0x58, 0x4f, 0x37, 0xb9, 0xdb, 0x72,
	0x31, 0x66, 0x04, 0xbd, 0x84, 0x49, 0x3f, 0x93, 0x54, 0xd2, 0xef, 0x58, 0x12, 0x1b, 0xc0, 0x4d,
	0x1e, 0x88, 0x43, 0xdb, 0x86, 0x46, 0x53, 0xb8, 0xc8, 0x79, 0x55, 0x99, 0x9a, 0xe9, 0x55, 0x2a,
	0x38, 0x2f, 0xd3, 0xcf, 0xa6, 0x26, 0xca, 0xc6, 0x72, 0x13, 0xd4, 0x6b, 0x31, 0xe7, 0xe5, 0xfb,
	0x46, 0x79, 0xb3, 0x71, 0x60, 0x3c, 0x57, 0x05, 0xfa, 0x08, 0x70, 0xb4, 0x97, 0xa7, 0xe1, 0xbf,
	0xbb, 0x0b, 0x4f, 0xe6, 0x78, 0xf5, 0xe2, 0x3f, 0x05, 0xfd, 0xfb, 0xbe, 0xc2, 0xc3, 0xa1, 0x29,
	0x5f, 0x0f, 0xfe, 0x3f, 0x50, 0x79, 0x35, 0xbd, 0x69, 0x65, 0x67, 0xf9, 0xee, 0xc3, 0xaf, 0xad,
	0xef, 0x6c, 0xb6, 0xbe, 0xf3, 0x67, 0xeb, 0x3b, 0x3f, 0x76, 0xfe, 0x68, 0xb3, 0xf3, 0x47, 0xbf,
	0x77, 0xfe, 0xe8, 0x53, 0x58, 0x30, 0xbd, 0x30, 0x59, 0x98, 0xf3, 0x2a, 0xca, 0xd6, 0xf4, 0x15,
	0x2e, 0xc5, 0x02, 0x6b, 0x8a, 0x2d, 0x8a, 0x96, 0xd1, 0xd1, 0x11, 0xaf, 0x04, 0x55, 0xd9, 0x99,
	0x3d, 0xe4, 0xb7, 0x7f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x9e, 0x65, 0xc4, 0x6b, 0xde, 0x02, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	AddArticle(ctx context.Context, in *MsgAddArticle, opts ...grpc.CallOption) (*MsgAddArticleResponse, error)
	PayPublisherRespect(ctx context.Context, in *MsgPayPublisherRespect, opts ...grpc.CallOption) (*MsgPayPublisherRespectResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) AddArticle(ctx context.Context, in *MsgAddArticle, opts ...grpc.CallOption) (*MsgAddArticleResponse, error) {
	out := new(MsgAddArticleResponse)
	err := c.cc.Invoke(ctx, "/bze.cointrunk.v1.Msg/AddArticle", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) PayPublisherRespect(ctx context.Context, in *MsgPayPublisherRespect, opts ...grpc.CallOption) (*MsgPayPublisherRespectResponse, error) {
	out := new(MsgPayPublisherRespectResponse)
	err := c.cc.Invoke(ctx, "/bze.cointrunk.v1.Msg/PayPublisherRespect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	AddArticle(context.Context, *MsgAddArticle) (*MsgAddArticleResponse, error)
	PayPublisherRespect(context.Context, *MsgPayPublisherRespect) (*MsgPayPublisherRespectResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) AddArticle(ctx context.Context, req *MsgAddArticle) (*MsgAddArticleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddArticle not implemented")
}
func (*UnimplementedMsgServer) PayPublisherRespect(ctx context.Context, req *MsgPayPublisherRespect) (*MsgPayPublisherRespectResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PayPublisherRespect not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_AddArticle_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddArticle)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddArticle(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bze.cointrunk.v1.Msg/AddArticle",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddArticle(ctx, req.(*MsgAddArticle))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_PayPublisherRespect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgPayPublisherRespect)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).PayPublisherRespect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bze.cointrunk.v1.Msg/PayPublisherRespect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).PayPublisherRespect(ctx, req.(*MsgPayPublisherRespect))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "bze.cointrunk.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddArticle",
			Handler:    _Msg_AddArticle_Handler,
		},
		{
			MethodName: "PayPublisherRespect",
			Handler:    _Msg_PayPublisherRespect_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cointrunk/tx.proto",
}

func (m *MsgAddArticle) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgAddArticle) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgAddArticle) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Picture) > 0 {
		i -= len(m.Picture)
		copy(dAtA[i:], m.Picture)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Picture)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Url) > 0 {
		i -= len(m.Url)
		copy(dAtA[i:], m.Url)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Url)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Publisher) > 0 {
		i -= len(m.Publisher)
		copy(dAtA[i:], m.Publisher)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Publisher)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgAddArticleResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgAddArticleResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgAddArticleResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgPayPublisherRespect) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgPayPublisherRespect) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgPayPublisherRespect) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Amount) > 0 {
		i -= len(m.Amount)
		copy(dAtA[i:], m.Amount)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Amount)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgPayPublisherRespectResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgPayPublisherRespectResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgPayPublisherRespectResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.CommunityPoolFunds != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.CommunityPoolFunds))
		i--
		dAtA[i] = 0x18
	}
	if m.PublisherReward != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.PublisherReward))
		i--
		dAtA[i] = 0x10
	}
	if m.RespectPaid != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.RespectPaid))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgAddArticle) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Publisher)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Url)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Picture)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgAddArticleResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgPayPublisherRespect) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Amount)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgPayPublisherRespectResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RespectPaid != 0 {
		n += 1 + sovTx(uint64(m.RespectPaid))
	}
	if m.PublisherReward != 0 {
		n += 1 + sovTx(uint64(m.PublisherReward))
	}
	if m.CommunityPoolFunds != 0 {
		n += 1 + sovTx(uint64(m.CommunityPoolFunds))
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgAddArticle) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgAddArticle: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgAddArticle: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Publisher", wireType)
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
			m.Publisher = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
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
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Url", wireType)
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
			m.Url = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Picture", wireType)
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
			m.Picture = string(dAtA[iNdEx:postIndex])
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
func (m *MsgAddArticleResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgAddArticleResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgAddArticleResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
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
func (m *MsgPayPublisherRespect) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgPayPublisherRespect: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgPayPublisherRespect: illegal tag %d (wire type %d)", fieldNum, wire)
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
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
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
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
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
			m.Amount = string(dAtA[iNdEx:postIndex])
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
func (m *MsgPayPublisherRespectResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgPayPublisherRespectResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgPayPublisherRespectResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RespectPaid", wireType)
			}
			m.RespectPaid = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RespectPaid |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PublisherReward", wireType)
			}
			m.PublisherReward = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PublisherReward |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CommunityPoolFunds", wireType)
			}
			m.CommunityPoolFunds = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CommunityPoolFunds |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
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
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
