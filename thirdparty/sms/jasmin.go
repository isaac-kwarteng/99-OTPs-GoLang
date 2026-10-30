package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// jasminSender uses the Jasmin SMS gateway HTTP API (POST /send with form-encoded fields).
type jasminSender struct {
	config     Config
	httpClient *http.Client
}

func (sender *jasminSender) Send(ctx context.Context, phoneNumber, message string) error {
	form := url.Values{
		"username": {sender.config.Username},
		"password": {sender.config.Password},
		"to":       {internationalNumber(phoneNumber)}, // SMPP destinations are digits only
		"from":     {sender.config.SenderID},
		"content":  {message},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, sender.config.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create SMS request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := do(sender.httpClient, request)
	if err != nil {
		return err
	}
	// Jasmin answers `Success "<message-id>"` on acceptance and `Error "<reason>"` otherwise.
	if !strings.HasPrefix(string(body), "Success") {
		return fmt.Errorf("jasmin rejected message: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
