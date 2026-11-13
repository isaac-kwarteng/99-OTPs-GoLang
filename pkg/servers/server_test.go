package servers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	v1 "github.com/marketdome/otp-api/gen/marketdome/otp/v1"
	"github.com/marketdome/otp-api/gen/marketdome/otp/v1/otpv1connect"
	"github.com/marketdome/otp-api/pkg/services"
	"google.golang.org/genproto/googleapis/type/phone_number"
)

type memoryOTPRepository struct {
	codes map[string]string
}

func (repository *memoryOTPRepository) Save(_ context.Context, phoneNumber, code string, _ time.Duration) error {
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
	message string
}

func (sender *recordingSMSSender) Send(_ context.Context, _, message string) error {
	sender.message = message
	return nil
}

func phone(e164 string) *phone_number.PhoneNumber {
	return &phone_number.PhoneNumber{Kind: &phone_number.PhoneNumber_E164Number{E164Number: e164}}
}

func newTestClient(t *testing.T, sender *recordingSMSSender) otpv1connect.OTPServiceClient {
	t.Helper()
	service := services.NewOTPService(&memoryOTPRepository{codes: map[string]string{}}, sender)
	path, handler := otpv1connect.NewOTPServiceHandler(NewOTPServer(service), connect.WithInterceptors(validate.NewInterceptor()))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return otpv1connect.NewOTPServiceClient(server.Client(), server.URL)
}

func TestConnectClientSendsAndVerifiesOTP(t *testing.T) {
	sender := &recordingSMSSender{}
	client := newTestClient(t, sender)
	ctx := context.Background()

	sendResponse, err := client.SendOTP(ctx, connect.NewRequest(&v1.SendOTPRequest{PhoneNumber: phone("+233241234567")}))
	if err != nil {
		t.Fatalf("SendOTP() error = %v", err)
	}
	if sendResponse.Msg.GetMessage() != "OTP sent" || sendResponse.Msg.GetExpiresIn() != 120 {
		t.Fatalf("SendOTP() response = %v", sendResponse.Msg)
	}

	code := regexp.MustCompile(`[0-9]{6}`).FindString(sender.message)
	verifyResponse, err := client.VerifyOTP(ctx, connect.NewRequest(&v1.VerifyOTPRequest{PhoneNumber: phone("+233241234567"), Otp: code}))
	if err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if !verifyResponse.Msg.GetSuccess() {
		t.Fatal("VerifyOTP() success = false, want true")
	}
}

func TestConnectClientRejectsInvalidRequests(t *testing.T) {
	client := newTestClient(t, &recordingSMSSender{})
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "missing phone", call: func() error {
			_, err := client.SendOTP(ctx, connect.NewRequest(&v1.SendOTPRequest{}))
			return err
		}},
		{name: "non-Ghana phone", call: func() error {
			_, err := client.SendOTP(ctx, connect.NewRequest(&v1.SendOTPRequest{PhoneNumber: phone("+14155552671")}))
			return err
		}},
		{name: "local format phone", call: func() error {
			_, err := client.SendOTP(ctx, connect.NewRequest(&v1.SendOTPRequest{PhoneNumber: phone("0241234567")}))
			return err
		}},
		{name: "malformed otp", call: func() error {
			_, err := client.VerifyOTP(ctx, connect.NewRequest(&v1.VerifyOTPRequest{PhoneNumber: phone("+233241234567"), Otp: "12ab"}))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if code := connect.CodeOf(tt.call()); code != connect.CodeInvalidArgument {
				t.Fatalf("code = %v, want InvalidArgument", code)
			}
		})
	}
}
