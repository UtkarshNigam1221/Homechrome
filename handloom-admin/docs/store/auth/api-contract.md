# Store Auth API Documentation

Phone OTP-based authentication for B2C storefront customers.

## Base Path
`/api/v1/store/auth`

## Endpoints

### Send OTP

Send a one-time password to a customer's phone number via SMS.

**Endpoint:** `POST /api/v1/store/auth/otp/send`
**Authentication:** None (public)

**Request Body:**
```json
{
  "phone": "+919876543210"
}
```

| Field   | Type   | Required | Validation          | Description                    |
|---------|--------|----------|---------------------|--------------------------------|
| `phone` | string | Yes      | E.164 format        | Customer phone number          |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "OTP sent successfully"
  }
}
```

**Error Responses:**
- `400 Bad Request` - `VALIDATION_ERROR` - Invalid phone number format
- `429 Too Many Requests` - `RATE_LIMIT_EXCEEDED` - Max 30 requests/minute exceeded

---

### Verify OTP

Verify the OTP code sent to a phone number. On success, creates a new customer (if first login) or retrieves the existing customer, then issues JWT tokens in HttpOnly cookies.

**Endpoint:** `POST /api/v1/store/auth/otp/verify`
**Authentication:** None (public)

**Request Body:**
```json
{
  "phone": "+919876543210",
  "code": "483921"
}
```

| Field   | Type   | Required | Validation              | Description            |
|---------|--------|----------|-------------------------|------------------------|
| `phone` | string | Yes      | E.164 format            | Customer phone number  |
| `code`  | string | Yes      | Exactly 6 characters    | OTP code from SMS      |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "customer": {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "phone": "+919876543210",
      "phone_verified": true,
      "first_name": "",
      "last_name": "",
      "email": "",
      "status": "ACTIVE",
      "total_orders": 0,
      "total_spent": 0,
      "addresses": [],
      "created_at": "2026-02-20T10:30:00Z",
      "updated_at": "2026-02-20T10:30:00Z"
    },
    "is_new_customer": true
  }
}
```

Returning customer example:
```json
{
  "success": true,
  "data": {
    "customer": {
      "id": "b7d1e4a2-3c5f-4890-9abc-def012345678",
      "phone": "+919876543210",
      "phone_verified": true,
      "first_name": "Priya",
      "last_name": "Sharma",
      "email": "priya.sharma@gmail.com",
      "status": "ACTIVE",
      "total_orders": 5,
      "total_spent": 24500.00,
      "addresses": [
        {
          "id": "addr-001",
          "first_name": "Priya",
          "last_name": "Sharma",
          "phone": "+919876543210",
          "address_line1": "42, MG Road",
          "address_line2": "Near City Mall",
          "city": "Bengaluru",
          "state": "Karnataka",
          "postal_code": "560001",
          "country": "IN",
          "is_default": true
        }
      ],
      "created_at": "2025-11-15T08:20:00Z",
      "updated_at": "2026-02-18T14:45:00Z"
    },
    "is_new_customer": false
  }
}
```

**Side Effects:**
- Sets `store_token` HttpOnly cookie (15 min TTL, path=`/`)
- Sets `store_refresh` HttpOnly cookie (7 day TTL, path=`/api/v1/store/auth`)
- Creates new Customer record on first login (status=ACTIVE, phone_verified=true)

**Error Responses:**
- `400 Bad Request` - `INVALID_CREDENTIALS` - Invalid OTP code
- `400 Bad Request` - `INVALID_TOKEN` - OTP not found or expired, or too many attempts (max 3)
- `400 Bad Request` - `VALIDATION_ERROR` - Invalid phone or code format
- `429 Too Many Requests` - `RATE_LIMIT_EXCEEDED` - Rate limit exceeded

---

### Refresh Token

Refresh an expired access token using the refresh token cookie. Issues a new token pair and rotates the refresh token.

**Endpoint:** `POST /api/v1/store/auth/refresh`
**Authentication:** None (uses `store_refresh` cookie)

**Request Body:** None

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "customer": {
      "id": "b7d1e4a2-3c5f-4890-9abc-def012345678",
      "phone": "+919876543210",
      "phone_verified": true,
      "first_name": "Priya",
      "last_name": "Sharma",
      "email": "priya.sharma@gmail.com",
      "status": "ACTIVE",
      "total_orders": 5,
      "total_spent": 24500.00,
      "addresses": [],
      "created_at": "2025-11-15T08:20:00Z",
      "updated_at": "2026-02-18T14:45:00Z"
    },
    "message": "Token refreshed"
  }
}
```

**Side Effects:**
- Sets new `store_token` HttpOnly cookie (15 min TTL)
- Sets new `store_refresh` HttpOnly cookie (7 day TTL)
- Revokes the old refresh token (single-use rotation)

**Error Responses:**
- `401 Unauthorized` - `TOKEN_EXPIRED` - Refresh token missing, expired, or revoked
- `401 Unauthorized` - `USER_INACTIVE` - Customer account is not active

---

### Logout

Log out the current customer session. Clears auth cookies and revokes the refresh token.

**Endpoint:** `POST /api/v1/store/auth/logout`
**Authentication:** Required (Customer JWT via `store_token` cookie)

**Request Body:** None

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

**Side Effects:**
- Clears `store_token` cookie (MaxAge=-1)
- Clears `store_refresh` cookie (MaxAge=-1)
- Revokes refresh token hash in DynamoDB

**Error Responses:**
- `401 Unauthorized` - Customer not authenticated

---
