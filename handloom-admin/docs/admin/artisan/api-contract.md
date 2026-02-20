# Artisan Lambda API Documentation

Artisan (weaver/craftsperson) management service.

## Base Path
`/admin/artisans`

## Authentication
All endpoints require authentication.

---

### List Artisans

Get paginated list of all artisans.

**Endpoint:** `GET /admin/artisans`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by status (active, inactive, pending) |
| skill | string | - | Filter by skill type |
| location | string | - | Filter by location/region |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "artisan_abc123",
      "name": "Rajesh Kumar",
      "email": "rajesh@example.com",
      "phone": "+91-9876543210",
      "location": {
        "city": "Kanchipuram",
        "state": "Tamil Nadu",
        "country": "India"
      },
      "skills": ["silk_weaving", "zari_work"],
      "experience_years": 25,
      "status": "active",
      "total_products": 45,
      "total_earnings": 450000.00,
      "rating": 4.8,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 50,
    "total_pages": 5
  }
}
```

---

### Create Artisan

**Endpoint:** `POST /admin/artisans`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Lakshmi Devi",
  "email": "lakshmi@example.com",
  "phone": "+91-9876543211",
  "location": {
    "address": "123 Weaver Colony",
    "city": "Varanasi",
    "state": "Uttar Pradesh",
    "postal_code": "221001",
    "country": "India"
  },
  "skills": ["silk_weaving", "brocade"],
  "experience_years": 15,
  "bank_details": {
    "account_name": "Lakshmi Devi",
    "account_number": "XXXX1234",
    "bank_name": "State Bank of India",
    "ifsc_code": "SBIN0001234"
  },
  "documents": {
    "id_type": "aadhaar",
    "id_number": "XXXX-XXXX-1234"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "artisan_xyz789",
  "name": "Lakshmi Devi",
  "status": "pending",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Artisan by ID

**Endpoint:** `GET /admin/artisans/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "artisan_abc123",
  "name": "Rajesh Kumar",
  "email": "rajesh@example.com",
  "phone": "+91-9876543210",
  "location": {
    "address": "45 Silk Weaver Street",
    "city": "Kanchipuram",
    "state": "Tamil Nadu",
    "postal_code": "631501",
    "country": "India"
  },
  "skills": ["silk_weaving", "zari_work"],
  "experience_years": 25,
  "bio": "Master weaver with 25 years of experience in traditional Kanchipuram silk sarees",
  "certifications": ["National Award for Handloom 2020"],
  "status": "active",
  "bank_details": {
    "account_name": "Rajesh Kumar",
    "bank_name": "State Bank of India",
    "ifsc_code": "SBIN0001234"
  },
  "stats": {
    "total_products": 45,
    "active_products": 42,
    "total_orders": 320,
    "total_earnings": 450000.00,
    "pending_payout": 25000.00
  },
  "rating": 4.8,
  "review_count": 85,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### Update Artisan

**Endpoint:** `PUT /admin/artisans/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Rajesh Kumar",
  "phone": "+91-9876543210",
  "skills": ["silk_weaving", "zari_work", "brocade"],
  "experience_years": 26
}
```

---

### Delete Artisan

**Endpoint:** `DELETE /admin/artisans/{id}`
**Authentication:** Required

**Response (204 No Content)**

**Error Responses:**
- `400 Bad Request` - Artisan has active products

---

### Update Artisan Status

**Endpoint:** `PUT /admin/artisans/{id}/status`
**Authentication:** Required

**Request Body:**
```json
{
  "status": "active",
  "reason": "Verification completed"
}
```

**Response (200 OK):**
```json
{
  "id": "artisan_abc123",
  "status": "active",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Get Artisan Products

Get all products created by an artisan.

**Endpoint:** `GET /admin/artisans/{id}/products`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by product status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "prod_abc123",
      "name": "Kanchipuram Silk Saree - Peacock Design",
      "sku": "SAR-SLK-001",
      "base_price": 15000.00,
      "status": "active",
      "total_sold": 30,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 45,
    "total_pages": 5
  }
}
```

---

### Get Artisan Payouts

Get payout history for an artisan.

**Endpoint:** `GET /admin/artisans/{id}/payouts`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by status (pending, completed, failed) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "payout_abc123",
      "amount": 25000.00,
      "status": "completed",
      "payment_method": "bank_transfer",
      "reference": "TXN123456789",
      "period": {
        "start": "2024-01-01T00:00:00Z",
        "end": "2024-01-15T23:59:59Z"
      },
      "orders_count": 15,
      "created_at": "2024-01-16T10:00:00Z",
      "completed_at": "2024-01-16T14:00:00Z"
    }
  ],
  "summary": {
    "total_earned": 450000.00,
    "total_paid": 425000.00,
    "pending_payout": 25000.00
  },
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 24,
    "total_pages": 3
  }
}
```

---

### Create Artisan Payout

Initiate a payout for an artisan.

**Endpoint:** `POST /admin/artisans/{id}/payouts`
**Authentication:** Required

**Request Body:**
```json
{
  "amount": 25000.00,
  "payment_method": "bank_transfer",
  "notes": "Bi-weekly payout for orders"
}
```

**Response (201 Created):**
```json
{
  "id": "payout_xyz789",
  "artisan_id": "artisan_abc123",
  "amount": 25000.00,
  "status": "pending",
  "payment_method": "bank_transfer",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Search Artisans

**Endpoint:** `GET /admin/artisans/search`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Search query (name, email, phone, location) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "artisan_abc123",
      "name": "Rajesh Kumar",
      "location": "Kanchipuram, Tamil Nadu",
      "skills": ["silk_weaving"],
      "status": "active"
    }
  ]
}
```

---

## Artisan Status

| Status | Description |
|--------|-------------|
| `pending` | Awaiting verification |
| `active` | Verified and active |
| `inactive` | Temporarily deactivated |
| `suspended` | Account suspended |

## Skills

| Skill | Description |
|-------|-------------|
| `silk_weaving` | Traditional silk weaving |
| `cotton_weaving` | Cotton handloom weaving |
| `zari_work` | Gold/silver thread work |
| `brocade` | Brocade weaving |
| `ikat` | Ikat dyeing and weaving |
| `block_printing` | Block printing on fabric |

---

## TODO

No pending TODO items identified.
