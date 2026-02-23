package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
)

// AnalyticsAggregator computes daily aggregate metrics from raw events and
// writes them to the analytics table. It is invoked once per day by an
// EventBridge schedule (or manually) for the previous day's data.
type AnalyticsAggregator struct {
	eventsRepo    domain.EventsRepository
	analyticsRepo domain.AnalyticsRepository
	logger        *logger.Logger
}

// NewAnalyticsAggregator creates a new AnalyticsAggregator.
func NewAnalyticsAggregator(
	eventsRepo domain.EventsRepository,
	analyticsRepo domain.AnalyticsRepository,
	log *logger.Logger,
) *AnalyticsAggregator {
	return &AnalyticsAggregator{
		eventsRepo:    eventsRepo,
		analyticsRepo: analyticsRepo,
		logger:        log,
	}
}

// AggregateDate queries all raw events for the given date (YYYY-MM-DD),
// computes aggregates across five categories, and writes them to the analytics table.
// Each category is best-effort: errors are logged but do not abort the overall run.
func (a *AnalyticsAggregator) AggregateDate(ctx context.Context, date string) error {
	events, err := a.eventsRepo.QueryByDate(ctx, date)
	if err != nil {
		return fmt.Errorf("query events for %s: %w", date, err)
	}

	if len(events) == 0 {
		a.logger.WithContext(ctx).Infof("No events to aggregate for %s", date)
		return nil
	}

	a.logger.WithContext(ctx).Infof("Aggregating %d events for %s", len(events), date)

	// Aggregate each category (best-effort, log errors)
	a.aggregateFunnel(ctx, date, events)
	a.aggregateRevenue(ctx, date, events)
	a.aggregateCustomers(ctx, date, events)
	a.aggregateEngagement(ctx, date, events)
	a.aggregateProducts(ctx, date, events)

	// Archive yesterday's live counters and reset for the new day
	a.resetDashboardCounters(ctx, date)

	return nil
}

// resetDashboardCounters archives the current live dashboard counters as a
// historical record for the given date and then resets DASHBOARD#CURRENT to
// zeros so the new day starts fresh.
func (a *AnalyticsAggregator) resetDashboardCounters(ctx context.Context, date string) {
	// Read current counters
	stats, err := a.analyticsRepo.GetDashboardStats(ctx)
	if err != nil {
		a.logger.WithContext(ctx).WithError(err).Error("Failed to read dashboard stats for archival")
		return
	}

	// Archive as historical record
	if err := a.analyticsRepo.PutDailyStats(ctx, date, stats); err != nil {
		a.logger.WithContext(ctx).WithError(err).Error("Failed to archive dashboard stats")
		return
	}

	// Reset current counters
	if err := a.analyticsRepo.ResetDashboardCurrent(ctx); err != nil {
		a.logger.WithContext(ctx).WithError(err).Error("Failed to reset dashboard counters")
	}
}

// ---------------------------------------------------------------------------
// Funnel aggregation
// ---------------------------------------------------------------------------

// funnelAggregate holds conversion-funnel metrics for the day.
type funnelAggregate struct {
	Date             string  `dynamodbav:"date" json:"date"`
	PageViews        int     `dynamodbav:"page_views" json:"page_views"`
	ProductViews     int     `dynamodbav:"product_views" json:"product_views"`
	AddToCarts       int     `dynamodbav:"add_to_carts" json:"add_to_carts"`
	CheckoutsStarted int     `dynamodbav:"checkouts_started" json:"checkouts_started"`
	OrdersCreated    int     `dynamodbav:"orders_created" json:"orders_created"`
	ViewToCartRate   float64 `dynamodbav:"view_to_cart_rate" json:"view_to_cart_rate"`
	CartToCheckout   float64 `dynamodbav:"cart_to_checkout_rate" json:"cart_to_checkout_rate"`
	CheckoutToOrder  float64 `dynamodbav:"checkout_to_order_rate" json:"checkout_to_order_rate"`
	OverallRate      float64 `dynamodbav:"overall_rate" json:"overall_rate"`
}

func (a *AnalyticsAggregator) aggregateFunnel(ctx context.Context, date string, events []domain.StoreEvent) {
	// Count unique sessions per funnel step
	pageViewSessions := make(map[string]struct{})
	productViewSessions := make(map[string]struct{})
	addToCartSessions := make(map[string]struct{})
	checkoutSessions := make(map[string]struct{})
	orderSessions := make(map[string]struct{})

	for _, evt := range events {
		sid := evt.SessionID
		switch evt.EventType {
		case "page_view":
			pageViewSessions[sid] = struct{}{}
		case "product_viewed":
			productViewSessions[sid] = struct{}{}
		case "add_to_cart":
			addToCartSessions[sid] = struct{}{}
		case "checkout_started":
			checkoutSessions[sid] = struct{}{}
		case "order_created", "order.created":
			orderSessions[sid] = struct{}{}
		}
	}

	pvCount := len(pageViewSessions)
	prodCount := len(productViewSessions)
	atcCount := len(addToCartSessions)
	coCount := len(checkoutSessions)
	ordCount := len(orderSessions)

	agg := funnelAggregate{
		Date:             date,
		PageViews:        pvCount,
		ProductViews:     prodCount,
		AddToCarts:       atcCount,
		CheckoutsStarted: coCount,
		OrdersCreated:    ordCount,
		ViewToCartRate:   safeRate(atcCount, prodCount),
		CartToCheckout:   safeRate(coCount, atcCount),
		CheckoutToOrder:  safeRate(ordCount, coCount),
		OverallRate:      safeRate(ordCount, pvCount),
	}

	pk := fmt.Sprintf("FUNNEL#DAILY#%s", date)
	if err := a.analyticsRepo.PutDailyAggregate(ctx, pk, "METADATA", agg); err != nil {
		a.logger.WithContext(ctx).WithError(err).Errorf("failed to write funnel aggregate for %s", date)
	}
}

// ---------------------------------------------------------------------------
// Revenue aggregation
// ---------------------------------------------------------------------------

// revenueAggregate holds revenue metrics for the day.
type revenueAggregate struct {
	Date              string            `dynamodbav:"date" json:"date"`
	TotalRevenue      int64             `dynamodbav:"total_revenue" json:"total_revenue"`
	TotalOrders       int               `dynamodbav:"total_orders" json:"total_orders"`
	AverageOrderValue int64             `dynamodbav:"average_order_value" json:"average_order_value"`
	RevenueByMethod   map[string]int64  `dynamodbav:"revenue_by_method" json:"revenue_by_method"`
	RevenueByCategory map[string]int64  `dynamodbav:"revenue_by_category" json:"revenue_by_category"`
	OrdersByStatus    map[string]int    `dynamodbav:"orders_by_status" json:"orders_by_status"`
}

func (a *AnalyticsAggregator) aggregateRevenue(ctx context.Context, date string, events []domain.StoreEvent) {
	var totalRevenue int64
	var orderCount int
	revenueByMethod := make(map[string]int64)
	revenueByCategory := make(map[string]int64)
	ordersByStatus := make(map[string]int)

	for _, evt := range events {
		switch evt.EventType {
		case "order_created", "order.created":
			orderCount++
			amount := extractInt64Prop(evt.Properties, "total_amount")
			totalRevenue += amount

			method := extractStringProp(evt.Properties, "payment_method")
			if method == "" {
				method = "unknown"
			}
			revenueByMethod[method] += amount

			categoryID := extractStringProp(evt.Properties, "category_id")
			if categoryID != "" {
				revenueByCategory[categoryID] += amount
			}

		case "order.cancelled":
			ordersByStatus["cancelled"]++

		case "payment.received":
			ordersByStatus["paid"]++

		case "payment.refunded":
			ordersByStatus["refunded"]++
		}
	}

	ordersByStatus["created"] = orderCount

	var aov int64
	if orderCount > 0 {
		aov = totalRevenue / int64(orderCount)
	}

	agg := revenueAggregate{
		Date:              date,
		TotalRevenue:      totalRevenue,
		TotalOrders:       orderCount,
		AverageOrderValue: aov,
		RevenueByMethod:   revenueByMethod,
		RevenueByCategory: revenueByCategory,
		OrdersByStatus:    ordersByStatus,
	}

	pk := fmt.Sprintf("REVENUE#DAILY#%s", date)
	if err := a.analyticsRepo.PutDailyAggregate(ctx, pk, "METADATA", agg); err != nil {
		a.logger.WithContext(ctx).WithError(err).Errorf("failed to write revenue aggregate for %s", date)
	}
}

// ---------------------------------------------------------------------------
// Customer aggregation
// ---------------------------------------------------------------------------

// customerAggregate holds customer metrics for the day.
type customerAggregate struct {
	Date              string         `dynamodbav:"date" json:"date"`
	NewRegistrations  int            `dynamodbav:"new_registrations" json:"new_registrations"`
	UniqueVisitors    int            `dynamodbav:"unique_visitors" json:"unique_visitors"`
	ReturningVisitors int            `dynamodbav:"returning_visitors" json:"returning_visitors"`
	ByDeviceType      map[string]int `dynamodbav:"by_device_type" json:"by_device_type"`
	ByLocation        map[string]int `dynamodbav:"by_location" json:"by_location"`
}

func (a *AnalyticsAggregator) aggregateCustomers(ctx context.Context, date string, events []domain.StoreEvent) {
	var newRegistrations int
	visitorSessions := make(map[string]int) // visitor_id -> session count
	deviceCounts := make(map[string]int)
	locationCounts := make(map[string]int)

	for _, evt := range events {
		if evt.EventType == "customer.registered" || evt.EventType == "customer_registered" {
			newRegistrations++
		}

		if evt.VisitorID != "" && evt.VisitorID != "backend" {
			visitorSessions[evt.VisitorID]++
		}

		if evt.DeviceType != "" && evt.DeviceType != "server" {
			deviceCounts[evt.DeviceType]++
		}

		location := extractStringProp(evt.Properties, "location")
		if location != "" {
			locationCounts[location]++
		}
	}

	uniqueVisitors := len(visitorSessions)
	var returning int
	for _, count := range visitorSessions {
		if count > 1 {
			returning++
		}
	}

	agg := customerAggregate{
		Date:              date,
		NewRegistrations:  newRegistrations,
		UniqueVisitors:    uniqueVisitors,
		ReturningVisitors: returning,
		ByDeviceType:      deviceCounts,
		ByLocation:        locationCounts,
	}

	pk := fmt.Sprintf("CUSTOMERS#DAILY#%s", date)
	if err := a.analyticsRepo.PutDailyAggregate(ctx, pk, "METADATA", agg); err != nil {
		a.logger.WithContext(ctx).WithError(err).Errorf("failed to write customer aggregate for %s", date)
	}
}

// ---------------------------------------------------------------------------
// Engagement aggregation
// ---------------------------------------------------------------------------

// engagementAggregate holds engagement metrics for the day.
type engagementAggregate struct {
	Date               string         `dynamodbav:"date" json:"date"`
	TotalSessions      int            `dynamodbav:"total_sessions" json:"total_sessions"`
	BounceCount        int            `dynamodbav:"bounce_count" json:"bounce_count"`
	BounceRate         float64        `dynamodbav:"bounce_rate" json:"bounce_rate"`
	AvgSessionDuration float64        `dynamodbav:"avg_session_duration" json:"avg_session_duration"`
	TopPages           []pageCount    `dynamodbav:"top_pages" json:"top_pages"`
	AvgScrollDepth     float64        `dynamodbav:"avg_scroll_depth" json:"avg_scroll_depth"`
}

// pageCount is a helper for top-pages reporting.
type pageCount struct {
	Path  string `dynamodbav:"path" json:"path"`
	Views int    `dynamodbav:"views" json:"views"`
}

func (a *AnalyticsAggregator) aggregateEngagement(ctx context.Context, date string, events []domain.StoreEvent) {
	sessionEvents := make(map[string][]domain.StoreEvent) // session_id -> events
	pageCounts := make(map[string]int)
	var scrollDepthSum float64
	var scrollDepthCount int

	for _, evt := range events {
		if evt.SessionID != "" && evt.SessionID != "backend" {
			sessionEvents[evt.SessionID] = append(sessionEvents[evt.SessionID], evt)
		}

		if evt.PagePath != "" {
			pageCounts[evt.PagePath]++
		}

		if evt.EventType == "scroll_depth" {
			depth := extractFloat64Prop(evt.Properties, "max_depth_percent")
			if depth > 0 {
				scrollDepthSum += depth
				scrollDepthCount++
			}
		}
	}

	totalSessions := len(sessionEvents)
	var bounceCount int
	var totalDuration float64
	var durationCount int

	for _, sessEvts := range sessionEvents {
		if len(sessEvts) <= 1 {
			bounceCount++
		}
		if len(sessEvts) >= 2 {
			// Session duration = last event timestamp - first event timestamp
			first := sessEvts[0].Timestamp
			last := sessEvts[len(sessEvts)-1].Timestamp
			dur := last.Sub(first).Seconds()
			if dur > 0 {
				totalDuration += dur
				durationCount++
			}
		}
	}

	// Build top pages (top 20)
	type pc struct {
		path  string
		count int
	}
	var pages []pc
	for p, c := range pageCounts {
		pages = append(pages, pc{path: p, count: c})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].count > pages[j].count })
	if len(pages) > 20 {
		pages = pages[:20]
	}
	topPages := make([]pageCount, len(pages))
	for i, p := range pages {
		topPages[i] = pageCount{Path: p.path, Views: p.count}
	}

	var avgDuration float64
	if durationCount > 0 {
		avgDuration = totalDuration / float64(durationCount)
	}
	var avgScroll float64
	if scrollDepthCount > 0 {
		avgScroll = scrollDepthSum / float64(scrollDepthCount)
	}

	agg := engagementAggregate{
		Date:               date,
		TotalSessions:      totalSessions,
		BounceCount:        bounceCount,
		BounceRate:         safeRate(bounceCount, totalSessions),
		AvgSessionDuration: avgDuration,
		TopPages:           topPages,
		AvgScrollDepth:     avgScroll,
	}

	pk := fmt.Sprintf("ENGAGEMENT#DAILY#%s", date)
	if err := a.analyticsRepo.PutDailyAggregate(ctx, pk, "METADATA", agg); err != nil {
		a.logger.WithContext(ctx).WithError(err).Errorf("failed to write engagement aggregate for %s", date)
	}
}

// ---------------------------------------------------------------------------
// Product aggregation
// ---------------------------------------------------------------------------

// productAggregate holds product-level metrics for the day.
type productAggregate struct {
	Date            string         `dynamodbav:"date" json:"date"`
	TotalViews      int            `dynamodbav:"total_views" json:"total_views"`
	TotalAddToCarts int            `dynamodbav:"total_add_to_carts" json:"total_add_to_carts"`
	TopByViews      []productStat  `dynamodbav:"top_by_views" json:"top_by_views"`
	TopByAddToCart  []productStat  `dynamodbav:"top_by_add_to_cart" json:"top_by_add_to_cart"`
}

// productStat tracks per-product view/cart counts.
type productStat struct {
	ProductID string `dynamodbav:"product_id" json:"product_id"`
	Count     int    `dynamodbav:"count" json:"count"`
}

func (a *AnalyticsAggregator) aggregateProducts(ctx context.Context, date string, events []domain.StoreEvent) {
	viewCounts := make(map[string]int)    // product_id -> views
	cartCounts := make(map[string]int)    // product_id -> add_to_cart

	var totalViews, totalATC int

	for _, evt := range events {
		productID := extractStringProp(evt.Properties, "product_id")
		if productID == "" {
			continue
		}

		switch evt.EventType {
		case "product_viewed":
			viewCounts[productID]++
			totalViews++
		case "add_to_cart":
			cartCounts[productID]++
			totalATC++
		}
	}

	topViews := topN(viewCounts, 20)
	topCarts := topN(cartCounts, 20)

	agg := productAggregate{
		Date:            date,
		TotalViews:      totalViews,
		TotalAddToCarts: totalATC,
		TopByViews:      topViews,
		TopByAddToCart:  topCarts,
	}

	pk := fmt.Sprintf("PRODUCTS#DAILY#%s", date)
	if err := a.analyticsRepo.PutDailyAggregate(ctx, pk, "METADATA", agg); err != nil {
		a.logger.WithContext(ctx).WithError(err).Errorf("failed to write product aggregate for %s", date)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// safeRate computes numerator/denominator as a float64, returning 0 if denominator is 0.
func safeRate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

// extractStringProp extracts a string value from the event properties map.
func extractStringProp(props map[string]interface{}, key string) string {
	if props == nil {
		return ""
	}
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractInt64Prop extracts an int64 value from the event properties map.
// JSON numbers are typically unmarshalled as float64, so we handle that case.
func extractInt64Prop(props map[string]interface{}, key string) int64 {
	if props == nil {
		return 0
	}
	v, ok := props[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// extractFloat64Prop extracts a float64 value from the event properties map.
func extractFloat64Prop(props map[string]interface{}, key string) float64 {
	if props == nil {
		return 0
	}
	v, ok := props[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

// topN returns the top N entries from a count map, sorted by count descending.
func topN(counts map[string]int, n int) []productStat {
	type entry struct {
		id    string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for id, c := range counts {
		entries = append(entries, entry{id: id, count: c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })
	if len(entries) > n {
		entries = entries[:n]
	}
	result := make([]productStat, len(entries))
	for i, e := range entries {
		result[i] = productStat{ProductID: e.id, Count: e.count}
	}
	return result
}
