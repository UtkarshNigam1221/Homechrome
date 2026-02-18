package service

import (
	"context"
	"testing"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCustomerService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := domain.CreateCustomerRequest{
			Email:     "john@example.com",
			Phone:     "+919876543210",
			FirstName: "John",
			LastName:  "Doe",
			Tags:      []string{"VIP"},
			Notes:     "Premium customer",
		}

		mockCustomerRepo.EXPECT().
			GetByEmail(ctx, "john@example.com").
			Return(nil, errors.NotFound("Customer"))

		mockCustomerRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Contains(t, customer.ID, "cust_")
				assert.Equal(t, "john@example.com", customer.Email)
				assert.Equal(t, "+919876543210", customer.Phone)
				assert.Equal(t, "John", customer.FirstName)
				assert.Equal(t, "Doe", customer.LastName)
				assert.Equal(t, domain.CustomerStatusActive, customer.Status)
				assert.Equal(t, []string{"VIP"}, customer.Tags)
				assert.Equal(t, "admin_123", customer.CreatedBy)
				return nil
			})

		customer, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Contains(t, customer.ID, "cust_")
		assert.Equal(t, domain.CustomerStatusActive, customer.Status)
	})

	t.Run("duplicate email", func(t *testing.T) {
		req := domain.CreateCustomerRequest{
			Email:     "existing@example.com",
			FirstName: "Jane",
			LastName:  "Doe",
		}

		existing := &domain.Customer{
			ID:    "cust_existing",
			Email: "existing@example.com",
		}

		mockCustomerRepo.EXPECT().
			GetByEmail(ctx, "existing@example.com").
			Return(existing, nil)

		customer, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, customer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("creation with address", func(t *testing.T) {
		address := &domain.Address{
			AddressLine1: "123 Main St",
			City:         "Mumbai",
			State:        "Maharashtra",
			PostalCode:   "400001",
			Country:      "India",
		}

		req := domain.CreateCustomerRequest{
			Email:     "withaddr@example.com",
			FirstName: "Test",
			LastName:  "User",
			Address:   address,
		}

		mockCustomerRepo.EXPECT().
			GetByEmail(ctx, "withaddr@example.com").
			Return(nil, errors.NotFound("Customer"))

		mockCustomerRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 1)
				assert.Equal(t, "123 Main St", customer.Addresses[0].AddressLine1)
				return nil
			})

		customer, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})
}

func TestCustomerService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		expected := &domain.Customer{
			ID:        "cust_abc123",
			Email:     "john@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(expected, nil)

		customer, err := service.GetByID(ctx, "cust_abc123")

		require.NoError(t, err)
		assert.Equal(t, "cust_abc123", customer.ID)
	})

	t.Run("not found", func(t *testing.T) {
		mockCustomerRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Customer"))

		customer, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, customer)
		require.Error(t, err)
	})
}

func TestCustomerService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		existing := &domain.Customer{
			ID:        "cust_abc123",
			Email:     "john@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Phone:     "+919876543210",
			Status:    domain.CustomerStatusActive,
		}

		req := domain.UpdateCustomerRequest{
			FirstName: "Jonathan",
			Phone:     "+919876543211",
			Tags:      []string{"VIP", "Returning"},
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Equal(t, "Jonathan", customer.FirstName)
				assert.Equal(t, "+919876543211", customer.Phone)
				assert.Equal(t, []string{"VIP", "Returning"}, customer.Tags)
				assert.Equal(t, "Doe", customer.LastName) // Unchanged
				assert.Equal(t, "admin_456", customer.UpdatedBy)
				return nil
			})

		customer, err := service.Update(ctx, "cust_abc123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, "Jonathan", customer.FirstName)
	})

	t.Run("customer not found", func(t *testing.T) {
		req := domain.UpdateCustomerRequest{
			FirstName: "Test",
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Customer"))

		customer, err := service.Update(ctx, "nonexistent", req, "admin_456")

		assert.Nil(t, customer)
		require.Error(t, err)
	})
}

func TestCustomerService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful soft delete", func(t *testing.T) {
		existing := &domain.Customer{
			ID:     "cust_abc123",
			Email:  "john@example.com",
			Status: domain.CustomerStatusActive,
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockOrderRepo.EXPECT().
			GetByCustomer(ctx, "cust_abc123", domain.PaginationRequest{Limit: 1}).
			Return(&domain.ListOrdersResponse{
				Orders: []*domain.Order{},
			}, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Equal(t, domain.CustomerStatusInactive, customer.Status)
				return nil
			})

		err := service.Delete(ctx, "cust_abc123")

		require.NoError(t, err)
	})

	t.Run("blocked - customer has orders", func(t *testing.T) {
		existing := &domain.Customer{
			ID:     "cust_abc123",
			Email:  "john@example.com",
			Status: domain.CustomerStatusActive,
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockOrderRepo.EXPECT().
			GetByCustomer(ctx, "cust_abc123", domain.PaginationRequest{Limit: 1}).
			Return(&domain.ListOrdersResponse{
				Orders: []*domain.Order{
					{ID: "order_1"},
				},
			}, nil)

		err := service.Delete(ctx, "cust_abc123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "existing orders")
	})

	t.Run("customer not found", func(t *testing.T) {
		mockCustomerRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Customer"))

		err := service.Delete(ctx, "nonexistent")

		require.Error(t, err)
	})
}

func TestCustomerService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListCustomersRequest{
			Pagination: domain.PaginationRequest{Limit: 20},
		}

		expectedResponse := &domain.ListCustomersResponse{
			Customers: []*domain.Customer{
				{ID: "cust_1", FirstName: "John"},
				{ID: "cust_2", FirstName: "Jane"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockCustomerRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Customers, 2)
	})
}

func TestCustomerService_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful search", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListCustomersResponse{
			Customers: []*domain.Customer{
				{ID: "cust_1", FirstName: "John", LastName: "Doe"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockCustomerRepo.EXPECT().
			Search(ctx, "John", pagination).
			Return(expectedResponse, nil)

		response, err := service.Search(ctx, "John", pagination)

		require.NoError(t, err)
		assert.Len(t, response.Customers, 1)
	})
}

func TestCustomerService_GetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful get orders", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListOrdersResponse{
			Orders: []*domain.Order{
				{ID: "order_1", CustomerID: "cust_abc123"},
				{ID: "order_2", CustomerID: "cust_abc123"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockOrderRepo.EXPECT().
			GetByCustomer(ctx, "cust_abc123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetOrders(ctx, "cust_abc123", pagination)

		require.NoError(t, err)
		assert.Len(t, response.Orders, 2)
	})
}

func TestCustomerService_AddAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("first address auto-defaults", func(t *testing.T) {
		existing := &domain.Customer{
			ID:        "cust_abc123",
			Email:     "john@example.com",
			Addresses: []domain.Address{}, // No addresses
		}

		newAddress := domain.Address{
			AddressLine1: "123 Main St",
			City:         "Mumbai",
			State:        "Maharashtra",
			PostalCode:   "400001",
			Country:      "India",
			IsDefault:    false, // Not marked as default
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 1)
				assert.True(t, customer.Addresses[0].IsDefault) // Auto-set to default
				assert.Contains(t, customer.Addresses[0].ID, "addr_")
				return nil
			})

		customer, err := service.AddAddress(ctx, "cust_abc123", newAddress, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})

	t.Run("second address marked default clears first", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{
					ID:           "addr_first",
					AddressLine1: "First St",
					IsDefault:    true,
				},
			},
		}

		newAddress := domain.Address{
			AddressLine1: "Second St",
			City:         "Delhi",
			IsDefault:    true, // Marking as default
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 2)
				assert.False(t, customer.Addresses[0].IsDefault) // First is no longer default
				assert.True(t, customer.Addresses[1].IsDefault)  // New one is default
				return nil
			})

		customer, err := service.AddAddress(ctx, "cust_abc123", newAddress, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})

	t.Run("second address not marked default keeps first default", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{
					ID:           "addr_first",
					AddressLine1: "First St",
					IsDefault:    true,
				},
			},
		}

		newAddress := domain.Address{
			AddressLine1: "Second St",
			City:         "Delhi",
			IsDefault:    false, // Not marking as default
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 2)
				assert.True(t, customer.Addresses[0].IsDefault)  // First stays default
				assert.False(t, customer.Addresses[1].IsDefault) // Second is not default
				return nil
			})

		customer, err := service.AddAddress(ctx, "cust_abc123", newAddress, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})
}

func TestCustomerService_UpdateAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{
					ID:           "addr_1",
					AddressLine1: "Old Address",
					City:         "Mumbai",
					IsDefault:    true,
				},
				{
					ID:           "addr_2",
					AddressLine1: "Second Address",
					City:         "Delhi",
					IsDefault:    false,
				},
			},
		}

		updatedAddress := domain.Address{
			AddressLine1: "New Address",
			City:         "Mumbai Updated",
			IsDefault:    true,
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 2)
				assert.Equal(t, "New Address", customer.Addresses[0].AddressLine1)
				assert.Equal(t, "Mumbai Updated", customer.Addresses[0].City)
				return nil
			})

		customer, err := service.UpdateAddress(ctx, "cust_abc123", "addr_1", updatedAddress, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})

	t.Run("address not found", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{ID: "addr_1", AddressLine1: "Main St"},
			},
		}

		updatedAddress := domain.Address{
			AddressLine1: "New Address",
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		customer, err := service.UpdateAddress(ctx, "cust_abc123", "addr_nonexistent", updatedAddress, "admin_123")

		assert.Nil(t, customer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Address not found")
	})
}

func TestCustomerService_RemoveAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful removal", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{ID: "addr_1", AddressLine1: "First St"},
				{ID: "addr_2", AddressLine1: "Second St"},
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Len(t, customer.Addresses, 1)
				assert.Equal(t, "addr_2", customer.Addresses[0].ID)
				return nil
			})

		customer, err := service.RemoveAddress(ctx, "cust_abc123", "addr_1", "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, customer)
	})

	t.Run("address not found", func(t *testing.T) {
		existing := &domain.Customer{
			ID:    "cust_abc123",
			Email: "john@example.com",
			Addresses: []domain.Address{
				{ID: "addr_1", AddressLine1: "First St"},
			},
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		customer, err := service.RemoveAddress(ctx, "cust_abc123", "addr_nonexistent", "admin_123")

		assert.Nil(t, customer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Address not found")
	})
}

func TestCustomerService_UpdateStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	log := logger.NewNoop()
	service := NewCustomerService(mockCustomerRepo, mockOrderRepo, log)
	ctx := context.Background()

	t.Run("successful stats update", func(t *testing.T) {
		existing := &domain.Customer{
			ID:          "cust_abc123",
			Email:       "john@example.com",
			TotalOrders: 5,
			TotalSpent:  50000.0,
		}

		mockCustomerRepo.EXPECT().
			GetByID(ctx, "cust_abc123").
			Return(existing, nil)

		mockCustomerRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, customer *domain.Customer) error {
				assert.Equal(t, 10, customer.TotalOrders)
				assert.Equal(t, 100000.0, customer.TotalSpent)
				return nil
			})

		err := service.UpdateStats(ctx, "cust_abc123", 10, 100000.0)

		require.NoError(t, err)
	})

	t.Run("customer not found", func(t *testing.T) {
		mockCustomerRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Customer"))

		err := service.UpdateStats(ctx, "nonexistent", 10, 100000.0)

		require.Error(t, err)
	})
}
