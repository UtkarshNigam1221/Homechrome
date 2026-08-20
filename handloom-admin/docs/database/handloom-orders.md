# handloom-orders Table

The orders table manages all order-related data including orders, customers, payments, shipments, carts, and price quotes.

## Table Configuration

```
Table Name: handloom-orders
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
TTL Attribute: ttl
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |
| GSI2 | GSI2PK | GSI2SK | ALL |

---

## Entities

### 1. Order

Customer orders with items, pricing, and fulfillment tracking.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ORDER#<id>` | `ORDER#ord-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `CUSTOMER#<customer_id>` | `CUSTOMER#cust-001` |
| GSI1SK | `<created_at>` | `2024-01-15T10:30:00Z` |
| GSI2PK | `ORDER#ALL` | `ORDER#ALL` |
| GSI2SK | `<created_at>` | `2024-01-15T10:30:00Z` |

#### Order Number Uniqueness Index

Created atomically with the order via `TransactWriteItems`.

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ORDER_NUMBER#<number>` | `ORDER_NUMBER#HL-2024-0001` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| OrderNumber | String | Yes | Human-readable order number (e.g., `HL-2024-0001`) |
| CustomerID | String | Yes | Customer ID |
| CustomerName | String | Yes | Customer name (denormalized) |
| CustomerEmail | String | Yes | Customer email (denormalized) |
| CustomerPhone | String | No | Customer phone (denormalized) |
| Items | List[Object] | Yes | Order line items |
| ItemCount | Number | Yes | Total number of items |
| Subtotal | Number | Yes | Subtotal in paise |
| DiscountAmount | Number | No | Discount in paise |
| TaxAmount | Number | No | Tax in paise |
| ShippingAmount | Number | No | Shipping in paise |
| TotalAmount | Number | Yes | Total in paise |
| Currency | String | Yes | `INR` |
| CouponID | String | No | Applied coupon ID |
| CouponCode | String | No | Applied coupon code |
| Status | String | Yes | Order status |
| PaymentStatus | String | Yes | Payment status |
| PaymentMethod | String | No | Payment method |
| PaymentID | String | No | External payment ID |
| ShippingAddress | Object | Yes | Shipping address |
| BillingAddress | Object | No | Billing address |
| TrackingNumber | String | No | Shipment tracking number |
| TrackingURL | String | No | Tracking URL |
| ShippingCarrier | String | No | Carrier name |
| CustomerNote | String | No | Customer note |
| InternalNotes | List[Object] | No | Internal notes (appended via `list_append`) |
| ShippedAt | String | No | Shipping timestamp |
| DeliveredAt | String | No | Delivery timestamp |
| CancelledAt | String | No | Cancellation timestamp |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Status Enum

| Status | Description |
|--------|-------------|
| `PENDING` | Order placed, awaiting confirmation |
| `CONFIRMED` | Order confirmed |
| `PROCESSING` | Order being prepared |
| `SHIPPED` | Order shipped |
| `DELIVERED` | Order delivered |
| `CANCELLED` | Order cancelled |
| `RETURNED` | Order returned |

#### PaymentStatus Enum

| Status | Description |
|--------|-------------|
| `PENDING` | Payment pending |
| `PAID` | Payment received |
| `FAILED` | Payment failed |
| `REFUNDED` | Payment refunded |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get order by ID | PK = `ORDER#<id>`, SK = `METADATA` |
| Get orders by customer | GSI1: GSI1PK = `CUSTOMER#<customer_id>` (filter for entity_type) |
| List all orders | GSI2: GSI2PK = `ORDER#ALL` (with filter expressions for status, search) |
| Get order by number | PK = `ORDER_NUMBER#<number>`, SK = `METADATA` → returns order_id |

#### Write Patterns

| Operation | Method | Details |
|-----------|--------|---------|
| Create | `TransactWriteItems` | Order + order number index (2 items) |
| Update | `PutItem` | `attribute_exists(PK)` |
| Add note | `UpdateItem` | `list_append(if_not_exists(internal_notes, :empty), :note)` |

---

### 2. Customer

Customer information with addresses and order history.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMER#<id>` | `CUSTOMER#cust-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | *(unset)* | — |
| GSI1SK | *(unset)* | — |
| GSI2PK | `CUSTOMER#ALL` | `CUSTOMER#ALL` |
| GSI2SK | `<created_at>` | `2024-01-15T10:30:00Z` |

#### Email Uniqueness Index

A pointer item, not a GSI: it enforces uniqueness and is read-after-write
consistent, neither of which a GSI can do.

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMER_EMAIL#<email>` | `CUSTOMER_EMAIL#john@example.com` |
| SK | `METADATA` | `METADATA` |

Written in the same `TransactWriteItems` as the customer, so a customer can
never exist without its pointer.

#### Phone Uniqueness Index

Created as a best-effort second item during customer creation.

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMER_PHONE#<phone>` | `CUSTOMER_PHONE#+919876543210` |
| SK | `METADATA` | `METADATA` |

Additional attributes: `customer_id`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Email | String | No | Email address |
| FirstName | String | No | First name |
| LastName | String | No | Last name |
| Phone | String | Yes | Phone number (primary identifier for B2C) |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `BLOCKED` |
| Addresses | List[Object] | No | Saved addresses |
| Tags | List[String] | No | Customer tags |
| Notes | String | No | Internal notes |
| TotalOrders | Number | No | Order count (denormalized) |
| TotalSpent | Number | No | Total spent in paise (denormalized) |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get customer by ID | PK = `CUSTOMER#<id>`, SK = `METADATA` |
| Get customer by email | PK = `CUSTOMER_EMAIL#<email>`, SK = `METADATA` → returns customer_id |
| Get customer by phone | PK = `CUSTOMER_PHONE#<phone>`, SK = `METADATA` → returns customer_id |
| List all customers | GSI2: GSI2PK = `CUSTOMER#ALL` (with filter/search) |

---

### 3. Payment

Payment records for orders.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PAYMENT#<id>` | `PAYMENT#pay-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `ORDER#<order_id>` | `ORDER#ord-001` |
| GSI1SK | `<created_at>` | `2024-01-15T10:30:00Z` |
| GSI2PK | `MERCHANT_TXN#<merchant_txn_id>` | `MERCHANT_TXN#MCHT_abc123` |
| GSI2SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| OrderID | String | Yes | Associated order |
| Amount | Number | Yes | Payment amount in paise |
| Currency | String | Yes | `INR` |
| Status | String | Yes | `PENDING`, `SUCCESS`, `FAILED`, `REFUNDED` |
| Method | String | No | Payment method (UPI, CARD, etc.) |
| MerchantTransactionID | String | No | Gateway transaction ID |
| GatewayResponse | Map | No | Raw gateway response |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get payment by ID | PK = `PAYMENT#<id>`, SK = `METADATA` |
| Get payments for order | GSI1: GSI1PK = `ORDER#<order_id>` (newest first) |
| Get payment by merchant txn | GSI2: GSI2PK = `MERCHANT_TXN#<merchant_txn_id>`, GSI2SK = `METADATA` |

#### Write Patterns

| Operation | Method | Details |
|-----------|--------|---------|
| Create | `PutItem` | `attribute_not_exists(PK)` |
| Update status | `UpdateItem` | Dynamic update with `buildDynamicUpdate` helper |

---

### 4. Refund

One attempt to send money back for part or all of an order. Separate from
Payment because an order can be refunded line by line, several times over;
`Payment.RefundAmount` is the running total across them.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `REFUND#<id>` | `REFUND#refund_a1b2c3d4` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `ORDER#<order_id>` | `ORDER#ord-001` |
| GSI1SK | `REFUND#<initiated_at>` | `REFUND#2024-01-15T10:30:00Z` |
| GSI2PK | `REFUND_PROVIDER#<provider_refund_id>` | `REFUND_PROVIDER#OMR123` |
| GSI2SK | `METADATA` | `METADATA` |

GSI2 is omitted entirely until initiation returns a provider id: DynamoDB
rejects an empty string on an indexed key attribute.

#### Read Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get refund by ID | PK = `REFUND#<id>`, SK = `METADATA` |
| List an order's refunds | GSI1: GSI1PK = `ORDER#<order_id>`, `begins_with(GSI1SK, 'REFUND#')` |
| Get refund by provider id | GSI2: GSI2PK = `REFUND_PROVIDER#<id>`, GSI2SK = `METADATA` |

The `begins_with` matters: GSI1's `ORDER#` partition holds the order's payments
too, so an unnarrowed query returns both.

---

### 5. Shipment

Shipment records stored under the order partition.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ORDER#<order_id>` | `ORDER#ord-001` |
| SK | `SHIPMENT#<shipment_id>` | `SHIPMENT#ship-001` |

Multiple shipments per order are supported as separate SK items.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| OrderID | String | Yes | Associated order |
| Status | String | Yes | Shipment status |
| TrackingNumber | String | No | Carrier tracking number |
| TrackingURL | String | No | Tracking URL |
| Carrier | String | No | Shipping carrier |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get shipments for order | PK = `ORDER#<order_id>`, SK begins_with `SHIPMENT#` |
| Get latest shipment | PK = `ORDER#<order_id>`, SK begins_with `SHIPMENT#`, Limit 1, ScanIndexForward false |

#### Write Patterns

| Operation | Method | Condition |
|-----------|--------|-----------|
| Create | `PutItem` | `attribute_not_exists(PK) AND attribute_not_exists(SK)` |
| Update | `UpdateItem` | `attribute_exists(PK)`, dynamic update |

---

### 6. Cart

Customer shopping cart with header + line items pattern.

#### Key Structure (Header)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CART#<customer_id>` | `CART#cust-001` |
| SK | `METADATA` | `METADATA` |

#### Key Structure (Items)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CART#<customer_id>` | `CART#cust-001` |
| SK | `ITEM#<product_id>` | `ITEM#prod-001` |

#### TTL

Cart items have a TTL of **30 days** from last update. DynamoDB auto-deletes stale carts.

#### Attributes (Item)

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ProductID | String | Yes | Product ID |
| ProductName | String | Yes | Product name (denormalized) |
| Quantity | Number | Yes | Item quantity |
| Price | Number | Yes | Unit price in paise |
| Images | List[Object] | No | Product images (denormalized) |
| ttl | Number | Yes | Unix timestamp (30 days from update) |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get full cart | PK = `CART#<customer_id>` (returns header + all items) |
| Get cart items | PK = `CART#<customer_id>`, SK begins_with `ITEM#` |
| Get specific item | PK = `CART#<customer_id>`, SK = `ITEM#<product_id>` |

#### Write Patterns

| Operation | Method | Details |
|-----------|--------|---------|
| Add/update item | `PutItem` | Upsert with TTL |
| Remove item | `DeleteItem` | By PK + SK |
| Clear cart | `BatchWriteItem` | Delete all items (25-item batches) |

---

### 7. Price Quote

Temporary price calculations for custom-sized products.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `QUOTE#<id>` | `QUOTE#quote-001` |
| SK | `METADATA` | `METADATA` |

#### TTL Configuration

- **TTL Attribute**: `ttl` (set to `ValidUntil` Unix timestamp)
- **Default Expiry**: 24 hours
- **Auto-Deletion**: DynamoDB automatically deletes expired quotes

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| CategoryID | String | Yes | Product category |
| ProductID | String | No | Specific product |
| Dimensions | Object | Yes | Requested dimensions |
| Attributes | Map | No | Selected attributes |
| Quantity | Number | Yes | Requested quantity |
| PricingRuleID | String | Yes | Applied pricing rule |
| CalculatedPrice | Number | Yes | Total price in paise |
| PriceBreakdown | Object | Yes | Detailed price breakdown |
| ValidUntil | String | Yes | Quote expiration time |
| UsedInOrder | Boolean | No | Whether quote was used |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| ttl | Number | Yes | Unix timestamp for TTL |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get quote by ID | PK = `QUOTE#<id>`, SK = `METADATA` |

---

## Table Relationships

### Order → Customer

```
Order.CustomerID → Customer.ID
Order.GSI1PK = CUSTOMER#<customer_id>
```

### Order → Payment

```
Payment.GSI1PK = ORDER#<order_id>
(Multiple payments per order possible)
```

### Order → Shipment

```
Shipment.PK = ORDER#<order_id>, SK = SHIPMENT#<id>
(Co-located with order partition)
```

### Order → Quote

```
Order.Items[].QuoteID → PriceQuote.ID
```

---

## Denormalized Fields

| Field | Source | Updated When |
|-------|--------|--------------|
| Order.CustomerName | Customer.FirstName + LastName | Order creation |
| Order.CustomerEmail | Customer.Email | Order creation |
| Customer.TotalOrders | Count of orders | Order status change |
| Customer.TotalSpent | Sum of order totals | Payment confirmed |
