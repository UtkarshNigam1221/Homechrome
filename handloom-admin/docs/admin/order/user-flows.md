# Order Lambda - User Flows

## Overview
This document describes the user flows for the Order Lambda service, covering Order and Customer management.

---

## 1. Create Order Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            CREATE ORDER FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Orders > New    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Search/Select   │◀────────────────┐
│ Customer        │                  │
└────────┬────────┘                  │
         │                           │
         ├── New Customer ───────────┤
         │                           │
         ▼                           │
┌─────────────────┐         ┌────────┴────────┐
│ Add Order Items:│         │ Create Customer │
│ - Search product│         │ (modal)         │
│ - Select qty    │         └─────────────────┘
│ - View price    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Apply Coupon    │
│ (optional)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Set Shipping    │
│ Address         │
│ (from customer  │
│  or new)        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Review Order:   │
│ - Items         │
│ - Subtotal      │
│ - Discount      │
│ - Tax           │
│ - Total         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Confirm Order   │────▶│ Reserve         │
│                 │     │ Inventory       │
└─────────────────┘     └────────┬────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │ Stock         │         │ Out of Stock  │
           │ Available     │         │ Error         │
           └───────┬───────┘         └───────┬───────┘
                   │                         │
                   ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │ Create Order  │         │ Show error    │
           │ Record        │         │ Remove item   │
           └───────┬───────┘         └───────────────┘
                   │
                   ▼
           ┌───────────────┐
           │ Send Order    │
           │ Confirmation  │
           │ Email         │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Redirect to   │
           │ Order Details │
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## 2. Order Status Update Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ORDER STATUS UPDATE FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ View Order      │
│ Details         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Update   │
│ Status"         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select New      │
│ Status:         │
│ ┌─────────────┐ │
│ │ ○ Confirmed │ │
│ │ ○ Processing│ │
│ │ ○ Shipped   │ │
│ │ ○ Delivered │ │
│ │ ○ Cancelled │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├────── Shipped ─────────┐
         │                        │
         ├────── Cancelled ───────┤
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌─────────────────┐
│ STATUS: SHIPPED │     │ STATUS: CANCEL  │
│ Enter:          │     │ Enter:          │
│ - Tracking #    │     │ - Reason        │
│ - Carrier       │     │ - Refund?       │
└────────┬────────┘     └────────┬────────┘
         │                       │
         └───────────┬───────────┘
                     │
                     ▼
             ┌───────────────┐
             │ Validate      │
             │ Transition    │
             └───────┬───────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
┌───────────────┐         ┌───────────────┐
│   VALID       │         │   INVALID     │
└───────┬───────┘         └───────┬───────┘
        │                         │
        ▼                         ▼
┌───────────────┐         ┌───────────────┐
│ Update Order  │         │ Show error    │
│ Status        │         │ "Cannot       │
│               │         │ transition"   │
└───────┬───────┘         └───────────────┘
        │
        ▼
┌───────────────┐
│ Send Customer │
│ Notification  │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│ Add Timeline  │
│ Entry         │
└───────┬───────┘
        │
        ▼
   ┌────────┐
   │  END   │
   └────────┘
```

---

## 3. Order Cancellation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ORDER CANCELLATION FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ View Order      │
│ Details         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Cancel   │
│ Order"          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check if        │
│ cancellable:    │
│ - Not shipped   │
│ - Not delivered │
└────────┬────────┘
         │
         ├── Not Allowed ─────────┐
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌─────────────────┐
│ Enter Cancel    │     │ Show error      │
│ Reason          │     │ "Order cannot   │
│ ┌─────────────┐ │     │ be cancelled"   │
│ │Customer req │ │     └─────────────────┘
│ │Out of stock │ │
│ │Payment fail │ │
│ │Other        │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Process Refund? │
│ ┌─────────────┐ │
│ │ ○ Yes       │ │
│ │ ○ No        │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├── Yes ─────────────────┐
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌─────────────────┐
│ Continue        │     │ Initiate Refund │
│ without refund  │     │ Process         │
└────────┬────────┘     └────────┬────────┘
         │                       │
         └───────────┬───────────┘
                     │
                     ▼
             ┌───────────────┐
             │ Release       │
             │ Inventory     │
             │ Reservation   │
             └───────┬───────┘
                     │
                     ▼
             ┌───────────────┐
             │ Update Order  │
             │ Status to     │
             │ CANCELLED     │
             └───────┬───────┘
                     │
                     ▼
             ┌───────────────┐
             │ Send Cancel   │
             │ Notification  │
             │ to Customer   │
             └───────┬───────┘
                     │
                     ▼
                ┌────────┐
                │  END   │
                └────────┘
```

---

## 4. Customer Search Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CUSTOMER SEARCH FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Customers       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Enter Search    │
│ Query:          │
│ - Name          │
│ - Email         │
│ - Phone         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Debounce        │
│ (300ms)         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Search API      │
│ Call            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display         │
│ Results         │
└────────┬────────┘
         │
         ├── No Results ──────────┐
         │                        │
         ├── Has Results ─────────┤
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌─────────────────┐
│ Show customer   │     │ Show "No        │
│ cards with:     │     │ customers       │
│ - Name          │     │ found"          │
│ - Email         │     │                 │
│ - Order count   │     │ Option to       │
│ - Total spent   │     │ create new      │
└────────┬────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Click Customer  │
│ to view details │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 5. Order List Filter Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ORDER LIST FILTER FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ View Orders     │
│ List            │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ Apply Filters:                                           │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │
│ │ Status:     │ │ Date Range: │ │ Customer:   │        │
│ │ ☐ All       │ │ From: _____ │ │ Search...   │        │
│ │ ☐ Pending   │ │ To: _______ │ └─────────────┘        │
│ │ ☐ Confirmed │ └─────────────┘                        │
│ │ ☐ Shipped   │ ┌─────────────┐ ┌─────────────┐        │
│ │ ☐ Delivered │ │ Payment:    │ │ Sort By:    │        │
│ │ ☐ Cancelled │ │ ☐ All       │ │ ○ Newest    │        │
│ └─────────────┘ │ ☐ Paid      │ │ ○ Oldest    │        │
│                 │ ☐ Pending   │ │ ○ Total $↑  │        │
│                 │ ☐ Failed    │ │ ○ Total $↓  │        │
│                 └─────────────┘ └─────────────┘        │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Build Query     │
│ String          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Update URL      │
│ (for sharing)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fetch Filtered  │
│ Orders          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display with    │
│ Pagination      │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## State Diagram - Order Lifecycle

```
                                    ┌─────────────────┐
                                    │     PENDING     │
                                    │  (Just Created) │
                                    └────────┬────────┘
                                             │
                                    Payment Confirmed
                                             │
                                             ▼
     ┌───────────┐              ┌─────────────────┐
     │ CANCELLED │◀─────────────│    CONFIRMED    │
     │           │   Cancel     │(Payment Received)│
     └───────────┘              └────────┬────────┘
           ▲                             │
           │                      Start Processing
           │                             │
           │                             ▼
           │                    ┌─────────────────┐
           └────────────────────│   PROCESSING    │
                    Cancel      │ (Being Prepared)│
                                └────────┬────────┘
                                         │
                                    Ship Order
                                         │
                                         ▼
                                ┌─────────────────┐
                                │     SHIPPED     │
                                │ (In Transit)    │
                                └────────┬────────┘
                                         │
                                    Deliver
                                         │
                                         ▼
                                ┌─────────────────┐
                                │    DELIVERED    │
                                │   (Complete)    │
                                └─────────────────┘
```
