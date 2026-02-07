# Order Lambda API Documentation

Order management and customer management service.

## Authentication
All endpoints require authentication.

---

# Orders

## Base Path
`/admin/orders`

### List Orders

**Endpoint:** `GET /admin/orders`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| status | string | - | Filter by status (pending, confirmed, processing, shipped, delivered, cancelled) |
| customer_id | string | - | Filter by customer |
| start_date | string | - | Filter orders from date (ISO 8601) |
| end_date | string | - | Filter orders until date (ISO 8601) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "ord_abc123",
      "order_number": "ORD-2024-001",
      "customer_id": "cust_xyz789",
      "customer": {
        "name": "John Doe",
        "email": "john@example.com"
      },
      "items": [
        {
          "product_id": "prod_abc123",
          "product_name": "Kanchipuram Silk Saree",
          "sku": "SAR-SLK-001",
          "quantity": 1,
          "unit_price": 15000.00,
          "total_price": 15000.00
        }
      ],
      "subtotal": 15000.00,
      "discount": 1500.00,
      "tax": 810.00,
      "shipping": 0.00,
      "total": 14310.00,
      "status": "confirmed",
      "payment_status": "paid",
      "shipping_address": {
        "name": "John Doe",
        "line1": "123 Main St",
        "city": "Chennai",
        "state": "Tamil Nadu",
        "postal_code": "600001",
        "country": "India"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 150,
    "total_pages": 15
  }
}
```

---

### Create Order

**Endpoint:** `POST /admin/orders`
**Authentication:** Required

**Request Body:**
```json
{
  "customer_id": "cust_xyz789",
  "items": [
    {
      "product_id": "prod_abc123",
      "quantity": 2
    }
  ],
  "shipping_address": {
    "name": "John Doe",
    "line1": "123 Main St",
    "line2": "Apt 4B",
    "city": "Chennai",
    "state": "Tamil Nadu",
    "postal_code": "600001",
    "country": "India",
    "phone": "+91-9876543210"
  },
  "coupon_code": "SAVE10",
  "notes": "Gift wrap requested"
}
```

**Response (201 Created):**
```json
{
  "id": "ord_xyz789",
  "order_number": "ORD-2024-002",
  "customer_id": "cust_xyz789",
  "subtotal": 30000.00,
  "discount": 3000.00,
  "tax": 1620.00,
  "total": 28620.00,
  "status": "pending",
  "payment_status": "pending",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Order by ID

**Endpoint:** `GET /admin/orders/{id}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Order ID |

**Response (200 OK):**
```json
{
  "id": "ord_abc123",
  "order_number": "ORD-2024-001",
  "customer_id": "cust_xyz789",
  "customer": {
    "id": "cust_xyz789",
    "name": "John Doe",
    "email": "john@example.com",
    "phone": "+91-9876543210"
  },
  "items": [...],
  "subtotal": 15000.00,
  "discount": 1500.00,
  "tax": 810.00,
  "shipping": 0.00,
  "total": 14310.00,
  "status": "confirmed",
  "payment_status": "paid",
  "payment_details": {
    "method": "razorpay",
    "transaction_id": "pay_abc123",
    "paid_at": "2024-01-15T10:35:00Z"
  },
  "shipping_address": {...},
  "billing_address": {...},
  "timeline": [
    {
      "status": "pending",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "status": "confirmed",
      "timestamp": "2024-01-15T10:35:00Z"
    }
  ],
  "notes": "Gift wrap requested",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Update Order

**Endpoint:** `PUT /admin/orders/{id}`
**Authentication:** Required

**Request Body:**
```json
{
  "notes": "Updated delivery instructions",
  "shipping_address": {
    "name": "John Doe",
    "line1": "456 New St",
    "city": "Chennai",
    "state": "Tamil Nadu",
    "postal_code": "600002",
    "country": "India"
  }
}
```

---

### Update Order Status

**Endpoint:** `PATCH /admin/orders/{id}/status`
**Authentication:** Required

**Request Body:**
```json
{
  "status": "shipped",
  "tracking_number": "SHIP123456789",
  "carrier": "BlueDart",
  "notes": "Shipped via express delivery"
}
```

**Response (200 OK):**
```json
{
  "id": "ord_abc123",
  "status": "shipped",
  "tracking_number": "SHIP123456789",
  "updated_at": "2024-01-16T09:00:00Z"
}
```

---

### Cancel Order

**Endpoint:** `POST /admin/orders/{id}/cancel`
**Authentication:** Required

**Request Body:**
```json
{
  "reason": "Customer requested cancellation",
  "refund": true
}
```

---

### Get Orders by Customer

**Endpoint:** `GET /admin/orders/customer/{customerId}`
**Authentication:** Required

---

# Customers

## Base Path
`/admin/customers`

### List Customers

**Endpoint:** `GET /admin/customers`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| search | string | - | Search by name/email/phone |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "cust_xyz789",
      "name": "John Doe",
      "email": "john@example.com",
      "phone": "+91-9876543210",
      "total_orders": 5,
      "total_spent": 75000.00,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 500,
    "total_pages": 50
  }
}
```

---

### Create Customer

**Endpoint:** `POST /admin/customers`
**Authentication:** Required

**Request Body:**
```json
{
  "name": "Jane Smith",
  "email": "jane@example.com",
  "phone": "+91-9876543211",
  "addresses": [
    {
      "type": "shipping",
      "is_default": true,
      "name": "Jane Smith",
      "line1": "789 Oak Ave",
      "city": "Bangalore",
      "state": "Karnataka",
      "postal_code": "560001",
      "country": "India"
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "id": "cust_abc123",
  "name": "Jane Smith",
  "email": "jane@example.com",
  "phone": "+91-9876543211",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Customer by ID

**Endpoint:** `GET /admin/customers/{id}`
**Authentication:** Required

---

### Update Customer

**Endpoint:** `PUT /admin/customers/{id}`
**Authentication:** Required

---

### Delete Customer

**Endpoint:** `DELETE /admin/customers/{id}`
**Authentication:** Required

---

### Get Customer Orders

**Endpoint:** `GET /admin/customers/{id}/orders`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "ord_abc123",
      "order_number": "ORD-2024-001",
      "total": 14310.00,
      "status": "delivered",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {...}
}
```

---

### Search Customers

**Endpoint:** `GET /admin/customers/search`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Search query |

---

## TODO

No pending TODO items identified.
