# Store Webhooks - User Flows

## Overview
This document describes the webhook processing flows for external service callbacks. These are system-to-system flows rather than user-initiated flows, but they directly impact the customer experience by updating order and payment statuses.

---

## 1. Payment Success (PhonePe Callback)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PAYMENT SUCCESS FLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────────┐
    │  PhonePe sends   │
    │  callback after  │
    │  UPI payment     │
    │  completes       │
    └────────┬─────────┘
             │
             ▼
    ┌─────────────────┐
    │ POST /webhooks/ │
    │ phonepe         │
    │ Authorization:  │
    │ SHA256(u:p)     │
    │ {event,payload} │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Verify auth:    │
    │ SHA256(username: │
    │ password) ==    │
    │ Authorization   │
    └────────┬────────┘
             │
             ├── Invalid Auth ───────┐
             │                       │
             ▼                       ▼
    ┌─────────────────┐    ┌─────────────────┐
    │ Parse JSON      │    │ Log warning:    │
    │ event payload   │    │ "Auth           │
    │                 │    │ mismatch"       │
    └────────┬────────┘    │                 │
             │             │ Return 200 OK   │
             ▼             └─────────────────┘
    ┌─────────────────┐
    │ Extract:        │
    │ - event:        │
    │   checkout.order│
    │   .completed    │
    │ - merchantOrder │
    │   Id, amount    │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Lookup payment  │
    │ by merchantOrder│
    │ Id              │
    └────────┬────────┘
             │
             ├── Already Processed ──┐
             │                       │
             ▼                       ▼
    ┌─────────────────┐    ┌─────────────────┐
    │ Update payment  │    │ Skip (idempotent│
    │ status:         │    │ duplicate)      │
    │ PENDING →       │    │ Return 200 OK   │
    │ SUCCESS         │    └─────────────────┘
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Update order    │
    │ status:         │
    │ PENDING →       │
    │ CONFIRMED       │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Write status    │
    │ history record: │
    │ PENDING →       │
    │ CONFIRMED       │
    │ "Payment        │
    │  received"      │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Send SMS to     │
    │ customer:       │
    │ "Your order     │
    │ ORD-2026-000123 │
    │ is confirmed!"  │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Return 200 OK   │
    │ {status: "ok"}  │
    └────────┬────────┘
             │
             ▼
        ┌────────┐
        │  END   │
        └────────┘
```

---

## 2. Payment Failure (PhonePe Callback)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PAYMENT FAILURE FLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────────┐
    │  PhonePe sends   │
    │  callback after  │
    │  payment fails   │
    │  or is declined  │
    └────────┬─────────┘
             │
             ▼
    ┌─────────────────┐
    │ POST /webhooks/ │
    │ phonepe         │
    │ Authorization:  │
    │ SHA256(u:p)     │
    │ {event,payload} │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Verify auth     │
    │ header          │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Parse JSON      │
    │ event payload   │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Extract:        │
    │ - event:        │
    │   checkout.order│
    │   .failed       │
    │ - merchantOrder │
    │   Id            │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Lookup payment  │
    │ by merchantOrder│
    │ Id              │
    └────────┬────────┘
             │
             ├── Already Processed ──┐
             │                       │
             ▼                       ▼
    ┌─────────────────┐    ┌─────────────────┐
    │ Update payment  │    │ Skip (idempotent│
    │ status:         │    │ duplicate)      │
    │ PENDING →       │    │ Return 200 OK   │
    │ FAILED          │    └─────────────────┘
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Release         │
    │ reserved        │
    │ inventory for   │
    │ all order items │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Send SMS to     │
    │ customer:       │
    │ "Payment failed │
    │ for order       │
    │ ORD-2026-000123.│
    │ Please retry."  │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Return 200 OK   │
    │ {status: "ok"}  │
    └────────┬────────┘
             │
             ▼
        ┌────────┐
        │  END   │
        └────────┘
```

---

## 3. Shipping Update (Shiprocket Callback)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     SHIPPING UPDATE FLOW                                     │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────────┐
    │  Shiprocket sends│
    │  webhook on      │
    │  status change   │
    └────────┬─────────┘
             │
             ▼
    ┌─────────────────┐
    │ POST /webhooks/ │
    │ shiprocket      │
    │ {awb, status,   │
    │  order_id, etd} │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Parse payload:  │
    │ - awb           │
    │ - current_status│
    │ - order_id      │
    │ - etd           │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │ Lookup order by │
    │ order_id (order │
    │ number)         │
    └────────┬────────┘
             │
             ├── Not Found ──────────┐
             │                       │
             ▼                       ▼
    ┌─────────────────┐    ┌─────────────────┐
    │ Update shipment │    │ Log warning:    │
    │ record:         │    │ "Order not      │
    │ - status        │    │ found for       │
    │ - etd           │    │ webhook"        │
    │ - updated_at    │    │ Return 200 OK   │
    └────────┬────────┘    └─────────────────┘
             │
             ▼
    ┌─────────────────┐
    │ Map Shiprocket  │
    │ status          │
    └────────┬────────┘
             │
    ┌────────┼─────────────────┬──────────────────┐
    │        │                 │                   │
    │        ▼                 ▼                   ▼
    │  ┌───────────┐    ┌───────────┐    ┌──────────────┐
    │  │ DELIVERED  │    │ RTO       │    │ Other:       │
    │  │           │    │ Delivered │    │ In Transit,  │
    │  └─────┬─────┘    └─────┬─────┘    │ Out For Dlvr │
    │        │                │          │ Picked Up    │
    │        ▼                ▼          └──────┬───────┘
    │  ┌───────────┐    ┌───────────┐          │
    │  │ Update    │    │ Update    │          ▼
    │  │ order     │    │ order     │    ┌───────────┐
    │  │ status:   │    │ status:   │    │ Log and   │
    │  │ SHIPPED → │    │ SHIPPED → │    │ update    │
    │  │ DELIVERED │    │ RETURNED  │    │ shipment  │
    │  └─────┬─────┘    └─────┬─────┘    │ only      │
    │        │                │          └─────┬─────┘
    │        ▼                ▼                │
    │  ┌───────────┐    ┌───────────┐          │
    │  │ Set       │    │ Initiate  │          │
    │  │ DeliveredAt│   │ refund if │          │
    │  │ timestamp │    │ PAID      │          │
    │  └─────┬─────┘    └─────┬─────┘          │
    │        │                │                │
    │        ▼                ▼                │
    │  ┌───────────┐    ┌───────────┐          │
    │  │ Update    │    │ Send RTO  │          │
    │  │ customer  │    │ SMS to    │          │
    │  │ stats:    │    │ customer  │          │
    │  │ +1 order  │    └─────┬─────┘          │
    │  │ +amount   │          │                │
    │  └─────┬─────┘          │                │
    │        │                │                │
    │        ▼                │                │
    │  ┌───────────┐          │                │
    │  │ Send SMS: │          │                │
    │  │ "Order    │          │                │
    │  │ delivered"│          │                │
    │  └─────┬─────┘          │                │
    │        │                │                │
    └────────┼────────────────┼────────────────┘
             │                │
             └───────┬────────┘
                     │
                     ▼
            ┌─────────────────┐
            │ Return 200 OK   │
            │ {status: "ok"}  │
            └────────┬────────┘
                     │
                     ▼
                ┌────────┐
                │  END   │
                └────────┘
```
