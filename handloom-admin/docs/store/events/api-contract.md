# Store Events API

Accepts batched frontend tracking events from the Next.js storefront for analytics. Events are stored raw in the `handloom-events` table and live dashboard counters are incremented in the `handloom-analytics` table.

## Base Path

`/api/v1/store/events`

**Authentication:** Public (no auth required, rate-limited to 60 req/min per IP).

---

### Ingest Tracking Events

Submit a batch of frontend tracking events for analytics processing.

**Endpoint:** `POST /api/v1/store/events`

**Authentication:** None (public, rate-limited)

**Request Body:**
```json
{
  "events": [
    {
      "event_type": "page_view",
      "timestamp": "2026-02-23T14:30:00Z",
      "session_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "visitor_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "device_type": "mobile",
      "page_path": "/p/red-silk-saree",
      "properties": {}
    },
    {
      "event_type": "product_viewed",
      "timestamp": "2026-02-23T14:30:05Z",
      "session_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "visitor_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "device_type": "mobile",
      "page_path": "/p/red-silk-saree",
      "properties": {
        "product_id": "prod-001-uuid",
        "price": 1850000
      }
    }
  ]
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| events | array | Yes | min=1, max=25 | Batch of tracking events |
| events[].event_type | string | Yes | enum | One of: `page_view`, `product_viewed`, `add_to_cart`, `checkout_started`, `scroll_depth` |
| events[].timestamp | string | Yes | ISO 8601 | Event timestamp (events older than 24h are filtered out) |
| events[].session_id | string | Yes | UUID | Browser session identifier |
| events[].visitor_id | string | Yes | UUID | Persistent visitor identifier |
| events[].device_type | string | No | - | Device type (e.g., `mobile`, `desktop`, `tablet`) |
| events[].page_path | string | No | - | Page path where the event occurred |
| events[].properties | object | No | - | Arbitrary event-specific properties |

**Response (202 Accepted):**
```json
{
  "success": true,
  "data": {
    "accepted": 5
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body, empty events array, or batch exceeds 25 items |
| 429 | `RATE_LIMIT_EXCEEDED` | More than 60 requests per minute from the same IP |

---

## Allowed Event Types

| Event Type | Description | Counter Updated |
|------------|-------------|-----------------|
| `page_view` | Page navigation | `today_page_views` |
| `product_viewed` | Product detail page viewed | `today_product_views` |
| `add_to_cart` | Item added to cart | `today_add_to_carts` |
| `checkout_started` | Checkout page loaded | - |
| `scroll_depth` | Scroll depth milestone reached | - |

---

## Rate Limiting

- **Limit:** 60 requests per minute per IP address
- **Response when exceeded:** HTTP 429 with standard error envelope
- **Reset:** Rolling 1-minute window
