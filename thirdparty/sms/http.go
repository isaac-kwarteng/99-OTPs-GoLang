package sms

import (
	"context"
	"net/http"
)

// httpSender posts to any provider that accepts this JSON body with a bearer token:
// {"phone": ["+233..."], "sender_id": "...", "message": "..."}
type httpSender struct {
	config     Config
	httpClient *http.Client
}

func (sender *httpSender) Send(ctx context.Context, phoneNumber, message string) error {
	payload := struct {
		Phone    []string `json:"phone"`
		SenderID string   `json:"sender_id"`
		Message  string   `json:"message"`
	}{
		Phone:    []string{phoneNumber},
		SenderID: sender.config.SenderID,
		Message:  message,
	}
	_, err := postJSON(ctx, sender.httpClient, sender.config.URL, map[string]string{
		"Authorization": "Bearer " + sender.config.APIKey,
	}, payload)
	return err
}
