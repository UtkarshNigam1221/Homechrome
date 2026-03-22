# Store Checkout - User Flows

## Overview

This document describes the user flows for the B2C Store Checkout service, covering delivery serviceability checks, the checkout and payment flow, and payment completion via webhook.

---

## 1. Serviceability Check Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       SERVICEABILITY CHECK FLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer enters │
│ delivery pincode│
│ (e.g. 560001)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Validate pincode│
│ - Required      │
│ - Exactly 6     │
│   digits        │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Valid format  │  │ Invalid format│
│               │  │ Return 400    │
│               │  │ VALIDATION_   │
│               │  │ ERROR         │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Call Shiprocket  │
│ serviceability  │
│ API:            │
│ - pickup_pincode│
│   (from config) │
│ - delivery_     │
│   pincode       │
│ - weight (est.) │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Serviceable   │  │ Not           │
│               │  │ serviceable   │
└───────┬───────┘  └───────┬───────┘
        │                  │
        ▼                  ▼
┌─────────────────┐  ┌─────────────────┐
│ Show courier    │  │ Show message:   │
│ options:        │  │ "Delivery not   │
│ ┌─────────────┐ │  │ available to    │
│ │ Delhivery   │ │  │ this pincode"   │
│ │ Rs 50, 4 d  │ │  └────────┬────────┘
│ ├─────────────┤ │           │
│ │ BlueDart    │ │           ▼
│ │ Rs 75, 3 d  │ │      ┌────────┐
│ ├─────────────┤ │      │  END   │
│ │ DTDC        │ │      └────────┘
│ │ Rs 45, 6 d  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Customer selects│
│ preferred       │
│ courier         │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 2. Checkout Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CHECKOUT FLOW                                       │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Proceed to     │
│ Checkout" from  │
│ cart page       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select shipping │
│ address:        │
│ ┌─────────────┐ │
│ │ ● Home      │ │
│ │   42 MG Road│ │
│ │   Bengaluru │ │
│ │   560038    │ │
│ ├─────────────┤ │
│ │ ○ Office    │ │
│ │   10 HSR Lay│ │
│ │   Bengaluru │ │
│ │   560102    │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check delivery  │
│ serviceability  │
│ for address     │
│ pincode         │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Serviceable   │  │ Not           │
│               │  │ serviceable   │
└───────┬───────┘  └───────┬───────┘
        │                  │
        │                  ▼
        │          ┌───────────────┐
        │          │ Show error:   │
        │          │ "Delivery not │
        │          │ available"    │
        │          │ Prompt to     │
        │          │ change address│
        │          └───────────────┘
        │
        ▼
┌─────────────────┐
│ Show order      │
│ summary:        │
│ ┌─────────────┐ │
│ │Items     2  │ │
│ │Subtotal     │ │
│ │  Rs 24,500  │ │
│ │Shipping     │ │
│ │  Rs 50      │ │
│ │─────────────│ │
│ │Total        │ │
│ │  Rs 24,550  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Place Order &  │
│ Pay"            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ POST /checkout/ │
│ initiate        │
│ {address_id,    │
│  courier_id}    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Server-side     │
│ pipeline:       │
│ 1. Validate cart│
│ 2. Check stock  │
│ 3. Create order │
│ 4. Reserve stock│
│ 5. Init payment │
│ 6. Clear cart   │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Success       │  │ Error         │
│ Got redirect  │  │ (stock, addr, │
│ URL           │  │  payment)     │
└───────┬───────┘  └───────┬───────┘
        │                  │
        │                  ▼
        │          ┌───────────────┐
        │          │ Show error    │
        │          │ message to    │
        │          │ customer      │
        │          └───────────────┘
        │
        ▼
┌─────────────────┐
│ Redirect to     │
│ PhonePe payment │
│ page            │
│ (redirect_url)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Customer        │
│ completes       │
│ payment on      │
│ PhonePe:        │
│ - UPI           │
│ - Card          │
│ - Net Banking   │
│ - Wallet        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ PhonePe         │
│ redirects back  │
│ to storefront   │
│ callback URL    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Frontend polls  │
│ GET /payment-   │
│ status/{orderID}│
└────────┬────────┘
         │
┌────────┴────────────────┐
│              │          │
▼              ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│ PAID     │ │ PENDING  │ │ FAILED   │
│ Show     │ │ Keep     │ │ Show     │
│ success  │ │ polling  │ │ failure  │
│ page     │ │ (3s      │ │ page     │
│          │ │ interval)│ │          │
└────┬─────┘ └──────────┘ └────┬─────┘
     │                         │
     ▼                         ▼
┌──────────┐            ┌──────────┐
│ Redirect │            │ Option:  │
│ to order │            │ Retry or │
│ details  │            │ go home  │
└────┬─────┘            └──────────┘
     │
     ▼
┌────────┐
│  END   │
└────────┘
```

---

## 3. Payment Completion Flow (Webhook)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PAYMENT COMPLETION FLOW (WEBHOOK)                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ PhonePe sends   │
│ webhook POST    │
│ /webhooks/      │
│ phonepe         │
│ (Authorization: │
│ SHA256(u:p))    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Verify webhook  │
│ auth header:    │
│ SHA256(username: │
│ password) ==    │
│ Authorization   │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Signature     │  │ Signature     │
│ valid         │  │ invalid       │
│               │  │ Return 401    │
│               │  │ Log warning   │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Extract merchant│
│ transaction ID  │
│ from payload    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Look up Payment │
│ by merchant     │
│ transaction ID  │
│ (GSI2: PAYMENT_ │
│  TXN index)     │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Payment found │  │ Payment not   │
│               │  │ found         │
│               │  │ Return 404    │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Already         │
│ processed?      │
│ (idempotency    │
│  check)         │
└────────┬────────┘
         │
         ├── Already PAID ───────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Process payment │       │ Return 200 OK   │
│ result          │       │ (no-op)         │
└────────┬────────┘       └─────────────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Payment       │  │ Payment       │
│ SUCCESS       │  │ FAILED        │
└───────┬───────┘  └───────┬───────┘
        │                  │
        ▼                  ▼
┌─────────────────┐  ┌─────────────────┐
│ Update Payment: │  │ Update Payment: │
│ status = PAID   │  │ status = FAILED │
│ completed_at    │  │ completed_at    │
│ provider_txn_id │  │ provider resp   │
│ payment_method  │  │                 │
└────────┬────────┘  └────────┬────────┘
         │                    │
         ▼                    ▼
┌─────────────────┐  ┌─────────────────┐
│ Update Order:   │  │ Update Order:   │
│ payment_status  │  │ payment_status  │
│   = PAID        │  │   = FAILED      │
│ status          │  │ status          │
│   = CONFIRMED   │  │   = CANCELLED   │
│ payment_method  │  │ cancelled_at    │
│ payment_id      │  │                 │
└────────┬────────┘  └────────┬────────┘
         │                    │
         │                    ▼
         │           ┌─────────────────┐
         │           │ Release reserved│
         │           │ inventory:      │
         │           │ For each item:  │
         │           │  available_qty  │
         │           │    += quantity  │
         │           │  reserved_qty   │
         │           │    -= quantity  │
         │           └────────┬────────┘
         │                    │
         └──────────┬─────────┘
                    │
                    ▼
           ┌───────────────┐
           │ Add status    │
           │ history entry │
           │ (ORDER#<id>   │
           │  STATUS#<ts>) │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Return 200 OK │
           │ to PhonePe    │
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## State Diagram - Checkout Lifecycle

```
                    ┌─────────────────┐
                    │   CART READY    │
                    │   (Items added) │
                    └────────┬────────┘
                             │
                    Customer clicks
                    "Place Order & Pay"
                             │
                             ▼
                    ┌─────────────────┐
                    │  ORDER CREATED  │
                    │  status=PENDING │
                    │  payment=PENDING│
                    └────────┬────────┘
                             │
                    Redirect to PhonePe
                             │
                             ▼
                    ┌─────────────────┐
                    │  AWAITING       │
                    │  PAYMENT        │
                    │  (PhonePe page) │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
          Payment success           Payment failed
                │                         │
                ▼                         ▼
       ┌─────────────────┐       ┌─────────────────┐
       │   CONFIRMED     │       │   CANCELLED     │
       │  payment=PAID   │       │  payment=FAILED │
       │  (Webhook)      │       │  (Inventory     │
       │                 │       │   released)     │
       └────────┬────────┘       └─────────────────┘
                │
           Admin actions
                │
                ▼
       ┌─────────────────┐
       │   PROCESSING    │──────▶ SHIPPED ──────▶ DELIVERED
       │  (Being packed) │
       └─────────────────┘
```
