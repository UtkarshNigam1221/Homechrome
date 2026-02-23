# Store Orders - User Flows

## Overview
This document describes the user flows for the B2C store order history module, covering how customers view their orders and cancel eligible orders.

---

## 1. View Order History

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          VIEW ORDER HISTORY FLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ "My Orders"     │
│ page            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Authenticate    │
│ via Customer    │
│ JWT cookie      │
└────────┬────────┘
         │
         ├── Not Authenticated ──────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Fetch orders:   │        │ Redirect to     │
│ GET /store/     │        │ Login page      │
│ orders?limit=20 │        └─────────────────┘
└────────┬────────┘
         │
         ├── Empty List ─────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Display orders  │        │ Show "No orders │
│ as cards:       │        │ yet" with       │
│ - Order number  │        │ "Start Shopping"│
│ - Date placed   │        │ CTA button      │
│ - Status badge  │        └─────────────────┘
│ - Item count    │
│ - Total amount  │
│ - First item    │
│   thumbnail     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Has more pages? │
└────────┬────────┘
         │
         ├── Yes ────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Done browsing   │        │ Click "Load     │
│                 │        │ More" button    │
└────────┬────────┘        └────────┬────────┘
         │                          │
         │                          ▼
         │                 ┌─────────────────┐
         │                 │ Fetch next page │
         │                 │ with cursor     │
         │                 └────────┬────────┘
         │                          │
         │                          ▼
         │                 ┌─────────────────┐
         │                 │ Append orders   │
         │                 │ to list         │
         │                 └────────┬────────┘
         │                          │
         │◀─────────────────────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 2. View Order Detail

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          VIEW ORDER DETAIL FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Click on order  │
│ card from order │
│ list            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fetch order:    │
│ GET /store/     │
│ orders/{id}     │
└────────┬────────┘
         │
         ├── 404 Not Found ──────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Display order   │        │ Show "Order not │
│ detail page:    │        │ found" error    │
│                 │        │ with back link  │
│ ┌─────────────┐ │        └─────────────────┘
│ │ Order Info  │ │
│ │ - Number    │ │
│ │ - Date      │ │
│ │ - Status    │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Items List  │ │
│ │ - Image     │ │
│ │ - Name/SKU  │ │
│ │ - Qty/Price │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Price       │ │
│ │ Summary     │ │
│ │ - Subtotal  │ │
│ │ - Discount  │ │
│ │ - Tax       │ │
│ │ - Shipping  │ │
│ │ - Total     │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Shipping    │ │
│ │ Address     │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Tracking    │ │
│ │ Info (if    │ │
│ │ shipped)    │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├── Cancellable? (PENDING/CONFIRMED) ──┐
         │                                       │
         ▼                                       ▼
┌─────────────────┐                    ┌─────────────────┐
│ No cancel       │                    │ Show "Cancel    │
│ button shown    │                    │ Order" button   │
└────────┬────────┘                    └────────┬────────┘
         │                                      │
         └──────────────┬───────────────────────┘
                        │
                        ▼
                   ┌────────┐
                   │  END   │
                   └────────┘
```

---

## 3. Cancel Order

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CANCEL ORDER FLOW                                  │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ View Order      │
│ Detail page     │
│ (PENDING or     │
│  CONFIRMED)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Click "Cancel   │
│ Order" button   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show            │
│ confirmation    │
│ dialog:         │
│ ┌─────────────┐ │
│ │ "Are you    │ │
│ │ sure you    │ │
│ │ want to     │ │
│ │ cancel?"    │ │
│ │             │ │
│ │ [No] [Yes]  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ├── No ─────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ POST /store/    │        │ Close dialog,   │
│ orders/{id}/    │        │ stay on detail  │
│ cancel          │        │ page            │
└────────┬────────┘        └─────────────────┘
         │
         ├── Success ────────────────┐
         │                           │
         ├── Error ──────────────────┤
         │                           │
         ▼                           │
┌─────────────────┐                  │
│ Show success    │                  │
│ toast:          │                  │
│ "Order          │                  │
│ cancelled"      │                  │
└────────┬────────┘                  │
         │                           │
         ▼                           ▼
┌─────────────────┐        ┌─────────────────┐
│ Update order    │        │ Show error      │
│ detail to show  │        │ toast:          │
│ CANCELLED       │        │ "Order cannot   │
│ status          │        │ be cancelled"   │
└────────┬────────┘        └─────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ Backend Side Effects (transparent to user):               │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │
│ │ Update      │ │ Release     │ │ Send cancel │        │
│ │ order       │ │ reserved    │ │ SMS to      │        │
│ │ status to   │ │ inventory   │ │ customer    │        │
│ │ CANCELLED   │ │ for items   │ │ phone       │        │
│ └─────────────┘ └─────────────┘ └─────────────┘        │
│                                                          │
│ If payment was PAID:                                     │
│ ┌─────────────┐                                         │
│ │ Initiate    │                                         │
│ │ refund via  │                                         │
│ │ PhonePe     │                                         │
│ └─────────────┘                                         │
└─────────────────────────────────────────────────────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```
