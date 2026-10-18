package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type SMSClient struct {
	endpoint   string
	apiKey     string
	senderID   string
	httpClient *http.Client
}

func NewSMSClient(endpoint, apiKey, senderID string) (*SMSClient, error) {
	parsedURL, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		return nil, fmt.Errorf("SMS_API_URL must be an absolute HTTP(S) URL")
	}
	if apiKey == "" || senderID == "" {
		return nil, fmt.Errorf("SMS_API_KEY and SMS_SENDER_ID are required")
	}
	return &SMSClient{
		endpoint: endpoint,
		apiKey: apiKey,
		senderID: senderID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (client *SMSClient) Send(ctx context.Context, phoneNumber, otp string) error {
	payload := struct {
		Phone    []string `json:"phone"`
		SenderID string   `json:"sender_id"`
		Message  string   `json:"message"`
	}{
		Phone:    []string{phoneNumber},
		SenderID: client.senderID,
		Message:  fmt.Sprintf("Your OTP is %s. It expires in 2 minutes. Do not share it with anyone.", otp),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode SMS request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create SMS request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.apiKey)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send SMS request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("SMS provider returned HTTP %d", response.StatusCode)
	}
	return nil
}