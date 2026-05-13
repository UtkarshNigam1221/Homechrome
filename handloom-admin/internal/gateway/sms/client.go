package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/handloom/admin/internal/domain"
)

// Client implements the MSG91 SMS gateway
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new MSG91 SMS client
func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// setJSONHeaders applies the MSG91-standard auth + JSON headers to req.
func (c *Client) setJSONHeaders(req *http.Request) {
	req.Header.Set("authkey", c.config.AuthKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// DevClient is a no-op SMS client that logs OTPs to stdout for local development
type DevClient struct{}

// NewDevClient creates a dev SMS client that prints OTPs to console
func NewDevClient() *DevClient {
	return &DevClient{}
}

// SendOTP logs the OTP to stdout instead of sending a real SMS
func (d *DevClient) SendOTP(_ context.Context, phone, code string) error {
	fmt.Printf("\n╔══════════════════════════════════════╗\n")
	fmt.Printf("║  DEV OTP: %s → %s           ║\n", phone, code)
	fmt.Printf("╚══════════════════════════════════════╝\n\n")
	return nil
}

// SendOTP sends an OTP code to the given phone number via MSG91 Flow API
func (c *Client) SendOTP(ctx context.Context, phone, code string) error {
	payload := otpFlowRequest{
		TemplateID: c.config.OTPTemplateID,
		ShortURL:   "0",
		Recipients: []otpFlowRecipient{{
			Mobiles: phone,
			Var1:    code,
			Var2:    domain.OTPValidityMinutesStr,
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal OTP payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/api/v5/flow/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setJSONHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send OTP SMS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MSG91 returned status %d", resp.StatusCode)
	}

	var result otpFlowResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil // Status 200 is good enough
	}

	if result.Type == "error" {
		return fmt.Errorf("MSG91 error: %s", result.Message)
	}

	return nil
}
