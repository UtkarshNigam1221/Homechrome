# Inventory Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for inventory management operations.

---

## 1. Add Stock Sequence

```
┌────────┐     ┌─────────┐     ┌────────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │ Auth MW    │     │Inventory Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └─────┬──────┘     └──────┬──────┘     └────┬─────┘
    │               │                │                   │                 │
    │ POST          │                │                   │                 │
    │ /inventory/   │                │                   │                 │
    │ {productId}/  │                │                   │                 │
    │ add           │                │                   │                 │
    │ {qty,reason}  │                │                   │                 │
    │──────────────▶│                │                   │                 │
    │               │                │                   │                 │
    │               │ Validate JWT   │                   │                 │
    │               │───────────────▶│                   │                 │
    │               │                │                   │                 │
    │               │ Valid, user_id │                   │                 │
    │               │◀───────────────│                   │                 │
    │               │                │                   │                 │
    │               │ Forward request│                   │                 │
    │               │────────────────────────────────────▶                 │
    │               │                │                   │                 │
    │               │                │                   │ Get current     │
    │               │                │                   │ inventory       │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Current qty     │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Calculate new   │
    │               │                │                   │ quantities      │
    │               │                │                   │──────────┐      │
    │               │                │                   │          │      │
    │               │                │                   │◀─────────┘      │
    │               │                │                   │                 │
    │               │                │                   │ Update inventory│
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Success         │
    │               │                │                   │◀────────────────│
    │               │                │                   │                 │
    │               │                │                   │ Create txn      │
    │               │                │                   │ record          │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │                │                   │ Update product  │
    │               │                │                   │ stock fields    │
    │               │                │                   │────────────────▶│
    │               │                │                   │                 │
    │               │ {result}       │                   │                 │
    │               │◀────────────────────────────────────                 │
    │               │                │                   │                 │
    │ 200 OK        │                │                   │                 │
    │◀──────────────│                │                   │                 │
    │               │                │                   │                 │
```

---

## 2. Remove Stock Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │Inventory Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ POST          │                 │                 │
    │ /inventory/   │                 │                 │
    │ {productId}/  │                 │                 │
    │ remove        │                 │                 │
    │ {qty,reason}  │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get current     │
    │               │                 │ inventory       │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Current record  │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Check available │
    │               │                 │ >= requested    │
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │                 │ [If insufficient]
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │ 400 Error       │◀─────────┘      │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │               │                 │ [If sufficient] │
    │               │                 │ Decrement qty   │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Success         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Create txn      │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Update product  │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │ {result}        │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 3. Adjust Stock Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐     ┌───────────┐
│ Client │     │ API GW  │     │Inventory Svc│     │ DynamoDB │     │ Audit Svc │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘     └─────┬─────┘
    │               │                 │                 │                 │
    │ POST          │                 │                 │                 │
    │ /inventory/   │                 │                 │                 │
    │ {productId}/  │                 │                 │                 │
    │ adjust        │                 │                 │                 │
    │ {new_qty,     │                 │                 │                 │
    │  reason}      │                 │                 │                 │
    │──────────────▶│                 │                 │                 │
    │               │                 │                 │                 │
    │               │ Forward request │                 │                 │
    │               │────────────────▶│                 │                 │
    │               │                 │                 │                 │
    │               │                 │ Get current     │                 │
    │               │                 │ inventory       │                 │
    │               │                 │────────────────▶│                 │
    │               │                 │                 │                 │
    │               │                 │ Current qty     │                 │
    │               │                 │◀────────────────│                 │
    │               │                 │                 │                 │
    │               │                 │ Calculate       │                 │
    │               │                 │ difference      │                 │
    │               │                 │──────────┐      │                 │
    │               │                 │          │      │                 │
    │               │                 │◀─────────┘      │                 │
    │               │                 │                 │                 │
    │               │                 │ Set new qty     │                 │
    │               │                 │────────────────▶│                 │
    │               │                 │                 │                 │
    │               │                 │ Success         │                 │
    │               │                 │◀────────────────│                 │
    │               │                 │                 │                 │
    │               │                 │ Create ADJUST   │                 │
    │               │                 │ transaction     │                 │
    │               │                 │────────────────▶│                 │
    │               │                 │                 │                 │
    │               │                 │ Log adjustment  │                 │
    │               │                 │ for audit       │                 │
    │               │                 │─────────────────────────────────▶│
    │               │                 │                 │                 │
    │               │ {result}        │                 │                 │
    │               │◀────────────────│                 │                 │
    │               │                 │                 │                 │
    │ 200 OK        │                 │                 │                 │
    │◀──────────────│                 │                 │                 │
    │               │                 │                 │                 │
```

---

## 4. Get Inventory by Product Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │Inventory Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET           │                 │                 │
    │ /inventory/   │                 │                 │
    │ {productId}   │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Get inventory   │
    │               │                 │ record          │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Inventory data  │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Calculate       │
    │               │                 │ available qty   │
    │               │                 │ (qty - reserved)│
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │ {inventory}     │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │

Response Structure:
┌─────────────────────────────────────────────────────────┐
│  {                                                      │
│    "product_id": "prod_abc123",                         │
│    "quantity": 75,                                      │
│    "reserved_qty": 5,                                   │
│    "available_qty": 70,                                 │
│    "low_stock_threshold": 10,                           │
│    "is_low_stock": false,                               │
│    "last_updated": "2024-01-15T10:30:00Z"              │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
```

---

## 5. Get Transaction History Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │Inventory Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET           │                 │                 │
    │ /inventory/   │                 │                 │
    │ {productId}/  │                 │                 │
    │ transactions  │                 │                 │
    │ ?page=1       │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Query txns      │
    │               │                 │ by product_id   │
    │               │                 │ (GSI)           │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Transaction     │
    │               │                 │ records         │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Sort by date    │
    │               │                 │ (descending)    │
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │ {transactions,  │                 │
    │               │  pagination}    │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 6. Get Low Stock Products Sequence

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌──────────┐
│ Client │     │ API GW  │     │Inventory Svc│     │ DynamoDB │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬─────┘
    │               │                 │                 │
    │ GET           │                 │                 │
    │ /inventory/   │                 │                 │
    │ low-stock     │                 │                 │
    │ ?page=1       │                 │                 │
    │──────────────▶│                 │                 │
    │               │                 │                 │
    │               │ Forward request │                 │
    │               │────────────────▶│                 │
    │               │                 │                 │
    │               │                 │ Scan inventory  │
    │               │                 │ where qty <=    │
    │               │                 │ threshold       │
    │               │                 │────────────────▶│
    │               │                 │                 │
    │               │                 │ Low stock items │
    │               │                 │◀────────────────│
    │               │                 │                 │
    │               │                 │ Sort by         │
    │               │                 │ urgency         │
    │               │                 │ (qty/threshold) │
    │               │                 │──────────┐      │
    │               │                 │          │      │
    │               │                 │◀─────────┘      │
    │               │                 │                 │
    │               │ {products,      │                 │
    │               │  pagination}    │                 │
    │               │◀────────────────│                 │
    │               │                 │                 │
    │ 200 OK        │                 │                 │
    │◀──────────────│                 │                 │
    │               │                 │                 │
```

---

## 7. Order Reserve/Release Sequence (Internal)

```
┌───────────┐     ┌─────────────┐     ┌──────────┐
│ Order Svc │     │Inventory Svc│     │ DynamoDB │
└─────┬─────┘     └──────┬──────┘     └────┬─────┘
      │                  │                 │
      │ ReserveInventory │                 │
      │ (orderId, items) │                 │
      │─────────────────▶│                 │
      │                  │                 │
      │                  │ For each item:  │
      │                  │ ┌─────────────┐ │
      │                  │ │ Check       │ │
      │                  │ │ available   │ │
      │                  │ │────────────▶│ │
      │                  │ │             │ │
      │                  │ │ Increment   │ │
      │                  │ │ reserved_qty│ │
      │                  │ │────────────▶│ │
      │                  │ └─────────────┘ │
      │                  │                 │
      │ Reservation OK   │                 │
      │◀─────────────────│                 │
      │                  │                 │
      │    ... order ships ...            │
      │                  │                 │
      │ CommitReservation│                 │
      │ (orderId)        │                 │
      │─────────────────▶│                 │
      │                  │                 │
      │                  │ For each item:  │
      │                  │ ┌─────────────┐ │
      │                  │ │ Decrement   │ │
      │                  │ │ quantity    │ │
      │                  │ │────────────▶│ │
      │                  │ │             │ │
      │                  │ │ Decrement   │ │
      │                  │ │ reserved_qty│ │
      │                  │ │────────────▶│ │
      │                  │ └─────────────┘ │
      │                  │                 │
      │ Commit OK        │                 │
      │◀─────────────────│                 │
      │                  │                 │

Order Cancelled:
┌───────────┐     ┌─────────────┐     ┌──────────┐
│ Order Svc │     │Inventory Svc│     │ DynamoDB │
└─────┬─────┘     └──────┬──────┘     └────┬─────┘
      │                  │                 │
      │ ReleaseReserve   │                 │
      │ (orderId)        │                 │
      │─────────────────▶│                 │
      │                  │                 │
      │                  │ For each item:  │
      │                  │ ┌─────────────┐ │
      │                  │ │ Decrement   │ │
      │                  │ │ reserved_qty│ │
      │                  │ │────────────▶│ │
      │                  │ └─────────────┘ │
      │                  │                 │
      │ Release OK       │                 │
      │◀─────────────────│                 │
      │                  │                 │
```
