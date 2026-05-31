package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRedactSpanAttr_ScrubsPII(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(rec),
	)
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	for _, a := range []attribute.KeyValue{
		attribute.String("password", "hunter2"),
		attribute.String("email", "a@b.com"),
		attribute.String("phone", "+919876543210"),
		attribute.String("user.id", "u_123"),
	} {
		span.SetAttributes(redactSpanAttr(a))
	}
	span.End()

	got := rec.Ended()[0].Attributes()
	attrs := map[string]string{}
	for _, a := range got {
		attrs[string(a.Key)] = a.Value.AsString()
	}
	if attrs["password"] != "[REDACTED]" {
		t.Errorf("password not redacted: %v", attrs)
	}
	if attrs["email"] != "***@b.com" {
		t.Errorf("email not redacted: %v", attrs)
	}
	if attrs["phone"] != "***3210" {
		t.Errorf("phone not redacted: %v", attrs)
	}
	if attrs["user.id"] != "u_123" {
		t.Errorf("user.id should pass: %v", attrs)
	}
}
