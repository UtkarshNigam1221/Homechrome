# Store Orders - Sequence Diagrams

## Overview
This document contains sequence diagrams for the B2C store order management operations, covering order listing and cancellation flows.

---

## 1. List Customer Orders

```
┌────────┐   ┌───────────┐   ┌───────────────┐   ┌──────────┐
│ Client │   │ Customer  │   │  StoreOrder   │   │ DynamoDB │
│        │   │ Auth MW   │   │  Handler      │   │          │
└───┬────┘   └─────┬─────┘   └──────┬────────┘   └────┬─────┘
    │              │                │                  │
    │ GET /api/v1/store/orders     │                  │
    │ ?limit=20&cursor=abc         │                  │
    │ Cookie: store_token=xxx      │                  │
    │─────────────▶│                │                  │
    │              │                │                  │
    │              │ Validate JWT   │                  │
    │              │──────┐         │                  │
    │              │      │         │                  │
    │              │◀─────┘         │                  │
    │              │                │                  │
    │              │ Set customer_id│                  │
    │              │ in context     │                  │
    │              │───────────────▶│                  │
    │              │                │                  │
    │              │                │ Extract          │
    │              │                │ customer_id      │
    │              │                │ from context     │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │              │                │ Query GSI1:      │
    │              │                │ GSI1PK=CUSTOMER# │
    │              │                │ {customer_id}    │
    │              │                │ Limit=20         │
    │              │                │ ScanForward=false│
    │              │                │─────────────────▶│
    │              │                │                  │
    │              │                │ Orders + LastKey  │
    │              │                │◀─────────────────│
    │              │                │                  │
    │              │                │ Build response:  │
    │              │                │ - Strip internal │
    │              │                │   fields         │
    │              │                │ - Encode cursor  │
    │              │                │ - Set has_more   │
    │              │                │──────┐           │
    │              │                │      │           │
    │              │                │◀─────┘           │
    │              │                │                  │
    │ 200 OK                       │                  │
    │ {data: [...], meta: {...}}   │                  │
    │◀─────────────────────────────│                  │
    │              │                │                  │
```

---

## 2. Cancel Order

```
┌────────┐   ┌───────────┐   ┌───────────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐
│ Client │   │ Customer  │   │  StoreOrder   │   │  Order    │   │ DynamoDB │   │ Inventory │
│        │   │ Auth MW   │   │  Handler      │   │  Service  │   │          │   │ Service   │
└───┬────┘   └─────┬─────┘   └──────┬────────┘   └─────┬─────┘   └────┬─────┘   └─────┬─────┘
    │              │                │                   │              │               │
    │ POST /api/v1/store/          │                   │              │               │
    │ orders/{id}/cancel           │                   │              │               │
    │ Cookie: store_token=xxx      │                   │              │               │
    │─────────────▶│                │                   │              │               │
    │              │                │                   │              │               │
    │              │ Validate JWT   │                   │              │               │
    │              │──────┐         │                   │              │               │
    │              │      │         │                   │              │               │
    │              │◀─────┘         │                   │              │               │
    │              │                │                   │              │               │
    │              │ Forward with   │                   │              │               │
    │              │ customer_id    │                   │              │               │
    │              │───────────────▶│                   │              │               │
    │              │                │                   │              │               │
    │              │                │ Get order by ID   │              │               │
    │              │                │──────────────────▶│              │               │
    │              │                │                   │              │               │
    │              │                │                   │ Fetch order  │               │
    │              │                │                   │─────────────▶│               │
    │              │                │                   │              │               │
    │              │                │                   │ Order record │               │
    │              │                │                   │◀─────────────│               │
    │              │                │                   │              │               │
    │              │                │ Order data        │              │               │
    │              │                │◀──────────────────│              │               │
    │              │                │                   │              │               │
    │              │                │ Validate:         │              │               │
    │              │                │ - customer owns   │              │               │
    │              │                │   order?          │              │               │
    │              │                │ - cancellable     │              │               │
    │              │                │   status?         │              │               │
    │              │                │──────┐            │              │               │
    │              │                │      │            │              │               │
    │              │                │◀─────┘            │              │               │
    │              │                │                   │              │               │
    │              │                │ CancelOrder(id)   │              │               │
    │              │                │──────────────────▶│              │               │
    │              │                │                   │              │               │
    │              │                │                   │ Update order │               │
    │              │                │                   │ status to    │               │
    │              │                │                   │ CANCELLED    │               │
    │              │                │                   │─────────────▶│               │
    │              │                │                   │              │               │
    │              │                │                   │ Success      │               │
    │              │                │                   │◀─────────────│               │
    │              │                │                   │              │               │
    │              │                │                   │ Write status │               │
    │              │                │                   │ history      │               │
    │              │                │                   │─────────────▶│               │
    │              │                │                   │              │               │
    │              │                │                   │ Release      │               │
    │              │                │                   │ inventory    │               │
    │              │                │                   │──────────────────────────────▶│
    │              │                │                   │              │               │
    │              │                │                   │ Stock        │               │
    │              │                │                   │ released     │               │
    │              │                │                   │◀──────────────────────────────│
    │              │                │                   │              │               │
    │              │                │ Cancel success    │              │               │
    │              │                │◀──────────────────│              │               │
    │              │                │                   │              │               │
    │ 200 OK                       │                   │              │               │
    │ {message: "Order cancelled"} │                   │              │               │
    │◀─────────────────────────────│                   │              │               │
    │              │                │                   │              │               │
```
