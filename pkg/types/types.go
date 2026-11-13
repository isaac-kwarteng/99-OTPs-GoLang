package types

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/marketdome/otp-api/gen/marketdome/otp/v1"
)

// Register services, repository, and other types here

type OTPService interface {
	SendOTP(ctx context.Context, request *connect.Request[v1.SendOTPRequest]) (*v1.SendOTPResponse, error)
	VerifyOTP(ctx context.Context, request *connect.Request[v1.VerifyOTPRequest]) (*v1.VerifyOTPResponse, error)
}

type OTPRepository interface {
	Save(ctx context.Context, phoneNumber, otp string, ttl time.Duration) error
	ConsumeIfMatch(ctx context.Context, phoneNumber, otp string) (bool, error)
}

type SMSSender interface {
	Send(ctx context.Context, phoneNumber, message string) error
}
