// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.26.1
// source: card/v1/card.proto

package cardv1

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
	Card_CreateUser_FullMethodName             = "/api.card.v1.card/CreateUser"
	Card_UpdateUserKey_FullMethodName          = "/api.card.v1.card/UpdateUserKey"
	Card_GetRecordOfConsumption_FullMethodName = "/api.card.v1.card/GetRecordOfConsumption"
)

// CardClient is the client API for Card service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CardClient interface {
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*OperationResponse, error)
	UpdateUserKey(ctx context.Context, in *UpdateUserKeyRequest, opts ...grpc.CallOption) (*OperationResponse, error)
	GetRecordOfConsumption(ctx context.Context, in *GetRecordOfConsumptionRequest, opts ...grpc.CallOption) (*GetRecordOfConsumptionResponse, error)
}

type cardClient struct {
	cc grpc.ClientConnInterface
}

func NewCardClient(cc grpc.ClientConnInterface) CardClient {
	return &cardClient{cc}
}

func (c *cardClient) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*OperationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OperationResponse)
	err := c.cc.Invoke(ctx, Card_CreateUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardClient) UpdateUserKey(ctx context.Context, in *UpdateUserKeyRequest, opts ...grpc.CallOption) (*OperationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OperationResponse)
	err := c.cc.Invoke(ctx, Card_UpdateUserKey_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardClient) GetRecordOfConsumption(ctx context.Context, in *GetRecordOfConsumptionRequest, opts ...grpc.CallOption) (*GetRecordOfConsumptionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRecordOfConsumptionResponse)
	err := c.cc.Invoke(ctx, Card_GetRecordOfConsumption_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CardServer is the server API for Card service.
// All implementations must embed UnimplementedCardServer
// for forward compatibility.
type CardServer interface {
	CreateUser(context.Context, *CreateUserRequest) (*OperationResponse, error)
	UpdateUserKey(context.Context, *UpdateUserKeyRequest) (*OperationResponse, error)
	GetRecordOfConsumption(context.Context, *GetRecordOfConsumptionRequest) (*GetRecordOfConsumptionResponse, error)
	mustEmbedUnimplementedCardServer()
}

// UnimplementedCardServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedCardServer struct{}

func (UnimplementedCardServer) CreateUser(context.Context, *CreateUserRequest) (*OperationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
func (UnimplementedCardServer) UpdateUserKey(context.Context, *UpdateUserKeyRequest) (*OperationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUserKey not implemented")
}
func (UnimplementedCardServer) GetRecordOfConsumption(context.Context, *GetRecordOfConsumptionRequest) (*GetRecordOfConsumptionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRecordOfConsumption not implemented")
}
func (UnimplementedCardServer) mustEmbedUnimplementedCardServer() {}
func (UnimplementedCardServer) testEmbeddedByValue()              {}

// UnsafeCardServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CardServer will
// result in compilation errors.
type UnsafeCardServer interface {
	mustEmbedUnimplementedCardServer()
}

func RegisterCardServer(s grpc.ServiceRegistrar, srv CardServer) {
	// If the following call pancis, it indicates UnimplementedCardServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Card_ServiceDesc, srv)
}

func _Card_CreateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServer).CreateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Card_CreateUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServer).CreateUser(ctx, req.(*CreateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Card_UpdateUserKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateUserKeyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServer).UpdateUserKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Card_UpdateUserKey_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServer).UpdateUserKey(ctx, req.(*UpdateUserKeyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Card_GetRecordOfConsumption_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRecordOfConsumptionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServer).GetRecordOfConsumption(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Card_GetRecordOfConsumption_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServer).GetRecordOfConsumption(ctx, req.(*GetRecordOfConsumptionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Card_ServiceDesc is the grpc.ServiceDesc for Card service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Card_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.card.v1.card",
	HandlerType: (*CardServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateUser",
			Handler:    _Card_CreateUser_Handler,
		},
		{
			MethodName: "UpdateUserKey",
			Handler:    _Card_UpdateUserKey_Handler,
		},
		{
			MethodName: "GetRecordOfConsumption",
			Handler:    _Card_GetRecordOfConsumption_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "card/v1/card.proto",
}
