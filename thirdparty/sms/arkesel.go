package sms

import (
	"context"
	"net/http"
)

const arkeselURL = "https://sms.arkesel.com/api/v2/sms/send"

// arkeselSender uses the Arkesel SMS v2 API.
type arkeselSender struct {
	config     Config
	httpClient *http.Client
}

func (sender *arkeselSender) Send(ctx context.Context, phoneNumber, message string) error {
	payload := struct {
		Sender     string   `json:"sender"`
		Message    string   `json:"message"`
		Recipients []string `json:"recipients"`
	}{
		Sender:     sender.config.SenderID,
		Message:    message,
		Recipients: []string{internationalNumber(phoneNumber)},
	}
	body, err := postJSON(ctx, sender.httpClient, sender.config.URL, map[string]string{
		"api-key": sender.config.APIKey,
	}, payload)
	if err != nil {
		return err
	}
	return requireStatusSuccess(body)
}
