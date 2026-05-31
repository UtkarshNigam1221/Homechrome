# Observability Runbook

## Triage flow — customer reports an error

1. Get the **Reference ID** from the user (it appears on the error page as `Reference: <trace_id>`).
2. In Grafana → Explore → Tempo, search by Trace ID.
3. Open the trace. The root span shows route, status, duration. Drill into red spans to find the failing layer (handler, service, DB, gateway).
4. Click "Logs for this span" — Grafana auto-links to Loki filtered by `trace_id`. You see every log line the request emitted.
5. If a gateway span is red, check **Gateway Health** dashboard for the same time window — was the gateway degraded for everyone, or just this user?

## Acceptance test for the observability foundation

Run after every deploy until ticked off.

- [ ] Issue `curl -i https://dev-api.homechrome.in/health`. Response includes `X-Request-ID` and `X-Trace-ID` headers.
- [ ] Within 30s, the same trace ID is visible in Grafana → Explore → Tempo.
- [ ] Click "Logs for this trace" — Loki shows ≥1 log line with the same `trace_id`.
- [ ] Place a test order in the storefront. Trace tree shows: `browser → nextjs SSR (server) → handloom-store-checkout → handloom-order → ddb.PutItem`.
- [ ] Grafana → Explore → Mimir → metric `business.orders.placed` shows a 1-bucket increment.
- [ ] CW Log Group `/aws/lambda/handloom-store-checkout-dev` has `RetentionInDays=1`.
- [ ] Force a 500 in dev (or use a debug endpoint that panics). Trace has red status, Loki shows ERROR-level log with `trace_id`, dashboard Overview shows error-rate blip.
- [ ] Search Loki for "password" or "Bearer " across last hour — zero results (PII redaction working).

## Quota watch (Grafana Cloud free tier)

Visit Grafana → Billing daily for the first 4 weeks after rollout.

| Signal | Limit | Action if 80% hit |
|---|---|---|
| Logs | 50 GB / 14d | Tighten DEBUG ring buffer; raise log level for storefront SSR to WARN; verify no log loops |
| Traces | 50 GB / 14d | Add tail sampler in collector — keep all errors, sample 10% of OK traces |
| Metrics | 10k series | Audit cardinality dashboard; the AST test in `pkg/telemetry/cardinality_test.go` catches new offenders at CI time |

## Operating notes

- **Trace correlation:** Every log line carries `trace_id` and `span_id` (injected automatically by `pkg/slogx`). Service code does NOT need to thread these through manually.
- **PII redaction is two-layer:** in-process slog handler (`pkg/slogx/redact.go`) scrubs known PII keys; collector-side `attributes/redact` processor (`handloom-admin/infra/configs/otel-collector.yaml`) scrubs span/log attribute keys before export. Adding a new PII field name? Update both denylists.
- **Emergency kill switch:** set `OTEL_SDK_DISABLED=true` on a Lambda to disable all OTel SDK init in that function. Falls back to slog → stdout → CloudWatch only. Use during incidents where telemetry export is suspected to be making things worse.
- **Cardinality budget:** `metric.WithAttributes` may NOT use `user_id`, `order_id`, `cart_id`, `email`, `phone`, `sku`, etc. The CI test in `pkg/telemetry/cardinality_test.go` enforces this. If a legitimate case arises, add to the `CardinalityAllowlist` map with a justification comment.

## Common Loki queries

```
{service="handloom-store-checkout"} |= "ERROR"
{service="handloom-store-checkout"} | json | trace_id="<id>"
{service=~"handloom-.+"} |= "payment" | json | level="ERROR"
{service="handloom-monolith"} | json | status >= 500
```

## Common Tempo queries

- By trace ID: paste the ID into the search bar.
- By service + status: `{ resource.service.name = "handloom-store-checkout" && status = error }`
- By duration: `{ duration > 2s && resource.service.name =~ "handloom-.+" }`

## Common Mimir / PromQL queries

```
sum(rate(http_server_requests_total[5m])) by (service_name)
sum(rate(http_server_errors_total[5m])) by (service_name, http_route)
histogram_quantile(0.95, sum(rate(http_server_request_duration_bucket[5m])) by (le, http_route))
sum(rate(business_orders_placed_total[1h])) by (channel)
sum(rate(gateway_call_outcome_total{status_class!="2xx"}[5m])) by (gateway, operation)
```

(Adjust metric names to match the actual emission once dashboards are built — OTel exporters may apply suffixes like `_total` for counters and `_bucket`/`_sum`/`_count` for histograms.)
