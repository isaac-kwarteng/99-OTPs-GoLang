package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type memoryOTPStore struct {
	codes map[string]string
}

func (store *memoryOTPStore) Save(_ context.Context, phoneNumber, code string, _ time.Duration) error {
	if store.codes == nil {
		store.codes = make(map[string]string)
	}
	store.codes[phoneNumber] = code
	return nil
}

func (store *memoryOTPStore) ConsumeIfMatch(_ context.Context, phoneNumber, code string) (bool, error) {
	if store.codes[phoneNumber] != code {
		return false, nil
	}
	delete(store.codes, phoneNumber)
	return true, nil
}

type recordingSMS struct {
	phoneNumber string
	code        string
}

func (sms *recordingSMS) Send(_ context.Context, phoneNumber, code string) error {
	sms.phoneNumber = phoneNumber
	sms.code = code
	return nil
}

func TestSendAndVerifyOTPConsumesCode(t *testing.T) {
	store := &memoryOTPStore{}
	sms := &recordingSMS{}
	service := NewOTPService(store, sms)
	request, err := structpb.NewStruct(map[string]any{"phone_number": "0241234567"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.SendOTP(context.Background(), request); err != nil {
		t.Fatalf("SendOTP() error = %v", err)
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(sms.code) {
		t.Fatalf("generated OTP %q is not six digits", sms.code)
	}
	if sms.phoneNumber != "0241234567" || store.codes[sms.phoneNumber] != sms.code {
		t.Fatal("OTP was not stored and delivered to the requested phone number")
	}

	verifyRequest, err := structpb.NewStruct(map[string]any{"phone_number": sms.phoneNumber, "otp": sms.code})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.VerifyOTP(context.Background(), verifyRequest); err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if _, err := service.VerifyOTP(context.Background(), verifyRequest); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("second VerifyOTP() status = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestSendOTPRejectsInvalidPhone(t *testing.T) {
	service := NewOTPService(&memoryOTPStore{}, &recordingSMS{})
	request, err := structpb.NewStruct(map[string]any{"phone_number": "+14155552671"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SendOTP(context.Background(), request); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("SendOTP() status = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestConnectClientCallsRegisteredHandler(t *testing.T) {
	store := &memoryOTPStore{}
	sms := &recordingSMS{}
	mux := http.NewServeMux()
	RegisterOTPServiceHandler(mux, NewOTPService(store, sms), newRateLimiter().UnaryInterceptor())
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewOTPServiceClient(server.Client(), server.URL)

	sendRequest, err := structpb.NewStruct(map[string]any{"phone_number": "0241234567"})
	if err != nil {
		t.Fatal(err)
	}
	sendResponse, err := client.SendOTP(context.Background(), sendRequest)
	if err != nil {
		t.Fatalf("Connect SendOTP() error = %v", err)
	}
	if sendResponse.Fields["message"].GetStringValue() != "OTP sent" {
		t.Fatalf("SendOTP() response = %v, want OTP sent", sendResponse.AsMap())
	}

	verifyRequest, err := structpb.NewStruct(map[string]any{"phone_number": sms.phoneNumber, "otp": sms.code})
	if err != nil {
		t.Fatal(err)
	}
	verifyResponse, err := client.VerifyOTP(context.Background(), verifyRequest)
	if err != nil {
		t.Fatalf("Connect VerifyOTP() error = %v", err)
	}
	if !verifyResponse.Fields["success"].GetBoolValue() {
		t.Fatalf("VerifyOTP() response = %v, want success", verifyResponse.AsMap())
	}
}

func TestRateLimiterUsesPerIPMethodWindows(t *testing.T) {
	limiter := newRateLimiter()
	start := time.Now()
	method := "/" + otpServiceName + "/SendOTP"
	for attempt := 0; attempt < 5; attempt++ {
		if !limiter.allow("192.0.2.1", method, 5, start) {
			t.Fatalf("request %d was unexpectedly limited", attempt+1)
		}
	}
	if limiter.allow("192.0.2.1", method, 5, start) {
		t.Fatal("sixth request from same IP was not limited")
	}
	if !limiter.allow("192.0.2.2", method, 5, start) {
		t.Fatal("request from a different IP was unexpectedly limited")
	}
	if !limiter.allow("192.0.2.1", method, 5, start.Add(time.Minute)) {
		t.Fatal("request after window expiry was unexpectedly limited")
	}
}
