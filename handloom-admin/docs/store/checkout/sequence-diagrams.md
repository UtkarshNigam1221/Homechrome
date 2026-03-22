# Store Checkout - Sequence Diagrams

## Overview

This document contains sequence diagrams for the B2C Store Checkout operations including serviceability checks, order placement with payment initiation, and payment webhook processing.

---

## 1. Serviceability Check Sequence

```
┌────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Client │   │CheckoutHndlr │   │ShippingService│   │Shiprocket GW │
└───┬────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘
    │               │                  │                   │
    │ POST          │                  │                   │
    │ /checkout/    │                  │                   │
    │ serviceability│                  │                   │
    │ {pincode:     │                  │                   │
    │  "560001"}    │                  │                   │
    │──────────────▶│                  │                   │
    │               │                  │                   │
    │               │ Validate body    │                   │
    │               │ (len=6)          │                   │
    │               │──────┐           │                   │
    │               │      │           │                   │
    │               │◀─────┘           │                   │
    │               │                  │                   │
    │               │ CheckService-    │                   │
    │               │ ability          │                   │
    │               │ (cID, "560001")  │                   │
    │               │─────────────────▶│                   │
    │               │                  │                   │
    │               │                  │ CheckService-     │
    │               │                  │ ability           │
    │               │                  │ (pickup="560049", │
    │               │                  │  delivery="560001"│
    │               │                  │  weight=500g)     │
    │               │                  │──────────────────▶│
    │               │                  │                   │
    │               │                  │                   │ GET /v1/
    │               │                  │                   │ external/
    │               │                  │                   │ courier/
    │               │                  │                   │ serviceability
    │               │                  │                   │──────┐
    │               │                  │                   │      │
    │               │                  │                   │◀─────┘
    │               │                  │                   │
    │               │                  │ Courier options:  │
    │               │                  │ [{Delhivery,      │
    │               │                  │   5000p, 4d},     │
    │               │                  │  {BlueDart,       │
    │               │                  │   7500p, 3d}]     │
    │               │                  │◀──────────────────│
    │               │                  │                   │
    │               │ ServiceabilityResult                 │
    │               │ {serviceable:true,                   │
    │               │  couriers:[...]}                     │
    │               │◀─────────────────│                   │
    │               │                  │                   │
    │ 200 OK        │                  │                   │
    │ {serviceable, │                  │                   │
    │  couriers}    │                  │                   │
    │◀──────────────│                  │                   │
    │               │                  │                   │
```

---

## 2. Order Placement Sequence

```
┌────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Client │  │Checkout  │  │Checkout  │  │  Cart    │  │Customer  │  │Inventory │  │  Order   │  │ Payment  │
│        │  │Handler   │  │Service   │  │ Service  │  │  Repo    │  │ Service  │  │  Repo    │  │ Service  │
└───┬────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
    │            │             │             │             │             │             │             │
    │ POST       │             │             │             │             │             │             │
    │ /checkout/ │             │             │             │             │             │             │
    │ initiate   │             │             │             │             │             │             │
    │ {address_id│             │             │             │             │             │             │
    │  courier_id│             │             │             │             │             │             │
    │────────────▶             │             │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │ Validate    │             │             │             │             │             │
    │            │ body        │             │             │             │             │             │
    │            │──────┐      │             │             │             │             │             │
    │            │      │      │             │             │             │             │             │
    │            │◀─────┘      │             │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │ Initiate    │             │             │             │             │             │
    │            │ (cID, req)  │             │             │             │             │             │
    │            │────────────▶│             │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ GetCart      │             │             │             │             │
    │            │             │ (customerID) │             │             │             │             │
    │            │             │─────────────▶│             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ CartWithItems│             │             │             │             │
    │            │             │ (2 items,    │             │             │             │             │
    │            │             │  24500 INR)  │             │             │             │             │
    │            │             │◀─────────────│             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Cart empty?  │             │             │             │             │
    │            │             │ No → continue│             │             │             │             │
    │            │             │──────┐       │             │             │             │             │
    │            │             │      │       │             │             │             │             │
    │            │             │◀─────┘       │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ GetByID      │             │             │             │             │
    │            │             │ (customerID) │             │             │             │             │
    │            │             │──────────────────────────▶│             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Customer     │             │             │             │             │
    │            │             │ + addresses  │             │             │             │             │
    │            │             │◀──────────────────────────│             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Find address │             │             │             │             │
    │            │             │ by ID        │             │             │             │             │
    │            │             │──────┐       │             │             │             │             │
    │            │             │      │       │             │             │             │             │
    │            │             │◀─────┘       │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ ── For each cart item ──  │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Check stock  │             │             │             │             │
    │            │             │ (product_id, │             │             │             │             │
    │            │             │  quantity)   │             │             │             │             │
    │            │             │─────────────────────────────────────────▶             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Stock OK     │             │             │             │             │
    │            │             │ (avail=15)   │             │             │             │             │
    │            │             │◀─────────────────────────────────────────             │             │
    │            │             │             │             │             │             │             │
    │            │             │ ── End loop ────────────  │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Build Order  │             │             │             │             │
    │            │             │ entity from  │             │             │             │             │
    │            │             │ cart + addr  │             │             │             │             │
    │            │             │──────┐       │             │             │             │             │
    │            │             │      │       │             │             │             │             │
    │            │             │◀─────┘       │             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Create order │             │             │             │             │
    │            │             │ (PENDING)    │             │             │             │             │
    │            │             │──────────────────────────────────────────────────────▶│             │
    │            │             │             │             │             │             │             │
    │            │             │ Order saved  │             │             │             │             │
    │            │             │◀──────────────────────────────────────────────────────│             │
    │            │             │             │             │             │             │             │
    │            │             │ Reserve      │             │             │             │             │
    │            │             │ inventory    │             │             │             │             │
    │            │             │ (all items)  │             │             │             │             │
    │            │             │─────────────────────────────────────────▶             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Reserved OK  │             │             │             │             │
    │            │             │◀─────────────────────────────────────────             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Initiate     │             │             │             │             │
    │            │             │ payment      │             │             │             │             │
    │            │             │ (order_id,   │             │             │             │             │
    │            │             │  amount,     │             │             │             │             │
    │            │             │  phone)      │             │             │             │             │
    │            │             │──────────────────────────────────────────────────────────────────▶│
    │            │             │             │             │             │             │             │
    │            │             │ PaymentResp  │             │             │             │             │
    │            │             │ {redirect_url│             │             │             │             │
    │            │             │  txn_id}     │             │             │             │             │
    │            │             │◀──────────────────────────────────────────────────────────────────│
    │            │             │             │             │             │             │             │
    │            │             │ ClearCart    │             │             │             │             │
    │            │             │ (customerID) │             │             │             │             │
    │            │             │─────────────▶│             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │             │ Cart cleared │             │             │             │             │
    │            │             │◀─────────────│             │             │             │             │
    │            │             │             │             │             │             │             │
    │            │ CheckoutResult             │             │             │             │             │
    │            │ {order,      │             │             │             │             │             │
    │            │  redirect_url│             │             │             │             │             │
    │            │  txn_id}     │             │             │             │             │             │
    │            │◀────────────│             │             │             │             │             │
    │            │             │             │             │             │             │             │
    │ 201 Created│             │             │             │             │             │             │
    │ {order,    │             │             │             │             │             │             │
    │  redirect} │             │             │             │             │             │             │
    │◀───────────│             │             │             │             │             │             │
    │            │             │             │             │             │             │             │
```

---

## 3. Payment Webhook Sequence

```
┌──────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ PhonePe  │   │WebhookHandler│   │PaymentService│   │  Order Repo  │   │InventoryRepo │
└────┬─────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘
     │                │                  │                   │                   │
     │ POST           │                  │                   │                   │
     │ /webhooks/     │                  │                   │                   │
     │ phonepe        │                  │                   │                   │
     │ Authorization: │                  │                   │                   │
     │  SHA256(u:p)   │                  │                   │                   │
     │ {event,payload}│                  │                   │                   │
     │───────────────▶│                  │                   │                   │
     │                │                  │                   │                   │
     │                │ HandleWebhook    │                   │                   │
     │                │ (body, authHdr)  │                   │                   │
     │                │─────────────────▶│                   │                   │
     │                │                  │                   │                   │
     │                │                  │ Verify auth:       │                   │
     │                │                  │ SHA256(username:   │                   │
     │                │                  │ password) ==       │                   │
     │                │                  │ Authorization hdr  │                   │
     │                │                  │──────┐             │                   │
     │                │                  │      │             │                   │
     │                │                  │◀─────┘             │                   │
     │                │                  │                   │                   │
     │                │                  │ GetByMerchant      │                   │
     │                │                  │ TxnID              │                   │
     │                │                  │ (GSI2 lookup)      │                   │
     │                │                  │──────┐             │                   │
     │                │                  │      │ (internal   │                   │
     │                │                  │      │  repo call) │                   │
     │                │                  │◀─────┘             │                   │
     │                │                  │                   │                   │
     │                │                  │ Payment found:     │                   │
     │                │                  │ status=PENDING     │                   │
     │                │                  │                   │                   │
     │                │                  │ ── Payment SUCCESS path ──            │
     │                │                  │                   │                   │
     │                │                  │ Update Payment    │                   │
     │                │                  │ status=PAID       │                   │
     │                │                  │ completed_at=now  │                   │
     │                │                  │ provider_txn_id   │                   │
     │                │                  │──────┐            │                   │
     │                │                  │      │            │                   │
     │                │                  │◀─────┘            │                   │
     │                │                  │                   │                   │
     │                │                  │ GetByID(orderID)  │                   │
     │                │                  │──────────────────▶│                   │
     │                │                  │                   │                   │
     │                │                  │ Order (PENDING)    │                   │
     │                │                  │◀──────────────────│                   │
     │                │                  │                   │                   │
     │                │                  │ Update Order       │                   │
     │                │                  │ status=CONFIRMED   │                   │
     │                │                  │ payment_status     │                   │
     │                │                  │ =PAID              │                   │
     │                │                  │──────────────────▶│                   │
     │                │                  │                   │                   │
     │                │                  │ Success            │                   │
     │                │                  │◀──────────────────│                   │
     │                │                  │                   │                   │
     │                │                  │ ── Payment FAILED path ──             │
     │                │                  │ (alternative)      │                   │
     │                │                  │                   │                   │
     │                │                  │ Update Payment    │                   │
     │                │                  │ status=FAILED     │                   │
     │                │                  │──────┐            │                   │
     │                │                  │      │            │                   │
     │                │                  │◀─────┘            │                   │
     │                │                  │                   │                   │
     │                │                  │ Update Order       │                   │
     │                │                  │ status=CANCELLED   │                   │
     │                │                  │ payment_status     │                   │
     │                │                  │ =FAILED            │                   │
     │                │                  │──────────────────▶│                   │
     │                │                  │                   │                   │
     │                │                  │ Release inventory  │                   │
     │                │                  │ (for each item)    │                   │
     │                │                  │──────────────────────────────────────▶│
     │                │                  │                   │                   │
     │                │                  │ Stock released     │                   │
     │                │                  │◀──────────────────────────────────────│
     │                │                  │                   │                   │
     │                │                  │ ── End paths ──── │                   │
     │                │                  │                   │                   │
     │                │ nil (success)    │                   │                   │
     │                │◀─────────────────│                   │                   │
     │                │                  │                   │                   │
     │ 200 OK         │                  │                   │                   │
     │◀───────────────│                  │                   │                   │
     │                │                  │                   │                   │
```

---

## 4. Payment Status Polling Sequence

```
┌────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Client │   │CheckoutHndlr │   │CheckoutSvc   │   │  Order Repo  │   │Payment Repo  │
└───┬────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘
    │               │                  │                   │                   │
    │ GET           │                  │                   │                   │
    │ /checkout/    │                  │                   │                   │
    │ payment-status│                  │                   │                   │
    │ /{orderID}    │                  │                   │                   │
    │──────────────▶│                  │                   │                   │
    │               │                  │                   │                   │
    │               │ GetPayment       │                   │                   │
    │               │ Status           │                   │                   │
    │               │ (cID, orderID)   │                   │                   │
    │               │─────────────────▶│                   │                   │
    │               │                  │                   │                   │
    │               │                  │ GetByID           │                   │
    │               │                  │ (orderID)         │                   │
    │               │                  │──────────────────▶│                   │
    │               │                  │                   │                   │
    │               │                  │ Order             │                   │
    │               │                  │◀──────────────────│                   │
    │               │                  │                   │                   │
    │               │                  │ Validate:          │                   │
    │               │                  │ order.customer_id  │                   │
    │               │                  │ == jwt.customer_id │                   │
    │               │                  │──────┐             │                   │
    │               │                  │      │             │                   │
    │               │                  │◀─────┘             │                   │
    │               │                  │                   │                   │
    │               │                  │ GetByOrderID       │                   │
    │               │                  │ (orderID)          │                   │
    │               │                  │──────────────────────────────────────▶│
    │               │                  │                   │                   │
    │               │                  │ Payment            │                   │
    │               │                  │ (status=PAID)      │                   │
    │               │                  │◀──────────────────────────────────────│
    │               │                  │                   │                   │
    │               │ PaymentStatus    │                   │                   │
    │               │ Result           │                   │                   │
    │               │ {payment_status: │                   │                   │
    │               │  "PAID",         │                   │                   │
    │               │  order: {...}}   │                   │                   │
    │               │◀─────────────────│                   │                   │
    │               │                  │                   │                   │
    │ 200 OK        │                  │                   │                   │
    │ {payment_     │                  │                   │                   │
    │  status,      │                  │                   │                   │
    │  order}       │                  │                   │                   │
    │◀──────────────│                  │                   │                   │
    │               │                  │                   │                   │
```
