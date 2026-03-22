# Store Checkout - High-Level Design (HLD)

## 1. Overview

The Store Checkout service orchestrates the order placement and payment flow for the B2C customer storefront. It validates the customer's cart, checks delivery serviceability via Shiprocket, creates an order in DynamoDB, reserves inventory, initiates payment via PhonePe, and returns a payment redirect URL. An asynchronous payment webhook updates the order and payment status. In dev mode, PhonePe and Shiprocket DevClients provide mock responses to enable local development without external API access.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CHECKOUT SYSTEM                                            │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │  Next.js Frontend │
                                    │  (B2C Storefront) │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (Customer JWT Cookie)
                                              ▼
                              ┌───────────────────────────────┐
                              │      API Gateway / Lambda     │
                              │         /api/v1/store         │
                              └───────────────┬───────────────┘
                                              │
                    ┌─────────────────────────┼──────────────────────────────┐
                    │                         │                              │
                    ▼                         ▼                              ▼
         ┌─────────────────┐       ┌─────────────────┐            ┌─────────────────┐
         │ Checkout Handler│       │ Webhook Handler  │            │ Payment Status  │
         │                 │       │ (PhonePe         │            │ Handler         │
         │ POST /service-  │       │  callback)       │            │                 │
         │   ability       │       │                  │            │ GET /payment-   │
         │ POST /initiate  │       │ POST /webhooks/  │            │   status/{id}   │
         │                 │       │   payment        │            │                 │
         └────────┬────────┘       └────────┬────────┘            └────────┬────────┘
                  │                         │                              │
                  └─────────────────────────┼──────────────────────────────┘
                                            │
                                            ▼
                                 ┌─────────────────────┐
                                 │  Checkout Service   │
                                 │  (Orchestrator)     │
                                 └──────────┬──────────┘
                                            │
         ┌──────────────────────────────────┼──────────────────────────────────┐
         │                    │             │             │                    │
         ▼                    ▼             ▼             ▼                    ▼
┌─────────────────┐  ┌─────────────┐  ┌──────────┐  ┌───────────┐  ┌─────────────────┐
│   Cart Service  │  │  Shipping   │  │ Payment  │  │  Order    │  │  Inventory      │
│                 │  │  Service    │  │ Service  │  │  Repo     │  │  Service        │
│  Get cart items │  │  (Shiprocket│  │(PhonePe) │  │ (DynamoDB)│  │  Reserve/       │
│  Clear cart     │  │   Gateway)  │  │          │  │           │  │  Release stock  │
└─────────────────┘  └─────────────┘  └──────────┘  └───────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CHECKOUT SERVICE LAYER                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Handler Layer                                │   │
│  │                                                                      │   │
│  │  CheckoutHandler (internal/handler/store/checkout_handler.go)       │   │
│  │  WebhookHandler  (internal/handler/store/webhook_handler.go)        │   │
│  │                                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │   │
│  │  │Serviceability│  │  Initiate    │  │PaymentStatus │              │   │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │   │
│  └─────────┼─────────────────┼─────────────────┼────────────────────────┘   │
│            │                 │                 │                            │
│            └─────────────────┼─────────────────┘                            │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Checkout Service                               │   │
│  │  implements domain.CheckoutService                                  │   │
│  │                                                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - CheckServiceability(cID, pincode) → ServiceabilityResult   │ │   │
│  │  │  - Initiate(cID, req)                → CheckoutResult         │ │   │
│  │  │  - GetPaymentStatus(cID, orderID)    → PaymentStatusResult    │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│    ┌────────────────┬────────┼────────┬────────────────┐                   │
│    │                │        │        │                │                   │
│    ▼                ▼        ▼        ▼                ▼                   │
│ ┌────────┐  ┌──────────┐ ┌───────┐ ┌─────────┐ ┌──────────┐              │
│ │ Cart   │  │ Shipping │ │Payment│ │ Order   │ │Inventory │              │
│ │Service │  │ Service  │ │Service│ │  Repo   │ │ Service  │              │
│ └────────┘  └──────────┘ └───────┘ └─────────┘ └──────────┘              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Checkout Initiation Pipeline

The `Initiate` method follows this strict sequence:

1. **Get cart** — fetch customer cart, fail if empty (`EMPTY_CART`)
2. **Validate address** — fetch customer, find address by ID, fail if missing (`ADDRESS_NOT_FOUND`)
3. **Check inventory** — for each cart item, verify `available_qty >= quantity` (`INSUFFICIENT_STOCK`)
4. **Create order** — build Order entity from cart items, address, and pricing; status=PENDING, payment_status=PENDING
5. **Reserve inventory** — decrement `available_qty`, increment `reserved_qty` for each item
6. **Initiate payment** — call PaymentService with order amount and customer phone; get redirect URL
7. **Clear cart** — remove all cart items after successful order creation
8. **Return** — CheckoutResult with order, redirect URL, and merchant transaction ID

### 3.3 Dev Mode Clients

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DEV MODE BEHAVIOR                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PhonePe DevClient (used when PHONEPE_CLIENT_ID or CLIENT_SECRET empty):    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Activated by credential presence, NOT by APP_ENV / IsDevelopment()│   │
│  │ - Returns redirect_url pointing to storefront confirmation page:    │   │
│  │   /order-confirmation?order_id=<order_id>&dev_payment=<txn_id>     │   │
│  │ - CheckPaymentStatus always returns COMPLETED                       │   │
│  │ - No actual PhonePe API calls made                                  │   │
│  │ - OTP printed to console (not sent via SMS)                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Shiprocket DevClient:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Returns mock courier options for any valid 6-digit pincode        │   │
│  │ - Default couriers: Delhivery (50.00 INR, 4 days),                  │   │
│  │   BlueDart (75.00 INR, 3 days), DTDC (45.00 INR, 6 days)           │   │
│  │ - No actual Shiprocket API calls made                               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 Order Entity (DynamoDB — handloom-orders)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       TABLE: handloom-orders                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ORDER                                                                       │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: METADATA                                                      │      │
│  │ GSI1PK: CUSTOMER#<customer_id>                                    │      │
│  │ GSI1SK: <created_at ISO 8601>                                     │      │
│  │ GSI2PK: ORDER#ALL                                                 │      │
│  │ GSI2SK: <created_at ISO 8601>                                     │      │
│  │ entity_type: ORDER                                                │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - id, order_number (ORD-YYYY-NNNNNN)                           │      │
│  │   - customer_id, customer_name, customer_email, customer_phone    │      │
│  │   - items[] (OrderItem embedded list)                             │      │
│  │   - item_count, subtotal, discount_amount, tax_amount             │      │
│  │   - shipping_amount, total_amount, currency ("INR")               │      │
│  │   - coupon_id, coupon_code                                        │      │
│  │   - status (PENDING|CONFIRMED|PROCESSING|SHIPPED|DELIVERED|       │      │
│  │             CANCELLED|RETURNED)                                    │      │
│  │   - payment_status (PENDING|PAID|FAILED|REFUNDED)                 │      │
│  │   - payment_method, payment_id                                    │      │
│  │   - shipping_address{}, billing_address{}                         │      │
│  │   - tracking_number, tracking_url, shipping_carrier               │      │
│  │   - customer_note, internal_notes[]                               │      │
│  │   - shipped_at, delivered_at, cancelled_at                        │      │
│  │   - created_at, updated_at, created_by, updated_by               │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PAYMENT                                                                     │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PAYMENT#<payment_id>                                          │      │
│  │ SK: METADATA                                                      │      │
│  │ GSI1PK: ORDER#<order_id>                                          │      │
│  │ GSI1SK: PAYMENT#<initiated_at>                                    │      │
│  │ GSI2PK: PAYMENT_TXN                                               │      │
│  │ GSI2SK: <merchant_transaction_id>                                 │      │
│  │ entity_type: PAYMENT                                              │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - id, order_id, customer_id                                     │      │
│  │   - amount (paise), currency                                      │      │
│  │   - status (PENDING|INITIATED|SUCCESS|PAID|FAILED|REFUNDED)       │      │
│  │   - provider (PHONEPE)                                            │      │
│  │   - merchant_transaction_id (HC-<order_id>)                       │      │
│  │   - provider_transaction_id                                       │      │
│  │   - payment_method (UPI|CARD|NET_BANKING|WALLET)                  │      │
│  │   - provider_response, initiated_at, completed_at                 │      │
│  │   - refund_amount, refunded_at                                    │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER NUMBER INDEX                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER_NUMBER#ORD-2026-000142                                  │      │
│  │ SK: METADATA                                                      │      │
│  │ entity_type: ORDER_NUMBER_INDEX                                   │      │
│  │ order_id: <order_id>                                              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER STATUS HISTORY                                                        │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: STATUS#<timestamp>                                            │      │
│  │ entity_type: ORDER_STATUS_HISTORY                                 │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - order_id, from_status, to_status, reason                     │      │
│  │   - created_by, created_at                                        │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Order Number Format

Order numbers follow the pattern `ORD-YYYY-NNNNNN`:
- `YYYY` — current year
- `NNNNNN` — zero-padded auto-incremented counter
- Example: `ORD-2026-000142`
- Counter stored in DynamoDB as an atomic counter item

### 4.3 Order Status Machine

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       ORDER STATUS TRANSITIONS                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                                                                              │
│  PENDING ─────── payment success ──────▶ CONFIRMED ──────▶ PROCESSING       │
│     │                                        │                   │           │
│     │ payment failed                         │                   │           │
│     ▼                                        ▼                   ▼           │
│  CANCELLED ◀─────── admin cancel ────── CANCELLED ◀──── CANCELLED           │
│                                                                              │
│                                                                              │
│  PROCESSING ──────── ship order ──────▶ SHIPPED ──────▶ DELIVERED           │
│                                                                              │
│                                                                              │
│  Checkout creates order with: status=PENDING, payment_status=PENDING        │
│  Payment webhook updates:     payment_status=PAID → status=CONFIRMED        │
│  Payment failure:             payment_status=FAILED → status=CANCELLED      │
│                               + release inventory                            │
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
│  │ - Customer JWT via HttpOnly cookie (CUSTOMER_JWT_SECRET)            │   │
│  │ - CustomerAuth middleware on all checkout routes                     │   │
│  │ - Webhook endpoint uses signature verification (not JWT)            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Authorization:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Orders scoped to authenticated customer_id                        │   │
│  │ - GetPaymentStatus validates order.customer_id == jwt.customer_id   │   │
│  │ - Shipping address validated against customer's saved addresses     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Payment Security:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - PhonePe webhook auth: SHA256(username:password) in Authorization  │   │
│  │   header (webhook credentials: PHONEPE_WEBHOOK_USERNAME/PASSWORD)   │   │
│  │ - Merchant transaction ID format: HC-<order_id>                     │   │
│  │ - Payment amount validated against order total (server-side)        │   │
│  │ - No client-side amount manipulation possible                       │   │
│  │ - Idempotent webhook handling (duplicate callbacks safe)            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Inventory Protection:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Inventory reserved atomically on order creation                   │   │
│  │ - Reserved stock released on payment failure/cancellation           │   │
│  │ - DynamoDB conditional writes prevent overselling                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Checkout Errors:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ VALIDATION_ERROR           │ 400 │ Invalid request body             │   │
│  │ EMPTY_CART                 │ 400 │ Cart has no items                │   │
│  │ UNAUTHORIZED               │ 401 │ Missing/invalid customer JWT     │   │
│  │ ADDRESS_NOT_FOUND          │ 404 │ Address not in customer profile  │   │
│  │ ORDER_NOT_FOUND            │ 404 │ Order not found or wrong owner   │   │
│  │ INSUFFICIENT_STOCK         │ 409 │ Stock depleted since add-to-cart │   │
│  │ DELIVERY_NOT_SERVICEABLE   │ 422 │ Pincode not serviceable          │   │
│  │ PAYMENT_INITIATION_FAILED  │ 500 │ PhonePe API call failed          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Rollback Strategy:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ If payment initiation fails after order creation:                   │   │
│  │   1. Mark order as CANCELLED                                        │   │
│  │   2. Release reserved inventory                                     │   │
│  │   3. Return PAYMENT_INITIATION_FAILED error                         │   │
│  │                                                                      │   │
│  │ If inventory reservation fails mid-way:                             │   │
│  │   1. Release already-reserved items (compensating transaction)      │   │
│  │   2. Do not create order                                            │   │
│  │   3. Return INSUFFICIENT_STOCK error                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Error Response Format:                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ {                                                                   │   │
│  │   "success": false,                                                 │   │
│  │   "error": {                                                        │   │
│  │     "code": "EMPTY_CART",                                           │   │
│  │     "message": "Your cart is empty. Add items before checkout."     │   │
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
│                          INTEGRATION POINTS                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Cart Service                                    │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ CartService                                    │   │
│  │    - GetCart(customerID) → get items for order creation              │   │
│  │    - ClearCart(customerID) → clear after successful order           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Shipping Service (Shiprocket Gateway)           │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ ShippingService                                │   │
│  │    - CheckServiceability(pickupPincode, deliveryPincode, weight)    │   │
│  │    - Returns: ServiceabilityResult{serviceable, couriers[]}         │   │
│  │    - Pickup pincode: from config (PICKUP_PINCODE env var)           │   │
│  │    - Weight: estimated from cart items                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Payment Service (PhonePe Gateway)               │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ PaymentService                                 │   │
│  │    - InitiatePayment(orderID, customerID, amount, phone)            │   │
│  │    - Returns: PaymentResponse{payment_id, redirect_url, txn_id}    │   │
│  │    - HandleWebhook(payload, signature) → update payment + order    │   │
│  │    - GetByOrderID(orderID) → current payment status                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Inventory Service                               │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ InventoryService                               │   │
│  │    - Reserve: decrement available_qty, increment reserved_qty       │   │
│  │    - Release: increment available_qty, decrement reserved_qty       │   │
│  │    - DynamoDB conditional writes prevent overselling                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Order Repository                                │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ OrderRepository                                │   │
│  │    - Create(order) → persist new order                              │   │
│  │    - GetByID(orderID) → fetch order for status checks               │   │
│  │    - UpdateStatus(orderID, status, paymentStatus) → webhook path   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  AWS Services:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - DynamoDB (handloom-orders) — Orders, Payments, Order history      │   │
│  │ - DynamoDB (handloom-core) — Inventory reads and updates            │   │
│  │ - CloudWatch — Logging and metrics                                  │   │
│  │ - API Gateway — HTTP endpoint (Lambda mode)                         │   │
│  │ - Lambda — Compute (production)                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - PhonePe Payment Gateway — UPI, card, net banking, wallet          │   │
│  │   API: Standard Checkout v2 (OAuth token + /checkout/v2/pay)         │   │
│  │   Dev: DevClient when PHONEPE_CLIENT_ID/SECRET empty                │   │
│  │                                                                      │   │
│  │ - Shiprocket Shipping API — Serviceability, rates, tracking         │   │
│  │   API: https://apiv2.shiprocket.in/v1/external                      │   │
│  │   Dev: DevClient with mock courier data                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - CustomerAuth Middleware — JWT authentication                      │   │
│  │ - Cart Service — Cart reading and clearing                          │   │
│  │ - Inventory Service — Stock reservation and release                 │   │
│  │ - Customer Repository — Address lookup and validation               │   │
│  │ - Notification Service — Order confirmation SMS (future)            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Go Packages:                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - github.com/go-chi/chi/v5 — HTTP router                           │   │
│  │ - github.com/aws/aws-sdk-go-v2/service/dynamodb — DynamoDB client  │   │
│  │ - github.com/google/wire — Compile-time dependency injection        │   │
│  │ - github.com/go-playground/validator/v10 — Request validation       │   │
│  │ - crypto/sha256 — PhonePe webhook Authorization header verification │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
