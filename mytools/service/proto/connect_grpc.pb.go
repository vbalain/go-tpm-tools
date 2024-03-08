// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: connect.proto

package connect

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Connect_GetPSK_FullMethodName = "/connect.Connect/GetPSK"
)

// ConnectClient is the client API for Connect service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConnectClient interface {
	GetPSK(ctx context.Context, in *PskRequest, opts ...grpc.CallOption) (*PskResponse, error)
}

type connectClient struct {
	cc grpc.ClientConnInterface
}

func NewConnectClient(cc grpc.ClientConnInterface) ConnectClient {
	return &connectClient{cc}
}

func (c *connectClient) GetPSK(ctx context.Context, in *PskRequest, opts ...grpc.CallOption) (*PskResponse, error) {
	out := new(PskResponse)
	err := c.cc.Invoke(ctx, Connect_GetPSK_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConnectServer is the server API for Connect service.
// All implementations must embed UnimplementedConnectServer
// for forward compatibility
type ConnectServer interface {
	GetPSK(context.Context, *PskRequest) (*PskResponse, error)
	mustEmbedUnimplementedConnectServer()
}

// UnimplementedConnectServer must be embedded to have forward compatible implementations.
type UnimplementedConnectServer struct {
}

func (UnimplementedConnectServer) GetPSK(context.Context, *PskRequest) (*PskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPSK not implemented")
}
func (UnimplementedConnectServer) mustEmbedUnimplementedConnectServer() {}

// UnsafeConnectServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConnectServer will
// result in compilation errors.
type UnsafeConnectServer interface {
	mustEmbedUnimplementedConnectServer()
}

func RegisterConnectServer(s grpc.ServiceRegistrar, srv ConnectServer) {
	s.RegisterService(&Connect_ServiceDesc, srv)
}

func _Connect_GetPSK_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConnectServer).GetPSK(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Connect_GetPSK_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConnectServer).GetPSK(ctx, req.(*PskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Connect_ServiceDesc is the grpc.ServiceDesc for Connect service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Connect_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "connect.Connect",
	HandlerType: (*ConnectServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPSK",
			Handler:    _Connect_GetPSK_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "connect.proto",
}
