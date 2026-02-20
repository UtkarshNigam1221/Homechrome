# Inventory Lambda API Documentation

Inventory management and stock tracking service.

## Base Path
`/admin/inventory`

## Authentication
All endpoints require authentication.

---

### List Inventory

Get inventory status for all products.

**Endpoint:** `GET /admin/inventory`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| low_stock | bool | - | Filter for low stock items only |
| out_of_stock | bool | - | Filter for out of stock items only |
| category_id | string | - | Filter by category |

**Response (200 OK):**
```json
{
  "data": [
    {
      "product_id": "prod_abc123",
      "product_name": "Kanchipuram Silk Saree",
      "sku": "SAR-SLK-001",
      "quantity": 25,
      "reserved_quantity": 3,
      "available_quantity": 22,
      "low_stock_threshold": 5,
      "is_low_stock": false,
      "is_out_of_stock": false,
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 100,
    "total_pages": 10
  }
}
```

---

### Get Inventory by Product ID

**Endpoint:** `GET /admin/inventory/{productId}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| productId | string | Product ID |

**Response (200 OK):**
```json
{
  "product_id": "prod_abc123",
  "product_name": "Kanchipuram Silk Saree",
  "sku": "SAR-SLK-001",
  "quantity": 25,
  "reserved_quantity": 3,
  "available_quantity": 22,
  "low_stock_threshold": 5,
  "is_low_stock": false,
  "history": [
    {
      "id": "inv_hist_001",
      "type": "restock",
      "quantity_change": 10,
      "quantity_before": 15,
      "quantity_after": 25,
      "reason": "New stock arrival",
      "created_by": "user_abc123",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### Update Inventory

**Endpoint:** `PUT /admin/inventory/{productId}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| productId | string | Product ID |

**Request Body:**
```json
{
  "quantity": 30,
  "adjustment_type": "restock",
  "reason": "New shipment received",
  "reference": "PO-2024-001"
}
```

**Adjustment Types:**
| Type | Description |
|------|-------------|
| `restock` | Adding new stock |
| `sale` | Reducing stock due to sale |
| `return` | Adding stock due to return |
| `damage` | Reducing stock due to damage |
| `adjustment` | Manual adjustment |

**Response (200 OK):**
```json
{
  "product_id": "prod_abc123",
  "quantity": 30,
  "previous_quantity": 25,
  "adjustment_type": "restock",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Bulk Update Inventory

**Endpoint:** `POST /admin/inventory/bulk`
**Authentication:** Required

**Request Body:**
```json
{
  "updates": [
    {
      "product_id": "prod_abc123",
      "quantity": 30,
      "adjustment_type": "restock"
    },
    {
      "product_id": "prod_xyz789",
      "quantity": 15,
      "adjustment_type": "restock"
    }
  ],
  "reason": "Monthly restock"
}
```

**Response (200 OK):**
```json
{
  "success_count": 2,
  "failure_count": 0,
  "results": [
    {
      "product_id": "prod_abc123",
      "success": true,
      "new_quantity": 30
    },
    {
      "product_id": "prod_xyz789",
      "success": true,
      "new_quantity": 15
    }
  ]
}
```

---

### Get Low Stock Alerts

**Endpoint:** `GET /admin/inventory/alerts`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "data": [
    {
      "product_id": "prod_def456",
      "product_name": "Cotton Saree Blue",
      "sku": "SAR-COT-002",
      "quantity": 3,
      "low_stock_threshold": 5,
      "days_of_stock": 2,
      "avg_daily_sales": 1.5,
      "alert_level": "critical"
    }
  ],
  "summary": {
    "critical_count": 5,
    "warning_count": 12,
    "out_of_stock_count": 2
  }
}
```

---

### Reserve Inventory

Reserve inventory for an order.

**Endpoint:** `POST /admin/inventory/reserve`
**Authentication:** Required

**Request Body:**
```json
{
  "order_id": "ord_abc123",
  "items": [
    {
      "product_id": "prod_abc123",
      "quantity": 2
    }
  ],
  "expiry_minutes": 30
}
```

**Response (200 OK):**
```json
{
  "reservation_id": "res_xyz789",
  "order_id": "ord_abc123",
  "items": [
    {
      "product_id": "prod_abc123",
      "reserved_quantity": 2,
      "available_after_reserve": 20
    }
  ],
  "expires_at": "2024-01-15T11:00:00Z"
}
```

---

### Release Reservation

**Endpoint:** `DELETE /admin/inventory/reserve/{reservationId}`
**Authentication:** Required

---

### Set Low Stock Threshold

**Endpoint:** `PATCH /admin/inventory/{productId}/threshold`
**Authentication:** Required

**Request Body:**
```json
{
  "low_stock_threshold": 10
}
```

---

## TODO

No pending TODO items identified.
