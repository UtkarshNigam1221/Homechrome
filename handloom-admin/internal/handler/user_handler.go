// Package handler implements HTTP handlers
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService domain.UserService
	logger      *logger.Logger
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService domain.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// Routes returns the user routes
func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}/status", h.UpdateStatus)

	return r
}

// List handles listing users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := domain.ListUsersRequest{
		PaginationRequest: parsePagination(r),
	}

	// Parse filters
	if role := r.URL.Query().Get("role"); role != "" {
		roleEnum := domain.UserRole(role)
		req.Role = &roleEnum
	}
	if status := r.URL.Query().Get("status"); status != "" {
		statusEnum := domain.UserStatus(status)
		req.Status = &statusEnum
	}
	req.Search = r.URL.Query().Get("search")

	result, err := h.userService.List(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Create handles creating a new user
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		response.Error(w, errors.Validation("Email, password, first name, and last name are required"))
		return
	}

	if len(req.Password) < 8 {
		response.Error(w, errors.Validation("Password must be at least 8 characters"))
		return
	}

	if req.Role == "" {
		response.Error(w, errors.Validation("Role is required"))
		return
	}

	createdBy := getUserIDFromContext(ctx)
	user, err := h.userService.Create(ctx, req, createdBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

// GetByID handles getting a user by ID
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	user, err := h.userService.GetByID(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

// Update handles updating a user
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req domain.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	updatedBy := getUserIDFromContext(ctx)
	user, err := h.userService.Update(ctx, id, req, updatedBy)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

// Delete handles deleting a user
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.userService.Delete(ctx, id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// UpdateStatus handles updating a user's status
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req struct {
		Status domain.UserStatus `json:"status"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.Status == "" {
		response.Error(w, errors.Validation("Status is required"))
		return
	}

	updatedBy := getUserIDFromContext(ctx)
	if err := h.userService.UpdateStatus(ctx, id, req.Status, updatedBy); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "User status updated successfully"})
}
