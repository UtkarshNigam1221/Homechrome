# Store Auth - High-Level Design (HLD)

## 1. Overview

The Store Auth service provides phone OTP-based authentication for B2C storefront customers. Unlike the admin auth (email/password), store auth uses a passwordless flow: customers receive a 6-digit OTP via SMS, verify it, and receive JWT tokens stored in HttpOnly cookies. New customers are automatically created on first successful OTP verification. The SMS gateway is MSG91 in production; in dev mode, OTPs are printed to the console.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              STORE AUTHENTICATION SYSTEM                                      │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   Next.js Store   │
                                    │   (B2C Frontend)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (HttpOnly cookies)
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway /   │
                                    │   Chi Router      │
                                    │   (Rate Limiter)  │
                                    └─────────┬─────────┘
                                              │
                                              │ 30 req/min rate limit
                                              ▼
                                    ┌───────────────────┐
                                    │   Auth Handler    │
                                    │   (store/)        │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │ CustomerAuth      │
                                    │ Service           │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   SMS Gateway   │  │   JWT Service   │
              │   (OTP Store,   │  │   (MSG91 /      │  │   (HS256,       │
              │   Customer,     │  │   DevClient)    │  │   CUSTOMER_JWT  │
              │   Token Store)  │  │                 │  │   _SECRET)      │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Auth Handler (store/auth_handler.go)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      STORE AUTH HANDLER                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Public Routes:                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │  SendOTP    │  │  VerifyOTP  │  │  Refresh    │                 │
│  │  POST       │  │  POST       │  │  Token      │                 │
│  │  /otp/send  │  │  /otp/verify│  │  POST       │                 │
│  └──────┬──────┘  └──────┬──────┘  │  /refresh   │                 │
│         │                │         └──────┬──────┘                  │
│         │                │                │                          │
│  Protected Route:        │                │                          │
│  ┌─────────────┐         │                │                          │
│  │  Logout     │         │                │                          │
│  │  POST       │         │                │                          │
│  │  /logout    │         │                │                          │
│  └──────┬──────┘         │                │                          │
│         │                │                │                          │
│         └────────────────┼────────────────┘                          │
│                          │                                           │
│                          ▼                                           │
│                ┌───────────────────┐                                 │
│                │  CustomerAuth     │                                 │
│                │  Service          │                                 │
│                └───────────────────┘                                 │
│                                                                      │
│  Cookie Management:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  setStoreCookies()   — store_token (15m) + store_refresh (7d) │  │
│  │  clearStoreCookies() — MaxAge=-1 on both cookies              │  │
│  │  cookieSettings()    — env-aware Secure/SameSite/Domain       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 CustomerAuthService (service/customer_auth_service.go)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CUSTOMER AUTH SERVICE                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Methods:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ SendOTP(phone)                                                │  │
│  │   1. Generate cryptographic 6-digit code (crypto/rand)        │  │
│  │   2. Hash code with SHA256                                    │  │
│  │   3. Store OTP in DynamoDB (TTL-based expiry)                 │  │
│  │   4. Send code via SMS gateway                                │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │ VerifyOTP(phone, code)                                        │  │
│  │   1. Retrieve stored OTP by phone                             │  │
│  │   2. Check max attempts (3)                                   │  │
│  │   3. Increment attempts counter                               │  │
│  │   4. Verify SHA256(code) == stored hash                       │  │
│  │   5. Delete OTP on success                                    │  │
│  │   6. Find or create Customer                                  │  │
│  │   7. Generate JWT token pair                                  │  │
│  │   8. Store refresh token hash                                 │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │ RefreshToken(refreshToken)                                    │  │
│  │   1. Parse JWT, extract customer ID                           │  │
│  │   2. Validate hash exists in token store                      │  │
│  │   3. Check customer status is ACTIVE                          │  │
│  │   4. Generate new token pair                                  │  │
│  │   5. Store new hash, revoke old hash                          │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │ Logout(customerID, refreshToken)                              │  │
│  │   1. Hash refresh token with SHA256                           │  │
│  │   2. Revoke token hash in store                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Dependencies:                                                       │
│  ┌───────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐ │
│  │ OTPRepository │ │ CustomerRepo │ │ TokenStore  │ │ SMSGateway │ │
│  └───────────────┘ └──────────────┘ └─────────────┘ └────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.3 JWT Token Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                       CUSTOMER ACCESS TOKEN                          │
├─────────────────────────────────────────────────────────────────────┤
│ Claims:                                                              │
│   {                                                                  │
│     "sub":   "b7d1e4a2-3c5f-4890-9abc-def012345678",  // Customer  │
│     "phone": "+919876543210",                           // Phone    │
│     "email": "priya.sharma@gmail.com",                  // Email    │
│     "type":  "customer",                                // Type     │
│     "iat":   1708420200,                                // Issued   │
│     "exp":   1708421100,                                // +15 min  │
│     "iss":   "homechrome",                              // Issuer   │
│     "jti":   "unique-token-id"                          // Token ID │
│   }                                                                  │
├─────────────────────────────────────────────────────────────────────┤
│ Signing: HMAC-SHA256 with CUSTOMER_JWT_SECRET                        │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                      CUSTOMER REFRESH TOKEN                          │
├─────────────────────────────────────────────────────────────────────┤
│ Claims:                                                              │
│   {                                                                  │
│     "sub":  "b7d1e4a2-3c5f-4890-9abc-def012345678",   // Customer  │
│     "type": "customer_refresh",                         // Type     │
│     "iat":  1708420200,                                 // Issued   │
│     "exp":  1709025000,                                 // +7 days  │
│     "iss":  "homechrome",                               // Issuer   │
│     "jti":  "unique-refresh-id"                         // Token ID │
│   }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Tables

```
┌─────────────────────────────────────────────────────────────────────┐
│                  TABLE: handloom-orders                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  CUSTOMER RECORDS                                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: CUSTOMER#<customer_id>                                    │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - id, phone, phone_verified, email                         │  │
│  │   - first_name, last_name, status                            │  │
│  │   - total_orders, total_spent                                │  │
│  │   - addresses[] (embedded)                                   │  │
│  │   - tags[], notes                                            │  │
│  │   - created_at, updated_at                                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  CUSTOMER PHONE INDEX (uniqueness guard)                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: CUSTOMER_PHONE#<phone>                                    │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - customer_id                                              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  GSI1: Customer Email Index                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GSI1PK: CUSTOMER_EMAIL                                        │  │
│  │ GSI1SK: <email> (or NONE#<id> if no email)                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  GSI2: Customer All Index                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GSI2PK: CUSTOMER#ALL                                          │  │
│  │ GSI2SK: <created_at>                                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  CUSTOMER REFRESH TOKEN STORE                                        │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: CUSTOMER_TOKEN#<customer_id>                              │  │
│  │ SK: TOKEN#<token_hash>                                        │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - ttl (DynamoDB TTL for auto-expiry)                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                  TABLE: handloom-core                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  OTP RECORDS                                                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: OTP#<phone>                                               │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - phone, code_hash (SHA256)                                │  │
│  │   - attempts (max 3)                                         │  │
│  │   - created_at                                               │  │
│  │   - ttl (DynamoDB TTL, ~5 min expiry)                        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security

### 5.1 OTP Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                        OTP SECURITY                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Generation:                                                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - crypto/rand for cryptographic randomness                    │  │
│  │ - 6-digit zero-padded code (000000-999999)                   │  │
│  │ - SHA256 hash stored, plaintext discarded after send          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Brute Force Protection:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - Max 3 verification attempts per OTP                         │  │
│  │ - OTP auto-expires via DynamoDB TTL (~5 min)                  │  │
│  │ - OTP deleted after successful verification                   │  │
│  │ - Rate limit: 30 requests/min on auth endpoints               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 Cookie Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                       COOKIE SECURITY                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Environment-Aware Configuration:                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  Custom Domain (COOKIE_DOMAIN set):                           │  │
│  │    Secure=true, SameSite=Lax, Domain=homechrome.lldlab.com   │  │
│  │                                                               │  │
│  │  Lambda URL (no custom domain):                               │  │
│  │    Secure=true, SameSite=None (cross-origin required)        │  │
│  │                                                               │  │
│  │  Local Dev:                                                   │  │
│  │    Secure=false, SameSite=Lax (Vite proxy, same-origin)      │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Cookie Properties:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  store_token:                                                 │  │
│  │    HttpOnly=true, Path=/, MaxAge=900 (15 min)                │  │
│  │                                                               │  │
│  │  store_refresh:                                               │  │
│  │    HttpOnly=true, Path=/api/v1/store/auth, MaxAge=604800 (7d)│  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Token Rotation:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  old_refresh ──▶ validate ──▶ generate new pair               │  │
│  │                         │                                     │  │
│  │                         └──▶ revoke old hash (best-effort)    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                       ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  OTP Errors:                                                         │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ VALIDATION_ERROR    │ Invalid phone or code format          │    │
│  │ INVALID_TOKEN       │ OTP not found or expired              │    │
│  │ INVALID_CREDENTIALS │ Incorrect OTP code                    │    │
│  │ RATE_LIMIT_EXCEEDED │ Too many requests (30/min)            │    │
│  │ INTERNAL_ERROR      │ Failed to generate or send OTP        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Token Errors:                                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ INVALID_TOKEN       │ Refresh token invalid or revoked      │    │
│  │ TOKEN_EXPIRED       │ Refresh token has expired             │    │
│  │ USER_INACTIVE       │ Customer account is not ACTIVE        │    │
│  │ UNAUTHORIZED        │ Missing store_token cookie            │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Response Format:                                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ {                                                           │    │
│  │   "success": false,                                        │    │
│  │   "error": {                                               │    │
│  │     "code": "INVALID_CREDENTIALS",                         │    │
│  │     "message": "Invalid OTP code"                          │    │
│  │   }                                                        │    │
│  │ }                                                           │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────┐
│                    INTEGRATION POINTS                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  SMS Gateway (MSG91):                                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Production:                                                   │  │
│  │   - MSG91 Flow API v5 (https://control.msg91.com/api/v5)     │  │
│  │   - OTP template with configurable template_id                │  │
│  │   - Auth via authkey header                                   │  │
│  │                                                               │  │
│  │ Development:                                                  │  │
│  │   - DevClient prints OTP to console stdout                    │  │
│  │   - No external SMS calls in dev mode                         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Downstream Consumers:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - Cart Service (uses customer ID from JWT)                    │  │
│  │ - Checkout Service (uses customer ID from JWT)                │  │
│  │ - Order Service (uses customer ID from JWT)                   │  │
│  │ - Profile Service (uses customer ID from JWT)                 │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Middleware:                                                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - CustomerAuth middleware reads store_token cookie             │  │
│  │ - Validates JWT via CustomerAuthService.ValidateCustomerToken  │  │
│  │ - Sets customer_id in request context                         │  │
│  │ - Applied to /cart, /checkout, /orders, /me, /logout routes   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                       DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  External Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - AWS DynamoDB (handloom-orders + handloom-core tables)       │  │
│  │ - MSG91 SMS gateway (production OTP delivery)                 │  │
│  │ - AWS SSM Parameter Store (CUSTOMER_JWT_SECRET in prod)       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Internal Interfaces:                                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - domain.CustomerAuthService (service interface)              │  │
│  │ - domain.OTPRepository (OTP CRUD + TTL)                       │  │
│  │ - domain.CustomerRepository (customer find/create)            │  │
│  │ - domain.CustomerTokenStore (refresh token hash storage)      │  │
│  │ - customerSMSGateway (SMS send interface)                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Libraries:                                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - golang-jwt/jwt/v5 — JWT creation and validation             │  │
│  │ - crypto/rand, crypto/sha256 — OTP generation and hashing     │  │
│  │ - go-chi/chi — HTTP routing                                   │  │
│  │ - aws-sdk-go-v2 — DynamoDB client                             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
