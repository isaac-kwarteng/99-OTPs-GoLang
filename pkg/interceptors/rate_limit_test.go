package interceptors

import (
	"testing"
	"time"
)

func TestRateLimiterUsesPerIPProcedureWindows(t *testing.T) {
	const procedure = "/marketdome.otp.v1.OTPService/SendOTP"
	limiter := NewRateLimiter(map[string]int{procedure: 5})
	start := time.Now()

	for attempt := 0; attempt < 5; attempt++ {
		if !limiter.allow("192.0.2.1", procedure, 5, start) {
			t.Fatalf("request %d was unexpectedly limited", attempt+1)
		}
	}
	if limiter.allow("192.0.2.1", procedure, 5, start) {
		t.Fatal("sixth request from same IP was not limited")
	}
	if !limiter.allow("192.0.2.2", procedure, 5, start) {
		t.Fatal("request from a different IP was unexpectedly limited")
	}
	if !limiter.allow("192.0.2.1", procedure, 5, start.Add(time.Minute)) {
		t.Fatal("request after window expiry was unexpectedly limited")
	}
}
