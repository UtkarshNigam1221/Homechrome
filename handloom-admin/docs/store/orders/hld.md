# Store Orders - High-Level Design (HLD)

## 1. Overview

The Store Orders module provides a customer-facing read-only view of their order history with the ability to cancel eligible orders. It reuses the existing admin OrderService but filters all queries by the authenticated customer's ID, ensuring customers can only access their own orders. The module exposes three endpoints: list orders, get order detail, and cancel order.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                  STORE ORDERS SYSTEM                                         │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │  Next.js Frontend │
                                    │  (B2C Storefront) │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (Customer JWT)
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway /   │
                                    │   Chi Router      │
                                    └─────────┬─────────┘
                                              │
                                              │ CustomerAuth Middleware
                                              ▼
                                    ┌───────────────────┐
                                    │  Store Order      │
                                    │  Handler          │
                                    │  - ListOrders     │
                                    │  - GetOrder       │
                                    │  - CancelOrder    │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │   Order Service   │
                                    │  (Shared with     │
                                    │   Admin)          │
                                    └─────────┬─────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
         ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
         │   DynamoDB      │       │  Inventory       │       │  Notification   │
         │  (handloom-     │       │  Service         │       │  Service        │
         │   orders)       │       │  (release stock) │       │  (cancel SMS)   │
         └─────────────────┘       └─────────────────┘       └─────────────────┘
```

---

## 3. Component Design

### 3.1 Handler Layer

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       STORE ORDER HANDLER                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      StoreOrderHandler                               │   │
│  │                                                                      │   │
│  │  Dependencies:                                                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - OrderService   (domain.OrderService)                       │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Routes (all require CustomerAuth middleware):                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  GET  /                → ListOrders(w, r)                     │ │   │
│  │  │  GET  /{id}            → GetOrder(w, r)                       │ │   │
│  │  │  POST /{id}/cancel     → CancelOrder(w, r)                   │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Key Behaviors:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  • Extracts customer_id from JWT claims (set by CustomerAuth)      │   │
│  │  • All queries are scoped to the authenticated customer_id         │   │
│  │  • GetOrder validates order.CustomerID == authenticated customer   │   │
│  │  • CancelOrder validates cancellable status before proceeding      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Order Status Machine (Customer-Visible)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      ORDER STATUS LIFECYCLE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Happy Path:                                                                 │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │                                                                   │      │
│  │  PENDING ──────▶ CONFIRMED ──────▶ SHIPPED ──────▶ DELIVERED     │      │
│  │                                                                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Cancellation (customer-initiated):                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │                                                                   │      │
│  │  PENDING ──────▶ CANCELLED                                       │      │
│  │  CONFIRMED ────▶ CANCELLED                                       │      │
│  │                                                                   │      │
│  │  SHIPPED / DELIVERED / CANCELLED ─── Cannot be cancelled ──▶ 400 │      │
│  │                                                                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Return (post-delivery):                                                     │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │                                                                   │      │
│  │  DELIVERED ────▶ RETURNED  (admin-initiated only)                │      │
│  │                                                                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Payment Statuses:                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │                                                                   │      │
│  │  PENDING ──────▶ PAID        (via PhonePe webhook)               │      │
│  │  PENDING ──────▶ FAILED      (via PhonePe webhook)               │      │
│  │  PAID ─────────▶ REFUNDED   (on cancellation of paid order)      │      │
│  │                                                                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Access Patterns

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-orders                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ORDER RECORD (reused from admin)                                            │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - OrderNumber           - CustomerID                            │      │
│  │   - CustomerName          - Items[]                               │      │
│  │   - Subtotal (paise)      - DiscountAmount (paise)                │      │
│  │   - TaxAmount (paise)     - ShippingAmount (paise)                │      │
│  │   - TotalAmount (paise)   - Currency ("INR")                      │      │
│  │   - Status                - PaymentStatus                         │      │
│  │   - ShippingAddress{}     - TrackingNumber                        │      │
│  │   - ShippingCarrier       - CreatedAt                             │      │
│  │   - UpdatedAt             - ShippedAt                             │      │
│  │   - DeliveredAt           - CancelledAt                           │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GSI1: Customer Order Listing                                                │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1PK: CUSTOMER#<customer_id>                                    │      │
│  │ GSI1SK: <created_at timestamp>                                    │      │
│  │                                                                   │      │
│  │ Access Patterns:                                                  │      │
│  │   • List all orders for customer (newest first)                   │      │
│  │   • Cursor-based pagination using GSI1SK as cursor                │      │
│  │   • Supports date-range filtering via GSI1SK BETWEEN              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER STATUS HISTORY                                                        │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: STATUS#<timestamp>                                            │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - FromStatus            - ToStatus                              │      │
│  │   - Reason                - CreatedAt                             │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SECURITY MODEL                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Authentication:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Customer JWT required for all endpoints                          │   │
│  │ • JWT extracted from `store_token` HttpOnly cookie                 │   │
│  │ • CustomerAuth middleware sets customer_id in request context       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Authorization:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • ListOrders: queries GSI1 with CUSTOMER#<customer_id> (inherent) │   │
│  │ • GetOrder: validates order.CustomerID == context customer_id      │   │
│  │ • CancelOrder: validates ownership + cancellable status            │   │
│  │ • No cross-customer data access is possible                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Data Filtering:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Internal notes (InternalNotes) are stripped from customer view   │   │
│  │ • Admin-only fields (CreatedBy, UpdatedBy) are not exposed         │   │
│  │ • Payment details (PaymentID, PaymentMethod) are not exposed       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            ERROR CODES                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Store Order Errors:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Code                  │ HTTP │ Description                          │   │
│  ├───────────────────────┼──────┼────────────────────────────────────┤   │
│  │ UNAUTHORIZED          │ 401  │ Missing or invalid customer JWT     │   │
│  │ ORDER_NOT_FOUND       │ 404  │ Order not found or not owned        │   │
│  │ ORDER_NOT_CANCELLABLE │ 400  │ Order status prevents cancellation  │   │
│  │ VALIDATION_ERROR      │ 400  │ Invalid query parameters            │   │
│  │ INTERNAL_ERROR        │ 500  │ Unexpected server error             │   │
│  └───────────────────────┴──────┴────────────────────────────────────┘   │
│                                                                              │
│  Standard response envelope:                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ {                                                                   │   │
│  │   "success": false,                                                 │   │
│  │   "error": {                                                        │   │
│  │     "code": "ORDER_NOT_CANCELLABLE",                                │   │
│  │     "message": "Order cannot be cancelled in SHIPPED status"        │   │
│  │   }                                                                 │   │
│  │ }                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INTEGRATION POINTS                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Order Service (shared)                            │   │
│  │                                                                      │   │
│  │  StoreOrderHandler ──▶ OrderService                                 │   │
│  │    • GetOrdersByCustomer(customerID, limit, cursor)                 │   │
│  │    • GetOrder(orderID) + ownership validation in handler            │   │
│  │    • CancelOrder(orderID) with status validation                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Inventory Service                                  │   │
│  │                                                                      │   │
│  │  OrderService ──▶ InventoryService (on cancel)                      │   │
│  │    • ReleaseReservation(orderID) — restores reserved stock          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Notification Service                               │   │
│  │                                                                      │   │
│  │  OrderService ──▶ NotificationService (on cancel)                   │   │
│  │    • SendCancellationSMS(phone, orderNumber)                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             DEPENDENCIES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB (handloom-orders table) — order storage & queries   │   │
│  │ • AWS CloudWatch — logging & monitoring                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • CustomerAuth Middleware — JWT validation, customer_id extraction │   │
│  │ • OrderService — shared business logic (admin + store)             │   │
│  │ • OrderRepository — DynamoDB data access (GSI1 for customer list) │   │
│  │ • InventoryService — stock release on cancellation                │   │
│  │ • NotificationService — SMS notifications on cancel               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Shared Domain Entities:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • domain.Order — order entity with items, pricing, status          │   │
│  │ • domain.OrderItem — line item with product info and pricing       │   │
│  │ • domain.Address — shipping address structure                      │   │
│  │ • domain.OrderStatusHistory — status change timeline               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
