# Shipping — Database Schema + Query Patterns

Shipping data lives across **two DynamoDB tables**: the existing `handloom-orders-{env}` (extended with new entities + GSIs) and the **new `handloom-shipping-{env}`** introduced in Phase 1. PostgreSQL is not used for shipping data.

---

## Table: `handloom-shipping-{env}`

Brand-new shipping-side table. Holds rate matrix, pincode cache, COD remittance records.

### Primary keys

| Attribute | Type | Role |
|-----------|------|------|
| `PK` | String | Partition key |
| `SK` | String | Sort key |
| `entity_type` | String | Discriminator for `entity-status-index` GSI |
| `ttl` | Number (Unix seconds) | DynamoDB TTL attribute — auto-deletes expired pincode cache rows |

Billing: `PAY_PER_REQUEST`. PITR enabled. RemovalPolicy: `RETAIN`.

### Entities

#### `ShippingRate` (rate matrix row)

```
PK = "RATE#<zone>#<weight_slab_grams>"   e.g. "RATE#A#500"
SK = "METADATA"
entity_type = "SHIPPING_RATE"
```

Attributes:

| Field | Type | Notes |
|-------|------|-------|
| `zone` | string | A / B / C / D / E |
| `weight_slab_grams` | int | 500 / 1000 / 2000 / 5000 / 10000 |
| `prepaid_paise` | int64 | |
| `cod_paise` | int64 | |
| `rto_paise` | int64 | Return-to-origin charge |
| `refreshed_at` | timestamp | Last update time |
| `source` | string | `delhivery_api` or `manual_override` |
| `created_at`, `updated_at`, `created_by`, `updated_by` | BaseEntity audit fields | |

5 zones × 5 weight slabs = **25 rows** at steady state.

#### `PincodeZone` (pincode → zone cache)

```
PK = "PIN#<pincode>"   e.g. "PIN#560001"
SK = "METADATA"
entity_type = "PINCODE_ZONE"
ttl = <unix seconds, now + 7 days>
```

Attributes:

| Field | Type | Notes |
|-------|------|-------|
| `pincode` | string | 6-digit |
| `zone` | string | A–E, mapped to the rate matrix |
| `city`, `state` | string | |
| `serviceable` | bool | |
| `cod_available`, `prepaid_available` | bool | |
| `refreshed_at` | timestamp | |
| `ttl` | number | Unix seconds, DynamoDB TTL auto-deletes after 7 days |
| BaseEntity audit fields | | |

Unbounded but bounded-in-practice (~30 K Indian pincodes max; TTL drops cold rows).

#### `CODRemittance` (daily UTR payout)

```
PK = "REMIT#<remittance_ref>"   e.g. "REMIT#UTR123456"
SK = "METADATA"
entity_type = "COD_REMITTANCE"
```

Attributes:

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | UUID v5 (deterministic, derived from UTR) for idempotent re-pulls |
| `remittance_ref` | string | UTR / bank ref |
| `amount_paise` | int64 | Total for the UTR group |
| `remitted_at` | timestamp | Max remittedAt across entries |
| `bank_ref` | string | Same as UTR |
| `status` | enum | `RECEIVED` (transient), `RECONCILED`, `UNMATCHED` (terminal) |
| `entries` | list<CODEntry> | Per-AWB breakdown: `{awb, order_id, amount_paise, matched}` |
| BaseEntity audit fields | | |

One row per UTR. `entries` list is bounded by transactions in the payout (~tens to hundreds typical).

### GSI

#### `entity-status-index`

```
PK = entity_type   e.g. "COD_REMITTANCE"
SK = status         e.g. "UNMATCHED"
Projection: ALL
```

Used by admin queries to list COD remittances by reconciliation status (and could serve future status-based queries on `SHIPPING_RATE` if needed).

---

## Table: `handloom-orders-{env}` (extended)

Existing table. Phase 1 added new entities + 2 GSIs. Phase 2 added 1 more GSI.

### New entities

#### `Shipment` (extended in Phase 1)

```
PK = "ORDER#<order_id>"
SK = "SHIPMENT#<shipment_id>"
entity_type = "SHIPMENT"
```

Phase 1 added these fields to the existing `Shipment` struct:

| Field | Type | Notes |
|-------|------|-------|
| `priority` | enum | `NORMAL` / `PRIORITY` |
| `priority_status` | string | Composite GSI key, format `<priority>#<status>` |
| `pickup_location` | string | Warehouse name |
| `manifest_id` | string | Delhivery manifest UUID, set after manifest |
| `ndr_count` | int | NDR re-attempts so far |
| `last_ndr_reason` | string | |
| `last_ndr_at` | timestamp | |
| `ndr_escalated` | bool | True after ≥ maxAttempts |
| `shipping_charge_paise` | int64 | Charged at checkout |
| `actual_weight_grams` | int | Post-pickup actual weight |
| `charged_weight_grams` | int | Delhivery's billed weight |
| `is_cod` | bool | |
| `cod_amount_paise` | int64 | |
| `cod_remitted` | bool | |
| `cod_remitted_at` | timestamp | |
| `cod_remittance_ref` | string | UTR matched against |

GSI key `priority_status` is recomputed atomically in `ShipmentRepository.UpdateStatus` (Phase 2 fix) — caller passes new priority + status, both written in single `UpdateItem`.

#### `ReturnRequest` (new in Phase 1)

```
PK = "ORDER#<order_id>"     (colocated with order)
SK = "RETURN#<return_id>"
entity_type = "RETURN_REQUEST"
```

Attributes:

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | UUID |
| `order_id` | string | Denormalized |
| `shipment_id` | string | Forward shipment being returned |
| `reverse_awb` | string | Delhivery reverse waybill |
| `reverse_shipment_id` | string | |
| `reason` | string | Admin-supplied |
| `items` | list<ReturnItem> | `{product_id, sku, quantity, unit_paise}` |
| `status` | enum | REQUESTED → PICKED_UP → IN_TRANSIT → RECEIVED → REFUNDED (or CANCELLED) |
| `refund_amount_paise` | int64 | |
| `refunded_at` | timestamp | |
| `created_by` | string | Admin user ID |
| BaseEntity audit fields | | |

### Order entity extensions (Phase 2)

Existing `Order` got 3 new attributes for COD reconciliation:

| Field | Type | Notes |
|-------|------|-------|
| `cod_remitted` | bool | |
| `cod_remitted_at` | timestamp | |
| `cod_remittance_ref` | string | UTR |

Written by `OrderRepository.UpdateCODRemittance` (Phase 2). Atomic UpdateItem with `attribute_exists(PK)` condition + `updated_at` bump.

### GSIs (Phase 1 + 2 additions)

#### `priority-status-index` (Phase 1, on orders table)

```
PK = priority_status   e.g. "NORMAL#CREATED"
SK = created_at
Projection: ALL
```

Composite key value `"<priority>#<status>"`. Atomically recomputed on every `UpdateStatus`.

Used by daily pickup batch + admin NDR queue.

#### `awb-index` (Phase 2, on orders table)

```
PK = awb_number
Projection: ALL
```

Used by webhook handler + COD reconciliation to look up shipment by AWB.

---

## Query patterns

### `ShipmentRepository`

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `Create(shipment)` | `PutItem` | base table | New forward shipment |
| `GetByID(orderID, shipmentID)` | `GetItem` PK + SK | base table | Direct lookup |
| `GetByOrderID(orderID)` | `Query` PK=`ORDER#<id>` + `begins_with(SK, "SHIPMENT#")` | base table | Order detail page |
| `GetByAWB(awb)` | `Query` on `awb-index` | awb-index GSI | Webhook + COD reconciliation |
| `UpdateStatus(orderID, shipmentID, priority, status, updates)` | `UpdateItem` (atomic SET status + priority_status + updated_at + extras) | base table | Status transitions — preserves GSI consistency |
| `QueryByPriorityStatus(priority, status, limit)` | `Query` PK=`<priority>#<status>` | priority-status-index GSI | Daily pickup batch (NORMAL+CREATED), NDR queue (PRIORITY+NDR_ESCALATED) |

### `ReturnRepository`

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `Create(rr)` | `PutItem` with `attribute_not_exists(PK)` condition (→ ErrCodeAlreadyExists) | base table | Admin initiates return |
| `GetByID(orderID, returnID)` | `GetItem` | base table | Direct lookup |
| `UpdateStatus(orderID, returnID, status, updates)` | `UpdateItem` via shared `buildDynamicUpdate` helper | base table | Status transitions |
| `ListByOrder(orderID)` | `Query` PK=`ORDER#<id>` + `begins_with(SK, "RETURN#")` | base table | Order detail returns section |
| `ListByStatus(status, limit, cursor)` | `Query` on `entity-status-index` PK=`RETURN_REQUEST` + SK=status | entity-status-index GSI (orders table — note: distinct from shipping table's same-named GSI) | Admin returns queue (Phase 2 also defers cursor) |

### `ShippingRateRepository`

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `Get(zone, slab)` | `GetItem` PK=`RATE#<zone>#<slab>` SK=`METADATA` | base table | Rate lookup at checkout (via `RateTableService.Lookup`) |
| `Upsert(rate)` | `PutItem` | base table | Manual override edit |
| `BatchUpsert(rates)` | `BatchWriteItem` chunked at 25 + retry on UnprocessedItems (5×, 100ms linear) | base table | Weekly rate refresh cron |
| `ListAll()` | `Scan` with `FilterExpression: entity_type = "SHIPPING_RATE"` | base table | Admin RatesPage + rate refresh override detection |

### `PincodeRepository`

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `Get(pincode)` | `GetItem` PK=`PIN#<pin>` | base table | First lookup in `ShippingService.CheckServiceability` |
| `Upsert(pz)` | `PutItem` with TTL attribute | base table | Cache write on miss; TTL auto-evicts after 7 days |

### `CODRemittanceRepository`

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `Get(utr)` | `GetItem` PK=`REMIT#<utr>` | base table | Admin remittance detail |
| `Upsert(rem)` | `PutItem` | base table | Daily COD pull (idempotent — deterministic `id` from `uuid.NewSHA1(nil, utr)`) |
| `ListByStatus(status, limit)` | `Query` on `entity-status-index` | shipping table entity-status-index GSI | Admin filter by RECONCILED / UNMATCHED |

### `OrderRepository` extension (Phase 2)

| Method | Operation | Index | Use case |
|--------|-----------|-------|----------|
| `UpdateCODRemittance(orderID, utr, remittedAt)` | `UpdateItem` SET cod_remitted, cod_remittance_ref, cod_remitted_at, updated_at + `attribute_exists(PK)` condition (→ NotFound on missing order) | base table | Per-entry write during COD reconciliation |

---

## Idempotency + correctness notes

### Atomic `priority_status` writes (Phase 2 fix)

Pre-Phase-2 `UpdateStatus(...)` wrote `status` only; `priority_status` GSI key would drift on every status change. Phase 2 changed the signature to accept `priority` and write both attributes atomically in one `UpdateItem`. This ensures the `priority-status-index` GSI is always consistent — `QueryByPriorityStatus` never misses a shipment that has transitioned.

### Deterministic COD remittance ID

`CODRemittance.ID = uuid.NewSHA1(uuid.Nil, []byte(utr))`. Re-running `cron-cod-remittance` for the same date range produces the same `ID` and the same `Upsert` writes the same row. The `entries` list is recomputed from the same source data, so matched flags are stable. Effective re-pulls are idempotent.

### Webhook replay (open)

Currently webhooks dedupe by status comparison: if new status equals existing, skip. Works for most replays. **Not safe for NDR**: two NDR replays before status persists could double-increment `ndr_count`. Mitigation tracked as Phase 5+ open item: maintain a `processed_webhook_ids` set with `(awb, event.timestamp)` keys and 7-day TTL.

### Conditional creates

`ReturnRepository.Create` uses `ConditionExpression: attribute_not_exists(PK)` → `ErrCodeAlreadyExists` on conflict. Prevents duplicate-create races.

`OrderRepository.UpdateCODRemittance` uses `attribute_exists(PK)` → `ErrCodeNotFound` on missing order. Prevents ghost-row writes if COD payout references an order that was deleted/refunded.

### Cursor pagination

`ReturnRepository.ListByStatus` exposes cursor (base64 of DynamoDB `LastEvaluatedKey`). `CODRemittanceRepository.ListByStatus` does not (limit-only); admin volume is low enough today.

`ShippingRateRepository.ListAll` is unpaginated `Scan` — fine at 25 rows. Will need pagination if pincode rows scale (currently filtered out via `entity_type=SHIPPING_RATE`).

---

## Capacity + cost characteristics

| Table | Steady state size | Hot keys | Notes |
|-------|------------------|---------|-------|
| `handloom-shipping-{env}` ShippingRate | 25 rows | Per zone+slab on rate lookup at checkout | Tiny; PAY_PER_REQUEST sub-cent |
| `handloom-shipping-{env}` PincodeZone | ~30 K rows max (Indian pincodes) | Customer pincode on cart entry | TTL prunes; estimate 5–10 K active |
| `handloom-shipping-{env}` CODRemittance | ~30 rows/month (one per UTR) | None — read by admin only | Negligible storage |
| `handloom-orders-{env}` Shipment | Grows with orders | `awb-index` on webhook ingestion | Same RCU profile as orders |
| `handloom-orders-{env}` ReturnRequest | Grows with returns (<1% of orders) | None — looked up per-order via SK prefix | Trivial |

GSI cost: `priority-status-index` + `awb-index` + `entity-status-index` on orders. PAY_PER_REQUEST so cost scales with writes/reads, not provisioned capacity.

---

## Migration / rollout notes

- Phase 1 CDK provisions `handloom-shipping-{env}` with `RemovalPolicy_RETAIN` — accidental stack-destroy preserves data.
- LocalStack seed script (`scripts/init-local-db.sh`) creates `handloom-shipping-local` for dev.
- Phase 2 + 3 added `awb-index` and `priority-status-index` GSIs on orders table — backfill on existing rows happens transparently via DynamoDB. Existing orders without `priority` or `priority_status` will not appear in `priority-status-index` queries (works for newly created shipments).
- Backward compat: deleting Shiprocket gateway in Phase 2 did not require data migration — Shiprocket never reached production. Existing `Shipment.Provider` field carries `"delhivery"` for all new rows.
