# Store Tracking API Documentation

Public order tracking by order number for the B2C storefront. No authentication required.

## Base Path
`/api/v1/store/track`

## Endpoints

### Track Order by Number

Look up an order's current status, status timeline, and shipment information using the order number. This is a public endpoint that does not require authentication, allowing customers to track orders via a shared link or by entering the order number directly.

**Endpoint:** `GET /api/v1/store/track/{orderNumber}`
**Authentication:** None (public)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| orderNumber | string | Human-readable order number (e.g., `ORD-2026-000123`) |

**Response (200 OK) - Shipped Order:**
```json
{
  "success": true,
  "data": {
    "order_number": "ORD-2026-000123",
    "status": "SHIPPED",
    "status_history": [
      {
        "status": "PENDING",
        "timestamp": "2026-02-15T10:30:00Z",
        "note": "Order placed"
      },
      {
        "status": "CONFIRMED",
        "timestamp": "2026-02-15T10:35:00Z",
        "note": "Payment received"
      },
      {
        "status": "SHIPPED",
        "timestamp": "2026-02-17T14:20:00Z",
        "note": "Shipped via Delhivery"
      }
    ],
    "shipment": {
      "awb_number": "SR12345678901",
      "courier_name": "Delhivery",
      "tracking_url": "https://www.delhivery.com/track/package/SR12345678901",
      "status": "IN TRANSIT",
      "estimated_delivery": "2026-02-20T18:00:00Z"
    }
  }
}
```

**Response (200 OK) - Pending Order (no shipment yet):**
```json
{
  "success": true,
  "data": {
    "order_number": "ORD-2026-000456",
    "status": "CONFIRMED",
    "status_history": [
      {
        "status": "PENDING",
        "timestamp": "2026-02-19T08:00:00Z",
        "note": "Order placed"
      },
      {
        "status": "CONFIRMED",
        "timestamp": "2026-02-19T08:05:00Z",
        "note": "Payment received"
      }
    ],
    "shipment": null
  }
}
```

**Response (200 OK) - Delivered Order:**
```json
{
  "success": true,
  "data": {
    "order_number": "ORD-2026-000089",
    "status": "DELIVERED",
    "status_history": [
      {
        "status": "PENDING",
        "timestamp": "2026-02-10T09:00:00Z",
        "note": "Order placed"
      },
      {
        "status": "CONFIRMED",
        "timestamp": "2026-02-10T09:05:00Z",
        "note": "Payment received"
      },
      {
        "status": "SHIPPED",
        "timestamp": "2026-02-11T16:30:00Z",
        "note": "Shipped via BlueDart"
      },
      {
        "status": "DELIVERED",
        "timestamp": "2026-02-14T11:45:00Z",
        "note": "Delivered successfully"
      }
    ],
    "shipment": {
      "awb_number": "BD9876543210",
      "courier_name": "BlueDart",
      "tracking_url": "https://www.bluedart.com/tracking/BD9876543210",
      "status": "DELIVERED",
      "estimated_delivery": "2026-02-14T18:00:00Z"
    }
  }
}
```

**Status History Notes:**

The `status_history` array is built from order status change records stored in DynamoDB. Each entry represents a status transition with the timestamp when it occurred. The `note` field provides a human-readable description of the transition reason.

| Status | Note Example |
|--------|-------------|
| `PENDING` | "Order placed" |
| `CONFIRMED` | "Payment received" |
| `SHIPPED` | "Shipped via {carrier}" |
| `DELIVERED` | "Delivered successfully" |
| `CANCELLED` | "Cancelled by customer" |
| `RETURNED` | "Return initiated" |

**Shipment Status Values (from Shiprocket):**

| Status | Description |
|--------|-------------|
| `PICKED UP` | Package picked up from seller |
| `IN TRANSIT` | Package in transit to destination |
| `OUT FOR DELIVERY` | Package out for delivery |
| `DELIVERED` | Package delivered to customer |
| `RTO` | Return to origin initiated |

**Error Responses:**
- `404 Not Found` - `ORDER_NOT_FOUND` - No order found with the given order number

**Security Notes:**
- This endpoint is intentionally public to allow order tracking via shared links
- Only order status, timeline, and shipping information is exposed
- No customer PII (name, address, email, phone) is returned
- No pricing or item details are returned
- The order number format (ORD-YYYY-NNNNNN) is not easily guessable

---
