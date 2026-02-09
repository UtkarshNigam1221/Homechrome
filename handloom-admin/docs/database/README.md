# Handloom Admin Database Design

## Overview

The Handloom Admin system uses **Amazon DynamoDB** as its primary database, implementing a **multi-table design** with single-table patterns within each table. This architecture provides:

- **Scalability**: Automatic scaling with DynamoDB's on-demand capacity
- **Performance**: Single-digit millisecond latency for all operations
- **Flexibility**: Schema-less design for evolving business requirements
- **Cost Efficiency**: Pay-per-request pricing model

## Table Architecture

The system uses **4 distinct DynamoDB tables**:

| Table | Purpose | Key Entities |
|-------|---------|--------------|
| [handloom-core](./handloom-core.md) | Core business data | Users, Categories, Products, Designs, Inventory, Pricing, Artisans, Coupons |
| [handloom-orders](./handloom-orders.md) | Order management | Orders, Customers, Price Quotes |
| [handloom-audit](./handloom-audit.md) | Audit logging | Audit Logs (with TTL) |
| [handloom-analytics](./handloom-analytics.md) | Analytics data | Dashboard Stats, Sales Analytics |

## Configuration

### Environment Variables

```bash
# Table Names
DYNAMODB_CORE_TABLE=handloom-core
DYNAMODB_ORDERS_TABLE=handloom-orders
DYNAMODB_AUDIT_TABLE=handloom-audit
DYNAMODB_ANALYTICS_TABLE=handloom-analytics

# AWS Configuration
AWS_REGION=ap-south-1
AWS_ACCESS_KEY_ID=<your-access-key>
AWS_SECRET_ACCESS_KEY=<your-secret-key>

# Local Development (DynamoDB Local)
AWS_ENDPOINT=http://localhost:8000
```

## Key Design Patterns

### 1. Composite Primary Keys

All tables use composite primary keys (PK + SK) enabling:
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
| GSI1 | Entity relationships | Products by Category |
| GSI2 | Cross-entity queries | All Products listing (PRODUCT#ALL) |

### 3. Key Prefix Conventions

```
USER#<uuid>              → User entities
CATEGORY#<uuid>          → Category entities
DESIGN#<uuid>            → Design entities
PRODUCT#<uuid>           → Product entities
SKU#<sku>                → Product SKU uniqueness items
INVENTORY#<product_id>   → Inventory records
PRICING_RULE#<uuid>      → Pricing rules
COUPON#<uuid>            → Coupon entities
ARTISAN#<uuid>           → Artisan entities
ORDER#<uuid>             → Order entities
CUSTOMER#<uuid>          → Customer entities
QUOTE#<uuid>             → Price quotes
AUDIT#<date>             → Audit log partitions
```

### 4. Sort Key Patterns

```
METADATA                      → Main entity record
TXN#<timestamp>#<id>          → Transactions (time-ordered)
STATUS#<timestamp>            → Status history
USAGE#<timestamp>#<order_id>  → Usage tracking
PAYOUT#<timestamp>            → Payout records
```

### 5. Denormalization Strategy

Common statistics are denormalized for read performance:

| Entity | Denormalized Fields |
|--------|---------------------|
| Product | Quantity, ReservedQty, AvailableQty |
| Category | ProductCount, DesignCount |
| Customer | TotalOrders, TotalSpent |
| Artisan | ProductCount, TotalSales, TotalEarnings |

### 6. TTL (Time-To-Live)

Automatic data expiration for:
- **Audit Logs**: 90 days retention
- **Price Quotes**: Configurable validity (default 24 hours)

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

| Entity | Primary Access | Secondary Access |
|--------|---------------|------------------|
| User | By ID | By Email (GSI1) |
| Category | By ID | By Parent, By Slug |
| Design | By ID | By Category (GSI1), By Slug |
| Product | By ID | By Category (GSI1), All Products (GSI2), By SKU (SKU# item) |
| Order | By ID | By Customer (GSI1), By Date |
| Customer | By ID | By Email (GSI1) |
| Audit | By Date | By User (GSI1) |

### Common Query Patterns

1. **List products in category**: Query GSI1 with `CATEGORY#<id>`, SK begins_with `PRODUCT#` (cursor pagination)
2. **List all products**: Query GSI2 with `PRODUCT#ALL` (cursor pagination)
3. **Get product by SKU**: GetItem PK = `SKU#<sku>`, SK = `METADATA` → returns `product_id`
4. **List all categories**: Query GSI1 with `CATEGORY#ALL` (cursor pagination)
5. **Get user by email**: Query GSI1 with `USER_EMAIL` + email
6. **Order history for customer**: Query GSI1 with `CUSTOMER#<id>`
7. **Daily audit logs**: Query PK `AUDIT#<date>`
8. **Inventory transactions**: Query PK `INVENTORY#<product_id>` with SK prefix `TXN#`

## Local Development

### Using DynamoDB Local

```bash
# Start DynamoDB Local with Docker
docker run -p 8000:8000 amazon/dynamodb-local

# Set environment variable
export AWS_ENDPOINT=http://localhost:8000
```

### Creating Tables Locally

See [scripts/create-tables.sh](../../scripts/create-tables.sh) for table creation scripts.

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

- [handloom-core.md](./handloom-core.md) - Core table schema details
- [handloom-orders.md](./handloom-orders.md) - Orders table schema details
- [handloom-audit.md](./handloom-audit.md) - Audit table schema details
- [handloom-analytics.md](./handloom-analytics.md) - Analytics table schema details
