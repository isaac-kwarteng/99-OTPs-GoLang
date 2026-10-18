# 99-OTPs Go Connect RPC Service

A Connect RPC service that generates six-digit OTPs, sends them through a configurable SMS provider, and verifies them once. Codes are stored in Redis for two minutes. The Connect protocol works over HTTP/1.1 or HTTP/2.

## Requirements

- Go 1.22 or later
- Redis
- An SMS provider accepting the configured JSON request

## Configuration

Set these environment variables, or put them in a `.env` file in the project directory:

```env
REDIS_URL=redis://localhost:6379
SMS_API_URL=https://your-sms-provider.com/api/send
SMS_API_KEY=your_api_key
SMS_SENDER_ID=YourSenderID
CONNECT_ADDR=:50051
```

`CONNECT_ADDR` is optional and defaults to `:50051`.

## Run

```bash
go mod tidy
go run .
```

The service definition is in `otp.proto`. Connect RPC procedures use `google.protobuf.Struct` and the paths `/otp.v1.OTPService/SendOTP` and `/otp.v1.OTPService/VerifyOTP`.

| RPC | Request fields | Success response |
|-----|----------------|------------------|
| `SendOTP` | `phone_number` (string) | `{ "message": "OTP sent" }` |
| `VerifyOTP` | `phone_number` (string), `otp` (string) | `{ "success": true }` |

Phone numbers must use Ghana format: `0XXXXXXXXX` or `+233XXXXXXXXX`. Send requests are limited to 5 per minute per client IP, and verify requests to 10 per minute per client IP. Invalid or expired codes return Connect `invalid_argument`; rate limits return `resource_exhausted`, and SMS delivery failures return `unavailable`.

Connect clients can call the API directly over HTTP. For example:

```bash
curl --header 'Content-Type: application/json' \
  --data '{"phone_number":"0241234567"}' \
  http://localhost:50051/otp.v1.OTPService/SendOTP
```

The Go `NewOTPServiceClient` helper provides a typed Connect client for both procedures.

The SMS provider receives a POST request with a bearer token and this JSON shape:

```json
{
  "phone": ["0241234567"],
  "sender_id": "YourSenderID",
  "message": "Your OTP is 123456. It expires in 2 minutes. Do not share it with anyone."
}
```

Run the unit tests with:

```bash
go test ./...
```