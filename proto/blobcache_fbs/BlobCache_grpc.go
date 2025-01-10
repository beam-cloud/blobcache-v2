//Generated by gRPC Go plugin
//If you make any local changes, they will be lost
//source: blobcache

package blobcache_fbs

import (
	context "context"
	flatbuffers "github.com/google/flatbuffers/go"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client API for BlobCache service
type BlobCacheClient interface {
	GetContent(ctx context.Context, in *flatbuffers.Builder,
		opts ...grpc.CallOption) (*GetContentResponse, error)
	StoreContent(ctx context.Context,
		opts ...grpc.CallOption) (BlobCache_StoreContentClient, error)
	StoreContentFromSource(ctx context.Context, in *flatbuffers.Builder,
		opts ...grpc.CallOption) (*StoreContentFromSourceResponse, error)
	GetState(ctx context.Context, in *flatbuffers.Builder,
		opts ...grpc.CallOption) (*GetStateResponse, error)
}

type blobCacheClient struct {
	cc grpc.ClientConnInterface
}

func NewBlobCacheClient(cc grpc.ClientConnInterface) BlobCacheClient {
	return &blobCacheClient{cc}
}

func (c *blobCacheClient) GetContent(ctx context.Context, in *flatbuffers.Builder,
	opts ...grpc.CallOption) (*GetContentResponse, error) {
	out := new(GetContentResponse)
	err := c.cc.Invoke(ctx, "/blobcache_fbs.BlobCache/GetContent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) StoreContent(ctx context.Context,
	opts ...grpc.CallOption) (BlobCache_StoreContentClient, error) {
	stream, err := c.cc.NewStream(ctx, &_BlobCache_serviceDesc.Streams[0], "/blobcache_fbs.BlobCache/StoreContent", opts...)
	if err != nil {
		return nil, err
	}
	x := &blobCacheStoreContentClient{stream}
	return x, nil
}

type BlobCache_StoreContentClient interface {
	Send(*flatbuffers.Builder) error
	CloseAndRecv() (*StoreContentResponse, error)
	grpc.ClientStream
}

type blobCacheStoreContentClient struct {
	grpc.ClientStream
}

func (x *blobCacheStoreContentClient) Send(m *flatbuffers.Builder) error {
	return x.ClientStream.SendMsg(m)
}

func (x *blobCacheStoreContentClient) CloseAndRecv() (*StoreContentResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(StoreContentResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *blobCacheClient) StoreContentFromSource(ctx context.Context, in *flatbuffers.Builder,
	opts ...grpc.CallOption) (*StoreContentFromSourceResponse, error) {
	out := new(StoreContentFromSourceResponse)
	err := c.cc.Invoke(ctx, "/blobcache_fbs.BlobCache/StoreContentFromSource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetState(ctx context.Context, in *flatbuffers.Builder,
	opts ...grpc.CallOption) (*GetStateResponse, error) {
	out := new(GetStateResponse)
	err := c.cc.Invoke(ctx, "/blobcache_fbs.BlobCache/GetState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for BlobCache service
type BlobCacheServer interface {
	GetContent(context.Context, *GetContentRequest) (*flatbuffers.Builder, error)
	StoreContent(BlobCache_StoreContentServer) error
	StoreContentFromSource(context.Context, *StoreContentFromSourceRequest) (*flatbuffers.Builder, error)
	GetState(context.Context, *GetStateRequest) (*flatbuffers.Builder, error)
	mustEmbedUnimplementedBlobCacheServer()
}

type UnimplementedBlobCacheServer struct {
}

func (UnimplementedBlobCacheServer) GetContent(context.Context, *GetContentRequest) (*flatbuffers.Builder, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContent not implemented")
}

func (UnimplementedBlobCacheServer) StoreContent(BlobCache_StoreContentServer) error {
	return status.Errorf(codes.Unimplemented, "method StoreContent not implemented")
}

func (UnimplementedBlobCacheServer) StoreContentFromSource(context.Context, *StoreContentFromSourceRequest) (*flatbuffers.Builder, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreContentFromSource not implemented")
}

func (UnimplementedBlobCacheServer) GetState(context.Context, *GetStateRequest) (*flatbuffers.Builder, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}

func (UnimplementedBlobCacheServer) mustEmbedUnimplementedBlobCacheServer() {}

type UnsafeBlobCacheServer interface {
	mustEmbedUnimplementedBlobCacheServer()
}

func RegisterBlobCacheServer(s grpc.ServiceRegistrar, srv BlobCacheServer) {
	s.RegisterService(&_BlobCache_serviceDesc, srv)
}

func _BlobCache_GetContent_Handler(srv interface{}, ctx context.Context,
	dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetContent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/blobcache_fbs.BlobCache/GetContent",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetContent(ctx, req.(*GetContentRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _BlobCache_StoreContent_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BlobCacheServer).StoreContent(&blobCacheStoreContentServer{stream})
}

type BlobCache_StoreContentServer interface {
	Recv() (*StoreContentRequest, error)
	SendAndClose(*flatbuffers.Builder) error
	grpc.ServerStream
}

type blobCacheStoreContentServer struct {
	grpc.ServerStream
}

func (x *blobCacheStoreContentServer) Recv() (*StoreContentRequest, error) {
	m := new(StoreContentRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (x *blobCacheStoreContentServer) SendAndClose(m *flatbuffers.Builder) error {
	return x.ServerStream.SendMsg(m)
}

func _BlobCache_StoreContentFromSource_Handler(srv interface{}, ctx context.Context,
	dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreContentFromSourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).StoreContentFromSource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/blobcache_fbs.BlobCache/StoreContentFromSource",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).StoreContentFromSource(ctx, req.(*StoreContentFromSourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _BlobCache_GetState_Handler(srv interface{}, ctx context.Context,
	dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/blobcache_fbs.BlobCache/GetState",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetState(ctx, req.(*GetStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}
var _BlobCache_serviceDesc = grpc.ServiceDesc{
	ServiceName: "blobcache_fbs.BlobCache",
	HandlerType: (*BlobCacheServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetContent",
			Handler:    _BlobCache_GetContent_Handler,
		},
		{
			MethodName: "StoreContentFromSource",
			Handler:    _BlobCache_StoreContentFromSource_Handler,
		},
		{
			MethodName: "GetState",
			Handler:    _BlobCache_GetState_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StoreContent",
			Handler:       _BlobCache_StoreContent_Handler,
			ClientStreams: true,
		},
	},
}
