# WhatsApp OTP Integration Design

**Date**: 2026-04-15
**Status**: Approved
**Scope**: Add WhatsApp as a user-selectable OTP delivery channel alongside existing MSG91 SMS

## Overview

Integrate the WhatsApp Business Cloud API as an additional OTP delivery channel for the homechrome-store login flow. Users see two buttons on the login page — "Send via WhatsApp" and "Send via SMS" — and choose how to receive their 6-digit code. The backend dispatches to the appropriate gateway. OTP generation, storage, and verification remain unchanged.

## Approach

New `WhatsAppGateway` alongside existing `SMSGateway` (Approach 1). Clean separation — SMS and WhatsApp code don't touch each other. Follows the existing gateway pattern (real client + DevClient, credential-based selection). No changes to the existing SMS flow.

## Backend

### New Package: `internal/gateway/whatsapp/`

Follows the exact pattern of `internal/gateway/sms/`.

**`types.go`**:
```go
type Config struct {
    AccessToken     string // Meta Graph API access token
    PhoneNumberID   string // Registered WhatsApp Business phone number ID
    OTPTemplateName string // Pre-approved authentication template name
}

type WhatsAppGateway interface {
    SendOTP(ctx context.Context, phone, code string) error
}
```

**`client.go`** — Real client calls the Meta Cloud API:
```
POST https://graph.facebook.com/v21.0/{PHONE_NUMBER_ID}/messages
Authorization: Bearer {ACCESS_TOKEN}
Content-Type: application/json

{
  "messaging_product": "whatsapp",
  "to": "{phone}",
  "type": "template",
  "template": {
    "name": "{otp_template_name}",
    "language": { "code": "en" },
    "components": [{
      "type": "body",
      "parameters": [{ "type": "text", "text": "{code}" }]
    }, {
      "type": "button",
      "sub_type": "url",
      "index": 0,
      "parameters": [{ "type": "text", "text": "{code}" }]
    }]
  }
}
```

HTTP client with 10-second timeout. Returns error on non-2xx status or when `messages[0].message_status` is absent from a 200 response. Error message includes the WhatsApp API error code and title when available.

**`dev_client.go`** — DevClient prints OTP to console:
```
╔══════════════════════════════════════════════╗
║  DEV WhatsApp OTP: +919876543210 → 123456    ║
╚══════════════════════════════════════════════╝
```

### Domain Interface

Add to `internal/domain/store_service.go`:
```go
type WhatsAppGateway interface {
    SendOTP(ctx context.Context, phone, code string) error
}
```

### Updated `SendOTPRequest`

In `internal/domain/entity.go`:
```go
type SendOTPRequest struct {
    Phone   string `json:"phone" validate:"required,e164"`
    Channel string `json:"channel" validate:"required,oneof=sms whatsapp"`
}
```

### Updated `CustomerAuthService` Interface

In `internal/domain/store_service.go`:
```go
SendOTP(ctx context.Context, phone, channel string) error
```

### Updated `CustomerAuthService` Implementation

In `internal/service/customer_auth_service.go`:

**Struct** — add `whatsappGateway domain.WhatsAppGateway` field.

**Constructor** — add `whatsappGateway` parameter.

**`SendOTP` method** — add `channel` parameter, dispatch based on value:
```go
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone, channel string) error {
    code, err := generateOTPCode()
    // ... store OTP (unchanged) ...

    switch channel {
    case "whatsapp":
        err = s.whatsappGateway.SendOTP(ctx, phone, code)
    default:
        err = s.smsGateway.SendOTP(ctx, phone, code)
    }
    // ... error handling (unchanged) ...
}
```

### Handler Change

In `internal/handler/store/auth_handler.go`, `SendOTP` passes `req.Channel`:
```go
if err := h.customerAuthService.SendOTP(ctx, req.Phone, req.Channel); err != nil {
```

### Wire DI

In `internal/wire/providers.go`, `ProvideCustomerAuthService`:
- If `WHATSAPP_ACCESS_TOKEN` or `WHATSAPP_PHONE_NUMBER_ID` is empty: `whatsapp.NewDevClient()`
- Otherwise: `whatsapp.NewClient(whatsapp.Config{...})`
- Pass whatsapp gateway to `NewCustomerAuthService()`

### Environment Variables (3 new)

```
WHATSAPP_ACCESS_TOKEN=        # Meta Graph API token (empty = DevClient)
WHATSAPP_PHONE_NUMBER_ID=     # WhatsApp Business phone number ID (empty = DevClient)
WHATSAPP_OTP_TEMPLATE_NAME=   # Pre-approved auth template name
```

Added to:
- `internal/config/` struct + loader
- `.env.example`
- `.env` / `.env.dev` with real values
- CDK `infra/stacks/api.go` for `store-auth` Lambda environment

## Frontend

### Auth Store (`homechrome-store/src/stores/auth.ts`)

```typescript
sendOTP: async (phone: string, channel: 'sms' | 'whatsapp') => {
    await api.post(ROUTES.AUTH.SEND_OTP, { phone, channel });
},
```

### Login Page (`homechrome-store/src/app/login/page.tsx`)

**Phone step** — replace single "Send OTP" button with two side-by-side buttons:
- "Send via WhatsApp" — calls `handleSendOTP('whatsapp')`
- "Send via SMS" — calls `handleSendOTP('sms')`
- Both disabled until phone is 10 digits

**New state**: `const [channel, setChannel] = useState<'sms' | 'whatsapp'>('sms')` — tracks which channel was used, so resend uses the same channel.

**OTP step** — message updated to say "via WhatsApp" or "via SMS". Resend uses stored channel. No other visual changes.

## What Does NOT Change

- OTP generation (6-digit, cryptographically secure)
- OTP storage (SHA256 hash in DynamoDB sessions table, 5-minute TTL)
- OTP verification (hash comparison, max 3 attempts)
- JWT token generation, refresh, and logout
- Cookie handling
- Guest cart merge
- VerifyOTP request/response format
- Any other page or component

## Infrastructure

No new Lambdas, DynamoDB tables, or S3 buckets. Only change is adding 3 env vars to the `store-auth` Lambda in CDK.

## Testing

- **Unit test**: `WhatsAppGateway.SendOTP` — mock HTTP, verify request body matches Meta Cloud API format
- **Unit test**: `CustomerAuthService.SendOTP` — verify dispatch to correct gateway based on `channel` parameter
- **Existing tests**: Update `SendOTP` calls to include channel parameter (default `"sms"` to keep behavior identical)
- DevClient requires no tests (console print only)
