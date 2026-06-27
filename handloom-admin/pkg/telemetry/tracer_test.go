package telemetry

import "testing"

func TestEndpointIsURL(t *testing.T) {
	for in, want := range map[string]bool{
		"https://otlp-gateway.grafana.net/otlp": true,
		"http://localhost:4318":                 true,
		"localhost:4317":                        false,
		"collector.internal:4318":               false,
	} {
		if got := endpointIsURL(in); got != want {
			t.Errorf("endpointIsURL(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestOTLPTracesURL(t *testing.T) {
	for in, want := range map[string]string{
		// base endpoint (what the collector-based lambdas share) → traces signal path
		"https://otlp-gateway.grafana.net/otlp":  "https://otlp-gateway.grafana.net/otlp/v1/traces",
		"https://otlp-gateway.grafana.net/otlp/": "https://otlp-gateway.grafana.net/otlp/v1/traces",
		// already a traces URL → unchanged (idempotent)
		"https://otlp-gateway.grafana.net/otlp/v1/traces": "https://otlp-gateway.grafana.net/otlp/v1/traces",
		"http://localhost:4318":                           "http://localhost:4318/v1/traces",
	} {
		if got := otlpTracesURL(in); got != want {
			t.Errorf("otlpTracesURL(%q) = %q, want %q", in, got, want)
		}
	}
}
