// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.1
// source: blobcache.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	BlobCache_GetContent_FullMethodName                     = "/blobcache.BlobCache/GetContent"
	BlobCache_GetContentStream_FullMethodName               = "/blobcache.BlobCache/GetContentStream"
	BlobCache_StoreContent_FullMethodName                   = "/blobcache.BlobCache/StoreContent"
	BlobCache_StoreContentFromSource_FullMethodName         = "/blobcache.BlobCache/StoreContentFromSource"
	BlobCache_StoreContentFromSourceWithLock_FullMethodName = "/blobcache.BlobCache/StoreContentFromSourceWithLock"
	BlobCache_GetState_FullMethodName                       = "/blobcache.BlobCache/GetState"
	BlobCache_GetAvailableHosts_FullMethodName              = "/blobcache.BlobCache/GetAvailableHosts"
	BlobCache_SetClientLock_FullMethodName                  = "/blobcache.BlobCache/SetClientLock"
	BlobCache_RemoveClientLock_FullMethodName               = "/blobcache.BlobCache/RemoveClientLock"
	BlobCache_SetStoreFromContentLock_FullMethodName        = "/blobcache.BlobCache/SetStoreFromContentLock"
	BlobCache_RemoveStoreFromContentLock_FullMethodName     = "/blobcache.BlobCache/RemoveStoreFromContentLock"
	BlobCache_RefreshStoreFromContentLock_FullMethodName    = "/blobcache.BlobCache/RefreshStoreFromContentLock"
	BlobCache_SetFsNode_FullMethodName                      = "/blobcache.BlobCache/SetFsNode"
	BlobCache_GetFsNode_FullMethodName                      = "/blobcache.BlobCache/GetFsNode"
	BlobCache_GetFsNodeChildren_FullMethodName              = "/blobcache.BlobCache/GetFsNodeChildren"
	BlobCache_AddFsNodeChild_FullMethodName                 = "/blobcache.BlobCache/AddFsNodeChild"
	BlobCache_AddHostToIndex_FullMethodName                 = "/blobcache.BlobCache/AddHostToIndex"
	BlobCache_SetHostKeepAlive_FullMethodName               = "/blobcache.BlobCache/SetHostKeepAlive"
)

// BlobCacheClient is the client API for BlobCache service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlobCacheClient interface {
	// Cache RPCs
	GetContent(ctx context.Context, in *GetContentRequest, opts ...grpc.CallOption) (*GetContentResponse, error)
	GetContentStream(ctx context.Context, in *GetContentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[GetContentResponse], error)
	StoreContent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[StoreContentRequest, StoreContentResponse], error)
	StoreContentFromSource(ctx context.Context, in *StoreContentFromSourceRequest, opts ...grpc.CallOption) (*StoreContentFromSourceResponse, error)
	StoreContentFromSourceWithLock(ctx context.Context, in *StoreContentFromSourceRequest, opts ...grpc.CallOption) (*StoreContentFromSourceWithLockResponse, error)
	GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error)
	// Coordinator RPCs
	GetAvailableHosts(ctx context.Context, in *GetAvailableHostsRequest, opts ...grpc.CallOption) (*GetAvailableHostsResponse, error)
	SetClientLock(ctx context.Context, in *SetClientLockRequest, opts ...grpc.CallOption) (*SetClientLockResponse, error)
	RemoveClientLock(ctx context.Context, in *RemoveClientLockRequest, opts ...grpc.CallOption) (*RemoveClientLockResponse, error)
	SetStoreFromContentLock(ctx context.Context, in *SetStoreFromContentLockRequest, opts ...grpc.CallOption) (*SetStoreFromContentLockResponse, error)
	RemoveStoreFromContentLock(ctx context.Context, in *RemoveStoreFromContentLockRequest, opts ...grpc.CallOption) (*RemoveStoreFromContentLockResponse, error)
	RefreshStoreFromContentLock(ctx context.Context, in *RefreshStoreFromContentLockRequest, opts ...grpc.CallOption) (*RefreshStoreFromContentLockResponse, error)
	SetFsNode(ctx context.Context, in *SetFsNodeRequest, opts ...grpc.CallOption) (*SetFsNodeResponse, error)
	GetFsNode(ctx context.Context, in *GetFsNodeRequest, opts ...grpc.CallOption) (*GetFsNodeResponse, error)
	GetFsNodeChildren(ctx context.Context, in *GetFsNodeChildrenRequest, opts ...grpc.CallOption) (*GetFsNodeChildrenResponse, error)
	AddFsNodeChild(ctx context.Context, in *AddFsNodeChildRequest, opts ...grpc.CallOption) (*AddFsNodeChildResponse, error)
	AddHostToIndex(ctx context.Context, in *AddHostToIndexRequest, opts ...grpc.CallOption) (*AddHostToIndexResponse, error)
	SetHostKeepAlive(ctx context.Context, in *SetHostKeepAliveRequest, opts ...grpc.CallOption) (*SetHostKeepAliveResponse, error)
}

type blobCacheClient struct {
	cc grpc.ClientConnInterface
}

func NewBlobCacheClient(cc grpc.ClientConnInterface) BlobCacheClient {
	return &blobCacheClient{cc}
}

func (c *blobCacheClient) GetContent(ctx context.Context, in *GetContentRequest, opts ...grpc.CallOption) (*GetContentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetContentResponse)
	err := c.cc.Invoke(ctx, BlobCache_GetContent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetContentStream(ctx context.Context, in *GetContentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[GetContentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &BlobCache_ServiceDesc.Streams[0], BlobCache_GetContentStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetContentRequest, GetContentResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BlobCache_GetContentStreamClient = grpc.ServerStreamingClient[GetContentResponse]

func (c *blobCacheClient) StoreContent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[StoreContentRequest, StoreContentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &BlobCache_ServiceDesc.Streams[1], BlobCache_StoreContent_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StoreContentRequest, StoreContentResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BlobCache_StoreContentClient = grpc.ClientStreamingClient[StoreContentRequest, StoreContentResponse]

func (c *blobCacheClient) StoreContentFromSource(ctx context.Context, in *StoreContentFromSourceRequest, opts ...grpc.CallOption) (*StoreContentFromSourceResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StoreContentFromSourceResponse)
	err := c.cc.Invoke(ctx, BlobCache_StoreContentFromSource_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) StoreContentFromSourceWithLock(ctx context.Context, in *StoreContentFromSourceRequest, opts ...grpc.CallOption) (*StoreContentFromSourceWithLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StoreContentFromSourceWithLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_StoreContentFromSourceWithLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetStateResponse)
	err := c.cc.Invoke(ctx, BlobCache_GetState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetAvailableHosts(ctx context.Context, in *GetAvailableHostsRequest, opts ...grpc.CallOption) (*GetAvailableHostsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetAvailableHostsResponse)
	err := c.cc.Invoke(ctx, BlobCache_GetAvailableHosts_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) SetClientLock(ctx context.Context, in *SetClientLockRequest, opts ...grpc.CallOption) (*SetClientLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetClientLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_SetClientLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) RemoveClientLock(ctx context.Context, in *RemoveClientLockRequest, opts ...grpc.CallOption) (*RemoveClientLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RemoveClientLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_RemoveClientLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) SetStoreFromContentLock(ctx context.Context, in *SetStoreFromContentLockRequest, opts ...grpc.CallOption) (*SetStoreFromContentLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetStoreFromContentLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_SetStoreFromContentLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) RemoveStoreFromContentLock(ctx context.Context, in *RemoveStoreFromContentLockRequest, opts ...grpc.CallOption) (*RemoveStoreFromContentLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RemoveStoreFromContentLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_RemoveStoreFromContentLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) RefreshStoreFromContentLock(ctx context.Context, in *RefreshStoreFromContentLockRequest, opts ...grpc.CallOption) (*RefreshStoreFromContentLockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RefreshStoreFromContentLockResponse)
	err := c.cc.Invoke(ctx, BlobCache_RefreshStoreFromContentLock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) SetFsNode(ctx context.Context, in *SetFsNodeRequest, opts ...grpc.CallOption) (*SetFsNodeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetFsNodeResponse)
	err := c.cc.Invoke(ctx, BlobCache_SetFsNode_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetFsNode(ctx context.Context, in *GetFsNodeRequest, opts ...grpc.CallOption) (*GetFsNodeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetFsNodeResponse)
	err := c.cc.Invoke(ctx, BlobCache_GetFsNode_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) GetFsNodeChildren(ctx context.Context, in *GetFsNodeChildrenRequest, opts ...grpc.CallOption) (*GetFsNodeChildrenResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetFsNodeChildrenResponse)
	err := c.cc.Invoke(ctx, BlobCache_GetFsNodeChildren_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) AddFsNodeChild(ctx context.Context, in *AddFsNodeChildRequest, opts ...grpc.CallOption) (*AddFsNodeChildResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddFsNodeChildResponse)
	err := c.cc.Invoke(ctx, BlobCache_AddFsNodeChild_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) AddHostToIndex(ctx context.Context, in *AddHostToIndexRequest, opts ...grpc.CallOption) (*AddHostToIndexResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddHostToIndexResponse)
	err := c.cc.Invoke(ctx, BlobCache_AddHostToIndex_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blobCacheClient) SetHostKeepAlive(ctx context.Context, in *SetHostKeepAliveRequest, opts ...grpc.CallOption) (*SetHostKeepAliveResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetHostKeepAliveResponse)
	err := c.cc.Invoke(ctx, BlobCache_SetHostKeepAlive_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlobCacheServer is the server API for BlobCache service.
// All implementations must embed UnimplementedBlobCacheServer
// for forward compatibility.
type BlobCacheServer interface {
	// Cache RPCs
	GetContent(context.Context, *GetContentRequest) (*GetContentResponse, error)
	GetContentStream(*GetContentRequest, grpc.ServerStreamingServer[GetContentResponse]) error
	StoreContent(grpc.ClientStreamingServer[StoreContentRequest, StoreContentResponse]) error
	StoreContentFromSource(context.Context, *StoreContentFromSourceRequest) (*StoreContentFromSourceResponse, error)
	StoreContentFromSourceWithLock(context.Context, *StoreContentFromSourceRequest) (*StoreContentFromSourceWithLockResponse, error)
	GetState(context.Context, *GetStateRequest) (*GetStateResponse, error)
	// Coordinator RPCs
	GetAvailableHosts(context.Context, *GetAvailableHostsRequest) (*GetAvailableHostsResponse, error)
	SetClientLock(context.Context, *SetClientLockRequest) (*SetClientLockResponse, error)
	RemoveClientLock(context.Context, *RemoveClientLockRequest) (*RemoveClientLockResponse, error)
	SetStoreFromContentLock(context.Context, *SetStoreFromContentLockRequest) (*SetStoreFromContentLockResponse, error)
	RemoveStoreFromContentLock(context.Context, *RemoveStoreFromContentLockRequest) (*RemoveStoreFromContentLockResponse, error)
	RefreshStoreFromContentLock(context.Context, *RefreshStoreFromContentLockRequest) (*RefreshStoreFromContentLockResponse, error)
	SetFsNode(context.Context, *SetFsNodeRequest) (*SetFsNodeResponse, error)
	GetFsNode(context.Context, *GetFsNodeRequest) (*GetFsNodeResponse, error)
	GetFsNodeChildren(context.Context, *GetFsNodeChildrenRequest) (*GetFsNodeChildrenResponse, error)
	AddFsNodeChild(context.Context, *AddFsNodeChildRequest) (*AddFsNodeChildResponse, error)
	AddHostToIndex(context.Context, *AddHostToIndexRequest) (*AddHostToIndexResponse, error)
	SetHostKeepAlive(context.Context, *SetHostKeepAliveRequest) (*SetHostKeepAliveResponse, error)
	mustEmbedUnimplementedBlobCacheServer()
}

// UnimplementedBlobCacheServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBlobCacheServer struct{}

func (UnimplementedBlobCacheServer) GetContent(context.Context, *GetContentRequest) (*GetContentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContent not implemented")
}
func (UnimplementedBlobCacheServer) GetContentStream(*GetContentRequest, grpc.ServerStreamingServer[GetContentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method GetContentStream not implemented")
}
func (UnimplementedBlobCacheServer) StoreContent(grpc.ClientStreamingServer[StoreContentRequest, StoreContentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method StoreContent not implemented")
}
func (UnimplementedBlobCacheServer) StoreContentFromSource(context.Context, *StoreContentFromSourceRequest) (*StoreContentFromSourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreContentFromSource not implemented")
}
func (UnimplementedBlobCacheServer) StoreContentFromSourceWithLock(context.Context, *StoreContentFromSourceRequest) (*StoreContentFromSourceWithLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreContentFromSourceWithLock not implemented")
}
func (UnimplementedBlobCacheServer) GetState(context.Context, *GetStateRequest) (*GetStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}
func (UnimplementedBlobCacheServer) GetAvailableHosts(context.Context, *GetAvailableHostsRequest) (*GetAvailableHostsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAvailableHosts not implemented")
}
func (UnimplementedBlobCacheServer) SetClientLock(context.Context, *SetClientLockRequest) (*SetClientLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetClientLock not implemented")
}
func (UnimplementedBlobCacheServer) RemoveClientLock(context.Context, *RemoveClientLockRequest) (*RemoveClientLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveClientLock not implemented")
}
func (UnimplementedBlobCacheServer) SetStoreFromContentLock(context.Context, *SetStoreFromContentLockRequest) (*SetStoreFromContentLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetStoreFromContentLock not implemented")
}
func (UnimplementedBlobCacheServer) RemoveStoreFromContentLock(context.Context, *RemoveStoreFromContentLockRequest) (*RemoveStoreFromContentLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveStoreFromContentLock not implemented")
}
func (UnimplementedBlobCacheServer) RefreshStoreFromContentLock(context.Context, *RefreshStoreFromContentLockRequest) (*RefreshStoreFromContentLockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RefreshStoreFromContentLock not implemented")
}
func (UnimplementedBlobCacheServer) SetFsNode(context.Context, *SetFsNodeRequest) (*SetFsNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetFsNode not implemented")
}
func (UnimplementedBlobCacheServer) GetFsNode(context.Context, *GetFsNodeRequest) (*GetFsNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFsNode not implemented")
}
func (UnimplementedBlobCacheServer) GetFsNodeChildren(context.Context, *GetFsNodeChildrenRequest) (*GetFsNodeChildrenResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFsNodeChildren not implemented")
}
func (UnimplementedBlobCacheServer) AddFsNodeChild(context.Context, *AddFsNodeChildRequest) (*AddFsNodeChildResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddFsNodeChild not implemented")
}
func (UnimplementedBlobCacheServer) AddHostToIndex(context.Context, *AddHostToIndexRequest) (*AddHostToIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddHostToIndex not implemented")
}
func (UnimplementedBlobCacheServer) SetHostKeepAlive(context.Context, *SetHostKeepAliveRequest) (*SetHostKeepAliveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetHostKeepAlive not implemented")
}
func (UnimplementedBlobCacheServer) mustEmbedUnimplementedBlobCacheServer() {}
func (UnimplementedBlobCacheServer) testEmbeddedByValue()                   {}

// UnsafeBlobCacheServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlobCacheServer will
// result in compilation errors.
type UnsafeBlobCacheServer interface {
	mustEmbedUnimplementedBlobCacheServer()
}

func RegisterBlobCacheServer(s grpc.ServiceRegistrar, srv BlobCacheServer) {
	// If the following call pancis, it indicates UnimplementedBlobCacheServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&BlobCache_ServiceDesc, srv)
}

func _BlobCache_GetContent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetContent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_GetContent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetContent(ctx, req.(*GetContentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_GetContentStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetContentRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BlobCacheServer).GetContentStream(m, &grpc.GenericServerStream[GetContentRequest, GetContentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BlobCache_GetContentStreamServer = grpc.ServerStreamingServer[GetContentResponse]

func _BlobCache_StoreContent_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BlobCacheServer).StoreContent(&grpc.GenericServerStream[StoreContentRequest, StoreContentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BlobCache_StoreContentServer = grpc.ClientStreamingServer[StoreContentRequest, StoreContentResponse]

func _BlobCache_StoreContentFromSource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreContentFromSourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).StoreContentFromSource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_StoreContentFromSource_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).StoreContentFromSource(ctx, req.(*StoreContentFromSourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_StoreContentFromSourceWithLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreContentFromSourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).StoreContentFromSourceWithLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_StoreContentFromSourceWithLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).StoreContentFromSourceWithLock(ctx, req.(*StoreContentFromSourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_GetState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_GetState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetState(ctx, req.(*GetStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_GetAvailableHosts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAvailableHostsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetAvailableHosts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_GetAvailableHosts_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetAvailableHosts(ctx, req.(*GetAvailableHostsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_SetClientLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetClientLockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).SetClientLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_SetClientLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).SetClientLock(ctx, req.(*SetClientLockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_RemoveClientLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveClientLockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).RemoveClientLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_RemoveClientLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).RemoveClientLock(ctx, req.(*RemoveClientLockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_SetStoreFromContentLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetStoreFromContentLockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).SetStoreFromContentLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_SetStoreFromContentLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).SetStoreFromContentLock(ctx, req.(*SetStoreFromContentLockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_RemoveStoreFromContentLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveStoreFromContentLockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).RemoveStoreFromContentLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_RemoveStoreFromContentLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).RemoveStoreFromContentLock(ctx, req.(*RemoveStoreFromContentLockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_RefreshStoreFromContentLock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RefreshStoreFromContentLockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).RefreshStoreFromContentLock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_RefreshStoreFromContentLock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).RefreshStoreFromContentLock(ctx, req.(*RefreshStoreFromContentLockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_SetFsNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetFsNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).SetFsNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_SetFsNode_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).SetFsNode(ctx, req.(*SetFsNodeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_GetFsNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFsNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetFsNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_GetFsNode_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetFsNode(ctx, req.(*GetFsNodeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_GetFsNodeChildren_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFsNodeChildrenRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).GetFsNodeChildren(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_GetFsNodeChildren_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).GetFsNodeChildren(ctx, req.(*GetFsNodeChildrenRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_AddFsNodeChild_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddFsNodeChildRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).AddFsNodeChild(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_AddFsNodeChild_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).AddFsNodeChild(ctx, req.(*AddFsNodeChildRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_AddHostToIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddHostToIndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).AddHostToIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_AddHostToIndex_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).AddHostToIndex(ctx, req.(*AddHostToIndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlobCache_SetHostKeepAlive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetHostKeepAliveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlobCacheServer).SetHostKeepAlive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BlobCache_SetHostKeepAlive_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlobCacheServer).SetHostKeepAlive(ctx, req.(*SetHostKeepAliveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BlobCache_ServiceDesc is the grpc.ServiceDesc for BlobCache service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlobCache_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "blobcache.BlobCache",
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
			MethodName: "StoreContentFromSourceWithLock",
			Handler:    _BlobCache_StoreContentFromSourceWithLock_Handler,
		},
		{
			MethodName: "GetState",
			Handler:    _BlobCache_GetState_Handler,
		},
		{
			MethodName: "GetAvailableHosts",
			Handler:    _BlobCache_GetAvailableHosts_Handler,
		},
		{
			MethodName: "SetClientLock",
			Handler:    _BlobCache_SetClientLock_Handler,
		},
		{
			MethodName: "RemoveClientLock",
			Handler:    _BlobCache_RemoveClientLock_Handler,
		},
		{
			MethodName: "SetStoreFromContentLock",
			Handler:    _BlobCache_SetStoreFromContentLock_Handler,
		},
		{
			MethodName: "RemoveStoreFromContentLock",
			Handler:    _BlobCache_RemoveStoreFromContentLock_Handler,
		},
		{
			MethodName: "RefreshStoreFromContentLock",
			Handler:    _BlobCache_RefreshStoreFromContentLock_Handler,
		},
		{
			MethodName: "SetFsNode",
			Handler:    _BlobCache_SetFsNode_Handler,
		},
		{
			MethodName: "GetFsNode",
			Handler:    _BlobCache_GetFsNode_Handler,
		},
		{
			MethodName: "GetFsNodeChildren",
			Handler:    _BlobCache_GetFsNodeChildren_Handler,
		},
		{
			MethodName: "AddFsNodeChild",
			Handler:    _BlobCache_AddFsNodeChild_Handler,
		},
		{
			MethodName: "AddHostToIndex",
			Handler:    _BlobCache_AddHostToIndex_Handler,
		},
		{
			MethodName: "SetHostKeepAlive",
			Handler:    _BlobCache_SetHostKeepAlive_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetContentStream",
			Handler:       _BlobCache_GetContentStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "StoreContent",
			Handler:       _BlobCache_StoreContent_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "blobcache.proto",
}
