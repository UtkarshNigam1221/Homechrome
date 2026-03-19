// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CustomerService implements customer business logic
type CustomerService struct {
	customerRepo domain.CustomerRepository
	orderRepo    domain.OrderRepository
}

// NewCustomerService creates a new CustomerService
func NewCustomerService(
	customerRepo domain.CustomerRepository,
	orderRepo domain.OrderRepository,
) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		orderRepo:    orderRepo,
	}
}

// Create creates a new customer
func (s *CustomerService) Create(ctx context.Context, req domain.CreateCustomerRequest, createdBy string) (*domain.Customer, error) {
	// Check if email already exists
	existing, _ := s.customerRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.Conflict("Customer with this email already exists")
	}

	customer := &domain.Customer{
		ID:        "cust_" + uuid.New().String()[:8],
		Email:     req.Email,
		Phone:     req.Phone,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Status:    domain.CustomerStatusActive,
		Addresses: []domain.Address{},
		Tags:      req.Tags,
		Notes:     req.Notes,
		BaseEntity: domain.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: createdBy,
			UpdatedBy: createdBy,
		},
	}

	if req.Address != nil {
		customer.Addresses = append(customer.Addresses, *req.Address)
	}

	customer.SetKeys()

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Created customer", "customer_id", customer.ID)
	return customer, nil
}

// GetByID retrieves a customer by ID
func (s *CustomerService) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	return s.customerRepo.GetByID(ctx, id)
}

// GetByEmail retrieves a customer by email
func (s *CustomerService) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	return s.customerRepo.GetByEmail(ctx, email)
}

// Update updates a customer
func (s *CustomerService) Update(ctx context.Context, id string, req domain.UpdateCustomerRequest, updatedBy string) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FirstName != "" {
		customer.FirstName = req.FirstName
	}
	if req.LastName != "" {
		customer.LastName = req.LastName
	}
	if req.Phone != "" {
		customer.Phone = req.Phone
	}
	if len(req.Tags) > 0 {
		customer.Tags = req.Tags
	}
	if req.Notes != "" {
		customer.Notes = req.Notes
	}
	if req.Status != "" {
		customer.Status = req.Status
	}

	customer.UpdatedAt = time.Now()
	customer.UpdatedBy = updatedBy

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Updated customer", "customer_id", id)
	return customer, nil
}

// Delete deletes a customer (soft delete)
func (s *CustomerService) Delete(ctx context.Context, id string) error {
	customer, err := s.customerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if customer has orders
	orders, _ := s.orderRepo.GetByCustomer(ctx, id, domain.PaginationRequest{Limit: 1})
	if orders != nil && len(orders.Orders) > 0 {
		return errors.BadRequest("Cannot delete customer with existing orders")
	}

	customer.Status = domain.CustomerStatusInactive
	customer.UpdatedAt = time.Now()

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Soft deleted customer", "customer_id", id)
	return nil
}

// List retrieves customers with filters
func (s *CustomerService) List(ctx context.Context, req domain.ListCustomersRequest) (*domain.ListCustomersResponse, error) {
	return s.customerRepo.List(ctx, req)
}

// Search searches customers by query
func (s *CustomerService) Search(ctx context.Context, query string, pagination domain.PaginationRequest) (*domain.ListCustomersResponse, error) {
	return s.customerRepo.Search(ctx, query, pagination)
}

// GetOrders retrieves orders for a customer
func (s *CustomerService) GetOrders(ctx context.Context, customerID string, pagination domain.PaginationRequest) (*domain.ListOrdersResponse, error) {
	return s.orderRepo.GetByCustomer(ctx, customerID, pagination)
}

// AddAddress adds an address to a customer
func (s *CustomerService) AddAddress(ctx context.Context, customerID string, address domain.Address, updatedBy string) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Generate address ID if not provided
	if address.ID == "" {
		address.ID = "addr_" + uuid.New().String()[:8]
	}

	// If this is the first address or marked as default, set it as default
	if len(customer.Addresses) == 0 || address.IsDefault {
		// Clear other default addresses
		for i := range customer.Addresses {
			customer.Addresses[i].IsDefault = false
		}
		address.IsDefault = true
	}

	customer.Addresses = append(customer.Addresses, address)
	customer.UpdatedAt = time.Now()
	customer.UpdatedBy = updatedBy

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Added address to customer", "customer_id", customerID)
	return customer, nil
}

// UpdateAddress updates an address for a customer
func (s *CustomerService) UpdateAddress(ctx context.Context, customerID, addressID string, address domain.Address, updatedBy string) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	found := false
	for i, addr := range customer.Addresses {
		if addr.ID == addressID {
			address.ID = addressID
			customer.Addresses[i] = address
			found = true
			break
		}
	}

	if !found {
		return nil, errors.NotFound("Address not found")
	}

	// Handle default address
	if address.IsDefault {
		for i := range customer.Addresses {
			if customer.Addresses[i].ID != addressID {
				customer.Addresses[i].IsDefault = false
			}
		}
	}

	customer.UpdatedAt = time.Now()
	customer.UpdatedBy = updatedBy

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Updated address for customer", "customer_id", customerID)
	return customer, nil
}

// RemoveAddress removes an address from a customer
func (s *CustomerService) RemoveAddress(ctx context.Context, customerID, addressID string, updatedBy string) (*domain.Customer, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	found := false
	newAddresses := make([]domain.Address, 0)
	for _, addr := range customer.Addresses {
		if addr.ID != addressID {
			newAddresses = append(newAddresses, addr)
		} else {
			found = true
		}
	}

	if !found {
		return nil, errors.NotFound("Address not found")
	}

	customer.Addresses = newAddresses
	customer.UpdatedAt = time.Now()
	customer.UpdatedBy = updatedBy

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Removed address from customer", "customer_id", customerID)
	return customer, nil
}

// UpdateStats updates customer statistics
func (s *CustomerService) UpdateStats(ctx context.Context, customerID string, totalOrders int, totalSpent float64) error {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return err
	}

	customer.TotalOrders = totalOrders
	customer.TotalSpent = totalSpent
	customer.UpdatedAt = time.Now()

	return s.customerRepo.Update(ctx, customer)
}
