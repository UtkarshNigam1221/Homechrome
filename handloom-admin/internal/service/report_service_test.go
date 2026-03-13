package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

func newTestReportService(ctrl *gomock.Controller) (*ReportService, *mocks.MockReportRepository) {
	mockReportRepo := mocks.NewMockReportRepository(ctrl)
	log := logger.NewNoop()

	// Pass nil for concrete service pointers - they are only used in processReport (async goroutine)
	service := NewReportService(mockReportRepo, nil, nil, nil, nil, nil, log)

	return service, mockReportRepo
}

func TestReportService_Generate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful generation", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		req := domain.GenerateReportRequest{
			Name:   "Monthly Sales Report",
			Type:   domain.ReportTypeSales,
			Format: domain.ReportFormatCSV,
			Parameters: map[string]interface{}{
				"include_refunds": true,
			},
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		// Capture the ID from Create mock to avoid race with processReport goroutine
		var createdID string
		mockReportRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, report *domain.Report) error {
				createdID = report.ID
				assert.Contains(t, report.ID, "report_")
				assert.Equal(t, "Monthly Sales Report", report.Name)
				assert.Equal(t, domain.ReportTypeSales, report.Type)
				assert.Equal(t, domain.ReportFormatCSV, report.Format)
				assert.Equal(t, domain.ReportStatusPending, report.Status)
				assert.Equal(t, "admin_123", report.CreatedBy)
				assert.NotNil(t, report.Parameters)
				return nil
			})

		// processReport runs in a goroutine - use AnyTimes for async Update
		mockReportRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			AnyTimes()

		report, err := service.Generate(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, report)
		// Only read ID — other fields may be mutated by the async goroutine
		assert.Equal(t, createdID, report.ID)
	})

	t.Run("invalid report type", func(t *testing.T) {
		req := domain.GenerateReportRequest{
			Name:   "Invalid Report",
			Type:   domain.ReportType("INVALID_TYPE"),
			Format: domain.ReportFormatCSV,
		}

		report, err := service.Generate(ctx, req, "admin_123")

		assert.Nil(t, report)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid report type")
	})

	t.Run("invalid report format", func(t *testing.T) {
		req := domain.GenerateReportRequest{
			Name:   "Bad Format Report",
			Type:   domain.ReportTypeSales,
			Format: domain.ReportFormat("INVALID_FORMAT"),
		}

		report, err := service.Generate(ctx, req, "admin_123")

		assert.Nil(t, report)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid report format")
	})

	t.Run("repo error on create", func(t *testing.T) {
		req := domain.GenerateReportRequest{
			Name:   "Sales Report",
			Type:   domain.ReportTypeSales,
			Format: domain.ReportFormatCSV,
		}

		mockReportRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.Internal("Database error"))

		report, err := service.Generate(ctx, req, "admin_123")

		assert.Nil(t, report)
		require.Error(t, err)
	})
}

func TestReportService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		expected := &domain.Report{
			ID:     "report_abc123",
			Name:   "Sales Report",
			Type:   domain.ReportTypeSales,
			Format: domain.ReportFormatCSV,
			Status: domain.ReportStatusCompleted,
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(expected, nil)

		report, err := service.GetByID(ctx, "report_abc123")

		require.NoError(t, err)
		assert.Equal(t, "report_abc123", report.ID)
		assert.Equal(t, domain.ReportStatusCompleted, report.Status)
	})

	t.Run("not found", func(t *testing.T) {
		mockReportRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Report"))

		report, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, report)
		require.Error(t, err)
	})
}

func TestReportService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListReportsRequest{
			Pagination: domain.PaginationRequest{Limit: 20},
		}

		expectedResponse := &domain.ListReportsResponse{
			Reports: []*domain.Report{
				{ID: "report_1", Name: "Sales Report", Type: domain.ReportTypeSales},
				{ID: "report_2", Name: "Orders Report", Type: domain.ReportTypeOrders},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockReportRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Reports, 2)
	})
}

func TestReportService_GetByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful get by user", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListReportsResponse{
			Reports: []*domain.Report{
				{ID: "report_1", Name: "My Sales Report"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockReportRepo.EXPECT().
			GetByUser(ctx, "admin_123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetByUser(ctx, "admin_123", pagination)

		require.NoError(t, err)
		assert.Len(t, response.Reports, 1)
	})
}

func TestReportService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful deletion of completed report", func(t *testing.T) {
		report := &domain.Report{
			ID:     "report_abc123",
			Name:   "Sales Report",
			Status: domain.ReportStatusCompleted,
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		mockReportRepo.EXPECT().
			Delete(ctx, "report_abc123").
			Return(nil)

		err := service.Delete(ctx, "report_abc123")

		require.NoError(t, err)
	})

	t.Run("successful deletion of pending report", func(t *testing.T) {
		report := &domain.Report{
			ID:     "report_abc123",
			Name:   "Sales Report",
			Status: domain.ReportStatusPending,
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		mockReportRepo.EXPECT().
			Delete(ctx, "report_abc123").
			Return(nil)

		err := service.Delete(ctx, "report_abc123")

		require.NoError(t, err)
	})

	t.Run("blocked - report is processing", func(t *testing.T) {
		report := &domain.Report{
			ID:     "report_abc123",
			Name:   "Sales Report",
			Status: domain.ReportStatusProcessing,
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		err := service.Delete(ctx, "report_abc123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "still processing")
	})

	t.Run("report not found", func(t *testing.T) {
		mockReportRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Report"))

		err := service.Delete(ctx, "nonexistent")

		require.Error(t, err)
	})
}

func TestReportService_GetDownloadURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful download URL", func(t *testing.T) {
		report := &domain.Report{
			ID:      "report_abc123",
			Name:    "Sales Report",
			Status:  domain.ReportStatusCompleted,
			FileURL: "https://s3.amazonaws.com/handloom-reports/report_abc123.csv",
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		url, err := service.GetDownloadURL(ctx, "report_abc123")

		require.NoError(t, err)
		assert.Equal(t, "https://s3.amazonaws.com/handloom-reports/report_abc123.csv", url)
	})

	t.Run("report not completed", func(t *testing.T) {
		report := &domain.Report{
			ID:     "report_abc123",
			Name:   "Sales Report",
			Status: domain.ReportStatusPending,
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		url, err := service.GetDownloadURL(ctx, "report_abc123")

		assert.Empty(t, url)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not ready for download")
	})

	t.Run("completed but no file URL", func(t *testing.T) {
		report := &domain.Report{
			ID:      "report_abc123",
			Name:    "Sales Report",
			Status:  domain.ReportStatusCompleted,
			FileURL: "", // Empty
		}

		mockReportRepo.EXPECT().
			GetByID(ctx, "report_abc123").
			Return(report, nil)

		url, err := service.GetDownloadURL(ctx, "report_abc123")

		assert.Empty(t, url)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
	})

	t.Run("report not found", func(t *testing.T) {
		mockReportRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Report"))

		url, err := service.GetDownloadURL(ctx, "nonexistent")

		assert.Empty(t, url)
		require.Error(t, err)
	})
}

func TestReportService_GenerateSalesReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful sales report generation", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		mockReportRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, report *domain.Report) error {
				assert.Equal(t, "Sales Report", report.Name)
				assert.Equal(t, domain.ReportTypeSales, report.Type)
				assert.Equal(t, domain.ReportFormatCSV, report.Format)
				assert.Equal(t, domain.ReportStatusPending, report.Status)
				assert.NotNil(t, report.Parameters)
				assert.Equal(t, true, report.Parameters["include_refunds"])
				return nil
			})

		mockReportRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			AnyTimes()

		report, err := service.GenerateSalesReport(ctx, startDate, endDate, domain.ReportFormatCSV, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Contains(t, report.ID, "report_")
	})
}

func TestReportService_GenerateInventoryReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful inventory report generation", func(t *testing.T) {
		mockReportRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, report *domain.Report) error {
				assert.Equal(t, "Inventory Report", report.Name)
				assert.Equal(t, domain.ReportTypeInventory, report.Type)
				assert.Equal(t, domain.ReportFormatXLSX, report.Format)
				assert.Equal(t, true, report.Parameters["include_out_of_stock"])
				return nil
			})

		mockReportRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			AnyTimes()

		report, err := service.GenerateInventoryReport(ctx, domain.ReportFormatXLSX, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Contains(t, report.ID, "report_")
	})
}

func TestReportService_GenerateArtisansReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	t.Run("successful artisans report generation", func(t *testing.T) {
		mockReportRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, report *domain.Report) error {
				assert.Equal(t, "Artisans Report", report.Name)
				assert.Equal(t, domain.ReportTypeArtisans, report.Type)
				assert.Equal(t, domain.ReportFormatPDF, report.Format)
				return nil
			})

		mockReportRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			AnyTimes()

		report, err := service.GenerateArtisansReport(ctx, domain.ReportFormatPDF, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Contains(t, report.ID, "report_")
	})
}

func TestReportService_ValidReportTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	validTypes := []domain.ReportType{
		domain.ReportTypeSales,
		domain.ReportTypeOrders,
		domain.ReportTypeInventory,
		domain.ReportTypeCustomers,
		domain.ReportTypeProducts,
		domain.ReportTypeArtisans,
	}

	for _, rt := range validTypes {
		t.Run("valid type: "+string(rt), func(t *testing.T) {
			req := domain.GenerateReportRequest{
				Name:   "Test Report",
				Type:   rt,
				Format: domain.ReportFormatCSV,
			}

			mockReportRepo.EXPECT().
				Create(ctx, gomock.Any()).
				Return(nil)

			mockReportRepo.EXPECT().
				Update(gomock.Any(), gomock.Any()).
				AnyTimes()

			report, err := service.Generate(ctx, req, "admin_123")

			require.NoError(t, err)
			assert.NotNil(t, report)
		})
	}
}

func TestReportService_ValidFormats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service, mockReportRepo := newTestReportService(ctrl)
	ctx := context.Background()

	validFormats := []domain.ReportFormat{
		domain.ReportFormatCSV,
		domain.ReportFormatXLSX,
		domain.ReportFormatPDF,
	}

	for _, f := range validFormats {
		t.Run("valid format: "+string(f), func(t *testing.T) {
			req := domain.GenerateReportRequest{
				Name:   "Test Report",
				Type:   domain.ReportTypeSales,
				Format: f,
			}

			mockReportRepo.EXPECT().
				Create(ctx, gomock.Any()).
				Return(nil)

			mockReportRepo.EXPECT().
				Update(gomock.Any(), gomock.Any()).
				AnyTimes()

			report, err := service.Generate(ctx, req, "admin_123")

			require.NoError(t, err)
			assert.NotNil(t, report)
		})
	}
}
