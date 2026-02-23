# Store Checkout API

Order placement and payment initiation for the B2C storefront. Handles delivery serviceability checks, order creation from cart, PhonePe payment integration, and payment status polling.

## Base Path

`/api/v1/store/checkout`

**Authentication:** All endpoints require Customer JWT (HttpOnly cookie).

---

### Check Serviceability

Check whether delivery is available to a given pincode. Returns available courier options with rates and estimated delivery times. Uses the Shiprocket API with the store's pickup pincode from configuration.

**Endpoint:** `POST /api/v1/store/checkout/serviceability`

**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "pincode": "560001"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| pincode | string | Yes | required, len=6 | Indian postal code (6 digits) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "serviceable": true,
    "couriers": [
      {
        "id": 1,
        "name": "Delhivery",
        "rate": 5000,
        "estimated_days": 4
      },
      {
        "id": 2,
        "name": "BlueDart",
        "rate": 7500,
        "estimated_days": 3
      },
      {
        "id": 3,
        "name": "DTDC",
        "rate": 4500,
        "estimated_days": 6
      }
    ]
  }
}
```

**Response (200 OK - Not Serviceable):**
```json
{
  "success": true,
  "data": {
    "serviceable": false,
    "couriers": []
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid pincode format (must be exactly 6 digits) |
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |

---

### Initiate Checkout

Place an order from the customer's cart and initiate payment via PhonePe. This is the primary checkout endpoint that orchestrates the entire order creation pipeline.

**Side effects:**
1. Validates cart is not empty
2. Validates shipping address belongs to customer
3. Checks inventory availability for all cart items
4. Creates Order (status=PENDING, payment_status=PENDING)
5. Reserves inventory for all items
6. Initiates PhonePe payment and obtains redirect URL
7. Clears the customer's cart

**Endpoint:** `POST /api/v1/store/checkout/initiate`

**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "shipping_address_id": "addr-001-uuid",
  "courier_id": 1
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| shipping_address_id | string | Yes | UUID | ID of saved shipping address |
| courier_id | int | No | - | Selected courier (from serviceability check) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "order": {
      "id": "ord-f47ac10b-58cc-4372",
      "order_number": "ORD-2026-000142",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Priya Sharma",
      "customer_email": "priya.sharma@gmail.com",
      "customer_phone": "+919876543210",
      "items": [
        {
          "id": "item-001-uuid",
          "product_id": "prod-001-uuid",
          "product_name": "Banarasi Silk Saree - Royal Blue",
          "product_sku": "HC-SAR-BNR-001",
          "product_image": "https://cdn.homechrome.lldlab.com/assets/products/saree-royal-blue.jpg",
          "is_custom_size": false,
          "dimensions": null,
          "attributes": null,
          "quote_id": null,
          "unit_price": 1850000,
          "quantity": 1,
          "total_price": 1850000
        },
        {
          "id": "item-002-uuid",
          "product_id": "prod-002-uuid",
          "product_name": "Handwoven Cotton Table Runner",
          "product_sku": "HC-TBR-CTN-005",
          "product_image": "https://cdn.homechrome.lldlab.com/assets/products/table-runner-natural.jpg",
          "is_custom_size": true,
          "dimensions": {
            "length": 180,
            "width": 35,
            "height": 0,
            "unit": "CM"
          },
          "attributes": {
            "material": "Cotton",
            "color": "Natural Beige"
          },
          "quote_id": "quote-xyz-789",
          "unit_price": 300000,
          "quantity": 2,
          "total_price": 600000
        }
      ],
      "item_count": 2,
      "subtotal": 2450000,
      "discount_amount": 0,
      "tax_amount": 0,
      "shipping_amount": 5000,
      "total_amount": 2455000,
      "currency": "INR",
      "coupon_id": null,
      "coupon_code": null,
      "status": "PENDING",
      "payment_status": "PENDING",
      "payment_method": "",
      "payment_id": "",
      "shipping_address": {
        "id": "addr-001-uuid",
        "first_name": "Priya",
        "last_name": "Sharma",
        "phone": "+919876543210",
        "address_line1": "42 MG Road",
        "address_line2": "Indiranagar",
        "city": "Bengaluru",
        "state": "Karnataka",
        "postal_code": "560038",
        "country": "India",
        "is_default": true
      },
      "billing_address": null,
      "tracking_number": "",
      "tracking_url": "",
      "shipping_carrier": "",
      "customer_note": "",
      "internal_notes": null,
      "shipped_at": null,
      "delivered_at": null,
      "cancelled_at": null,
      "created_at": "2026-02-20T15:00:00Z",
      "updated_at": "2026-02-20T15:00:00Z"
    },
    "redirect_url": "https://pay.phonepe.com/pg/checkout/HC-ord-f47ac10b-58cc-4372",
    "merchant_txn_id": "HC-ord-f47ac10b-58cc-4372"
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body or missing required fields |
| 400 | `EMPTY_CART` | Customer's cart is empty |
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `ADDRESS_NOT_FOUND` | Shipping address not found or does not belong to customer |
| 409 | `INSUFFICIENT_STOCK` | One or more items have insufficient inventory |
| 422 | `DELIVERY_NOT_SERVICEABLE` | Delivery to the address pincode is not available |
| 500 | `PAYMENT_INITIATION_FAILED` | PhonePe payment initiation failed |

---

### Get Payment Status

Poll the payment status for an order. Used by the frontend after the customer returns from the PhonePe payment page. Validates that the order belongs to the authenticated customer.

**Endpoint:** `GET /api/v1/store/checkout/payment-status/{orderID}`

**Authentication:** Required (Customer JWT)

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| orderID | string | Order ID to check payment status for |

**Response (200 OK - Payment Completed):**
```json
{
  "success": true,
  "data": {
    "payment_status": "PAID",
    "order": {
      "id": "ord-f47ac10b-58cc-4372",
      "order_number": "ORD-2026-000142",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Priya Sharma",
      "customer_email": "priya.sharma@gmail.com",
      "customer_phone": "+919876543210",
      "items": [],
      "item_count": 2,
      "subtotal": 2450000,
      "discount_amount": 0,
      "tax_amount": 0,
      "shipping_amount": 5000,
      "total_amount": 2455000,
      "currency": "INR",
      "status": "CONFIRMED",
      "payment_status": "PAID",
      "payment_method": "UPI",
      "payment_id": "PAY-abc123def456",
      "shipping_address": {},
      "created_at": "2026-02-20T15:00:00Z",
      "updated_at": "2026-02-20T15:02:30Z"
    }
  }
}
```

**Response (200 OK - Payment Pending):**
```json
{
  "success": true,
  "data": {
    "payment_status": "PENDING",
    "order": {
      "id": "ord-f47ac10b-58cc-4372",
      "order_number": "ORD-2026-000142",
      "status": "PENDING",
      "payment_status": "PENDING",
      "total_amount": 2455000,
      "currency": "INR",
      "created_at": "2026-02-20T15:00:00Z",
      "updated_at": "2026-02-20T15:00:00Z"
    }
  }
}
```

**Response (200 OK - Payment Failed):**
```json
{
  "success": true,
  "data": {
    "payment_status": "FAILED",
    "order": {
      "id": "ord-f47ac10b-58cc-4372",
      "order_number": "ORD-2026-000142",
      "status": "CANCELLED",
      "payment_status": "FAILED",
      "total_amount": 2455000,
      "currency": "INR",
      "cancelled_at": "2026-02-20T15:05:00Z",
      "created_at": "2026-02-20T15:00:00Z",
      "updated_at": "2026-02-20T15:05:00Z"
    }
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `ORDER_NOT_FOUND` | Order does not exist or does not belong to customer |
