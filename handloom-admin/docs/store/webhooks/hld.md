# Store Webhooks - High-Level Design (HLD)

## 1. Overview

The Store Webhooks module handles asynchronous callbacks from external service providers, specifically PhonePe (payment gateway) and Shiprocket (shipping provider). These webhooks are critical to the order lifecycle: payment webhooks transition orders from PENDING to CONFIRMED (or release inventory on failure), while shipping webhooks update shipment status and transition orders to DELIVERED or RETURNED. Both handlers always return HTTP 200 to prevent the external services from retrying, and process events idempotently to handle duplicate deliveries.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                  STORE WEBHOOKS SYSTEM                                       │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

     ┌───────────────────┐                              ┌───────────────────┐
     │    PhonePe        │                              │    Shiprocket     │
     │    Payment GW     │                              │    Shipping GW    │
     └─────────┬─────────┘                              └─────────┬─────────┘
               │                                                  │
               │ POST /webhooks/phonepe                           │ POST /webhooks/shiprocket
               │ (X-VERIFY signature)                             │
               ▼                                                  ▼
     ┌───────────────────┐                              ┌───────────────────┐
     │  PhonePe Webhook  │                              │  Shiprocket       │
     │  Handler          │                              │  Webhook Handler  │
     │  - Verify sig     │                              │  - Parse status   │
     │  - Decode base64  │                              │  - Map status     │
     │  - Process result │                              │  - Update order   │
     └─────────┬─────────┘                              └─────────┬─────────┘
               │                                                  │
               └──────────────────────┬───────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
         ┌─────────────────┐ ┌──────────────┐ ┌─────────────────┐
         │  Payment        │ │  Order       │ │  Inventory      │
         │  Service        │ │  Service     │ │  Service        │
         │  (status update)│ │  (status     │ │  (release on    │
         │                 │ │   transition)│ │   failure)      │
         └────────┬────────┘ └──────┬───────┘ └────────┬────────┘
                  │                 │                   │
                  └─────────────────┼───────────────────┘
                                    │
                                    ▼
                          ┌───────────────────┐
                          │     DynamoDB      │
                          │  (handloom-orders)│
                          └───────────────────┘
```

---

## 3. Component Design

### 3.1 PhonePe Webhook Handler

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     PHONEPE WEBHOOK HANDLER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Processing Pipeline:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Step 1: Verify Signature                                            │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Extract X-VERIFY header                                     │ │   │
│  │  │ • Compute: SHA256(base64Response + callbackURL + saltKey)     │ │   │
│  │  │ • Compare computed hash with X-VERIFY value                   │ │   │
│  │  │ • If mismatch: log warning, return 200 (do not process)       │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 2: Decode Response                                             │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Base64-decode the `response` field from request body        │ │   │
│  │  │ • Parse decoded JSON to extract:                              │ │   │
│  │  │   - code (PAYMENT_SUCCESS / PAYMENT_ERROR / PAYMENT_DECLINED) │ │   │
│  │  │   - data.merchantTransactionId (maps to PaymentID)            │ │   │
│  │  │   - data.transactionId (PhonePe transaction reference)        │ │   │
│  │  │   - data.amount (in paise)                                    │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 3: Idempotency Check                                           │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Lookup payment by merchantTransactionId                     │ │   │
│  │  │ • If payment.Status is already SUCCESS or FAILED:             │ │   │
│  │  │   → Skip processing, return 200 (already handled)             │ │   │
│  │  │ • If payment not found: log error, return 200                  │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 4: Process Payment Result                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ On PAYMENT_SUCCESS:                                           │ │   │
│  │  │   • Update payment status → SUCCESS                           │ │   │
│  │  │   • Update order status → CONFIRMED                           │ │   │
│  │  │   • Write status history record                               │ │   │
│  │  │   • Send confirmation SMS                                     │ │   │
│  │  │                                                               │ │   │
│  │  │ On PAYMENT_ERROR / PAYMENT_DECLINED:                          │ │   │
│  │  │   • Update payment status → FAILED                            │ │   │
│  │  │   • Release reserved inventory                                │ │   │
│  │  │   • Send failure notification SMS                             │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Shiprocket Webhook Handler

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SHIPROCKET WEBHOOK HANDLER                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Status Mapping:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Shiprocket Status       │ Order Status Change │ Side Effects        │   │
│  ├─────────────────────────┼─────────────────────┼─────────────────────┤   │
│  │ Pickup Scheduled        │ (none)              │ Update shipment     │   │
│  │ Picked Up               │ (none)              │ Update shipment     │   │
│  │ In Transit              │ (none)              │ Update shipment     │   │
│  │ Out For Delivery        │ (none)              │ Update + send SMS   │   │
│  │ Delivered               │ → DELIVERED         │ Stats + send SMS    │   │
│  │ RTO Initiated           │ (none)              │ Update + admin alert│   │
│  │ RTO Delivered           │ → RETURNED          │ Refund + send SMS   │   │
│  └─────────────────────────┴─────────────────────┴─────────────────────┘   │
│                                                                              │
│  Processing Pipeline:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Step 1: Parse Webhook Payload                                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Extract awb, current_status, shipment_id, order_id, etd    │ │   │
│  │  │ • Lookup order by order_id (order number)                     │ │   │
│  │  │ • If order not found: log warning, return 200                  │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 2: Update Shipment Record                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Update shipment status in DynamoDB                          │ │   │
│  │  │ • Update estimated delivery time if provided                   │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Step 3: Process Status-Specific Actions                             │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │ • Map Shiprocket status to internal status                    │ │   │
│  │  │ • If terminal (Delivered/RTO): update order status            │ │   │
│  │  │ • Execute side effects per status mapping table above         │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 Payment & Shipment Records

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-orders                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PAYMENT RECORD                                                              │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PAYMENT#<payment_id>                                          │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - ID (merchantTransactionId)                                    │      │
│  │   - OrderID               - Amount (paise)                        │      │
│  │   - Status (PENDING/SUCCESS/FAILED)                               │      │
│  │   - Provider ("PHONEPE")  - ProviderTxnID                        │      │
│  │   - PaymentMethod         - ResponseCode                          │      │
│  │   - ProcessedAt           - CreatedAt                             │      │
│  │   - UpdatedAt                                                     │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  SHIPMENT RECORD                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: SHIPMENT#<shipment_id>                                        │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - AWBNumber             - CourierName                           │      │
│  │   - ShiprocketShipmentID  - Status                                │      │
│  │   - TrackingURL           - EstimatedDelivery                     │      │
│  │   - CreatedAt             - UpdatedAt                             │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ORDER STATUS HISTORY (written by webhooks)                                  │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: ORDER#<order_id>                                              │      │
│  │ SK: STATUS#<timestamp>                                            │      │
│  │                                                                   │      │
│  │ Written when:                                                     │      │
│  │   - Payment SUCCESS → PENDING to CONFIRMED                        │      │
│  │   - Shiprocket DELIVERED → SHIPPED to DELIVERED                   │      │
│  │   - Shiprocket RTO → SHIPPED to RETURNED                         │      │
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
│  PhonePe Signature Verification:                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • X-VERIFY header contains SHA256 checksum + salt index            │   │
│  │ • Verification: SHA256(base64Response + callbackURL + saltKey)     │   │
│  │ • Salt key stored in AWS SSM: /handloom/{env}/phonepe-salt-key    │   │
│  │ • Salt index stored in config: PHONEPE_SALT_INDEX                  │   │
│  │ • Signature mismatch: log, return 200, do NOT process              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Shiprocket Validation:                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Webhook URL is configured in Shiprocket dashboard (secret URL)   │   │
│  │ • Order number validation ensures webhook matches a real order     │   │
│  │ • Unknown order numbers are logged and ignored                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  General Security:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Both endpoints always return 200 (no info leakage via status)    │   │
│  │ • Idempotent processing prevents replay attacks                     │   │
│  │ • All webhook events are logged for audit trail                    │   │
│  │ • Rate limiting applied to prevent abuse                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ERROR HANDLING                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Critical Design Decision: Always Return 200                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Both PhonePe and Shiprocket retry on non-200 responses           │   │
│  │ • Retries can cause duplicate processing if not handled            │   │
│  │ • Always return {status: "ok"} with HTTP 200                       │   │
│  │ • Handle errors internally via logging and monitoring               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Error Scenarios:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Scenario                    │ Response │ Action                     │   │
│  ├─────────────────────────────┼──────────┼────────────────────────────┤   │
│  │ Signature verification fail │ 200      │ Log warning, skip process  │   │
│  │ Payment not found           │ 200      │ Log error, skip process    │   │
│  │ Order not found             │ 200      │ Log error, skip process    │   │
│  │ Already processed (idemp)   │ 200      │ Log info, skip process     │   │
│  │ DynamoDB write failure      │ 200      │ Log error, alert, retry    │   │
│  │ Inventory release failure   │ 200      │ Log error, manual resolve  │   │
│  │ SMS send failure            │ 200      │ Log warning, non-blocking  │   │
│  │ Invalid/malformed payload   │ 200      │ Log error, skip process    │   │
│  └─────────────────────────────┴──────────┴────────────────────────────┘   │
│                                                                              │
│  Monitoring & Alerting:                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • CloudWatch metrics for webhook processing success/failure         │   │
│  │ • Alarm on signature verification failures (potential attack)       │   │
│  │ • Alarm on repeated payment-not-found errors (config issue)         │   │
│  │ • Dashboard for webhook processing latency and volume               │   │
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
│  │                    Payment Service                                    │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ PaymentService                                  │   │
│  │    • GetByMerchantTxnID(merchantTxnId) — lookup payment             │   │
│  │    • UpdateStatus(paymentID, status, providerTxnID) — save result   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Order Service                                      │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ OrderService                                    │   │
│  │    • ConfirmOrder(orderID) — PENDING → CONFIRMED on payment success │   │
│  │    • GetByOrderNumber(number) — lookup for Shiprocket webhook       │   │
│  │    • UpdateStatus(orderID, status) — DELIVERED, RETURNED            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Inventory Service                                   │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ InventoryService (on payment failure)           │   │
│  │    • ReleaseReservation(orderID) — restore reserved stock           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Shipment Repository                                │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ ShipmentRepository                              │   │
│  │    • GetByAWB(awbNumber) — lookup shipment record                   │   │
│  │    • UpdateStatus(shipmentID, status, etd) — save status change     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Customer Repository                                │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ CustomerRepository (on delivery)                │   │
│  │    • IncrementOrderStats(customerID, orderAmount) — update stats    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Notification Service                                │   │
│  │                                                                      │   │
│  │  WebhookHandler ──▶ NotificationService                             │   │
│  │    • SendOrderConfirmedSMS(phone, orderNumber)                      │   │
│  │    • SendPaymentFailedSMS(phone, orderNumber)                       │   │
│  │    • SendDeliveredSMS(phone, orderNumber)                           │   │
│  │    • SendOutForDeliverySMS(phone, orderNumber, etd)                 │   │
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
│  │ • PhonePe Payment Gateway — sends payment callbacks                │   │
│  │ • Shiprocket Shipping — sends shipment status updates              │   │
│  │ • AWS DynamoDB (handloom-orders) — order, payment, shipment data  │   │
│  │ • AWS SSM Parameter Store — PhonePe salt key                       │   │
│  │ • AWS CloudWatch — logging, metrics, alarms                        │   │
│  │ • MSG91 SMS Gateway — customer notifications                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • PaymentService — payment status management                       │   │
│  │ • OrderService — order status transitions                          │   │
│  │ • InventoryService — stock reservation release                     │   │
│  │ • CustomerRepository — customer stats update                       │   │
│  │ • ShipmentRepository — shipment record management                  │   │
│  │ • NotificationService — SMS notifications                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Configuration:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • PHONEPE_SALT_KEY — SHA256 signature verification                 │   │
│  │ • PHONEPE_SALT_INDEX — salt index for X-VERIFY header              │   │
│  │ • PHONEPE_MERCHANT_ID — merchant identifier                        │   │
│  │ • SHIPROCKET_WEBHOOK_SECRET — (optional) additional validation     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
