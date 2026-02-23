# handloom-analytics Table

The analytics table stores pre-aggregated metrics for the admin dashboard. All data is computed from raw events and written as daily aggregates across five categories plus a live-counter singleton.

## Table Configuration

```
Table Name: handloom-analytics
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
```

**Env var:** `DYNAMODB_ANALYTICS_TABLE=handloom-analytics`

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |

---

## Entities

### 1. Dashboard Live Counters

A singleton item that holds real-time counters for the admin dashboard. Updated in real-time via DynamoDB atomic `ADD` operations (no read-modify-write races). Reset daily by the aggregation cron after archiving to a historical snapshot.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `DASHBOARD#CURRENT` | `DASHBOARD#CURRENT` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| today_orders | Number | Orders placed today |
| today_revenue | Number | Revenue today (paise) |
| today_page_views | Number | Storefront page views today |
| today_product_views | Number | Product detail page views today |
| today_add_to_carts | Number | Add-to-cart events today |
| today_payments_success | Number | Successful payments today |
| today_payments_failed | Number | Failed payments today |
| today_new_customers | Number | New customer registrations today |
| week_orders | Number | Orders this week |
| week_revenue | Number | Revenue this week (paise) |
| month_orders | Number | Orders this month |
| month_revenue | Number | Revenue this month (paise) |
| total_orders | Number | All-time order count |
| total_revenue | Number | All-time revenue (paise) |
| total_customers | Number | All-time customer count |
| total_products | Number | Current product count |
| revenue_growth | Number | Revenue growth percentage |
| orders_growth | Number | Orders growth percentage |
| customers_growth | Number | Customers growth percentage |
| average_order_value | Number | Current AOV (paise) |
| low_stock_count | Number | Products below stock threshold |
| out_of_stock_count | Number | Products with zero stock |
| pending_orders | Number | Orders in PENDING status |
| processing_orders | Number | Orders in PROCESSING status |
| shipped_orders | Number | Orders in SHIPPED status |

All fields are atomic counters incremented via `UpdateItem` with `ADD` expressions.

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get current live counters | PK = `DASHBOARD#CURRENT`, SK = `METADATA` |

---

### 2. Dashboard Historical Snapshots

Daily snapshots of the live counters, archived before the daily reset. One item per day, same attribute set as the live counters.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `DASHBOARD#STATS#<date>` | `DASHBOARD#STATS#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

Same attributes as **Dashboard Live Counters** above, capturing the end-of-day values.

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get historical snapshot for a date | PK = `DASHBOARD#STATS#2026-02-22`, SK = `METADATA` |

---

### 3. Funnel Aggregates

Daily conversion funnel metrics tracking the customer journey from page view to order.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `FUNNEL#DAILY#<date>` | `FUNNEL#DAILY#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| date | String | Date (YYYY-MM-DD) |
| page_views | Number | Total storefront page views |
| product_views | Number | Product detail page views |
| add_to_carts | Number | Add-to-cart events |
| checkouts_started | Number | Checkout initiations |
| orders_created | Number | Completed orders |
| view_to_cart_rate | Number | product_views to add_to_carts conversion rate |
| cart_to_checkout_rate | Number | add_to_carts to checkouts_started conversion rate |
| checkout_to_order_rate | Number | checkouts_started to orders_created conversion rate |
| overall_rate | Number | End-to-end conversion (page_views to orders_created) |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get funnel for a date | PK = `FUNNEL#DAILY#2026-02-22`, SK = `METADATA` |

---

### 4. Revenue Aggregates

Daily revenue and order breakdown with category-level and product-level drill-down.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `REVENUE#DAILY#<date>` | `REVENUE#DAILY#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| date | String | Date (YYYY-MM-DD) |
| total_revenue | Number | Total revenue (paise) |
| total_orders | Number | Total orders |
| average_order_value | Number | AOV (paise) |
| revenue_by_category | Map | Map of category_id to revenue (paise) |
| top_by_views | List[Object] | Top products by views: `{product_id, count}` |

#### revenue_by_category Structure

```json
{
  "cat-001": 1500000,
  "cat-002": 850000
}
```

#### top_by_views Structure

```json
[
  { "product_id": "prod-001", "count": 245 },
  { "product_id": "prod-002", "count": 189 }
]
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get revenue for a date | PK = `REVENUE#DAILY#2026-02-22`, SK = `METADATA` |

---

### 5. Customer Aggregates

Daily customer traffic and registration metrics with device breakdowns.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMERS#DAILY#<date>` | `CUSTOMERS#DAILY#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| date | String | Date (YYYY-MM-DD) |
| unique_visitors | Number | Unique visitors for the day |
| new_registrations | Number | New customer sign-ups |
| returning_visitors | Number | Returning visitors |
| by_device_type | Map | Map of device type to visitor count |

#### by_device_type Structure

```json
{
  "mobile": 1250,
  "desktop": 430,
  "tablet": 85
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get customer data for a date | PK = `CUSTOMERS#DAILY#2026-02-22`, SK = `METADATA` |

---

### 6. Engagement Aggregates

Daily session and engagement metrics including bounce rate and page popularity.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ENGAGEMENT#DAILY#<date>` | `ENGAGEMENT#DAILY#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| date | String | Date (YYYY-MM-DD) |
| total_sessions | Number | Total sessions for the day |
| bounce_count | Number | Sessions with single page view |
| bounce_rate | Number | Bounce rate percentage |
| avg_session_duration | Number | Average session duration (seconds) |
| top_pages | List[Object] | Most visited pages: `{path, views}` |
| avg_scroll_depth | Number | Average scroll depth percentage |

#### top_pages Structure

```json
[
  { "path": "/", "views": 3200 },
  { "path": "/c/silk-sarees", "views": 1850 },
  { "path": "/p/red-banarasi-saree", "views": 920 }
]
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get engagement for a date | PK = `ENGAGEMENT#DAILY#2026-02-22`, SK = `METADATA` |

---

### 7. Product Aggregates

Daily product discovery metrics tracking which products are being viewed.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PRODUCTS#DAILY#<date>` | `PRODUCTS#DAILY#2026-02-22` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| date | String | Date (YYYY-MM-DD) |
| total_product_views | Number | Total product page views |
| unique_products_viewed | Number | Distinct products viewed |
| top_by_views | List[Object] | Top products by views: `{product_id, count}` |

#### top_by_views Structure

```json
[
  { "product_id": "prod-001", "count": 245 },
  { "product_id": "prod-002", "count": 189 }
]
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get product views for a date | PK = `PRODUCTS#DAILY#2026-02-22`, SK = `METADATA` |

---

## Access Patterns Summary

| Pattern | Key Condition |
|---------|---------------|
| Get current live counters | PK = `DASHBOARD#CURRENT`, SK = `METADATA` |
| Get historical snapshot | PK = `DASHBOARD#STATS#<date>`, SK = `METADATA` |
| Get funnel for a date | PK = `FUNNEL#DAILY#<date>`, SK = `METADATA` |
| Get revenue for a date | PK = `REVENUE#DAILY#<date>`, SK = `METADATA` |
| Get customer data for a date | PK = `CUSTOMERS#DAILY#<date>`, SK = `METADATA` |
| Get engagement for a date | PK = `ENGAGEMENT#DAILY#<date>`, SK = `METADATA` |
| Get product views for a date | PK = `PRODUCTS#DAILY#<date>`, SK = `METADATA` |

All reads are single-item `GetItem` calls (PK + SK). No scans or queries required for any access pattern.

---

## Data Aggregation

### Real-Time Updates

Live counters (`DASHBOARD#CURRENT`) are updated in real-time by event-driven workers as events flow through the system. Each event triggers an atomic `ADD` operation on the relevant counter fields -- no read-modify-write cycle, so concurrent updates are safe.

Events that trigger counter updates include:
- `order.created`, `order.confirmed` -- increment order counters and revenue
- `payment.success`, `payment.failed` -- increment payment counters
- `customer.registered` -- increment customer counter
- `product.created` -- increment product counter
- Page view and cart events -- increment storefront interaction counters

### Daily Aggregation

Triggered by an EventBridge schedule rule that invokes the `worker-analytics` Lambda, which calls `AnalyticsAggregator.AggregateDate()`:

1. **Read** raw events from the handloom-events table for the target date
2. **Compute** five aggregate categories: funnel, revenue, customer, engagement, product
3. **Write** each aggregate as a single item to the analytics table
4. **Archive** the current dashboard counters to `DASHBOARD#STATS#<date>`
5. **Reset** the live counters for the next day
