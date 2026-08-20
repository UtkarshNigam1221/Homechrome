# Observability Runbook

## Where each signal lives

| Signal | Backend | How to read |
|---|---|---|
| **Traces** | Grafana Cloud **Tempo** (via OTel collector) | Explore → Tempo, search by Trace ID |
| **Logs** | Grafana Cloud **Loki** (via OTel collector) | Explore → Loki, filter by `service` / `trace_id` |
| **Metrics** | Self-hosted **Postgres `metric_counters`** (Neon) | Admin **Dashboards** (Funnel/Geography/Products/RUM) or SQL on Neon |

Metrics are NOT in Grafana/Mimir. They flow:
`request → in-process buffer (pkg/metrics) → flush → SQS (METRICS_QUEUE_URL) → worker-metrics-consumer → metric_counters`.
Storefront browser events (RUM + funnel beacons) post to `/api/v1/store/events` → `handloom-store-events` → same SQS path.
Dashboards read `metric_counters` directly from the browser via the **Neon Data API** (JWT from Neon Auth, RLS-scoped).

## Triage flow — customer reports an error

1. Get the **Reference ID** from the user (shown on the error page as `Reference: <trace_id>`).
2. Grafana → Explore → Tempo, search by Trace ID.
3. Open the trace. Root span shows route, status, duration. Drill into red spans to find the failing layer (handler → service → DB → gateway).
4. Click "Logs for this span" — Grafana auto-links to Loki filtered by `trace_id`. Every log line the request emitted.
5. If a gateway span is red, check the same time window across users — was the gateway degraded for everyone, or just this one?

## Diagnosing a flow — what to check

Each flow lists: the **metrics** it emits (in `metric_counters`) + the dashboard that surfaces them, the **service** (Lambda / `OTEL_SERVICE_NAME`) for Loki/Tempo, and the **log signal** to grep when the metric is flat or wrong.

### Browse / top-of-funnel
- **Metrics:** `site_visitor` (labels: `city,country,device_type,utm_*`), `product_viewed` (`product_id,category_id,device_type`), `category_viewed`, `catalog_filter_applied` (`filter_key`).
- **Dashboards:** Funnel (top), Products, Geography.
- **Services:** `handloom-store-events` (beacons), `handloom-store-catalog` (SSR/data).
- **Symptom → check:** funnel top flat → confirm beacons arriving: `{service="handloom-store-events"} |= "rum_page_view"` is the wrong layer (metrics aren't logged) — instead verify the beacon endpoint is 2xx: `{service="handloom-store-events"} | json | status>=400`, and that the storefront analytics client is enabled (`NEXT_PUBLIC_ANALYTICS_ENABLED != "false"`).

### Cart
- **Metrics:** `cart_added` (first item, `country,device_type`), `item_added_to_cart` (every add, `product_id,country,device_type`), `cart_item_removed` (`product_id`).
- **Dashboard:** Funnel. **Service:** `handloom-store-cart`.
- **Symptom → check:** adds not counted → `{service="handloom-store-cart"} | json | level="ERROR"`; cart reads hit DynamoDB, so check `db_query` errors too.

### Checkout
- **Metrics:** `checkout_initiated` (`country,city,device_type`), `shipping_cost_shown` (`country`, sum = shipping paise), `orders_placed` (`country,city`), `orders_value` (sum = total paise, `gateway`), `cart_size` (`country,bucket`).
- **Dashboard:** Funnel. **Service:** `handloom-store-checkout`.
- **Symptom → check:** orders placed but not paid → that's the payment flow below. Checkout 5xx → `{service="handloom-store-checkout"} | json | status>=500`; serviceability/shipping calls show as `http_client_call{target_host=~".*shiprocket.*"}`.

### Payment
- **Metrics:** `payment_initiated` (`gateway,country`, at initiate), `payment_completed` (`gateway,country,city,device_type,utm_source`, on webhook success), `payment_outcome` (`gateway,outcome=success|failed|pending,country`, all webhook paths), `cart_to_payment_duration` (`bucket`).
- **Dashboard:** Funnel (Checkout→Pay-Initiated and Pay-Completion ratios).
- **Services:** `handloom-store-checkout` (initiate) + `handloom-store-webhooks` (PhonePe callback).
- **Symptom → check:**
  - `payment_initiated` high, `payment_completed` low → drop-off at gateway OR webhook not arriving. Check `{service="handloom-store-webhooks"} |= "Payment"` for `Payment completed` / `Payment failed` lines; confirm PhonePe is hitting the webhook (CloudFront/API GW access).
  - Webhook signature failures → `{service="handloom-store-webhooks"} | json | level="ERROR" |= "signature"`.
  - PhonePe degraded → `http_client_call{target_host=~".*phonepe.*", status_class!="2xx"}` and the gateway span in Tempo.
  - Idempotent replays are expected (no double-count) — `resolvePayment` logs `Payment already processed, skipping`.

### Purchase analytics (post-payment KPIs)
- **Metrics:** `product_purchased` (sum = line paise, `product_id,category_id`), `coupon_redeemed` (sum = discount, `coupon_code`), `customer_first_purchase` (`country,city,device_type,utm_source`), `repeat_purchase` (`country,city,device_type`).
- **Dashboards:** Products, Funnel (attribution). **Service:** `handloom-store-webhooks` (fired from `recordPurchaseAnalytics` after payment confirmed).
- **Symptom → check:** these only fire on webhook success — if absent but payments succeed, the post-payment block is erroring: `{service="handloom-store-webhooks"} | json | level="ERROR" |= "increment order count"`.

### Auth / OTP
- **Metrics:** `session_started` (`is_new_user`), `otp_outcome` (`outcome=sent|send_failed|verified|verify_failed`).
- **Dashboard:** Funnel (auth section). **Service:** `handloom-store-auth`.
- **Symptom → check:** `otp_outcome{outcome="send_failed"}` rising → MSG91 issue: `http_client_call{target_host=~".*msg91.*"}` + `{service="handloom-store-auth"} | json | level="ERROR"`.

### Coupons
- **Metric:** `coupon_applied` (`coupon_code,outcome=valid|rejected`). **Service:** `handloom-store-checkout`/`-cart`.
- **Symptom → check:** unexpected rejects → `{service=~"handloom-store-(checkout|cart)"} |= "coupon"`.

### Inventory
- **Metrics:** `inventory_out_of_stock` (`product_id`, admin stock removal crosses 0), `inventory_low_stock` (`product_id`, crosses threshold), `out_of_stock_shown` (`product_id`, storefront impression beacon), `back_in_stock_notify_requested` (`product_id`).
- **Services:** `handloom-inventory` (admin) for the first two; `handloom-store-events` for the beacons.
- **Stranded reservations:** the weekly **Inventory Reconciliation** workflow fails when stock is held against orders that never settled. See [Stranded reservations](#stranded-reservations) below.
- **Per-product history:** Inventory → a product's **Stock history** shows every movement and its effect on both counters.

### RUM (real-user monitoring)
- **Metrics:** `rum_lcp` / `rum_inp` / `rum_cls` / `rum_ttfb` (`bucket=good|needs_improvement|poor, page_type, device_type`), `rum_js_error` (`page_type,error_type`), `rum_page_view` (`page_type,device_type`).
- **Dashboard:** RUM. **Service:** `handloom-store-events`.
- **Symptom → check:** Web-Vitals degraded → break down by `page_type` on the RUM dashboard; JS error spike → `rum_js_error` grouped by `error_type`.

### Geography
- **Metrics:** `site_visitor` (`city,country`) joined to the `city_centroids` table for map dots; `centroid_upsert_error` (`reason`) signals first-sighting writes failing.
- **Dashboard:** Geography. **Service:** `handloom-store-events`.
- **Symptom → check:** map empty but visitors counted → `centroid_upsert_error` present, or `city_centroids` RLS grant missing (see migrations 009/010 manual steps).

### Golden signals (every service)
- **Metrics:** `http_request` (`service,method,route,status_class`), `http_request_duration` (`bucket`), `http_client_call`/`http_client_duration` (`target_host` — outbound gateways), `db_query`/`db_query_duration` (`operation,status`), `aws_sdk_call` (`sdk_service,operation,status`).
- **Use:** error rate = `http_request{status_class="5xx"}`; latency = `http_request_duration` bucket distribution; dependency health = `http_client_call` / `aws_sdk_call` non-2xx by host/service.

## Metrics pipeline health (metric is flat / missing)

A missing metric is usually pipeline, not the flow. Check in order:
1. **Was it emitted?** Confirm the request itself succeeded (its Loki logs / trace). Metrics are best-effort and never logged on success.
2. **SQS publish failed** (request path): `{service=~"handloom-.+"} |= "metrics: failed to publish to SQS"`. The full payload is logged for recovery.
3. **Consumer failed**: `{service="handloom-worker-metrics-consumer"} | json | level="ERROR"`. Check the metrics SQS **DLQ depth** in the AWS console.
4. **Centroid writes**: `centroid_upsert_error` metric / `{service="handloom-store-events"} |= "centroid upsert failed"`.
5. **Dashboard empty but `metric_counters` has rows** → frontend auth, not the pipeline: Neon Auth session (NeonAuthGate), Data API RLS grants (migrations 009/010), or `VITE_NEON_DATA_API_URL` unset. The browser console logs a loud error if rows are truncated past the 10k row cap.

## Querying metric_counters (SQL)

Dashboards use the Neon Data API (`GET /rest/v1/metric_counters?metric=eq.<name>&bucket_start=gte.<iso>`). For ad-hoc triage, run SQL in the Neon console. Schema: `metric, labels (jsonb), bucket_start, count, sum_value, retention_class`.

```sql
-- Payment success vs failure, last 6h
SELECT labels->>'outcome' AS outcome, sum(count)
FROM metric_counters
WHERE metric = 'payment_outcome' AND bucket_start >= now() - interval '6 hours'
GROUP BY 1;

-- Checkout funnel counts, last 24h
SELECT metric, sum(count)
FROM metric_counters
WHERE metric IN ('site_visitor','cart_added','checkout_initiated','payment_initiated','payment_completed')
  AND bucket_start >= now() - interval '24 hours'
GROUP BY metric;

-- Revenue (paise) by country, last 24h  — sum_value holds money/duration
SELECT labels->>'country' AS country, sum(sum_value) AS paise
FROM metric_counters
WHERE metric = 'orders_value' AND bucket_start >= now() - interval '24 hours'
GROUP BY 1 ORDER BY paise DESC;

-- RUM LCP bucket mix by page, last 1h
SELECT labels->>'page_type' AS page, labels->>'bucket' AS bucket, sum(count)
FROM metric_counters
WHERE metric = 'rum_lcp' AND bucket_start >= now() - interval '1 hour'
GROUP BY 1, 2;
```

## Common Loki queries

```
{service="handloom-store-checkout"} |= "ERROR"
{service="handloom-store-checkout"} | json | trace_id="<id>"
{service=~"handloom-store-.+"} |= "payment" | json | level="ERROR"
{service=~"handloom-.+"} |= "metrics: failed to publish to SQS"
{service="handloom-worker-metrics-consumer"} | json | level="ERROR"
```

## Common Tempo queries

- By trace ID: paste the ID into the search bar.
- By service + status: `{ resource.service.name = "handloom-store-checkout" && status = error }`
- By duration: `{ duration > 2s && resource.service.name =~ "handloom-.+" }`
- Outbound gateway spans: `{ name =~ "phonepe.*|shiprocket.*|msg91.*" && status = error }`

## Operating notes

- **Trace correlation:** every log line carries `trace_id` and `span_id` (injected by `pkg/slogx`). Service code does not thread these manually.
- **PII redaction is two-layer:** in-process slog handler (`pkg/slogx/redact.go`) scrubs known PII keys; collector-side `attributes/redact` processor (`handloom-admin/infra/configs/otel-collector.yaml`) scrubs span/log attribute keys before export. New PII field? Update both denylists.
- **Emergency kill switch:** set `OTEL_SDK_DISABLED=true` on a Lambda to disable OTel SDK init (traces/logs export). Falls back to slog → stdout → CloudWatch. Metrics (SQS pipeline) are unaffected.
- **Metric cardinality budget:** label KEYS in `metrics.L{...}` are governed by `pkg/metrics/cardinality_test.go` — only keys in its `AllowedLabels` set are permitted, whether written as a literal or a string constant. Adding a label is a deliberate decision (permanent PG rows). High-cardinality values (`user_id`, `order_id`, `email`, `phone`) must never be label values; bounded values only.

## Acceptance test (run after every deploy)

- [ ] `curl -i https://dev-api.homechrome.in/health` → response carries `X-Request-ID` / trace headers.
- [ ] Within 30s the trace is visible in Grafana → Tempo; "Logs for this trace" shows ≥1 Loki line with the same `trace_id`.
- [ ] Place a test order. Trace tree: `browser → nextjs SSR → handloom-store-checkout → handloom-order → ddb.PutItem`.
- [ ] `metric_counters` shows a fresh `orders_placed` row: `SELECT * FROM metric_counters WHERE metric='orders_placed' ORDER BY bucket_start DESC LIMIT 1;`
- [ ] Funnel dashboard renders (Neon Auth login works, Data API returns rows).
- [ ] CW Log Group `/aws/lambda/handloom-store-checkout-dev` has `RetentionInDays=1`.
- [ ] Force a 500 in dev → trace red, Loki ERROR with `trace_id`.
- [ ] Search Loki for "password" / "Bearer " over the last hour → zero results (redaction working).

## Stranded reservations

**Signal:** the **Inventory Reconciliation** workflow fails. It runs Sundays 02:30 UTC against dev and prod, and exits non-zero only when units are actually stranded. A failed *read* is reported separately — unknown is not the same as clean.

### What it means

A reservation settles when the same order either dispatches the stock (`COMMIT`) or gives it back (`RELEASE`). One that does neither is stock held against an order that never shipped and never cancelled. Those units are unsellable, and **no order transition will free them**: from `SHIPPED` the only routes are `DELIVERED`, which has no inventory effect, and `RETURNED`, which restocks only what actually committed.

This is not `inventory_mutation_failed`. That says a movement failed. This says what is stuck right now, whether or not anyone saw the failure that caused it.

### Check

The workflow output lists order id, SKU, product, units and age. To run it directly — after a suspected incident, rather than waiting for Sunday:

```
gh workflow run inventory-reconciliation.yml -f min_age=24h
```

Or locally against one environment:

```
cd handloom-admin
DEV_DSN="$POSTGRES_DSN" go run ./scripts/reconcile-inventory --min-age 24h
```

`min_age` defaults to 24h; anything younger is usually a customer mid-payment, not drift. The report caps at 500 orders and warns when it hits the cap, which means the problem is systemic rather than a handful of stuck orders.

### Diagnose before fixing

```sql
SELECT type, quantity, previous_qty, new_qty, created_at
FROM inventory_transactions
WHERE reference_id = 'order_dd90' AND reference_type = 'ORDER'
ORDER BY created_at, id;
```

Then check the order's status. Three cases:

| Order status | Meaning | Action |
|---|---|---|
| `PENDING` | Checkout never completed; no reaper exists yet | Release the reservation |
| `CANCELLED` | The release failed at the time | Release the reservation |
| `SHIPPED`/`DELIVERED` | The dispatch failed, so stock was never decremented | **Do not release** — commit it, or the units come back twice |

That last row is the one to be careful about: releasing a reservation whose order actually shipped returns stock that physically left the warehouse.

### Fix

Every order-scoped movement is idempotent per `(product, order, type)` and writes a ledger row, so a fix is safe to retry and leaves a trace. Cancel an order that should be cancelled:

```
POST /admin/orders/{order_id}/cancel
```

There is deliberately no bulk "release all orphans" action: each case needs the status check above, and a blanket release would silently corrupt the `SHIPPED` case. Do not edit `reserved_qty` by hand — it leaves no ledger row, so the next reconciliation cannot explain the number.

### If the number keeps growing

Recurring drift is not something to keep clearing by hand. Usual causes:

- No reaper for abandoned `PENDING` checkouts — the known permanent-leak path
- A failing release on the payment-failure rollback, which also raises `inventory_mutation_failed` with reason `release`
- Transient Postgres failures at dispatch. `CommitOrderStock` retries once and the pool bounds `ConnectTimeout`/`MaxConnLifetime`/`HealthCheckPeriod`, so a rising count here suggests something longer than a cold start

## Quota watch (Grafana Cloud free tier — traces + logs only)

Metrics no longer count against Grafana series (they're in Postgres). Watch logs/traces:

| Signal | Limit | Action at 80% |
|---|---|---|
| Logs | 50 GB / 14d | Raise storefront SSR log level to WARN; verify no log loops |
| Traces | 50 GB / 14d | Add tail sampler in the collector — keep all errors, sample 10% of OK traces |
| PG `metric_counters` | retention cron (90d) | Confirm the retention job is scheduled (migration 008 manual step); audit cardinality if row growth spikes |
