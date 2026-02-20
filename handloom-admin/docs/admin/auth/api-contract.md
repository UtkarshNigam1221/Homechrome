# Auth Lambda API Documentation

Authentication and authorization service for the Handloom Admin system.

## Base Path
`/admin/auth`

## Endpoints

### Login

Authenticate a user and receive JWT tokens.

**Endpoint:** `POST /admin/auth/login`
**Authentication:** None (public)

**Request Body:**
```json
{
  "email": "admin@handloom.com",
  "password": "securePassword123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": "user_abc123",
    "email": "admin@handloom.com",
    "name": "Admin User",
    "role": "admin"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid credentials
- `400 Bad Request` - Missing required fields

---

### Register

Create a new user account.

**Endpoint:** `POST /admin/auth/register`
**Authentication:** None (public)

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "name": "John Doe",
  "role": "staff"
}
```

**Response (201 Created):**
```json
{
  "id": "user_xyz789",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "staff",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input or email already exists
- `422 Unprocessable Entity` - Validation failed

---

### Refresh Token

Get a new access token using a refresh token.

**Endpoint:** `POST /admin/auth/refresh`
**Authentication:** None (public)

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or expired refresh token

---

### Logout

Invalidate the current session tokens.

**Endpoint:** `POST /admin/auth/logout`
**Authentication:** Required

**Request Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "message": "Successfully logged out"
}
```

---

### Get Current User Profile

Retrieve the authenticated user's profile.

**Endpoint:** `GET /admin/auth/me`
**Authentication:** Required

**Request Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "id": "user_abc123",
  "email": "admin@handloom.com",
  "name": "Admin User",
  "role": "admin",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### Change Password

Change the authenticated user's password.

**Endpoint:** `POST /admin/auth/change-password`
**Authentication:** Required

**Request Body:**
```json
{
  "current_password": "oldPassword123",
  "new_password": "newSecurePassword456"
}
```

**Response (200 OK):**
```json
{
  "message": "Password changed successfully"
}
```

**Error Responses:**
- `400 Bad Request` - Current password incorrect
- `422 Unprocessable Entity` - New password doesn't meet requirements

---

## TODO

No pending TODO items identified.
