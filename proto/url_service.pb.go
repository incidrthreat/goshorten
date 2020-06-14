// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.21.0-devel
// 	protoc        v3.11.4
// source: url_service.proto

package url_service

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type URLRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
}

func (x *URLRequest) Reset() {
	*x = URLRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_url_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *URLRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*URLRequest) ProtoMessage() {}

func (x *URLRequest) ProtoReflect() protoreflect.Message {
	mi := &file_url_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use URLRequest.ProtoReflect.Descriptor instead.
func (*URLRequest) Descriptor() ([]byte, []int) {
	return file_url_service_proto_rawDescGZIP(), []int{0}
}

func (x *URLRequest) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

type URLResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Shortened string `protobuf:"bytes,1,opt,name=shortened,proto3" json:"shortened,omitempty"`
}

func (x *URLResponse) Reset() {
	*x = URLResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_url_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *URLResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*URLResponse) ProtoMessage() {}

func (x *URLResponse) ProtoReflect() protoreflect.Message {
	mi := &file_url_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use URLResponse.ProtoReflect.Descriptor instead.
func (*URLResponse) Descriptor() ([]byte, []int) {
	return file_url_service_proto_rawDescGZIP(), []int{1}
}

func (x *URLResponse) GetShortened() string {
	if x != nil {
		return x.Shortened
	}
	return ""
}

var File_url_service_proto protoreflect.FileDescriptor

var file_url_service_proto_rawDesc = []byte{
	0x0a, 0x11, 0x75, 0x72, 0x6c, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x1e, 0x0a, 0x0a, 0x55, 0x52, 0x4c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x75, 0x72, 0x6c, 0x22, 0x2b, 0x0a, 0x0b, 0x55, 0x52, 0x4c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x68, 0x6f, 0x72, 0x74, 0x65, 0x6e, 0x65, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x68, 0x6f, 0x72, 0x74, 0x65, 0x6e, 0x65, 0x64,
	0x32, 0x30, 0x0a, 0x09, 0x53, 0x68, 0x6f, 0x72, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x12, 0x23, 0x0a,
	0x06, 0x47, 0x65, 0x74, 0x55, 0x52, 0x4c, 0x12, 0x0b, 0x2e, 0x55, 0x52, 0x4c, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x0c, 0x2e, 0x55, 0x52, 0x4c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x42, 0x0f, 0x5a, 0x0d, 0x2e, 0x3b, 0x75, 0x72, 0x6c, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_url_service_proto_rawDescOnce sync.Once
	file_url_service_proto_rawDescData = file_url_service_proto_rawDesc
)

func file_url_service_proto_rawDescGZIP() []byte {
	file_url_service_proto_rawDescOnce.Do(func() {
		file_url_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_url_service_proto_rawDescData)
	})
	return file_url_service_proto_rawDescData
}

var file_url_service_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_url_service_proto_goTypes = []interface{}{
	(*URLRequest)(nil),  // 0: URLRequest
	(*URLResponse)(nil), // 1: URLResponse
}
var file_url_service_proto_depIdxs = []int32{
	0, // 0: Shortener.GetURL:input_type -> URLRequest
	1, // 1: Shortener.GetURL:output_type -> URLResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_url_service_proto_init() }
func file_url_service_proto_init() {
	if File_url_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_url_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*URLRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_url_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*URLResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_url_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_url_service_proto_goTypes,
		DependencyIndexes: file_url_service_proto_depIdxs,
		MessageInfos:      file_url_service_proto_msgTypes,
	}.Build()
	File_url_service_proto = out.File
	file_url_service_proto_rawDesc = nil
	file_url_service_proto_goTypes = nil
	file_url_service_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// ShortenerClient is the client API for Shortener service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ShortenerClient interface {
	GetURL(ctx context.Context, in *URLRequest, opts ...grpc.CallOption) (*URLResponse, error)
}

type shortenerClient struct {
	cc grpc.ClientConnInterface
}

func NewShortenerClient(cc grpc.ClientConnInterface) ShortenerClient {
	return &shortenerClient{cc}
}

func (c *shortenerClient) GetURL(ctx context.Context, in *URLRequest, opts ...grpc.CallOption) (*URLResponse, error) {
	out := new(URLResponse)
	err := c.cc.Invoke(ctx, "/Shortener/GetURL", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ShortenerServer is the server API for Shortener service.
type ShortenerServer interface {
	GetURL(context.Context, *URLRequest) (*URLResponse, error)
}

// UnimplementedShortenerServer can be embedded to have forward compatible implementations.
type UnimplementedShortenerServer struct {
}

func (*UnimplementedShortenerServer) GetURL(context.Context, *URLRequest) (*URLResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetURL not implemented")
}

func RegisterShortenerServer(s *grpc.Server, srv ShortenerServer) {
	s.RegisterService(&_Shortener_serviceDesc, srv)
}

func _Shortener_GetURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(URLRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ShortenerServer).GetURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Shortener/GetURL",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ShortenerServer).GetURL(ctx, req.(*URLRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Shortener_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Shortener",
	HandlerType: (*ShortenerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetURL",
			Handler:    _Shortener_GetURL_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "url_service.proto",
}
