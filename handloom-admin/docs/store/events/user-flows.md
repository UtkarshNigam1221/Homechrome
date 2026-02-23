# Store Events - User Flows

## Overview

This document describes the user flows for the B2C Store Events (Tracking) service, covering how frontend tracking events are collected, batched, and ingested for analytics.

---

## 1. Anonymous Visitor Browsing

```
+---------------------------------------------------------------------------+
|                     ANONYMOUS VISITOR BROWSING FLOW                         |
+---------------------------------------------------------------------------+

    +----------+
    |  START   |
    +----+-----+
         |
         v
+-----------------+
| Visitor opens   |
| storefront      |
| initAnalytics() |
| starts buffer   |
| + 30s timer     |
+--------+--------+
         |
         v
+-----------------+
| Visitor         |
| navigates pages |
+--------+--------+
         |
         v
+-----------------+
| Each navigation |
| fires:          |
| track(          |
|  'page_view',   |
|  {page_path})   |
+--------+--------+
         |
         v
+-----------------+
| Event added to  |
| in-memory buffer|
+--------+--------+
         |
         v
+-----------------+
| Buffer full     |
| (10 events)?    |
+--------+--------+
         |
         +-- Yes ---------+
         |                 |
         v                 v
+-----------------+ +-----------------+
| No: wait for    | | Flush buffer:   |
| 30s timer or    | | POST /api/v1/   |
| next event      | | store/events    |
+--------+--------+ | {events: [...]} |
         |          +--------+--------+
         |                   |
         +---<---<---<-------+
         |
         v (on tab close / visibility change)
+-----------------+
| sendBeacon      |
| flushes any     |
| remaining events|
+--------+--------+
         |
         v
    +--------+
    |  END   |
    +--------+
```

---

## 2. Customer Adds to Cart

```
+---------------------------------------------------------------------------+
|                     ADD TO CART TRACKING FLOW                                |
+---------------------------------------------------------------------------+

    +----------+
    |  START   |
    +----+-----+
         |
         v
+-----------------+
| Customer views  |
| product page    |
| (product_viewed |
|  already tracked|
|  by browsing    |
|  flow)          |
+--------+--------+
         |
         v
+-----------------+
| Customer clicks |
| "Add to Cart"   |
+--------+--------+
         |
         v
+-----------------+
| Cart API call   |
| POST /cart/items|
| completes       |
| successfully    |
+--------+--------+
         |
         v
+-----------------+
| Frontend fires: |
| track(          |
|  'add_to_cart', |
|  {product_id,   |
|   quantity,      |
|   price})        |
+--------+--------+
         |
         v
+-----------------+
| Event added to  |
| buffer          |
+--------+--------+
         |
         v
+-----------------+
| Flushed on next |
| cycle (10 events|
| or 30s timer)   |
+--------+--------+
         |
         v
    +--------+
    |  END   |
    +--------+
```

---

## 3. Checkout Started

```
+---------------------------------------------------------------------------+
|                     CHECKOUT STARTED TRACKING FLOW                          |
+---------------------------------------------------------------------------+

    +----------+
    |  START   |
    +----+-----+
         |
         v
+-----------------+
| Customer clicks |
| "Proceed to     |
|  Checkout"      |
+--------+--------+
         |
         v
+-----------------+
| Checkout page   |
| loads           |
+--------+--------+
         |
         v
+-----------------+
| Frontend fires: |
| track(          |
|  'checkout_     |
|   started',     |
|  {cart_total,   |
|   item_count})  |
+--------+--------+
         |
         v
+-----------------+
| Event added to  |
| buffer          |
+--------+--------+
         |
         v
+-----------------+
| Flushed on next |
| cycle (10 events|
| or 30s timer)   |
+--------+--------+
         |
         v
    +--------+
    |  END   |
    +--------+
```

---

## State Diagram - Event Buffer Lifecycle

```
                          +-----------------+
                          |   BUFFER EMPTY  |
                          |   (0 events)    |
                          +--------+--------+
                                   |
                          track() called
                          (event added)
                                   |
                                   v
                          +-----------------+
    track() called ------+|  BUFFER FILLING |+---- 30s timer fires
    (event added)        ||  (1-9 events)   ||     (flush + reset)
                          +--------+--------+
                                   |
                          10th event added
                                   |
                                   v
                          +-----------------+
                          |  BUFFER FULL    |
                          |  (10 events)    |
                          +--------+--------+
                                   |
                          Immediate flush:
                          POST /api/v1/store/events
                                   |
                                   v
                          +-----------------+
                          |  BUFFER EMPTY   |
                          |  Timer reset    |
                          +-----------------+

    Special triggers (any buffer state):
    - Page unload  --> sendBeacon flush
    - Tab hidden   --> visibility change flush
```
