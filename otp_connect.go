package main

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
)

const otpServiceName = "otp.v1.OTPService"

type OTPServiceClient interface {
	SendOTP(context.Context, *structpb.Struct) (*structpb.Struct, error)
	VerifyOTP(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

type otpServiceClient struct {
	sendOTP   *connect.Client[structpb.Struct, structpb.Struct]
	verifyOTP *connect.Client[structpb.Struct, structpb.Struct]
}

func NewOTPServiceClient(httpClient connect.HTTPClient, baseURL string) OTPServiceClient {
	baseURL = strings.TrimRight(baseURL, "/") + "/" + otpServiceName + "/"
	return &otpServiceClient{
		sendOTP:   connect.NewClient[structpb.Struct, structpb.Struct](httpClient, baseURL+"SendOTP"),
		verifyOTP: connect.NewClient[structpb.Struct, structpb.Struct](httpClient, baseURL+"VerifyOTP"),
	}
}

func (client *otpServiceClient) SendOTP(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	response, err := client.sendOTP.CallUnary(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (client *otpServiceClient) VerifyOTP(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	response, err := client.verifyOTP.CallUnary(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func RegisterOTPServiceHandler(mux *http.ServeMux, service *OTPService, limiter connect.Interceptor) {
	options := []connect.HandlerOption{connect.WithInterceptors(limiter)}
	sendProcedure := "/" + otpServiceName + "/SendOTP"
	verifyProcedure := "/" + otpServiceName + "/VerifyOTP"

	mux.Handle(sendProcedure, connect.NewUnaryHandler[structpb.Struct, structpb.Struct](
		sendProcedure,
		func(ctx context.Context, request *connect.Request[structpb.Struct]) (*connect.Response[structpb.Struct], error) {
			response, err := service.SendOTP(ctx, request.Msg)
			if err != nil {
				return nil, err
			}
			return connect.NewResponse(response), nil
		},
		options...,
	))
	mux.Handle(verifyProcedure, connect.NewUnaryHandler[structpb.Struct, structpb.Struct](
		verifyProcedure,
		func(ctx context.Context, request *connect.Request[structpb.Struct]) (*connect.Response[structpb.Struct], error) {
			response, err := service.VerifyOTP(ctx, request.Msg)
			if err != nil {
				return nil, err
			}
			return connect.NewResponse(response), nil
		},
		options...,
	))
}