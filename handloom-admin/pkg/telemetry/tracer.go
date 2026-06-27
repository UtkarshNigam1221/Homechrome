package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider wraps the OpenTelemetry TracerProvider with additional utilities.
// loggerProvider is optional — set by MustInitDirect when OTLP logs export is
// enabled — so ForceFlush/Shutdown drain both signals.
type TracerProvider struct {
	provider       *sdktrace.TracerProvider
	loggerProvider *sdklog.LoggerProvider
	config         *Config
}

// newResource builds the OTel resource shared by the tracer + logger providers.
// Avoid resource.Merge(resource.Default(), ...) — the default resource uses the
// SDK's latest schema URL which may drift ahead of our semconv import and cause
// Merge to error on schema mismatch. Build it explicitly with our pinned schema.
func newResource(cfg *Config) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", cfg.ServiceVersion),
		attribute.String("service.namespace", "handloom"),
		attribute.String("deployment.environment.name", cfg.Environment),
	)
}

// NewTracerProvider creates and configures a new TracerProvider.
func NewTracerProvider(ctx context.Context, cfg *Config) (*TracerProvider, error) {
	if !cfg.Tracing.Enabled {
		// Return a no-op provider
		return &TracerProvider{
			provider: nil,
			config:   cfg,
		}, nil
	}

	// Create the exporter
	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res := newResource(cfg)

	// Create the sampler based on sample rate
	var sampler sdktrace.Sampler
	if cfg.Tracing.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.Tracing.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.Tracing.SampleRate)
	}

	// Pick span processor based on runtime. In Lambda, BatchSpanProcessor's
	// 5s timeout often fires AFTER the runtime freezes the process between
	// invocations — buffered spans are lost. Use SimpleSpanProcessor (synchronous
	// export on Span.End()) in Lambda. Outside Lambda (monolith) batching is fine.
	var spanProcessor sdktrace.TracerProviderOption
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		spanProcessor = sdktrace.WithSyncer(exporter)
	} else {
		spanProcessor = sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(cfg.Tracing.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(cfg.Tracing.MaxExportBatchSize),
		)
	}

	tp := sdktrace.NewTracerProvider(
		spanProcessor,
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	// Set as global TracerProvider
	otel.SetTracerProvider(tp)

	// Set up propagators for distributed tracing
	// Support both W3C Trace Context and B3 (used by Zipkin, some service meshes)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	))

	return &TracerProvider{
		provider: tp,
		config:   cfg,
	}, nil
}

// createExporter creates the appropriate exporter based on configuration.
func createExporter(ctx context.Context, cfg *Config) (sdktrace.SpanExporter, error) {
	switch cfg.Tracing.Exporter {
	case "otlp-grpc":
		return createOTLPGRPCExporter(ctx, cfg)
	case "otlp-http":
		return createOTLPHTTPExporter(ctx, cfg)
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "none", "":
		// Return a no-op exporter
		return &noopExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown exporter type: %s", cfg.Tracing.Exporter)
	}
}

// createOTLPGRPCExporter creates an OTLP gRPC exporter.
func createOTLPGRPCExporter(ctx context.Context, cfg *Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Tracing.Endpoint),
	}

	if cfg.Tracing.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if len(cfg.Tracing.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Tracing.Headers))
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// endpointIsURL reports whether an OTLP endpoint is a full URL (scheme://host/path)
// rather than a bare host[:port].
func endpointIsURL(endpoint string) bool {
	return strings.Contains(endpoint, "://")
}

// otlpSignalURL ensures a full OTLP/HTTP endpoint URL targets a specific signal
// path (/v1/<signal>), the spec path WithEndpointURL would otherwise leave as-is.
// A base like https://otlp-gateway.../otlp + "traces" → .../otlp/v1/traces. Idempotent.
func otlpSignalURL(raw, signal string) string {
	trimmed := strings.TrimRight(raw, "/")
	suffix := "/v1/" + signal
	if strings.HasSuffix(trimmed, suffix) {
		return trimmed
	}
	return trimmed + suffix
}

// otlpTracesURL is otlpSignalURL for the traces signal.
func otlpTracesURL(raw string) string { return otlpSignalURL(raw, "traces") }

// createOTLPHTTPExporter creates an OTLP HTTP exporter.
func createOTLPHTTPExporter(ctx context.Context, cfg *Config) (sdktrace.SpanExporter, error) {
	var opts []otlptracehttp.Option
	// A full URL (e.g. Grafana Cloud's https://otlp-gateway-.../otlp) carries its
	// own scheme + base path, so use WithEndpointURL. Unlike WithEndpoint (bare
	// host[:port], which auto-targets /v1/traces), WithEndpointURL uses the path
	// VERBATIM — so normalize it to the OTLP/HTTP traces signal path or a base
	// endpoint silently 404s and drops every span.
	if endpointIsURL(cfg.Tracing.Endpoint) {
		opts = append(opts, otlptracehttp.WithEndpointURL(otlpTracesURL(cfg.Tracing.Endpoint)))
	} else {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.Tracing.Endpoint))
	}

	if cfg.Tracing.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(cfg.Tracing.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Tracing.Headers))
	}

	client := otlptracehttp.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// Shutdown gracefully shuts down the tracer + logger providers.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		if err := tp.provider.Shutdown(ctx); err != nil {
			return err
		}
	}
	if tp.loggerProvider != nil {
		return tp.loggerProvider.Shutdown(ctx)
	}
	return nil
}

// ForceFlush exports any buffered spans + logs without tearing down the
// providers. Call it per-invocation in Lambda so in-flight telemetry drains
// before the runtime freezes (Shutdown would kill the providers after one request).
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	if tp.provider != nil {
		if err := tp.provider.ForceFlush(ctx); err != nil {
			return err
		}
	}
	if tp.loggerProvider != nil {
		return tp.loggerProvider.ForceFlush(ctx)
	}
	return nil
}

// Tracer returns a named tracer.
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	if tp.provider != nil {
		return tp.provider.Tracer(name)
	}
	return otel.Tracer(name)
}

// noopExporter is a no-op span exporter.
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

// ============================================================================
// Helper functions for creating spans and adding attributes
// ============================================================================

// StartSpan starts a new span with the given name.
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError marks the span as errored with the given error.
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanOK marks the span as successful.
func SetSpanOK(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Ok, "")
}

// SetSpanAttributes adds attributes to the current span.
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// GetTraceID returns the trace ID from the current span.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the current span.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsTraceIDValid checks if the context has a valid trace ID.
func IsTraceIDValid(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().HasTraceID() && span.SpanContext().IsSampled()
}
