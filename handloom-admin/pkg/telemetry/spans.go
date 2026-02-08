package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
)

// Span kind constants for better readability
const (
	SpanKindInternal = trace.SpanKindInternal
	SpanKindServer   = trace.SpanKindServer
	SpanKindClient   = trace.SpanKindClient
)

// ============================================================================
// Database Span Helpers (DynamoDB)
// ============================================================================

// DBSpan represents a database operation span.
type DBSpan struct {
	ctx    context.Context
	span   trace.Span
	tracer trace.Tracer
}

// StartDBSpan starts a new database span.
func StartDBSpan(ctx context.Context, operation, table string) (context.Context, *DBSpan) {
	tracer := otel.Tracer("handloom.dynamodb")
	spanName := fmt.Sprintf("DynamoDB %s", operation)

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemNameAWSDynamoDB,
			semconv.DBOperationNameKey.String(operation),
			attribute.String("db.dynamodb.table", table),
		),
	)

	return ctx, &DBSpan{
		ctx:    ctx,
		span:   span,
		tracer: tracer,
	}
}

// SetQueryParams sets DynamoDB query parameters as span attributes.
func (s *DBSpan) SetQueryParams(params map[string]interface{}) {
	for k, v := range params {
		s.span.SetAttributes(attribute.String(fmt.Sprintf("db.dynamodb.%s", k), fmt.Sprintf("%v", v)))
	}
}

// SetItemCount sets the number of items returned/affected.
func (s *DBSpan) SetItemCount(count int) {
	s.span.SetAttributes(attribute.Int("db.dynamodb.item_count", count))
}

// SetConsumedCapacity sets the consumed capacity units.
func (s *DBSpan) SetConsumedCapacity(readUnits, writeUnits float64) {
	s.span.SetAttributes(
		attribute.Float64("db.dynamodb.consumed_read_units", readUnits),
		attribute.Float64("db.dynamodb.consumed_write_units", writeUnits),
	)
}

// End ends the database span.
func (s *DBSpan) End() {
	s.span.End()
}

// EndWithError ends the span with an error.
func (s *DBSpan) EndWithError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	s.span.End()
}

// Context returns the span's context.
func (s *DBSpan) Context() context.Context {
	return s.ctx
}

// ============================================================================
// Service Span Helpers
// ============================================================================

// ServiceSpan represents a service operation span.
type ServiceSpan struct {
	ctx    context.Context
	span   trace.Span
	tracer trace.Tracer
}

// StartServiceSpan starts a new service layer span.
func StartServiceSpan(ctx context.Context, serviceName, operation string) (context.Context, *ServiceSpan) {
	tracer := otel.Tracer(fmt.Sprintf("handloom.service.%s", serviceName))
	spanName := fmt.Sprintf("%s.%s", serviceName, operation)

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(SpanKindInternal),
		trace.WithAttributes(
			attribute.String("service.layer", "service"),
			attribute.String("service.name", serviceName),
			attribute.String("service.operation", operation),
		),
	)

	return ctx, &ServiceSpan{
		ctx:    ctx,
		span:   span,
		tracer: tracer,
	}
}

// SetAttribute sets a custom attribute on the span.
func (s *ServiceSpan) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SetAttributes sets multiple attributes on the span.
func (s *ServiceSpan) SetAttributes(attrs map[string]interface{}) {
	for k, v := range attrs {
		s.SetAttribute(k, v)
	}
}

// AddEvent adds an event to the span.
func (s *ServiceSpan) AddEvent(name string, attrs ...attribute.KeyValue) {
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}

// End ends the service span.
func (s *ServiceSpan) End() {
	s.span.SetStatus(codes.Ok, "")
	s.span.End()
}

// EndWithError ends the span with an error.
func (s *ServiceSpan) EndWithError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	s.span.End()
}

// Context returns the span's context.
func (s *ServiceSpan) Context() context.Context {
	return s.ctx
}

// ============================================================================
// Handler Span Helpers
// ============================================================================

// HandlerSpan represents a handler operation span.
type HandlerSpan struct {
	ctx    context.Context
	span   trace.Span
	tracer trace.Tracer
}

// StartHandlerSpan starts a new handler layer span.
func StartHandlerSpan(ctx context.Context, handlerName, operation string) (context.Context, *HandlerSpan) {
	tracer := otel.Tracer(fmt.Sprintf("handloom.handler.%s", handlerName))
	spanName := fmt.Sprintf("%s.%s", handlerName, operation)

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(SpanKindInternal),
		trace.WithAttributes(
			attribute.String("service.layer", "handler"),
			attribute.String("handler.name", handlerName),
			attribute.String("handler.operation", operation),
		),
	)

	return ctx, &HandlerSpan{
		ctx:    ctx,
		span:   span,
		tracer: tracer,
	}
}

// SetUserID sets the user ID attribute.
func (s *HandlerSpan) SetUserID(userID string) {
	s.span.SetAttributes(attribute.String("user.id", userID))
}

// SetRequestID sets the request ID attribute.
func (s *HandlerSpan) SetRequestID(requestID string) {
	s.span.SetAttributes(attribute.String("request.id", requestID))
}

// End ends the handler span.
func (s *HandlerSpan) End() {
	s.span.SetStatus(codes.Ok, "")
	s.span.End()
}

// EndWithError ends the span with an error.
func (s *HandlerSpan) EndWithError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
	s.span.End()
}

// Context returns the span's context.
func (s *HandlerSpan) Context() context.Context {
	return s.ctx
}

// ============================================================================
// Generic Span Helpers
// ============================================================================

// WithSpan executes a function within a span and handles errors automatically.
func WithSpan[T any](ctx context.Context, tracerName, spanName string, fn func(context.Context) (T, error)) (T, error) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

// WithSpanVoid executes a function within a span that doesn't return a value.
func WithSpanVoid(ctx context.Context, tracerName, spanName string, fn func(context.Context) error) error {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// ============================================================================
// Common Attributes
// ============================================================================

// CommonAttributes contains commonly used attribute keys.
var CommonAttributes = struct {
	UserID      string
	RequestID   string
	EntityID    string
	EntityType  string
	Operation   string
	Count       string
	Duration    string
	ErrorCode   string
	ErrorMsg    string
	Environment string
}{
	UserID:      "user.id",
	RequestID:   "request.id",
	EntityID:    "entity.id",
	EntityType:  "entity.type",
	Operation:   "operation",
	Count:       "count",
	Duration:    "duration_ms",
	ErrorCode:   "error.code",
	ErrorMsg:    "error.message",
	Environment: "deployment.environment",
}
