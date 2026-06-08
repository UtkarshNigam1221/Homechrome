package service

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/geo"
	"github.com/handloom/admin/pkg/metrics"
)

// centroidUpserter lazily records a (city,country)->(lat,lng) centroid on first
// sighting. Implemented by the postgres CentroidsRepository.
type centroidUpserter interface {
	Upsert(ctx context.Context, city, country string, lat, lng float64) error
}

// StoreEventService maps validated storefront beacon events to PG metrics
// (counters + RUM histograms) and lazy-seeds city centroids. It owns the
// event->metric domain logic that previously lived in the HTTP handler.
type StoreEventService struct {
	centroids centroidUpserter
}

// NewStoreEventService creates a StoreEventService.
func NewStoreEventService(centroids centroidUpserter) *StoreEventService {
	return &StoreEventService{centroids: centroids}
}

// Record routes one event to its metric mapper. Unknown event types are
// ignored (the handler already filters them; this is defensive).
func (s *StoreEventService) Record(ctx context.Context, evt domain.StoreEvent) {
	device := geo.NormalizeDevice(evt.DeviceType)
	if m, ok := storeEventMappers[evt.EventType]; ok {
		m(s, ctx, evt, device)
	}
}

// storeEventMappers is the event-type -> mapper registry. Add a new event by
// adding one entry here — there is no switch to extend.
var storeEventMappers = map[string]func(*StoreEventService, context.Context, domain.StoreEvent, string){
	"product_viewed":                 (*StoreEventService).mapProductViewed,
	"category_viewed":                (*StoreEventService).mapCategoryViewed,
	"out_of_stock_shown":             (*StoreEventService).mapOutOfStockShown,
	"back_in_stock_notify_requested": (*StoreEventService).mapBackInStockNotify,
	"catalog_filter_applied":         (*StoreEventService).mapCatalogFilterApplied,
	"rum_lcp":                        rumMapper("rum_lcp"),
	"rum_inp":                        rumMapper("rum_inp"),
	"rum_cls":                        rumMapper("rum_cls"),
	"rum_ttfb":                       rumMapper("rum_ttfb"),
	"rum_js_error":                   (*StoreEventService).mapRUMJSError,
	"rum_page_view":                  (*StoreEventService).mapRUMPageView,
}

func (s *StoreEventService) mapProductViewed(ctx context.Context, evt domain.StoreEvent, device string) {
	productID, _ := evt.Properties["product_id"].(string)
	categoryID, _ := evt.Properties["category_id"].(string)
	if categoryID == "" {
		categoryID = labelUnknown
	}
	if productID != "" {
		metrics.Record(ctx, "product_viewed", metrics.L{
			"product_id":  productID,
			"category_id": categoryID,
			"device_type": device,
		})
	}
}

func (s *StoreEventService) mapCategoryViewed(ctx context.Context, evt domain.StoreEvent, _ string) {
	categoryID, _ := evt.Properties["category_id"].(string)
	if categoryID == "" {
		categoryID = labelUnknown
	}
	metrics.Record(ctx, "category_viewed", metrics.L{"category_id": categoryID})
}

func (s *StoreEventService) mapOutOfStockShown(ctx context.Context, evt domain.StoreEvent, _ string) {
	productID, _ := evt.Properties["product_id"].(string)
	if productID == "" {
		productID = labelUnknown
	}
	metrics.Record(ctx, "out_of_stock_shown", metrics.L{"product_id": productID})
}

func (s *StoreEventService) mapBackInStockNotify(ctx context.Context, evt domain.StoreEvent, _ string) {
	productID, _ := evt.Properties["product_id"].(string)
	if productID == "" {
		productID = labelUnknown
	}
	metrics.Record(ctx, "back_in_stock_notify_requested", metrics.L{"product_id": productID})
}

func (s *StoreEventService) mapCatalogFilterApplied(ctx context.Context, evt domain.StoreEvent, _ string) {
	filterKey, _ := evt.Properties["filter_key"].(string)
	if filterKey == "" {
		filterKey = labelUnknown
	}
	metrics.Record(ctx, "catalog_filter_applied", metrics.L{"filter_key": truncate(filterKey, 32)})
}

func (s *StoreEventService) mapRUMJSError(ctx context.Context, evt domain.StoreEvent, _ string) {
	pageType, _ := evt.Properties["page_type"].(string)
	errorType, _ := evt.Properties["error_type"].(string)
	if pageType == "" {
		pageType = labelOther
	}
	if errorType == "" {
		errorType = "Error"
	}
	metrics.Record(ctx, "rum_js_error", metrics.L{
		"page_type":  pageType,
		"error_type": truncate(errorType, 64),
	})
}

func (s *StoreEventService) mapRUMPageView(ctx context.Context, evt domain.StoreEvent, device string) {
	pageType, _ := evt.Properties["page_type"].(string)
	if pageType == "" {
		pageType = labelOther
	}
	metrics.Record(ctx, "rum_page_view", metrics.L{
		"page_type":   pageType,
		"device_type": device,
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
		"device_type":  device,
		"utm_source":   utmSource,
		"utm_medium":   utmMedium,
		"utm_campaign": utmCampaign,
	})
	// Lazy-upsert centroid on first sighting (per (city, country) pair).
	// Skipped when city/country unknown or lat/lng absent — keeps the table
	// free of nonsense rows. ON CONFLICT DO NOTHING means repeat emits cost one
	// round-trip and one no-op.
	if lat, lng, ok := middleware.GetLatLng(ctx); ok && city != labelUnknown && country != labelUnknown {
		if err := s.centroids.Upsert(ctx, city, country, lat, lng); err != nil {
			slog.WarnContext(ctx, "centroid upsert failed", "city", city, "country", country, "err", err)
			// Observable failure signal — constant label keeps cardinality bounded.
			metrics.Record(ctx, "centroid_upsert_error", metrics.L{"reason": "db_write"})
		}
	}
}

// rumMapper builds a mapper for a Web-Vitals metric: resolves the value,
// buckets it, and emits a count-1 PG event with bucket + page_type + device.
func rumMapper(name string) func(*StoreEventService, context.Context, domain.StoreEvent, string) {
	return func(_ *StoreEventService, ctx context.Context, evt domain.StoreEvent, device string) {
		emitRUM(ctx, name, evt, device)
	}
}

func emitRUM(ctx context.Context, metricName string, evt domain.StoreEvent, device string) {
	pageType, _ := evt.Properties["page_type"].(string)
	if pageType == "" {
		pageType = labelOther
	}
	v := rumValue(evt.Properties["value"])
	metrics.Record(ctx, metricName, metrics.L{
		"bucket":      bucketForRUM(metricName, v),
		"page_type":   pageType,
		"device_type": device,
	})
}

// Web-Vitals bucket labels.
const (
	rumGood             = "good"
	rumNeedsImprovement = "needs_improvement"
	rumPoor             = "poor"
)

// bucketForRUM returns the Web-Vitals "good / needs-improvement / poor" bucket
// label for the named metric.
func bucketForRUM(name string, value float64) string {
	switch name {
	case "rum_lcp":
		switch {
		case value <= 2500:
			return rumGood
		case value <= 4000:
			return rumNeedsImprovement
		default:
			return rumPoor
		}
	case "rum_inp":
		switch {
		case value <= 200:
			return rumGood
		case value <= 500:
			return rumNeedsImprovement
		default:
			return rumPoor
		}
	case "rum_cls":
		// CLS shipped from client as value*100 (see rum.ts).
		switch {
		case value <= 10:
			return rumGood
		case value <= 25:
			return rumNeedsImprovement
		default:
			return rumPoor
		}
	case "rum_ttfb":
		switch {
		case value <= 800:
			return rumGood
		case value <= 1800:
			return rumNeedsImprovement
		default:
			return rumPoor
		}
	}
	return labelUnknown
}

// rumValue coerces the beacon-shipped value (number-as-JSON or string) to a float.
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

// truncate clips s to at most n bytes — defensive bound on metric labels.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
