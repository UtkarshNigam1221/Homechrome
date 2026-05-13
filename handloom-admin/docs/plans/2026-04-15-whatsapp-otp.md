# WhatsApp OTP Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add WhatsApp as a user-selectable OTP delivery channel alongside SMS for the homechrome-store login flow.

**Architecture:** New `WhatsAppGateway` in `internal/gateway/whatsapp/` alongside existing `SMSGateway`. The `CustomerAuthService.SendOTP` method gains a `channel` parameter and dispatches to the correct gateway. Frontend shows two buttons ("Send via WhatsApp" / "Send via SMS"). OTP generation, storage, and verification are unchanged.

**Tech Stack:** Go 1.25, Meta WhatsApp Cloud API (Graph API v21.0), React 19 / Next.js 16, Zustand

**Spec:** `docs/specs/2026-04-15-whatsapp-otp-design.md`

---

### Task 1: Create WhatsApp Gateway Package

**Files:**
- Create: `internal/gateway/whatsapp/types.go`
- Create: `internal/gateway/whatsapp/client.go`
- Create: `internal/gateway/whatsapp/dev_client.go`

- [ ] **Step 1: Create `internal/gateway/whatsapp/types.go`**

```go
package whatsapp

import "context"

// Config holds WhatsApp Cloud API configuration
type Config struct {
	AccessToken     string
	PhoneNumberID   string
	OTPTemplateName string
}

// WhatsAppGateway defines the interface for sending WhatsApp messages
type WhatsAppGateway interface {
	SendOTP(ctx context.Context, phone, code string) error
}
```

- [ ] **Step 2: Create `internal/gateway/whatsapp/dev_client.go`**

```go
package whatsapp

import (
	"context"
	"fmt"
)

// DevClient is a no-op WhatsApp client that logs OTPs to stdout for local development
type DevClient struct{}

// NewDevClient creates a dev WhatsApp client that prints OTPs to console
func NewDevClient() *DevClient {
	return &DevClient{}
}

// SendOTP logs the OTP to stdout instead of sending a real WhatsApp message
func (d *DevClient) SendOTP(_ context.Context, phone, code string) error {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV WhatsApp OTP: %s → %s           ║\n", phone, code)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")
	return nil
}
```

- [ ] **Step 3: Create `internal/gateway/whatsapp/client.go`**

```go
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://graph.facebook.com/v21.0"

// Client implements the WhatsApp Cloud API gateway
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new WhatsApp Cloud API client
func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// templatePayload builds the WhatsApp authentication template message
func templatePayload(to, templateName, code string) map[string]interface{} {
	return map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name":     templateName,
			"language": map[string]string{"code": "en"},
			"components": []map[string]interface{}{
				{
					"type": "body",
					"parameters": []map[string]string{
						{"type": "text", "text": code},
					},
				},
				{
					"type":     "button",
					"sub_type": "url",
					"index":    "0",
					"parameters": []map[string]string{
						{"type": "text", "text": code},
					},
				},
			},
		},
	}
}

// SendOTP sends an OTP code to the given phone number via WhatsApp Cloud API
func (c *Client) SendOTP(ctx context.Context, phone, code string) error {
	payload := templatePayload(phone, c.config.OTPTemplateName, code)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", baseURL, c.config.PhoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp OTP: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WhatsApp API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Messages []struct {
			ID            string `json:"id"`
			MessageStatus string `json:"message_status"`
		} `json:"messages"`
		Error *struct {
			Code    int    `json:"code"`
			Title   string `json:"title"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil // Status 2xx is good enough
	}

	if result.Error != nil {
		return fmt.Errorf("WhatsApp API error %d: %s", result.Error.Code, result.Error.Title)
	}

	return nil
}
```

- [ ] **Step 4: Verify the package compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./internal/gateway/whatsapp/`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/gateway/whatsapp/
git commit -m "feat: add WhatsApp gateway package with Cloud API client and DevClient"
```

---

### Task 2: Add Domain Interface and Update SendOTPRequest

**Files:**
- Modify: `internal/domain/store_service.go` (add `WhatsAppGateway` interface, update `CustomerAuthService.SendOTP` signature)
- Modify: `internal/domain/entity.go` (add `Channel` field to `SendOTPRequest`)

- [ ] **Step 1: Add `WhatsAppGateway` interface to `internal/domain/store_service.go`**

Add after the existing `SMSGateway` interface (after line 8):

```go
// WhatsAppGateway defines the WhatsApp messaging interface
type WhatsAppGateway interface {
	SendOTP(ctx context.Context, phone, code string) error
}
```

- [ ] **Step 2: Update `CustomerAuthService.SendOTP` signature in `internal/domain/store_service.go`**

Change `SendOTP(ctx context.Context, phone string) error` to:

```go
SendOTP(ctx context.Context, phone, channel string) error
```

- [ ] **Step 3: Add `Channel` field to `SendOTPRequest` in `internal/domain/entity.go`**

Change (around line 679):

```go
type SendOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164"`
}
```

To:

```go
type SendOTPRequest struct {
	Phone   string `json:"phone" validate:"required,e164"`
	Channel string `json:"channel" validate:"required,oneof=sms whatsapp"`
}
```

- [ ] **Step 4: Verify domain compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./internal/domain/`
Expected: No errors (domain has no upstream dependencies)

- [ ] **Step 5: Commit**

```bash
git add internal/domain/store_service.go internal/domain/entity.go
git commit -m "feat: add WhatsAppGateway domain interface and channel field to SendOTPRequest"
```

---

### Task 3: Update CustomerAuthService Implementation

**Files:**
- Modify: `internal/service/customer_auth_service.go` (add whatsappGateway field, update constructor and SendOTP)

- [ ] **Step 1: Add `whatsappGateway` field to the struct**

In `internal/service/customer_auth_service.go`, add after `smsGateway domain.SMSGateway` (line 41):

```go
whatsappGateway domain.WhatsAppGateway
```

- [ ] **Step 2: Update the constructor to accept `whatsappGateway`**

Change `NewCustomerAuthService` (line 48) from:

```go
func NewCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	smsGateway domain.SMSGateway,
	publisher event.EventPublisher,
	cfg CustomerAuthConfig,
) *CustomerAuthService {
	return &CustomerAuthService{
		otpRepo:      otpRepo,
		customerRepo: customerRepo,
		tokenStore:   tokenStore,
		smsGateway:   smsGateway,
		publisher:    publisher,
		config:       cfg,
		jwtSecret:    []byte(cfg.JWTSecret),
	}
}
```

To:

```go
func NewCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	smsGateway domain.SMSGateway,
	whatsappGateway domain.WhatsAppGateway,
	publisher event.EventPublisher,
	cfg CustomerAuthConfig,
) *CustomerAuthService {
	return &CustomerAuthService{
		otpRepo:         otpRepo,
		customerRepo:    customerRepo,
		tokenStore:      tokenStore,
		smsGateway:      smsGateway,
		whatsappGateway: whatsappGateway,
		publisher:       publisher,
		config:          cfg,
		jwtSecret:       []byte(cfg.JWTSecret),
	}
}
```

- [ ] **Step 3: Update `SendOTP` method to accept and dispatch on `channel`**

Change the `SendOTP` method (line 68) from:

```go
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone string) error {
	code, err := generateOTPCode()
	if err != nil {
		return errors.Internal("Failed to generate OTP code")
	}

	otp := &domain.OTP{
		Phone:    phone,
		CodeHash: hashSHA256(code),
		Attempts: 0,
	}

	if err := s.otpRepo.Store(ctx, otp); err != nil {
		return err
	}

	if err := s.smsGateway.SendOTP(ctx, phone, code); err != nil {
		slog.ErrorContext(ctx, "Failed to send OTP SMS", "error", err)
		return errors.Internal("Failed to send OTP")
	}

	slog.InfoContext(ctx, "OTP sent", "phone", phone)
	return nil
}
```

To:

```go
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone, channel string) error {
	code, err := generateOTPCode()
	if err != nil {
		return errors.Internal("Failed to generate OTP code")
	}

	otp := &domain.OTP{
		Phone:    phone,
		CodeHash: hashSHA256(code),
		Attempts: 0,
	}

	if err := s.otpRepo.Store(ctx, otp); err != nil {
		return err
	}

	switch channel {
	case "whatsapp":
		err = s.whatsappGateway.SendOTP(ctx, phone, code)
	default:
		err = s.smsGateway.SendOTP(ctx, phone, code)
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send OTP", "channel", channel, "error", err)
		return errors.Internal("Failed to send OTP")
	}

	slog.InfoContext(ctx, "OTP sent", "phone", phone, "channel", channel)
	return nil
}
```

- [ ] **Step 4: Verify service compiles (expect compilation errors from callers — that's expected)**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./internal/service/`
Expected: May have errors from callers; the package itself should be valid.

- [ ] **Step 5: Commit**

```bash
git add internal/service/customer_auth_service.go
git commit -m "feat: update CustomerAuthService with WhatsApp gateway and channel dispatch"
```

---

### Task 4: Update Handler to Pass Channel

**Files:**
- Modify: `internal/handler/store/auth_handler.go` (pass `req.Channel` to `SendOTP`)

- [ ] **Step 1: Update SendOTP handler**

In `internal/handler/store/auth_handler.go`, change the `SendOTP` method (line 122) from:

```go
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.SendOTPRequest](ctx)

	if err := h.customerAuthService.SendOTP(ctx, req.Phone); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "OTP sent successfully",
	})
}
```

To:

```go
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.SendOTPRequest](ctx)

	if err := h.customerAuthService.SendOTP(ctx, req.Phone, req.Channel); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "OTP sent successfully",
	})
}
```

- [ ] **Step 2: Verify handler compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./internal/handler/store/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/handler/store/auth_handler.go
git commit -m "feat: pass OTP channel from request to auth service in store handler"
```

---

### Task 5: Add Config and Wire Up WhatsApp Gateway

**Files:**
- Modify: `internal/config/config.go` (add 3 WhatsApp fields to `StoreConfig`, load them in `Load()`)
- Modify: `internal/wire/providers.go` (update `ProvideCustomerAuthService` to create WhatsApp gateway)
- Modify: `cmd/api/main.go` (add WhatsApp gateway to monolith wiring)

- [ ] **Step 1: Add WhatsApp fields to `StoreConfig` in `internal/config/config.go`**

Add after the MSG91 block (after line 55):

```go
// WhatsApp OTP
WhatsAppAccessToken     string
WhatsAppPhoneNumberID   string
WhatsAppOTPTemplateName string
```

- [ ] **Step 2: Load WhatsApp env vars in `Load()` in `internal/config/config.go`**

Add after the MSG91 loading lines (after line 174):

```go
WhatsAppAccessToken:     getEnv("WHATSAPP_ACCESS_TOKEN", ""),
WhatsAppPhoneNumberID:   getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
WhatsAppOTPTemplateName: getEnv("WHATSAPP_OTP_TEMPLATE_NAME", ""),
```

- [ ] **Step 3: Update `ProvideCustomerAuthService` in `internal/wire/providers.go`**

Change from (line 537):

```go
func ProvideCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	publisher event.EventPublisher,
	cfg *config.Config,
) *service.CustomerAuthService {
	var smsGateway domain.SMSGateway
	if cfg.Store.MSG91AuthKey == "" || cfg.Store.MSG91OTPTemplateID == "" {
		smsGateway = sms.NewDevClient()
	} else {
		smsGateway = sms.NewClient(sms.Config{
			AuthKey:       cfg.Store.MSG91AuthKey,
			OTPTemplateID: cfg.Store.MSG91OTPTemplateID,
			BaseURL:       cfg.Store.MSG91BaseURL,
		})
	}
	return service.NewCustomerAuthService(
		otpRepo, customerRepo, tokenStore, smsGateway, publisher,
		service.CustomerAuthConfig{
			JWTSecret:            cfg.Store.CustomerJWTSecret,
			AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
			RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
			Issuer:               "handloom-store",
		},
	)
}
```

To:

```go
func ProvideCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	publisher event.EventPublisher,
	cfg *config.Config,
) *service.CustomerAuthService {
	var smsGateway domain.SMSGateway
	if cfg.Store.MSG91AuthKey == "" || cfg.Store.MSG91OTPTemplateID == "" {
		smsGateway = sms.NewDevClient()
	} else {
		smsGateway = sms.NewClient(sms.Config{
			AuthKey:       cfg.Store.MSG91AuthKey,
			OTPTemplateID: cfg.Store.MSG91OTPTemplateID,
			BaseURL:       cfg.Store.MSG91BaseURL,
		})
	}

	var whatsappGateway domain.WhatsAppGateway
	if cfg.Store.WhatsAppAccessToken == "" || cfg.Store.WhatsAppPhoneNumberID == "" {
		whatsappGateway = whatsapp.NewDevClient()
	} else {
		whatsappGateway = whatsapp.NewClient(whatsapp.Config{
			AccessToken:     cfg.Store.WhatsAppAccessToken,
			PhoneNumberID:   cfg.Store.WhatsAppPhoneNumberID,
			OTPTemplateName: cfg.Store.WhatsAppOTPTemplateName,
		})
	}

	return service.NewCustomerAuthService(
		otpRepo, customerRepo, tokenStore, smsGateway, whatsappGateway, publisher,
		service.CustomerAuthConfig{
			JWTSecret:            cfg.Store.CustomerJWTSecret,
			AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
			RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
			Issuer:               "handloom-store",
		},
	)
}
```

Add the import for the whatsapp package at the top of `providers.go`:

```go
"github.com/handloom/admin/internal/gateway/whatsapp"
```

- [ ] **Step 4: Update monolith wiring in `cmd/api/main.go`**

First, add import:

```go
"github.com/handloom/admin/internal/gateway/whatsapp"
```

Then find the `customerAuthService` creation (around line 199) and add WhatsApp gateway creation before it. Change from:

```go
// B2C services
customerAuthService := service.NewCustomerAuthService(
	otpRepo,
	customerRepo,
	customerTokenStore,
	smsGateway,
	publisher,
	service.CustomerAuthConfig{
		JWTSecret:            cfg.Store.CustomerJWTSecret,
		AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
		RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
		Issuer:               "handloom-store",
	},
)
```

To:

```go
// WhatsApp gateway
var whatsappGateway domain.WhatsAppGateway
if cfg.Store.WhatsAppAccessToken == "" || cfg.Store.WhatsAppPhoneNumberID == "" {
	whatsappGateway = whatsapp.NewDevClient()
} else {
	whatsappGateway = whatsapp.NewClient(whatsapp.Config{
		AccessToken:     cfg.Store.WhatsAppAccessToken,
		PhoneNumberID:   cfg.Store.WhatsAppPhoneNumberID,
		OTPTemplateName: cfg.Store.WhatsAppOTPTemplateName,
	})
}

// B2C services
customerAuthService := service.NewCustomerAuthService(
	otpRepo,
	customerRepo,
	customerTokenStore,
	smsGateway,
	whatsappGateway,
	publisher,
	service.CustomerAuthConfig{
		JWTSecret:            cfg.Store.CustomerJWTSecret,
		AccessTokenDuration:  cfg.Store.CustomerAccessTokenTTL,
		RefreshTokenDuration: cfg.Store.CustomerRefreshTokenTTL,
		Issuer:               "handloom-store",
	},
)
```

- [ ] **Step 5: Regenerate Wire**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make wire`
Expected: `wire_gen.go` regenerated successfully

- [ ] **Step 6: Regenerate mocks** (domain interfaces changed)

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make generate-mocks`
Expected: Mocks regenerated. `internal/mocks/store_service_mock.go` now has `MockWhatsAppGateway` and updated `MockCustomerAuthService.SendOTP` with `channel` parameter.

- [ ] **Step 7: Verify full backend compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./...`
Expected: No errors

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/wire/providers.go internal/wire/wire_gen.go internal/mocks/ cmd/api/main.go
git commit -m "feat: wire WhatsApp gateway into config, DI, and monolith entry point"
```

---

### Task 6: Add CDK and Environment Configuration

**Files:**
- Modify: `infra/stacks/api.go` (add 3 WhatsApp env vars to `commonEnv`)
- Modify: `.env.example` (document new vars)

- [ ] **Step 1: Add WhatsApp env vars to CDK `commonEnv` in `infra/stacks/api.go`**

Add after the PhonePe env var block (after line 110):

```go
// Add WhatsApp OTP env vars when configured
for _, key := range []string{"WHATSAPP_ACCESS_TOKEN", "WHATSAPP_PHONE_NUMBER_ID", "WHATSAPP_OTP_TEMPLATE_NAME"} {
	if v := os.Getenv(key); v != "" {
		commonEnv[key] = jsii.String(v)
	}
}
```

- [ ] **Step 2: Add vars to `.env.example`**

Add after the MSG91 block (after line 75):

```
# WhatsApp OTP (empty = DevClient that prints to console)
WHATSAPP_ACCESS_TOKEN=
WHATSAPP_PHONE_NUMBER_ID=
WHATSAPP_OTP_TEMPLATE_NAME=
```

- [ ] **Step 3: Verify CDK compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin/infra && go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add infra/stacks/api.go .env.example
git commit -m "feat: add WhatsApp OTP env vars to CDK and .env.example"
```

---

### Task 7: Write Backend Unit Tests

**Files:**
- Create: `internal/gateway/whatsapp/client_test.go`
- Create: `internal/service/customer_auth_service_test.go`

- [ ] **Step 1: Write WhatsApp client test in `internal/gateway/whatsapp/client_test.go`**

```go
package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_SendOTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and auth
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		// Verify body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if payload["messaging_product"] != "whatsapp" {
			t.Errorf("expected messaging_product=whatsapp, got %v", payload["messaging_product"])
		}
		if payload["to"] != "+919876543210" {
			t.Errorf("expected to=+919876543210, got %v", payload["to"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{
				{"id": "wamid.xxx", "message_status": "accepted"},
			},
		})
	}))
	defer server.Close()

	// Override baseURL for test
	client := &Client{
		config: Config{
			AccessToken:     "test-token",
			PhoneNumberID:   "12345",
			OTPTemplateName: "otp_template",
		},
		httpClient: server.Client(),
	}
	// Point to test server (we need to override the URL the client calls)
	origBaseURL := baseURL
	defer func() { /* baseURL is a const, so we test via httptest transport instead */ }()
	_ = origBaseURL

	// Use a custom transport to redirect requests to test server
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	err := client.SendOTP(context.Background(), "+919876543210", "123456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestClient_SendOTP_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":100,"title":"Invalid parameter"}}`))
	}))
	defer server.Close()

	client := &Client{
		config: Config{
			AccessToken:     "test-token",
			PhoneNumberID:   "12345",
			OTPTemplateName: "otp_template",
		},
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				req.URL.Scheme = "http"
				req.URL.Host = server.Listener.Addr().String()
				return http.DefaultTransport.RoundTrip(req)
			}),
		},
	}

	err := client.SendOTP(context.Background(), "+919876543210", "123456")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// roundTripFunc adapts a function to http.RoundTripper
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
```

- [ ] **Step 2: Run WhatsApp client tests**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go test -v ./internal/gateway/whatsapp/`
Expected: PASS

- [ ] **Step 3: Write SendOTP channel dispatch test in `internal/service/customer_auth_service_test.go`**

```go
package service

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
)

func newTestCustomerAuthService(t *testing.T) (
	*CustomerAuthService,
	*mocks.MockOTPRepository,
	*mocks.MockSMSGateway,
	*mocks.MockWhatsAppGateway,
) {
	ctrl := gomock.NewController(t)
	otpRepo := mocks.NewMockOTPRepository(ctrl)
	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	tokenStore := mocks.NewMockCustomerTokenStore(ctrl)
	smsGW := mocks.NewMockSMSGateway(ctrl)
	waGW := mocks.NewMockWhatsAppGateway(ctrl)
	publisher := newSpyPublisher()

	svc := NewCustomerAuthService(
		otpRepo, customerRepo, tokenStore, smsGW, waGW, publisher,
		CustomerAuthConfig{
			JWTSecret:            "test-secret",
			AccessTokenDuration:  900000000000,  // 15m in nanoseconds
			RefreshTokenDuration: 6048000000000000, // 7d
			Issuer:               "test",
		},
	)
	return svc, otpRepo, smsGW, waGW
}

func TestSendOTP_SMSChannel(t *testing.T) {
	svc, otpRepo, smsGW, _ := newTestCustomerAuthService(t)
	ctx := context.Background()

	otpRepo.EXPECT().Store(ctx, gomock.Any()).Return(nil)
	smsGW.EXPECT().SendOTP(ctx, "+919876543210", gomock.Any()).Return(nil)

	err := svc.SendOTP(ctx, "+919876543210", "sms")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendOTP_WhatsAppChannel(t *testing.T) {
	svc, otpRepo, _, waGW := newTestCustomerAuthService(t)
	ctx := context.Background()

	otpRepo.EXPECT().Store(ctx, gomock.Any()).Return(nil)
	waGW.EXPECT().SendOTP(ctx, "+919876543210", gomock.Any()).Return(nil)

	err := svc.SendOTP(ctx, "+919876543210", "whatsapp")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendOTP_DefaultsToSMS(t *testing.T) {
	svc, otpRepo, smsGW, _ := newTestCustomerAuthService(t)
	ctx := context.Background()

	otpRepo.EXPECT().Store(ctx, gomock.Any()).Return(nil)
	smsGW.EXPECT().SendOTP(ctx, "+919876543210", gomock.Any()).Return(nil)

	err := svc.SendOTP(ctx, "+919876543210", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendOTP_WhatsAppFailure(t *testing.T) {
	svc, otpRepo, _, waGW := newTestCustomerAuthService(t)
	ctx := context.Background()

	otpRepo.EXPECT().Store(ctx, gomock.Any()).Return(nil)
	waGW.EXPECT().SendOTP(ctx, "+919876543210", gomock.Any()).Return(
		apperrors.Internal("WhatsApp API error"),
	)

	err := svc.SendOTP(ctx, "+919876543210", "whatsapp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

Add this import to the test file alongside the others:

```go
apperrors "github.com/handloom/admin/pkg/errors"
```

- [ ] **Step 4: Run customer auth service tests**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go test -v ./internal/service/ -run TestSendOTP`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Run the full test suite to check for regressions**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make test`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/whatsapp/client_test.go internal/service/customer_auth_service_test.go
git commit -m "test: add WhatsApp gateway client and channel dispatch unit tests"
```

---

### Task 8: Update Storefront Frontend

**Files:**
- Modify: `homechrome-store/src/stores/auth.ts` (add `channel` param to `sendOTP`)
- Modify: `homechrome-store/src/app/login/page.tsx` (two buttons, channel state, updated messages)

- [ ] **Step 1: Update auth store in `homechrome-store/src/stores/auth.ts`**

Change the `sendOTP` function and its type from:

```typescript
sendOTP: (phone: string) => Promise<void>;
```

To:

```typescript
sendOTP: (phone: string, channel: 'sms' | 'whatsapp') => Promise<void>;
```

And the implementation from:

```typescript
sendOTP: async (phone: string) => {
  await api.post(ROUTES.AUTH.SEND_OTP, { phone });
},
```

To:

```typescript
sendOTP: async (phone: string, channel: 'sms' | 'whatsapp') => {
  await api.post(ROUTES.AUTH.SEND_OTP, { phone, channel });
},
```

- [ ] **Step 2: Update login page in `homechrome-store/src/app/login/page.tsx`**

Add `channel` state after the existing state declarations (around line 23):

```typescript
const [channel, setChannel] = useState<'sms' | 'whatsapp'>('sms');
```

Update `handleSendOTP` to accept and use a channel parameter. Change from:

```typescript
const handleSendOTP = useCallback(async () => {
  const cleaned = phone.replace(/\D/g, '');
  if (cleaned.length !== 10) {
    setError('Please enter a valid 10-digit phone number.');
    return;
  }
  setError('');
  setSending(true);
  try {
    await sendOTP(`+91${cleaned}`);
    setStep('otp');
    startCountdown(OTP_RESEND_SECONDS);
  } catch {
    setError('Failed to send OTP. Please try again.');
  } finally {
    setSending(false);
  }
}, [phone, sendOTP, startCountdown]);
```

To:

```typescript
const handleSendOTP = useCallback(
  async (selectedChannel: 'sms' | 'whatsapp') => {
    const cleaned = phone.replace(/\D/g, '');
    if (cleaned.length !== 10) {
      setError('Please enter a valid 10-digit phone number.');
      return;
    }
    setError('');
    setSending(true);
    try {
      await sendOTP(`+91${cleaned}`, selectedChannel);
      setChannel(selectedChannel);
      setStep('otp');
      startCountdown(OTP_RESEND_SECONDS);
    } catch {
      setError('Failed to send OTP. Please try again.');
    } finally {
      setSending(false);
    }
  },
  [phone, sendOTP, startCountdown],
);
```

Update `handleResendOTP` to use the stored channel. Change from:

```typescript
const handleResendOTP = useCallback(async () => {
  if (countdown > 0) return;
  setError('');
  setSending(true);
  const cleaned = phone.replace(/\D/g, '');
  try {
    await sendOTP(`+91${cleaned}`);
    startCountdown(OTP_RESEND_SECONDS);
  } catch {
    setError('Failed to resend OTP. Please try again.');
  } finally {
    setSending(false);
  }
}, [countdown, phone, sendOTP, startCountdown]);
```

To:

```typescript
const handleResendOTP = useCallback(async () => {
  if (countdown > 0) return;
  setError('');
  setSending(true);
  const cleaned = phone.replace(/\D/g, '');
  try {
    await sendOTP(`+91${cleaned}`, channel);
    startCountdown(OTP_RESEND_SECONDS);
  } catch {
    setError('Failed to resend OTP. Please try again.');
  } finally {
    setSending(false);
  }
}, [countdown, phone, sendOTP, startCountdown, channel]);
```

Replace the phone step form submit and single button. Change from:

```tsx
<form
  onSubmit={(e) => {
    e.preventDefault();
    handleSendOTP();
  }}
>
```

To (remove `onSubmit` since we have two buttons now):

```tsx
<div>
```

And change the closing `</form>` to `</div>`.

Replace the single "Send OTP" button:

```tsx
<Button
  type="submit"
  variant="primary"
  size="default"
  className="mt-4 w-full"
  loading={sending}
  disabled={phone.replace(/\D/g, '').length !== 10}
>
  Send OTP
</Button>
```

With two side-by-side buttons:

```tsx
<div className="mt-4 flex gap-3">
  <Button
    variant="primary"
    size="default"
    className="flex-1"
    loading={sending}
    disabled={phone.replace(/\D/g, '').length !== 10}
    onClick={() => handleSendOTP('whatsapp')}
  >
    Send via WhatsApp
  </Button>
  <Button
    variant="outline"
    size="default"
    className="flex-1"
    loading={sending}
    disabled={phone.replace(/\D/g, '').length !== 10}
    onClick={() => handleSendOTP('sms')}
  >
    Send via SMS
  </Button>
</div>
```

Update the OTP step message to show which channel was used. Change from:

```tsx
<p className="mb-4 text-sm text-muted-foreground">
  We sent a 6-digit code to{' '}
  <span className="font-medium text-foreground">+91 {phone}</span>
</p>
```

To:

```tsx
<p className="mb-4 text-sm text-muted-foreground">
  We sent a 6-digit code via {channel === 'whatsapp' ? 'WhatsApp' : 'SMS'} to{' '}
  <span className="font-medium text-foreground">+91 {phone}</span>
</p>
```

- [ ] **Step 3: Verify frontend compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npm run typecheck`
Expected: No type errors

- [ ] **Step 4: Verify frontend lint**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npm run lint`
Expected: No lint errors

- [ ] **Step 5: Start dev server and visually verify the login page**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npm run dev`

Open `http://localhost:3000/login` and verify:
- Phone entry shows two buttons: "Send via WhatsApp" and "Send via SMS"
- Both buttons are disabled until 10 digits entered
- Clicking either button transitions to OTP step
- OTP step says "via WhatsApp" or "via SMS"
- Resend uses the same channel

- [ ] **Step 6: Commit**

```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
git add src/stores/auth.ts src/app/login/page.tsx
git commit -m "feat: add WhatsApp/SMS channel selection to storefront login page"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run full backend tests**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make test`
Expected: All tests pass

- [ ] **Step 2: Run backend lint**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && golangci-lint run`
Expected: No lint errors

- [ ] **Step 3: Run frontend checks**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npm run check`
Expected: typecheck + lint + format:check all pass

- [ ] **Step 4: Test end-to-end locally with monolith**

Start backend: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make run`
Start frontend: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store && npm run dev`

1. Go to `http://localhost:3000/login`
2. Enter a phone number
3. Click "Send via WhatsApp" — backend console should show DEV WhatsApp OTP box
4. Click "Change phone number", re-enter, click "Send via SMS" — backend console should show DEV SMS OTP box
5. Enter the OTP from console, verify login completes

- [ ] **Step 5: Commit any final fixes if needed**
