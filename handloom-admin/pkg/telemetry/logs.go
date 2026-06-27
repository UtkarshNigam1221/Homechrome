package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// newLoggerProvider builds an OTLP/HTTP log exporter → LoggerProvider that ships
// directly to the configured endpoint's /v1/logs path with the same auth headers
// as the tracer. Used by the container embedder Lambda, which can't use the OTel
// Collector layer's telemetryAPI log tap, so the app must export its own logs.
func newLoggerProvider(ctx context.Context, cfg *Config) (*sdklog.LoggerProvider, error) {
	var opts []otlploghttp.Option
	if endpointIsURL(cfg.Tracing.Endpoint) {
		opts = append(opts, otlploghttp.WithEndpointURL(otlpSignalURL(cfg.Tracing.Endpoint, "logs")))
	} else {
		opts = append(opts, otlploghttp.WithEndpoint(cfg.Tracing.Endpoint))
	}
	if cfg.Tracing.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if len(cfg.Tracing.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.Tracing.Headers))
	}
	exp, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// Synchronous export in Lambda (the runtime freezes between invocations, so a
	// batch timer can't reliably fire); batch outside Lambda.
	var proc sdklog.Processor
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		proc = sdklog.NewSimpleProcessor(exp)
	} else {
		proc = sdklog.NewBatchProcessor(exp)
	}
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(proc),
		sdklog.WithResource(newResource(cfg)),
	), nil
}

// installSlogFanout points the default slog logger at BOTH stdout (→ CloudWatch)
// and the OTel logger provider (→ Grafana Loki), so CloudWatch stays a fallback
// while logs also reach Grafana.
func installSlogFanout(lp *sdklog.LoggerProvider, serviceName string) {
	otelHandler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp))
	stdoutHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(fanoutHandler{handlers: []slog.Handler{stdoutHandler, otelHandler}}))
}

// fanoutHandler dispatches each slog record to every wrapped handler.
type fanoutHandler struct{ handlers []slog.Handler }

func (h fanoutHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, lvl) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, r.Level) {
			_ = hh.Handle(ctx, r.Clone()) // best-effort: one sink failing must not drop the others
		}
	}
	return nil
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		next[i] = hh.WithAttrs(attrs)
	}
	return fanoutHandler{handlers: next}
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		next[i] = hh.WithGroup(name)
	}
	return fanoutHandler{handlers: next}
}
