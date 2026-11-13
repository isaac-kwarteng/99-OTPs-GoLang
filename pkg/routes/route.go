package routes

import "github.com/marketdome/otp-api/gen/marketdome/otp/v1/otpv1connect"

// RateLimits Define the maximum requests per minute per client IP for each procedure
var RateLimits = map[string]int{
	otpv1connect.OTPServiceSendOTPProcedure:   5,
	otpv1connect.OTPServiceVerifyOTPProcedure: 10,
}
