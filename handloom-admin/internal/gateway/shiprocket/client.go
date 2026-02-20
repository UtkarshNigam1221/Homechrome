package shiprocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client implements the Shiprocket shipping gateway
type Client struct {
	config      Config
	httpClient  *http.Client
	tokenMu     sync.Mutex
	cachedToken *tokenCache
}

// NewClient creates a new Shiprocket client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://apiv2.shiprocket.in/v1/external"
	}
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate gets an auth token from Shiprocket (cached for 24h)
func (c *Client) Authenticate(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.cachedToken != nil && time.Now().Before(c.cachedToken.expiresAt) {
		return c.cachedToken.token, nil
	}

	payload, _ := json.Marshal(map[string]string{
		"email":    c.config.Email,
		"password": c.config.Password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate with Shiprocket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Shiprocket auth returned status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}

	c.cachedToken = &tokenCache{
		token:     authResp.Token,
		expiresAt: time.Now().Add(24 * time.Hour),
	}

	return authResp.Token, nil
}

// CheckServiceability checks courier availability for a given route
func (c *Client) CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightKG float64) (*CourierServiceabilityResponse, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/courier/serviceability/?pickup_postcode=%s&delivery_postcode=%s&weight=%f&cod=0",
		c.config.BaseURL, pickupPincode, deliveryPincode, weightKG)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check serviceability: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result CourierServiceabilityResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse serviceability response: %w", err)
	}

	return &result, nil
}

// CreateOrder creates a new shipment order in Shiprocket
func (c *Client) CreateOrder(ctx context.Context, order *CreateOrderRequest) (*CreateOrderResponse, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	body, _ := json.Marshal(order)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/orders/create/adhoc", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create Shiprocket order: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result CreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create order response: %w", err)
	}

	return &result, nil
}

// AssignAWB assigns an AWB number to a shipment
func (c *Client) AssignAWB(ctx context.Context, shipmentID, courierID int) (*AssignAWBResponse, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	body, _ := json.Marshal(map[string]int{
		"shipment_id": shipmentID,
		"courier_id":  courierID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/courier/assign/awb", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to assign AWB: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result AssignAWBResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AWB response: %w", err)
	}

	return &result, nil
}

// GenerateLabel generates a shipping label
func (c *Client) GenerateLabel(ctx context.Context, shipmentID int) (string, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return "", err
	}

	body, _ := json.Marshal(map[string][]int{
		"shipment_id": {shipmentID},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/courier/generate/label", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate label: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result GenerateLabelResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse label response: %w", err)
	}

	return result.LabelURL, nil
}

// TrackByAWB tracks a shipment by AWB number
func (c *Client) TrackByAWB(ctx context.Context, awb string) (*TrackingResponse, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/courier/track/awb/"+awb, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to track shipment: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result TrackingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tracking response: %w", err)
	}

	return &result, nil
}
