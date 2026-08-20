# Refund Flow — Design

**Goal:** let an admin refund a customer, in full or per line item, when Homechrome cannot
serve part or all of an order — and actually move the money through PhonePe rather than
just marking a status.

**Today:** `POST /admin/orders/{id}/refund` exists and calls `OrderService.RefundOrder`,
which sets `PaymentStatus = REFUNDED` and returns. No gateway call, no partial tracking,
no record of what was refunded or why. `PaymentService.RefundPayment` returns
`"Refund functionality is not implemented yet"`. `Payment.RefundAmount` and
`Payment.RefundedAt` exist and are never written. The admin frontend has an
`ordersApi.refund` client with no UI that calls it.

**Scope:** admin-initiated refunds only. Customer-initiated refund requests, return
merchandise authorisation, and automatic refunds on cancellation are out of scope.

---

## Decisions

| Question | Decision |
|---|---|
| How does the admin choose what to refund? | Line items + quantity. The backend derives the amount; the client never sends one. |
| How is the async refund settled? | Persist PENDING, settle on webhook, plus a manual re-check endpoint for missed webhooks. |
| Order state after a partial refund | Per-item refunded quantity; new `PaymentStatus` value `PARTIALLY_REFUNDED`. Order keeps its fulfilment status so the rest still ships. |
| Inventory per refunded line | Admin picks restock or write-off per line. Default write-off — "cannot serve" usually means the units are not there. |
| Refunds after dispatch | Allowed, but money only. `RETURNED` already owns post-dispatch restocking. |
| Who may refund | `ADMIN` role only. Audit record per refund. |

---

## Provider contract (verified against PhonePe docs, August 2026)

Standard Checkout v2. Base URL is the existing `phonepe.Config.BaseURL`.

**Initiate** — `POST {base}/payments/v2/refund`, header `Authorization: O-Bearer <token>`

```json
{ "merchantRefundId": "...", "originalMerchantOrderId": "...", "amount": 1234 }
```

Response: `{ "refundId": "...", "amount": 1234, "state": "PENDING" }`.

**Status** — `GET {base}/payments/v2/refund/{merchantRefundId}/status`

Response adds `originalMerchantOrderId`, `errorCode`, `detailedErrorCode`, and `rail` /
`instrument` objects.

**Webhooks** — `pg.refund.completed` and `pg.refund.failed`. There is no `accepted`
event; a refund goes PENDING → terminal. Payload: `originalMerchantOrderId`, `refundId`,
`amount`, `state`, `timestamp`, plus `errorCode` / `detailedErrorCode` on failure.

These arrive on the **existing** `/api/v1/store/webhooks/phonepe` route under the same
SHA256 `Authorization` verification already implemented. No new endpoint, no new
credentials.

**Correlation constraint.** The webhook identifies a refund by `originalMerchantOrderId`
plus PhonePe's `refundId` — it does **not** echo our `merchantRefundId`. Since an order
can have several partial refunds, `originalMerchantOrderId` alone is ambiguous. We store
PhonePe's `refundId` from the initiation response and match on it. When that response is
lost to a timeout we hold a `merchantRefundId` with no `refundId`; the status endpoint is
keyed on `merchantRefundId`, which is exactly the recovery path.

`amount` is in **paise**, consistent with the rest of the codebase.

---

## Data model

### New entity: `Refund`

Stored in the orders table (`TableOrders`), keyed to match `Payment` so no new index
patterns are introduced:

```
PK     = REFUND#<id>            SK     = METADATA
GSI1PK = ORDER#<orderID>        GSI1SK = REFUND#<initiatedAt>   → refunds for an order
GSI2PK = REFUND_TXN             GSI2SK = <merchantRefundID>     → lookup for status re-check
```

| Field | Type | Notes |
|---|---|---|
| `ID` | string | `refund_<uuid>` |
| `OrderID`, `PaymentID`, `CustomerID` | string | |
| `Amount` | int64 | paise, derived server-side |
| `Status` | `RefundStatus` | `PENDING` / `COMPLETED` / `FAILED` |
| `MerchantRefundID` | string | ours, unique per attempt |
| `ProviderRefundID` | string | PhonePe's `refundId`, empty until initiation returns |
| `Reason` | string | from a bounded set, see below |
| `Items` | `[]RefundItem` | |
| `ErrorCode`, `DetailedErrorCode` | string | populated on failure |
| `InitiatedAt`, `CompletedAt` | time | |
| `CreatedBy` | string | admin user ID |

`RefundItem`: `OrderItemID`, `ProductID`, `Quantity`, `Amount`, `Restock bool`
(`true` = return to sale, `false` = write off).

A separate entity is required rather than fields on `Payment` because partial refunds
mean many refunds per payment. The existing unused `Payment.RefundAmount` and
`Payment.RefundedAt` become the running total and the full-refund timestamp.

### Changes to existing types

- `OrderItem` gains `RefundedQuantity int` — how much of that line has been refunded.
- `PaymentStatus` gains `PaymentStatusPartiallyRefunded = "PARTIALLY_REFUNDED"`.
- `internal/validator/custom_rules.go` status sets gain the same value.
- Both frontends' `PaymentStatus` unions gain `'PARTIALLY_REFUNDED'`
  (`handloom-admin-frontend/src/features/orders/types.ts`,
  `homechrome-store/src/types/index.ts`), and the storefront's status-colour maps get an
  entry. Per `CLAUDE.md`, these ship in the same PR as the API change.

`Reason` is a bounded set so it can label metrics without unbounded cardinality:
`OUT_OF_STOCK`, `DAMAGED`, `CUSTOMER_REQUEST`, `PRICING_ERROR`, `OTHER`. Free text goes
in an order note, not the reason field.

---

## Amount derivation

The client sends line IDs and quantities only. The server computes the amount; a
client-supplied amount is rejected. Money must not be a client input.

For each requested line:

```
line_refund = order_item.unit_price × requested_quantity
```

Discount is prorated by the line's share of the order subtotal:

```
line_discount = round(order.discount_amount × line_subtotal / order.subtotal)
```

Tax is prorated by the same rule. Note that `CheckoutService` currently sets both
`taxAmount` and `shippingAmount` to zero for every order, so today only the discount term
does any work — the tax and shipping terms are written so the formula stays correct if
either starts being charged.

Shipping is refunded **only** when the refund clears the last unrefunded unit in the
order — a partial refund keeps it, since the parcel still ships.

```
refund_total = Σ(line_refund − line_discount + line_tax) [+ shipping if final]
```

**Rounding.** All arithmetic is integer paise. Proration uses round-half-up per line, and
the refund that clears the last unrefunded unit absorbs any accumulated residual, so the
sum of all refunds against an order equals `order.TotalAmount` exactly. Without this,
prorated rounding leaves an order permanently a few paise short of fully refunded and it
never reaches `REFUNDED`.

**Validation, all server-side:**

- The order has a payment in `PAID` / `SUCCESS`.
- Each line exists on the order, and `requested_quantity ≤ quantity − refunded_quantity`.
- `Σ existing refunds + this refund ≤ order.TotalAmount`.
- At least one line, quantity ≥ 1.

---

## Gateway

`phonepe.Gateway` gains two methods; `Client` and `DevClient` both implement them.

```go
InitiateRefund(ctx context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error)
CheckRefundStatus(ctx context.Context, merchantRefundID string) (*RefundStatusResponse, error)
```

`RefundResponse{ RefundID, Amount, State }`,
`RefundStatusResponse{ OriginalMerchantOrderID, RefundID, Amount, State, ErrorCode, DetailedErrorCode }`.

Both reuse the existing OAuth `getToken` and the instrumented HTTP client. `DevClient`
returns `state: COMPLETED` immediately and prints the refund to the console, matching how
it handles payments.

`originalMerchantOrderID` is the `Payment.MerchantTransactionID` recorded at checkout —
the same value passed as `merchantOrderId` when the payment was created.

---

## Service

New `RefundService` (`internal/service/refund_service.go`) implementing
`domain.RefundService`. Refunds are their own concern with their own lifecycle; folding
them into `OrderService`, already the largest service in the codebase, would make it
worse.

### Create

1. Load order and payment. Reject unless the payment succeeded.
2. Validate lines and quantities; compute the amount.
3. Build the `Refund` with `Status = PENDING` and a fresh `MerchantRefundID`, and
   **persist it before calling PhonePe.** A refund that leaves the building without a
   local record is unreconcilable; a PENDING record whose gateway call never happened is
   recoverable through the status endpoint.
4. Call `InitiateRefund`. On transport failure, mark the record `FAILED` with the error
   and return — no inventory effect.
5. On success, store `ProviderRefundID`. Status stays `PENDING`.
6. Apply the inventory effect (below) — after the record is persisted, never before.
7. Write an audit record via `AuditService.Log` with the refund ID, order ID, amount and
   per-line breakdown.
8. Record metric `refund_initiated` labelled by `reason` (bounded) and `gateway`.

### Settlement

`HandleRefundCompleted` / `HandleRefundFailed`, called from the existing webhook handler.

On completed:

- No-op if the refund is already terminal (see idempotency below).
- Mark `COMPLETED`, set `CompletedAt`.
- Increment `Payment.RefundAmount`; set `Payment.RefundedAt` when it reaches
  `order.TotalAmount`.
- Increment `OrderItem.RefundedQuantity` for each refunded line.
- Set order `PaymentStatus` to `REFUNDED` when fully refunded, otherwise
  `PARTIALLY_REFUNDED`. **Order status is not changed** — the unrefunded remainder still
  ships.
- Send the customer notification using the existing `NotificationTriggerRefund`, which
  already has a subject line and is currently unreachable.
- Record `refund_completed`.

On failed: mark `FAILED` with `ErrorCode` / `DetailedErrorCode`, record `refund_failed`,
leave order and payment untouched. **Inventory is not reversed** — see Known Gaps.

### Idempotency

Built in from the start. PhonePe retries webhooks, and Lambda can process two deliveries
concurrently.

- Settlement reads the refund and returns early if `Status != PENDING`.
- That read is only an optimisation. The authority is the refund record's own conditional
  update, `ConditionExpression: status = PENDING`, so of two concurrent deliveries exactly
  one wins and the loser fails the condition instead of double-applying.
- Every downstream effect — payment total, item quantities, order status, notification —
  runs **only** on the delivery that won that condition. The refund record is the single
  gate.
- `Payment.RefundAmount` is then incremented with a DynamoDB `ADD` update expression
  rather than read-modify-write, so concurrent settlements of *different* refunds against
  the same payment cannot lose an increment. (A condition on the refund item cannot guard
  a write to the payment item; only a transaction could, and the `ADD` plus the single
  gate above make one unnecessary.)

The inventory backlog (`2026-08-18-inventory-integrity-backlog.md`, item 1) documents
what a read-then-write settlement costs on an existing money path. This is new code;
there is no reason to inherit that.

### Re-check

`RecheckStatus(refundID)` loads the refund, calls `CheckRefundStatus` with the stored
`MerchantRefundID`, and applies the same settlement logic. This is the escape hatch for a
webhook that never arrived, and the recovery path when `ProviderRefundID` was never
stored because the initiation response was lost.

---

## Inventory interaction

Only refunds on orders **not yet dispatched** move stock. Dispatch is the dividing line
because `CommitStock` consumes the reservation at `SHIPPED`.

| Order status | Line marked | Effect |
|---|---|---|
| PENDING / CONFIRMED / PROCESSING | restock | `ReleaseStock` — reservation returns to sale |
| PENDING / CONFIRMED / PROCESSING | write off | `WriteOffStock` — reservation released *and* on-hand reduced |
| SHIPPED / DELIVERED | either | none — `RETURNED` owns post-dispatch restocking |

Post-dispatch refunds move money only. If the goods physically come back, the admin marks
the order `RETURNED`, which restocks through the existing path. Letting a refund also
restock would double-count against that transition.

### New repository method: `WriteOffStock`

Write-off is two mutations — release the reservation, then reduce on-hand. Doing it as
`ReleaseStock` followed by `RemoveStock` is not atomic: a crash between them leaves the
units released and back on sale, which is the opposite of a write-off.

```go
WriteOffStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)
```

One transaction, `SELECT ... FOR UPDATE`, `reserved_qty -= q` and `quantity -= q` (so
`available_qty` is unchanged — the units were already unavailable while reserved, and now
they do not exist). Guards mirror `CommitStock`: `reserved_qty >= q && quantity >= q`.
One ledger row, new type `WRITE_OFF`, `reference_id = orderID`.

This is arithmetically identical to `CommitStock` but semantically distinct, and the
ledger must be able to tell a dispatch from a write-off. The frontend
`TransactionType` union gains both `'COMMIT'` (already outstanding from the inventory
branch) and `'WRITE_OFF'`.

---

## API

All under the existing admin order router, wrapped in
`authMiddleware.RequireRole(domain.UserRoleAdmin)` following `internal/router/audit.go`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/admin/orders/{id}/refunds` | Create a refund |
| `GET` | `/admin/orders/{id}/refunds` | List refunds for an order |
| `POST` | `/admin/orders/{id}/refunds/{refundID}/recheck` | Force a provider status re-check |

The re-check route is nested under the order rather than given its own `/admin/refunds`
router so all three live on the existing order router and the existing order Lambda — no
new route tree, no new Lambda, no CDK change.

Request body:

```json
{
  "reason": "OUT_OF_STOCK",
  "items": [
    { "order_item_id": "...", "quantity": 2, "restock": false }
  ]
}
```

Validated with `middleware.ValidateJSONTyped[CreateRefundRequest]` per house convention.

**Removed:** `POST /admin/orders/{id}/refund` (singular) and
`OrderService.RefundOrder`. The stub sets `PaymentStatus = REFUNDED` with no gateway
call, so leaving it in place is a route that silently lies about money having moved.
`domain.OrderService.RefundOrder`, `RefundOrderRequest`, its tests, and the frontend's
unused `ordersApi.refund` go with it.

`PaymentService.RefundPayment` is likewise removed from `domain.PaymentService` rather
than implemented — `RefundService` owns this.

---

## Wiring

`RefundService` is needed in two places, because creation and settlement run in different
Lambdas:

| Wire initializer | Lambda | Needs it for |
|---|---|---|
| `InitializeOrderDeps` | `cmd/lambda/order` | create, list, re-check |
| `InitializeStoreWebhooksDeps` | `cmd/lambda/store-webhooks` | `pg.refund.*` settlement |
| `InitializeMonolithDeps` | `cmd/api` | both, locally |

Settlement needs the refund, order, payment and customer repositories plus the
notification service. It does **not** need the inventory repository — inventory moves at
creation time, never at settlement — though `InitializeStoreWebhooksDeps` already wires
`ProvideInventoryRepository` for `PaymentService.releaseOrderInventory`, so this adds no
new dependency to that Lambda either way.

`NotificationService` is *not* currently wired into the webhooks Lambda. Sending the
refund notification from settlement therefore adds it, along with whatever repositories
it needs — the one genuinely new dependency in this design. If that proves awkward, the
fallback is to send the notification from the re-check path only and accept that
webhook-settled refunds notify late.

`make wire` after adding the providers, `make generate-mocks` after changing the domain
interfaces.

## Admin UI

`handloom-admin-frontend`, on the existing order detail page.

**Refund modal** — a line table: product, ordered quantity, already-refunded quantity,
a quantity stepper capped at the remainder, and a restock / write-off toggle per line
defaulting to write-off. Below it, a live breakdown — line subtotal, prorated discount,
shipping (shown as retained unless the refund is final) and the total. Reason is a select
over the bounded set. The computed total is **display only**; the server recomputes it.

Post-dispatch orders show an inline note that the refund moves money only and that
returning stock is done by marking the order `RETURNED`.

**Refund list** on the order detail page: amount, status badge, reason, who and when,
with a re-check button on anything still `PENDING`.

The Refund action is hidden for non-`ADMIN` users, matching the server guard.

---

## Testing

- **Amount derivation** — table-driven unit tests: single line, multiple lines, full
  refund including shipping, proration rounding, the residual-absorption rule, and the
  cap against an already-partially-refunded order.
- **Service** — mocked gateway and repositories: happy path, gateway transport failure
  marks FAILED with no inventory effect, quantity exceeding the remainder is rejected,
  refund on an unpaid order is rejected, per-line restock vs write-off dispatch the right
  repository calls, post-dispatch refund makes no inventory call.
- **Idempotency** — duplicate `pg.refund.completed` applies once; a settlement racing a
  terminal state is a no-op.
- **Gateway client** — `httptest` server asserting method, path, `O-Bearer` header and
  body shape for both new calls, mirroring `internal/gateway/phonepe/client_test.go`.
- **Webhook handler** — both refund events routed correctly, unknown events still ignored.
- **`WriteOffStock`** — repository integration test against PostgreSQL, in the style of
  `TestInventoryRepository_CommitStock`, asserting both columns move and `available_qty`
  holds. This runs in CI now that the backend job has a database.

---

## Known gaps, accepted

- **A failed refund does not reverse its inventory effect.** Stock is moved when the
  refund is initiated, but PhonePe may fail it minutes later. A written-off line stays
  written off. Accepted because the write-off reflects a physical fact — the goods are
  not there — that a payment failure does not change. The admin corrects stock manually
  if the refund is genuinely retried.
- **No automatic retry of failed refunds.** The admin creates a new refund. Automatic
  retry would need the same idempotency guarantees against PhonePe that a fresh
  `merchantRefundId` sidesteps.
- **The in-flight bound is best-effort, and that is the decision.** `Create` folds
  every non-failed refund into the remainder, which closes the window between
  creation and settlement that the settled counters left open. It does not close it
  entirely: `ListByOrder` reads GSI1, a GSI is always eventually consistent, and no
  conditional write serialises two creates, so two attempts inside the replication
  lag can each fail to see the other.

  Accepted, not deferred. Refunds are raised by hand by a single admin, so two
  creates landing within a GSI replication lag is not a path the product has. The
  fix — carrying the claim on a consistently-readable item, an `ADD` counter on the
  payment bounded by a `ConditionExpression` — would change how a refund is
  recorded rather than how it is priced, and buys nothing against a single operator.
  Revisit only if refunds ever become concurrent: more than one admin raising them,
  an automated retry, or a bulk tool.
- **`Customer.TotalSpent` is not adjusted.** It is documented as gross order value; a
  refund does not reduce it. Changing that is a reporting decision beyond this scope.
- **Refunds are per order, not per payment attempt.** An order has one successful
  payment, so this holds today; it would need revisiting if split payments arrive.
