package services

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/marketdome/otp-api/gen/marketdome/otp/v1"
	"google.golang.org/genproto/googleapis/type/phone_number"
)

type memoryOTPRepository struct {
	codes map[string]string
}

func (repository *memoryOTPRepository) Save(_ context.Context, phoneNumber, code string, _ time.Duration) error {
	if repository.codes == nil {
		repository.codes = make(map[string]string)
	}
	repository.codes[phoneNumber] = code
	return nil
}

func (repository *memoryOTPRepository) ConsumeIfMatch(_ context.Context, phoneNumber, code string) (bool, error) {
	if repository.codes[phoneNumber] != code {
		return false, nil
	}
	delete(repository.codes, phoneNumber)
	return true, nil
}

type recordingSMSSender struct {
	phoneNumber string
	message     string
	err         error
}

func (sender *recordingSMSSender) Send(_ context.Context, phoneNumber, message string) error {
	sender.phoneNumber = phoneNumber
	sender.message = message
	return sender.err
}

func ghanaPhone(e164 string) *phone_number.PhoneNumber {
	return &phone_number.PhoneNumber{Kind: &phone_number.PhoneNumber_E164Number{E164Number: e164}}
}

func TestSendAndVerifyOTPConsumesCode(t *testing.T) {
	repository := &memoryOTPRepository{}
	sender := &recordingSMSSender{}
	service := NewOTPService(repository, sender)
	ctx := context.Background()

	if _, err := service.SendOTP(ctx, connect.NewRequest(&v1.SendOTPRequest{PhoneNumber: ghanaPhone("+233241234567")})); err != nil {
		t.Fatalf("SendOTP() error = %v", err)
	}
	code := regexp.MustCompile(`[0-9]{6}`).FindString(sender.message)
	if code == "" || sender.phoneNumber != "+233241234567" || repository.codes[sender.phoneNumber] != code {
		t.Fatalf("OTP was not stored and delivered: phone=%q message=%q", sender.phoneNumber, sender.message)
	}

	verifyRequest := &v1.VerifyOTPRequest{PhoneNumber: ghanaPhone("+233241234567"), Otp: code}
	if _, err := service.VerifyOTP(ctx, connect.NewRequest(verifyRequest)); err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if _, err := service.VerifyOTP(ctx, connect.NewRequest(verifyRequest)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("second VerifyOTP() code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestSendOTPReportsSMSFailureAsUnavailable(t *testing.T) {
	service := NewOTPService(&memoryOTPRepository{}, &recordingSMSSender{err: errors.New("gateway down")})

	_, err := service.SendOTP(context.Background(), connect.NewRequest(&v1.SendOTPRequest{PhoneNumber: ghanaPhone("+233241234567")}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("SendOTP() code = %v, want Unavailable", connect.CodeOf(err))
	}
}
