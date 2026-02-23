package dynamodb

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
)

// AnalyticsRepository implements domain.AnalyticsRepository
type AnalyticsRepository struct {
	client *Client
}

// NewAnalyticsRepository creates a new AnalyticsRepository
func NewAnalyticsRepository(client *Client) *AnalyticsRepository {
	return &AnalyticsRepository{
		client: client,
	}
}

// GetDashboardStats retrieves dashboard statistics from the DASHBOARD#CURRENT item
func (r *AnalyticsRepository) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.analyticsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DASHBOARD#CURRENT"},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return &domain.DashboardStats{}, nil
	}

	var stats domain.DashboardStats
	if err := attributevalue.UnmarshalMap(result.Item, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetSalesAnalytics retrieves sales analytics for a period by reading REVENUE#DAILY# aggregates.
func (r *AnalyticsRepository) GetSalesAnalytics(ctx context.Context, period string, startDate, endDate time.Time) (*domain.SalesAnalytics, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	rows, err := r.GetDailyAggregates(ctx, "REVENUE", startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get revenue aggregates: %w", err)
	}

	var totalRevenue int64
	var totalOrders int
	dataPoints := make([]domain.SalesDataPoint, 0, len(rows))
	salesByDay := make([]domain.DailySales, 0, len(rows))

	for _, row := range rows {
		dayRevenue := extractMapInt64(row, "total_revenue")
		dayOrders := extractMapInt(row, "total_orders")
		dayDate := extractMapString(row, "date")

		totalRevenue += dayRevenue
		totalOrders += dayOrders

		dataPoints = append(dataPoints, domain.SalesDataPoint{
			Date:   dayDate,
			Sales:  dayRevenue,
			Orders: dayOrders,
		})
		salesByDay = append(salesByDay, domain.DailySales{
			Date:   dayDate,
			Sales:  dayRevenue,
			Orders: dayOrders,
		})
	}

	var aov float64
	if totalOrders > 0 {
		aov = float64(totalRevenue) / float64(totalOrders)
	}

	return &domain.SalesAnalytics{
		Period:            period,
		StartDate:         startDate,
		EndDate:           endDate,
		TotalSales:        float64(totalRevenue),
		TotalOrders:       totalOrders,
		AverageOrderValue: aov,
		DataPoints:        dataPoints,
		SalesByDay:        salesByDay,
	}, nil
}

// GetTopProducts retrieves top products by view count from PRODUCTS#DAILY# aggregates.
// It aggregates the top_by_views arrays across all days and returns the top N by total views.
func (r *AnalyticsRepository) GetTopProducts(ctx context.Context, limit int, startDate, endDate time.Time) ([]domain.TopProduct, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	rows, err := r.GetDailyAggregates(ctx, "PRODUCTS", startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get product aggregates: %w", err)
	}

	// Aggregate view counts across all days by product_id
	viewCounts := make(map[string]int)
	for _, row := range rows {
		topByViews, ok := row["top_by_views"]
		if !ok {
			continue
		}
		// DynamoDB unmarshals lists as []interface{}
		items, ok := topByViews.([]interface{})
		if !ok {
			continue
		}
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			productID := extractMapString(m, "product_id")
			count := extractMapInt(m, "count")
			if productID != "" {
				viewCounts[productID] += count
			}
		}
	}

	// Sort by views descending
	type entry struct {
		id    string
		views int
	}
	entries := make([]entry, 0, len(viewCounts))
	for id, views := range viewCounts {
		entries = append(entries, entry{id: id, views: views})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].views > entries[j].views })
	if len(entries) > limit {
		entries = entries[:limit]
	}

	products := make([]domain.TopProduct, len(entries))
	for i, e := range entries {
		products[i] = domain.TopProduct{
			ProductID: e.id,
			UnitsSold: e.views,
		}
	}
	return products, nil
}

// GetTopCategories retrieves top performing categories by revenue from REVENUE#DAILY# aggregates.
// It extracts the revenue_by_category maps across all days and sums revenue per category.
func (r *AnalyticsRepository) GetTopCategories(ctx context.Context, limit int, startDate, endDate time.Time) ([]domain.TopCategory, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	rows, err := r.GetDailyAggregates(ctx, "REVENUE", startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get revenue aggregates: %w", err)
	}

	// Aggregate revenue by category across all days
	categoryRevenue := make(map[string]int64)
	for _, row := range rows {
		byCategory, ok := row["revenue_by_category"]
		if !ok {
			continue
		}
		catMap, ok := byCategory.(map[string]interface{})
		if !ok {
			continue
		}
		for catID, rev := range catMap {
			categoryRevenue[catID] += toInt64(rev)
		}
	}

	// Sort by revenue descending
	type entry struct {
		id      string
		revenue int64
	}
	entries := make([]entry, 0, len(categoryRevenue))
	for id, rev := range categoryRevenue {
		entries = append(entries, entry{id: id, revenue: rev})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].revenue > entries[j].revenue })
	if len(entries) > limit {
		entries = entries[:limit]
	}

	categories := make([]domain.TopCategory, len(entries))
	for i, e := range entries {
		categories[i] = domain.TopCategory{
			CategoryID: e.id,
			Revenue:    e.revenue,
		}
	}
	return categories, nil
}

// GetCustomerAnalytics retrieves customer analytics from CUSTOMERS#DAILY# aggregates.
func (r *AnalyticsRepository) GetCustomerAnalytics(ctx context.Context, startDate, endDate time.Time) (*domain.CustomerAnalytics, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	rows, err := r.GetDailyAggregates(ctx, "CUSTOMERS", startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get customer aggregates: %w", err)
	}

	var totalNew int
	var totalUniqueVisitors int
	var totalReturning int

	for _, row := range rows {
		totalNew += extractMapInt(row, "new_registrations")
		totalUniqueVisitors += extractMapInt(row, "unique_visitors")
		totalReturning += extractMapInt(row, "returning_visitors")
	}

	// Total customers = new registrations in the period (the best available from aggregates)
	totalCustomers := totalUniqueVisitors
	if totalCustomers == 0 {
		totalCustomers = totalNew
	}

	var avgOrdersPerCustomer float64
	if totalCustomers > 0 {
		// Read total orders from revenue aggregates for the same period to compute the ratio
		revenueRows, err := r.GetDailyAggregates(ctx, "REVENUE", startStr, endStr)
		if err == nil {
			var totalOrders int
			for _, row := range revenueRows {
				totalOrders += extractMapInt(row, "total_orders")
			}
			if totalOrders > 0 {
				avgOrdersPerCustomer = float64(totalOrders) / float64(totalCustomers)
			}
		}
	}

	return &domain.CustomerAnalytics{
		TotalCustomers:           totalCustomers,
		NewCustomers:             totalNew,
		ReturningCustomers:       totalReturning,
		AverageOrdersPerCustomer: avgOrdersPerCustomer,
	}, nil
}

// GetInventoryAnalytics retrieves inventory analytics.
// The daily aggregator does not write CATALOG#DAILY# records yet, so this reads
// the dashboard live counters for low_stock_count and out_of_stock_count, and
// returns empty/zero values for fields that are not yet aggregated.
func (r *AnalyticsRepository) GetInventoryAnalytics(ctx context.Context) (*domain.InventoryAnalytics, error) {
	// Read current dashboard stats for inventory counters
	stats, err := r.GetDashboardStats(ctx)
	if err != nil {
		// Return empty defaults rather than propagating the error
		return &domain.InventoryAnalytics{}, nil
	}

	return &domain.InventoryAnalytics{
		TotalProducts:   stats.TotalProducts,
		LowStockCount:   stats.LowStockCount,
		OutOfStockCount: stats.OutOfStockCount,
	}, nil
}

// RecordPageView records a page view for analytics
func (r *AnalyticsRepository) RecordPageView(ctx context.Context, page string, userID string) error {
	// TODO: Implement with DynamoDB put
	return nil
}

// RecordEvent records a custom event
func (r *AnalyticsRepository) RecordEvent(ctx context.Context, eventType string, data map[string]interface{}) error {
	// TODO: Implement with DynamoDB put
	return nil
}

// allowedCounterFields is the set of fields that IncrementDashboardCounter may write.
var allowedCounterFields = map[string]struct{}{
	"today_page_views":    {},
	"today_product_views": {},
	"today_add_to_carts":  {},
	"today_orders":        {},
	"today_revenue":       {},
	"total_orders":        {},
	"total_revenue":       {},
	"total_products":      {},
	"total_customers":     {},
	"low_stock_count":     {},
	"out_of_stock_count":  {},
}

// IncrementDashboardCounter atomically increments a counter field on the DASHBOARD#CURRENT item.
// Uses DynamoDB ADD to perform an atomic increment. Only allowed field names are accepted.
func (r *AnalyticsRepository) IncrementDashboardCounter(ctx context.Context, field string, amount int64) error {
	if _, ok := allowedCounterFields[field]; !ok {
		return fmt.Errorf("invalid dashboard counter field: %s", field)
	}
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.analyticsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DASHBOARD#CURRENT"},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("ADD #field :val"),
		ExpressionAttributeNames: map[string]string{
			"#field": field,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", amount)},
		},
	})
	return err
}

// PutDailyAggregate writes a pre-computed daily aggregate record to the analytics table.
// The data struct is marshalled and the PK/SK keys are set explicitly.
func (r *AnalyticsRepository) PutDailyAggregate(ctx context.Context, pk string, sk string, data interface{}) error {
	item, err := attributevalue.MarshalMap(data)
	if err != nil {
		return err
	}
	item["PK"] = &types.AttributeValueMemberS{Value: pk}
	item["SK"] = &types.AttributeValueMemberS{Value: sk}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.analyticsTable),
		Item:      item,
	})
	return err
}

// PutDailyStats archives the current dashboard counters as a historical record for the given date.
// Stored under PK=DASHBOARD#STATS#<date> SK=METADATA.
func (r *AnalyticsRepository) PutDailyStats(ctx context.Context, date string, stats *domain.DashboardStats) error {
	item, err := attributevalue.MarshalMap(stats)
	if err != nil {
		return err
	}
	item["PK"] = &types.AttributeValueMemberS{Value: "DASHBOARD#STATS#" + date}
	item["SK"] = &types.AttributeValueMemberS{Value: "METADATA"}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.analyticsTable),
		Item:      item,
	})
	return err
}

// ResetDashboardCurrent resets all counters to zero by deleting the DASHBOARD#CURRENT item.
// The next IncrementDashboardCounter call will recreate it via DynamoDB ADD on a non-existent item.
func (r *AnalyticsRepository) ResetDashboardCurrent(ctx context.Context) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.analyticsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "DASHBOARD#CURRENT"},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	return err
}

// GetDailyAggregates retrieves pre-computed daily aggregate records for a prefix and date range.
// It iterates over each date from startDate to endDate and does a GetItem for
// PK={prefix}#DAILY#{date}, SK=METADATA.
func (r *AnalyticsRepository) GetDailyAggregates(ctx context.Context, prefix string, startDate string, endDate string) ([]map[string]interface{}, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	var results []map[string]interface{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		pk := fmt.Sprintf("%s#DAILY#%s", prefix, dateStr)

		result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(r.client.analyticsTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: pk},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get daily aggregate for %s: %w", dateStr, err)
		}
		if result.Item == nil {
			continue
		}

		var item map[string]interface{}
		if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
			return nil, fmt.Errorf("failed to unmarshal daily aggregate for %s: %w", dateStr, err)
		}
		results = append(results, item)
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// Map extraction helpers for untyped DynamoDB aggregate data
// ---------------------------------------------------------------------------

// extractMapString extracts a string value from an untyped map.
func extractMapString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// extractMapInt extracts an int value from an untyped map.
// DynamoDB SDK unmarshals numbers as float64 via attributevalue.UnmarshalMap into interface{}.
func extractMapInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toInt(v)
}

// extractMapInt64 extracts an int64 value from an untyped map.
func extractMapInt64(m map[string]interface{}, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toInt64(v)
}

// toInt converts an interface{} numeric value to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// toInt64 converts an interface{} numeric value to int64.
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// Ensure interface compliance
var _ domain.AnalyticsRepository = (*AnalyticsRepository)(nil)
