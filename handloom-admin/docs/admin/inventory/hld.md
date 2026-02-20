# Inventory Lambda - High-Level Design (HLD)

## 1. Overview

The Inventory Lambda service manages stock levels, reservations, and transaction history for all products in the Handloom Admin system. It handles inventory tracking, low stock alerts, and integrates with the order system for reservation management.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    INVENTORY SYSTEM                                          │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   React Frontend  │
                                    │   (Admin Portal)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway     │
                                    │   (REST API)      │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │  Stock Mgmt     │  │  Transactions   │  │  Low Stock      │
              │  - Add          │  │  - History      │  │  - Alerts       │
              │  - Remove       │  │  - By product   │  │  - Reports      │
              │  - Adjust       │  │                 │  │                 │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                                            ▼
                                  ┌─────────────────────┐
                                  │  Inventory Service  │
                                  │   (Business Logic)  │
                                  └──────────┬──────────┘
                                             │
                         ┌───────────────────┼───────────────────┐
                         │                   │                   │
                         ▼                   ▼                   ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   Product       │  │   CloudWatch    │
              │   (Inventory,   │  │   Service       │  │   (Logs &       │
              │   Transactions) │  │                 │  │   Metrics)      │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INVENTORY SERVICE LAYER                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       InventoryService                               │   │
│  │                                                                      │   │
│  │  Stock Management:                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - GetByProductID()    - AddStock()                           │ │   │
│  │  │  - RemoveStock()       - AdjustStock()                        │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Reservation Management:                                             │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - ReserveInventory()  - CommitReservation()                  │ │   │
│  │  │  - ReleaseReservation()                                       │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Reporting:                                                          │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - GetTransactions()   - GetLowStockProducts()                │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                             │                                               │
│              ┌──────────────┴──────────────┐                               │
│              │                             │                                │
│              ▼                             ▼                                │
│  ┌───────────────────────┐    ┌───────────────────────┐                   │
│  │ Inventory Repository  │    │   Product Repository  │                   │
│  └───────────────────────┘    └───────────────────────┘                   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TABLE: handloom-admin                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  INVENTORY RECORDS                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: INVENTORY#<product_id>                                        │      │
│  │ SK: CURRENT                                                       │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - product_id             - quantity                             │      │
│  │   - reserved_qty           - available_qty (calculated)           │      │
│  │   - low_stock_threshold    - is_low_stock                         │      │
│  │   - last_transaction_id    - last_updated                         │      │
│  │   - created_at             - updated_at                           │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  TRANSACTION RECORDS                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: INVENTORY#<product_id>                                        │      │
│  │ SK: TXN#<timestamp>#<txn_id>                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - id                     - product_id                           │      │
│  │   - type (ADD|REMOVE|ADJUST|RESERVE|RELEASE|COMMIT)              │      │
│  │   - quantity_change        - previous_qty                         │      │
│  │   - new_qty                - reason                               │      │
│  │   - reference_id           - reference_type                       │      │
│  │   - user_id                - created_at                           │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  RESERVATION RECORDS                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: RESERVATION#<order_id>                                        │      │
│  │ SK: ITEM#<product_id>                                             │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - order_id               - product_id                           │      │
│  │   - quantity               - status (PENDING|COMMITTED|RELEASED)  │      │
│  │   - created_at             - expires_at (TTL)                     │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GLOBAL SECONDARY INDEXES                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1: low-stock-index     (is_low_stock = true, sorted by qty)   │      │
│  │       GSI1PK: LOW_STOCK                                           │      │
│  │       GSI1SK: <quantity>                                          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Inventory Quantity Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INVENTORY QUANTITY MODEL                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Total Quantity (quantity)                                          │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                                                              │   │   │
│  │  │  ┌────────────────────────┐  ┌────────────────────────┐    │   │   │
│  │  │  │   Reserved Quantity    │  │  Available Quantity    │    │   │   │
│  │  │  │   (reserved_qty)       │  │  (available_qty)       │    │   │   │
│  │  │  │                        │  │                        │    │   │   │
│  │  │  │  - Pending orders      │  │  - Can be sold         │    │   │   │
│  │  │  │  - Not yet shipped     │  │  - qty - reserved_qty  │    │   │   │
│  │  │  │                        │  │                        │    │   │   │
│  │  │  └────────────────────────┘  └────────────────────────┘    │   │   │
│  │  │                                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                      │   │
│  │  Formulas:                                                           │   │
│  │  - available_qty = quantity - reserved_qty                          │   │
│  │  - is_low_stock = available_qty <= low_stock_threshold              │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Transaction Types

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TRANSACTION TYPES                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ADD - Increase total quantity                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Purchase order received                                            │   │
│  │ • Return from customer                                               │   │
│  │ • Transfer in from another location                                  │   │
│  │ • quantity_change: positive                                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  REMOVE - Decrease total quantity                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Damaged goods                                                      │   │
│  │ • Lost/stolen inventory                                              │   │
│  │ • Expired products                                                   │   │
│  │ • quantity_change: negative                                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ADJUST - Set to specific quantity                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Physical count correction                                          │   │
│  │ • System reconciliation                                              │   │
│  │ • Audit adjustment                                                   │   │
│  │ • quantity_change: calculated from old - new                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  RESERVE - Hold quantity for pending order                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Order created, payment pending                                     │   │
│  │ • Increases reserved_qty                                             │   │
│  │ • Does not change total quantity                                     │   │
│  │ • Has expiration (30 min default)                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  COMMIT - Confirm reservation (order shipped)                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Order payment confirmed and shipped                                │   │
│  │ • Decreases both quantity and reserved_qty                          │   │
│  │ • Finalizes the inventory change                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  RELEASE - Cancel reservation (order cancelled)                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Order cancelled or reservation expired                             │   │
│  │ • Decreases reserved_qty only                                        │   │
│  │ • Makes quantity available again                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Reservation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          RESERVATION LIFECYCLE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────┐     ┌──────────┐     ┌───────────┐     ┌──────────┐          │
│  │ Order   │────▶│ Reserve  │────▶│  Pending  │────▶│ Outcome  │          │
│  │ Created │     │ Stock    │     │           │     │          │          │
│  └─────────┘     └──────────┘     └─────┬─────┘     └──────────┘          │
│                                         │                                   │
│                         ┌───────────────┼───────────────┐                  │
│                         │               │               │                  │
│                         ▼               ▼               ▼                  │
│                  ┌───────────┐   ┌───────────┐   ┌───────────┐            │
│                  │  Payment  │   │   Order   │   │   Timer   │            │
│                  │ Confirmed │   │ Cancelled │   │  Expired  │            │
│                  └─────┬─────┘   └─────┬─────┘   └─────┬─────┘            │
│                        │               │               │                   │
│                        ▼               ▼               ▼                   │
│                  ┌───────────┐   ┌───────────┐   ┌───────────┐            │
│                  │  COMMIT   │   │  RELEASE  │   │  RELEASE  │            │
│                  │ (Ship)    │   │           │   │  (Auto)   │            │
│                  └───────────┘   └───────────┘   └───────────┘            │
│                                                                              │
│  Timeline:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  0 min          15 min          30 min                              │   │
│  │  │               │               │                                   │   │
│  │  ▼               ▼               ▼                                   │   │
│  │  Reserve ───────────────────────▶ Auto-Release (if not committed)   │   │
│  │  Created                           Reservation expires               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ERROR CODES                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Inventory Errors:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ INV001 │ Product not found                                          │   │
│  │ INV002 │ Inventory record not found                                 │   │
│  │ INV003 │ Insufficient stock                                         │   │
│  │ INV004 │ Insufficient available stock (reserved)                    │   │
│  │ INV005 │ Invalid quantity (must be positive)                        │   │
│  │ INV006 │ Invalid adjustment (negative result)                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Reservation Errors:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ RES001 │ Reservation not found                                      │   │
│  │ RES002 │ Reservation expired                                        │   │
│  │ RES003 │ Reservation already committed                              │   │
│  │ RES004 │ Reservation already released                               │   │
│  │ RES005 │ Cannot reserve more than available                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Transaction Errors:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ TXN001 │ Transaction not found                                      │   │
│  │ TXN002 │ Invalid transaction type                                   │   │
│  │ TXN003 │ Missing required reason                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Low Stock Management

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         LOW STOCK MANAGEMENT                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Threshold Configuration:                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Each product has a low_stock_threshold (default: 10)              │   │
│  │ • Configurable per product based on sales velocity                  │   │
│  │ • is_low_stock updated on every inventory change                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Detection Logic:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  After every inventory update:                                       │   │
│  │    available_qty = quantity - reserved_qty                          │   │
│  │    is_low_stock = available_qty <= low_stock_threshold              │   │
│  │                                                                      │   │
│  │  If is_low_stock changed to true:                                   │   │
│  │    - Update GSI for low stock query                                 │   │
│  │    - Trigger notification (optional)                                │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Low Stock Query:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Query GSI1 where GSI1PK = "LOW_STOCK"                             │   │
│  │ • Sorted by quantity (ascending) for urgency                        │   │
│  │ • Supports pagination for large catalogs                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Scalability & Performance

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     SCALABILITY CONSIDERATIONS                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Lambda Configuration:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Memory: 256 MB                                                    │   │
│  │ • Timeout: 30 seconds                                               │   │
│  │ • Concurrent executions: 200 (reserved)                             │   │
│  │ • Provisioned concurrency: 10 (for order processing)               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  DynamoDB Configuration:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Capacity: On-demand                                               │   │
│  │ • TTL: Enabled for expired reservations                             │   │
│  │ • Conditional writes: For concurrent update safety                  │   │
│  │ • Transaction support: For atomic reservation operations            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Concurrency Control:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Optimistic locking with version attribute                         │   │
│  │ • Conditional updates to prevent overselling                        │   │
│  │ • Retry logic with exponential backoff                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DEPENDENCIES                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB - Inventory and transaction storage                  │   │
│  │ • AWS CloudWatch - Logging and metrics                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Auth Lambda - Authentication                                      │   │
│  │ • Catalog Lambda - Product information                              │   │
│  │ • Order Lambda - Reservation triggers                               │   │
│  │ • Notification Lambda - Low stock alerts                            │   │
│  │ • Audit Lambda - Stock change logging                               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Libraries:                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • aws-sdk-go-v2/dynamodb - DynamoDB client                          │   │
│  │ • go-chi/chi - HTTP routing                                         │   │
│  │ • google/uuid - ID generation                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
