# B2C Storefront Design — Homechrome

**Date:** 2026-02-20
**Status:** Approved
**Scope:** Lean MVP — catalog browsing, cart, checkout with PhonePe, order tracking, OTP-based customer auth

## 1. Overview

Homechrome is currently an admin-only panel for handloom e-commerce. This design adds a customer-facing B2C storefront — a separate Next.js application backed by new public API endpoints on the existing Go backend.

### MVP Scope

**In scope:** customer auth (phone OTP), catalog browsing with filters, server-side cart, checkout with PhonePe payments, Shiprocket shipping, order tracking, SMS/email notifications.

**Out of scope:** wishlists, reviews/ratings, artisan profile pages, coupon application at checkout, social login, product recommendations, order returns self-service, WhatsApp sharing, loyalty program.

### Tech Stack

| Component | Technology |
|-----------|------------|
| Storefront | Next.js 15, App Router, TypeScript, Tailwind CSS |
| Backend | Existing Go monolith — new `/api/v1/store/*` routes |
| Database | Existing DynamoDB tables — new entities + fixed key patterns |
| Payments | PhonePe Payment Gateway |
| Shipping | Shiprocket API |
| SMS/OTP | MSG91 |
| Deployment | Vercel (storefront), AWS Lambda (backend) |

## 2. High-Level Architecture

```
                         CloudFront
    homechrome.lldlab.com (storefront)
    admin.homechrome.lldlab.com (admin SPA)
              |                    |
         Next.js               React SPA
          (SSR)                (S3, existing)
         Vercel
              |                    |
              |  /api/v1/store/*   |  /admin/*
              +--------+-----------+
                       |
                 API Gateway
                  (Lambda)
                       |
            +----------+----------+
            |          |          |
         Store      Admin      Shared
        Handler    Handler    Services
         Layer      Layer    (Product,
            |          |     Inventory,
            +----------+    Order,Price)
                       |
            +----------+----------+
            |          |          |
         DynamoDB     S3       External
        (4 tables)  (assets)   APIs
                                 |
                    +------------+------------+
                    |            |            |
                 PhonePe    Shiprocket     MSG91
```

### Request Flow Layers

**Public (no auth):** catalog browsing, pricing calculator, pincode serviceability, public order tracking.

**Customer auth (JWT):** cart, checkout, order history, profile management.

**Webhook (signature verification):** PhonePe payment callbacks, Shiprocket shipment status updates.

**Admin (unchanged):** all `/admin/*` routes with admin JWT.

### Auth Architecture

Two JWT namespaces sharing the same infrastructure:

| | Admin | Customer |
|--|-------|----------|
| Login method | Email + password | Phone + OTP |
| Token prefix in DynamoDB | `USER#<id>` / `REFRESH_TOKEN#` | `CUST_TOKEN#<id>` / `REFRESH_TOKEN#` |
| JWT claims | UserID, Email, Role, Permissions | CustomerID, Phone, Email |
| Access token TTL | 15 min | 1 hour |
| Refresh token TTL | 7 days | 30 days |
| Cookie name | `access_token` | `store_token` |

## 3. Database Schema

### 3.1 New Entities — `handloom-orders` Table

#### Cart Header

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CART#<customer_id\|session_id>` | `CART#cust-001` or `CART#sess-abc123` |
| SK | `METADATA` | `METADATA` |
| TTL | `ttl` | Unix timestamp (30 days from last update) |

Fields: `customer_id` (nullable for guests), `session_id`, `item_count`, `subtotal` (paise), `currency`, `updated_at`.

#### Cart Item (co-located under cart PK)

| Key | Pattern |
|-----|---------|
| PK | `CART#<customer_id\|session_id>` |
| SK | `ITEM#<product_id>` |
| TTL | `ttl` (same 30-day expiry) |

Fields: `product_id`, `product_name`, `product_sku`, `product_image`, `quantity`, `unit_price` (paise, snapshot at add time), `total_price` (paise), `is_custom_size`, `dimensions` ({length, width, unit}, nullable), `quote_id` (nullable), `attributes` (map), `added_at`.

#### Payment

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PAYMENT#<id>` | `PAYMENT#pay-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `ORDER#<order_id>` | `ORDER#ord-001` |
| GSI1SK | `PAYMENT#<created_at>` | `PAYMENT#2026-02-20T10:30:00Z` |
| GSI2PK | `PAYMENT_TXN` | `PAYMENT_TXN` |
| GSI2SK | `<merchant_transaction_id>` | `MCHT_ord001_1708419000` |

Fields: `id`, `order_id`, `customer_id`, `amount` (paise), `currency`, `status` (INITIATED|SUCCESS|FAILED|REFUNDED), `provider` (PHONEPE), `merchant_transaction_id`, `provider_transaction_id`, `payment_method` (UPI|CARD|NET_BANKING|WALLET), `provider_response` (JSON), `initiated_at`, `completed_at`, `refund_amount` (paise, nullable), `refunded_at`.

#### Shipment (co-located under order PK)

| Key | Pattern |
|-----|---------|
| PK | `ORDER#<order_id>` |
| SK | `SHIPMENT#<shipment_id>` |

Fields: `id`, `order_id`, `provider` (SHIPROCKET), `provider_order_id`, `provider_shipment_id`, `awb_number`, `courier_name`, `status` (CREATED|PICKED_UP|IN_TRANSIT|OUT_FOR_DELIVERY|DELIVERED|RTO), `label_url`, `estimated_delivery`, `weight_grams`, `shipped_at`, `delivered_at`.

#### Order Number Index (lookup item)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ORDER_NUMBER#<order_number>` | `ORDER_NUMBER#HL20260220ABC123` |
| SK | `METADATA` | `METADATA` |

Fields: `order_id`.

#### Customer Phone Index (lookup + uniqueness guard)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMER_PHONE#<phone>` | `CUSTOMER_PHONE#+919876543210` |
| SK | `METADATA` | `METADATA` |

Fields: `customer_id`.

### 3.2 New Entity — `handloom-core` Table

#### OTP

| Key | Pattern |
|-----|---------|
| PK | `OTP#<phone>` |
| SK | `METADATA` |
| TTL | `ttl` (5 minutes) |

Fields: `phone`, `code_hash` (SHA256 of 6-digit OTP), `attempts` (max 3), `created_at`.

#### Customer Refresh Token

| Key | Pattern |
|-----|---------|
| PK | `CUST_TOKEN#<customer_id>` |
| SK | `REFRESH_TOKEN#<token_hash>` |
| TTL | `ttl` (30 days) |

Fields: `customer_id`, `created_at`, `expires_at`.

### 3.3 Fixed Key Patterns — Existing Entities

#### Order — add GSI2 keys + order number index

| Key | Current | New |
|-----|---------|-----|
| GSI2PK | *(not set)* | `ORDER#ALL` |
| GSI2SK | *(not set)* | `<created_at>` |

Write `ORDER_NUMBER#<number>` lookup item on creation (TransactWriteItems).

#### Customer — add GSI2 keys + phone index

| Key | Current | New |
|-----|---------|-----|
| GSI2PK | *(not set)* | `CUSTOMER#ALL` |
| GSI2SK | *(not set)* | `<created_at>` |

Write `CUSTOMER_PHONE#<phone>` lookup item on creation. Add field: `phone_verified` (bool).

#### Pricing Rule — add GSI2 keys

| Key | Current | New |
|-----|---------|-----|
| GSI2PK | *(not set)* | `PRICING_RULE#ALL` |
| GSI2SK | *(not set)* | `PRICING_RULE#<id>` |

#### Inventory — sparse GSI for low stock

| Key | Current | New (only when available_qty <= threshold) |
|-----|---------|-----|
| GSI2PK | *(not set)* | `LOW_STOCK` |
| GSI2SK | *(not set)* | `INVENTORY#<product_id>` |

Remove GSI2 keys when stock is replenished above threshold.

#### AuditLog — fix SetKeys

```
GSI1PK = <EntityType>#<EntityID>    (e.g., "ORDER#ord-001")
GSI1SK = <timestamp>
GSI2PK = USER#<user_id>
GSI2SK = <timestamp>
```

### 3.4 Infrastructure Fixes

| Fix | Table | Description |
|-----|-------|-------------|
| Add GSI2 to CDK | `handloom-orders` | GSI2PK + GSI2SK (exists in local init, missing from CDK) |
| Enable TTL | `handloom-orders` | `ttl` attribute — for PriceQuote and Cart expiry |
| Enable TTL | `handloom-core` | `ttl` attribute — for OTP and refresh token expiry |
| Add GSI2 to CDK | `handloom-audit` | GSI2PK + GSI2SK for user-based audit queries |
| Add GSI1 to CDK | `handloom-analytics` | GSI1PK + GSI1SK (exists in local init, missing from CDK) |

## 4. Access Patterns — Zero Scans

Every DynamoDB operation uses GetItem, Query, PutItem, UpdateItem, DeleteItem, TransactWriteItems, or BatchGetItem. No Scan operations.

### 4.1 Customer Auth (OTP Flow)

| # | Operation | DynamoDB Op | Table | Key Condition |
|---|-----------|-------------|-------|---------------|
| 1 | Store OTP | PutItem | core | PK=`OTP#<phone>`, SK=`METADATA`. TTL=5min. Overwrites existing. |
| 2 | Verify OTP | GetItem | core | PK=`OTP#<phone>`, SK=`METADATA`. Check code_hash, attempts < 3. |
| 3 | Increment OTP attempts | UpdateItem | core | PK=`OTP#<phone>`, SK=`METADATA`. `ADD attempts 1`. |
| 4 | Delete OTP on success | DeleteItem | core | PK=`OTP#<phone>`, SK=`METADATA`. |
| 5 | Lookup phone | GetItem | orders | PK=`CUSTOMER_PHONE#<phone>`, SK=`METADATA`. |
| 6 | Register customer | TransactWriteItems | orders | Put `CUSTOMER#<id>` + Put `CUSTOMER_PHONE#<phone>` + Put `CUSTOMER_EMAIL#<email>` (if provided). All with `attribute_not_exists(PK)`. |
| 7 | Login by phone | GetItem + GetItem | orders | `CUSTOMER_PHONE#<phone>` -> customer_id -> `CUSTOMER#<id>`. |
| 8 | Store refresh token | PutItem | core | PK=`CUST_TOKEN#<customer_id>`, SK=`REFRESH_TOKEN#<hash>`. TTL=30d. |
| 9 | Validate refresh token | GetItem | core | PK=`CUST_TOKEN#<customer_id>`, SK=`REFRESH_TOKEN#<hash>`. |
| 10 | Revoke token (logout) | DeleteItem | core | PK=`CUST_TOKEN#<customer_id>`, SK=`REFRESH_TOKEN#<hash>`. |
| 11 | Revoke all tokens | Query + BatchWrite | core | PK=`CUST_TOKEN#<customer_id>`, SK begins_with `REFRESH_TOKEN#`. |

### 4.2 Customer Profile

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 12 | Get by ID | GetItem | orders | PK=`CUSTOMER#<id>`, SK=`METADATA`. |
| 13 | Get by email | Query | orders GSI1 | GSI1PK=`CUSTOMER_EMAIL`, GSI1SK=`<email>`, Limit=1. |
| 14 | Get by phone | GetItem + GetItem | orders | `CUSTOMER_PHONE#<phone>` -> customer_id -> `CUSTOMER#<id>`. |
| 15 | Update profile | PutItem | orders | PK=`CUSTOMER#<id>`, `attribute_exists(PK)`. Phone change: TransactWriteItems to swap phone guard. |
| 16 | Add address | UpdateItem | orders | PK=`CUSTOMER#<id>`, SK=`METADATA`. `list_append` to addresses. |
| 17 | Update address | UpdateItem | orders | PK=`CUSTOMER#<id>`, SK=`METADATA`. Read-modify-write on addresses. |
| 18 | Remove address | UpdateItem | orders | PK=`CUSTOMER#<id>`, SK=`METADATA`. `REMOVE addresses[i]`. |
| 19 | List all customers (admin) | Query | orders GSI2 | GSI2PK=`CUSTOMER#ALL`, GSI2SK <= `<cursor>`. **Replaces SCAN.** |
| 20 | Soft delete | UpdateItem | orders | PK=`CUSTOMER#<id>`. Set status=INACTIVE. |

### 4.3 Cart

| # | Operation | DynamoDB Op | Table | Key Condition |
|---|-----------|-------------|-------|---------------|
| 21 | Get full cart | Query | orders | PK=`CART#<id>`. Returns header + all items. |
| 22 | Add item | PutItem | orders | PK=`CART#<id>`, SK=`ITEM#<product_id>`. Upsert. |
| 23 | Update cart header | UpdateItem | orders | PK=`CART#<id>`, SK=`METADATA`. |
| 24 | Update item quantity | UpdateItem | orders | PK=`CART#<id>`, SK=`ITEM#<product_id>`. |
| 25 | Remove item | DeleteItem | orders | PK=`CART#<id>`, SK=`ITEM#<product_id>`. |
| 26 | Clear cart | Query + BatchWrite | orders | Query PK=`CART#<id>` -> BatchDeleteItem all. |
| 27 | Merge guest -> customer | Query + BatchWrite + PutItems | orders | Rewrite items from `CART#<sess>` to `CART#<cust>`. |

### 4.4 Catalog Browsing (Store-facing)

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 28 | List active categories | Query | core GSI1 | GSI1PK=`CATEGORY#ALL`. FilterExpression: status=ACTIVE. |
| 29 | Get category by ID | GetItem | core | PK=`CATEGORY#<id>`, SK=`METADATA`. |
| 30 | List products in category | Query | core GSI1 | GSI1PK=`CATEGORY#<cat_id>`, SK begins_with `PRODUCT#`. Filter: status=ACTIVE. Cursor pagination. |
| 31 | List all active products | Query | core GSI2 | GSI2PK=`PRODUCT#ALL`. Filter: status=ACTIVE. Cursor pagination. |
| 32 | Filter by attributes | Query | core GSI1 | GSI1PK=`ATTR#<cat_id>#<attr>`, SK begins_with `<value>#PRODUCT#`. Intersect + BatchGetItem. |
| 33 | Get filter options | GetItem | core | PK=`CATEGORY#<cat_id>`, SK=`ATTR_VALUES`. |
| 34 | Get product detail | GetItem | core | PK=`PRODUCT#<id>`, SK=`METADATA`. Exclude cost_price. |
| 35 | Get product stock | GetItem | core | PK=`INVENTORY#<product_id>`, SK=`METADATA`. Return available_qty only. |
| 36 | Batch get products | BatchGetItem | core | Multiple PK=`PRODUCT#<id>`. |
| 37 | Search products | Query | core GSI2 | GSI2PK=`PRODUCT#ALL`. FilterExpression: contains(name, :q) OR contains(tags, :q). |

### 4.5 Pricing (unchanged)

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 38 | Calculate price | Multiple Queries | core GSI1 | GLOBAL, CATEGORY, PRODUCT, MATERIAL scopes. |
| 39 | Get dimension options | GetItem | core | PK=`CATEGORY#<cat_id>`, SK=`METADATA`. |
| 40 | Store quote | PutItem | orders | PK=`QUOTE#<id>`, SK=`METADATA`. TTL set. |
| 41 | Get quote | GetItem | orders | PK=`QUOTE#<id>`, SK=`METADATA`. |
| 42 | List pricing rules (admin) | Query | core GSI2 | GSI2PK=`PRICING_RULE#ALL`. **Replaces SCAN.** |

### 4.6 Checkout & Order Creation

| # | Operation | DynamoDB Op | Table | Key Condition |
|---|-----------|-------------|-------|---------------|
| 43 | Validate stock | BatchGetItem | core | Multiple PK=`INVENTORY#<product_id>`. |
| 44 | Validate coupon (future) | GetItem + Query | core | PK=`COUPON#<id>` + Query PK=`COUPON#<id>`, SK begins_with `USAGE#`. |
| 45 | Create order (atomic) | TransactWriteItems | orders + core | Put order + Put order_number index + Put status history + Reserve inventory per item. Cross-table transaction. |
| 46 | Initiate payment | PutItem | orders | PK=`PAYMENT#<id>`, SK=`METADATA`. Status=INITIATED. |
| 47 | Clear cart | Query + BatchWrite | orders | PK=`CART#<customer_id>`. |
| 48 | Mark quote used | UpdateItem | orders | PK=`QUOTE#<id>`, SK=`METADATA`. |

### 4.7 Payment Webhook

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 49 | Find payment by txn ID | Query | orders GSI2 | GSI2PK=`PAYMENT_TXN`, GSI2SK=`<merchant_txn_id>`, Limit=1. |
| 50 | Update payment status | UpdateItem | orders | PK=`PAYMENT#<id>`, SK=`METADATA`. |
| 51 | Confirm order (on success) | UpdateItem | orders | PK=`ORDER#<order_id>`. Set status=CONFIRMED, payment_status=PAID. |
| 52 | Write status history | PutItem | orders | PK=`ORDER#<order_id>`, SK=`STATUS#<timestamp>`. |
| 53 | Release inventory (on failure) | UpdateItem per item | core | PK=`INVENTORY#<product_id>`. Release reserved_qty. |
| 54 | Mark payment failed | UpdateItem | orders | PK=`ORDER#<order_id>`. Set payment_status=FAILED. |

### 4.8 Order Lifecycle

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 55 | Get order by ID | GetItem | orders | PK=`ORDER#<order_id>`, SK=`METADATA`. |
| 56 | Get order by number | GetItem + GetItem | orders | `ORDER_NUMBER#<number>` -> order_id -> `ORDER#<id>`. **Replaces broken GSI query.** |
| 57 | Get status history | Query | orders | PK=`ORDER#<order_id>`, SK begins_with `STATUS#`. |
| 58 | Get shipment | Query | orders | PK=`ORDER#<order_id>`, SK begins_with `SHIPMENT#`. |
| 59 | Get payment(s) | Query | orders GSI1 | GSI1PK=`ORDER#<order_id>`, GSI1SK begins_with `PAYMENT#`. |
| 60 | Customer's orders | Query | orders GSI1 | GSI1PK=`CUSTOMER#<customer_id>`, GSI1SK <= cursor. Newest first. |
| 61 | Update status (admin) | UpdateItem | orders | PK=`ORDER#<order_id>`. Validate transition in service. |
| 62 | Write status history | PutItem | orders | PK=`ORDER#<order_id>`, SK=`STATUS#<timestamp>`. |
| 63 | Add note | UpdateItem | orders | PK=`ORDER#<order_id>`. list_append to internal_notes. |
| 64 | Update tracking | UpdateItem | orders | PK=`ORDER#<order_id>`. |
| 65 | Create shipment | PutItem | orders | PK=`ORDER#<order_id>`, SK=`SHIPMENT#<id>`. |
| 66 | Cancel order (atomic) | TransactWriteItems | orders + core | Update order CANCELLED + status history + release inventory. |
| 67 | List all orders (admin) | Query | orders GSI2 | GSI2PK=`ORDER#ALL`, GSI2SK <= cursor. **Replaces SCAN.** |

### 4.9 Shipping

| # | Operation | DynamoDB Op | Table | Key Condition |
|---|-----------|-------------|-------|---------------|
| 68 | Store shipment | PutItem | orders | PK=`ORDER#<order_id>`, SK=`SHIPMENT#<id>`. |
| 69 | Update shipment status | UpdateItem | orders | PK=`ORDER#<order_id>`, SK=`SHIPMENT#<id>`. |

Shiprocket API calls (check serviceability, create order, assign AWB, generate label, track) are external HTTP calls, not DynamoDB operations.

### 4.10 Inventory (fixed)

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 70 | Get inventory | GetItem | core | PK=`INVENTORY#<product_id>`, SK=`METADATA`. |
| 71 | Add/adjust stock | UpdateItem | core | PK=`INVENTORY#<product_id>`. Update sparse GSI2 (LOW_STOCK) based on threshold. |
| 72 | Get low stock | Query | core GSI2 | GSI2PK=`LOW_STOCK`. **Replaces SCAN.** Sparse GSI. |
| 73 | Reserve stock | UpdateItem | core | PK=`INVENTORY#<product_id>`. Condition: available_qty >= qty. |
| 74 | Release stock | UpdateItem | core | PK=`INVENTORY#<product_id>`. |
| 75 | Log transaction | PutItem | core | PK=`INVENTORY#<product_id>`, SK=`TXN#<timestamp>#<id>`. |

### 4.11 Audit (fixed)

| # | Operation | DynamoDB Op | Table/Index | Key Condition |
|---|-----------|-------------|-------------|---------------|
| 76 | Create log | PutItem | audit | PK=`AUDIT#<YYYY-MM-DD>`, SK=`<time>#<id>`. |
| 77 | List by date range | Query per day | audit | PK=`AUDIT#<date>`. **Replaces SCAN.** Parallelize across days. |
| 78 | Get by entity | Query | audit GSI1 | GSI1PK=`<EntityType>#<EntityID>`. **Fixed SetKeys.** |
| 79 | Get by user | Query | audit GSI2 | GSI2PK=`USER#<user_id>`. **Fixed: GSI2 added to CDK.** |

### Scan Elimination Summary

| Original Scan | Replaced With | Pattern # |
|---------------|---------------|-----------|
| OrderRepository.List | Query GSI2 `ORDER#ALL` | #67 |
| CustomerRepository.List | Query GSI2 `CUSTOMER#ALL` | #19 |
| CustomerRepository.Search | Query GSI2 `CUSTOMER#ALL` + FilterExpression | #19 |
| PricingRuleRepository.List | Query GSI2 `PRICING_RULE#ALL` | #42 |
| InventoryRepository.GetLowStockProducts | Query GSI2 `LOW_STOCK` (sparse) | #72 |
| AuditRepository.List | Query by date partition PK | #77 |

### Bug Fix Summary

| Bug | Fix | Pattern # |
|-----|-----|-----------|
| GetByOrderNumber queries wrong GSI | Separate lookup item `ORDER_NUMBER#<number>` | #56 |
| GetByCustomer uses missing GSI2 | Use GSI1 `CUSTOMER#<customer_id>` (already set by SetKeys) | #60 |
| Audit GetByEntity queries wrong GSI1PK | Fix SetKeys: GSI1PK = `<EntityType>#<EntityID>` | #78 |
| Audit GetByUser queries nonexistent GSI2 | Add GSI2 to CDK | #79 |
| Orders table GSI2 missing from CDK | Add to CDK | Infra fix |
| TTL not enabled on core/orders tables | Enable in CDK | Infra fix |

## 5. API Endpoints

### 5.1 Auth — `/api/v1/store/auth`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| POST | `/auth/otp/send` | public | `{phone}` | `{message}` |
| POST | `/auth/otp/verify` | public | `{phone, code}` | `{customer, is_new}` + cookies |
| POST | `/auth/refresh` | cookie | — | `{customer}` + new cookies |
| POST | `/auth/logout` | customer | — | `{message}` |

Rate limits: 3 OTP sends per phone per 10 min, 5 verify attempts per phone per 10 min.

### 5.2 Customer Profile — `/api/v1/store/me`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| GET | `/me` | customer | — | `Customer` (safe fields) |
| PATCH | `/me` | customer | `{first_name?, last_name?, email?}` | `Customer` |
| POST | `/me/addresses` | customer | `Address` | `{addresses}` |
| PUT | `/me/addresses/{id}` | customer | `Address` | `{addresses}` |
| DELETE | `/me/addresses/{id}` | customer | — | `{addresses}` |

### 5.3 Catalog — `/api/v1/store/categories`, `/api/v1/store/products`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| GET | `/categories` | public | — | `[]Category` |
| GET | `/categories/{idOrSlug}` | public | — | `Category` with filter options |
| GET | `/products` | public | `?category_id=&material=&color=&weave_type=&price_min=&price_max=&sort=&cursor=&limit=` | `{products, next_cursor}` |
| GET | `/products/{idOrSlug}` | public | — | `Product` with artisan name, stock |
| GET | `/products/search` | public | `?q=&cursor=&limit=` | `{products, next_cursor}` |
| GET | `/products/{id}/availability` | public | — | `{in_stock, available_qty}` |

### 5.4 Cart — `/api/v1/store/cart`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| GET | `/cart` | customer | — | `Cart` |
| POST | `/cart/items` | customer | `{product_id, quantity, dimensions?, quote_id?}` | `Cart` |
| PATCH | `/cart/items/{product_id}` | customer | `{quantity}` | `Cart` |
| DELETE | `/cart/items/{product_id}` | customer | — | `Cart` |
| DELETE | `/cart` | customer | — | `{}` |
| POST | `/cart/merge` | customer | `{items}` | `Cart` |

### 5.5 Checkout — `/api/v1/store/checkout`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| POST | `/checkout/serviceability` | customer | `{pincode}` | `{serviceable, couriers: [{name, rate, estimated_days}]}` |
| POST | `/checkout/initiate` | customer | `{shipping_address_id, courier_id?}` | `{order, payment: {merchant_txn_id, redirect_url}}` |
| GET | `/checkout/payment-status/{order_id}` | customer | — | `{status, order}` |

### 5.6 Webhooks

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| POST | `/webhooks/phonepe` | signature | PhonePe callback | `200 OK` |
| POST | `/webhooks/shiprocket` | token | Shiprocket push | `200 OK` |

### 5.7 Orders — `/api/v1/store/orders`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| GET | `/orders` | customer | `?cursor=&limit=` | `{orders, next_cursor}` |
| GET | `/orders/{id}` | customer | — | `OrderDetail` (items + history + shipment) |
| POST | `/orders/{id}/cancel` | customer | `{reason?}` | `Order` |

### 5.8 Public Tracking — `/api/v1/store/track`

| Method | Path | Auth | Request | Response |
|--------|------|------|---------|----------|
| GET | `/track/{order_number}` | public | — | `{status, timeline, shipment}` |

### 5.9 Rate Limiting

| Endpoint Group | Limit |
|---------------|-------|
| OTP send | 3 req / phone / 10 min |
| OTP verify | 5 req / phone / 10 min |
| Catalog browsing | 100 req / min / IP |
| Cart operations | 30 req / min / customer |
| Checkout initiate | 5 req / min / customer |
| Webhooks | No limit (verified by signature) |

## 6. Service Layer

### 6.1 New Services & Directory Structure

```
internal/
  handler/
    store/                          # NEW — all B2C handlers
      auth_handler.go
      profile_handler.go
      catalog_handler.go
      cart_handler.go
      checkout_handler.go
      order_handler.go
      tracking_handler.go
      webhook_handler.go
  service/
    customer_auth_service.go        # NEW
    cart_service.go                  # NEW
    checkout_service.go             # NEW
    payment_service.go              # NEW
    shipping_service.go             # NEW
  middleware/
    customer_auth.go                # NEW — customer JWT validation
  gateway/                          # NEW — external API integrations
    phonepe/
      client.go
      types.go
    shiprocket/
      client.go
      types.go
    sms/
      client.go
      types.go
```

### 6.2 Service Interfaces

```go
type CustomerAuthService interface {
    SendOTP(ctx context.Context, phone string) error
    VerifyOTP(ctx context.Context, phone, code string) (*Customer, *TokenPair, bool, error)
    RefreshToken(ctx context.Context, refreshToken string) (*Customer, *TokenPair, error)
    Logout(ctx context.Context, customerID, refreshToken string) error
    RevokeAllTokens(ctx context.Context, customerID string) error
}

type CartService interface {
    GetCart(ctx context.Context, customerID string) (*Cart, error)
    AddItem(ctx context.Context, customerID string, req AddCartItemRequest) (*Cart, error)
    UpdateItemQuantity(ctx context.Context, customerID, productID string, qty int) (*Cart, error)
    RemoveItem(ctx context.Context, customerID, productID string) (*Cart, error)
    ClearCart(ctx context.Context, customerID string) error
    MergeGuestCart(ctx context.Context, customerID string, guestItems []CartItem) (*Cart, error)
}

type CheckoutService interface {
    CheckServiceability(ctx context.Context, customerID, pincode string) (*ServiceabilityResult, error)
    Initiate(ctx context.Context, customerID string, req CheckoutRequest) (*CheckoutResult, error)
    GetPaymentStatus(ctx context.Context, customerID, orderID string) (*PaymentStatusResult, error)
}

type PaymentService interface {
    InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*PaymentResponse, error)
    HandleWebhook(ctx context.Context, payload []byte, signature string) error
    GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error)
    GetPaymentByTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
    RefundPayment(ctx context.Context, paymentID string, amount int64, reason string) error
}

type ShippingService interface {
    CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightGrams int) (*ServiceabilityResult, error)
    CreateShipment(ctx context.Context, orderID string) (*Shipment, error)
    TrackShipment(ctx context.Context, orderID string) (*TrackingResult, error)
    HandleWebhook(ctx context.Context, payload []byte, token string) error
}
```

### 6.3 Checkout Flow

```
Customer -> POST /checkout/initiate
  |
  v
CheckoutService.Initiate():
  1. CartService.GetCart(customerID)           -- get cart items
  2. BatchGetItem inventory                    -- validate stock for all items
  3. Calculate totals:
     - Subtotal = sum(item.unit_price * item.quantity)
     - Shipping = Shiprocket rate (from serviceability check, cached)
     - Tax = subtotal * 0.18 (GST)
     - Total = subtotal + shipping + tax
  4. TransactWriteItems (atomic, cross-table):
     - Put ORDER#<id> in orders table (status=PENDING)
     - Put ORDER_NUMBER#<number> in orders table
     - Put STATUS history (PENDING) in orders table
     - UpdateItem INVENTORY per item in core table (reserve stock, condition: available_qty >= qty)
  5. PaymentService.InitiatePayment():
     - Put PAYMENT#<id> in orders table (status=INITIATED)
     - Call PhonePe /pg/v1/pay -> get redirect URL
  6. CartService.ClearCart(customerID)
  7. Return {order, redirect_url}
```

### 6.4 Payment Webhook Flow

```
PhonePe -> POST /webhooks/phonepe
  |
  v
PaymentService.HandleWebhook():
  1. Verify X-VERIFY signature (SHA256 + salt)
  2. Query PAYMENT by merchant_txn_id (GSI2)
  3. Check idempotency (if payment.status != INITIATED, return 200 OK)
  4. If SUCCESS:
     - UpdateItem PAYMENT (status=SUCCESS, provider_txn_id, payment_method)
     - UpdateItem ORDER (status=CONFIRMED, payment_status=PAID)
     - PutItem STATUS history (PENDING -> CONFIRMED)
     - NotificationService: send order confirmation SMS + email
  5. If FAILED:
     - UpdateItem PAYMENT (status=FAILED)
     - UpdateItem ORDER (payment_status=FAILED)
     - UpdateItem INVENTORY per item (release reserved stock)
     - PutItem STATUS history (note: payment failed)
  6. Return 200 OK
```

## 7. External Integrations

### 7.1 PhonePe Payment Gateway

| Config | Value |
|--------|-------|
| Prod API | `https://api.phonepe.com/apis/hermes` |
| Sandbox API | `https://api-preprod.phonepe.com/apis/pg-sandbox` |
| Auth | Merchant ID + Salt Key + Salt Index |
| Flow | Standard Payment Page redirect |

**Initiate:** Generate merchantTransactionId (`MCHT_<order_id>_<timestamp>`). Build payload, base64 encode, SHA256 sign with salt. POST `/pg/v1/pay`. Get redirect URL.

**Webhook verify:** SHA256(`base64_response + "/pg/v1/status/" + merchantId + "/" + merchantTxnId + saltKey`) + "###" + saltIndex === X-VERIFY header.

**Idempotency:** Always check payment status before processing. PhonePe may send duplicate callbacks.

### 7.2 Shiprocket

| Config | Value |
|--------|-------|
| API | `https://apiv2.shiprocket.in/v1/external` |
| Auth | Email/password -> bearer token (cache, refresh on 401) |

| Operation | Endpoint | When |
|-----------|----------|------|
| Serviceability | `GET /courier/serviceability` | Checkout |
| Create order | `POST /orders/create/adhoc` | Admin: PROCESSING |
| Assign AWB | `POST /courier/assign/awb` | After order created |
| Generate label | `POST /courier/generate/label` | After AWB |
| Track | `GET /courier/track/awb/{awb}` | Tracking page |
| Cancel | `POST /orders/cancel` | Order cancellation |

### 7.3 SMS OTP — MSG91

| Config | Value |
|--------|-------|
| API | `https://control.msg91.com/api/v5/` |
| Auth | `authkey` header |

Self-managed OTP: generate 6-digit code, hash with SHA256, store in DynamoDB (TTL 5min), send raw SMS via MSG91 flow API. This gives full control over retry logic and rate limiting.

## 8. Next.js Storefront

### 8.1 Project: `homechrome-store/`

```
homechrome-store/
  src/
    app/                            # App Router
      layout.tsx                    # Root layout (header, footer, cart provider)
      page.tsx                      # Home (SSG, ISR 5min)
      categories/page.tsx           # Category listing (SSG, ISR 5min)
      c/[slug]/page.tsx             # Product listing by category (SSG, ISR 2min)
      p/[slug]/page.tsx             # Product detail (SSG, ISR 2min)
      cart/page.tsx                 # Cart (CSR)
      checkout/page.tsx             # Checkout (SSR, auth)
      checkout/confirmation/page.tsx # Post-payment (SSR)
      account/page.tsx              # Profile (SSR, auth)
      account/orders/page.tsx       # My orders (SSR, auth)
      account/orders/[id]/page.tsx  # Order detail (SSR, auth)
      account/addresses/page.tsx    # Manage addresses (SSR, auth)
      track/[orderNumber]/page.tsx  # Public tracking (SSR, no auth)
      login/page.tsx                # Phone + OTP
    components/
      layout/                       # Header, Footer, MobileNav
      catalog/                      # ProductCard, ProductGrid, FilterSidebar, CategoryCard
      cart/                         # CartDrawer, CartItem, CartSummary
      checkout/                     # AddressForm, ShippingOptions, OrderSummary
      order/                        # OrderCard, OrderTimeline, TrackingStatus
      common/                       # Button, Input, Modal, Spinner, Badge
    lib/
      api.ts                        # Fetch wrapper for Go backend
      auth.ts                       # Token management
      utils.ts                      # Format paise, date formatting
    hooks/
      useCart.ts                    # Cart state + mutations
      useAuth.ts                   # Auth state
    types/                          # Shared TypeScript types
```

### 8.2 Rendering Strategy

| Page | Strategy | Revalidation | Reason |
|------|----------|-------------|--------|
| Home | SSG | ISR 5 min | SEO + performance |
| Category listing | SSG | ISR 5 min | Rarely changes |
| Product listing | SSG | ISR 2 min | SEO. Client-side filtering on top. |
| Product detail | SSG | ISR 2 min | SEO critical. Stock check client-side. |
| Cart | CSR | — | Dynamic personal data |
| Checkout | SSR | — | Fresh stock + auth |
| Order confirmation | SSR | — | Post-payment |
| Account / Orders | SSR | — | Auth required |
| Public tracking | SSR | — | Real-time status |

### 8.3 SEO

- JSON-LD Product schema on every PDP (name, image, price, availability, brand)
- Breadcrumb schema on PLP and PDP
- Sitemap generated at build time (all active categories + products)
- Meta tags: requires adding `meta_title`, `meta_description` fields to Product and Category entities
- Open Graph tags for social sharing
- Canonical URLs using slugs

### 8.4 Deployment

Vercel for MVP — zero-config Next.js hosting, built-in ISR, image optimization, edge functions. Migrate to CloudFront + Lambda@Edge post-MVP if needed.
