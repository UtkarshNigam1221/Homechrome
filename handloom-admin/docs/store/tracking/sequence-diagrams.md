# Store Tracking - Sequence Diagrams

## Overview
This document contains the sequence diagram for the B2C store public order tracking operation.

---

## 1. Track Order by Order Number

```
┌────────┐   ┌──────────┐   ┌──────────────┐   ┌──────────┐   ┌────────────┐
│ Client │   │ Rate     │   │  Tracking    │   │ DynamoDB │   │ Shiprocket │
│        │   │ Limiter  │   │  Handler     │   │          │   │ Gateway    │
└───┬────┘   └────┬─────┘   └──────┬───────┘   └────┬─────┘   └─────┬──────┘
    │             │                │                 │               │
    │ GET /api/v1/store/track/    │                 │               │
    │ ORD-2026-000123             │                 │               │
    │────────────▶│                │                 │               │
    │             │                │                 │               │
    │             │ Check rate     │                 │               │
    │             │ limit (30/min) │                 │               │
    │             │──────┐         │                 │               │
    │             │      │         │                 │               │
    │             │◀─────┘         │                 │               │
    │             │                │                 │               │
    │             │ Forward        │                 │               │
    │             │───────────────▶│                 │               │
    │             │                │                 │               │
    │             │                │ Lookup order    │               │
    │             │                │ by number:      │               │
    │             │                │ Scan where      │               │
    │             │                │ OrderNumber =   │               │
    │             │                │ ORD-2026-000123 │               │
    │             │                │────────────────▶│               │
    │             │                │                 │               │
    │             │                │ Order record    │               │
    │             │                │ (ID, Status,    │               │
    │             │                │  TrackingNumber,│               │
    │             │                │  ShippingCarrier│               │
    │             │                │  CreatedAt...)  │               │
    │             │                │◀────────────────│               │
    │             │                │                 │               │
    │             │                │ Query status    │               │
    │             │                │ history:        │               │
    │             │                │ PK=ORDER#{id}   │               │
    │             │                │ SK begins_with  │               │
    │             │                │ STATUS#         │               │
    │             │                │────────────────▶│               │
    │             │                │                 │               │
    │             │                │ Status history  │               │
    │             │                │ records         │               │
    │             │                │◀────────────────│               │
    │             │                │                 │               │
    │             │                │ Build status    │               │
    │             │                │ timeline        │               │
    │             │                │──────┐          │               │
    │             │                │      │          │               │
    │             │                │◀─────┘          │               │
    │             │                │                 │               │
    │             │                │ Has AWB?        │               │
    │             │                │──────┐          │               │
    │             │                │      │ Yes      │               │
    │             │                │◀─────┘          │               │
    │             │                │                 │               │
    │             │                │ Track by AWB:   │               │
    │             │                │ SR12345678901   │               │
    │             │                │────────────────────────────────▶│
    │             │                │                 │               │
    │             │                │                 │  Shiprocket   │
    │             │                │                 │  API call     │
    │             │                │                 │               │
    │             │                │ Shipment status:│               │
    │             │                │ IN TRANSIT,     │               │
    │             │                │ ETD: 2026-02-20 │               │
    │             │                │◀────────────────────────────────│
    │             │                │                 │               │
    │             │                │ Build response: │               │
    │             │                │ - order_number  │               │
    │             │                │ - status        │               │
    │             │                │ - timeline[]    │               │
    │             │                │ - shipment{}    │               │
    │             │                │──────┐          │               │
    │             │                │      │          │               │
    │             │                │◀─────┘          │               │
    │             │                │                 │               │
    │ 200 OK                      │                 │               │
    │ {order_number, status,      │                 │               │
    │  status_history, shipment}  │                 │               │
    │◀────────────────────────────│                 │               │
    │             │                │                 │               │
```

---

## 2. Track Order - Shiprocket Unavailable (Fallback)

```
┌────────┐   ┌──────────────┐   ┌──────────┐   ┌────────────┐
│ Client │   │  Tracking    │   │ DynamoDB │   │ Shiprocket │
│        │   │  Handler     │   │          │   │ Gateway    │
└───┬────┘   └──────┬───────┘   └────┬─────┘   └─────┬──────┘
    │               │                │               │
    │ GET /api/v1/store/track/      │               │
    │ ORD-2026-000123               │               │
    │──────────────▶│                │               │
    │               │                │               │
    │               │ Lookup order   │               │
    │               │───────────────▶│               │
    │               │                │               │
    │               │ Order record   │               │
    │               │◀───────────────│               │
    │               │                │               │
    │               │ Query status   │               │
    │               │ history        │               │
    │               │───────────────▶│               │
    │               │                │               │
    │               │ History records│               │
    │               │◀───────────────│               │
    │               │                │               │
    │               │ Track AWB      │               │
    │               │───────────────────────────────▶│
    │               │                │               │
    │               │                │    TIMEOUT /  │
    │               │                │    5xx Error  │
    │               │◀───────────────────────────────│
    │               │                │               │
    │               │ Fallback:      │               │
    │               │ build shipment │               │
    │               │ from order     │               │
    │               │ record only    │               │
    │               │ (no ETD)       │               │
    │               │──────┐         │               │
    │               │      │         │               │
    │               │◀─────┘         │               │
    │               │                │               │
    │ 200 OK                        │               │
    │ {shipment: {awb, courier,     │               │
    │  status: "SHIPPED",           │               │
    │  estimated_delivery: null}}   │               │
    │◀──────────────│                │               │
    │               │                │               │
```
