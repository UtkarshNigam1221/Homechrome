# Store Tracking - User Flows

## Overview
This document describes the user flow for the B2C store public order tracking feature, which allows anyone with an order number to check the order status.

---

## 1. Track Order

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TRACK ORDER FLOW                                   │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ├── Via SMS Link ───────────┐
         │                           │
         ├── Via Website ────────────┤
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Navigate to     │        │ Click tracking  │
│ Track Order     │        │ link from SMS:  │
│ page            │        │ homechrome.     │
└────────┬────────┘        │ homechrome.in/     │
         │                 │ track/ORD-2026- │
         │                 │ 000123          │
         │                 └────────┬────────┘
         │                          │
         ▼                          │
┌─────────────────┐                 │
│ Enter order     │                 │
│ number:         │                 │
│ ┌─────────────┐ │                 │
│ │ORD-2026-   │ │                 │
│ │000123      │ │                 │
│ └─────────────┘ │                 │
│ [Track Order]   │                 │
└────────┬────────┘                 │
         │                          │
         └──────────┬───────────────┘
                    │
                    ▼
           ┌───────────────┐
           │ GET /store/   │
           │ track/ORD-    │
           │ 2026-000123   │
           └───────┬───────┘
                   │
      ┌────────────┴────────────┐
      │                         │
      ▼                         ▼
┌───────────────┐         ┌───────────────┐
│ Order Found   │         │ 404 Not Found │
└───────┬───────┘         └───────┬───────┘
        │                         │
        │                         ▼
        │                 ┌───────────────┐
        │                 │ Show error:   │
        │                 │ "Order not    │
        │                 │ found. Please │
        │                 │ check the     │
        │                 │ order number  │
        │                 │ and try again"│
        │                 └───────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│ Display Tracking Page:                                    │
│                                                           │
│ ┌───────────────────────────────────────────────────────┐│
│ │  Order Number: ORD-2026-000123                        ││
│ │  Status: SHIPPED                                      ││
│ └───────────────────────────────────────────────────────┘│
│                                                           │
│ ┌───────────────────────────────────────────────────────┐│
│ │  Status Timeline:                                     ││
│ │                                                       ││
│ │  ● Order Placed         15 Feb 2026, 4:00 PM         ││
│ │  │                                                    ││
│ │  ● Payment Received     15 Feb 2026, 4:05 PM         ││
│ │  │                                                    ││
│ │  ● Shipped              17 Feb 2026, 7:50 PM         ││
│ │  │  via Delhivery                                     ││
│ │  │                                                    ││
│ │  ○ Delivered (pending)                                ││
│ │     Est: 20 Feb 2026                                  ││
│ └───────────────────────────────────────────────────────┘│
│                                                           │
│ ┌───────────────────────────────────────────────────────┐│
│ │  Shipment Details:                                    ││
│ │  Courier: Delhivery                                   ││
│ │  AWB: SR12345678901                                   ││
│ │  Status: IN TRANSIT                                   ││
│ │  [Track on Delhivery →]                               ││
│ └───────────────────────────────────────────────────────┘│
│                                                           │
└─────────────────────────────────────────────────────────┘
         │
         ├── Has tracking URL ───────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Done            │        │ Click "Track on │
│                 │        │ Delhivery"      │
│                 │        │ → opens courier │
│                 │        │   tracking page │
│                 │        │   in new tab    │
└────────┬────────┘        └────────┬────────┘
         │                          │
         └──────────┬───────────────┘
                    │
                    ▼
               ┌────────┐
               │  END   │
               └────────┘
```
