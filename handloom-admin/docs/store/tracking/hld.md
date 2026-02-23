# Store Tracking - High-Level Design (HLD)

## 1. Overview

The Store Tracking module provides a public endpoint for customers to track their order status by order number. It does not require authentication, enabling tracking via shared links (e.g., SMS notifications, email). The module looks up orders by the order number, builds a status timeline from order status history records, and fetches live shipment details from the Shiprocket integration when an AWB (Air Waybill) number is available.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 STORE TRACKING SYSTEM                                        │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                    ┌───────────────────┐          ┌───────────────────┐
                    │  Next.js Frontend │          │  SMS Tracking     │
                    │  (Track Page)     │          │  Link (Direct)    │
                    └─────────┬─────────┘          └─────────┬─────────┘
                              │                              │
                              └──────────────┬───────────────┘
                                             │
                                             │ HTTPS (No Auth)
                                             ▼
                                   ┌───────────────────┐
                                   │   API Gateway /   │
                                   │   Chi Router      │
                                   │   (No Auth MW)    │
                                   └─────────┬─────────┘
                                             │
                                             ▼
                                   ┌───────────────────┐
                                   │  Store Tracking   │
                                   │  Handler          │
                                   │  - TrackOrder     │
                                   └─────────┬─────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              │              │              │
                              ▼              ▼              ▼
                    ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐
                    │  Order      │  │  Order      │  │  Shiprocket     │
                    │  Repo       │  │  Status     │  │  Gateway        │
                    │  (by number)│  │  History    │  │  (live track)   │
                    └──────┬──────┘  └──────┬──────┘  └────────┬────────┘
                           │               │                   │
                           ▼               ▼                   ▼
                    ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐
                    │  DynamoDB   │  │  DynamoDB   │  │  Shiprocket     │
                    │  (GSI1 or   │  │  (STATUS#   │  │  API            │
                    │   Scan)     │  │   records)  │  │                 │
                    └─────────────┘  └─────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Tracking Handler

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      STORE TRACKING HANDLER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    StoreTrackingHandler                              │   │
│  │                                                                      │   │
│  │  Dependencies:                                                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - OrderRepository     (domain.OrderRepository)               │ │   │
│  │  │  - ShiprocketGateway   (gateway.ShiprocketGateway)            │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Routes (public, no auth middleware):                                │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  GET  /{orderNumber}   → TrackOrder(w, r)                     │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Processing Pipeline:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Step 1: Lookup order by order_number                                │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Query OrderNumberIndex GSI or scan with filter              │ │   │
│  │  │ • Return 404 if not found                                     │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 2: Build status timeline                                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Query ORDER#{id} with SK begins_with STATUS#                │ │   │
│  │  │ • Map status history records to timeline entries              │ │   │
│  │  │ • Sort by timestamp ascending                                 │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 3: Fetch shipment info (if shipped)                            │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Check if order has TrackingNumber (AWB)                     │ │   │
│  │  │ • If AWB present: query Shiprocket API for live status        │ │   │
│  │  │ • If no AWB: return shipment as null                          │ │   │
│  │  │ • Build tracking_url from courier + AWB                       │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 Data Lookup Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     DATA LOOKUP STRATEGY                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Step 1: Order Lookup by OrderNumber                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Table: handloom-orders                                            │      │
│  │ Method: Scan with filter (OrderNumber = :orderNumber)             │      │
│  │                                                                   │      │
│  │ Note: If volume grows, add a dedicated GSI:                       │      │
│  │   OrderNumberIndex: GSI1PK=ORDER_NUMBER#{number} GSI1SK=METADATA  │      │
│  │                                                                   │      │
│  │ Returns: Order record (ID, Status, TrackingNumber, ShippingCarrier│      │
│  │          ShippedAt, DeliveredAt, CancelledAt, CreatedAt)          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Step 2: Status History Records                                              │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Table: handloom-orders                                            │      │
│  │ Query: PK = ORDER#{order_id}, SK begins_with STATUS#              │      │
│  │                                                                   │      │
│  │ Returns: Array of status transitions                              │      │
│  │   [{FromStatus, ToStatus, Reason, CreatedAt}, ...]                │      │
│  │                                                                   │      │
│  │ Mapped to response:                                               │      │
│  │   [{status: ToStatus, timestamp: CreatedAt, note: Reason}, ...]   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Step 3: Live Shipment Data (external)                                       │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Source: Shiprocket API                                            │      │
│  │ Endpoint: GET /courier/track/awb/{awb_number}                     │      │
│  │                                                                   │      │
│  │ Returns: Current courier status, ETD (estimated time of delivery) │      │
│  │                                                                   │      │
│  │ Fallback: If Shiprocket API is unavailable, use last known status │      │
│  │ from order record (TrackingNumber, ShippingCarrier)               │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SECURITY MODEL                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Public Endpoint:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • No authentication required                                       │   │
│  │ • No CustomerAuth middleware on tracking routes                     │   │
│  │ • Rate-limited to prevent enumeration attacks                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Information Minimization:                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Response only includes: order_number, status, timeline, shipment │   │
│  │ • No customer PII (name, phone, email, address) is returned        │   │
│  │ • No item details or pricing information is returned               │   │
│  │ • No customer_id or order_id is exposed in the response            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Enumeration Protection:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Order number format (ORD-YYYY-NNNNNN) has large keyspace         │   │
│  │ • Consistent 404 response for non-existent orders                   │   │
│  │ • Rate limiting (30 req/min per IP) prevents brute-force scanning  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            ERROR CODES                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Tracking Errors:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Code                  │ HTTP │ Description                          │   │
│  ├───────────────────────┼──────┼────────────────────────────────────┤   │
│  │ ORDER_NOT_FOUND       │ 404  │ No order matches the order number   │   │
│  │ RATE_LIMIT_EXCEEDED   │ 429  │ Too many tracking requests          │   │
│  │ INTERNAL_ERROR        │ 500  │ Unexpected server error             │   │
│  └───────────────────────┴──────┴────────────────────────────────────┘   │
│                                                                              │
│  Shiprocket Fallback:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • If Shiprocket API call fails, do NOT fail the entire request     │   │
│  │ • Return shipment with data from order record only                  │   │
│  │ • Set shipment.status to order's last known shipping status         │   │
│  │ • Set shipment.estimated_delivery to null                           │   │
│  │ • Log the Shiprocket error for monitoring                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INTEGRATION POINTS                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Order Repository                                   │   │
│  │                                                                      │   │
│  │  StoreTrackingHandler ──▶ OrderRepository                           │   │
│  │    • GetByOrderNumber(orderNumber) — order lookup                   │   │
│  │    • GetStatusHistory(orderID) — status timeline records            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Shiprocket Gateway                                  │   │
│  │                                                                      │   │
│  │  StoreTrackingHandler ──▶ ShiprocketGateway                         │   │
│  │    • TrackByAWB(awbNumber) — live shipment tracking                 │   │
│  │    • Returns: courier status, ETD, tracking URL                     │   │
│  │    • Timeout: 5 seconds (fail gracefully)                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    SMS Notification (upstream)                         │   │
│  │                                                                      │   │
│  │  When order is shipped, SMS sent with tracking link:                 │   │
│  │    "Your order ORD-2026-000123 has been shipped.                    │   │
│  │     Track at: https://homechrome.lldlab.com/track/ORD-2026-000123" │   │
│  │                                                                      │   │
│  │  This drives traffic to the public tracking endpoint.               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             DEPENDENCIES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB (handloom-orders table) — order + status history    │   │
│  │ • Shiprocket API — live shipment tracking by AWB number            │   │
│  │ • AWS CloudWatch — logging & monitoring                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • OrderRepository — order lookup by number, status history query   │   │
│  │ • ShiprocketGateway — courier tracking integration                │   │
│  │ • Rate Limiter Middleware — 30 req/min per IP for public endpoint  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Shared Domain Entities:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • domain.Order — order entity (status, tracking fields)            │   │
│  │ • domain.OrderStatusHistory — status transition record             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
