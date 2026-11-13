package servers

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/marketdome/otp-api/gen/marketdome/otp/v1"
	"github.com/marketdome/otp-api/gen/marketdome/otp/v1/otpv1connect"
	"github.com/marketdome/otp-api/pkg/types"
)

type otpServer struct {
	otpService types.OTPService
}

func (server *otpServer) SendOTP(ctx context.Context, request *connect.Request[v1.SendOTPRequest]) (*connect.Response[v1.SendOTPResponse], error) {
	sendOTPResponse, err := server.otpService.SendOTP(ctx, request)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(sendOTPResponse), nil
}

func (server *otpServer) VerifyOTP(ctx context.Context, request *connect.Request[v1.VerifyOTPRequest]) (*connect.Response[v1.VerifyOTPResponse], error) {
	verifyOTPResponse, err := server.otpService.VerifyOTP(ctx, request)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(verifyOTPResponse), nil
}

func NewOTPServer(otpService types.OTPService) otpv1connect.OTPServiceHandler {
	return &otpServer{otpService: otpService}
}
