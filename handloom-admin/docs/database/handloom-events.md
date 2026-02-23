# handloom-events Table

The events table stores raw tracking events from the B2C storefront for analytics processing. Events are partitioned by date for efficient daily aggregation and auto-expire after 30 days via TTL.

## Table Configuration

```
Table Name: handloom-events
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST (prod) / Provisioned (local dev)
TTL Attribute: ttl
```

**Env var:** `DYNAMODB_EVENTS_TABLE=handloom-events`

### Global Secondary Indexes

None. All access is via primary key.

---

## Design Philosophy

### Date-Based Partitioning

The events table uses **date-based partitioning** to align with the daily aggregation workflow:

```
PK: EVENT#<YYYY-MM-DD>
SK: <ISO-8601-timestamp>#<uuid>
```

**Benefits:**
- One query retrieves all events for a single aggregation run
- Natural time-ordering within each partition via sort key
- Even data distribution (one partition per day)
- Simple TTL-based cleanup with 30-day retention

### Write-Heavy, Read-Light

Events are written in high-throughput batches from the storefront and read only once during daily aggregation. The table is optimized for writes with no GSIs and minimal attribute overhead.

---

## Entity: Raw Tracking Event

### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `EVENT#<date>` | `EVENT#2026-02-23` |
| SK | `<timestamp>#<uuid>` | `2026-02-23T14:30:00.123456789Z#550e8400-e29b-41d4-a716-446655440000` |

### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| event_type | String | Yes | Event type: `page_view`, `product_viewed`, `add_to_cart`, `checkout_started`, `scroll_depth` |
| timestamp | String | Yes | ISO 8601 event timestamp |
| session_id | String | Yes | Browser session ID (crypto.randomUUID, stored in sessionStorage) |
| visitor_id | String | Yes | Persistent visitor ID (crypto.randomUUID, stored in localStorage) |
| device_type | String | No | `mobile`, `tablet`, or `desktop` |
| page_path | String | No | URL path (e.g., `/p/red-silk-saree`) |
| properties | String | No | JSON-encoded event-specific properties |
| ttl | Number | Yes | Unix epoch for 30-day auto-deletion |

### Event Types

| Type | Description |
|------|-------------|
| `page_view` | Page loaded by visitor |
| `product_viewed` | Product detail page viewed |
| `add_to_cart` | Item added to cart |
| `checkout_started` | Checkout flow initiated |
| `scroll_depth` | Maximum scroll depth recorded |

### Event Properties by Type

Each event type encodes type-specific data in the `properties` attribute as a JSON string.

#### `page_view`

```json
{}
```

Path is captured in the `page_path` attribute; no additional properties needed.

#### `product_viewed`

```json
{
  "product_id": "prod-001",
  "product_name": "Red Silk Saree",
  "category_id": "cat-001",
  "price": 150000
}
```

#### `add_to_cart`

```json
{
  "product_id": "prod-001",
  "product_name": "Red Silk Saree",
  "category_id": "cat-001",
  "price": 150000,
  "quantity": 1
}
```

#### `checkout_started`

```json
{
  "cart_total": 450000,
  "item_count": 3
}
```

#### `scroll_depth`

```json
{
  "max_depth_percent": 75,
  "page_type": "product"
}
```

> **Note:** All prices are in **paise** (1 INR = 100 paise), consistent with the rest of the platform.

---

## Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get all events for a date | PK = `EVENT#2026-02-23` (paginated query) |
| Get events in time range within a day | PK = `EVENT#2026-02-23`, SK between timestamps |

### Get All Events for a Date

Query all events recorded on a specific date. Used by the daily aggregation job.

```
Table: handloom-events
KeyCondition:
  PK = "EVENT#2026-02-23"
```

**Use Case**: Daily analytics aggregation

### Get Events in Time Range

Query events within a time window on a specific date.

```
Table: handloom-events
KeyCondition:
  PK = "EVENT#2026-02-23"
  SK BETWEEN "2026-02-23T10:00:00Z" AND "2026-02-23T18:00:00Z"
```

**Use Case**: Debugging, ad-hoc analysis

---

## Data Flow

```
1. Next.js storefront batches events (up to 10) using navigator.sendBeacon or fetch
2. POST /api/v1/store/events validates and writes events via BatchWriteItem
3. Daily cron (EventBridge) triggers AnalyticsAggregator.AggregateDate()
   → reads all events for yesterday
   → writes pre-computed aggregates to the handloom-analytics table
4. Raw events auto-expire after 30 days via TTL
```

### Write Path

1. Storefront collects events in memory, batching up to 10 per request
2. `navigator.sendBeacon` fires on page unload; `fetch` used as fallback
3. `POST /api/v1/store/events` receives the batch
4. Handler validates each event and writes via `BatchWriteItem` (max 25 items per DynamoDB batch)
5. Each event gets a `ttl` set to `now + 30 days`

### Read Path (Aggregation)

1. EventBridge rule triggers the analytics worker daily
2. `AnalyticsAggregator.AggregateDate()` queries all events for the previous day
3. Paginated query reads all items under `PK = EVENT#<yesterday>`
4. Events are grouped and counted to produce aggregates (page views, conversion funnel, top products, etc.)
5. Aggregated results are written to the `handloom-analytics` table

---

## TTL Configuration

### How TTL Works

1. Each event has a `ttl` attribute set to 30 days from creation (Unix timestamp)
2. DynamoDB periodically scans for expired items
3. Expired items are deleted automatically (within 48 hours of expiration)
4. No read/write capacity consumed for deletions

### Calculating TTL

```go
// 30 days retention
ttl := time.Now().Add(30 * 24 * time.Hour).Unix()
```

### Retention Rationale

Raw events are only needed until daily aggregation completes. The 30-day window provides a generous buffer for:
- Reprocessing failed aggregation runs
- Ad-hoc debugging and analysis
- Backfilling if aggregation logic changes

After 30 days, all insights are preserved in the `handloom-analytics` table.

---

## Example Event Records

### Page View

```json
{
  "PK": "EVENT#2026-02-23",
  "SK": "2026-02-23T14:30:00.123456789Z#550e8400-e29b-41d4-a716-446655440000",
  "event_type": "page_view",
  "timestamp": "2026-02-23T14:30:00.123456789Z",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "visitor_id": "f0e1d2c3-b4a5-6789-0123-456789abcdef",
  "device_type": "mobile",
  "page_path": "/p/red-silk-saree",
  "ttl": 1743004200
}
```

### Product Viewed

```json
{
  "PK": "EVENT#2026-02-23",
  "SK": "2026-02-23T14:30:05.456789012Z#661f9511-f30c-52e5-b827-557766551111",
  "event_type": "product_viewed",
  "timestamp": "2026-02-23T14:30:05.456789012Z",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "visitor_id": "f0e1d2c3-b4a5-6789-0123-456789abcdef",
  "device_type": "mobile",
  "page_path": "/p/red-silk-saree",
  "properties": "{\"product_id\":\"prod-001\",\"product_name\":\"Red Silk Saree\",\"category_id\":\"cat-001\",\"price\":150000}",
  "ttl": 1743004205
}
```

### Add to Cart

```json
{
  "PK": "EVENT#2026-02-23",
  "SK": "2026-02-23T14:32:15.789012345Z#772a0622-a41d-63f6-c938-668877662222",
  "event_type": "add_to_cart",
  "timestamp": "2026-02-23T14:32:15.789012345Z",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "visitor_id": "f0e1d2c3-b4a5-6789-0123-456789abcdef",
  "device_type": "mobile",
  "page_path": "/p/red-silk-saree",
  "properties": "{\"product_id\":\"prod-001\",\"product_name\":\"Red Silk Saree\",\"category_id\":\"cat-001\",\"price\":150000,\"quantity\":1}",
  "ttl": 1743004335
}
```

### Checkout Started

```json
{
  "PK": "EVENT#2026-02-23",
  "SK": "2026-02-23T14:35:00.000000000Z#883b1733-b52e-74g7-d049-779988773333",
  "event_type": "checkout_started",
  "timestamp": "2026-02-23T14:35:00.000000000Z",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "visitor_id": "f0e1d2c3-b4a5-6789-0123-456789abcdef",
  "device_type": "mobile",
  "page_path": "/checkout",
  "properties": "{\"cart_total\":450000,\"item_count\":3}",
  "ttl": 1743004500
}
```

### Scroll Depth

```json
{
  "PK": "EVENT#2026-02-23",
  "SK": "2026-02-23T14:30:30.234567890Z#994c2844-c63f-85h8-e150-880099884444",
  "event_type": "scroll_depth",
  "timestamp": "2026-02-23T14:30:30.234567890Z",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "visitor_id": "f0e1d2c3-b4a5-6789-0123-456789abcdef",
  "device_type": "mobile",
  "page_path": "/p/red-silk-saree",
  "properties": "{\"max_depth_percent\":75,\"page_type\":\"product\"}",
  "ttl": 1743004230
}
```
