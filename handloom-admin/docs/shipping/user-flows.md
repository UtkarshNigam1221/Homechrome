# Shipping — User Flows

End-to-end flows for customer (B2C storefront) and admin (back-office) actors interacting with the Delhivery shipping subsystem.

---

## Customer Flows

### CF-1: Pincode serviceability check on Product Detail Page

**Trigger:** Customer enters 6-digit pincode in `<PincodeChecker>` on a PDP.

```
Customer enters pincode
  → PincodeChecker validates regex /^[1-9][0-9]{5}$/
  → GET /api/v1/store/catalog/check-pincode/{pin}
  → Storefront receives ServiceabilityResult
      → Pincode cache hit (DynamoDB PIN#<pin>, TTL 7 days): return cached
      → Cache miss / stale: ShippingService.CheckServiceability calls courier.CheckPincode (Delhivery)
        → Upsert PincodeZone row with new TTL
  → UI renders Alert success ("Delivery in N days") or warning ("Not serviceable")
  → Per-component in-memory Map memoizes within PDP visit
```

**Failure modes:**
- Invalid pincode format → inline error, no network call.
- Carrier API failure → toast error, no cache write.

### CF-2: Cart shipping display

**Trigger:** Customer views cart or checkout page.

```
Cart fetch returns cart.shipping_charge (paise) when backend has quoted (post-pincode).
  → <ShippingLine chargePaise> renders:
      undefined → "Calculated at checkout" (cart) / "Calculated next" (checkout)
      0         → "Free"
      > 0       → ₹X.XX (formatPrice)
```

**Computation (backend side):**
- Phase 2 checkout flow calls `RateTableService.Lookup(pincode, weight, mode)`.
- Lookup chain: `PincodeRepo.Get(pincode) → Zone → RateRepo.Get(zone, weight_slab) → paise`.
- Weight slab ceiling: 500g, 1000g, 2000g, 5000g, 10000g. Default item weight 500g.

### CF-3: Order tracking

**Trigger:** Customer enters order number on `/track`.

```
Customer submits order number
  → GET /api/v1/store/track/{orderNumber}
  → StoreTrackingService loads Order + Shipment by order ID
  → If shipment has AWB and status non-terminal AND last updated > 30 min ago:
      → ShippingService.TrackShipment calls courier.TrackByAWB (Delhivery /api/v1/packages/json/)
      → Status mapping via courier.ToShipmentStatus
      → ShipmentRepository.UpdateStatus (atomic: status + priority_status GSI key)
      → If newly DELIVERED: OrderRepository.UpdateStatus → DELIVERED
  → Returns TrackingResult with shipment + status_history
  → <TrackingTimeline> renders ordered scans (newest first), one highlighted dot per current status
```

**Failure modes:**
- 404 (order not found) → "Order not found. Check the order number and try again."
- 5xx / network → "Could not fetch tracking. Please try again."

### CF-4: Return policy page

`/return-policy` — Server Component, fully static, 1MB-cached at CDN. Surfaces 7-day return window, contact flow (`support@homechrome.in`), refund timeline (5–7 business days), exclusions, damaged-item flow.

---

## Admin Flows

### AF-1: Create shipment (NORMAL or PRIORITY)

**Trigger:** Admin clicks "Create shipment" on OrderDetailPage when `order.status` ∈ {CONFIRMED, PROCESSING} and no shipment yet.

```
Admin toggles priority (NORMAL | PRIORITY) → clicks button
  → POST /admin/orders/{id}/shipments?priority=NORMAL|PRIORITY
  → ShippingService.CreateShipment:
      → Build courier.CreateShipmentRequest from Order + ShippingAddress + items
      → courier.CreateShipment (Delhivery /api/cmu/create.json)
          → Returns AWB + uploadWBN
      → courier.GenerateLabel (Delhivery /api/p/packing_slip)
      → ShipmentRepository.Create (status=CREATED, priority, awb_number)
      → Publish event.ShipmentCreated
      → OrderRepository.UpdateStatus → SHIPPED
      → OrderRepository.UpdateTracking (AWB + courier name)
  → Returns Shipment
  → If priority=PRIORITY (Phase 2+): caller can immediately call ManifestService.CreatePerOrderManifest
      → courier.CreateManifest([awb], today) + courier.SchedulePickup
      → ShipmentRepository.UpdateStatus → MANIFESTED, manifest_id set
      → Publish event.ShipmentManifested
```

**Open gap:** `POST /admin/orders/{id}/shipments` route currently not registered in backend router (frontend calls; backend 404s). Phase 4 review flagged.

### AF-2: Daily pickup batch (cron)

**Trigger:** EventBridge schedule `cron(30 11 * * MON-FRI *)` (17:00 IST Mon-Fri) fires `cron-pickup-batch` Lambda. Or admin clicks "Run pickup batch now" → synchronous via admin handler.

```
cron-pickup-batch.Handle:
  → ManifestService.RunDailyBatch(tomorrow_09:00_IST):
      → ShipmentRepository.QueryByPriorityStatus(NORMAL, CREATED, limit=500)
          → Uses orders table priority-status-index GSI
      → If 0 shipments: return BatchResult{ShipmentCount: 0}
      → Else: collect AWBs, courier.CreateManifest(awbs, pickupDate)
          → Single Delhivery POST /api/p/manifest call
      → courier.SchedulePickup(manifestID, pickupLocation, pickupDate)
          → Single Delhivery POST /fm/request/new/ call
      → For each shipment: ShipmentRepository.UpdateStatus → MANIFESTED, manifest_id
          → Tracks marked / failed shipment IDs in BatchResult
      → Publish event.ShipmentPickupScheduled
  → CloudWatch logs result
```

**Concurrency safety:** `ReservedConcurrentExecutions: 1` on Lambda — EventBridge replay cannot fire two batches in parallel.

### AF-3: Daily COD remittance pull (cron)

**Trigger:** EventBridge daily 08:00 IST (`cron(30 2 * * ? *)`) fires `cron-cod-remittance`.

```
cron-cod-remittance.Handle:
  → CODReconciliationService.RunDailyPull(from=24h_ago, to=now):
      → courier.FetchCODRemittances (Delhivery /api/cmu/get_invoice)
      → Group remittance rows by UTR
      → For each UTR group:
          → Compute total amount + max remittedAt up-front (idempotent)
          → For each entry: ShipmentRepository.GetByAWB(awb)
              → Match: OrderRepository.UpdateCODRemittance(orderID, utr, remittedAt)
                  → Atomic SET cod_remitted=true, cod_remittance_ref, cod_remitted_at, updated_at
                  → Conditional attribute_exists(PK) — returns NotFound if order gone
                  → Publish event.CODRemitted
              → No match: collect as unmatched, publish event.CODUnmatched
          → CODRemittanceRepository.Upsert (deterministic ID = uuid.NewSHA1(nil, utr))
              → Status: RECONCILED (all matched) or UNMATCHED
  → CloudWatch logs counts
```

**Idempotency:** Re-pull on same date range produces same `CODRemittance.ID` (SHA1 of UTR) and same `remittedAt` (max-of-group, not running). Upsert overwrites cleanly.

### AF-4: Weekly rate refresh (cron + admin button)

**Trigger A:** EventBridge `cron(30 21 ? * SAT *)` (Sun 03:00 IST) fires `cron-rate-refresh`.
**Trigger B:** Admin clicks "Refresh from Delhivery" on RatesPage → admin order Lambda invokes `cron-rate-refresh` asynchronously via `LambdaInvoker.Invoke` (`InvocationType=Event`).

```
cron-rate-refresh.Handle:
  → RateTableService.Refresh:
      → courier.FetchRateMatrix (75 sequential Delhivery /api/kinko/v1/invoice/charges calls — 5 zones × 5 slabs × 3 modes)
      → ShippingRateRepository.ListAll (Scan, filter entity_type=SHIPPING_RATE)
      → Build set of (zone, slab) keys with source=manual_override → skip those rows
      → For non-overridden rows: BatchUpsert with source=delhivery_api, refreshed_at=now
          → Chunked at 25/batch with UnprocessedItems retry (5 attempts, 100ms linear)
  → Returns RefreshResult{RowsUpdated, RowsSkipped}
```

**Async invoke fallback:** When `RATE_REFRESH_LAMBDA_NAME` env unset (local dev), admin handler falls back to synchronous `rateTable.Refresh` call.

### AF-5: Edit rate (manual override)

**Trigger:** Admin clicks "Edit" on a row in RatesPage.

```
RateEditModal opens (key-remounted on rate change)
  → Form with paise inputs (Prepaid, COD, RTO)
  → On submit: PATCH /admin/shipping/rates/{zone}/{slab}
  → ShippingRateRepository.Upsert with source=manual_override
  → Manual rows skipped by subsequent automatic refreshes (see AF-4)
  → UI invalidates ['shipping', 'rates'] query
```

### AF-6: Webhook ingestion (Delhivery → backend)

**Trigger:** Delhivery POSTs forward or reverse webhook to `/api/v1/store/webhooks/delhivery` (single handler — IsReverse flag in payload routes).

```
handleCourierWebhook reads body (io.LimitReader 1MB), passes headers + body to ShippingService.HandleWebhook:
  → courier.VerifyWebhookSignature (HMAC-SHA256, timing-safe via subtle.ConstantTimeCompare)
      → Failure → return AppError{Code: Unauthorized} → HTTP 401
  → courier.ParseWebhook → WebhookEvent{AWB, Status, Timestamp, IsReverse, NDRReason}
  → If IsReverse: delegate to ReturnService.HandleReverseWebhook (currently logging stub)
  → Else (forward):
      → ShipmentRepository.GetByAWB → Shipment
          → Unknown AWB: log warn, ACK with HTTP 200 (drop replay)
      → courier.ToShipmentStatus(ev.Status) → newStatus
      → If newStatus == sh.Status: ACK (dedupe via state)
      → If ev.Status == NDR: publish event.ShipmentUpdated (NDRService handles separately)
      → ShipmentRepository.UpdateStatus (atomic status + priority_status update)
      → Terminal status side-effects:
          DELIVERED → OrderRepository.UpdateStatus → DELIVERED
          RTO       → OrderRepository.UpdateStatus → RETURNED
  → Return HTTP 200 {status: ok}
```

### AF-7: NDR (failed delivery) handling

**Trigger:** Webhook with `Status=NDR` reaches `ShippingService.HandleWebhook` (or a future cron polls undelivered AWBs).

```
NDRService.HandleNDREvent(awb, reason):
  → ShipmentRepository.GetByAWB
  → nextCount = sh.NDRCount + 1
  → If nextCount >= maxAttempts (default 3):
      → ShipmentRepository.UpdateStatus → NDR_ESCALATED, ndr_escalated=true
      → Publish event.ShipmentNDREscalated → admin queue
  → Else:
      → ShipmentRepository.UpdateStatus → NDR with updates {ndr_count, last_ndr_reason, last_ndr_at}
      → courier.ReAttemptDelivery(awb, REATTEMPT)
      → Publish event.ShipmentNDRReattempted
```

**Admin queue:** `GET /admin/shipping/ndr-queue` returns shipments with `Status=NDR_ESCALATED` via priority-status-index GSI (NORMAL + PRIORITY tiers queried separately, results unioned).

**Admin action:** Per-row `<NDRActionMenu>` exposes Re-attempt / Mark Contacted / RTO. Backend handler returns 501 (Phase 4 stub) — UI shows graceful toast.

### AF-8: Initiate customer return (admin-only)

**Trigger:** Admin clicks "Initiate return" on OrderDetailPage when `order.status === DELIVERED` and within return window (default 7 days).

```
<ReturnFormModal> — admin picks item quantities + reason
  → POST /admin/orders/{id}/returns
  → ReturnService.Create:
      → OrderRepository.GetByID — validate status=DELIVERED, has DeliveredAt
      → Check DeliveredAt + returnWindowDays > now → else 400 "Return window expired"
      → ShipmentRepository.GetByOrderID — locate forward shipment
      → courier.CreateReversePickup (Delhivery /api/cmu/create.json with REV- prefix)
          → Returns ReverseAWB
      → ReturnRepository.Create (PK=ORDER#<id>, SK=RETURN#<uuid>, status=REQUESTED)
          → ConditionalCheckFailed → ErrCodeAlreadyExists
      → Publish event.ReturnRequested
  → Returns ReturnRequest
```

### AF-9: Process refund (admin)

**Trigger:** Admin clicks "Process refund" on a row in ReturnsListPage where status=RECEIVED.

```
<RefundAmountModal> opens with suggested amount prefilled
  → POST /admin/returns/{id}/refund?amount_paise=...
  → ReturnService.ProcessRefund:
      → Currently STUB: returns ErrCodeNotImplemented → HTTP 501
      → Planned: integrate PaymentService.RefundPayment (PhonePe refund API), set Return.Status=REFUNDED + RefundedAt, restore inventory, publish event.ReturnRefunded
```

**Phase 4 review:** `window.prompt` replaced with proper modal. Backend integration deferred.

### AF-10: COD remittance audit

**Trigger:** Admin navigates to `/shipping/cod-remittance`.

```
RatesPage chips filter status (RECONCILED / UNMATCHED / RECEIVED)
  → GET /admin/shipping/cod-remittances?status=...
  → CODRemittanceRepository.ListByStatus
      → Query shipping table entity-status-index GSI (PK=COD_REMITTANCE, SK=status)
  → Admin clicks remittance → CODRemittanceDetail modal
      → GET /admin/shipping/cod-remittances/{id}
      → Shows per-AWB entries with matched flag, unmatched rows highlighted
```

---

## Status state machines

### Shipment.Status

```
CREATED ───┐
           ▼
      MANIFESTED ──→ PICKED_UP ──→ IN_TRANSIT ──→ OUT_FOR_DELIVERY ──→ DELIVERED (terminal)
                                                          │
                                                          ▼
                                                         NDR ──→ (re-attempt) IN_TRANSIT
                                                          │
                                                  (3rd attempt)
                                                          ▼
                                                    NDR_ESCALATED ──→ RTO (terminal alt)

(separate reverse flow)
DELIVERED ──→ RETURNING ──→ RETURNED (terminal)
```

### ReturnRequest.Status

```
REQUESTED ──→ PICKED_UP ──→ IN_TRANSIT ──→ RECEIVED ──→ REFUNDED (terminal)
    │
    └──(admin)──→ CANCELLED (terminal alt)
```

### CODRemittance.Status

```
RECEIVED (initial transient) ──→ RECONCILED (all entries matched, terminal)
                             │
                             └──→ UNMATCHED (≥1 entry unmatched, terminal — needs admin)
```

---

## Event publication sites

| Event | Published by | Trigger |
|-------|--------------|---------|
| `shipment.created` | ShippingService.CreateShipment | New forward shipment |
| `shipment.manifested` | ManifestService (per-order + batch) | Manifest accepted + pickup scheduled |
| `shipment.ndr_reattempted` | NDRService.HandleNDREvent | NDR count < limit, re-attempt requested |
| `shipment.ndr_escalated` | NDRService.HandleNDREvent | NDR count ≥ limit, escalated to admin |
| `shipment.pickup_scheduled` | ManifestService.RunDailyBatch | Daily batch manifest + pickup booked |
| `shipment.updated` | ShippingService.HandleWebhook | Generic status transition (NDR specifically) |
| `cod.remitted` | CODReconciliationService | Per-entry on successful match |
| `cod.unmatched` | CODReconciliationService | Per-entry on AWB lookup failure |
| `return.requested` | ReturnService.Create | Admin creates return |
| `return.refunded` | ReturnService.ProcessRefund | (currently stub — endpoint returns 501) |
| `return.received` | (Phase 5+ via reverse webhook) | Reverse webhook reports REVERSE_DELIVERED |
