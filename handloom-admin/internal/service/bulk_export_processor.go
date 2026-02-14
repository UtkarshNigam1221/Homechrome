// Package service implements the business logic layer
package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/handloom/admin/internal/domain"
)

// ExportProcessor handles bulk export operations
type ExportProcessor struct {
	productService *ProductService
	orderService   *OrderService
	exportDir      string
}

// NewExportProcessor creates a new ExportProcessor
func NewExportProcessor(productService *ProductService, orderService *OrderService) *ExportProcessor {
	// Create exports directory if it doesn't exist
	exportDir := "./exports"
	_ = os.MkdirAll(exportDir, 0750)

	return &ExportProcessor{
		productService: productService,
		orderService:   orderService,
		exportDir:      exportDir,
	}
}

// ProcessExport processes an export operation and returns the output file path
func (p *ExportProcessor) ProcessExport(ctx context.Context, operation *domain.BulkOperation) (*domain.BulkOperationResult, error) {
	format := "CSV"
	if f, ok := operation.Metadata["format"].(string); ok && f != "" {
		format = f
	}

	var filters map[string]interface{}
	if f, ok := operation.Metadata["filters"].(map[string]interface{}); ok {
		filters = f
	}

	switch operation.EntityType {
	case domain.BulkOperationEntityProduct:
		return p.exportProducts(ctx, operation.ID, format, filters)
	case domain.BulkOperationEntityOrder:
		return p.exportOrders(ctx, operation.ID, format, filters)
	default:
		return nil, fmt.Errorf("unsupported entity type for export: %s", operation.EntityType)
	}
}

// exportProducts exports products to CSV or JSON
func (p *ExportProcessor) exportProducts(ctx context.Context, operationID, format string, filters map[string]interface{}) (*domain.BulkOperationResult, error) {
	// Build request from filters
	req := domain.ListProductsRequest{
		PaginationRequest: domain.PaginationRequest{
			Limit: 100, // TODO: implement full cursor-based iteration for bulk export
		},
	}

	if filters != nil {
		if status, ok := filters["status"].(string); ok && status != "" {
			s := domain.ProductStatus(status)
			req.Status = &s
		}
	}

	// Fetch all products
	resp, err := p.productService.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}

	totalRecords := len(resp.Products)
	if totalRecords == 0 {
		return &domain.BulkOperationResult{
			TotalRecords:  0,
			SuccessCount:  0,
			FailureCount:  0,
			OutputFileURL: "",
		}, nil
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	var filename string
	var outputData []byte

	if strings.EqualFold(format, "JSON") {
		filename = fmt.Sprintf("products_export_%s_%s.json", operationID, timestamp)
		outputData, err = p.productsToJSON(resp.Products)
	} else {
		filename = fmt.Sprintf("products_export_%s_%s.csv", operationID, timestamp)
		outputData, err = p.productsToCSV(resp.Products)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to convert products to %s: %w", format, err)
	}

	// Write to file
	filePath := filepath.Join(p.exportDir, filename)
	if err := os.WriteFile(filePath, outputData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write export file: %w", err)
	}

	// Return result with file URL (for local dev, use file path; in production, would upload to S3)
	return &domain.BulkOperationResult{
		TotalRecords:  totalRecords,
		SuccessCount:  totalRecords,
		FailureCount:  0,
		OutputFileURL: "/exports/" + filename,
	}, nil
}

// productsToCSV converts products to CSV format
func (p *ExportProcessor) productsToCSV(products []*domain.Product) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"id", "sku", "name", "description", "category_id",
		"base_price", "selling_price", "cost_price", "currency",
		"quantity", "available_qty", "reserved_qty", "low_stock_threshold",
		"material", "color", "weave_type", "origin", "craft_type",
		"weight", "status", "tags", "created_at", "updated_at",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, product := range products {
		tags := ""
		if len(product.Tags) > 0 {
			tags = strings.Join(product.Tags, ";")
		}

		row := []string{
			product.ID,
			product.SKU,
			product.Name,
			product.Description,
			product.CategoryID,
			strconv.FormatInt(product.BasePrice, 10),
			strconv.FormatInt(product.SellingPrice, 10),
			strconv.FormatInt(product.CostPrice, 10),
			product.Currency,
			strconv.Itoa(product.Quantity),
			strconv.Itoa(product.AvailableQty),
			strconv.Itoa(product.ReservedQty),
			strconv.Itoa(product.LowStockThreshold),
			product.Material,
			product.Color,
			product.WeaveType,
			product.Origin,
			product.CraftType,
			strconv.Itoa(product.Weight),
			string(product.Status),
			tags,
			product.CreatedAt.Format(time.RFC3339),
			product.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// productsToJSON converts products to JSON format
func (p *ExportProcessor) productsToJSON(products []*domain.Product) ([]byte, error) {
	return json.MarshalIndent(products, "", "  ")
}

// exportOrders exports orders to CSV or JSON
func (p *ExportProcessor) exportOrders(ctx context.Context, operationID, format string, filters map[string]interface{}) (*domain.BulkOperationResult, error) {
	// Check if order service is available
	if p.orderService == nil {
		return nil, fmt.Errorf("order export not available: order service not initialized")
	}

	// Build request from filters
	req := domain.ListOrdersRequest{
		PaginationRequest: domain.PaginationRequest{
			Limit: 100, // TODO: implement full cursor-based iteration for bulk export
		},
	}

	if filters != nil {
		if status, ok := filters["status"].(string); ok && status != "" {
			s := domain.OrderStatus(status)
			req.Status = &s
		}
	}

	// Fetch all orders
	resp, err := p.orderService.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}

	totalRecords := len(resp.Orders)
	if totalRecords == 0 {
		return &domain.BulkOperationResult{
			TotalRecords:  0,
			SuccessCount:  0,
			FailureCount:  0,
			OutputFileURL: "",
		}, nil
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	var filename string
	var outputData []byte

	if strings.EqualFold(format, "JSON") {
		filename = fmt.Sprintf("orders_export_%s_%s.json", operationID, timestamp)
		outputData, err = p.ordersToJSON(resp.Orders)
	} else {
		filename = fmt.Sprintf("orders_export_%s_%s.csv", operationID, timestamp)
		outputData, err = p.ordersToCSV(resp.Orders)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to convert orders to %s: %w", format, err)
	}

	// Write to file
	filePath := filepath.Join(p.exportDir, filename)
	if err := os.WriteFile(filePath, outputData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write export file: %w", err)
	}

	return &domain.BulkOperationResult{
		TotalRecords:  totalRecords,
		SuccessCount:  totalRecords,
		FailureCount:  0,
		OutputFileURL: "/exports/" + filename,
	}, nil
}

// ordersToCSV converts orders to CSV format
func (p *ExportProcessor) ordersToCSV(orders []*domain.Order) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"id", "order_number", "customer_id", "customer_name", "customer_email", "customer_phone",
		"item_count", "subtotal", "discount_amount", "tax_amount", "shipping_amount", "total_amount", "currency",
		"status", "payment_status", "payment_method",
		"shipping_address", "tracking_number", "shipping_carrier",
		"created_at", "updated_at",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, order := range orders {
		shippingAddr := ""
		if order.ShippingAddress != nil {
			shippingAddr = fmt.Sprintf("%s, %s, %s, %s %s",
				order.ShippingAddress.AddressLine1,
				order.ShippingAddress.City,
				order.ShippingAddress.State,
				order.ShippingAddress.Country,
				order.ShippingAddress.PostalCode)
		}

		row := []string{
			order.ID,
			order.OrderNumber,
			order.CustomerID,
			order.CustomerName,
			order.CustomerEmail,
			order.CustomerPhone,
			strconv.Itoa(order.ItemCount),
			strconv.FormatInt(order.Subtotal, 10),
			strconv.FormatInt(order.DiscountAmount, 10),
			strconv.FormatInt(order.TaxAmount, 10),
			strconv.FormatInt(order.ShippingAmount, 10),
			strconv.FormatInt(order.TotalAmount, 10),
			order.Currency,
			string(order.Status),
			string(order.PaymentStatus),
			order.PaymentMethod,
			shippingAddr,
			order.TrackingNumber,
			order.ShippingCarrier,
			order.CreatedAt.Format(time.RFC3339),
			order.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ordersToJSON converts orders to JSON format
func (p *ExportProcessor) ordersToJSON(orders []*domain.Order) ([]byte, error) {
	return json.MarshalIndent(orders, "", "  ")
}
