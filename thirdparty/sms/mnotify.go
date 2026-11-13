package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

const mNotifyURL = "https://api.mnotify.com/api/sms/quick"

// mNotifySender uses the mNotify quick SMS API; the API key is passed as the `key` query parameter.
type mNotifySender struct {
	config     Config
	httpClient *http.Client
}

func (sender *mNotifySender) Send(ctx context.Context, phoneNumber, message string) error {
	endpoint, err := url.Parse(sender.config.URL)
	if err != nil {
		return fmt.Errorf("parse mNotify URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("key", sender.config.APIKey)
	endpoint.RawQuery = query.Encode()

	payload := struct {
		Recipient    []string `json:"recipient"`
		Sender       string   `json:"sender"`
		Message      string   `json:"message"`
		IsSchedule   bool     `json:"is_schedule"`
		ScheduleDate string   `json:"schedule_date"`
	}{
		Recipient: []string{localGhanaNumber(phoneNumber)},
		Sender:    sender.config.SenderID,
		Message:   message,
	}
	body, err := postJSON(ctx, sender.httpClient, endpoint.String(), nil, payload)
	if err != nil {
		return err
	}
	return requireStatusSuccess(body)
}
