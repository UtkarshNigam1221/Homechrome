# Report Lambda - High Level Design

## 1. Overview

The Report Lambda provides comprehensive reporting capabilities for the Handloom Admin platform. It handles report generation, scheduling, and delivery for sales, inventory, customer, and other business analytics.

### Key Features
- Multiple report templates (Sales, Inventory, Customer, etc.)
- Customizable date ranges and filters
- Scheduled report generation
- Multiple export formats (PDF, Excel, CSV)
- Email delivery for scheduled reports
- Report history and archival

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         REPORT LAMBDA ARCHITECTURE                           │
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
│  │  /reports/generate      POST    - Generate report                    │    │
│  │  /reports               GET     - List reports                       │    │
│  │  /reports/{id}          GET     - Get report details                 │    │
│  │  /reports/{id}/download GET     - Download report                    │    │
│  │  /reports/schedules     GET/POST - Manage schedules                  │    │
│  │  /reports/schedules/{id} PUT/DELETE - Update/delete schedule         │    │
│  │  /reports/templates     GET     - List templates                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Report Lambda                                     │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │  Generate    │ │    List      │ │  Download    │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐                                 │     │
│  │  │  Schedule    │ │  Template    │                                 │     │
│  │  │   Handler    │ │   Handler    │                                 │     │
│  │  └──────────────┘ └──────────────┘                                 │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                     Report Service                            │  │     │
│  │  │  - GenerateReport()       - GetReport()                      │  │     │
│  │  │  - ListReports()          - DeleteReport()                   │  │     │
│  │  │  - CreateSchedule()       - UpdateSchedule()                 │  │     │
│  │  │  - ExecuteSchedule()      - GetDownloadURL()                 │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  │                                                                     │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                   Report Generators                           │  │     │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐                │  │     │
│  │  │  │   Sales    │ │ Inventory  │ │  Customer  │                │  │     │
│  │  │  │ Generator  │ │ Generator  │ │ Generator  │                │  │     │
│  │  │  └────────────┘ └────────────┘ └────────────┘                │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                     ┌───────────────┼───────────────┬───────────────┐
                     │               │               │               │
                     ▼               ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ DynamoDB │    │    S3    │    │   SES    │    │EventBridge│
              │ (Reports)│    │ (Files)  │    │ (Email)  │    │ (Cron)   │
              └──────────┘    └──────────┘    └──────────┘    └──────────┘
```

---

## 3. Component Design

### 3.1 Report Handler

```go
type ReportHandler struct {
    reportService domain.ReportService
    logger        *logger.Logger
}

// Handler Methods
- GenerateReport(c *gin.Context)
- GetReport(c *gin.Context)
- ListReports(c *gin.Context)
- DownloadReport(c *gin.Context)
- CreateSchedule(c *gin.Context)
- UpdateSchedule(c *gin.Context)
- DeleteSchedule(c *gin.Context)
- ListSchedules(c *gin.Context)
- ListTemplates(c *gin.Context)
```

### 3.2 Report Service

```go
type ReportService interface {
    // Report Generation
    GenerateReport(ctx context.Context, req *GenerateReportRequest) (*Report, error)
    GetReport(ctx context.Context, id string) (*Report, error)
    ListReports(ctx context.Context, filter *ReportFilter) (*ReportList, error)
    DeleteReport(ctx context.Context, id string) error
    GetDownloadURL(ctx context.Context, id string, format string) (string, error)

    // Scheduling
    CreateSchedule(ctx context.Context, req *CreateScheduleRequest) (*ReportSchedule, error)
    UpdateSchedule(ctx context.Context, id string, req *UpdateScheduleRequest) (*ReportSchedule, error)
    DeleteSchedule(ctx context.Context, id string) error
    ListSchedules(ctx context.Context) ([]*ReportSchedule, error)
    ExecuteSchedule(ctx context.Context, scheduleID string) error

    // Templates
    ListTemplates(ctx context.Context) ([]*ReportTemplate, error)
}
```

### 3.3 Report Repository

```go
type ReportRepository interface {
    Create(ctx context.Context, report *Report) error
    GetByID(ctx context.Context, id string) (*Report, error)
    List(ctx context.Context, filter *ReportFilter) ([]*Report, error)
    Delete(ctx context.Context, id string) error
    CreateSchedule(ctx context.Context, schedule *ReportSchedule) error
    GetSchedule(ctx context.Context, id string) (*ReportSchedule, error)
    UpdateSchedule(ctx context.Context, schedule *ReportSchedule) error
    DeleteSchedule(ctx context.Context, id string) error
    ListSchedules(ctx context.Context) ([]*ReportSchedule, error)
    GetDueSchedules(ctx context.Context, time time.Time) ([]*ReportSchedule, error)
}
```

---

## 4. Data Model

### 4.1 Report Entity

```go
type Report struct {
    ID           string          `json:"id" dynamodbav:"id"`
    Name         string          `json:"name" dynamodbav:"name"`
    Type         ReportType      `json:"type" dynamodbav:"type"`
    Config       *ReportConfig   `json:"config" dynamodbav:"config"`
    Status       ReportStatus    `json:"status" dynamodbav:"status"`
    FileURL      string          `json:"file_url,omitempty" dynamodbav:"file_url,omitempty"`
    FileSize     int64           `json:"file_size,omitempty" dynamodbav:"file_size,omitempty"`
    Format       string          `json:"format" dynamodbav:"format"`
    GeneratedAt  *time.Time      `json:"generated_at,omitempty" dynamodbav:"generated_at,omitempty"`
    ExpiresAt    *time.Time      `json:"expires_at,omitempty" dynamodbav:"expires_at,omitempty"`
    ScheduleID   string          `json:"schedule_id,omitempty" dynamodbav:"schedule_id,omitempty"`
    CreatedAt    time.Time       `json:"created_at" dynamodbav:"created_at"`
    CreatedBy    string          `json:"created_by" dynamodbav:"created_by"`
}
```

### 4.2 Report Types

```go
type ReportType string

const (
    ReportTypeSales       ReportType = "SALES"
    ReportTypeInventory   ReportType = "INVENTORY"
    ReportTypeCustomer    ReportType = "CUSTOMER"
    ReportTypeArtisan     ReportType = "ARTISAN"
    ReportTypeProduct     ReportType = "PRODUCT"
    ReportTypeFinancial   ReportType = "FINANCIAL"
    ReportTypeCoupon      ReportType = "COUPON"
    ReportTypeCustom      ReportType = "CUSTOM"
)
```

### 4.3 Report Config

```go
type ReportConfig struct {
    DateRange    *DateRange        `json:"date_range" dynamodbav:"date_range"`
    GroupBy      string            `json:"group_by,omitempty" dynamodbav:"group_by,omitempty"`
    Filters      map[string]string `json:"filters,omitempty" dynamodbav:"filters,omitempty"`
    Columns      []string          `json:"columns,omitempty" dynamodbav:"columns,omitempty"`
    IncludeSummary bool            `json:"include_summary" dynamodbav:"include_summary"`
    IncludeCharts  bool            `json:"include_charts" dynamodbav:"include_charts"`
}

type DateRange struct {
    StartDate time.Time `json:"start_date" dynamodbav:"start_date"`
    EndDate   time.Time `json:"end_date" dynamodbav:"end_date"`
}
```

### 4.4 Report Schedule

```go
type ReportSchedule struct {
    ID           string          `json:"id" dynamodbav:"id"`
    Name         string          `json:"name" dynamodbav:"name"`
    Type         ReportType      `json:"type" dynamodbav:"type"`
    Config       *ReportConfig   `json:"config" dynamodbav:"config"`
    Frequency    string          `json:"frequency" dynamodbav:"frequency"` // DAILY, WEEKLY, MONTHLY
    CronExpression string        `json:"cron_expression" dynamodbav:"cron_expression"`
    Timezone     string          `json:"timezone" dynamodbav:"timezone"`
    NextRunAt    time.Time       `json:"next_run_at" dynamodbav:"next_run_at"`
    LastRunAt    *time.Time      `json:"last_run_at,omitempty" dynamodbav:"last_run_at,omitempty"`
    Recipients   []string        `json:"recipients" dynamodbav:"recipients"`
    Format       string          `json:"format" dynamodbav:"format"`
    Enabled      bool            `json:"enabled" dynamodbav:"enabled"`
    CreatedAt    time.Time       `json:"created_at" dynamodbav:"created_at"`
    CreatedBy    string          `json:"created_by" dynamodbav:"created_by"`
}
```

---

## 5. DynamoDB Schema

### 5.1 Reports Table

```
Table: handloom-reports

Primary Key:
- PK: REPORT#<report_id>
- SK: REPORT#<report_id>

Attributes:
- id: string
- name: string
- type: string
- config: map
- status: string
- file_url: string
- file_size: number
- format: string
- generated_at: string
- expires_at: string
- schedule_id: string
- created_at: string
- created_by: string
- ttl: number (for auto-expiry)

GSI1: type-date-index
- PK: type
- SK: created_at

GSI2: user-reports-index
- PK: created_by
- SK: created_at
```

### 5.2 Report Schedules Table

```
Table: handloom-report-schedules

Primary Key:
- PK: SCHEDULE#<schedule_id>
- SK: SCHEDULE#<schedule_id>

Attributes:
- id: string
- name: string
- type: string
- config: map
- frequency: string
- cron_expression: string
- timezone: string
- next_run_at: string
- last_run_at: string
- recipients: list
- format: string
- enabled: boolean
- created_at: string
- created_by: string

GSI1: next-run-index
- PK: enabled
- SK: next_run_at
```

---

## 6. API Endpoints

### 6.1 Generate Report

```
POST /reports/generate

Request:
{
    "type": "SALES",
    "name": "January Sales Report",
    "config": {
        "date_range": {
            "start_date": "2024-01-01",
            "end_date": "2024-01-31"
        },
        "group_by": "week",
        "filters": {
            "category_id": "cat_001"
        },
        "include_summary": true,
        "include_charts": true
    },
    "format": "pdf"
}

Response:
{
    "success": true,
    "data": {
        "id": "rpt_123",
        "name": "January Sales Report",
        "status": "COMPLETED",
        "file_url": "https://reports.s3.../rpt_123.pdf",
        "generated_at": "2024-01-20T10:00:00Z"
    }
}
```

### 6.2 Create Schedule

```
POST /reports/schedules

Request:
{
    "name": "Weekly Sales Summary",
    "type": "SALES",
    "config": {
        "date_range": {
            "preset": "LAST_7_DAYS"
        },
        "include_summary": true
    },
    "frequency": "WEEKLY",
    "day_of_week": 1,
    "time": "09:00",
    "timezone": "Asia/Kolkata",
    "recipients": ["admin@example.com"],
    "format": "excel"
}

Response:
{
    "success": true,
    "data": {
        "id": "sch_456",
        "name": "Weekly Sales Summary",
        "next_run_at": "2024-01-22T09:00:00+05:30",
        "enabled": true
    }
}
```

### 6.3 Download Report

```
GET /reports/{id}/download?format=pdf

Response:
{
    "success": true,
    "data": {
        "download_url": "https://reports.s3.../signed-url",
        "expires_at": 1705766400
    }
}
```

---

## 7. Report Generation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        REPORT GENERATION PIPELINE                            │
└─────────────────────────────────────────────────────────────────────────────┘

  Request          Validate        Fetch Data       Generate        Store
  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
  │ Receive  │───>│ Validate │───>│  Query   │───>│ Generate │───>│ Upload   │
  │ Request  │    │  Config  │    │   Data   │    │  Output  │    │  to S3   │
  └──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
                                                        │
                                                        ▼
                                                 ┌──────────┐
                                                 │  Notify  │
                                                 │  User    │
                                                 └──────────┘
```

---

## 8. Error Handling

### 8.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_DATE_RANGE | End date before start date | 400 |
| NO_DATA | No data for specified filters | 404 |
| REPORT_NOT_FOUND | Report does not exist | 404 |
| SCHEDULE_NOT_FOUND | Schedule does not exist | 404 |
| GENERATION_FAILED | Report generation error | 500 |
| INVALID_FORMAT | Unsupported export format | 400 |

---

## 9. Security

### 9.1 Access Control

| Role | Generate | View | Download | Schedule |
|------|----------|------|----------|----------|
| Admin | All | All | All | All |
| Manager | All | All | All | Own |
| Staff | Limited | Own | Own | No |

---

## 10. Monitoring

### 10.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Generation Time | Report generation duration | < 60s |
| Success Rate | % of successful generations | > 99% |
| Schedule Execution | On-time schedule runs | > 95% |
| Download Rate | Reports downloaded | Monitor |

---

## 11. Dependencies

### Internal Dependencies
- Order Service: Sales data
- Product Service: Product data
- Customer Service: Customer data
- Analytics Service: Aggregated data

### External Dependencies
- AWS DynamoDB: Report metadata
- AWS S3: Report file storage
- AWS SES: Email delivery
- AWS EventBridge: Schedule triggers

