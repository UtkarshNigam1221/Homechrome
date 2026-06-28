import { registerOTel } from "@vercel/otel";

// Manual OpenTelemetry for the Next.js server — mirrors how the Go lambdas
// instrument in-app (pkg/telemetry). We deliberately do NOT use the ADOT Node
// auto-instrumentation layer: it wraps the runtime and double-manages Next's
// own spans, throwing "Operation attempted on ended Span" during SSR/revalidate.
//
// @vercel/otel builds its OTLP exporter from the OTEL_EXPORTER_OTLP_ENDPOINT /
// _PROTOCOL env vars set by the CDK stack (→ the in-Lambda OTel Collector on
// localhost:4318, which ships to Grafana). Next auto-runs this register() hook.
export function register() {
  registerOTel({ serviceName: process.env.OTEL_SERVICE_NAME ?? "homechrome-store" });
}
