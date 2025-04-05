// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: classService/v1/free_classroom.proto

package classServicev1

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
	FreeClassroomSvc_QueryFreeClassroom_FullMethodName = "/classService.v1.FreeClassroomSvc/QueryFreeClassroom"
)

// FreeClassroomSvcClient is the client API for FreeClassroomSvc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FreeClassroomSvcClient interface {
	QueryFreeClassroom(ctx context.Context, in *QueryFreeClassroomReq, opts ...grpc.CallOption) (*QueryFreeClassroomResp, error)
}

type freeClassroomSvcClient struct {
	cc grpc.ClientConnInterface
}

func NewFreeClassroomSvcClient(cc grpc.ClientConnInterface) FreeClassroomSvcClient {
	return &freeClassroomSvcClient{cc}
}

func (c *freeClassroomSvcClient) QueryFreeClassroom(ctx context.Context, in *QueryFreeClassroomReq, opts ...grpc.CallOption) (*QueryFreeClassroomResp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryFreeClassroomResp)
	err := c.cc.Invoke(ctx, FreeClassroomSvc_QueryFreeClassroom_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FreeClassroomSvcServer is the server API for FreeClassroomSvc service.
// All implementations must embed UnimplementedFreeClassroomSvcServer
// for forward compatibility.
type FreeClassroomSvcServer interface {
	QueryFreeClassroom(context.Context, *QueryFreeClassroomReq) (*QueryFreeClassroomResp, error)
	mustEmbedUnimplementedFreeClassroomSvcServer()
}

// UnimplementedFreeClassroomSvcServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedFreeClassroomSvcServer struct{}

func (UnimplementedFreeClassroomSvcServer) QueryFreeClassroom(context.Context, *QueryFreeClassroomReq) (*QueryFreeClassroomResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryFreeClassroom not implemented")
}
func (UnimplementedFreeClassroomSvcServer) mustEmbedUnimplementedFreeClassroomSvcServer() {}
func (UnimplementedFreeClassroomSvcServer) testEmbeddedByValue()                          {}

// UnsafeFreeClassroomSvcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FreeClassroomSvcServer will
// result in compilation errors.
type UnsafeFreeClassroomSvcServer interface {
	mustEmbedUnimplementedFreeClassroomSvcServer()
}

func RegisterFreeClassroomSvcServer(s grpc.ServiceRegistrar, srv FreeClassroomSvcServer) {
	// If the following call pancis, it indicates UnimplementedFreeClassroomSvcServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&FreeClassroomSvc_ServiceDesc, srv)
}

func _FreeClassroomSvc_QueryFreeClassroom_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryFreeClassroomReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FreeClassroomSvcServer).QueryFreeClassroom(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FreeClassroomSvc_QueryFreeClassroom_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FreeClassroomSvcServer).QueryFreeClassroom(ctx, req.(*QueryFreeClassroomReq))
	}
	return interceptor(ctx, in, info, handler)
}

// FreeClassroomSvc_ServiceDesc is the grpc.ServiceDesc for FreeClassroomSvc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FreeClassroomSvc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "classService.v1.FreeClassroomSvc",
	HandlerType: (*FreeClassroomSvcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryFreeClassroom",
			Handler:    _FreeClassroomSvc_QueryFreeClassroom_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "classService/v1/free_classroom.proto",
}
