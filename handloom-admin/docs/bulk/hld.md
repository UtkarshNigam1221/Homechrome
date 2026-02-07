# Bulk Lambda - High Level Design

## 1. Overview

The Bulk Lambda provides bulk import and export capabilities for the Handloom Admin platform. It handles large-scale data operations for products, inventory, pricing, and other entities with validation, progress tracking, and error handling.

### Key Features
- CSV/XLSX file import with validation
- Configurable export with filtering
- Async processing for large datasets
- Progress tracking and status updates
- Error reporting and recovery
- Template management

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          BULK LAMBDA ARCHITECTURE                            │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Client     │
                              │  (Browser)   │
                              └──────┬───────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Gateway                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  /bulk/import           POST    - Start import job                   │    │
│  │  /bulk/export           POST    - Start export job                   │    │
│  │  /bulk/jobs             GET     - List jobs                          │    │
│  │  /bulk/jobs/{id}        GET     - Get job status                     │    │
│  │  /bulk/jobs/{id}/cancel POST    - Cancel job                         │    │
│  │  /bulk/templates/{type} GET     - Download template                  │    │
│  │  /bulk/inventory        POST    - Bulk update inventory              │    │
│  │  /bulk/prices           POST    - Bulk update prices                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                             Bulk Lambda                                      │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Import     │ │   Export     │ │  Job Status  │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │  Template    │ │  Inventory   │ │    Price     │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                      Bulk Service                             │  │     │
│  │  │  - StartImport()          - StartExport()                    │  │     │
│  │  │  - GetJobStatus()         - CancelJob()                      │  │     │
│  │  │  - ProcessImport()        - ProcessExport()                  │  │     │
│  │  │  - ValidateFile()         - GenerateExport()                 │  │     │
│  │  │  - BulkUpdateInventory()  - BulkUpdatePrices()               │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  │                                                                     │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                    Processor Layer                            │  │     │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐                │  │     │
│  │  │  │  Product   │ │ Inventory  │ │   Price    │                │  │     │
│  │  │  │ Processor  │ │ Processor  │ │ Processor  │                │  │     │
│  │  │  └────────────┘ └────────────┘ └────────────┘                │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                    Bulk Repository                            │  │     │
│  │  │  - CreateJob()      - GetJob()        - UpdateJob()          │  │     │
│  │  │  - ListJobs()       - GetJobProgress()                       │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                     ┌───────────────┼───────────────┐
                     │               │               │
                     ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ DynamoDB │    │    S3    │    │   SQS    │
              │  (Jobs)  │    │ (Files)  │    │ (Queue)  │
              └──────────┘    └──────────┘    └──────────┘
```

---

## 3. Component Design

### 3.1 Bulk Handler

```go
type BulkHandler struct {
    bulkService domain.BulkService
    logger      *logger.Logger
}

// Handler Methods
- StartImport(c *gin.Context)
- StartExport(c *gin.Context)
- GetJobStatus(c *gin.Context)
- ListJobs(c *gin.Context)
- CancelJob(c *gin.Context)
- GetTemplate(c *gin.Context)
- BulkUpdateInventory(c *gin.Context)
- BulkUpdatePrices(c *gin.Context)
```

### 3.2 Bulk Service

```go
type BulkService interface {
    // Import Operations
    StartImport(ctx context.Context, req *ImportRequest) (*BulkJob, error)
    ProcessImport(ctx context.Context, jobID string) error
    ValidateFile(ctx context.Context, fileURL string, entityType string) (*ValidationResult, error)

    // Export Operations
    StartExport(ctx context.Context, req *ExportRequest) (*BulkJob, error)
    ProcessExport(ctx context.Context, jobID string) error
    GenerateExport(ctx context.Context, entityType string, filter *ExportFilter) (string, error)

    // Job Management
    GetJobStatus(ctx context.Context, jobID string) (*BulkJob, error)
    ListJobs(ctx context.Context, filter *JobFilter) (*JobList, error)
    CancelJob(ctx context.Context, jobID string) error

    // Bulk Updates
    BulkUpdateInventory(ctx context.Context, updates []*InventoryUpdate) (*BulkResult, error)
    BulkUpdatePrices(ctx context.Context, updates []*PriceUpdate) (*BulkResult, error)
}
```

### 3.3 Bulk Repository

```go
type BulkRepository interface {
    CreateJob(ctx context.Context, job *BulkJob) error
    GetJob(ctx context.Context, id string) (*BulkJob, error)
    UpdateJob(ctx context.Context, job *BulkJob) error
    ListJobs(ctx context.Context, filter *JobFilter) ([]*BulkJob, error)
    UpdateJobProgress(ctx context.Context, jobID string, progress *Progress) error
}
```

---

## 4. Data Model

### 4.1 Bulk Job Entity

```go
type BulkJob struct {
    ID            string          `json:"id" dynamodbav:"id"`
    Type          JobType         `json:"type" dynamodbav:"type"`
    EntityType    string          `json:"entity_type" dynamodbav:"entity_type"`
    Status        JobStatus       `json:"status" dynamodbav:"status"`
    FileURL       string          `json:"file_url,omitempty" dynamodbav:"file_url,omitempty"`
    ResultURL     string          `json:"result_url,omitempty" dynamodbav:"result_url,omitempty"`
    TotalRecords  int             `json:"total_records" dynamodbav:"total_records"`
    ProcessedRecords int          `json:"processed_records" dynamodbav:"processed_records"`
    SuccessCount  int             `json:"success_count" dynamodbav:"success_count"`
    FailedCount   int             `json:"failed_count" dynamodbav:"failed_count"`
    Errors        []JobError      `json:"errors,omitempty" dynamodbav:"errors,omitempty"`
    Config        *JobConfig      `json:"config,omitempty" dynamodbav:"config,omitempty"`
    StartedAt     *time.Time      `json:"started_at,omitempty" dynamodbav:"started_at,omitempty"`
    CompletedAt   *time.Time      `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
    CreatedAt     time.Time       `json:"created_at" dynamodbav:"created_at"`
    CreatedBy     string          `json:"created_by" dynamodbav:"created_by"`
}
```

### 4.2 Job Types and Status

```go
type JobType string

const (
    JobTypeImport JobType = "IMPORT"
    JobTypeExport JobType = "EXPORT"
)

type JobStatus string

const (
    JobStatusPending    JobStatus = "PENDING"
    JobStatusValidating JobStatus = "VALIDATING"
    JobStatusProcessing JobStatus = "PROCESSING"
    JobStatusCompleted  JobStatus = "COMPLETED"
    JobStatusPartial    JobStatus = "PARTIAL"
    JobStatusFailed     JobStatus = "FAILED"
    JobStatusCancelled  JobStatus = "CANCELLED"
)
```

### 4.3 Job Error

```go
type JobError struct {
    Row      int    `json:"row" dynamodbav:"row"`
    Field    string `json:"field" dynamodbav:"field"`
    Value    string `json:"value" dynamodbav:"value"`
    Message  string `json:"message" dynamodbav:"message"`
    Severity string `json:"severity" dynamodbav:"severity"` // ERROR, WARNING
}
```

### 4.4 Import/Export Request

```go
type ImportRequest struct {
    EntityType   string            `json:"entity_type" binding:"required"`
    FileURL      string            `json:"file_url" binding:"required"`
    SkipErrors   bool              `json:"skip_errors"`
    UpdateExisting bool            `json:"update_existing"`
    Mappings     map[string]string `json:"mappings,omitempty"`
}

type ExportRequest struct {
    EntityType string            `json:"entity_type" binding:"required"`
    Format     string            `json:"format"` // csv, xlsx, json
    Columns    []string          `json:"columns,omitempty"`
    Filters    map[string]string `json:"filters,omitempty"`
}
```

### 4.5 Validation Result

```go
type ValidationResult struct {
    Valid       bool        `json:"valid"`
    TotalRows   int         `json:"total_rows"`
    ValidRows   int         `json:"valid_rows"`
    Errors      []JobError  `json:"errors,omitempty"`
    Warnings    []JobError  `json:"warnings,omitempty"`
    SampleData  []map[string]interface{} `json:"sample_data,omitempty"`
}
```

---

## 5. DynamoDB Schema

### 5.1 Bulk Jobs Table

```
Table: handloom-bulk-jobs

Primary Key:
- PK: JOB#<job_id>
- SK: JOB#<job_id>

Attributes:
- id: string
- type: string (IMPORT, EXPORT)
- entity_type: string (products, inventory, prices, orders)
- status: string
- file_url: string
- result_url: string
- total_records: number
- processed_records: number
- success_count: number
- failed_count: number
- errors: list
- config: map
- started_at: string
- completed_at: string
- created_at: string
- created_by: string

GSI1: type-status-index
- PK: type
- SK: status#created_at

GSI2: user-jobs-index
- PK: created_by
- SK: created_at
```

### 5.2 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get job by ID | PK = JOB#{id} | Main |
| List jobs by type | PK = {type} | GSI1 |
| List jobs by user | PK = {user_id} | GSI2 |
| Get pending jobs | PK = IMPORT, SK begins_with PENDING | GSI1 |

---

## 6. API Endpoints

### 6.1 Start Import

```
POST /bulk/import

Request:
{
    "entity_type": "products",
    "file_url": "s3://bucket/uploads/products.csv",
    "skip_errors": true,
    "update_existing": false
}

Response:
{
    "success": true,
    "data": {
        "job_id": "job_123",
        "status": "VALIDATING",
        "validation": {
            "total_rows": 500,
            "valid_rows": 485,
            "errors": [
                {"row": 12, "field": "sku", "message": "Duplicate SKU"}
            ]
        }
    }
}
```

### 6.2 Start Export

```
POST /bulk/export

Request:
{
    "entity_type": "products",
    "format": "csv",
    "columns": ["id", "name", "sku", "price", "stock"],
    "filters": {
        "status": "ACTIVE",
        "category_id": "cat_123"
    }
}

Response:
{
    "success": true,
    "data": {
        "job_id": "job_456",
        "status": "PROCESSING",
        "estimated_records": 1234
    }
}
```

### 6.3 Get Job Status

```
GET /bulk/jobs/{job_id}

Response:
{
    "success": true,
    "data": {
        "id": "job_123",
        "type": "IMPORT",
        "entity_type": "products",
        "status": "COMPLETED",
        "total_records": 500,
        "processed_records": 500,
        "success_count": 485,
        "failed_count": 15,
        "result_url": "s3://bucket/results/job_123_errors.csv",
        "started_at": "2024-01-20T10:00:00Z",
        "completed_at": "2024-01-20T10:05:30Z"
    }
}
```

### 6.4 Bulk Update Inventory

```
POST /bulk/inventory

Request:
{
    "updates": [
        {"sku": "SKU-001", "quantity_change": 25, "reason": "PURCHASE_ORDER"},
        {"sku": "SKU-002", "quantity_change": -5, "reason": "DAMAGED"}
    ]
}

Response:
{
    "success": true,
    "data": {
        "total": 2,
        "updated": 2,
        "failed": 0,
        "results": [
            {"sku": "SKU-001", "previous": 50, "new": 75, "status": "SUCCESS"},
            {"sku": "SKU-002", "previous": 20, "new": 15, "status": "SUCCESS"}
        ]
    }
}
```

---

## 7. Processing Flow

### 7.1 Import Processing

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          IMPORT PROCESSING FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

  Upload File        Validate           Queue Job          Process
  ┌──────────┐      ┌──────────┐      ┌──────────┐      ┌──────────┐
  │  Upload  │─────>│ Parse &  │─────>│  Create  │─────>│ Process  │
  │  to S3   │      │ Validate │      │   Job    │      │  Async   │
  └──────────┘      └──────────┘      └──────────┘      └──────────┘
                          │                                   │
                          ▼                                   ▼
                    ┌──────────┐                       ┌──────────┐
                    │ Return   │                       │ Update   │
                    │ Errors   │                       │ Progress │
                    └──────────┘                       └──────────┘
                                                             │
                                                             ▼
                                                      ┌──────────┐
                                                      │ Complete │
                                                      │  /Fail   │
                                                      └──────────┘
```

### 7.2 Batch Processing

```go
const (
    BatchSize = 25  // DynamoDB batch write limit
    MaxRetries = 3
)

func (s *bulkService) ProcessImport(ctx context.Context, jobID string) error {
    // Get job details
    job, _ := s.repo.GetJob(ctx, jobID)

    // Download and parse file
    records := parseFile(job.FileURL)

    // Process in batches
    for i := 0; i < len(records); i += BatchSize {
        batch := records[i:min(i+BatchSize, len(records))]

        // Process batch
        results := processBatch(batch)

        // Update progress
        s.repo.UpdateJobProgress(ctx, jobID, &Progress{
            Processed: i + len(batch),
            Success:   results.Success,
            Failed:    results.Failed,
        })
    }

    return nil
}
```

---

## 8. Error Handling

### 8.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_FILE_FORMAT | Unsupported file format | 400 |
| FILE_TOO_LARGE | File exceeds size limit | 400 |
| VALIDATION_FAILED | File validation errors | 400 |
| JOB_NOT_FOUND | Job does not exist | 404 |
| JOB_NOT_CANCELLABLE | Job cannot be cancelled | 400 |
| PROCESSING_ERROR | Error during processing | 500 |

### 8.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "VALIDATION_FAILED",
        "message": "File validation failed with 15 errors",
        "details": {
            "total_errors": 15,
            "errors": [
                {"row": 12, "field": "sku", "message": "Duplicate SKU"}
            ]
        }
    }
}
```

---

## 9. File Specifications

### 9.1 Product Import Template

| Column | Required | Description |
|--------|----------|-------------|
| name | Yes | Product name |
| sku | Yes | Unique SKU |
| category_id | Yes | Category ID |
| base_price | Yes | Base price |
| stock_quantity | Yes | Initial stock |
| description | No | Product description |
| artisan_id | No | Associated artisan |
| weight | No | Weight in grams |
| images | No | Comma-separated URLs |

### 9.2 Inventory Update Template

| Column | Required | Description |
|--------|----------|-------------|
| sku | Yes | Product SKU |
| quantity_change | Yes | +/- quantity |
| reason | Yes | Update reason |
| reference | No | Reference ID |

---

## 10. Performance Optimization

### 10.1 Processing Strategies

- Batch writes (25 items per batch)
- Parallel processing for large files
- Async processing via SQS
- Chunked file reading

### 10.2 Resource Limits

| Resource | Limit |
|----------|-------|
| Max file size | 10 MB |
| Max rows per import | 10,000 |
| Max concurrent jobs | 5 per user |
| Job retention | 30 days |
| Export URL validity | 24 hours |

---

## 11. Monitoring

### 11.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Processing Time | Time per 1000 records | < 60s |
| Success Rate | % of successful imports | > 95% |
| Queue Depth | Pending jobs | < 100 |
| Error Rate | Processing errors | < 5% |

### 11.2 Alerts

- Processing time exceeded
- High error rate
- Queue backlog
- Job stuck in processing

---

## 12. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                           Bulk Lambda
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
           ▼                   ▼                   ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │  DynamoDB   │    │     S3      │    │    SQS      │
    │   (Jobs)    │    │  (Files)    │    │  (Queue)    │
    └─────────────┘    └─────────────┘    └─────────────┘
           │
           ▼
    ┌─────────────┐
    │ Entity Svcs │
    │(Product/Inv)│
    └─────────────┘
```

### Internal Dependencies
- Product Service: Product import/export
- Inventory Service: Stock updates
- Pricing Service: Price updates

### External Dependencies
- AWS DynamoDB: Job storage
- AWS S3: File storage
- AWS SQS: Job queue
- AWS CloudWatch: Logging

