# handloom-orders Table

The orders table manages all order-related data including orders, customers, and price quotes.

## Table Configuration

```
Table Name: handloom-orders
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |

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
| GSI1SK | `<timestamp>` | `2024-01-15T10:30:00Z` |

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
| InternalNotes | List[Object] | No | Internal notes |
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

#### Items Structure (OrderItem)

```json
{
  "ID": "item-001",
  "ProductID": "prod-001",
  "ProductName": "Red Silk Saree",
  "SKU": "HL-SAR-001",
  "Image": "https://cdn.example.com/image.jpg",
  "IsCustomSize": true,
  "Dimensions": {
    "Length": 550,
    "Width": 46,
    "Unit": "CM"
  },
  "Attributes": {
    "Material": "Silk",
    "Color": "Red"
  },
  "QuoteID": "quote-001",
  "UnitPrice": 1500000,
  "Quantity": 1,
  "TotalPrice": 1500000
}
```

#### Address Structure

```json
{
  "FirstName": "John",
  "LastName": "Doe",
  "Phone": "+91-9876543210",
  "AddressLine1": "123 Main Street",
  "AddressLine2": "Apartment 4B",
  "City": "Mumbai",
  "State": "Maharashtra",
  "PostalCode": "400001",
  "Country": "India"
}
```

#### InternalNotes Structure

```json
{
  "ID": "note-001",
  "Note": "Customer requested gift wrapping",
  "CreatedBy": "user-001",
  "CreatedAt": "2024-01-15T10:30:00Z"
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get order by ID | PK = `ORDER#<id>`, SK = `METADATA` |
| Get orders by customer | GSI1: GSI1PK = `CUSTOMER#<customer_id>` |
| Get customer orders in date range | GSI1: GSI1PK = `CUSTOMER#<customer_id>`, GSI1SK between dates |

---

### 2. Order Status History

Track order status changes over time.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ORDER#<order_id>` | `ORDER#ord-001` |
| SK | `STATUS#<timestamp>` | `STATUS#2024-01-15T10:30:00Z` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| OrderID | String | Yes | Order ID |
| FromStatus | String | Yes | Previous status |
| ToStatus | String | Yes | New status |
| Reason | String | No | Reason for change |
| CreatedBy | String | Yes | User ID |
| CreatedAt | String | Yes | ISO 8601 timestamp |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get order status history | PK = `ORDER#<order_id>`, SK begins_with `STATUS#` |
| Get status changes in date range | PK = `ORDER#<order_id>`, SK between `STATUS#<start>` and `STATUS#<end>` |

#### Example Query

```
Query:
  PK = "ORDER#ord-001"
  SK begins_with "STATUS#"

Returns:
  - STATUS#2024-01-15T10:00:00Z (PENDING → CONFIRMED)
  - STATUS#2024-01-15T14:30:00Z (CONFIRMED → PROCESSING)
  - STATUS#2024-01-16T09:00:00Z (PROCESSING → SHIPPED)
```

---

### 3. Customer

Customer information with addresses and order history.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CUSTOMER#<id>` | `CUSTOMER#cust-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `CUSTOMER_EMAIL` | `CUSTOMER_EMAIL` |
| GSI1SK | `<email>` | `john@example.com` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Email | String | Yes | Unique email address |
| FirstName | String | Yes | First name |
| LastName | String | Yes | Last name |
| Phone | String | No | Phone number |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `BLOCKED` |
| Addresses | List[Object] | No | Saved addresses |
| Tags | List[String] | No | Customer tags |
| Notes | String | No | Internal notes |
| TotalOrders | Number | No | Order count (denormalized) |
| TotalSpent | Number | No | Total spent in paise (denormalized) |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Status Enum

| Status | Description |
|--------|-------------|
| `ACTIVE` | Active customer |
| `INACTIVE` | Inactive customer |
| `BLOCKED` | Blocked customer (cannot place orders) |

#### Addresses Structure

```json
{
  "ID": "addr-001",
  "FirstName": "John",
  "LastName": "Doe",
  "Phone": "+91-9876543210",
  "AddressLine1": "123 Main Street",
  "AddressLine2": "Apartment 4B",
  "City": "Mumbai",
  "State": "Maharashtra",
  "PostalCode": "400001",
  "Country": "India",
  "IsDefault": true
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get customer by ID | PK = `CUSTOMER#<id>`, SK = `METADATA` |
| Get customer by email | GSI1: GSI1PK = `CUSTOMER_EMAIL`, GSI1SK = `<email>` |
| List all customers | Scan with PK begins_with `CUSTOMER#` |

---

### 4. Price Quote

Temporary price calculations for custom-sized products.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `QUOTE#<id>` | `QUOTE#quote-001` |
| SK | `METADATA` | `METADATA` |

#### TTL Configuration

- **TTL Attribute**: `ttl`
- **Default Expiry**: 24 hours (configurable via `QUOTE_VALIDITY_HRS`)
- **Purpose**: Auto-delete expired quotes to save storage

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| CategoryID | String | Yes | Product category |
| ProductID | String | No | Specific product (if applicable) |
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

#### Dimensions Structure

```json
{
  "Length": 550,
  "Width": 46,
  "Unit": "CM"
}
```

#### PriceBreakdown Structure

```json
{
  "Area": 25300,
  "AreaUnit": "SQ_CM",
  "BaseCost": 1000000,
  "MaterialMultiplier": 1.5,
  "MaterialCost": 1500000,
  "Surcharges": [
    {
      "Name": "Zari Work",
      "Type": "PERCENTAGE",
      "Value": 20,
      "Amount": 300000
    }
  ],
  "SurchargesTotal": 300000,
  "SubtotalPerUnit": 1800000,
  "Quantity": 1,
  "Total": 1800000
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get quote by ID | PK = `QUOTE#<id>`, SK = `METADATA` |

#### Quote Lifecycle

```
1. Customer requests custom size quote
   → Quote created with 24-hour validity

2. Customer adds to cart
   → Quote ID stored with cart item

3. Customer places order
   → Quote marked as UsedInOrder = true
   → Quote preserved for order reference

4. Quote expires (TTL)
   → Automatically deleted by DynamoDB
```

---

## Table Relationships

### Order → Customer

```
Order.CustomerID → Customer.ID
Order.GSI1PK = CUSTOMER#<customer_id>
```

### Order → Quote

```
Order.Items[].QuoteID → PriceQuote.ID
```

### Order → Coupon (in handloom-core)

```
Order.CouponID → Coupon.ID
Order.CouponCode → Coupon.Code
```

---

## Common Queries

### Get All Orders for a Customer

```
Table: handloom-orders
Index: GSI1
KeyCondition: GSI1PK = "CUSTOMER#cust-001"
SortOrder: Descending (newest first)
```

### Get Orders in Date Range for Customer

```
Table: handloom-orders
Index: GSI1
KeyCondition:
  GSI1PK = "CUSTOMER#cust-001"
  GSI1SK BETWEEN "2024-01-01T00:00:00Z" AND "2024-01-31T23:59:59Z"
```

### Get Order with Status History

```
Table: handloom-orders
KeyCondition: PK = "ORDER#ord-001"
(Returns both METADATA and STATUS# items)
```

### Find Customer by Email

```
Table: handloom-orders
Index: GSI1
KeyCondition:
  GSI1PK = "CUSTOMER_EMAIL"
  GSI1SK = "john@example.com"
```

---

## Data Integrity

### Denormalized Fields

| Field | Source | Updated When |
|-------|--------|--------------|
| Order.CustomerName | Customer.FirstName + LastName | Order creation |
| Order.CustomerEmail | Customer.Email | Order creation |
| Customer.TotalOrders | Count of orders | Order status change |
| Customer.TotalSpent | Sum of order totals | Order payment confirmed |

### Consistency Considerations

1. **Customer stats**: Updated asynchronously after order completion
2. **Quote validity**: Checked before order placement
3. **Inventory**: Reserved when order placed, released on cancellation

---

## Backup & Recovery

### Point-in-Time Recovery

- **Enabled**: Yes
- **Retention**: 35 days
- **Granularity**: Per-second

### Backup Strategy

1. Daily automated backups
2. Pre-deployment backups
3. Cross-region replication for DR
