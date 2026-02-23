# Storefront Analytics Design

## Overview

Self-hosted analytics pipeline for the Homechrome B2C storefront. Captures frontend behavioral events and backend transactional events to produce business metrics across five areas: purchase funnel, revenue, customers, product catalog, and engagement.

## Goals

- Track the full customer journey from page view to purchase
- Provide real-time dashboard counters for daily operations
- Enable historical trend analysis (daily, monthly)
- Keep costs under $1/month at 300 visitors/day

## Non-Goals

- Session replay or heatmaps (use a third-party tool later if needed)
- A/B testing infrastructure
- Real-time alerting (admin checks dashboard manually)

---

## Metrics

### 1. Purchase Funnel

| Metric | Source | Computation |
|--------|--------|-------------|
| Product page views | `product_viewed` event | Count per product/day |
| Add-to-cart rate | `product_viewed` + `add_to_cart` | add_to_carts / product_views |
| Cart abandonment rate | `add_to_cart` + `checkout_step` | carts_with_items - checkout_starts / carts_with_items |
| Checkout start rate | `checkout_step(step=address)` | checkout_starts / carts_with_items |
| Checkout completion rate | `payment.received` | payments_success / checkout_starts |
| Payment failure rate | `payment.failed` + `payment.received` | failures / (failures + successes) |
| Overall conversion rate | `product_viewed` → `payment.received` | successful_orders / product_views |

### 2. Revenue & Orders

| Metric | Source | Computation |
|--------|--------|-------------|
| Total revenue (daily/weekly/monthly) | `order.created` | Sum of order totals |
| Average order value | `order.created` | total_revenue / order_count |
| Orders count | `order.created` | Count per period |
| Revenue by category | `order.created` items (requires `category_id` on OrderItem) | Sum per category |
| Revenue by region/city | `order.created` shipping_address | Sum per state/city |
| Refund/cancellation rate | `order.cancelled` + `payment.refunded` | (cancelled + refunded) / total_orders |

### 3. Customer

| Metric | Source | Computation |
|--------|--------|-------------|
| New vs returning customers | `customer.registered` + order history | First-order customers vs repeat |
| Customer lifetime value | Order totals per customer | Sum of all orders / customer count |
| Repeat purchase rate | Order count per customer | customers_with_2plus_orders / total_customers |
| Time between purchases | Order timestamps per customer | Avg gap between consecutive orders |
| Acquisition by channel | `page_view` UTM params | Count customers by utm_source/utm_medium |
| Registration-to-first-purchase time | `customer.registered` → first `order.created` | Avg time delta |

### 4. Product & Catalog

| Metric | Source | Computation |
|--------|--------|-------------|
| Top viewed products | `product_viewed` | Count by product_id, ranked |
| Top selling products | `order.created` items | Units sold + revenue, ranked |
| View-to-purchase ratio | `product_viewed` + order items | purchases / views per product |
| Category performance | `order.created` items | Orders + revenue per category |
| Inventory turnover | Inventory events + order items | Units sold / avg inventory |
| Low/out-of-stock frequency | `inventory.low_stock`, `inventory.out_of_stock` | Count per product/period |

### 5. Storefront Engagement

| Metric | Source | Computation |
|--------|--------|-------------|
| Page views by type | `page_view` | Count by page_type |
| Session duration | First/last event per session_id | Avg time delta |
| Bounce rate | Sessions with 1 page_view | single_page_sessions / total_sessions |
| Device breakdown | `device_type` on all events | % by device |
| Scroll depth | `scroll_depth` | Avg max_depth_percent by page_type |
| Image gallery interaction rate | `product_image_interaction` | sessions_with_interaction / product_page_sessions |
| Filter/sort usage | `filter_used`, `sort_used` | Count by filter_type/sort_by |
| Exit pages | Last page_view per session | Count by page_path |
| Checkout step drop-off | `checkout_step` | Count by step where action=abandoned |
| OTP login completion rate | `otp_flow` | verified / requested |
| Error page views | `error_viewed` | Count by error_type |
| Share actions | `share_action` | Count by product/method |
| Search queries | `search_query` | Count by query, avg results_count |
| Return visit frequency | visitor_id across sessions | Avg sessions per visitor |

---

## Event Schema

### Common Fields (all frontend events)

```json
{
  "event_type":  "string",
  "timestamp":   "ISO-8601",
  "session_id":  "UUID (per browser session)",
  "visitor_id":  "UUID (persisted in localStorage across sessions)",
  "device_type": "mobile | desktop | tablet",
  "page_path":   "/p/silk-bedsheet-001",
  "properties":  {}
}
```

### Frontend Events

| Event Type | Properties |
|------------|------------|
| `page_view` | `page_type`, `referrer`, `category_slug`, `product_slug`, `utm_source`, `utm_medium`, `utm_campaign` |
| `product_viewed` | `product_id`, `product_name`, `category_id`, `price` |
| `product_image_interaction` | `product_id`, `image_count`, `images_viewed` |
| `add_to_cart` | `product_id`, `product_name`, `category_id`, `price`, `quantity`, `success` |
| `remove_from_cart` | `product_id`, `category_id` |
| `filter_used` | `filter_type`, `filter_value`, `category_slug` |
| `sort_used` | `sort_by`, `category_slug` |
| `scroll_depth` | `page_type`, `page_path`, `max_depth_percent` |
| `checkout_step` | `step` (address/shipping/payment), `action` (entered/completed/abandoned) |
| `otp_flow` | `action` (requested/entered/verified/failed) |
| `error_viewed` | `error_type` (404/500/out_of_stock), `page_path` |
| `share_action` | `product_id`, `method` (copy_link/whatsapp) |
| `search_query` | `query`, `results_count` |

### Backend Events (existing SNS pipeline, unchanged)

| Event Type | Key Data in Payload |
|------------|---------------------|
| `order.created` | Full order: items (with category_id), totals, shipping_address (city, state, postal_code) |
| `order.cancelled` | Order ID, reason |
| `payment.received` | Order ID, amount, method |
| `payment.failed` | Order ID, amount, error |
| `payment.refunded` | Order ID, amount |
| `customer.registered` | Customer ID, phone |
| `inventory.low_stock` | Product ID, current quantity |
| `inventory.out_of_stock` | Product ID |

---

## Storage

### Raw Events Table: `handloom-events-{env}` (new)

Stores every event as-is. Source of truth for recomputing aggregates.

| Attribute | Value |
|-----------|-------|
| PK | `EVENT#YYYY-MM-DD` |
| SK | `{timestamp}#{event_id}` |
| TTL | 30 days from timestamp |
| Billing | PAY_PER_REQUEST |

No GSIs. Queried only during daily aggregation by date partition.

### Aggregated Metrics Table: `handloom-analytics-{env}` (existing)

Pre-computed counters and summaries queried by the admin dashboard.

| PK Pattern | SK | Content |
|------------|-----|---------|
| `DASHBOARD#CURRENT` | `METADATA` | Live counters: today's orders, revenue, visitors, conversion, add-to-carts, payment failures |
| `FUNNEL#DAILY#{date}` | `METADATA` | product_views, add_to_carts, checkout_starts, payments_success, payments_failed, conversion_rate, checkout_drop_off by step |
| `REVENUE#DAILY#{date}` | `METADATA` | total, aov, order_count, by_category (list), by_region (list) |
| `REVENUE#MONTHLY#{month}` | `METADATA` | Same as daily, monthly rollup |
| `PRODUCT_VIEWS#DAILY#{date}` | `METADATA` | top_products: [{product_id, views, add_to_carts, purchases, view_to_purchase}], top_shared |
| `CUSTOMERS#DAILY#{date}` | `METADATA` | new, returning, repeat_rate, avg_clv, otp_completion_rate, acquisition_channels |
| `ENGAGEMENT#DAILY#{date}` | `METADATA` | sessions, avg_duration, bounce_rate, device_breakdown, top_pages, top_exit_pages, scroll_depth_avg, image_interaction_rate, filter_usage, errors |
| `CHECKOUT#DAILY#{date}` | `METADATA` | step_counts, drop_offs by step, address_abandonment, payment_failures |
| `LOCATION#DAILY#{date}` | `METADATA` | by_state: [{state, orders, revenue}], by_city: top 20 |
| `CATALOG#DAILY#{date}` | `METADATA` | category_performance, filter_usage, sort_usage, search_queries |

---

## Data Flow

### Frontend Events

```
Browser (Next.js storefront)
  │
  │  Events buffer in memory (page_view, add_to_cart, ...)
  │
  ├─ Every 30 seconds ──────────►  POST /api/v1/store/events
  ├─ Batch hits 10 events ──────►    (rate-limited, no auth required)
  └─ Page unload (sendBeacon) ──►       │
                                        ▼
                                 Store Events Lambda
                                        │
                               ┌────────┴────────┐
                               ▼                  ▼
                         BatchWriteItem      UpdateItem + ADD
                         to handloom-events  on DASHBOARD#CURRENT
```

### Backend Events (existing, unchanged)

```
Order/Payment/Inventory Service
  │
  ▼
event.Publish(order.created, order)
  │
  ▼
SNS (handloom-events topic)
  │
  ▼
SQS (analytics queue)
  │
  ▼
worker-analytics Lambda
  │
  ├─► Write raw event to handloom-events table
  └─► Update live counters on DASHBOARD#CURRENT
```

### Daily Aggregation

```
EventBridge cron (00:30 UTC daily)
  │
  ▼
worker-analytics Lambda (scheduled mode)
  │
  ▼
Query handloom-events for yesterday (PK = EVENT#YYYY-MM-DD)
  │
  ▼
Compute all DAILY aggregates → BatchWriteItem to handloom-analytics
  │
  ▼
Reset DASHBOARD#CURRENT counters for new day
```

Monthly rollups run on the 1st of each month — sum daily records into MONTHLY records.

### Admin Dashboard Reads

```
GET /admin/analytics/dashboard    → DASHBOARD#CURRENT
GET /admin/analytics/funnel       → FUNNEL#DAILY records for date range
GET /admin/analytics/sales        → REVENUE#DAILY/MONTHLY records
GET /admin/analytics/customers    → CUSTOMERS#DAILY records
GET /admin/analytics/engagement   → ENGAGEMENT#DAILY records
GET /admin/analytics/products     → PRODUCT_VIEWS#DAILY records
GET /admin/analytics/inventory    → CATALOG#DAILY + inventory events
```

---

## API Contracts

### Event Ingestion

**`POST /api/v1/store/events`** — public, no auth, rate-limited (60 req/min per IP)

Request:
```json
{
  "events": [
    {
      "event_type": "page_view",
      "timestamp": "2026-02-23T14:30:00Z",
      "session_id": "550e8400-e29b-41d4-a716-446655440000",
      "visitor_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "device_type": "mobile",
      "page_path": "/p/silk-bedsheet-001",
      "properties": {
        "page_type": "product",
        "product_slug": "silk-bedsheet-001",
        "referrer": "https://google.com",
        "utm_source": "google",
        "utm_medium": "cpc"
      }
    }
  ]
}
```

Response: `202 Accepted`
```json
{ "success": true, "data": { "accepted": 1 } }
```

Validation:
- Max 25 events per batch
- Required: event_type, timestamp, session_id, visitor_id
- Events older than 24 hours silently dropped

### Analytics Dashboard Endpoints

**`GET /admin/analytics/dashboard`** — live counters
```json
{
  "today": {
    "orders": 12, "revenue": 3499700, "visitors": 287,
    "new_customers": 4, "conversion_rate": 4.18,
    "avg_order_value": 291641, "add_to_cart_count": 34,
    "payment_failures": 1
  },
  "trends": { "revenue_growth": 12.5, "orders_growth": 8.3, "customers_growth": 15.0 },
  "inventory": { "low_stock_count": 3, "out_of_stock_count": 1 }
}
```

**`GET /admin/analytics/funnel?start_date=X&end_date=Y`** — purchase funnel (new)
```json
{
  "period": { "start": "2026-02-16", "end": "2026-02-23" },
  "steps": [
    { "name": "product_views", "count": 2340 },
    { "name": "add_to_cart", "count": 312, "rate": 13.3 },
    { "name": "checkout_started", "count": 98, "rate": 31.4 },
    { "name": "payment_started", "count": 87, "rate": 88.8 },
    { "name": "payment_success", "count": 79, "rate": 90.8 }
  ],
  "overall_conversion": 3.38,
  "checkout_drop_off": { "address": 8, "shipping": 2, "payment": 9 }
}
```

**`GET /admin/analytics/sales?period=daily&start_date=X&end_date=Y`** — revenue
```json
{
  "period": "daily",
  "total_revenue": 24497900, "total_orders": 84, "avg_order_value": 291641,
  "data_points": [
    { "date": "2026-02-16", "revenue": 3200000, "orders": 11 }
  ],
  "by_category": [
    { "category_id": "cat_003", "name": "Bedsheets", "revenue": 12400000, "orders": 42 }
  ],
  "by_region": [
    { "state": "Maharashtra", "revenue": 7200000, "orders": 24,
      "city_breakdown": [{ "city": "Mumbai", "revenue": 4800000, "orders": 16 }] }
  ]
}
```

**`GET /admin/analytics/customers?start_date=X&end_date=Y`** — customer insights
```json
{
  "new_customers": 28, "returning_customers": 56,
  "repeat_purchase_rate": 32.1, "avg_lifetime_value": 584300,
  "avg_time_between_purchases_days": 18.5,
  "registration_to_purchase_days": 2.3,
  "otp_completion_rate": 91.2,
  "acquisition_channels": [
    { "source": "google", "medium": "cpc", "customers": 12 }
  ]
}
```

**`GET /admin/analytics/engagement?start_date=X&end_date=Y`** — behavioral (new)
```json
{
  "total_sessions": 1842, "avg_session_duration_seconds": 245, "bounce_rate": 38.2,
  "device_breakdown": { "mobile": 68.4, "desktop": 28.1, "tablet": 3.5 },
  "top_pages": [{ "path": "/", "views": 1842, "avg_time_seconds": 12 }],
  "top_exit_pages": [{ "path": "/cart", "exits": 124 }],
  "scroll_depth_avg": { "product": 72, "category": 58, "home": 45 },
  "image_interaction_rate": 64.2,
  "filter_usage": [{ "type": "price", "count": 312 }],
  "errors": [{ "type": "404", "count": 12 }]
}
```

**`GET /admin/analytics/products?start_date=X&end_date=Y&limit=10`** — product performance
```json
{
  "top_viewed": [
    { "product_id": "prod_001", "name": "Silk Bedsheet", "views": 234,
      "add_to_carts": 31, "purchases": 12, "view_to_purchase": 5.1 }
  ],
  "top_selling": [
    { "product_id": "prod_001", "name": "Silk Bedsheet", "units_sold": 12, "revenue": 2998800 }
  ],
  "top_shared": [
    { "product_id": "prod_001", "name": "Silk Bedsheet", "shares": 8 }
  ]
}
```

---

## Required Backend Changes

### 1. Add category_id to OrderItem

Add `CategoryID` and `CategoryName` fields to the `OrderItem` struct in `internal/domain/order.go`. Populate them from the product at order creation time in `CheckoutService.Initiate()`.

### 2. New DynamoDB table

Create `handloom-events-{env}` with PK (String), SK (String), TTL enabled on `ttl` attribute. PAY_PER_REQUEST billing. No GSIs.

### 3. New store Lambda endpoint

Add `POST /api/v1/store/events` handler for frontend event ingestion. Accepts batched events, validates, writes to raw events table, updates live counters.

### 4. New admin analytics endpoints

Add `/admin/analytics/funnel` and `/admin/analytics/engagement` to the existing analytics handler.

### 5. Enhance worker-analytics Lambda

- Process both SQS events (from SNS) and frontend events (from store Lambda writes)
- Update DASHBOARD#CURRENT live counters on each event
- Add scheduled mode triggered by EventBridge for daily aggregation

### 6. EventBridge scheduled rule

Daily cron at 00:30 UTC triggers worker-analytics in aggregation mode. Monthly cron on the 1st at 02:00 UTC for monthly rollups.

---

## Cost Estimate

At 300 daily visitors, ~20 orders/day:

| Component | Monthly Volume | Cost (ap-south-1) |
|-----------|---------------|-------------------|
| DynamoDB writes (events) | ~165K WRUs | ~$0.25 |
| DynamoDB reads (dashboard) | ~5K RRUs | ~$0.002 |
| DynamoDB storage (30-day TTL) | ~80 MB | ~$0.02 |
| Lambda (analytics worker) | ~165K invocations | Free tier |
| SNS/SQS | ~165K messages | Free tier |
| API Gateway (event calls) | ~15K (batched) | ~$0.02 |

**Total: ~$0.30/month.** At 10x scale (3,000 visitors/day): ~$3-5/month.

Cost kept minimal by: batching frontend events (10 per request), sendBeacon for reliable delivery, 30-day TTL on raw events, atomic counter updates instead of read-modify-write.
