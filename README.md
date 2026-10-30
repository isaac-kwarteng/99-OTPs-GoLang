# Marketdome OTP API

A Connect RPC service that generates six-digit OTPs, sends them by SMS through a configurable provider, and verifies them once. Codes are stored in Redis for two minutes.

## Project layout

```text
cmd/main.go                    service wiring and HTTP server
proto/marketdome/otp/v1/       protobuf API (source of truth)
gen/marketdome/otp/v1/         generated code (buf generate)
pkg/config                     environment-driven configuration
pkg/database                   Redis connection
pkg/data/repositories          OTP storage in Redis
pkg/interceptors               per-IP rate limiting
pkg/routes                     per-procedure rate limits
pkg/servers                    Connect handlers
pkg/services                   OTP business logic
pkg/types                      service, repository and SMS interfaces
thirdparty/sms                 SMS providers (one file each) and the provider factory
jasmin/provision.sh            local Jasmin setup used by docker-compose
```

## Configuration

Copy `.env.sample` to `.env`:

```env
APP.PORT=8080
APP.ENV=development

REDIS.URL=redis://localhost:6379/1

SMS.PROVIDER=arkesel
SMS.API_KEY=your_api_key
SMS.SENDER_ID=YourSenderID
```

### SMS providers

`SMS.PROVIDER` selects how OTPs are delivered. `SMS.SENDER_ID` is always required and must be a sender ID approved by the provider.

| Provider | Required settings | Request sent |
|----------|-------------------|--------------|
| `arkesel` | `SMS.API_KEY` | `POST https://sms.arkesel.com/api/v2/sms/send`, `api-key` header, `{sender, message, recipients: ["233…"]}` |
| `mnotify` | `SMS.API_KEY` | `POST https://api.mnotify.com/api/sms/quick?key=…`, `{recipient: ["0…"], sender, message}` |
| `http` | `SMS.API_URL`, `SMS.API_KEY` | `POST SMS.API_URL`, bearer token, `{phone: ["+233…"], sender_id, message}` |
| `jasmin` | `SMS.API_URL`, `SMS.USERNAME`, `SMS.PASSWORD` | Jasmin HTTP API form post (`username`, `password`, `to`, `from`, `content`) |

For `arkesel` and `mnotify`, `SMS.API_URL` is optional and overrides the default endpoint. Arkesel and mNotify responses must have `"status": "success"`; the `http` provider accepts any 2xx response.

To add a provider, add a file in `thirdparty/sms` that implements `types.SMSSender` and register it in `sms.New`.

## API

The service is `marketdome.otp.v1.OTPService`, defined in `proto/marketdome/otp/v1/service.proto`. Server reflection is enabled.

| RPC | Request | Success response |
|-----|---------|------------------|
| `SendOTP` | `phone_number` (`google.type.PhoneNumber`) | `{ "message": "OTP sent", "expiresIn": 120 }` |
| `VerifyOTP` | `phone_number`, `otp` (6 digits) | `{ "success": true }` |

Phone numbers must be Ghana numbers in E.164 format (`+233XXXXXXXXX`); requests are validated with protovalidate. `SendOTP` is limited to 5 and `VerifyOTP` to 10 requests per minute per client IP. Invalid input and invalid or expired codes return `invalid_argument`, rate limits return `resource_exhausted`, and SMS delivery failures return `unavailable`.

## Local development

For local testing without a real SMS account, use the `jasmin` provider (the defaults in `.env.sample`). `docker-compose.yml` runs Redis, RabbitMQ, Jasmin, and [smscsim](https://github.com/ukarim/smscsim), an SMPP simulator that stands in for a mobile carrier, so no real SMS is sent. A one-shot `jasmin-setup` job (`jasmin/provision.sh`) configures Jasmin through jcli: an SMPP connector to smscsim, a default MT route, and the `otpuser` HTTP API user.

```text
OTP API --HTTP :1401--> Jasmin --SMPP--> smscsim
```

```bash
docker-compose up -d
docker-compose logs jasmin-setup   # wait for "Jasmin provisioning finished"
go run ./cmd

# Request a code
curl --header 'Content-Type: application/json' \
  --data '{"phoneNumber":{"e164Number":"+233241234567"}}' \
  http://localhost:8080/marketdome.otp.v1.OTPService/SendOTP

# Read the code from the logged SMS-MT content
docker-compose exec jasmin tail -f /var/log/jasmin/messages.log

# Verify it
curl --header 'Content-Type: application/json' \
  --data '{"phoneNumber":{"e164Number":"+233241234567"},"otp":"123456"}' \
  http://localhost:8080/marketdome.otp.v1.OTPService/VerifyOTP
```

The Jasmin admin console is available with `telnet localhost 8990` (`jcliadmin` / `jclipwd`). To send real messages, switch `SMS.PROVIDER` to `arkesel`, `mnotify` or `http`.

## Code generation

```bash
buf lint
buf generate
```

The Connect plugin is pinned to `v1.19.1` in `buf.gen.yaml` because newer plugin versions generate code for `connectrpc.com/connect/v2`.

## Tests

```bash
go test ./...
```
