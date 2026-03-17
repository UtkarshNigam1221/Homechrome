# Store Orders API Documentation

Customer-facing order history and management for the B2C storefront.

## Base Path
`/api/v1/store/orders`

## Endpoints

### List Customer Orders

Retrieve a paginated list of orders belonging to the authenticated customer. Orders are returned newest first by default.

**Endpoint:** `GET /api/v1/store/orders`
**Authentication:** Required (Customer JWT)

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Items per page (max 100) |
| cursor | string | - | Cursor for next page (from `meta.next_cursor`) |
| sort_by | string | `created_at` | Sort field: `created_at`, `total_amount`, `status` |
| sort_order | string | `desc` | Sort direction: `asc`, `desc` |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "ord-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "order_number": "ORD-2026-000123",
      "customer_id": "cust-b7d1e4a2-3c5f-4890-9abc-def012345678",
      "customer_name": "Priya Sharma",
      "items": [
        {
          "id": "item-001",
          "product_id": "prod-silk-001",
          "product_name": "Kanchipuram Silk Saree - Maroon & Gold",
          "product_sku": "HL-SAR-KAN-001",
          "product_image": "https://cdn.homechrome.in/assets/products/silk-saree-001.jpg",
          "unit_price": 1250000,
          "quantity": 1,
          "total_price": 1250000
        },
        {
          "id": "item-002",
          "product_id": "prod-cot-015",
          "product_name": "Handloom Cotton Table Runner",
          "product_sku": "HL-HOM-CTR-015",
          "product_image": "https://cdn.homechrome.in/assets/products/table-runner-015.jpg",
          "unit_price": 85000,
          "quantity": 2,
          "total_price": 170000
        }
      ],
      "subtotal": 1420000,
      "discount_amount": 142000,
      "tax_amount": 76644,
      "shipping_amount": 0,
      "total_amount": 1354644,
      "currency": "INR",
      "status": "SHIPPED",
      "payment_status": "PAID",
      "shipping_address": {
        "first_name": "Priya",
        "last_name": "Sharma",
        "phone": "+919876543210",
        "address_line1": "42, MG Road",
        "address_line2": "Near City Mall",
        "city": "Bengaluru",
        "state": "Karnataka",
        "postal_code": "560001",
        "country": "IN"
      },
      "tracking_number": "SR12345678901",
      "shipping_carrier": "Delhivery",
      "created_at": "2026-02-15T10:30:00Z",
      "updated_at": "2026-02-17T14:20:00Z"
    }
  ],
  "meta": {
    "limit": 20,
    "next_cursor": "eyJQSyI6Ik9SREVSIzEyMyIsIlNLIjoiTUVUQURBVEEifQ==",
    "has_more": true
  }
}
```

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `400 Bad Request` - `VALIDATION_ERROR` - Invalid query parameters (e.g., limit > 100)

---

### Get Order Detail

Retrieve full details of a single order. Validates that the order belongs to the authenticated customer.

**Endpoint:** `GET /api/v1/store/orders/{id}`
**Authentication:** Required (Customer JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Order ID (UUID) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "ord-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "order_number": "ORD-2026-000123",
    "customer_id": "cust-b7d1e4a2-3c5f-4890-9abc-def012345678",
    "customer_name": "Priya Sharma",
    "items": [
      {
        "id": "item-001",
        "product_id": "prod-silk-001",
        "product_name": "Kanchipuram Silk Saree - Maroon & Gold",
        "product_sku": "HL-SAR-KAN-001",
        "product_image": "https://cdn.homechrome.in/assets/products/silk-saree-001.jpg",
        "unit_price": 1250000,
        "quantity": 1,
        "total_price": 1250000
      },
      {
        "id": "item-002",
        "product_id": "prod-cot-015",
        "product_name": "Handloom Cotton Table Runner",
        "product_sku": "HL-HOM-CTR-015",
        "product_image": "https://cdn.homechrome.in/assets/products/table-runner-015.jpg",
        "unit_price": 85000,
        "quantity": 2,
        "total_price": 170000
      }
    ],
    "subtotal": 1420000,
    "discount_amount": 142000,
    "tax_amount": 76644,
    "shipping_amount": 0,
    "total_amount": 1354644,
    "currency": "INR",
    "status": "SHIPPED",
    "payment_status": "PAID",
    "shipping_address": {
      "first_name": "Priya",
      "last_name": "Sharma",
      "phone": "+919876543210",
      "address_line1": "42, MG Road",
      "address_line2": "Near City Mall",
      "city": "Bengaluru",
      "state": "Karnataka",
      "postal_code": "560001",
      "country": "IN"
    },
    "tracking_number": "SR12345678901",
    "shipping_carrier": "Delhivery",
    "created_at": "2026-02-15T10:30:00Z",
    "updated_at": "2026-02-17T14:20:00Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `404 Not Found` - `ORDER_NOT_FOUND` - Order does not exist or does not belong to this customer

---

### Cancel Order

Cancel a pending or confirmed order. Only orders with status PENDING or CONFIRMED can be cancelled. Cancellation releases any reserved inventory.

**Endpoint:** `POST /api/v1/store/orders/{id}/cancel`
**Authentication:** Required (Customer JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Order ID (UUID) |

**Request Body:** None

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Order cancelled successfully"
  }
}
```

**Side Effects:**
- Updates order status to `CANCELLED`
- Sets `CancelledAt` timestamp on the order
- Releases reserved inventory for all order items
- Initiates refund if payment status is `PAID`
- Sends cancellation notification SMS to customer

**Error Responses:**
- `401 Unauthorized` - `UNAUTHORIZED` - Customer not authenticated
- `404 Not Found` - `ORDER_NOT_FOUND` - Order does not exist or does not belong to this customer
- `400 Bad Request` - `ORDER_NOT_CANCELLABLE` - Order status is not PENDING or CONFIRMED (e.g., already shipped, delivered, or cancelled)

---
