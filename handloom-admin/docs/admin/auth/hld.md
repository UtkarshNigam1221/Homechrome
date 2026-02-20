# Auth Lambda - High-Level Design (HLD)

## 1. Overview

The Auth Lambda service handles all authentication and authorization functionality for the Handloom Admin system. It provides secure user authentication using JWT tokens, session management, and role-based access control.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    AUTHENTICATION SYSTEM                                     │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   React Frontend  │
                                    │   (Admin Portal)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway     │
                                    │   (REST API)      │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │  Auth Lambda    │  │  Auth Lambda    │  │  Auth Lambda    │
              │  (Login)        │  │  (Register)     │  │  (Refresh)      │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                         ┌──────────────────┼──────────────────┐
                         │                  │                  │
                         ▼                  ▼                  ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   Secrets       │  │   CloudWatch    │
              │   (Users,       │  │   Manager       │  │   (Logs &       │
              │   Sessions)     │  │   (JWT Keys)    │  │   Metrics)      │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Auth Handler
```
┌─────────────────────────────────────────────────────────────────────┐
│                         AUTH HANDLER                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   Login     │  │  Register   │  │   Refresh   │  │   Logout   │ │
│  │   Handler   │  │  Handler    │  │   Handler   │  │   Handler  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │               │         │
│         └────────────────┼────────────────┼───────────────┘         │
│                          │                │                          │
│                          ▼                ▼                          │
│                   ┌─────────────────────────────┐                   │
│                   │       Auth Service          │                   │
│                   │  - ValidateCredentials()    │                   │
│                   │  - CreateUser()             │                   │
│                   │  - GenerateTokens()         │                   │
│                   │  - RefreshTokens()          │                   │
│                   │  - InvalidateSession()      │                   │
│                   └──────────────┬──────────────┘                   │
│                                  │                                   │
│                   ┌──────────────┴──────────────┐                   │
│                   │                             │                    │
│                   ▼                             ▼                    │
│          ┌───────────────┐            ┌───────────────┐             │
│          │ User          │            │ JWT           │             │
│          │ Repository    │            │ Service       │             │
│          └───────────────┘            └───────────────┘             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 JWT Token Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ACCESS TOKEN                                 │
├─────────────────────────────────────────────────────────────────────┤
│ Header:                                                              │
│   {                                                                  │
│     "alg": "HS256",                                                 │
│     "typ": "JWT"                                                    │
│   }                                                                  │
├─────────────────────────────────────────────────────────────────────┤
│ Payload:                                                             │
│   {                                                                  │
│     "sub": "user_abc123",          // User ID                       │
│     "email": "user@example.com",   // User email                    │
│     "role": "admin",               // User role                     │
│     "iat": 1704067200,             // Issued at                     │
│     "exp": 1704070800,             // Expires (1 hour)              │
│     "type": "access"               // Token type                    │
│   }                                                                  │
├─────────────────────────────────────────────────────────────────────┤
│ Signature:                                                           │
│   HMACSHA256(base64UrlEncode(header) + "." +                        │
│              base64UrlEncode(payload), secret)                       │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                        REFRESH TOKEN                                 │
├─────────────────────────────────────────────────────────────────────┤
│ Payload:                                                             │
│   {                                                                  │
│     "sub": "user_abc123",          // User ID                       │
│     "jti": "token_unique_id",      // Token ID (for revocation)     │
│     "iat": 1704067200,             // Issued at                     │
│     "exp": 1704672000,             // Expires (7 days)              │
│     "type": "refresh"              // Token type                    │
│   }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-admin                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  USER RECORDS                                                        │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ PK: USER#<user_id>                                          │    │
│  │ SK: PROFILE                                                 │    │
│  │                                                             │    │
│  │ Attributes:                                                 │    │
│  │   - email (GSI1-PK)                                        │    │
│  │   - password_hash                                          │    │
│  │   - name                                                   │    │
│  │   - role                                                   │    │
│  │   - status                                                 │    │
│  │   - created_at                                             │    │
│  │   - updated_at                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  SESSION RECORDS (Refresh Tokens)                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ PK: USER#<user_id>                                          │    │
│  │ SK: SESSION#<token_id>                                      │    │
│  │                                                             │    │
│  │ Attributes:                                                 │    │
│  │   - refresh_token_hash                                     │    │
│  │   - device_info                                            │    │
│  │   - ip_address                                             │    │
│  │   - created_at                                             │    │
│  │   - expires_at (TTL)                                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  GSI1: Email Index                                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ GSI1-PK: email                                              │    │
│  │ GSI1-SK: USER#<user_id>                                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security Design

### 5.1 Password Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                      PASSWORD SECURITY                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Password Requirements:                                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Minimum 8 characters                                        │  │
│  │ • At least one uppercase letter                               │  │
│  │ • At least one lowercase letter                               │  │
│  │ • At least one number                                         │  │
│  │ • At least one special character (!@#$%^&*)                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Hashing Algorithm:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  bcrypt with cost factor = 12                                │  │
│  │                                                               │  │
│  │  password + salt ──▶ bcrypt ──▶ hash (60 chars)              │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Rate Limiting:                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • 5 failed login attempts → 15 minute lockout                │  │
│  │ • 10 failed attempts → 1 hour lockout                        │  │
│  │ • Exponential backoff for repeated failures                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 Token Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                        TOKEN SECURITY                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Access Token:                                                       │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Short-lived (1 hour)                                        │  │
│  │ • Stored in memory only (not localStorage)                    │  │
│  │ • Contains minimal claims                                     │  │
│  │ • Signed with HS256 (HMAC-SHA256)                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Refresh Token:                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Long-lived (7 days)                                         │  │
│  │ • Stored in httpOnly cookie                                   │  │
│  │ • Rotated on each use                                        │  │
│  │ • Hash stored in database for revocation                      │  │
│  │ • Single-use (invalidated after refresh)                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Token Rotation:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  Old Refresh Token ──▶ Verify ──▶ Generate New Pair          │  │
│  │                              │                                │  │
│  │                              └──▶ Invalidate Old Token       │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Role-Based Access Control (RBAC)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      RBAC DESIGN                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Roles Hierarchy:                                                    │
│                                                                      │
│                    ┌─────────────┐                                   │
│                    │   ADMIN     │                                   │
│                    │  (Full)     │                                   │
│                    └──────┬──────┘                                   │
│                           │                                          │
│              ┌────────────┴────────────┐                            │
│              │                         │                             │
│       ┌──────▼──────┐          ┌──────▼──────┐                      │
│       │   MANAGER   │          │  STAFF      │                      │
│       │ (Write+Read)│          │ (Read Only) │                      │
│       └─────────────┘          └─────────────┘                      │
│                                                                      │
│  Permissions Matrix:                                                 │
│  ┌──────────────┬─────────┬─────────┬─────────┐                     │
│  │ Resource     │ Admin   │ Manager │ Staff   │                     │
│  ├──────────────┼─────────┼─────────┼─────────┤                     │
│  │ Users        │ CRUD    │ R       │ -       │                     │
│  │ Products     │ CRUD    │ CRUD    │ R       │                     │
│  │ Orders       │ CRUD    │ CRUD    │ R       │                     │
│  │ Inventory    │ CRUD    │ CRUD    │ R       │                     │
│  │ Reports      │ CRUD    │ R       │ R       │                     │
│  │ Audit Logs   │ R       │ -       │ -       │                     │
│  │ Settings     │ CRUD    │ R       │ -       │                     │
│  └──────────────┴─────────┴─────────┴─────────┘                     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Authentication Errors:                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ AUTH001 │ Invalid credentials                               │    │
│  │ AUTH002 │ Account locked                                    │    │
│  │ AUTH003 │ Account not verified                              │    │
│  │ AUTH004 │ Invalid token                                     │    │
│  │ AUTH005 │ Token expired                                     │    │
│  │ AUTH006 │ Token revoked                                     │    │
│  │ AUTH007 │ Insufficient permissions                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Validation Errors:                                                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ VAL001  │ Invalid email format                              │    │
│  │ VAL002  │ Password too weak                                 │    │
│  │ VAL003  │ Email already registered                          │    │
│  │ VAL004  │ Required field missing                            │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Monitoring & Observability

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MONITORING DESIGN                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Metrics:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Login success/failure rate                                  │  │
│  │ • Token refresh rate                                          │  │
│  │ • Average authentication latency                              │  │
│  │ • Active sessions count                                       │  │
│  │ • Failed login attempts (by IP, by user)                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Alerts:                                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • High failed login rate (potential attack)                   │  │
│  │ • Unusual login location                                      │  │
│  │ • Multiple concurrent sessions                                │  │
│  │ • Token generation spikes                                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Audit Logging:                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • All login attempts (success/failure)                        │  │
│  │ • Password changes                                            │  │
│  │ • Permission changes                                          │  │
│  │ • Session invalidations                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. Scalability Considerations

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SCALABILITY DESIGN                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Lambda Configuration:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Memory: 256 MB                                              │  │
│  │ • Timeout: 30 seconds                                         │  │
│  │ • Concurrent executions: 100 (reserved)                       │  │
│  │ • Provisioned concurrency: 5 (for cold start mitigation)      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  DynamoDB Configuration:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • On-demand capacity mode                                     │  │
│  │ • GSI for email lookups                                       │  │
│  │ • DAX cluster for read-heavy operations (future)              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Caching Strategy:                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • JWT secret cached in Lambda memory                          │  │
│  │ • User permissions cached with 5-min TTL                      │  │
│  │ • Rate limit counters in DynamoDB                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                      DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  External Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • AWS DynamoDB - User & session storage                       │  │
│  │ • AWS Secrets Manager - JWT signing keys                      │  │
│  │ • AWS CloudWatch - Logging & monitoring                       │  │
│  │ • AWS SES - Email notifications (password reset)              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Internal Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Notification Lambda - Welcome emails                        │  │
│  │ • Audit Lambda - Security event logging                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Libraries:                                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • golang-jwt/jwt/v5 - JWT handling                            │  │
│  │ • golang.org/x/crypto/bcrypt - Password hashing               │  │
│  │ • aws-sdk-go-v2 - AWS service clients                         │  │
│  │ • go-chi/chi - HTTP routing                                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
