# Store Profile API Documentation

Customer profile and address management for the B2C storefront.

## Base Path
`/api/v1/store/me`

## Endpoints

### Get Profile

Retrieve the authenticated customer's profile including saved addresses and order statistics.

**Endpoint:** `GET /api/v1/store/me`
**Authentication:** Required (Customer JWT)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "cust-b7d1e4a2-3c5f-4890-9abc-def012345678",
    "email": "priya.sharma@gmail.com",
    "first_name": "Priya",
    "last_name": "Sharma",
    "phone": "+919876543210",
    "phone_verified": true,
    "status": "ACTIVE",
    "total_orders": 5,
    "total_spent": 2450000,
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
      },
      {
        "id": "addr-002",
        "first_name": "Priya",
        "last_name": "Sharma",
        "phone": "+919876543210",
        "address_line1": "15, Residency Road",
        "address_line2": "Flat 3B, Sunshine Apartments",
        "city": "Chennai",
        "state": "Tamil Nadu",
        "postal_code": "600001",
        "country": "IN",
        "is_default": false
      }
    ],
    "created_at": "2025-11-15T08:20:00Z",
    "updated_at": "2026-02-18T14:45:00Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated

---

### Update Profile

Update the authenticated customer's profile fields. All fields are optional; only provided fields are updated.

**Endpoint:** `PATCH /api/v1/store/me`
**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "first_name": "Priya",
  "last_name": "Sharma",
  "email": "priya.sharma@outlook.com"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `first_name` | string | No | 1-100 chars | Customer first name |
| `last_name` | string | No | 1-100 chars | Customer last name |
| `email` | string | No | Valid email format | Customer email address |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "cust-b7d1e4a2-3c5f-4890-9abc-def012345678",
    "email": "priya.sharma@outlook.com",
    "first_name": "Priya",
    "last_name": "Sharma",
    "phone": "+919876543210",
    "phone_verified": true,
    "status": "ACTIVE",
    "total_orders": 5,
    "total_spent": 2450000,
    "addresses": [],
    "created_at": "2025-11-15T08:20:00Z",
    "updated_at": "2026-02-20T10:15:00Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `400 Bad Request` - `VALIDATION_ERROR` - Invalid field values (e.g., malformed email)
- `409 Conflict` - `EMAIL_ALREADY_EXISTS` - Email is already in use by another customer

---

### Add Address

Add a new address to the authenticated customer's address book. If `is_default` is true, all other addresses are set to non-default.

**Endpoint:** `POST /api/v1/store/me/addresses`
**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "first_name": "Rajesh",
  "last_name": "Kumar",
  "phone": "+919123456789",
  "address_line1": "27, Nehru Nagar",
  "address_line2": "Behind Reliance Fresh",
  "city": "Hyderabad",
  "state": "Telangana",
  "postal_code": "500001",
  "country": "IN",
  "is_default": false
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `first_name` | string | Yes | 1-100 chars | Recipient first name |
| `last_name` | string | Yes | 1-100 chars | Recipient last name |
| `phone` | string | Yes | E.164 format | Recipient phone |
| `address_line1` | string | Yes | 1-200 chars | Street address line 1 |
| `address_line2` | string | No | 0-200 chars | Street address line 2 |
| `city` | string | Yes | 1-100 chars | City |
| `state` | string | Yes | 1-100 chars | State/province |
| `postal_code` | string | Yes | 6 digits | Indian postal code |
| `country` | string | Yes | 2-char ISO code | Country code (e.g., `IN`) |
| `is_default` | bool | No | - | Set as default address (default: false) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "addr-003",
    "first_name": "Rajesh",
    "last_name": "Kumar",
    "phone": "+919123456789",
    "address_line1": "27, Nehru Nagar",
    "address_line2": "Behind Reliance Fresh",
    "city": "Hyderabad",
    "state": "Telangana",
    "postal_code": "500001",
    "country": "IN",
    "is_default": false
  }
}
```

**Side Effects:**
- If `is_default` is `true`, clears the `is_default` flag on all existing addresses

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `400 Bad Request` - `VALIDATION_ERROR` - Missing or invalid address fields

---

### Update Address

Update an existing address in the customer's address book. All address fields can be updated. If `is_default` is set to true, clears default on other addresses.

**Endpoint:** `PUT /api/v1/store/me/addresses/{id}`
**Authentication:** Required (Customer JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Address ID |

**Request Body:**
```json
{
  "first_name": "Rajesh",
  "last_name": "Kumar",
  "phone": "+919123456789",
  "address_line1": "27, Nehru Nagar, 2nd Floor",
  "address_line2": "Behind Reliance Fresh",
  "city": "Hyderabad",
  "state": "Telangana",
  "postal_code": "500001",
  "country": "IN",
  "is_default": true
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `first_name` | string | Yes | 1-100 chars | Recipient first name |
| `last_name` | string | Yes | 1-100 chars | Recipient last name |
| `phone` | string | Yes | E.164 format | Recipient phone |
| `address_line1` | string | Yes | 1-200 chars | Street address line 1 |
| `address_line2` | string | No | 0-200 chars | Street address line 2 |
| `city` | string | Yes | 1-100 chars | City |
| `state` | string | Yes | 1-100 chars | State/province |
| `postal_code` | string | Yes | 6 digits | Indian postal code |
| `country` | string | Yes | 2-char ISO code | Country code |
| `is_default` | bool | No | - | Set as default address |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
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
      "is_default": false
    },
    {
      "id": "addr-003",
      "first_name": "Rajesh",
      "last_name": "Kumar",
      "phone": "+919123456789",
      "address_line1": "27, Nehru Nagar, 2nd Floor",
      "address_line2": "Behind Reliance Fresh",
      "city": "Hyderabad",
      "state": "Telangana",
      "postal_code": "500001",
      "country": "IN",
      "is_default": true
    }
  ]
}
```

**Side Effects:**
- If `is_default` is `true`, clears the `is_default` flag on all other addresses

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `400 Bad Request` - `VALIDATION_ERROR` - Missing or invalid address fields
- `404 Not Found` - `ADDRESS_NOT_FOUND` - Address ID not found in customer's address book

---

### Delete Address

Remove an address from the customer's address book.

**Endpoint:** `DELETE /api/v1/store/me/addresses/{id}`
**Authentication:** Required (Customer JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Address ID |

**Response (204 No Content):**
No response body.

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `404 Not Found` - `ADDRESS_NOT_FOUND` - Address ID not found in customer's address book

---
