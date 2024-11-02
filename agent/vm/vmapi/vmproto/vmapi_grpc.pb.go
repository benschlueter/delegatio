// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.28.3
// source: vmapi.proto

package vmproto

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
	API_GetJoinDataKube_FullMethodName = "/vmapi.API/GetJoinDataKube"
	API_InitFirstMaster_FullMethodName = "/vmapi.API/InitFirstMaster"
)

// APIClient is the client API for API service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type APIClient interface {
	GetJoinDataKube(ctx context.Context, in *GetJoinDataKubeRequest, opts ...grpc.CallOption) (*GetJoinDataKubeResponse, error)
	InitFirstMaster(ctx context.Context, in *InitFirstMasterRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[InitFirstMasterResponse], error)
}

type aPIClient struct {
	cc grpc.ClientConnInterface
}

func NewAPIClient(cc grpc.ClientConnInterface) APIClient {
	return &aPIClient{cc}
}

func (c *aPIClient) GetJoinDataKube(ctx context.Context, in *GetJoinDataKubeRequest, opts ...grpc.CallOption) (*GetJoinDataKubeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetJoinDataKubeResponse)
	err := c.cc.Invoke(ctx, API_GetJoinDataKube_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aPIClient) InitFirstMaster(ctx context.Context, in *InitFirstMasterRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[InitFirstMasterResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &API_ServiceDesc.Streams[0], API_InitFirstMaster_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[InitFirstMasterRequest, InitFirstMasterResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type API_InitFirstMasterClient = grpc.ServerStreamingClient[InitFirstMasterResponse]

// APIServer is the server API for API service.
// All implementations must embed UnimplementedAPIServer
// for forward compatibility.
type APIServer interface {
	GetJoinDataKube(context.Context, *GetJoinDataKubeRequest) (*GetJoinDataKubeResponse, error)
	InitFirstMaster(*InitFirstMasterRequest, grpc.ServerStreamingServer[InitFirstMasterResponse]) error
	mustEmbedUnimplementedAPIServer()
}

// UnimplementedAPIServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedAPIServer struct{}

func (UnimplementedAPIServer) GetJoinDataKube(context.Context, *GetJoinDataKubeRequest) (*GetJoinDataKubeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetJoinDataKube not implemented")
}
func (UnimplementedAPIServer) InitFirstMaster(*InitFirstMasterRequest, grpc.ServerStreamingServer[InitFirstMasterResponse]) error {
	return status.Errorf(codes.Unimplemented, "method InitFirstMaster not implemented")
}
func (UnimplementedAPIServer) mustEmbedUnimplementedAPIServer() {}
func (UnimplementedAPIServer) testEmbeddedByValue()             {}

// UnsafeAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to APIServer will
// result in compilation errors.
type UnsafeAPIServer interface {
	mustEmbedUnimplementedAPIServer()
}

func RegisterAPIServer(s grpc.ServiceRegistrar, srv APIServer) {
	// If the following call pancis, it indicates UnimplementedAPIServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&API_ServiceDesc, srv)
}

func _API_GetJoinDataKube_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetJoinDataKubeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServer).GetJoinDataKube(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: API_GetJoinDataKube_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServer).GetJoinDataKube(ctx, req.(*GetJoinDataKubeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _API_InitFirstMaster_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(InitFirstMasterRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(APIServer).InitFirstMaster(m, &grpc.GenericServerStream[InitFirstMasterRequest, InitFirstMasterResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type API_InitFirstMasterServer = grpc.ServerStreamingServer[InitFirstMasterResponse]

// API_ServiceDesc is the grpc.ServiceDesc for API service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var API_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "vmapi.API",
	HandlerType: (*APIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetJoinDataKube",
			Handler:    _API_GetJoinDataKube_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "InitFirstMaster",
			Handler:       _API_InitFirstMaster_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "vmapi.proto",
}
