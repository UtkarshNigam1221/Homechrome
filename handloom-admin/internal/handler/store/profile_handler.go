package store

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// ProfileHandler handles customer profile requests.
// All routes require customer authentication (applied at router level).
type ProfileHandler struct {
	customerRepo domain.CustomerRepository
	validation   *middleware.Validation
	logger       *logger.Logger
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(
	cr domain.CustomerRepository,
	v *middleware.Validation,
	l *logger.Logger,
) *ProfileHandler {
	return &ProfileHandler{
		customerRepo: cr,
		validation:   v,
		logger:       l,
	}
}

// Routes returns the customer profile routes.
func (h *ProfileHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetProfile)
	r.With(middleware.ValidateJSONTyped[UpdateProfileRequest](h.validation)).Patch("/", h.UpdateProfile)
	r.With(middleware.ValidateJSONTyped[AddAddressRequest](h.validation)).Post("/addresses", h.AddAddress)
	r.With(middleware.ValidateJSONTyped[AddAddressRequest](h.validation)).Put("/addresses/{id}", h.UpdateAddress)
	r.Delete("/addresses/{id}", h.RemoveAddress)

	return r
}

// ==================== Request Types ====================

// UpdateProfileRequest contains data for updating a customer profile.
type UpdateProfileRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

// AddAddressRequest contains data for adding or updating an address.
type AddAddressRequest struct {
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name" validate:"required"`
	Phone        string `json:"phone" validate:"required"`
	AddressLine1 string `json:"address_line1" validate:"required"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	PostalCode   string `json:"postal_code" validate:"required"`
	Country      string `json:"country" validate:"required"`
	IsDefault    bool   `json:"is_default,omitempty"`
}

// ==================== Handlers ====================

// GetProfile handles retrieving the authenticated customer's profile.
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, customer)
}

// UpdateProfile handles updating the authenticated customer's profile.
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	req := middleware.MustGetValidatedBody[UpdateProfileRequest](ctx)

	customer, err := h.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Apply updates
	if req.FirstName != "" {
		customer.FirstName = req.FirstName
	}
	if req.LastName != "" {
		customer.LastName = req.LastName
	}
	if req.Email != "" {
		customer.Email = req.Email
	}
	customer.UpdatedAt = time.Now()

	if err := h.customerRepo.Update(ctx, customer); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to update customer profile")
		response.Error(w, err)
		return
	}

	response.Success(w, customer)
}

// AddAddress handles adding a new address to the customer's address list.
func (h *ProfileHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	req := middleware.MustGetValidatedBody[AddAddressRequest](ctx)

	customer, err := h.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	newAddress := addressFromRequest(uuid.New().String(), *req)

	// If the new address is default, clear default on existing addresses
	if newAddress.IsDefault {
		for i := range customer.Addresses {
			customer.Addresses[i].IsDefault = false
		}
	}

	customer.Addresses = append(customer.Addresses, newAddress)
	customer.UpdatedAt = time.Now()

	if err := h.customerRepo.Update(ctx, customer); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to add address")
		response.Error(w, err)
		return
	}

	response.Created(w, newAddress)
}

// UpdateAddress handles updating a specific address in the customer's address list.
func (h *ProfileHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	addressID := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[AddAddressRequest](ctx)

	customer, err := h.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Find the address to update
	found := false
	for i := range customer.Addresses {
		if customer.Addresses[i].ID == addressID {
			// If the updated address is being set as default, clear others first
			if req.IsDefault {
				for j := range customer.Addresses {
					customer.Addresses[j].IsDefault = false
				}
			}

			customer.Addresses[i] = addressFromRequest(addressID, *req)
			found = true
			break
		}
	}

	if !found {
		response.Error(w, errors.NotFound("Address"))
		return
	}

	customer.UpdatedAt = time.Now()

	if err := h.customerRepo.Update(ctx, customer); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to update address")
		response.Error(w, err)
		return
	}

	response.Success(w, customer.Addresses)
}

// RemoveAddress handles removing an address from the customer's address list.
func (h *ProfileHandler) RemoveAddress(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	ctx := r.Context()
	addressID := chi.URLParam(r, "id")

	customer, err := h.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Find and remove the address
	found := false
	addresses := make([]domain.Address, 0, len(customer.Addresses))
	for _, addr := range customer.Addresses {
		if addr.ID == addressID {
			found = true
			continue
		}
		addresses = append(addresses, addr)
	}

	if !found {
		response.Error(w, errors.NotFound("Address"))
		return
	}

	customer.Addresses = addresses
	customer.UpdatedAt = time.Now()

	if err := h.customerRepo.Update(ctx, customer); err != nil {
		h.logger.WithContext(ctx).WithError(err).Error("Failed to remove address")
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// addressFromRequest maps an AddAddressRequest to a domain.Address with the given ID.
func addressFromRequest(id string, req AddAddressRequest) domain.Address {
	return domain.Address{
		ID:           id,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
		IsDefault:    req.IsDefault,
	}
}
