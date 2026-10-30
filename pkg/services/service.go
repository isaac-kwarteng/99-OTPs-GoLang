package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/marketdome/otp-api/gen/marketdome/otp/v1"
	"github.com/marketdome/otp-api/pkg/types"
)

const otpTTL = 2 * time.Minute

type otpService struct {
	otpRepository types.OTPRepository
	smsSender     types.SMSSender
}

func (service *otpService) SendOTP(ctx context.Context, request *connect.Request[v1.SendOTPRequest]) (*v1.SendOTPResponse, error) {
	phoneNumber := request.Msg.GetPhoneNumber().GetE164Number()

	code, err := generateOTP()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not generate OTP"))
	}
	if err := service.otpRepository.Save(ctx, phoneNumber, code, otpTTL); err != nil {
		log.Println("failed to store OTP:", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not store OTP"))
	}

	message := fmt.Sprintf("Your OTP is %s. It expires in %d minutes. Do not share it with anyone.", code, int(otpTTL.Minutes()))
	if err := service.smsSender.Send(ctx, phoneNumber, message); err != nil {
		log.Println("failed to send OTP SMS:", err)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("SMS delivery failed"))
	}

	return &v1.SendOTPResponse{
		Message:   "OTP sent",
		ExpiresIn: int32(otpTTL.Seconds()),
	}, nil
}

func (service *otpService) VerifyOTP(ctx context.Context, request *connect.Request[v1.VerifyOTPRequest]) (*v1.VerifyOTPResponse, error) {
	verifyRequest := request.Msg

	valid, err := service.otpRepository.ConsumeIfMatch(ctx, verifyRequest.GetPhoneNumber().GetE164Number(), verifyRequest.GetOtp())
	if err != nil {
		log.Println("failed to verify OTP:", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not verify OTP"))
	}
	if !valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or expired OTP"))
	}

	return &v1.VerifyOTPResponse{Success: true}, nil
}

func generateOTP() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()+100000), nil
}

func NewOTPService(otpRepository types.OTPRepository, smsSender types.SMSSender) types.OTPService {
	return &otpService{otpRepository: otpRepository, smsSender: smsSender}
}
