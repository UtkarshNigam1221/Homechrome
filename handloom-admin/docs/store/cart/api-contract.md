# Store Cart API

Shopping cart management for the B2C storefront. Supports add, update, remove, clear, and guest cart merge operations.

## Base Path

`/api/v1/store/cart`

**Authentication:** All endpoints require Customer JWT (HttpOnly cookie).

---

### Get Cart

Retrieve the current customer's cart with all items.

**Endpoint:** `GET /api/v1/store/cart`

**Authentication:** Required (Customer JWT)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "cart": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "session_id": "sess_abc123def456",
      "item_count": 2,
      "subtotal": 2450000,
      "currency": "INR",
      "updated_at": "2026-02-20T14:30:00Z"
    },
    "items": [
      {
        "product_id": "prod-001-uuid",
        "product_name": "Banarasi Silk Saree - Royal Blue",
        "product_sku": "HC-SAR-BNR-001",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/saree-royal-blue.jpg",
        "quantity": 1,
        "unit_price": 1850000,
        "total_price": 1850000,
        "is_custom_size": false,
        "dimensions": null,
        "quote_id": null,
        "attributes": {
          "material": "Silk",
          "color": "Royal Blue",
          "weave_type": "Jacquard"
        },
        "added_at": "2026-02-20T14:25:00Z"
      },
      {
        "product_id": "prod-002-uuid",
        "product_name": "Handwoven Cotton Table Runner",
        "product_sku": "HC-TBR-CTN-005",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/table-runner-natural.jpg",
        "quantity": 2,
        "unit_price": 300000,
        "total_price": 600000,
        "is_custom_size": true,
        "dimensions": {
          "length": 180,
          "width": 35,
          "height": 0,
          "unit": "CM"
        },
        "quote_id": "quote-xyz-789",
        "attributes": {
          "material": "Cotton",
          "color": "Natural Beige"
        },
        "added_at": "2026-02-20T14:28:00Z"
      }
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |

---

### Add Item to Cart

Add a product to the shopping cart. If the product already exists in the cart, its quantity is incremented. For custom-sized products, a valid `quote_id` must be provided.

**Endpoint:** `POST /api/v1/store/cart/items`

**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "product_id": "prod-001-uuid",
  "quantity": 2,
  "dimensions": {
    "length": 180,
    "width": 35,
    "height": 0,
    "unit": "CM"
  },
  "quote_id": "quote-xyz-789"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| product_id | string | Yes | UUID | Product to add |
| quantity | int | Yes | gt=0 | Quantity to add |
| dimensions | object | No | - | Custom dimensions (required for custom-size products) |
| quote_id | string | No | UUID | Price quote ID (required for custom-size products) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "cart": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "session_id": "sess_abc123def456",
      "item_count": 3,
      "subtotal": 4150000,
      "currency": "INR",
      "updated_at": "2026-02-20T14:35:00Z"
    },
    "items": [
      {
        "product_id": "prod-001-uuid",
        "product_name": "Banarasi Silk Saree - Royal Blue",
        "product_sku": "HC-SAR-BNR-001",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/saree-royal-blue.jpg",
        "quantity": 2,
        "unit_price": 1850000,
        "total_price": 3700000,
        "is_custom_size": false,
        "dimensions": null,
        "quote_id": null,
        "attributes": {
          "material": "Silk",
          "color": "Royal Blue"
        },
        "added_at": "2026-02-20T14:25:00Z"
      }
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body or missing required fields |
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `PRODUCT_NOT_FOUND` | Product does not exist or is inactive |
| 409 | `INSUFFICIENT_STOCK` | Requested quantity exceeds available inventory |

---

### Update Item Quantity

Update the quantity of an existing cart item. Setting quantity to `0` removes the item from the cart.

**Endpoint:** `PATCH /api/v1/store/cart/items/{productID}`

**Authentication:** Required (Customer JWT)

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| productID | string | Product ID of the cart item |

**Request Body:**
```json
{
  "quantity": 3
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| quantity | int | Yes | gte=0 | New quantity (0 removes the item) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "cart": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "session_id": "sess_abc123def456",
      "item_count": 3,
      "subtotal": 5550000,
      "currency": "INR",
      "updated_at": "2026-02-20T14:40:00Z"
    },
    "items": [
      {
        "product_id": "prod-001-uuid",
        "product_name": "Banarasi Silk Saree - Royal Blue",
        "product_sku": "HC-SAR-BNR-001",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/saree-royal-blue.jpg",
        "quantity": 3,
        "unit_price": 1850000,
        "total_price": 5550000,
        "is_custom_size": false,
        "dimensions": null,
        "quote_id": null,
        "attributes": {
          "material": "Silk",
          "color": "Royal Blue"
        },
        "added_at": "2026-02-20T14:25:00Z"
      }
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid quantity value |
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `PRODUCT_NOT_FOUND` | Product not found in cart |
| 409 | `INSUFFICIENT_STOCK` | Requested quantity exceeds available inventory |

---

### Remove Item from Cart

Remove a specific item from the cart entirely.

**Endpoint:** `DELETE /api/v1/store/cart/items/{productID}`

**Authentication:** Required (Customer JWT)

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| productID | string | Product ID of the cart item to remove |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "cart": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "session_id": "sess_abc123def456",
      "item_count": 1,
      "subtotal": 600000,
      "currency": "INR",
      "updated_at": "2026-02-20T14:45:00Z"
    },
    "items": [
      {
        "product_id": "prod-002-uuid",
        "product_name": "Handwoven Cotton Table Runner",
        "product_sku": "HC-TBR-CTN-005",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/table-runner-natural.jpg",
        "quantity": 2,
        "unit_price": 300000,
        "total_price": 600000,
        "is_custom_size": true,
        "dimensions": {
          "length": 180,
          "width": 35,
          "height": 0,
          "unit": "CM"
        },
        "quote_id": "quote-xyz-789",
        "attributes": {
          "material": "Cotton",
          "color": "Natural Beige"
        },
        "added_at": "2026-02-20T14:28:00Z"
      }
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `PRODUCT_NOT_FOUND` | Product not found in cart |

---

### Clear Cart

Remove all items from the customer's cart.

**Endpoint:** `DELETE /api/v1/store/cart`

**Authentication:** Required (Customer JWT)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Cart cleared successfully"
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |

---

### Merge Guest Cart

Merge a guest (session-based) cart into the authenticated customer's cart after login. If a product already exists in the customer cart, the guest quantity is added. Inventory is validated for all merged items.

**Endpoint:** `POST /api/v1/store/cart/merge`

**Authentication:** Required (Customer JWT)

**Request Body:**
```json
{
  "items": [
    {
      "product_id": "prod-003-uuid",
      "quantity": 1,
      "dimensions": null,
      "quote_id": null
    },
    {
      "product_id": "prod-004-uuid",
      "quantity": 2,
      "dimensions": {
        "length": 200,
        "width": 120,
        "height": 0,
        "unit": "CM"
      },
      "quote_id": "quote-merge-001"
    }
  ]
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| items | array | Yes | min=1 | Items from the guest cart |
| items[].product_id | string | Yes | UUID | Product ID |
| items[].quantity | int | Yes | gt=0 | Quantity |
| items[].dimensions | object | No | - | Custom dimensions |
| items[].quote_id | string | No | UUID | Price quote ID |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "cart": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "customer_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "session_id": "sess_abc123def456",
      "item_count": 4,
      "subtotal": 5800000,
      "currency": "INR",
      "updated_at": "2026-02-20T14:50:00Z"
    },
    "items": [
      {
        "product_id": "prod-001-uuid",
        "product_name": "Banarasi Silk Saree - Royal Blue",
        "product_sku": "HC-SAR-BNR-001",
        "product_image": "https://cdn.homechrome.lldlab.com/assets/products/saree-royal-blue.jpg",
        "quantity": 1,
        "unit_price": 1850000,
        "total_price": 1850000,
        "is_custom_size": false,
        "dimensions": null,
        "quote_id": null,
        "attributes": {
          "material": "Silk",
          "color": "Royal Blue"
        },
        "added_at": "2026-02-20T14:25:00Z"
      }
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body or empty items array |
| 401 | `UNAUTHORIZED` | Missing or invalid customer JWT |
| 404 | `PRODUCT_NOT_FOUND` | One or more products do not exist |
| 409 | `INSUFFICIENT_STOCK` | Merged quantity exceeds available inventory |
