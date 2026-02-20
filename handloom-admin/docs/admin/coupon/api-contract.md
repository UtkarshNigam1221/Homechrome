# Coupon Lambda API Documentation

Discount coupon management service.

## Base Path
`/admin/coupons`

## Authentication
All endpoints require authentication.

---

### List Coupons

Get paginated list of all coupons.

**Endpoint:** `GET /admin/coupons`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by status (active, inactive, expired) |
| type | string | - | Filter by type |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "coupon_abc123",
      "code": "SUMMER20",
      "name": "Summer Sale 20%",
      "description": "Get 20% off on all products",
      "type": "percentage",
      "value": 20.0,
      "min_order_value": 1000.00,
      "max_discount": 5000.00,
      "usage_limit": 100,
      "usage_count": 45,
      "usage_limit_per_user": 1,
      "valid_from": "2024-06-01T00:00:00Z",
      "valid_until": "2024-08-31T23:59:59Z",
      "applicable_categories": ["cat_abc123"],
      "applicable_products": [],
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

### Create Coupon

**Endpoint:** `POST /admin/coupons`
**Authentication:** Required

**Request Body:**
```json
{
  "code": "NEWUSER15",
  "name": "New User Discount",
  "description": "15% off for new users",
  "type": "percentage",
  "value": 15.0,
  "min_order_value": 500.00,
  "max_discount": 2000.00,
  "usage_limit": 500,
  "usage_limit_per_user": 1,
  "valid_from": "2024-01-01T00:00:00Z",
  "valid_until": "2024-12-31T23:59:59Z",
  "first_order_only": true
}
```

**Response (201 Created):**
```json
{
  "id": "coupon_xyz789",
  "code": "NEWUSER15",
  "name": "New User Discount",
  "type": "percentage",
  "value": 15.0,
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Coupon by ID

**Endpoint:** `GET /admin/coupons/{id}`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "coupon_abc123",
  "code": "SUMMER20",
  "name": "Summer Sale 20%",
  "description": "Get 20% off on all products",
  "type": "percentage",
  "value": 20.0,
  "min_order_value": 1000.00,
  "max_discount": 5000.00,
  "usage_limit": 100,
  "usage_count": 45,
  "usage_limit_per_user": 1,
  "valid_from": "2024-06-01T00:00:00Z",
  "valid_until": "2024-08-31T23:59:59Z",
  "applicable_categories": ["cat_abc123"],
  "applicable_products": [],
  "excluded_products": [],
  "status": "active",
  "usage_history": [
    {
      "order_id": "ord_abc123",
      "customer_id": "cust_xyz789",
      "discount_amount": 2000.00,
      "used_at": "2024-06-15T14:30:00Z"
    }
  ],
  "created_at": "2024-05-15T10:00:00Z",
  "updated_at": "2024-06-15T14:30:00Z"
}
```

---

### Update Coupon

**Endpoint:** `PUT /admin/coupons/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Summer Sale 25%",
  "value": 25.0,
  "max_discount": 7500.00,
  "usage_limit": 150
}
```

**Response (200 OK):**
```json
{
  "id": "coupon_abc123",
  "code": "SUMMER20",
  "name": "Summer Sale 25%",
  "value": 25.0,
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Delete Coupon

**Endpoint:** `DELETE /admin/coupons/{id}`
**Authentication:** Required

**Response (204 No Content)**

---

### Validate Coupon

Check if a coupon is valid for a given order.

**Endpoint:** `POST /admin/coupons/validate`
**Authentication:** Required

**Request Body:**
```json
{
  "code": "SUMMER20",
  "customer_id": "cust_xyz789",
  "order_value": 5000.00,
  "items": [
    {
      "product_id": "prod_abc123",
      "category_id": "cat_abc123",
      "quantity": 1,
      "price": 5000.00
    }
  ]
}
```

**Response (200 OK) - Valid:**
```json
{
  "valid": true,
  "coupon": {
    "code": "SUMMER20",
    "type": "percentage",
    "value": 20.0
  },
  "discount_amount": 1000.00,
  "message": "Coupon applied successfully"
}
```

**Response (200 OK) - Invalid:**
```json
{
  "valid": false,
  "reason": "minimum_order_not_met",
  "message": "Minimum order value of ₹1000 required",
  "min_order_value": 1000.00
}
```

---

### Apply Coupon

Apply a coupon to an order (internal use).

**Endpoint:** `POST /admin/coupons/apply`
**Authentication:** Required

**Request Body:**
```json
{
  "code": "SUMMER20",
  "order_id": "ord_abc123",
  "customer_id": "cust_xyz789"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "coupon_id": "coupon_abc123",
  "discount_amount": 1000.00,
  "message": "Coupon applied to order"
}
```

---

### Get Coupon by Code

**Endpoint:** `GET /admin/coupons/code/{code}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| code | string | Coupon code |

**Response (200 OK):**
```json
{
  "id": "coupon_abc123",
  "code": "SUMMER20",
  "name": "Summer Sale 20%",
  "type": "percentage",
  "value": 20.0,
  "status": "active"
}
```

---

## Coupon Types

| Type | Description |
|------|-------------|
| `percentage` | Percentage discount off order total |
| `fixed` | Fixed amount discount |
| `free_shipping` | Free shipping on order |
| `buy_x_get_y` | Buy X items get Y free |

## Validation Reasons

| Reason | Description |
|--------|-------------|
| `expired` | Coupon validity period has ended |
| `not_yet_valid` | Coupon validity period hasn't started |
| `usage_limit_reached` | Total usage limit exceeded |
| `user_limit_reached` | Per-user usage limit exceeded |
| `minimum_order_not_met` | Order value below minimum |
| `not_applicable` | Products not eligible for coupon |
| `first_order_only` | Only valid for first orders |
| `invalid_code` | Coupon code doesn't exist |
| `inactive` | Coupon is deactivated |

---

## TODO

No pending TODO items identified.
