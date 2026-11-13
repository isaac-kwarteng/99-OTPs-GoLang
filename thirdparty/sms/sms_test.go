package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type capturedRequest struct {
	header http.Header
	query  url.Values
	body   string
}

func newProviderServer(t *testing.T, status int, response string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured.header, captured.query, captured.body = r.Header, r.URL.Query(), string(body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func jsonField(t *testing.T, body, field string) any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("request body %q is not JSON: %v", body, err)
	}
	return decoded[field]
}

func TestProvidersSendExpectedRequests(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		status   int
		response string
		wantErr  bool
		check    func(t *testing.T, captured *capturedRequest)
	}{
		{
			name: "http", provider: ProviderHTTP, status: http.StatusOK, response: `{}`,
			check: func(t *testing.T, captured *capturedRequest) {
				if captured.header.Get("Authorization") != "Bearer secret" {
					t.Fatalf("Authorization = %q", captured.header.Get("Authorization"))
				}
				if phones := jsonField(t, captured.body, "phone").([]any); phones[0] != "+233241234567" {
					t.Fatalf("phone = %v", phones)
				}
				if jsonField(t, captured.body, "sender_id") != "TestOTP" {
					t.Fatalf("body = %s", captured.body)
				}
			},
		},
		{name: "http rejected", provider: ProviderHTTP, status: http.StatusUnauthorized, response: `{}`, wantErr: true},
		{
			name: "arkesel", provider: ProviderArkesel, status: http.StatusOK, response: `{"status":"success","data":[]}`,
			check: func(t *testing.T, captured *capturedRequest) {
				if captured.header.Get("api-key") != "secret" {
					t.Fatalf("api-key = %q", captured.header.Get("api-key"))
				}
				if recipients := jsonField(t, captured.body, "recipients").([]any); recipients[0] != "233241234567" {
					t.Fatalf("recipients = %v", recipients)
				}
				if jsonField(t, captured.body, "sender") != "TestOTP" || jsonField(t, captured.body, "message") != "hello" {
					t.Fatalf("body = %s", captured.body)
				}
			},
		},
		{name: "arkesel rejected", provider: ProviderArkesel, status: http.StatusOK, response: `{"status":"error","message":"Insufficient balance"}`, wantErr: true},
		{
			name: "mnotify", provider: ProviderMNotify, status: http.StatusOK, response: `{"status":"success","code":"2000"}`,
			check: func(t *testing.T, captured *capturedRequest) {
				if captured.query.Get("key") != "secret" {
					t.Fatalf("key = %q", captured.query.Get("key"))
				}
				if recipient := jsonField(t, captured.body, "recipient").([]any); recipient[0] != "0241234567" {
					t.Fatalf("recipient = %v", recipient)
				}
				if jsonField(t, captured.body, "is_schedule") != false {
					t.Fatalf("body = %s", captured.body)
				}
			},
		},
		{name: "mnotify rejected", provider: ProviderMNotify, status: http.StatusOK, response: `{"status":"error","message":"Invalid sender"}`, wantErr: true},
		{
			name: "jasmin", provider: ProviderJasmin, status: http.StatusOK, response: `Success "abc-123"`,
			check: func(t *testing.T, captured *capturedRequest) {
				form, err := url.ParseQuery(captured.body)
				if err != nil {
					t.Fatal(err)
				}
				if form.Get("to") != "233241234567" || form.Get("username") != "otpuser" || form.Get("from") != "TestOTP" {
					t.Fatalf("form = %v", form)
				}
			},
		},
		{name: "jasmin rejected", provider: ProviderJasmin, status: http.StatusPreconditionFailed, response: `Error "No route found"`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := newProviderServer(t, tt.status, tt.response)
			sender, err := New(Config{
				Provider: tt.provider,
				URL:      server.URL,
				APIKey:   "secret",
				Username: "otpuser",
				Password: "otppass",
				SenderID: "TestOTP",
			})
			if err != nil {
				t.Fatal(err)
			}

			err = sender.Send(context.Background(), "+233241234567", "hello")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, captured)
			}
		})
	}
}

func TestNewRejectsIncompleteConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "unknown provider", config: Config{Provider: "carrier-pigeon", SenderID: "TestOTP"}},
		{name: "missing sender", config: Config{Provider: ProviderArkesel, APIKey: "secret"}},
		{name: "http without url", config: Config{Provider: ProviderHTTP, APIKey: "secret", SenderID: "TestOTP"}},
		{name: "arkesel without key", config: Config{Provider: ProviderArkesel, SenderID: "TestOTP"}},
		{name: "mnotify without key", config: Config{Provider: ProviderMNotify, SenderID: "TestOTP"}},
		{name: "jasmin without password", config: Config{Provider: ProviderJasmin, URL: "http://localhost:1401/send", Username: "otpuser", SenderID: "TestOTP"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.config); err == nil {
				t.Fatal("New() error = nil, want configuration error")
			}
		})
	}
}
