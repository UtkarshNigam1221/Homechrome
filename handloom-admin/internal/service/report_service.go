// Package service implements the business logic layer
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// ReportService implements report generation business logic
type ReportService struct {
	reportRepo       domain.ReportRepository
	orderService     *OrderService
	productService   *ProductService
	customerService  *CustomerService
	inventoryService *InventoryService
	analyticsService *AnalyticsService
	logger           *logger.Logger
}

// NewReportService creates a new ReportService
func NewReportService(
	reportRepo domain.ReportRepository,
	orderService *OrderService,
	productService *ProductService,
	customerService *CustomerService,
	inventoryService *InventoryService,
	analyticsService *AnalyticsService,
	logger *logger.Logger,
) *ReportService {
	return &ReportService{
		reportRepo:       reportRepo,
		orderService:     orderService,
		productService:   productService,
		customerService:  customerService,
		inventoryService: inventoryService,
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// Generate initiates report generation
func (s *ReportService) Generate(ctx context.Context, req domain.GenerateReportRequest, createdBy string) (*domain.Report, error) {
	// Validate report type and format
	if !s.isValidReportType(req.Type) {
		return nil, errors.BadRequest("Invalid report type")
	}
	if !s.isValidFormat(req.Format) {
		return nil, errors.BadRequest("Invalid report format")
	}

	report := &domain.Report{
		ID:         "report_" + uuid.New().String()[:8],
		Name:       req.Name,
		Type:       req.Type,
		Format:     req.Format,
		Status:     domain.ReportStatusPending,
		Parameters: req.Parameters,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		BaseEntity: domain.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: createdBy,
			UpdatedBy: createdBy,
		},
	}
	report.SetKeys()

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, err
	}

	// TODO: In production, push to SQS for async processing
	// For now, we'll just create the report record
	// The actual generation would be handled by a worker service
	go s.processReport(context.Background(), report)

	s.logger.WithContext(ctx).Infof("Created report: %s", report.ID)
	return report, nil
}

// processReport handles the actual report generation
func (s *ReportService) processReport(ctx context.Context, report *domain.Report) {
	// Update status to processing
	now := time.Now()
	report.Status = domain.ReportStatusProcessing
	report.StartedAt = &now
	report.UpdatedAt = now

	if err := s.reportRepo.Update(ctx, report); err != nil {
		s.logger.WithContext(ctx).WithError(err).Errorf("Failed to update report status: %s", report.ID)
		return
	}

	// TODO: Implement actual report generation based on type
	// This would:
	// 1. Query the relevant data based on report type and parameters
	// 2. Generate the file in the requested format
	// 3. Upload to S3
	// 4. Update the report record with file URL

	// For now, simulate processing
	time.Sleep(1 * time.Second)

	// Mark as completed
	completedAt := time.Now()
	report.Status = domain.ReportStatusCompleted
	report.CompletedAt = &completedAt
	report.UpdatedAt = completedAt
	report.FileURL = "https://s3.amazonaws.com/handloom-reports/" + report.ID + "." + s.getExtension(report.Format)
	report.RowCount = 100 // Placeholder

	if err := s.reportRepo.Update(ctx, report); err != nil {
		s.logger.WithContext(ctx).WithError(err).Errorf("Failed to update completed report: %s", report.ID)
	}
}

// GetByID retrieves a report by ID
func (s *ReportService) GetByID(ctx context.Context, id string) (*domain.Report, error) {
	return s.reportRepo.GetByID(ctx, id)
}

// List retrieves reports with filters
func (s *ReportService) List(ctx context.Context, req domain.ListReportsRequest) (*domain.ListReportsResponse, error) {
	return s.reportRepo.List(ctx, req)
}

// GetByUser retrieves reports for a specific user
func (s *ReportService) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListReportsResponse, error) {
	return s.reportRepo.GetByUser(ctx, userID, pagination)
}

// Delete deletes a report
func (s *ReportService) Delete(ctx context.Context, id string) error {
	report, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Don't allow deleting reports that are still processing
	if report.Status == domain.ReportStatusProcessing {
		return errors.BadRequest("Cannot delete a report that is still processing")
	}

	// TODO: Delete file from S3 if exists

	if err := s.reportRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Deleted report: %s", id)
	return nil
}

// GetDownloadURL generates a presigned URL for downloading a report
func (s *ReportService) GetDownloadURL(ctx context.Context, id string) (string, error) {
	report, err := s.reportRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	if report.Status != domain.ReportStatusCompleted {
		return "", errors.BadRequest("Report is not ready for download")
	}

	if report.FileURL == "" {
		return "", errors.NotFound("Report file not available")
	}

	// TODO: Generate actual S3 presigned URL
	return report.FileURL, nil
}

// GenerateSalesReport generates a sales report
func (s *ReportService) GenerateSalesReport(ctx context.Context, startDate, endDate time.Time, format domain.ReportFormat, createdBy string) (*domain.Report, error) {
	req := domain.GenerateReportRequest{
		Name:      "Sales Report",
		Type:      domain.ReportTypeSales,
		Format:    format,
		StartDate: &startDate,
		EndDate:   &endDate,
		Parameters: map[string]interface{}{
			"include_refunds": true,
		},
	}
	return s.Generate(ctx, req, createdBy)
}

// GenerateInventoryReport generates an inventory report
func (s *ReportService) GenerateInventoryReport(ctx context.Context, format domain.ReportFormat, createdBy string) (*domain.Report, error) {
	req := domain.GenerateReportRequest{
		Name:   "Inventory Report",
		Type:   domain.ReportTypeInventory,
		Format: format,
		Parameters: map[string]interface{}{
			"include_out_of_stock": true,
		},
	}
	return s.Generate(ctx, req, createdBy)
}

// GenerateOrdersReport generates an orders report
func (s *ReportService) GenerateOrdersReport(ctx context.Context, startDate, endDate time.Time, format domain.ReportFormat, status string, createdBy string) (*domain.Report, error) {
	req := domain.GenerateReportRequest{
		Name:      "Orders Report",
		Type:      domain.ReportTypeOrders,
		Format:    format,
		StartDate: &startDate,
		EndDate:   &endDate,
		Parameters: map[string]interface{}{
			"status": status,
		},
	}
	return s.Generate(ctx, req, createdBy)
}

// GenerateCustomersReport generates a customers report
func (s *ReportService) GenerateCustomersReport(ctx context.Context, startDate, endDate time.Time, format domain.ReportFormat, createdBy string) (*domain.Report, error) {
	req := domain.GenerateReportRequest{
		Name:      "Customers Report",
		Type:      domain.ReportTypeCustomers,
		Format:    format,
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	return s.Generate(ctx, req, createdBy)
}

// GenerateProductsReport generates a products report
func (s *ReportService) GenerateProductsReport(ctx context.Context, format domain.ReportFormat, categoryID string, createdBy string) (*domain.Report, error) {
	params := make(map[string]interface{})
	if categoryID != "" {
		params["category_id"] = categoryID
	}

	req := domain.GenerateReportRequest{
		Name:       "Products Report",
		Type:       domain.ReportTypeProducts,
		Format:     format,
		Parameters: params,
	}
	return s.Generate(ctx, req, createdBy)
}

// GenerateArtisansReport generates an artisans report
func (s *ReportService) GenerateArtisansReport(ctx context.Context, format domain.ReportFormat, createdBy string) (*domain.Report, error) {
	req := domain.GenerateReportRequest{
		Name:   "Artisans Report",
		Type:   domain.ReportTypeArtisans,
		Format: format,
	}
	return s.Generate(ctx, req, createdBy)
}

// Helpers

func (s *ReportService) isValidReportType(t domain.ReportType) bool {
	validTypes := []domain.ReportType{
		domain.ReportTypeSales,
		domain.ReportTypeOrders,
		domain.ReportTypeInventory,
		domain.ReportTypeCustomers,
		domain.ReportTypeProducts,
		domain.ReportTypeArtisans,
	}
	for _, vt := range validTypes {
		if t == vt {
			return true
		}
	}
	return false
}

func (s *ReportService) isValidFormat(f domain.ReportFormat) bool {
	validFormats := []domain.ReportFormat{
		domain.ReportFormatCSV,
		domain.ReportFormatXLSX,
		domain.ReportFormatPDF,
	}
	for _, vf := range validFormats {
		if f == vf {
			return true
		}
	}
	return false
}

func (s *ReportService) getExtension(format domain.ReportFormat) string {
	switch format {
	case domain.ReportFormatCSV:
		return "csv"
	case domain.ReportFormatXLSX:
		return "xlsx"
	case domain.ReportFormatPDF:
		return "pdf"
	default:
		return "csv"
	}
}
