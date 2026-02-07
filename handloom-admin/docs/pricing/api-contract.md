# Pricing Lambda API Documentation

Pricing rules management and price calculation service.

## Authentication
Admin routes require authentication. Public routes are available without auth.

---

# Admin Pricing Routes

## Base Path
`/admin/pricing`

### List Pricing Rules

**Endpoint:** `GET /admin/pricing/rules`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| type | string | - | Filter by rule type |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "rule_abc123",
      "name": "Summer Sale Discount",
      "type": "percentage",
      "value": 15.0,
      "conditions": {
        "category_ids": ["cat_abc123"],
        "min_quantity": 1,
        "start_date": "2024-06-01T00:00:00Z",
        "end_date": "2024-08-31T23:59:59Z"
      },
      "priority": 10,
      "status": "active",
      "created_at": "2024-05-15T10:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 25,
    "total_pages": 3
  }
}
```

---

### Create Pricing Rule

**Endpoint:** `POST /admin/pricing/rules`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Bulk Purchase Discount",
  "type": "percentage",
  "value": 10.0,
  "conditions": {
    "min_quantity": 5,
    "category_ids": ["cat_abc123", "cat_xyz789"]
  },
  "priority": 5,
  "status": "active"
}
```

**Response (201 Created):**
```json
{
  "id": "rule_xyz789",
  "name": "Bulk Purchase Discount",
  "type": "percentage",
  "value": 10.0,
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Pricing Rule by ID

**Endpoint:** `GET /admin/pricing/rules/{id}`
**Authentication:** Required

---

### Update Pricing Rule

**Endpoint:** `PUT /admin/pricing/rules/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Updated Discount Name",
  "value": 12.0,
  "status": "inactive"
}
```

---

### Delete Pricing Rule

**Endpoint:** `DELETE /admin/pricing/rules/{id}`
**Authentication:** Required

---

### List Price Quotes

**Endpoint:** `GET /admin/pricing/quotes`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| customer_id | string | - | Filter by customer |
| status | string | - | Filter by status |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "quote_abc123",
      "customer_id": "cust_xyz789",
      "items": [
        {
          "product_id": "prod_abc123",
          "quantity": 2,
          "unit_price": 15000.00,
          "discount": 3000.00,
          "final_price": 27000.00
        }
      ],
      "subtotal": 30000.00,
      "total_discount": 3000.00,
      "total": 27000.00,
      "valid_until": "2024-01-22T10:30:00Z",
      "status": "active",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {...}
}
```

---

### Get Price Quote by ID

**Endpoint:** `GET /admin/pricing/quotes/{id}`
**Authentication:** Required

---

### Delete Price Quote

**Endpoint:** `DELETE /admin/pricing/quotes/{id}`
**Authentication:** Required

---

# Public Pricing Routes

## Base Path
`/api/v1/pricing`

### Calculate Price

Calculate price for products (public/B2C endpoint).

**Endpoint:** `POST /api/v1/pricing/calculate`
**Authentication:** None (public)
**Rate Limit:** 100 requests/minute per IP

**Request Body:**
```json
{
  "items": [
    {
      "product_id": "prod_abc123",
      "quantity": 2
    },
    {
      "product_id": "prod_xyz789",
      "quantity": 1
    }
  ],
  "coupon_code": "SAVE10"
}
```

**Response (200 OK):**
```json
{
  "items": [
    {
      "product_id": "prod_abc123",
      "product_name": "Kanchipuram Silk Saree",
      "quantity": 2,
      "unit_price": 15000.00,
      "subtotal": 30000.00,
      "discounts": [
        {
          "rule_name": "Bulk Purchase Discount",
          "type": "percentage",
          "value": 10.0,
          "amount": 3000.00
        }
      ],
      "discount_amount": 3000.00,
      "final_price": 27000.00
    },
    {
      "product_id": "prod_xyz789",
      "product_name": "Cotton Saree",
      "quantity": 1,
      "unit_price": 3500.00,
      "subtotal": 3500.00,
      "discounts": [],
      "discount_amount": 0.00,
      "final_price": 3500.00
    }
  ],
  "subtotal": 33500.00,
  "coupon_discount": 3050.00,
  "total_discount": 6050.00,
  "tax": 1647.00,
  "total": 29097.00,
  "applied_coupon": {
    "code": "SAVE10",
    "type": "percentage",
    "value": 10.0,
    "discount_amount": 3050.00
  }
}
```

---

### Create Price Quote

Create a price quote for a customer.

**Endpoint:** `POST /api/v1/pricing/quote`
**Authentication:** None (public)

**Request Body:**
```json
{
  "customer_email": "customer@example.com",
  "customer_name": "Customer Name",
  "items": [
    {
      "product_id": "prod_abc123",
      "quantity": 5
    }
  ],
  "notes": "Request for bulk purchase"
}
```

**Response (201 Created):**
```json
{
  "id": "quote_xyz789",
  "reference_number": "QT-2024-001",
  "items": [...],
  "subtotal": 75000.00,
  "discount": 7500.00,
  "total": 67500.00,
  "valid_until": "2024-01-22T10:30:00Z",
  "status": "pending",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

## Pricing Rule Types

| Type | Description |
|------|-------------|
| `percentage` | Percentage discount off the price |
| `fixed` | Fixed amount discount |
| `tiered` | Different discounts based on quantity |
| `bundle` | Discount when products are purchased together |

## Rule Priority

Rules are applied in order of priority (lower number = higher priority). When multiple rules apply:
1. Only one rule of each type is applied
2. Higher priority rules take precedence
3. Bundle discounts are applied after individual discounts

---

## TODO

No pending TODO items identified.
