# 99-OTPs Go gRPC Service

A gRPC service that generates six-digit OTPs, sends them through a configurable SMS provider, and verifies them once. Codes are stored in Redis for two minutes.

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
GRPC_ADDR=:50051
```

`GRPC_ADDR` is optional and defaults to `:50051`.

## Run

```bash
go mod tidy
go run .
```

The service definition is in `otp.proto`. Requests and responses use `google.protobuf.Struct`, with these fields:

| RPC | Request fields | Success response |
|-----|----------------|------------------|
| `SendOTP` | `phone_number` (string) | `{ "message": "OTP sent" }` |
| `VerifyOTP` | `phone_number` (string), `otp` (string) | `{ "success": true }` |

Phone numbers must use Ghana format: `0XXXXXXXXX` or `+233XXXXXXXXX`. Send requests are limited to 5 per minute per client IP, and verify requests to 10 per minute per client IP. Invalid or expired codes return gRPC `INVALID_ARGUMENT`; rate limits return `RESOURCE_EXHAUSTED`, and SMS delivery failures return `UNAVAILABLE`.

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