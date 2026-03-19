package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// CustomerHandler handles customer-related HTTP requests
type CustomerHandler struct {
	customerService *service.CustomerService
	validation      *middleware.Validation
}

// NewCustomerHandler creates a new CustomerHandler
func NewCustomerHandler(customerService *service.CustomerService, validation *middleware.Validation) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		validation:      validation,
	}
}

// Routes returns the customer routes
func (h *CustomerHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateCustomerRequest](h.validation)).Post("/", h.Create)
	r.Get("/search", h.Search)
	r.Get("/{id}", h.GetByID)
	r.With(middleware.ValidateJSONTyped[domain.UpdateCustomerRequest](h.validation)).Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/orders", h.GetOrders)
	r.With(middleware.ValidateJSONTyped[domain.Address](h.validation)).Post("/{id}/addresses", h.AddAddress)
	r.With(middleware.ValidateJSONTyped[domain.Address](h.validation)).Put("/{id}/addresses/{addressId}", h.UpdateAddress)
	r.Delete("/{id}/addresses/{addressId}", h.RemoveAddress)

	return r
}

// Create creates a new customer
// POST /admin/customers
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.CreateCustomerRequest](r.Context())

	userID, _ := r.Context().Value("user_id").(string)
	customer, err := h.customerService.Create(r.Context(), *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, customer)
}

// GetByID retrieves a customer by ID
// GET /admin/customers/{id}
func (h *CustomerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Customer ID is required"))
		return
	}

	customer, err := h.customerService.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customer)
}

// Update updates a customer
// PUT /admin/customers/{id}
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Customer ID is required"))
		return
	}

	req := middleware.MustGetValidatedBody[domain.UpdateCustomerRequest](r.Context())

	userID, _ := r.Context().Value("user_id").(string)
	customer, err := h.customerService.Update(r.Context(), id, *req, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customer)
}

// Delete deletes a customer
// DELETE /admin/customers/{id}
func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Customer ID is required"))
		return
	}

	if err := h.customerService.Delete(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Customer deleted successfully",
	})
}

// List retrieves customers with filters
// GET /admin/customers
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	req := domain.ListCustomersRequest{
		Pagination: parsePagination(r),
	}

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = domain.CustomerStatus(status)
	}
	if search := r.URL.Query().Get("search"); search != "" {
		req.Search = search
	}

	customers, err := h.customerService.List(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customers)
}

// Search searches customers
// GET /admin/customers/search
func (h *CustomerHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, errors.BadRequest("Search query is required"))
		return
	}

	pagination := parsePagination(r)
	customers, err := h.customerService.Search(r.Context(), query, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customers)
}

// GetOrders retrieves orders for a customer
// GET /admin/customers/{id}/orders
func (h *CustomerHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Customer ID is required"))
		return
	}

	pagination := parsePagination(r)
	orders, err := h.customerService.GetOrders(r.Context(), id, pagination)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, orders)
}

// AddAddress adds an address to a customer
// POST /admin/customers/{id}/addresses
func (h *CustomerHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, errors.BadRequest("Customer ID is required"))
		return
	}

	address := middleware.MustGetValidatedBody[domain.Address](r.Context())

	userID, _ := r.Context().Value("user_id").(string)
	customer, err := h.customerService.AddAddress(r.Context(), id, *address, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customer)
}

// UpdateAddress updates an address for a customer
// PUT /admin/customers/{id}/addresses/{addressId}
func (h *CustomerHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	addressID := chi.URLParam(r, "addressId")
	if customerID == "" || addressID == "" {
		response.Error(w, errors.BadRequest("Customer ID and Address ID are required"))
		return
	}

	address := middleware.MustGetValidatedBody[domain.Address](r.Context())

	userID, _ := r.Context().Value("user_id").(string)
	customer, err := h.customerService.UpdateAddress(r.Context(), customerID, addressID, *address, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customer)
}

// RemoveAddress removes an address from a customer
// DELETE /admin/customers/{id}/addresses/{addressId}
func (h *CustomerHandler) RemoveAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	addressID := chi.URLParam(r, "addressId")
	if customerID == "" || addressID == "" {
		response.Error(w, errors.BadRequest("Customer ID and Address ID are required"))
		return
	}

	userID, _ := r.Context().Value("user_id").(string)
	customer, err := h.customerService.RemoveAddress(r.Context(), customerID, addressID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, customer)
}
