package domain

import (
	"context"
	"time"
)

// ==================== ASSET TYPES ====================
// Assets use a tmp/ → assets/ S3 flow with no DynamoDB records.
// Files are uploaded to tmp/{TYPE}/{uuid}.ext via presigned PUT URL,
// then moved to assets/{TYPE}/{date}/{uuid}.ext on finalize.
// S3 lifecycle auto-deletes tmp/ objects after 24h.

// AssetType defines the type of asset
type AssetType string

const (
	AssetTypeImage    AssetType = "IMAGE"
	AssetTypeDocument AssetType = "DOCUMENT"
	AssetTypeVideo    AssetType = "VIDEO"
)

// ==================== REPORT ENTITY ====================

// ReportType defines the type of report
type ReportType string

const (
	ReportTypeSales     ReportType = "SALES"
	ReportTypeOrders    ReportType = "ORDERS"
	ReportTypeInventory ReportType = "INVENTORY"
	ReportTypeCustomers ReportType = "CUSTOMERS"
	ReportTypeProducts  ReportType = "PRODUCTS"
	ReportTypeArtisans  ReportType = "ARTISANS"
)

// ReportFormat defines the format of report
type ReportFormat string

const (
	ReportFormatCSV  ReportFormat = "CSV"
	ReportFormatXLSX ReportFormat = "XLSX"
	ReportFormatPDF  ReportFormat = "PDF"
)

// ReportStatus defines the status of a report
type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "PENDING"
	ReportStatusProcessing ReportStatus = "PROCESSING"
	ReportStatusCompleted  ReportStatus = "COMPLETED"
	ReportStatusFailed     ReportStatus = "FAILED"
)

// Report represents a generated report
type Report struct {
	ID           string                 `json:"id" dynamodbav:"id"`
	PK           string                 `json:"-" dynamodbav:"PK"`
	SK           string                 `json:"-" dynamodbav:"SK"`
	GSI1PK       string                 `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK       string                 `json:"-" dynamodbav:"GSI1SK"`
	EntityType   string                 `json:"-" dynamodbav:"entity_type"`

	Name         string                 `json:"name" dynamodbav:"name"`
	Type         ReportType             `json:"type" dynamodbav:"type"`
	Format       ReportFormat           `json:"format" dynamodbav:"format"`
	Status       ReportStatus           `json:"status" dynamodbav:"status"`

	// Filters/Parameters
	Parameters   map[string]interface{} `json:"parameters,omitempty" dynamodbav:"parameters,omitempty"`
	StartDate    *time.Time             `json:"start_date,omitempty" dynamodbav:"start_date,omitempty"`
	EndDate      *time.Time             `json:"end_date,omitempty" dynamodbav:"end_date,omitempty"`

	// Result
	FileURL      string                 `json:"file_url,omitempty" dynamodbav:"file_url,omitempty"`
	FileSize     int64                  `json:"file_size,omitempty" dynamodbav:"file_size,omitempty"`
	RowCount     int                    `json:"row_count,omitempty" dynamodbav:"row_count,omitempty"`

	// Error
	ErrorMessage string                 `json:"error_message,omitempty" dynamodbav:"error_message,omitempty"`

	StartedAt    *time.Time             `json:"started_at,omitempty" dynamodbav:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Report
func (r *Report) TableName() string {
	return "handloom-core"
}

// SetKeys sets the DynamoDB keys for Report
func (r *Report) SetKeys() {
	r.PK = "REPORT#" + r.ID
	r.SK = "METADATA"
	r.GSI1PK = "USER#" + r.CreatedBy
	r.GSI1SK = "REPORT#" + r.ID
	r.EntityType = "REPORT"
}

// ==================== REPOSITORY INTERFACES ====================


// ReportRepository defines report data access methods
type ReportRepository interface {
	Create(ctx context.Context, report *Report) error
	GetByID(ctx context.Context, id string) (*Report, error)
	Update(ctx context.Context, report *Report) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, req ListReportsRequest) (*ListReportsResponse, error)
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListReportsResponse, error)
}

// ==================== REQUEST/RESPONSE TYPES ====================

// UploadAssetRequest represents a request to get a presigned upload URL
type UploadAssetRequest struct {
	FileName    string    `json:"file_name" validate:"required"`
	ContentType string    `json:"content_type" validate:"required"`
	Size        int64     `json:"size" validate:"required"`
	Type        AssetType `json:"type" validate:"required"`
}

// UploadURLResponse represents a presigned URL response for the tmp/ upload flow
type UploadURLResponse struct {
	UploadURL string    `json:"upload_url"`
	TmpKey    string    `json:"tmp_key"`
	TmpURL    string    `json:"tmp_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DeleteAssetRequest represents a request to delete an asset by its public URL
type DeleteAssetRequest struct {
	URL string `json:"url" validate:"required"`
}

// GenerateReportRequest represents a request to generate a report
type GenerateReportRequest struct {
	Name       string                 `json:"name" validate:"required"`
	Type       ReportType             `json:"type" validate:"required"`
	Format     ReportFormat           `json:"format" validate:"required"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	StartDate  *time.Time             `json:"start_date,omitempty"`
	EndDate    *time.Time             `json:"end_date,omitempty"`
}

// ListReportsRequest represents a request to list reports
type ListReportsRequest struct {
	Type       ReportType        `json:"type,omitempty"`
	Status     ReportStatus      `json:"status,omitempty"`
	StartDate  time.Time         `json:"start_date,omitempty"`
	EndDate    time.Time         `json:"end_date,omitempty"`
	Pagination PaginationRequest `json:"pagination"`
}

// ListReportsResponse represents the response for listing reports
type ListReportsResponse struct {
	Reports    []*Report          `json:"reports"`
	Pagination PaginationResponse `json:"pagination"`
}
