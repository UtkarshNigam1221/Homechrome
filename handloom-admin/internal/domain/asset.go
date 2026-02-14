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

// ==================== BULK JOB ENTITY ====================

// BulkJobType defines the type of bulk job
type BulkJobType string

const (
	BulkJobTypeProductImport   BulkJobType = "PRODUCT_IMPORT"
	BulkJobTypeProductExport   BulkJobType = "PRODUCT_EXPORT"
	BulkJobTypeInventoryUpdate BulkJobType = "INVENTORY_UPDATE"
	BulkJobTypePriceUpdate     BulkJobType = "PRICE_UPDATE"
	BulkJobTypeOrderExport     BulkJobType = "ORDER_EXPORT"
)

// BulkJobStatus defines the status of a bulk job
type BulkJobStatus string

const (
	BulkJobStatusPending    BulkJobStatus = "PENDING"
	BulkJobStatusProcessing BulkJobStatus = "PROCESSING"
	BulkJobStatusCompleted  BulkJobStatus = "COMPLETED"
	BulkJobStatusFailed     BulkJobStatus = "FAILED"
	BulkJobStatusPartial    BulkJobStatus = "PARTIAL_SUCCESS"
)

// BulkJob represents a bulk operation job
type BulkJob struct {
	ID             string        `json:"id" dynamodbav:"id"`
	PK             string        `json:"-" dynamodbav:"PK"`
	SK             string        `json:"-" dynamodbav:"SK"`
	GSI1PK         string        `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK         string        `json:"-" dynamodbav:"GSI1SK"`
	EntityType     string        `json:"-" dynamodbav:"entity_type"`

	Type           BulkJobType   `json:"type" dynamodbav:"type"`
	Status         BulkJobStatus `json:"status" dynamodbav:"status"`

	// File Info
	FileName       string        `json:"file_name" dynamodbav:"file_name"`
	FileURL        string        `json:"file_url" dynamodbav:"file_url"`
	FileSize       int64         `json:"file_size" dynamodbav:"file_size"`

	// Progress
	TotalRows      int           `json:"total_rows" dynamodbav:"total_rows"`
	ProcessedRows  int           `json:"processed_rows" dynamodbav:"processed_rows"`
	SuccessCount   int           `json:"success_count" dynamodbav:"success_count"`
	ErrorCount     int           `json:"error_count" dynamodbav:"error_count"`

	// Errors
	Errors         []BulkJobError `json:"errors,omitempty" dynamodbav:"errors,omitempty"`

	// Result
	ResultFileURL  string        `json:"result_file_url,omitempty" dynamodbav:"result_file_url,omitempty"`

	StartedAt      *time.Time    `json:"started_at,omitempty" dynamodbav:"started_at,omitempty"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`

	BaseEntity
}

// TableName returns the DynamoDB table name for BulkJob
func (b *BulkJob) TableName() string {
	return "handloom-core"
}

// SetKeys sets the DynamoDB keys for BulkJob
func (b *BulkJob) SetKeys() {
	b.PK = "BULK_JOB#" + b.ID
	b.SK = "METADATA"
	b.GSI1PK = "USER#" + b.CreatedBy
	b.GSI1SK = "BULK_JOB#" + b.ID
	b.EntityType = "BULK_JOB"
}

// BulkJobError represents an error in a bulk job
type BulkJobError struct {
	Row     int    `json:"row" dynamodbav:"row"`
	Field   string `json:"field" dynamodbav:"field"`
	Message string `json:"message" dynamodbav:"message"`
	Value   string `json:"value,omitempty" dynamodbav:"value,omitempty"`
}

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


// BulkJobRepository defines bulk job data access methods
type BulkJobRepository interface {
	Create(ctx context.Context, job *BulkJob) error
	GetByID(ctx context.Context, id string) (*BulkJob, error)
	Update(ctx context.Context, job *BulkJob) error
	List(ctx context.Context, req ListBulkJobsRequest) (*ListBulkJobsResponse, error)
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListBulkJobsResponse, error)
}

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

// ListBulkJobsRequest represents a request to list bulk jobs
type ListBulkJobsRequest struct {
	Type       BulkJobType       `json:"type,omitempty"`
	Status     BulkJobStatus     `json:"status,omitempty"`
	StartDate  time.Time         `json:"start_date,omitempty"`
	EndDate    time.Time         `json:"end_date,omitempty"`
	Pagination PaginationRequest `json:"pagination"`
}

// ListBulkJobsResponse represents the response for listing bulk jobs
type ListBulkJobsResponse struct {
	Jobs       []*BulkJob         `json:"jobs"`
	Pagination PaginationResponse `json:"pagination"`
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
