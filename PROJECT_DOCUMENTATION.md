# 99-OTPs Go Project Documentation

This repository continues [99-OTPs](https://github.com/isaac-kwarteng/99-OTPs), whose original OTP service was written as a Python REST API. This repository implements the service in Go and exposes it through Connect RPC.

This guide documents the Go implementation: its components, configuration, RPC contract, OTP lifecycle, SMS integration, rate limiting, and tests. For the Python REST API, see its [feature documentation](https://github.com/isaac-kwarteng/99-OTPs/blob/main/restAPI_feature_documentation.md).

## Overview

The service sends cryptographically random six-digit OTPs to Ghana phone numbers and verifies them once. Redis holds each code for two minutes. SMS delivery is performed by an HTTP provider configured through environment variables.

The service uses the Connect protocol over HTTP/1.1 or HTTP/2. It does not expose a conventional REST resource API; its two unary procedures are defined in `otp.proto`.

## Components

- `main.go` loads an optional `.env` file, validates configuration, connects to Redis, registers the RPC handlers, starts the HTTP server, and handles graceful shutdown.
- `otp.proto` defines the `otp.v1.OTPService` service and its `SendOTP` and `VerifyOTP` procedures. Both procedures accept and return `google.protobuf.Struct` values.
- `otp_connect.go` registers the Connect handlers and provides the typed `NewOTPServiceClient` helper.
- `service.go` validates requests, generates OTPs, coordinates storage and SMS delivery, and maps failures to Connect status codes.
- `store.go` implements the OTP store using Redis. A Lua script compares and deletes a matching code atomically.
- `sms.go` sends the provider HTTP request using JSON and bearer-token authentication.
- `rate_limit.go` applies per-IP request limits to the two RPC procedures.
- `service_test.go` covers OTP send/verify behavior, phone validation, the Connect client/handler path, and rate-limiter windows.

## Requirements

- Go 1.22 or later
- Redis
- An SMS provider with an HTTP endpoint that accepts the request format below

## Configuration

The application loads a `.env` file from its current working directory when present. Environment variables already set in the process can also provide the configuration.

| Variable | Required | Description |
|---|---|---|
| `REDIS_URL` | Yes | Redis connection URL, for example `redis://localhost:6379`. The service parses the URL and checks the connection during startup. |
| `SMS_API_URL` | Yes | Absolute `http://` or `https://` URL for the SMS provider endpoint. |
| `SMS_API_KEY` | Yes | Sent to the provider as a bearer token. |
| `SMS_SENDER_ID` | Yes | Sender ID included in the provider request body. |
| `CONNECT_ADDR` | No | HTTP listen address. Defaults to `:50051`. |

Example `.env`:

```env
REDIS_URL=redis://localhost:6379
SMS_API_URL=https://your-sms-provider.com/api/send
SMS_API_KEY=your_api_key
SMS_SENDER_ID=YourSenderID
CONNECT_ADDR=:50051
```

Start Redis, then run the service from the project directory:

```bash
go mod tidy
go run .
```

The process exits during startup if required configuration is missing, Redis cannot be configured or reached, or the SMS endpoint settings are invalid. It listens on `CONNECT_ADDR` and attempts a graceful shutdown on interrupt or termination signals.

## RPC API

The service name is `otp.v1.OTPService`.

| Procedure | Path | Request fields | Success response |
|---|---|---|---|
| `SendOTP` | `/otp.v1.OTPService/SendOTP` | `phone_number` (string) | `{ "message": "OTP sent" }` |
| `VerifyOTP` | `/otp.v1.OTPService/VerifyOTP` | `phone_number` (string), `otp` (string) | `{ "success": true }` |

Requests and responses use JSON objects when calling the Connect endpoint directly over HTTP. Required fields must be non-empty strings.

### Send an OTP

`SendOTP` accepts Ghana phone numbers in either `0XXXXXXXXX` or `+233XXXXXXXXX` form. It generates a six-digit code, saves it in Redis with a two-minute expiration, then asks the SMS provider to deliver it.

```bash
curl --header 'Content-Type: application/json' \
  --data '{"phone_number":"0241234567"}' \
  http://localhost:50051/otp.v1.OTPService/SendOTP
```

On success:

```json
{"message":"OTP sent"}
```

Sending another code for the same phone number replaces the previously stored code. The service stores the code before attempting SMS delivery; if delivery fails, the request returns an error, but the stored code remains until it is replaced or expires.

### Verify an OTP

`VerifyOTP` accepts the phone number and code as strings. Submit the same phone-number representation used to send the code.

```bash
curl --header 'Content-Type: application/json' \
  --data '{"phone_number":"0241234567","otp":"123456"}' \
  http://localhost:50051/otp.v1.OTPService/VerifyOTP
```

On success:

```json
{"success":true}
```

A matching code is deleted atomically when verified. A wrong, expired, or already-consumed code cannot be used successfully.

### Connect status codes

| Status | Meaning |
|---|---|
| `invalid_argument` | Required field is missing or not a string, the send phone number is invalid, or the OTP is wrong, expired, or already used. |
| `resource_exhausted` | The request exceeded its per-IP rate limit. |
| `unavailable` | The SMS provider request failed or returned a non-success HTTP status. |
| `internal` | OTP generation or Redis storage/verification failed. |

Connect errors are returned using the protocol's standard error response format.

## SMS provider integration

The service sends a `POST` request to `SMS_API_URL` with `Content-Type: application/json` and `Authorization: Bearer <SMS_API_KEY>`. The HTTP client timeout is 10 seconds. Any 2xx provider response is treated as successful delivery; other HTTP statuses are treated as failures.

The JSON body has this shape:

```json
{
  "phone": ["0241234567"],
  "sender_id": "YourSenderID",
  "message": "Your OTP is 123456. It expires in 2 minutes. Do not share it with anyone."
}
```

The provider must accept the phone number as a one-element array and the sender and message fields as shown. Adapt the provider or its endpoint if it expects a different format.

## OTP storage and expiration

Redis keys use the form `otp:<phone_number>`. Each value expires after two minutes. Verification runs a Redis Lua script that checks the submitted code and deletes it in one atomic operation, preventing two concurrent matching requests from both consuming the same code successfully.

Redis is required at runtime. The service pings Redis during startup and exits if the connection check fails.

## Rate limiting

The service applies independent fixed one-minute windows per client IP and procedure:

| Procedure | Limit |
|---|---:|
| `SendOTP` | 5 requests per minute |
| `VerifyOTP` | 10 requests per minute |

The limiter is held in process memory. Limits are therefore per server instance, reset when the process restarts, and are not shared across multiple replicas. The IP is taken from the connection address observed by the service; when deployed behind a proxy, requests may be counted under the proxy's address.

Rate-limited requests return `resource_exhausted`.

## Go client

`NewOTPServiceClient` creates a typed client for both procedures. The base URL is the server origin, including its scheme and port.

```go
client := NewOTPServiceClient(http.DefaultClient, "http://localhost:50051")

request, err := structpb.NewStruct(map[string]any{
	"phone_number": "0241234567",
})
if err != nil {
	return err
}

response, err := client.SendOTP(ctx, request)
if err != nil {
	return err
}
_ = response
```

The snippet requires imports for `net/http` and `google.golang.org/protobuf/types/known/structpb`, plus a `context.Context` value named `ctx`.

## Tests

Run the test suite with:

```bash
go test ./...
```

The current tests use an in-memory OTP store and a recording SMS fake for service behavior, plus an `httptest` server for the Connect client/handler path. They do not require a live Redis instance or SMS provider.

## Operational notes

- Keep `SMS_API_KEY` out of source control and restrict access to deployment environment settings.
- Use TLS at the public deployment boundary; the current `http.Server` configuration does not configure TLS itself.
- The in-memory rate limiter is a basic per-process control, not a distributed abuse-prevention system.
- OTPs are short-lived and one-time, but delivery and verification still depend on the security and availability of Redis, the SMS provider, and the application network.