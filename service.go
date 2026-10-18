package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var ghanaPhonePattern = regexp.MustCompile(`^(\+233|0)[2-9][0-9]{8}$`)

type SMSDelivery interface {
	Send(ctx context.Context, phoneNumber, otp string) error
}

type OTPService struct {
	store OTPStore
	sms   SMSDelivery
}

func NewOTPService(store OTPStore, sms SMSDelivery) *OTPService {
	return &OTPService{store: store, sms: sms}
}

func (service *OTPService) SendOTP(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	phoneNumber, err := structField(request, "phone_number")
	if err != nil {
		return nil, err
	}
	if !ghanaPhonePattern.MatchString(phoneNumber) {
		return nil, status.Error(codes.InvalidArgument, "invalid Ghana phone number")
	}
	code, err := generateOTP()
	if err != nil {
		return nil, status.Error(codes.Internal, "could not generate OTP")
	}
	if err := service.store.Save(ctx, phoneNumber, code, otpTTL); err != nil {
		return nil, status.Error(codes.Internal, "could not store OTP")
	}
	if err := service.sms.Send(ctx, phoneNumber, code); err != nil {
		return nil, status.Error(codes.Unavailable, "SMS delivery failed")
	}
	return structpb.NewStruct(map[string]any{"message": "OTP sent"})
}

func (service *OTPService) VerifyOTP(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	phoneNumber, err := structField(request, "phone_number")
	if err != nil {
		return nil, err
	}
	code, err := structField(request, "otp")
	if err != nil {
		return nil, err
	}
	valid, err := service.store.ConsumeIfMatch(ctx, phoneNumber, code)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not verify OTP")
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired OTP")
	}
	return structpb.NewStruct(map[string]any{"success": true})
}

func generateOTP() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()+100000), nil
}