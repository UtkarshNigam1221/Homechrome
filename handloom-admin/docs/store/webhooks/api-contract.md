# Store Webhooks API Documentation

Webhook handlers for external service callbacks. These endpoints receive payment and shipping notifications from PhonePe and Shiprocket respectively. All endpoints are public but protected via signature verification (PhonePe) or IP whitelisting (Shiprocket).

## Base Path
`/api/v1/store/webhooks`

## Endpoints

### PhonePe Payment Callback

Receive payment status callback from PhonePe Standard Checkout v2. The payload contains a JSON body with an `event` field and payment details. The `Authorization` header contains `SHA256(username:password)` for webhook verification. This endpoint is idempotent: if the payment has already been processed, it skips the update and still returns 200.

**Endpoint:** `POST /api/v1/store/webhooks/phonepe`
**Authentication:** None (verified via `Authorization` header with SHA256 of webhook credentials)

**Request Headers:**
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Authorization` | string | Yes | `SHA256(PHONEPE_WEBHOOK_USERNAME:PHONEPE_WEBHOOK_PASSWORD)` |

**Request Body:**
```json
{
  "event": "checkout.order.completed",
  "payload": {
    "orderId": "OMO2502070957301614536178",
    "merchantOrderId": "HC-ord-f47ac10b-58cc-4372",
    "state": "COMPLETED",
    "amount": 1354644,
    "paymentDetails": [
      {
        "paymentMode": "UPI",
        "transactionId": "T2026022012345678901234",
        "amount": 1354644
      }
    ]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event` | string | Yes | Event type: `checkout.order.completed` or `checkout.order.failed` |
| `payload` | object | Yes | Payment result details |
| `payload.orderId` | string | Yes | PhonePe order reference |
| `payload.merchantOrderId` | string | Yes | Our merchant transaction ID (maps to PaymentID) |
| `payload.state` | string | Yes | `COMPLETED` or `FAILED` |
| `payload.amount` | number | Yes | Amount in paise |
| `payload.paymentDetails` | array | Yes | Payment instrument details |
```

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

**Side Effects (on `checkout.order.completed`):**
- Verifies Authorization header against SHA256(webhook_username:webhook_password)
- Parses JSON payload directly (no base64 decoding)
- Looks up payment by `payload.merchantOrderId`
- If payment already processed (idempotent check): skip, return 200
- Updates payment status to `SUCCESS`
- Updates order status from `PENDING` to `CONFIRMED`
- Sends order confirmation SMS to customer

**Side Effects (on `checkout.order.failed`):**
- Verifies Authorization header against SHA256(webhook_username:webhook_password)
- Parses JSON payload directly (no base64 decoding)
- Looks up payment by `payload.merchantOrderId`
- If payment already processed (idempotent check): skip, return 200
- Updates payment status to `FAILED`
- Releases reserved inventory for all order items
- Sends payment failure SMS to customer

**Error Handling:**
- Always returns `200 OK` regardless of internal processing outcome to prevent PhonePe retries
- Authorization verification failure is logged but still returns 200
- Internal errors are logged and monitored via CloudWatch

---

### Shiprocket Shipping Callback

Receive shipment status updates from Shiprocket. Called whenever a shipment status changes (picked up, in transit, delivered, RTO, etc.).

**Endpoint:** `POST /api/v1/store/webhooks/shiprocket`
**Authentication:** None (public, validated by Shiprocket webhook configuration)

**Request Body:**
```json
{
  "awb": "SR12345678901",
  "current_status": "Delivered",
  "shipment_id": 123456789,
  "order_id": "ORD-2026-000123",
  "etd": "2026-02-20 18:00:00"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `awb` | string | Yes | Air Waybill number |
| `current_status` | string | Yes | Current shipment status from Shiprocket |
| `shipment_id` | number | Yes | Shiprocket shipment ID |
| `order_id` | string | Yes | Order number (as configured in Shiprocket) |
| `etd` | string | No | Estimated time of delivery |

**Shiprocket Status Mapping:**

| Shiprocket Status | Internal Order Status | Action |
|-------------------|----------------------|--------|
| `Pickup Scheduled` | (no change) | Update shipment record |
| `Picked Up` | (no change) | Update shipment record |
| `In Transit` | (no change) | Update shipment record |
| `Out For Delivery` | (no change) | Update shipment record, send SMS |
| `Delivered` | `DELIVERED` | Update order status, update customer stats |
| `RTO Initiated` | (no change) | Update shipment record, alert admin |
| `RTO Delivered` | `RETURNED` | Update order status, initiate refund |

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

**Side Effects (on Delivered):**
- Updates shipment record with final status
- Updates order status to `DELIVERED`
- Sets `DeliveredAt` timestamp on order
- Increments customer `TotalOrders` count
- Adds order `TotalAmount` to customer `TotalSpent`
- Sends delivery confirmation SMS to customer

**Side Effects (on RTO Delivered):**
- Updates shipment record with RTO status
- Updates order status to `RETURNED`
- Initiates refund process if payment was `PAID`
- Sends return notification SMS to customer
- Creates admin alert for manual review

**Error Handling:**
- Always returns `200 OK` regardless of internal processing outcome to prevent Shiprocket retries
- Unknown status values are logged and the shipment record is updated, but no order status change occurs
- Internal errors are logged and monitored via CloudWatch

---
