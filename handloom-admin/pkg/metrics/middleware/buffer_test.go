package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/handloom/admin/pkg/metrics"
	mw "github.com/handloom/admin/pkg/metrics/middleware"
)

type recordingPublisher struct {
	events []metrics.Event
}

func (r *recordingPublisher) Publish(_ context.Context, events []metrics.Event) error {
	r.events = append(r.events, events...)
	return nil
}

func TestBuffer_capturesEmits(t *testing.T) {
	rec := &recordingPublisher{}
	metrics.SetDefault(rec)
	defer metrics.SetDefault(metrics.NoopPublisher{})

	handler := mw.Buffer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.Record(r.Context(), "test", metrics.L{"a": "b"})
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Len(t, rec.events, 1)
	assert.Equal(t, "test", rec.events[0].Metric)
}

func TestBuffer_noEmits_noPublish(t *testing.T) {
	rec := &recordingPublisher{}
	metrics.SetDefault(rec)
	defer metrics.SetDefault(metrics.NoopPublisher{})

	handler := mw.Buffer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Len(t, rec.events, 0)
}
