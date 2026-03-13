package telemetry

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware returns an HTTP middleware that instruments requests with OpenTelemetry.
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use otelhttp for automatic HTTP instrumentation
		handler := otelhttp.NewHandler(next, serviceName,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
			otelhttp.WithSpanOptions(
				trace.WithAttributes(
					attribute.String("service.name", serviceName),
				),
			),
		)
		return handler
	}
}

// HTTPMiddlewareWithOptions returns an HTTP middleware with custom options.
func HTTPMiddlewareWithOptions(serviceName string, opts ...otelhttp.Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		defaultOpts := make([]otelhttp.Option, 0, 2+len(opts))
		defaultOpts = append(defaultOpts,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		)
		allOpts := append(defaultOpts, opts...)
		return otelhttp.NewHandler(next, serviceName, allOpts...)
	}
}

// TraceIDMiddleware extracts or generates trace ID and adds it to response headers.
// This is useful for debugging and correlating logs with traces.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get trace ID from span context (set by otelhttp middleware)
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

		if span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()
			spanID := span.SpanContext().SpanID().String()

			// Add trace ID to response headers for debugging
			w.Header().Set("X-Trace-ID", traceID)
			w.Header().Set("X-Span-ID", spanID)
		}

		next.ServeHTTP(w, r)
	})
}

// RouteTagMiddleware adds route pattern as span attribute.
// This is useful for grouping spans by route instead of individual URLs.
func RouteTagMiddleware(routePattern string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			span.SetAttributes(
				attribute.String("http.route", routePattern),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDMiddleware adds user ID as span attribute when available in context.
func UserIDMiddleware(getUserID func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID := getUserID(r); userID != "" {
				span := trace.SpanFromContext(r.Context())
				span.SetAttributes(
					attribute.String("user.id", userID),
				)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MetricsAttributes returns common HTTP metrics attributes from a request.
func MetricsAttributes(r *http.Request) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", r.Method),
		attribute.String("http.scheme", getScheme(r)),
		attribute.String("http.host", r.Host),
		attribute.String("http.target", r.URL.Path),
		attribute.String("http.user_agent", r.UserAgent()),
	}
}

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// TracingResponseWriter wraps http.ResponseWriter to capture status code.
type TracingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Written    int64
}

// NewTracingResponseWriter creates a new TracingResponseWriter.
func NewTracingResponseWriter(w http.ResponseWriter) *TracingResponseWriter {
	return &TracingResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code.
func (rw *TracingResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written.
func (rw *TracingResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.Written += int64(n)
	return n, err
}

// Flush implements http.Flusher.
func (rw *TracingResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker.
func (rw *TracingResponseWriter) Hijack() (interface{}, interface{}, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}
