package domain

import (
	"context"
	"time"
)

// ==================== ANALYTICS ENTITIES ====================

// DashboardStats contains overall dashboard statistics
type DashboardStats struct {
	// Today's metrics (live counters from DASHBOARD#CURRENT)
	TodayOrders          int   `json:"today_orders" dynamodbav:"today_orders"`
	TodayRevenue         int64 `json:"today_revenue" dynamodbav:"today_revenue"`
	TodayVisitors        int   `json:"today_visitors" dynamodbav:"today_visitors"`
	TodayAddToCarts      int   `json:"today_add_to_carts" dynamodbav:"today_add_to_carts"`
	TodayProductViews    int   `json:"today_product_views" dynamodbav:"today_product_views"`
	TodayPaymentsSuccess int   `json:"today_payments_success" dynamodbav:"today_payments_success"`
	TodayPaymentsFailed  int   `json:"today_payments_failed" dynamodbav:"today_payments_failed"`
	TodayNewCustomers    int   `json:"today_new_customers" dynamodbav:"today_new_customers"`

	// This week
	WeekOrders  int   `json:"week_orders" dynamodbav:"week_orders"`
	WeekRevenue int64 `json:"week_revenue" dynamodbav:"week_revenue"`

	// This month
	MonthOrders  int   `json:"month_orders" dynamodbav:"month_orders"`
	MonthRevenue int64 `json:"month_revenue" dynamodbav:"month_revenue"`

	// All time
	TotalOrders    int     `json:"total_orders" dynamodbav:"total_orders"`
	TotalRevenue   float64 `json:"total_revenue" dynamodbav:"total_revenue"`
	TotalCustomers int     `json:"total_customers" dynamodbav:"total_customers"`
	TotalProducts  int     `json:"total_products" dynamodbav:"total_products"`

	// Growth metrics
	RevenueGrowth     float64 `json:"revenue_growth" dynamodbav:"revenue_growth"`
	OrdersGrowth      float64 `json:"orders_growth" dynamodbav:"orders_growth"`
	CustomersGrowth   float64 `json:"customers_growth" dynamodbav:"customers_growth"`
	AverageOrderValue float64 `json:"average_order_value" dynamodbav:"average_order_value"`

	// Inventory
	LowStockCount   int `json:"low_stock_count" dynamodbav:"low_stock_count"`
	OutOfStockCount int `json:"out_of_stock_count" dynamodbav:"out_of_stock_count"`

	// Orders by status
	PendingOrders    int `json:"pending_orders" dynamodbav:"pending_orders"`
	ProcessingOrders int `json:"processing_orders" dynamodbav:"processing_orders"`
	ShippedOrders    int `json:"shipped_orders" dynamodbav:"shipped_orders"`
}

// SalesAnalytics contains sales analytics data
type SalesAnalytics struct {
	Period           string           `json:"period"` // daily, weekly, monthly
	StartDate        time.Time        `json:"start_date"`
	EndDate          time.Time        `json:"end_date"`
	TotalSales       float64          `json:"total_sales"`
	TotalOrders      int              `json:"total_orders"`
	AverageOrderValue float64         `json:"average_order_value"`
	DataPoints       []SalesDataPoint `json:"data_points"`
	SalesByDay       []DailySales     `json:"sales_by_day,omitempty"`
	TopSellingItems  []TopProduct     `json:"top_selling_items,omitempty"`
}

// SalesDataPoint represents a single data point in sales analytics
type SalesDataPoint struct {
	Date     string `json:"date"`
	Sales    int64  `json:"sales"`
	Orders   int    `json:"orders"`
}

// TopProduct represents a top-selling product
type TopProduct struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	SKU          string `json:"sku"`
	UnitsSold    int    `json:"units_sold"`
	Revenue      int64  `json:"revenue"`
}

// TopCategory represents a top-performing category
type TopCategory struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	UnitsSold    int    `json:"units_sold"`
	Revenue      int64  `json:"revenue"`
}

// CustomerAnalytics contains customer analytics data
type CustomerAnalytics struct {
	TotalCustomers          int           `json:"total_customers"`
	NewCustomers            int           `json:"new_customers"`
	ReturningCustomers      int           `json:"returning_customers"`
	AverageOrdersPerCustomer float64      `json:"average_orders_per_customer"`
	TopCustomers            []TopCustomer `json:"top_customers,omitempty"`
}

// TopCustomer represents a top customer
type TopCustomer struct {
	CustomerID   string `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	OrderCount   int    `json:"order_count"`
	TotalSpent   int64  `json:"total_spent"`
}

// InventoryAnalytics contains inventory analytics data
type InventoryAnalytics struct {
	TotalProducts       int          `json:"total_products"`
	TotalInventoryValue float64      `json:"total_inventory_value"`
	LowStockCount       int          `json:"low_stock_count"`
	OutOfStockCount     int          `json:"out_of_stock_count"`
	LowStockItems       []*Inventory `json:"low_stock_items,omitempty"`
	OutOfStockItems     []*Inventory `json:"out_of_stock_items,omitempty"`
	TopMovingItems      []MovingItem `json:"top_moving_items,omitempty"`
	SlowMovingItems     []MovingItem `json:"slow_moving_items,omitempty"`
}

// MovingItem represents a product's movement stats
type MovingItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	UnitsSold   int    `json:"units_sold"`
	DaysInStock int    `json:"days_in_stock"`
}

// DateRange represents a start and end date for analytics queries
type DateRange struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// FunnelStep represents a single step in the conversion funnel
type FunnelStep struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate,omitempty"`
}

// FunnelAnalytics contains conversion funnel data for a date range
type FunnelAnalytics struct {
	Period            DateRange    `json:"period"`
	Steps             []FunnelStep `json:"steps"`
	OverallConversion float64      `json:"overall_conversion"`
}

// PageStats represents page view statistics
type PageStats struct {
	Path  string `json:"path"`
	Views int    `json:"views"`
}

// EngagementAnalytics contains user engagement data for a date range
type EngagementAnalytics struct {
	Period             DateRange          `json:"period"`
	TotalSessions      int                `json:"total_sessions"`
	AvgSessionDuration int                `json:"avg_session_duration_seconds"`
	BounceRate         float64            `json:"bounce_rate"`
	DeviceBreakdown    map[string]float64 `json:"device_breakdown"`
	TopPages           []PageStats        `json:"top_pages"`
}

// ==================== ANALYTICS REPOSITORY ====================

// AnalyticsRepository defines the interface for analytics data access
type AnalyticsRepository interface {
	// GetDashboardStats retrieves overall dashboard statistics
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// GetSalesAnalytics retrieves sales analytics for a period
	GetSalesAnalytics(ctx context.Context, period string, startDate, endDate time.Time) (*SalesAnalytics, error)

	// GetTopProducts retrieves top selling products
	GetTopProducts(ctx context.Context, limit int, startDate, endDate time.Time) ([]TopProduct, error)

	// GetTopCategories retrieves top performing categories
	GetTopCategories(ctx context.Context, limit int, startDate, endDate time.Time) ([]TopCategory, error)

	// GetCustomerAnalytics retrieves customer analytics
	GetCustomerAnalytics(ctx context.Context, startDate, endDate time.Time) (*CustomerAnalytics, error)

	// GetInventoryAnalytics retrieves inventory analytics
	GetInventoryAnalytics(ctx context.Context) (*InventoryAnalytics, error)

	// RecordPageView records a page view for analytics
	RecordPageView(ctx context.Context, page string, userID string) error

	// RecordEvent records a custom event
	RecordEvent(ctx context.Context, eventType string, data map[string]interface{}) error

	// IncrementDashboardCounter atomically increments a counter field on the DASHBOARD#CURRENT item
	IncrementDashboardCounter(ctx context.Context, field string, amount int64) error

	// PutDailyAggregate writes a pre-computed daily aggregate record to the analytics table
	PutDailyAggregate(ctx context.Context, pk string, sk string, data interface{}) error

	// PutDailyStats archives the current dashboard counters as a historical record for the given date
	PutDailyStats(ctx context.Context, date string, stats *DashboardStats) error

	// ResetDashboardCurrent resets all dashboard counters by deleting the DASHBOARD#CURRENT item
	ResetDashboardCurrent(ctx context.Context) error

	// GetDailyAggregates retrieves pre-computed daily aggregate records for a prefix and date range
	GetDailyAggregates(ctx context.Context, prefix string, startDate string, endDate string) ([]map[string]interface{}, error)
}

// ==================== ANALYTICS SERVICE ====================

// AnalyticsService defines the interface for analytics operations
type AnalyticsService interface {
	// GetDashboardStats retrieves overall dashboard statistics
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// GetSalesAnalytics retrieves sales analytics for a period
	GetSalesAnalytics(ctx context.Context, req SalesAnalyticsRequest) (*SalesAnalytics, error)

	// GetTopProducts retrieves top selling products
	GetTopProducts(ctx context.Context, limit int, startDate, endDate time.Time) ([]TopProduct, error)

	// GetTopCategories retrieves top performing categories
	GetTopCategories(ctx context.Context, limit int, startDate, endDate time.Time) ([]TopCategory, error)

	// GetCustomerAnalytics retrieves customer analytics
	GetCustomerAnalytics(ctx context.Context, startDate, endDate time.Time) (*CustomerAnalytics, error)

	// GetInventoryAnalytics retrieves inventory analytics
	GetInventoryAnalytics(ctx context.Context) (*InventoryAnalytics, error)

	// GetFunnelAnalytics retrieves conversion funnel analytics for a date range
	GetFunnelAnalytics(ctx context.Context, startDate, endDate string) (*FunnelAnalytics, error)

	// GetEngagementAnalytics retrieves user engagement analytics for a date range
	GetEngagementAnalytics(ctx context.Context, startDate, endDate string) (*EngagementAnalytics, error)
}

// SalesAnalyticsRequest contains parameters for sales analytics
type SalesAnalyticsRequest struct {
	Period    string    `json:"period"` // daily, weekly, monthly
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// DailySales represents daily sales data
type DailySales struct {
	Date   string `json:"date"`
	Sales  int64  `json:"sales"`
	Orders int    `json:"orders"`
}

// DailyStats represents daily statistics snapshot
type DailyStats struct {
	ID         string    `json:"id"`
	Date       string    `json:"date"`
	Orders     int       `json:"orders"`
	Revenue    int64     `json:"revenue"`
	Customers  int       `json:"customers"`
	Products   int       `json:"products"`
	CreatedAt  time.Time `json:"created_at"`
}
