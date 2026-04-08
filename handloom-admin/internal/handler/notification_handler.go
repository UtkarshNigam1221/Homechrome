package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService *service.NotificationService
	validation          *middleware.Validation
}

// NewNotificationHandler creates a new NotificationHandler
func NewNotificationHandler(notificationService *service.NotificationService, validation *middleware.Validation) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		validation:          validation,
	}
}

// Routes returns the notification routes
func (h *NotificationHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.SendNotificationRequest](h.validation)).Post("/", h.Send)
	r.With(middleware.ValidateJSONTyped[domain.SendBulkNotificationRequest](h.validation)).Post("/bulk", h.SendBulk)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/read", h.MarkAsRead)
	r.Post("/read-all", h.MarkAllAsRead)
	r.Get("/my", h.GetMyNotifications)

	return r
}

// Send sends a notification
// POST /admin/notifications
func (h *NotificationHandler) Send(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.SendNotificationRequest](r.Context())

	createdBy := middleware.GetCreatedBy(r.Context())

	notification, err := h.notificationService.Send(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, notification)
}

// SendBulk sends notifications to multiple recipients
// POST /admin/notifications/bulk
func (h *NotificationHandler) SendBulk(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.SendBulkNotificationRequest](r.Context())

	createdBy := middleware.GetCreatedBy(r.Context())

	result, err := h.notificationService.SendBulk(r.Context(), *req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetByID retrieves a notification by ID
// GET /admin/notifications/{id}
func (h *NotificationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	notification, err := h.notificationService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, notification)
}

// List retrieves notifications with filters
// GET /admin/notifications
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListNotificationsRequest{
		PaginationRequest: parsePagination(r),
	}

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		req.UserID = &userID
	}

	if notifType := r.URL.Query().Get("type"); notifType != "" {
		t := domain.NotificationType(notifType)
		req.Type = &t
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.NotificationStatus(status)
		req.Status = &s
	}

	result, err := h.notificationService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetUserNotifications retrieves notifications for the current user
// GET /admin/notifications/me
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		response.Error(w, nil)
		return
	}

	pagination := parsePagination(r)
	result, err := h.notificationService.GetByUser(r.Context(), user.ID, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetMyNotifications is an alias for GetUserNotifications
// GET /admin/notifications/my
func (h *NotificationHandler) GetMyNotifications(w http.ResponseWriter, r *http.Request) {
	h.GetUserNotifications(w, r)
}

// MarkAsRead marks a notification as read
// POST /admin/notifications/{id}/read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.notificationService.MarkAsRead(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "read"})
}

// MarkAllAsRead marks all notifications for the current user as read
// POST /admin/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		response.Error(w, nil)
		return
	}

	if err := h.notificationService.MarkAllAsRead(r.Context(), user.ID); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "all read"})
}
