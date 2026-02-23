# Store Events - High-Level Design (HLD)

## 1. Overview

The Store Events service ingests batched frontend tracking events from the Next.js B2C storefront. Events are written raw to the `handloom-events` DynamoDB table and used to increment live dashboard counters in the `handloom-analytics` table. The endpoint is public (no authentication required) but rate-limited to 60 requests per minute per IP. All writes are best-effort -- failures are logged but never propagated to the caller.

---

## 2. Architecture Diagram

```
+---------------------------------------------------------------------------+
|                          EVENTS SYSTEM                                      |
+---------------------------------------------------------------------------+

                          +-------------------+
                          |  Next.js Frontend |
                          |  (B2C Storefront) |
                          |                   |
                          |  In-memory buffer  |
                          |  (max 10 events)  |
                          |  Flush: 30s timer |
                          |  / page unload /  |
                          |  visibility change|
                          +--------+----------+
                                   |
                                   | POST /api/v1/store/events
                                   | (sendBeacon on unload)
                                   v
                          +-------------------+
                          |   API Gateway /   |
                          |   Lambda / Local  |
                          +--------+----------+
                                   |
                                   | Rate Limit: 60 req/min per IP
                                   v
                          +-------------------+
                          |  Events Handler   |
                          | (store/events)    |
                          |                   |
                          |  POST /           |
                          +--------+----------+
                                   |
                    +--------------+--------------+
                    |                              |
                    v                              v
          +------------------+          +--------------------+
          | Events Repository|          | Analytics Repository|
          | (DynamoDB)       |          | (DynamoDB)          |
          |                  |          |                     |
          | handloom-events  |          | handloom-analytics  |
          | BatchWriteItem   |          | UpdateItem ADD      |
          | (25-item chunks, |          | DASHBOARD#CURRENT   |
          |  retry unproc.)  |          |                     |
          +------------------+          +--------------------+
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
+-------------------------------------------------------------------------+
|                       EVENTS SERVICE LAYER                                |
+-------------------------------------------------------------------------+
|                                                                           |
|  +-------------------------------------------------------------------+  |
|  |                       Handler Layer                                 |  |
|  |                                                                     |  |
|  |  EventsHandler (internal/handler/store/events_handler.go)           |  |
|  |  Routes() chi.Router                                                |  |
|  |                                                                     |  |
|  |  +-------------------+                                              |  |
|  |  |  IngestEvents     |                                              |  |
|  |  |  POST /           |                                              |  |
|  |  +--------+----------+                                              |  |
|  +-----------|-----------------------------------------------------+   |  |
|              |                                                          |  |
|              v                                                          |  |
|  +-------------------------------------------------------------------+  |
|  |                  Inline Processing (no service layer)               |  |
|  |                                                                     |  |
|  |  1. Validate batch (struct validation)                              |  |
|  |  2. Filter: remove events > 24h old + invalid event types           |  |
|  |  3. Write raw events to handloom-events (EventsRepository)          |  |
|  |  4. Increment dashboard counters (AnalyticsRepository)              |  |
|  +-------------------------------------------------------------------+  |
|              |                                                          |  |
|     +--------+--------+                                                 |  |
|     |                  |                                                |  |
|     v                  v                                                |  |
|  +-----------------+  +--------------------+                            |  |
|  | EventsRepository|  | AnalyticsRepository|                            |  |
|  | (DynamoDB)      |  | (DynamoDB)         |                            |  |
|  +-----------------+  +--------------------+                            |  |
|                                                                           |
+-------------------------------------------------------------------------+
```

### 3.2 Key Design Decisions

- **No service layer**: The handler performs validation, filtering, and repository calls directly. The flow is simple enough that an intermediate service adds no value.
- **Best-effort writes**: Both raw event writes and counter increments are fire-and-forget. Failures are logged but the handler always returns 202 with the count of accepted events.
- **Batch chunking**: DynamoDB `BatchWriteItem` accepts at most 25 items. Events are chunked accordingly, with automatic retry for unprocessed items.
- **Atomic counter increments**: Dashboard counters use DynamoDB `UpdateItem` with `ADD` expression to atomically increment, avoiding read-modify-write races.
- **24-hour cutoff**: Events with timestamps older than 24 hours are silently filtered to prevent replay attacks and stale data injection.
- **Event type allowlist**: Only the five recognized event types are accepted; unknown types are filtered out before writing.

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
+-------------------------------------------------------------------------+
|                    TABLE: handloom-events                                  |
+-------------------------------------------------------------------------+
|                                                                           |
|  RAW EVENT                                                                |
|  +-------------------------------------------------------------------+  |
|  | PK: EVENT#<YYYY-MM-DD>                                             |  |
|  | SK: <timestamp>#<uuid>                                             |  |
|  | entity_type: TRACKING_EVENT                                        |  |
|  |                                                                     |  |
|  | Attributes:                                                         |  |
|  |   - event_type      (page_view, product_viewed, etc.)              |  |
|  |   - timestamp        (ISO 8601)                                     |  |
|  |   - session_id       (UUID)                                         |  |
|  |   - visitor_id       (UUID)                                         |  |
|  |   - device_type      (mobile, desktop, tablet)                     |  |
|  |   - page_path        (string)                                       |  |
|  |   - properties        (map)                                         |  |
|  |   - ttl               (Unix timestamp, 90 days)                     |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  ACCESS PATTERNS                                                          |
|  +-------------------------------------------------------------------+  |
|  | Query by date: PK = EVENT#<YYYY-MM-DD> (paginated)                 |  |
|  | Used by daily aggregation cron to compute aggregates                |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+

+-------------------------------------------------------------------------+
|                    TABLE: handloom-analytics                               |
+-------------------------------------------------------------------------+
|                                                                           |
|  DASHBOARD COUNTER (live)                                                 |
|  +-------------------------------------------------------------------+  |
|  | PK: DASHBOARD#CURRENT                                              |  |
|  | SK: COUNTERS                                                       |  |
|  |                                                                     |  |
|  | Attributes (atomically incremented via ADD):                        |  |
|  |   - today_page_views     (number)                                   |  |
|  |   - today_product_views  (number)                                   |  |
|  |   - today_add_to_carts   (number)                                   |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+
```

### 4.2 Counter Mapping

| Event Type | Counter Field |
|------------|---------------|
| `page_view` | `today_page_views` |
| `product_viewed` | `today_product_views` |
| `add_to_cart` | `today_add_to_carts` |
| `checkout_started` | (no live counter) |
| `scroll_depth` | (no live counter) |

---

## 5. Security

```
+-------------------------------------------------------------------------+
|                           SECURITY MODEL                                  |
+-------------------------------------------------------------------------+
|                                                                           |
|  Authentication:                                                          |
|  +-------------------------------------------------------------------+  |
|  | - No authentication required (public endpoint)                     |  |
|  | - Rate-limited to 60 requests per minute per IP address            |  |
|  | - Rate limiting prevents abuse and DoS                             |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Input Validation:                                                        |
|  +-------------------------------------------------------------------+  |
|  | - Batch size: min 1, max 25 events                                 |  |
|  | - Event type: allowlist of 5 recognized types                      |  |
|  | - Timestamp: required, events older than 24h silently filtered      |  |
|  | - session_id / visitor_id: required, UUID format                    |  |
|  | - Properties: arbitrary map (no schema enforcement)                 |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Data Integrity:                                                          |
|  +-------------------------------------------------------------------+  |
|  | - 24h timestamp cutoff prevents replay of stale events             |  |
|  | - Event type allowlist prevents injection of unknown event types    |  |
|  | - Best-effort writes ensure no data loss blocks the caller          |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+
```

---

## 6. Error Handling

```
+-------------------------------------------------------------------------+
|                              ERROR CODES                                  |
+-------------------------------------------------------------------------+
|                                                                           |
|  Events Errors:                                                           |
|  +-------------------------------------------------------------------+  |
|  | VALIDATION_ERROR     | 400 | Invalid body, empty or oversized batch|  |
|  | RATE_LIMIT_EXCEEDED  | 429 | Exceeded 60 req/min per IP            |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Error Response Format:                                                   |
|  +-------------------------------------------------------------------+  |
|  | {                                                                   |  |
|  |   "success": false,                                                 |  |
|  |   "error": {                                                        |  |
|  |     "code": "RATE_LIMIT_EXCEEDED",                                  |  |
|  |     "message": "Too many requests, please try again later"          |  |
|  |   }                                                                 |  |
|  | }                                                                   |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Internal Failures (not exposed to client):                               |
|  +-------------------------------------------------------------------+  |
|  | - DynamoDB BatchWriteItem failures: logged, retried for             |  |
|  |   unprocessed items, remaining failures logged and dropped          |  |
|  | - Counter increment failures: logged and dropped                    |  |
|  | - Handler always returns 202 with count of accepted events          |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+
```

---

## 7. Integration Points

```
+-------------------------------------------------------------------------+
|                          INTEGRATION POINTS                               |
+-------------------------------------------------------------------------+
|                                                                           |
|  +-------------------------------------------------------------------+  |
|  |                     Next.js Storefront (Upstream)                   |  |
|  |                                                                     |  |
|  |  initAnalytics() → starts in-memory buffer + 30s flush timer       |  |
|  |  track(eventType, properties) → adds event to buffer               |  |
|  |  Buffer flushed: 10 events reached / 30s timer / page unload       |  |
|  |  sendBeacon used on page unload for reliable delivery              |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  +-------------------------------------------------------------------+  |
|  |                     Daily Aggregation (Downstream)                  |  |
|  |                                                                     |  |
|  |  EventBridge → worker-analytics (scheduled daily)                  |  |
|  |  Reads raw events from handloom-events by date partition            |  |
|  |  Computes 5 aggregate types: funnel, revenue, customers,           |  |
|  |    engagement, products                                             |  |
|  |  Writes aggregates to handloom-analytics                            |  |
|  |  Resets DASHBOARD#CURRENT counters                                  |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+
```

---

## 8. Dependencies

```
+-------------------------------------------------------------------------+
|                              DEPENDENCIES                                 |
+-------------------------------------------------------------------------+
|                                                                           |
|  AWS Services:                                                            |
|  +-------------------------------------------------------------------+  |
|  | - DynamoDB (handloom-events table) -- Raw event storage            |  |
|  | - DynamoDB (handloom-analytics table) -- Dashboard counters        |  |
|  | - CloudWatch -- Logging and metrics                                |  |
|  | - API Gateway -- HTTP endpoint (Lambda mode)                       |  |
|  | - Lambda -- Compute (production)                                   |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Internal Components:                                                     |
|  +-------------------------------------------------------------------+  |
|  | - EventsRepository (handloom-events) -- BatchWriteItem operations  |  |
|  | - AnalyticsRepository (handloom-analytics) -- Counter increments   |  |
|  | - Rate limiter middleware -- IP-based throttling                    |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
|  Go Packages:                                                             |
|  +-------------------------------------------------------------------+  |
|  | - github.com/go-chi/chi/v5 -- HTTP router                         |  |
|  | - github.com/aws/aws-sdk-go-v2/service/dynamodb -- DynamoDB client |  |
|  | - github.com/google/wire -- Compile-time dependency injection       |  |
|  | - github.com/go-playground/validator/v10 -- Request validation      |  |
|  +-------------------------------------------------------------------+  |
|                                                                           |
+-------------------------------------------------------------------------+
```
