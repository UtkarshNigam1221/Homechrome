# Store Webhooks - Sequence Diagrams

## Overview
This document contains sequence diagrams for the B2C store webhook processing operations, covering PhonePe payment callbacks and Shiprocket shipping callbacks.

---

## 1. PhonePe Payment Webhook (Success)

```
┌─────────┐   ┌──────────────┐   ┌───────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐   ┌─────────┐
│ PhonePe │   │  Webhook     │   │ Payment   │   │  Order    │   │ DynamoDB │   │ Inventory │   │ Notif   │
│         │   │  Handler     │   │ Service   │   │  Service  │   │          │   │ Service   │   │ (SMS)   │
└────┬────┘   └──────┬───────┘   └─────┬─────┘   └─────┬─────┘   └────┬─────┘   └─────┬─────┘   └────┬────┘
     │               │                │               │              │               │              │
     │ POST /webhooks│                │               │              │               │              │
     │ /phonepe      │                │               │              │               │              │
     │ Authorization:│                │               │              │               │              │
     │  SHA256(u:p)  │                │               │              │               │              │
     │ {event,       │                │               │              │               │              │
     │  payload}     │                │               │              │               │              │
     │──────────────▶│                │               │              │               │              │
     │               │                │               │              │               │              │
     │               │ Verify auth    │               │              │               │              │
     │               │ SHA256(user:   │               │              │               │              │
     │               │ pass) == hdr   │               │              │               │              │
     │               │──────┐         │               │              │               │              │
     │               │      │         │               │              │               │              │
     │               │◀─────┘ Valid   │               │              │               │              │
     │               │                │               │              │               │              │
     │               │ Parse event:   │               │              │               │              │
     │               │ checkout.order │               │              │               │              │
     │               │ .completed     │               │              │               │              │
     │               │                │               │              │               │              │
     │               │ GetByMerchant  │               │              │               │              │
     │               │ TxnID(id)      │               │              │               │              │
     │               │───────────────▶│               │              │               │              │
     │               │                │               │              │               │              │
     │               │                │ Query by      │              │               │              │
     │               │                │ merchant txn  │              │               │              │
     │               │                │──────────────────────────────▶               │              │
     │               │                │               │              │               │              │
     │               │                │ Payment record│              │               │              │
     │               │                │ (PENDING)     │              │               │              │
     │               │                │◀──────────────────────────────               │              │
     │               │                │               │              │               │              │
     │               │ Payment found  │               │              │               │              │
     │               │ (not processed)│               │              │               │              │
     │               │◀───────────────│               │              │               │              │
     │               │                │               │              │               │              │
     │               │ UpdateStatus   │               │              │               │              │
     │               │ (SUCCESS)      │               │              │               │              │
     │               │───────────────▶│               │              │               │              │
     │               │                │               │              │               │              │
     │               │                │ Update payment│              │               │              │
     │               │                │ → SUCCESS     │              │               │              │
     │               │                │──────────────────────────────▶               │              │
     │               │                │               │              │               │              │
     │               │ ConfirmOrder   │               │              │               │              │
     │               │ (orderID)      │               │              │               │              │
     │               │────────────────────────────────▶              │               │              │
     │               │                │               │              │               │              │
     │               │                │               │ Update order │               │              │
     │               │                │               │ PENDING →    │               │              │
     │               │                │               │ CONFIRMED    │               │              │
     │               │                │               │─────────────▶│               │              │
     │               │                │               │              │               │              │
     │               │                │               │ Write status │               │              │
     │               │                │               │ history      │               │              │
     │               │                │               │─────────────▶│               │              │
     │               │                │               │              │               │              │
     │               │ Send SMS       │               │              │               │              │
     │               │ (confirmed)    │               │              │               │              │
     │               │─────────────────────────────────────────────────────────────────────────────▶│
     │               │                │               │              │               │              │
     │ 200 OK        │                │               │              │               │              │
     │ {status: "ok"}│                │               │              │               │              │
     │◀──────────────│                │               │              │               │              │
     │               │                │               │              │               │              │
```

---

## 2. PhonePe Payment Webhook (Failure)

```
┌─────────┐   ┌──────────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐   ┌─────────┐
│ PhonePe │   │  Webhook     │   │ Payment   │   │ DynamoDB │   │ Inventory │   │ Notif   │
│         │   │  Handler     │   │ Service   │   │          │   │ Service   │   │ (SMS)   │
└────┬────┘   └──────┬───────┘   └─────┬─────┘   └────┬─────┘   └─────┬─────┘   └────┬────┘
     │               │                │              │               │              │
     │ POST /webhooks│                │              │               │              │
     │ /phonepe      │                │              │               │              │
     │ Authorization:│                │              │               │              │
     │  SHA256(u:p)  │                │              │               │              │
     │ {event:       │                │              │               │              │
     │  checkout.    │                │              │               │              │
     │  order.failed}│                │              │               │              │
     │──────────────▶│                │              │               │              │
     │               │                │              │               │              │
     │               │ Verify + parse │              │               │              │
     │               │──────┐         │              │               │              │
     │               │      │         │              │               │              │
     │               │◀─────┘         │              │               │              │
     │               │                │              │               │              │
     │               │ GetByMerchant  │              │               │              │
     │               │ TxnID(id)      │              │               │              │
     │               │───────────────▶│              │               │              │
     │               │                │              │               │              │
     │               │                │ Query payment│               │              │
     │               │                │─────────────▶│               │              │
     │               │                │              │               │              │
     │               │                │ Payment      │               │              │
     │               │                │ (PENDING)    │               │              │
     │               │                │◀─────────────│               │              │
     │               │                │              │               │              │
     │               │ Payment found  │              │               │              │
     │               │◀───────────────│              │               │              │
     │               │                │              │               │              │
     │               │ UpdateStatus   │              │               │              │
     │               │ (FAILED)       │              │               │              │
     │               │───────────────▶│              │               │              │
     │               │                │              │               │              │
     │               │                │ Update       │               │              │
     │               │                │ → FAILED     │               │              │
     │               │                │─────────────▶│               │              │
     │               │                │              │               │              │
     │               │ Release        │              │               │              │
     │               │ inventory      │              │               │              │
     │               │ (orderID)      │              │               │              │
     │               │──────────────────────────────────────────────▶│              │
     │               │                │              │               │              │
     │               │                │              │  Restore      │              │
     │               │                │              │  reserved qty │              │
     │               │                │              │◀──────────────│              │
     │               │                │              │               │              │
     │               │ Stock released │              │               │              │
     │               │◀──────────────────────────────────────────────│              │
     │               │                │              │               │              │
     │               │ Send failure   │              │               │              │
     │               │ SMS            │              │               │              │
     │               │─────────────────────────────────────────────────────────────▶│
     │               │                │              │               │              │
     │ 200 OK        │                │              │               │              │
     │ {status: "ok"}│                │              │               │              │
     │◀──────────────│                │              │               │              │
     │               │                │              │               │              │
```

---

## 3. Shiprocket Shipping Webhook (Delivered)

```
┌────────────┐   ┌──────────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐   ┌─────────┐
│ Shiprocket │   │  Webhook     │   │ Shipping  │   │  Order   │   │ DynamoDB │   │ Notif   │
│            │   │  Handler     │   │ Service   │   │  Service │   │          │   │ (SMS)   │
└─────┬──────┘   └──────┬───────┘   └─────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬────┘
      │                 │                 │              │              │              │
      │ POST /webhooks/ │                 │              │              │              │
      │ shiprocket      │                 │              │              │              │
      │ {awb, status:   │                 │              │              │              │
      │  "Delivered",   │                 │              │              │              │
      │  order_id,etd}  │                 │              │              │              │
      │────────────────▶│                 │              │              │              │
      │                 │                 │              │              │              │
      │                 │ Parse payload   │              │              │              │
      │                 │──────┐          │              │              │              │
      │                 │      │          │              │              │              │
      │                 │◀─────┘          │              │              │              │
      │                 │                 │              │              │              │
      │                 │ Lookup order by │              │              │              │
      │                 │ order_id        │              │              │              │
      │                 │────────────────────────────────▶              │              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Query by     │              │
      │                 │                 │              │ order number │              │
      │                 │                 │              │─────────────▶│              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Order record │              │
      │                 │                 │              │◀─────────────│              │
      │                 │                 │              │              │              │
      │                 │ Order found     │              │              │              │
      │                 │◀────────────────────────────────              │              │
      │                 │                 │              │              │              │
      │                 │ Update shipment │              │              │              │
      │                 │ status          │              │              │              │
      │                 │────────────────▶│              │              │              │
      │                 │                 │              │              │              │
      │                 │                 │ UpdateItem   │              │              │
      │                 │                 │ SHIPMENT#    │              │              │
      │                 │                 │─────────────────────────────▶              │
      │                 │                 │              │              │              │
      │                 │ Status =        │              │              │              │
      │                 │ Delivered       │              │              │              │
      │                 │──────┐          │              │              │              │
      │                 │      │          │              │              │              │
      │                 │◀─────┘          │              │              │              │
      │                 │                 │              │              │              │
      │                 │ UpdateOrder     │              │              │              │
      │                 │ Status(DELIVERED│              │              │              │
      │                 │────────────────────────────────▶              │              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Update order │              │
      │                 │                 │              │ SHIPPED →    │              │
      │                 │                 │              │ DELIVERED    │              │
      │                 │                 │              │ +DeliveredAt │              │
      │                 │                 │              │─────────────▶│              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Write status │              │
      │                 │                 │              │ history      │              │
      │                 │                 │              │─────────────▶│              │
      │                 │                 │              │              │              │
      │                 │ Update customer │              │              │              │
      │                 │ stats           │              │              │              │
      │                 │──────────────────────────────────────────────▶│              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Increment    │              │
      │                 │                 │              │ TotalOrders  │              │
      │                 │                 │              │ TotalSpent   │              │
      │                 │                 │              │              │              │
      │                 │ Send delivery   │              │              │              │
      │                 │ SMS             │              │              │              │
      │                 │─────────────────────────────────────────────────────────────▶│
      │                 │                 │              │              │              │
      │ 200 OK          │                 │              │              │              │
      │ {status: "ok"}  │                 │              │              │              │
      │◀────────────────│                 │              │              │              │
      │                 │                 │              │              │              │
```

---

## 4. Shiprocket Shipping Webhook (RTO - Return to Origin)

```
┌────────────┐   ┌──────────────┐   ┌───────────┐   ┌──────────┐   ┌──────────┐   ┌─────────┐
│ Shiprocket │   │  Webhook     │   │ Shipping  │   │  Order   │   │ DynamoDB │   │ Notif   │
│            │   │  Handler     │   │ Service   │   │  Service │   │          │   │ (SMS)   │
└─────┬──────┘   └──────┬───────┘   └─────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬────┘
      │                 │                 │              │              │              │
      │ POST /webhooks/ │                 │              │              │              │
      │ shiprocket      │                 │              │              │              │
      │ {awb, status:   │                 │              │              │              │
      │  "RTO Delivered"│                 │              │              │              │
      │  order_id}      │                 │              │              │              │
      │────────────────▶│                 │              │              │              │
      │                 │                 │              │              │              │
      │                 │ Parse + lookup  │              │              │              │
      │                 │ order           │              │              │              │
      │                 │────────────────────────────────▶              │              │
      │                 │                 │              │              │              │
      │                 │ Order found     │              │              │              │
      │                 │◀────────────────────────────────              │              │
      │                 │                 │              │              │              │
      │                 │ Update shipment │              │              │              │
      │                 │ (RTO Delivered) │              │              │              │
      │                 │────────────────▶│              │              │              │
      │                 │                 │              │              │              │
      │                 │                 │ Update       │              │              │
      │                 │                 │ shipment rec │              │              │
      │                 │                 │─────────────────────────────▶              │
      │                 │                 │              │              │              │
      │                 │ UpdateOrder     │              │              │              │
      │                 │ Status(RETURNED)│              │              │              │
      │                 │────────────────────────────────▶              │              │
      │                 │                 │              │              │              │
      │                 │                 │              │ Update order │              │
      │                 │                 │              │ → RETURNED   │              │
      │                 │                 │              │─────────────▶│              │
      │                 │                 │              │              │              │
      │                 │                 │              │ If PAID:     │              │
      │                 │                 │              │ initiate     │              │
      │                 │                 │              │ refund       │              │
      │                 │                 │              │──────┐       │              │
      │                 │                 │              │      │       │              │
      │                 │                 │              │◀─────┘       │              │
      │                 │                 │              │              │              │
      │                 │ Send return SMS │              │              │              │
      │                 │─────────────────────────────────────────────────────────────▶│
      │                 │                 │              │              │              │
      │ 200 OK          │                 │              │              │              │
      │ {status: "ok"}  │                 │              │              │              │
      │◀────────────────│                 │              │              │              │
      │                 │                 │              │              │              │
```
