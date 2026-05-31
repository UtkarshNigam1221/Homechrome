package store

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/response"
)

// EventsHandler handles frontend tracking event ingestion for the storefront.
type EventsHandler struct {
	validation *middleware.Validation
	centroids  *postgres.CentroidsRepository
}

// NewEventsHandler creates a new EventsHandler.
func NewEventsHandler(validation *middleware.Validation, centroids *postgres.CentroidsRepository) *EventsHandler {
	return &EventsHandler{validation: validation, centroids: centroids}
}

// Routes returns the events routes.
// This endpoint is PUBLIC (no auth middleware) — tracking works for anonymous visitors.
func (h *EventsHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.StoreEventBatch](h.validation)).
		Post("/", h.IngestEvents)

	return r
}

// IngestEvents handles POST / — accepts a batch of frontend tracking events.
// Events are routed to PG metric_counters via routeStoreEventToMetric.
// Raw events are no longer persisted (DDB events table dropped in M39).
func (h *EventsHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	batch := middleware.MustGetValidatedBody[domain.StoreEventBatch](ctx)

	// Filter out events older than 24 hours and with unknown event types.
	cutoff := time.Now().Add(-24 * time.Hour)
	valid := make([]domain.StoreEvent, 0, len(batch.Events))
	for _, evt := range batch.Events {
		if evt.Timestamp.After(cutoff) && domain.IsValidStoreEventType(evt.EventType) {
			valid = append(valid, evt)
		}
	}

	if len(valid) == 0 {
		response.JSON(w, http.StatusAccepted, map[string]int{"accepted": 0})
		return
	}

	// Route each event to a PG metric (counters + RUM histograms).
	for _, evt := range valid {
		h.routeStoreEventToMetric(ctx, evt)
	}

	response.JSON(w, http.StatusAccepted, map[string]int{"accepted": len(valid)})
}

// routeStoreEventToMetric maps a beacon event to the matching PG metric.
// On rum_page_view it also lazy-upserts the (city, country, lat, lng)
// tuple into city_centroids so the geomap on the admin dashboard always
// has a marker placement without a hand-curated allowlist.
func (h *EventsHandler) routeStoreEventToMetric(ctx context.Context, evt domain.StoreEvent) {
	deviceType := normaliseDevice(evt.DeviceType)

	switch evt.EventType {
	case "product_viewed":
		productID, _ := evt.Properties["product_id"].(string)
		categoryID, _ := evt.Properties["category_id"].(string)
		if categoryID == "" {
			categoryID = "unknown"
		}
		if productID != "" {
			metrics.Record(ctx, "product_viewed", metrics.L{
				"product_id":  productID,
				"category_id": categoryID,
				"device_type": deviceType,
			})
		}

	case "category_viewed":
		categoryID, _ := evt.Properties["category_id"].(string)
		if categoryID == "" {
			categoryID = "unknown"
		}
		metrics.Record(ctx, "category_viewed", metrics.L{
			"category_id": categoryID,
		})

	case "out_of_stock_shown":
		productID, _ := evt.Properties["product_id"].(string)
		if productID == "" {
			productID = "unknown"
		}
		metrics.Record(ctx, "out_of_stock_shown", metrics.L{
			"product_id": productID,
		})

	case "back_in_stock_notify_requested":
		productID, _ := evt.Properties["product_id"].(string)
		if productID == "" {
			productID = "unknown"
		}
		metrics.Record(ctx, "back_in_stock_notify_requested", metrics.L{
			"product_id": productID,
		})

	case "catalog_filter_applied":
		filterKey, _ := evt.Properties["filter_key"].(string)
		if filterKey == "" {
			filterKey = "unknown"
		}
		metrics.Record(ctx, "catalog_filter_applied", metrics.L{
			"filter_key": truncate(filterKey, 32),
		})

	case "rum_lcp":
		emitRUM(ctx, "rum_lcp", evt, deviceType)
	case "rum_inp":
		emitRUM(ctx, "rum_inp", evt, deviceType)
	case "rum_cls":
		emitRUM(ctx, "rum_cls", evt, deviceType)
	case "rum_ttfb":
		emitRUM(ctx, "rum_ttfb", evt, deviceType)

	case "rum_js_error":
		pageType, _ := evt.Properties["page_type"].(string)
		errorType, _ := evt.Properties["error_type"].(string)
		if pageType == "" {
			pageType = "other"
		}
		if errorType == "" {
			errorType = "Error"
		}
		metrics.Record(ctx, "rum_js_error", metrics.L{
			"page_type":  pageType,
			"error_type": truncate(errorType, 64),
		})

	case "rum_page_view":
		pageType, _ := evt.Properties["page_type"].(string)
		if pageType == "" {
			pageType = "other"
		}
		metrics.Record(ctx, "rum_page_view", metrics.L{
			"page_type":   pageType,
			"device_type": deviceType,
		})
		// site_visitor is the top-of-funnel signal — every page load, anonymous-
		// friendly. Carries the full visitor-attribution tuple so the funnel
		// dashboard can break down by device + acquisition source.
		city := middleware.GetCity(ctx)
		country := middleware.GetCountry(ctx)
		utmSource, utmMedium, utmCampaign := middleware.GetUTM(ctx)
		metrics.Record(ctx, "site_visitor", metrics.L{
			"city":         city,
			"country":      country,
			"device_type":  deviceType,
			"utm_source":   utmSource,
			"utm_medium":   utmMedium,
			"utm_campaign": utmCampaign,
		})
		// Lazy-upsert centroid on first sighting (per (city, country) pair).
		// Skipped when city/country unknown or lat/lng absent — keeps the
		// table free of nonsense rows. ON CONFLICT DO NOTHING means repeat
		// emits cost one round-trip and one no-op.
		if lat, lng, ok := middleware.GetLatLng(ctx); ok && city != "unknown" && country != "unknown" {
			if err := h.centroids.Upsert(ctx, city, country, lat, lng); err != nil {
				slog.WarnContext(ctx, "centroid upsert failed",
					"city", city, "country", country, "err", err)
			}
		}
	}
}

// emitRUM resolves the metric value, buckets it, and emits a count-1 PG event
// with bucket + page_type + device_type labels.
func emitRUM(ctx context.Context, metricName string, evt domain.StoreEvent, deviceType string) {
	pageType, _ := evt.Properties["page_type"].(string)
	if pageType == "" {
		pageType = "other"
	}
	v := rumValue(evt.Properties["value"])
	metrics.Record(ctx, metricName, metrics.L{
		"bucket":      bucketForRUM(metricName, v),
		"page_type":   pageType,
		"device_type": deviceType,
	})
}

// bucketForRUM returns the Web-Vitals "good / needs-improvement / poor"
// bucket label for the named metric.
func bucketForRUM(name string, value float64) string {
	switch name {
	case "rum_lcp":
		switch {
		case value <= 2500:
			return "good"
		case value <= 4000:
			return "needs_improvement"
		default:
			return "poor"
		}
	case "rum_inp":
		switch {
		case value <= 200:
			return "good"
		case value <= 500:
			return "needs_improvement"
		default:
			return "poor"
		}
	case "rum_cls":
		// CLS shipped from client as value*100 (see rum.ts).
		switch {
		case value <= 10:
			return "good"
		case value <= 25:
			return "needs_improvement"
		default:
			return "poor"
		}
	case "rum_ttfb":
		switch {
		case value <= 800:
			return "good"
		case value <= 1800:
			return "needs_improvement"
		default:
			return "poor"
		}
	}
	return "unknown"
}

// rumValue coerces the beacon-shipped value (number-as-JSON or string) into a float.
func rumValue(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	}
	return 0
}

// normaliseDevice maps any client-supplied device_type to a bounded label set.
func normaliseDevice(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "mobile", "tablet", "desktop":
		return strings.ToLower(d)
	default:
		return "unknown"
	}
}

// truncate clips s to at most n bytes — defensive bound on metric labels.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
