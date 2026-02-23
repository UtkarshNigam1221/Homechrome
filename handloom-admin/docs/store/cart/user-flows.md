# Store Cart - User Flows

## Overview

This document describes the user flows for the B2C Store Cart service, covering all cart operations from adding items through guest cart merging.

---

## 1. Add to Cart Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            ADD TO CART FLOW                                   │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Add to Cart"   │
│ on product page │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Custom-sized    │
│ product?        │
└────────┬────────┘
         │
         ├── Yes ────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Standard product│       │ Validate quote: │
│ Use selling_    │       │ - quote_id set? │
│ price from      │       │ - Not expired?  │
│ catalog         │       │ - Dimensions ok?│
└────────┬────────┘       └────────┬────────┘
         │                         │
         │                ┌────────┴────────┐
         │                │                 │
         │                ▼                 ▼
         │       ┌───────────────┐  ┌───────────────┐
         │       │ Quote valid   │  │ Quote invalid │
         │       │ Use calculated│  │ Return error  │
         │       │ price         │  │ QUOTE_EXPIRED │
         │       └───────┬───────┘  └───────────────┘
         │               │
         └───────┬───────┘
                 │
                 ▼
        ┌─────────────────┐
        │ Validate product│
        │ - Exists?       │
        │ - Status=ACTIVE?│
        └────────┬────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Product OK    │  │ Product not   │
│               │  │ found/inactive│
│               │  │ Return 404    │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Check inventory │
│ available_qty   │
│ >= quantity     │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Stock OK      │  │ Insufficient  │
│               │  │ stock         │
│               │  │ Return 409    │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Item already    │
│ in cart?        │
└────────┬────────┘
         │
         ├── Yes ────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Create new      │       │ Increment       │
│ CartItem with   │       │ existing item   │
│ product snapshot│       │ quantity and    │
│ (name, sku,     │       │ recalculate     │
│  image, price)  │       │ total_price     │
└────────┬────────┘       └────────┬────────┘
         │                         │
         └───────────┬─────────────┘
                     │
                     ▼
            ┌───────────────┐
            │ Recalculate   │
            │ cart totals:  │
            │ - subtotal    │
            │ - item_count  │
            └───────┬───────┘
                    │
                    ▼
            ┌───────────────┐
            │ Update cart   │
            │ header +      │
            │ refresh TTL   │
            └───────┬───────┘
                    │
                    ▼
            ┌───────────────┐
            │ Return        │
            │ CartWithItems │
            └───────┬───────┘
                    │
                    ▼
               ┌────────┐
               │  END   │
               └────────┘
```

---

## 2. Update Quantity Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         UPDATE QUANTITY FLOW                                  │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer changes│
│ quantity in     │
│ cart page       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Quantity = 0?   │
└────────┬────────┘
         │
         ├── Yes ────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ Quantity > 0    │       │ Remove item     │
│ Validate stock  │       │ (same as        │
│ available_qty   │       │  Remove flow)   │
│ >= new quantity │       └────────┬────────┘
└────────┬────────┘                │
         │                         │
┌────────┴────────┐                │
│                 │                │
▼                 ▼                │
┌───────────────┐  ┌───────────────┐  │
│ Stock OK      │  │ Insufficient  │  │
│               │  │ Return 409    │  │
└───────┬───────┘  └───────────────┘  │
        │                             │
        ▼                             │
┌─────────────────┐                   │
│ Update CartItem │                   │
│ - quantity      │                   │
│ - total_price = │                   │
│   qty * unit_   │                   │
│   price         │                   │
└────────┬────────┘                   │
         │                            │
         └──────────┬─────────────────┘
                    │
                    ▼
           ┌───────────────┐
           │ Recalculate   │
           │ cart totals   │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Update cart   │
           │ header +      │
           │ refresh TTL   │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Return        │
           │ CartWithItems │
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## 3. Remove Item Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          REMOVE ITEM FLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Remove" on     │
│ cart item       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Verify item     │
│ exists in cart: │
│ PK=CART#<cid>   │
│ SK=ITEM#<pid>   │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Item found    │  │ Item not in   │
│               │  │ cart          │
│               │  │ Return 404    │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Delete CartItem │
│ from DynamoDB   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Recalculate     │
│ cart totals     │
│ (subtotal,      │
│  item_count)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Update cart     │
│ header +        │
│ refresh TTL     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Return          │
│ CartWithItems   │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 4. Clear Cart Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CLEAR CART FLOW                                     │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer clicks │
│ "Clear Cart"    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Query all items │
│ PK=CART#<cid>   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ BatchDelete     │
│ all ITEM#*      │
│ records and     │
│ METADATA record │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Return success  │
│ message         │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 5. Guest Cart Merge Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       GUEST CART MERGE FLOW                                   │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Guest browses   │
│ site, adds      │
│ items to local  │
│ cart (frontend) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Guest logs in   │
│ via OTP         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Frontend sends  │
│ POST /merge     │
│ with guest cart │
│ items           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ For each item   │
│ in merge request│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Validate:       │
│ - Product exists│
│ - Product active│
│ - Stock ok      │
│ - Quote valid   │
│   (if custom)   │
└────────┬────────┘
         │
┌────────┴────────┐
│                 │
▼                 ▼
┌───────────────┐  ┌───────────────┐
│ Valid item    │  │ Invalid item  │
│               │  │ (skip, log    │
│               │  │  warning)     │
└───────┬───────┘  └───────────────┘
        │
        ▼
┌─────────────────┐
│ Already in      │
│ customer cart?  │
└────────┬────────┘
         │
         ├── Yes ────────────────────┐
         │                           │
         ▼                           ▼
┌─────────────────┐       ┌─────────────────┐
│ New item:       │       │ Existing item:  │
│ Create CartItem │       │ Add guest qty   │
│ with product    │       │ to existing qty │
│ snapshot        │       │ Recheck stock   │
└────────┬────────┘       └────────┬────────┘
         │                         │
         └───────────┬─────────────┘
                     │
                     ▼
            ┌───────────────┐
            │ After all     │
            │ items merged: │
            │ Recalculate   │
            │ cart totals   │
            └───────┬───────┘
                    │
                    ▼
            ┌───────────────┐
            │ Update cart   │
            │ header +      │
            │ refresh TTL   │
            └───────┬───────┘
                    │
                    ▼
            ┌───────────────┐
            │ Return        │
            │ CartWithItems │
            └───────┬───────┘
                    │
                    ▼
            ┌───────────────┐
            │ Frontend      │
            │ clears local  │
            │ guest cart    │
            └───────┬───────┘
                    │
                    ▼
               ┌────────┐
               │  END   │
               └────────┘
```

---

## State Diagram - Cart Item Lifecycle

```
                              ┌─────────────────┐
                              │   NO CART       │
                              │   (Empty)       │
                              └────────┬────────┘
                                       │
                              First AddItem call
                              (cart auto-created)
                                       │
                                       ▼
                              ┌─────────────────┐
     Add more items ─────────▶│   CART ACTIVE   │◀─── Update quantity
     Merge guest cart ───────▶│  (Has Items)    │◀─── Remove item (if >1)
                              └────────┬────────┘
                                       │
                        ┌──────────────┼──────────────┐
                        │              │              │
                   Clear Cart    Remove last     Checkout
                        │         item            (clear after
                        │              │           order placed)
                        ▼              ▼              │
                  ┌─────────────────┐                │
                  │   CART EMPTY    │◀───────────────┘
                  │   (Header only) │
                  └────────┬────────┘
                           │
                     30 days TTL
                     expires
                           │
                           ▼
                  ┌─────────────────┐
                  │   DELETED       │
                  │   (DynamoDB TTL)│
                  └─────────────────┘
```
