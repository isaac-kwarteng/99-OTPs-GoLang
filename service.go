package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"connectrpc.com/connect"
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
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid Ghana phone number"))
	}
	code, err := generateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not generate OTP"))
	}
	if err := service.store.Save(ctx, phoneNumber, code, otpTTL); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not store OTP"))
	}
	if err := service.sms.Send(ctx, phoneNumber, code); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("SMS delivery failed"))
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
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not verify OTP"))
	}
	if !valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or expired OTP"))
	}
	return structpb.NewStruct(map[string]any{"success": true})
}

func structField(request *structpb.Struct, field string) (string, error) {
	if request == nil {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("request body is required"))
	}
	value, exists := request.Fields[field]
	if !exists || value.GetKind() == nil {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is required", field))
	}
	if _, ok := value.Kind.(*structpb.Value_StringValue); !ok {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s must be a string", field))
	}
	if value.GetStringValue() == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is required", field))
	}
	return value.GetStringValue(), nil
}

func generateOTP() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()+100000), nil
}
