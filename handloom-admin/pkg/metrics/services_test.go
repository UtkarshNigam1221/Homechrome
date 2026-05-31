package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKnownService(t *testing.T) {
	known := []string{
		"handloom-auth-dev",
		"handloom-auth-prod",
		"handloom-store-events-dev",
		"handloom-store-events-prod",
		"handloom-metrics-consumer-dev",
		"handloom-metrics-consumer-prod",
		"handloom-worker-audit-prod",
		"handloom-embedder-dev",
	}
	for _, name := range known {
		assert.True(t, IsKnownService(name), "expected known: %s", name)
	}

	unknown := []string{"", "unknown", "handloom-auth", "handloom-auth-staging"}
	for _, name := range unknown {
		assert.False(t, IsKnownService(name), "expected unknown: %s", name)
	}
}
