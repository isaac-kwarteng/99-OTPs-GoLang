package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/marketdome/otp-api/pkg/types"
)

const (
	ProviderHTTP    = "http"
	ProviderArkesel = "arkesel"
	ProviderMNotify = "mnotify"
	ProviderJasmin  = "jasmin"
)

// Config holds the settings for every provider; each provider uses the subset it needs.
type Config struct {
	Provider string
	URL      string // required for http and jasmin; overrides the default endpoint for arkesel and mnotify
	APIKey   string // http, arkesel, mnotify
	Username string // jasmin
	Password string // jasmin
	SenderID string
}

// New returns the SMS sender for config.Provider.
func New(config Config) (types.SMSSender, error) {
	if config.SenderID == "" {
		return nil, fmt.Errorf("SMS.SENDER_ID is required")
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}

	switch strings.ToLower(config.Provider) {
	case ProviderHTTP:
		if err := requireURL(config.URL); err != nil {
			return nil, err
		}
		if config.APIKey == "" {
			return nil, fmt.Errorf("SMS.API_KEY is required for provider %q", ProviderHTTP)
		}
		return &httpSender{config: config, httpClient: httpClient}, nil
	case ProviderArkesel:
		config.URL = withDefault(config.URL, arkeselURL)
		if config.APIKey == "" {
			return nil, fmt.Errorf("SMS.API_KEY is required for provider %q", ProviderArkesel)
		}
		return &arkeselSender{config: config, httpClient: httpClient}, nil
	case ProviderMNotify:
		config.URL = withDefault(config.URL, mNotifyURL)
		if config.APIKey == "" {
			return nil, fmt.Errorf("SMS.API_KEY is required for provider %q", ProviderMNotify)
		}
		return &mNotifySender{config: config, httpClient: httpClient}, nil
	case ProviderJasmin:
		if err := requireURL(config.URL); err != nil {
			return nil, err
		}
		if config.Username == "" || config.Password == "" {
			return nil, fmt.Errorf("SMS.USERNAME and SMS.PASSWORD are required for provider %q", ProviderJasmin)
		}
		return &jasminSender{config: config, httpClient: httpClient}, nil
	default:
		return nil, fmt.Errorf("unsupported SMS.PROVIDER %q (use %s, %s, %s or %s)",
			config.Provider, ProviderHTTP, ProviderArkesel, ProviderMNotify, ProviderJasmin)
	}
}

func requireURL(endpoint string) error {
	parsedURL, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		return fmt.Errorf("SMS.API_URL must be an absolute HTTP(S) URL")
	}
	return nil
}

func withDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// internationalNumber converts +233XXXXXXXXX to 233XXXXXXXXX.
func internationalNumber(e164 string) string {
	return strings.TrimPrefix(e164, "+")
}

// localGhanaNumber converts +233XXXXXXXXX to 0XXXXXXXXX.
func localGhanaNumber(e164 string) string {
	if strings.HasPrefix(e164, "+233") {
		return "0" + strings.TrimPrefix(e164, "+233")
	}
	return e164
}

// postJSON sends payload as JSON and returns the response body when the status is 2xx.
func postJSON(ctx context.Context, httpClient *http.Client, endpoint string, headers map[string]string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode SMS request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create SMS request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return do(httpClient, request)
}

func do(httpClient *http.Client, request *http.Request) ([]byte, error) {
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send SMS request: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("SMS provider returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// requireStatusSuccess checks the `{"status":"success"}` envelope used by Arkesel and mNotify.
func requireStatusSuccess(body []byte) error {
	var envelope struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode SMS response: %w", err)
	}
	if !strings.EqualFold(envelope.Status, "success") {
		return fmt.Errorf("SMS provider rejected message: status=%q message=%q", envelope.Status, envelope.Message)
	}
	return nil
}
