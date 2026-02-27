# Handloom Admin Database Design

## Overview

The Handloom Admin system uses a **hybrid database architecture**:

- **Amazon DynamoDB** (7 tables) — for core data, orders, sessions, audit, analytics, notifications, and events
- **PostgreSQL** (RDS) — for catalog data (categories, products, inventory) requiring relational queries and full-text search

## Table Architecture

### DynamoDB Tables (7)

| Table | Purpose | Key Entities |
|-------|---------|--------------|
| [handloom-core](./handloom-core.md) | Core business data | Users, Pricing Rules, Coupons |
| [handloom-orders](./handloom-orders.md) | Order management | Orders, Customers, Payments, Shipments, Carts, Price Quotes |
| [handloom-sessions](./handloom-sessions.md) | Auth sessions | Admin Tokens, Customer Tokens, OTP, Password Reset (all TTL) |
| [handloom-notifications](./handloom-notifications.md) | Notifications | Notifications, Templates |
| [handloom-audit](./handloom-audit.md) | Audit logging | Audit Logs (with TTL) |
| [handloom-analytics](./handloom-analytics.md) | Analytics data | Dashboard Counters, Daily Aggregates (Funnel, Revenue, Customers, Engagement, Products) |
| [handloom-events](./handloom-events.md) | Raw tracking events | Frontend tracking events from storefront (30-day TTL) |

### PostgreSQL (Catalog)

| Documentation | Purpose | Key Tables |
|---------------|---------|------------|
| [Catalog Schema](./handloom-catalog.md) | Product catalog | categories, products, inventory, product_images, product_attribute_values, inventory_transactions |

## Configuration

### Environment Variables

```bash
# DynamoDB Table Names
DYNAMODB_CORE_TABLE=handloom-core
DYNAMODB_ORDERS_TABLE=handloom-orders
DYNAMODB_SESSIONS_TABLE=handloom-sessions
DYNAMODB_NOTIFICATIONS_TABLE=handloom-notifications
DYNAMODB_AUDIT_TABLE=handloom-audit
DYNAMODB_ANALYTICS_TABLE=handloom-analytics
DYNAMODB_EVENTS_TABLE=handloom-events

# PostgreSQL (Catalog)
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/handloom?sslmode=disable
# Or for production (Lambda): RDS_SECRET_ARN, RDS_ENDPOINT, RDS_PORT, RDS_DATABASE

# AWS Configuration
AWS_REGION=ap-south-1
AWS_ACCESS_KEY_ID=<your-access-key>
AWS_SECRET_ACCESS_KEY=<your-secret-key>

# Local Development (LocalStack)
AWS_ENDPOINT=http://localhost:4566
```

## DynamoDB Design Patterns

### 1. Composite Primary Keys

All DynamoDB tables use composite primary keys (PK + SK) enabling:
- Hierarchical data organization
- Efficient range queries
- Related data co-location

```
PK: ENTITY_TYPE#<id>
SK: METADATA | SUB_TYPE#<timestamp>#<id>
```

### 2. Global Secondary Indexes (GSIs)

Strategic GSI usage for common access patterns:

| GSI | Purpose | Example |
|-----|---------|---------|
| GSI1 | Entity relationships | Orders by Customer |
| GSI2 | Cross-entity queries | All Orders listing (ORDER#ALL) |

### 3. Key Prefix Conventions

```
# Core table
USER#<uuid>              → User entities
USER_EMAIL#<email>       → Email uniqueness guard
PRICING_RULE#<uuid>      → Pricing rules
COUPON#<uuid>            → Coupon entities

# Catalog data is now in PostgreSQL (see handloom-catalog.md)

# Orders table
ORDER#<uuid>             → Order entities
ORDER_NUMBER#<number>    → Order number uniqueness index
CUSTOMER#<uuid>          → Customer entities
CUSTOMER_PHONE#<phone>   → Phone uniqueness index
PAYMENT#<uuid>           → Payment entities
CART#<customer_id>       → Cart header + items
QUOTE#<uuid>             → Price quotes

# Sessions table
USER#<uuid>              → Admin refresh tokens
CUST_TOKEN#<customer_id> → Customer refresh tokens
OTP#<phone>              → OTP records
PASSWORD_RESET#<hash>    → Password reset tokens

# Audit table
AUDIT#<date>             → Audit log partitions

# Analytics table
DASHBOARD#CURRENT        → Live dashboard counters (atomic ADD)
DASHBOARD#STATS#<date>   → Archived daily dashboard snapshots
FUNNEL#DAILY#<date>      → Conversion funnel aggregates
REVENUE#DAILY#<date>     → Revenue aggregates
CUSTOMERS#DAILY#<date>   → Customer aggregates
ENGAGEMENT#DAILY#<date>  → Engagement aggregates
PRODUCTS#DAILY#<date>    → Product view aggregates

# Events table
EVENT#<date>             → Raw tracking events (30-day TTL)
```

### 4. Sort Key Patterns

```
METADATA                      → Main entity record
TXN#<timestamp>#<id>          → Inventory transactions (time-ordered)
ITEM#<product_id>             → Cart line items
SHIPMENT#<shipment_id>        → Shipments under an order
ATTR#<name>#<value>           → Product attribute indexes
STATUS#<timestamp>            → Status history
USAGE#<timestamp>#<order_id>  → Coupon usage tracking
REFRESH_TOKEN#<hash>          → Session refresh tokens
UNIQUENESS                    → Uniqueness guard items
```

### 5. Denormalization Strategy

Common statistics are denormalized for read performance:

| Entity | Denormalized Fields |
|--------|---------------------|
| Customer | TotalOrders, TotalSpent |

### 6. Uniqueness Enforcement

DynamoDB has no unique constraints. Uniqueness is enforced via **guard items** written atomically with the main entity using `TransactWriteItems` + `attribute_not_exists(PK)`:

| Entity | Guard Item PK | Purpose |
|--------|--------------|---------|
| User | `USER_EMAIL#<email>` | Email uniqueness |
| Order | `ORDER_NUMBER#<number>` | Order number uniqueness |
| Customer | `CUSTOMER_PHONE#<phone>` | Phone uniqueness |

> **Note:** Product SKU and category slug uniqueness are enforced by PostgreSQL `UNIQUE` constraints (no guard items needed).

### 7. TTL (Time-To-Live)

TTL is enabled on 6 tables. Automatic data expiration for:
- **Sessions**: OTP (5 min), refresh tokens (7 days), password reset (1 hour)
- **Cart items**: 30 days from last update
- **Price Quotes**: 24 hours
- **Audit Logs**: 90 days retention
- **Events**: 30 days from event timestamp

## PostgreSQL Design Patterns

Catalog data uses different patterns from DynamoDB. See [handloom-catalog.md](./handloom-catalog.md) for full schema.

### 1. Relational Schema with Foreign Keys

Normalized tables with `REFERENCES ... ON DELETE CASCADE`:
- `products.category_id → categories.id`
- `product_images.product_id → products.id`
- `inventory.product_id → products.id`

Deleting a category cascades to its attributes/options. Deleting a product cascades to its images, attribute values, and inventory.

### 2. Connection Pooling

Uses `pgxpool` (jackc/pgx v5) for connection pooling. Local dev uses a direct DSN; Lambda resolves credentials from AWS Secrets Manager and builds the DSN at startup.

### 3. Row-Level Locking for Inventory

All inventory mutations use `SELECT ... FOR UPDATE` within a transaction to prevent race conditions on concurrent stock changes. Each mutation records an `inventory_transaction` atomically.

### 4. Full-Text Search

Products have a `search_vector` generated column combining `name` (weight A) and `description` (weight B) with a GIN index. Queries use `websearch_to_tsquery('english', ...)` for relevance-ranked results via `ts_rank()`, with an `ILIKE` fallback for partial/substring matches. The trigram index on `name` is kept for the ILIKE path.

### 5. Dynamic Attribute Filtering

Products have flexible attributes stored in `product_attribute_values` (EAV pattern). Filtering uses `EXISTS` subqueries per attribute, allowing any combination of attribute filters without pre-defined indexes.

### 6. In-Process Caching

TTL-based in-process cache (`go-cache`) for read-heavy data:
- Categories: 2-5 min TTL
- Product detail: 2 min TTL
- Attribute filter options: 5 min TTL

Invalidated on writes. Product list queries are NOT cached (too many filter combinations).

### 7. Cursor-Based Pagination (Offset-Encoded)

PostgreSQL repos use base64-encoded integer offsets as cursors (vs DynamoDB's ExclusiveStartKey). Fetch `LIMIT+1` rows to detect HasMore, trim before returning.

---

## Data Types & Conventions

### Currency
All monetary values stored in **paise** (1/100 of INR):
- `1000` = ₹10.00
- `150000` = ₹1,500.00

### Timestamps
All timestamps in **ISO 8601 format** (UTC):
- `2024-01-15T10:30:00Z`

### IDs
UUIDs in **standard format**:
- `550e8400-e29b-41d4-a716-446655440000`

### Status Enums

| Entity | Statuses |
|--------|----------|
| User | ACTIVE, INACTIVE, PENDING |
| Product | ACTIVE, INACTIVE, DRAFT |
| Order | PENDING, CONFIRMED, PROCESSING, SHIPPED, DELIVERED, CANCELLED, RETURNED |
| Payment | PENDING, PAID, FAILED, REFUNDED |

## Access Patterns Summary

### By Entity

| Entity | Table | Primary Access | Secondary Access |
|--------|-------|---------------|------------------|
| User | core | By ID | By Email (GSI1) |
| Pricing Rule | core | By ID | By Scope (GSI1), All Rules (GSI2) |
| Coupon | core | By ID | By Code (GSI1) |
| Category | PostgreSQL | By ID, By Slug | All Categories (SQL) |
| Product | PostgreSQL | By ID, By SKU, By Slug | By Category, Full-text search, Attribute filtering |
| Inventory | PostgreSQL | By Product ID | Low Stock (WHERE clause) |
| Order | orders | By ID | By Customer (GSI1), All Orders (GSI2) |
| Customer | orders | By ID | By Email (GSI1), By Phone (phone index item), All Customers (GSI2) |
| Payment | orders | By ID | By Order (GSI1), By Merchant Txn ID (GSI2) |
| Shipment | orders | By Order+Shipment | — |
| Cart | orders | By Customer ID | — |
| Quote | orders | By ID | — (TTL auto-delete) |
| Admin Token | sessions | By User ID | — (TTL auto-delete) |
| Customer Token | sessions | By Customer ID | — (TTL auto-delete) |
| OTP | sessions | By Phone | — (TTL auto-delete) |
| Audit Log | audit | By Date | By Entity (GSI1), By User (GSI2) |
| Dashboard Counters | analytics | By PK (`DASHBOARD#CURRENT`) | — |
| Daily Aggregates | analytics | By PK (`{PREFIX}#DAILY#{date}`) | — |
| Raw Events | events | By Date (`EVENT#{date}`) | — (30-day TTL) |

### Common Query Patterns

1. **List products in category**: SQL query on products table with category_id filter (offset pagination)
2. **List all products**: SQL query on products table (offset pagination)
3. **Get product by SKU**: SQL query `WHERE sku = ?`
4. **List all categories**: SQL query on categories table
5. **Get user by email**: Query core GSI1 with `USER_EMAIL` + email
6. **Order history for customer**: Query orders GSI1 with `CUSTOMER#<id>`
7. **All orders listing**: Query orders GSI2 with `ORDER#ALL`
8. **Customer by phone**: GetItem PK = `CUSTOMER_PHONE#<phone>`, SK = `METADATA`
9. **Payments for order**: Query orders GSI1 with `ORDER#<orderId>`
10. **Low stock products**: SQL query `WHERE available_qty <= low_stock_threshold`
11. **Daily audit logs**: Query PK `AUDIT#<date>`
12. **Inventory transactions**: SQL query on inventory_transactions table with product_id filter
13. **Customer cart**: Query PK `CART#<customerId>` with SK prefix `ITEM#`
14. **Dashboard live counters**: GetItem PK `DASHBOARD#CURRENT`, SK `METADATA`
15. **Daily funnel aggregate**: GetItem PK `FUNNEL#DAILY#<date>`, SK `METADATA`
16. **All events for a date**: Query PK `EVENT#<date>` (paginated)

## Local Development

### Using LocalStack

```bash
# Start LocalStack (DynamoDB, S3, Lambda, API Gateway, IAM)
make docker-up

# Create tables and seed data
make setup-local

# Set environment variable
export AWS_ENDPOINT=http://localhost:4566
```

### Creating Tables Locally

See [scripts/init-local-db.sh](../../scripts/init-local-db.sh) for table creation scripts.

## Monitoring & Operations

### Key Metrics to Monitor
- ConsumedReadCapacityUnits
- ConsumedWriteCapacityUnits
- ThrottledRequests
- SystemErrors

### Backup Strategy
- Point-in-time recovery enabled
- Daily snapshots to S3
- Cross-region replication for DR

## Further Reading

- [handloom-core.md](./handloom-core.md) - Core table (DynamoDB): Users, Pricing Rules, Coupons
- [handloom-catalog.md](./handloom-catalog.md) - Catalog schema (PostgreSQL): Categories, Products, Inventory
- [handloom-orders.md](./handloom-orders.md) - Orders table (DynamoDB): Orders, Customers, Payments, Shipments, Carts
- [handloom-sessions.md](./handloom-sessions.md) - Sessions table (DynamoDB): Tokens, OTP, Password Reset
- [handloom-notifications.md](./handloom-notifications.md) - Notifications table (DynamoDB)
- [handloom-audit.md](./handloom-audit.md) - Audit table (DynamoDB): Audit Logs
- [handloom-analytics.md](./handloom-analytics.md) - Analytics table (DynamoDB): Dashboard Counters, Daily Aggregates
- [handloom-events.md](./handloom-events.md) - Events table (DynamoDB): Raw Tracking Events (30-day TTL)
