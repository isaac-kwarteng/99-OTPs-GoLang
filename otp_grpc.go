package main

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const otpServiceName = "otp.v1.OTPService"

type OTPServiceClient interface {
	SendOTP(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error)
	VerifyOTP(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error)
}

type otpServiceClient struct {
	connection grpc.ClientConnInterface
}

func NewOTPServiceClient(connection grpc.ClientConnInterface) OTPServiceClient {
	return &otpServiceClient{connection: connection}
}

func (client *otpServiceClient) SendOTP(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error) {
	response := new(structpb.Struct)
	if err := client.connection.Invoke(ctx, "/"+otpServiceName+"/SendOTP", request, response, options...); err != nil {
		return nil, err
	}
	return response, nil
}

func (client *otpServiceClient) VerifyOTP(ctx context.Context, request *structpb.Struct, options ...grpc.CallOption) (*structpb.Struct, error) {
	response := new(structpb.Struct)
	if err := client.connection.Invoke(ctx, "/"+otpServiceName+"/VerifyOTP", request, response, options...); err != nil {
		return nil, err
	}
	return response, nil
}

type OTPServiceServer interface {
	SendOTP(context.Context, *structpb.Struct) (*structpb.Struct, error)
	VerifyOTP(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

func RegisterOTPServiceServer(registrar grpc.ServiceRegistrar, server OTPServiceServer) {
	registrar.RegisterService(&otpServiceDescription, server)
}

func sendOTPHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	request := new(structpb.Struct)
	if err := decode(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return server.(OTPServiceServer).SendOTP(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: server, FullMethod: "/" + otpServiceName + "/SendOTP"}
	handler := func(ctx context.Context, request any) (any, error) {
		return server.(OTPServiceServer).SendOTP(ctx, request.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

func verifyOTPHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	request := new(structpb.Struct)
	if err := decode(request); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return server.(OTPServiceServer).VerifyOTP(ctx, request)
	}
	info := &grpc.UnaryServerInfo{Server: server, FullMethod: "/" + otpServiceName + "/VerifyOTP"}
	handler := func(ctx context.Context, request any) (any, error) {
		return server.(OTPServiceServer).VerifyOTP(ctx, request.(*structpb.Struct))
	}
	return interceptor(ctx, request, info, handler)
}

var otpServiceDescription = grpc.ServiceDesc{
	ServiceName: otpServiceName,
	HandlerType: (*OTPServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SendOTP", Handler: sendOTPHandler},
		{MethodName: "VerifyOTP", Handler: verifyOTPHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "otp.proto",
}

func structField(request *structpb.Struct, field string) (string, error) {
	if request == nil {
		return "", status.Error(codes.InvalidArgument, "request body is required")
	}
	value, exists := request.Fields[field]
	if !exists || value.GetKind() == nil {
		return "", status.Errorf(codes.InvalidArgument, "%s is required", field)
	}
	if _, ok := value.Kind.(*structpb.Value_StringValue); !ok {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a string", field)
	}
	if value.GetStringValue() == "" {
		return "", status.Errorf(codes.InvalidArgument, "%s is required", field)
	}
	return value.GetStringValue(), nil
}