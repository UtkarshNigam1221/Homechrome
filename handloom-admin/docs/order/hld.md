# Order Lambda - High-Level Design (HLD)

## 1. Overview

The Order Lambda service manages order lifecycle and customer data for the Handloom Admin system. It handles order creation, status management, payments, and customer relationship management.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                     ORDER SYSTEM                                             │
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
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
         ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
         │  Order Handler  │       │ Customer Handler│       │ Payment Handler │
         │  - Create       │       │  - CRUD         │       │  - Process      │
         │  - Status       │       │  - Search       │       │  - Refund       │
         │  - Cancel       │       │  - Orders       │       │                 │
         └────────┬────────┘       └────────┬────────┘       └────────┬────────┘
                  │                         │                         │
                  └─────────────────────────┼─────────────────────────┘
                                            │
                                            ▼
                                 ┌─────────────────────┐
                                 │    Order Service    │
                                 │   (Business Logic)  │
                                 └──────────┬──────────┘
                                            │
         ┌──────────────────────────────────┼──────────────────────────────────┐
         │                    │             │             │                    │
         ▼                    ▼             ▼             ▼                    ▼
┌─────────────────┐  ┌─────────────┐  ┌──────────┐  ┌───────────┐  ┌─────────────────┐
│   DynamoDB      │  │  Inventory  │  │ Pricing  │  │ Notifica- │  │   CloudWatch    │
│  (Orders,       │  │  Lambda     │  │ Lambda   │  │ tion Svc  │  │   (Logs)        │
│   Customers)    │  │             │  │          │  │           │  │                 │
└─────────────────┘  └─────────────┘  └──────────┘  └───────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ORDER SERVICE LAYER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Handler Layer                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │   Order     │  │  Customer   │  │   Status    │                 │   │
│  │  │  Handler    │  │  Handler    │  │  Handler    │                 │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │   │
│  └─────────┼────────────────┼────────────────┼──────────────────────────┘   │
│            │                │                │                              │
│            └────────────────┼────────────────┘                              │
│                             ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Order Service                                │   │
│  │                                                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - CreateOrder()           - UpdateOrderStatus()              │ │   │
│  │  │  - GetOrder()              - CancelOrder()                    │ │   │
│  │  │  - ListOrders()            - GetOrdersByCustomer()            │ │   │
│  │  │  - CreateCustomer()        - SearchCustomers()                │ │   │
│  │  │  - GetCustomer()           - UpdateCustomer()                 │ │   │
│  │  │  - GetCustomerOrders()     - CalculateOrderTotal()            │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                             │                                               │
│         ┌───────────────────┼───────────────────┐                          │
│         │                   │                   │                          │
│         ▼                   ▼                   ▼                          │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐              │
│  │ Order           │ │ Customer        │ │ External        │              │
│  │ Repository      │ │ Repository      │ │ Services        │              │
│  │                 │ │                 │ │ (Inventory,     │              │
│  │                 │ │                 │ │  Pricing, etc)  │              │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘              │
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
│  ORDER RECORDS                                                               │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - order_number (GSI1-PK)   - customer_id (GSI2-PK)             │      │
│  │   - status (GSI2-SK)         - payment_status                     │      │
│  │   - items[]                  - subtotal                           │      │
│  │   - discount                 - tax                                │      │
│  │   - shipping                 - total                              │      │
│  │   - shipping_address{}       - billing_address{}                  │      │
│  │   - payment_details{}        - timeline[]                         │      │
│  │   - notes                    - created_by                         │      │
│  │   - created_at (GSI3-SK)     - updated_at                         │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER ITEMS (Embedded in Order)                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ items: [                                                          │      │
│  │   {                                                               │      │
│  │     product_id, product_name, sku,                                │      │
│  │     quantity, unit_price, discount, total_price                   │      │
│  │   }                                                               │      │
│  │ ]                                                                 │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER TIMELINE (Embedded in Order)                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ timeline: [                                                       │      │
│  │   { status, timestamp, notes, updated_by }                        │      │
│  │ ]                                                                 │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  CUSTOMER RECORDS                                                            │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CUSTOMER#<customer_id>                                        │      │
│  │ SK: PROFILE                                                       │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name                     - email (GSI4-PK)                    │      │
│  │   - phone (GSI5-PK)          - addresses[]                        │      │
│  │   - total_orders             - total_spent                        │      │
│  │   - last_order_date          - created_at                         │      │
│  │   - updated_at               - notes                              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GLOBAL SECONDARY INDEXES                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1: order-number-index   (order number lookups)                 │      │
│  │ GSI2: customer-status      (orders by customer + status)          │      │
│  │ GSI3: status-date          (orders by status + date)              │      │
│  │ GSI4: email-index          (customer by email)                    │      │
│  │ GSI5: phone-index          (customer by phone)                    │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Order Status Machine

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ORDER STATUS TRANSITIONS                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Valid Transitions:                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │                                                                   │      │
│  │  PENDING ──────────▶ CONFIRMED ──────────▶ PROCESSING            │      │
│  │     │                    │                      │                 │      │
│  │     │                    │                      │                 │      │
│  │     ▼                    ▼                      ▼                 │      │
│  │  CANCELLED ◀──────── CANCELLED ◀────────── CANCELLED             │      │
│  │                                                                   │      │
│  │                                                                   │      │
│  │  PROCESSING ─────────▶ SHIPPED ──────────▶ DELIVERED             │      │
│  │                                                                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Transition Rules:                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ From        │ To           │ Conditions                           │      │
│  ├─────────────┼──────────────┼──────────────────────────────────────┤      │
│  │ PENDING     │ CONFIRMED    │ Payment received                     │      │
│  │ PENDING     │ CANCELLED    │ Manual cancel or payment failed      │      │
│  │ CONFIRMED   │ PROCESSING   │ Order picked for fulfillment         │      │
│  │ CONFIRMED   │ CANCELLED    │ Manual cancel (with refund)          │      │
│  │ PROCESSING  │ SHIPPED      │ Tracking number provided             │      │
│  │ PROCESSING  │ CANCELLED    │ Manual cancel (with refund)          │      │
│  │ SHIPPED     │ DELIVERED    │ Delivery confirmed                   │      │
│  └─────────────┴──────────────┴──────────────────────────────────────┘      │
│                                                                              │
│  Side Effects:                                                               │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Transition        │ Side Effects                                  │      │
│  ├───────────────────┼───────────────────────────────────────────────┤      │
│  │ → CONFIRMED       │ Reserve inventory, send confirmation email   │      │
│  │ → PROCESSING      │ Update inventory (committed)                 │      │
│  │ → SHIPPED         │ Send shipping notification with tracking     │      │
│  │ → DELIVERED       │ Request review, update customer stats        │      │
│  │ → CANCELLED       │ Release inventory, process refund if paid    │      │
│  └───────────────────┴───────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Order Creation Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ORDER CREATION PIPELINE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Step 1: Validate Request                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Validate customer exists                                        │      │
│  │ • Validate product IDs exist                                      │      │
│  │ • Validate quantities are positive                                │      │
│  │ • Validate shipping address                                       │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                             │                                                │
│                             ▼                                                │
│  Step 2: Check Inventory                                                     │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Call Inventory Lambda for each item                             │      │
│  │ • Check available quantity >= requested                           │      │
│  │ • Return error if any item out of stock                           │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                             │                                                │
│                             ▼                                                │
│  Step 3: Calculate Pricing                                                   │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Get current prices for products                                 │      │
│  │ • Apply pricing rules                                             │      │
│  │ • Validate and apply coupon if provided                           │      │
│  │ • Calculate tax based on shipping address                         │      │
│  │ • Calculate shipping cost                                         │      │
│  │ • Generate order total                                            │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                             │                                                │
│                             ▼                                                │
│  Step 4: Reserve Inventory                                                   │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Create reservation for each item                                │      │
│  │ • Reservation expires after payment timeout (30 mins)             │      │
│  │ • Link reservation to order ID                                    │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                             │                                                │
│                             ▼                                                │
│  Step 5: Create Order Record                                                 │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Generate order ID and order number                              │      │
│  │ • Store order with status PENDING                                 │      │
│  │ • Initialize order timeline                                       │      │
│  │ • Create audit log entry                                          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                             │                                                │
│                             ▼                                                │
│  Step 6: Send Notifications                                                  │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ • Send order confirmation to customer                             │      │
│  │ • Send admin notification for new order                           │      │
│  │ • Update customer order statistics                                │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Order Errors:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ ORD001 │ Order not found                                            │   │
│  │ ORD002 │ Invalid status transition                                  │   │
│  │ ORD003 │ Order already cancelled                                    │   │
│  │ ORD004 │ Cannot cancel shipped order                                │   │
│  │ ORD005 │ Insufficient inventory                                     │   │
│  │ ORD006 │ Invalid coupon code                                        │   │
│  │ ORD007 │ Payment processing failed                                  │   │
│  │ ORD008 │ Inventory reservation expired                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Customer Errors:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ CUS001 │ Customer not found                                         │   │
│  │ CUS002 │ Email already registered                                   │   │
│  │ CUS003 │ Invalid phone number                                       │   │
│  │ CUS004 │ Customer has pending orders (cannot delete)                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          INTEGRATION POINTS                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Inventory Integration                          │   │
│  │                                                                      │   │
│  │  Order Service ──▶ Inventory Lambda                                 │   │
│  │    • CheckStock(productId, quantity)                                │   │
│  │    • ReserveInventory(orderId, items, expiryMins)                   │   │
│  │    • CommitReservation(orderId)                                     │   │
│  │    • ReleaseReservation(orderId)                                    │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Pricing Integration                           │   │
│  │                                                                      │   │
│  │  Order Service ──▶ Pricing Lambda                                   │   │
│  │    • CalculatePrice(items, couponCode)                              │   │
│  │    • ValidateCoupon(code, customerId, orderValue)                   │   │
│  │    • ApplyCoupon(couponId, orderId)                                 │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Notification Integration                         │   │
│  │                                                                      │   │
│  │  Order Service ──▶ Notification Lambda                              │   │
│  │    • SendOrderConfirmation(orderId, customerEmail)                  │   │
│  │    • SendStatusUpdate(orderId, newStatus)                           │   │
│  │    • SendShippingNotification(orderId, trackingInfo)                │   │
│  │    • SendCancellationNotification(orderId, reason)                  │   │
│  │                                                                      │   │
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
│  │ • Memory: 512 MB                                                    │   │
│  │ • Timeout: 30 seconds                                               │   │
│  │ • Concurrent executions: 300 (reserved)                             │   │
│  │ • Provisioned concurrency: 20 (for peak hours)                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  DynamoDB Configuration:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Capacity: On-demand                                               │   │
│  │ • GSI count: 5 (for various access patterns)                        │   │
│  │ • TTL: Enabled for reservations                                     │   │
│  │ • Streams: Enabled for analytics                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Consistency Requirements:                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Order creation: Eventually consistent                             │   │
│  │ • Inventory updates: Strongly consistent                            │   │
│  │ • Status updates: Conditional writes with version                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB - Order & customer storage                           │   │
│  │ • AWS SES - Email notifications                                     │   │
│  │ • AWS CloudWatch - Logging & monitoring                             │   │
│  │ • Payment Gateway (Razorpay/Stripe) - Payment processing            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Auth Lambda - Authentication                                      │   │
│  │ • Inventory Lambda - Stock management                               │   │
│  │ • Pricing Lambda - Price calculation                                │   │
│  │ • Coupon Lambda - Discount validation                               │   │
│  │ • Notification Lambda - Email/SMS                                   │   │
│  │ • Audit Lambda - Change logging                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
