// Package domain contains the business entities and interfaces
package domain

import (
	"context"
	"time"
)

// BulkOperationType represents the type of bulk operation
type BulkOperationType string

const (
	BulkOperationTypeImport BulkOperationType = "IMPORT"
	BulkOperationTypeExport BulkOperationType = "EXPORT"
	BulkOperationTypeUpdate BulkOperationType = "UPDATE"
	BulkOperationTypeDelete BulkOperationType = "DELETE"
)

// BulkOperationStatus represents the status of a bulk operation
type BulkOperationStatus string

const (
	BulkOperationStatusPending    BulkOperationStatus = "PENDING"
	BulkOperationStatusProcessing BulkOperationStatus = "PROCESSING"
	BulkOperationStatusCompleted  BulkOperationStatus = "COMPLETED"
	BulkOperationStatusFailed     BulkOperationStatus = "FAILED"
	BulkOperationStatusCancelled  BulkOperationStatus = "CANCELLED"
)

// BulkOperationEntityType represents the entity type for bulk operations
type BulkOperationEntityType string

const (
	BulkOperationEntityProduct   BulkOperationEntityType = "PRODUCT"
	BulkOperationEntityInventory BulkOperationEntityType = "INVENTORY"
	BulkOperationEntityCategory  BulkOperationEntityType = "CATEGORY"
	BulkOperationEntityOrder     BulkOperationEntityType = "ORDER"
	BulkOperationEntityCustomer  BulkOperationEntityType = "CUSTOMER"
	BulkOperationEntityArtisan   BulkOperationEntityType = "ARTISAN"
)

// BulkOperation represents a bulk operation job
type BulkOperation struct {
	ID              string                  `json:"id"`
	Type            BulkOperationType       `json:"type"`
	EntityType      BulkOperationEntityType `json:"entity_type"`
	Status          BulkOperationStatus     `json:"status"`
	TotalRecords    int                     `json:"total_records"`
	ProcessedCount  int                     `json:"processed_count"`
	SuccessCount    int                     `json:"success_count"`
	FailureCount    int                     `json:"failure_count"`
	Errors          []BulkOperationError    `json:"errors,omitempty"`
	InputFileURL    string                  `json:"input_file_url,omitempty"`
	OutputFileURL   string                  `json:"output_file_url,omitempty"`
	ErrorFileURL    string                  `json:"error_file_url,omitempty"`
	Metadata        map[string]interface{}  `json:"metadata,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	StartedAt       *time.Time              `json:"started_at,omitempty"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	CreatedBy       string                  `json:"created_by"`
}

// BulkOperationError represents an error in a bulk operation
type BulkOperationError struct {
	RowNumber int    `json:"row_number"`
	Field     string `json:"field,omitempty"`
	Value     string `json:"value,omitempty"`
	Message   string `json:"message"`
}

// BulkProductImportRow represents a row in product import CSV
type BulkProductImportRow struct {
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	CategoryID   string  `json:"category_id"`
	BasePrice    float64 `json:"base_price"`
	SellingPrice float64 `json:"selling_price"`
	Quantity     int     `json:"quantity"`
	LowStockThreshold int `json:"low_stock_threshold,omitempty"`
	Weight       float64 `json:"weight,omitempty"`
	Dimensions   string  `json:"dimensions,omitempty"`
	Tags         string  `json:"tags,omitempty"`
	Status       string  `json:"status,omitempty"`
}

// BulkInventoryUpdateRow represents a row in inventory update CSV
type BulkInventoryUpdateRow struct {
	ProductID    string `json:"product_id"`
	SKU          string `json:"sku,omitempty"`
	Quantity     int    `json:"quantity"`
	Operation    string `json:"operation"` // SET, ADD, SUBTRACT
	Reason       string `json:"reason,omitempty"`
}

// BulkPriceUpdateRow represents a row in price update CSV
type BulkPriceUpdateRow struct {
	ProductID    string  `json:"product_id"`
	SKU          string  `json:"sku,omitempty"`
	BasePrice    float64 `json:"base_price,omitempty"`
	SellingPrice float64 `json:"selling_price,omitempty"`
	ComparePrice float64 `json:"compare_price,omitempty"`
}

// BulkOperationRepository defines bulk operation data access methods
type BulkOperationRepository interface {
	Create(ctx context.Context, operation *BulkOperation) error
	GetByID(ctx context.Context, id string) (*BulkOperation, error)
	Update(ctx context.Context, operation *BulkOperation) error
	List(ctx context.Context, req ListBulkOperationsRequest) (*ListBulkOperationsResponse, error)
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListBulkOperationsResponse, error)
}

// CreateBulkOperationRequest represents a request to create a bulk operation
type CreateBulkOperationRequest struct {
	Type         BulkOperationType       `json:"type" validate:"required"`
	EntityType   BulkOperationEntityType `json:"entity_type" validate:"required"`
	InputFileURL string                  `json:"input_file_url,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
}

// ListBulkOperationsRequest represents a request to list bulk operations
type ListBulkOperationsRequest struct {
	Type       BulkOperationType       `json:"type,omitempty"`
	EntityType BulkOperationEntityType `json:"entity_type,omitempty"`
	Status     BulkOperationStatus     `json:"status,omitempty"`
	StartDate  time.Time               `json:"start_date,omitempty"`
	EndDate    time.Time               `json:"end_date,omitempty"`
	Pagination PaginationRequest       `json:"pagination"`
}

// ListBulkOperationsResponse represents the response for listing bulk operations
type ListBulkOperationsResponse struct {
	Operations []*BulkOperation   `json:"operations"`
	Pagination PaginationResponse `json:"pagination"`
}

// BulkProductImportRequest represents a request to import products in bulk
type BulkProductImportRequest struct {
	FileURL  string                 `json:"file_url" validate:"required"`
	DryRun   bool                   `json:"dry_run"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BulkInventoryUpdateRequest represents a request to update inventory in bulk
type BulkInventoryUpdateRequest struct {
	FileURL  string                 `json:"file_url" validate:"required"`
	DryRun   bool                   `json:"dry_run"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BulkPriceUpdateRequest represents a request to update prices in bulk
type BulkPriceUpdateRequest struct {
	FileURL  string                 `json:"file_url" validate:"required"`
	DryRun   bool                   `json:"dry_run"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BulkExportRequest represents a request to export data in bulk
type BulkExportRequest struct {
	EntityType BulkOperationEntityType `json:"entity_type" validate:"required"`
	Filters    map[string]interface{}  `json:"filters,omitempty"`
	Fields     []string                `json:"fields,omitempty"`
	Format     string                  `json:"format,omitempty"` // CSV, JSON
}

// BulkOperationResult represents the result of a bulk operation
type BulkOperationResult struct {
	OperationID    string              `json:"operation_id"`
	Status         BulkOperationStatus `json:"status"`
	TotalRecords   int                 `json:"total_records"`
	SuccessCount   int                 `json:"success_count"`
	FailureCount   int                 `json:"failure_count"`
	OutputFileURL  string              `json:"output_file_url,omitempty"`
	ErrorFileURL   string              `json:"error_file_url,omitempty"`
}
