package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// ReportHandler handles report-related HTTP requests
type ReportHandler struct {
	reportService *service.ReportService
	validation    *middleware.Validation
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(reportService *service.ReportService, validation *middleware.Validation) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		validation:    validation,
	}
}

// Routes returns the report routes
func (h *ReportHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.GenerateReportRequest](h.validation)).Post("/", h.Generate)
	r.Get("/my", h.GetMyReports)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/download", h.GetDownloadURL)
	r.Post("/sales", h.GenerateSalesReport)
	r.Post("/inventory", h.GenerateInventoryReport)
	r.Post("/orders", h.GenerateOrdersReport)
	r.Post("/customers", h.GenerateCustomersReport)
	r.Post("/products", h.GenerateProductsReport)
	r.Post("/artisans", h.GenerateArtisansReport)

	return r
}

// Generate initiates report generation
// POST /admin/reports
func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.GenerateReportRequest](r.Context())

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.Generate(r.Context(), *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GetByID retrieves a report by ID
// GET /admin/reports/{id}
func (h *ReportHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Report ID is required"))
		return
	}

	report, err := h.reportService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

// List retrieves reports with filters
// GET /admin/reports
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListReportsRequest{
		Pagination: parsePagination(r),
	}

	if t := r.URL.Query().Get("type"); t != "" {
		req.Type = domain.ReportType(t)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = domain.ReportStatus(status)
	}
	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			req.StartDate = t
		}
	}
	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			req.EndDate = t
		}
	}

	reports, err := h.reportService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, reports)
}

// GetMyReports retrieves reports for the current user
// GET /admin/reports/my
func (h *ReportHandler) GetMyReports(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	pagination := parsePagination(r)

	reports, err := h.reportService.GetByUser(r.Context(), userID, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, reports)
}

// Delete deletes a report
// DELETE /admin/reports/{id}
func (h *ReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Report ID is required"))
		return
	}

	if err := h.reportService.Delete(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Report deleted successfully",
	})
}

// GetDownloadURL generates a presigned URL for downloading a report
// GET /admin/reports/{id}/download
func (h *ReportHandler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Report ID is required"))
		return
	}

	url, err := h.reportService.GetDownloadURL(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"download_url": url,
	})
}

// GenerateSalesReport generates a sales report
// POST /admin/reports/sales
func (h *ReportHandler) GenerateSalesReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Format    string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0) // Default to last month
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		endDate = time.Now()
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateSalesReport(r.Context(), startDate, endDate, format, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GenerateInventoryReport generates an inventory report
// POST /admin/reports/inventory
func (h *ReportHandler) GenerateInventoryReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Format = "CSV"
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateInventoryReport(r.Context(), format, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GenerateOrdersReport generates an orders report
// POST /admin/reports/orders
func (h *ReportHandler) GenerateOrdersReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Format    string `json:"format"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		endDate = time.Now()
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateOrdersReport(r.Context(), startDate, endDate, format, req.Status, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GenerateCustomersReport generates a customers report
// POST /admin/reports/customers
func (h *ReportHandler) GenerateCustomersReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Format    string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		endDate = time.Now()
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateCustomersReport(r.Context(), startDate, endDate, format, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GenerateProductsReport generates a products report
// POST /admin/reports/products
func (h *ReportHandler) GenerateProductsReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format     string `json:"format"`
		CategoryID string `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Format = "CSV"
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateProductsReport(r.Context(), format, req.CategoryID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}

// GenerateArtisansReport generates an artisans report
// POST /admin/reports/artisans
func (h *ReportHandler) GenerateArtisansReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Format = "CSV"
	}

	format := domain.ReportFormat(req.Format)
	if format == "" {
		format = domain.ReportFormatCSV
	}

	userID := r.Context().Value("user_id").(string)
	report, err := h.reportService.GenerateArtisansReport(r.Context(), format, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, report)
}
