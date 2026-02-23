package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// MockPricingRuleRepository is a mock implementation of PricingRuleRepository
type MockPricingRuleRepository struct {
	mock.Mock
}

func (m *MockPricingRuleRepository) Create(ctx context.Context, rule *domain.PricingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockPricingRuleRepository) GetByID(ctx context.Context, id string) (*domain.PricingRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PricingRule), args.Error(1)
}

func (m *MockPricingRuleRepository) Update(ctx context.Context, rule *domain.PricingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockPricingRuleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPricingRuleRepository) List(ctx context.Context, req domain.ListPricingRulesRequest) (*domain.ListPricingRulesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListPricingRulesResponse), args.Error(1)
}

func (m *MockPricingRuleRepository) GetByScope(ctx context.Context, scopeType domain.PricingRuleScope, scopeID string) ([]*domain.PricingRule, error) {
	args := m.Called(ctx, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PricingRule), args.Error(1)
}

func (m *MockPricingRuleRepository) GetApplicableRules(ctx context.Context, categoryID string, productID *string, material *string) ([]*domain.PricingRule, error) {
	args := m.Called(ctx, categoryID, productID, material)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PricingRule), args.Error(1)
}

func (m *MockPricingRuleRepository) GetGlobalRule(ctx context.Context) (*domain.PricingRule, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PricingRule), args.Error(1)
}

// MockPriceQuoteRepository is a mock implementation of PriceQuoteRepository
type MockPriceQuoteRepository struct {
	mock.Mock
}

func (m *MockPriceQuoteRepository) Create(ctx context.Context, quote *domain.PriceQuote) error {
	args := m.Called(ctx, quote)
	return args.Error(0)
}

func (m *MockPriceQuoteRepository) GetByID(ctx context.Context, id string) (*domain.PriceQuote, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PriceQuote), args.Error(1)
}

func (m *MockPriceQuoteRepository) MarkAsUsed(ctx context.Context, id string, orderID string) error {
	args := m.Called(ctx, id, orderID)
	return args.Error(0)
}

// MockCategoryRepository is a mock implementation of CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListCategoriesResponse), args.Error(1)
}

func (m *MockCategoryRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}

// MockProductRepository is a mock implementation of ProductRepository
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(ctx context.Context, product *domain.Product, inventory *domain.Inventory) error {
	args := m.Called(ctx, product, inventory)
	return args.Error(0)
}

func (m *MockProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *MockProductRepository) Update(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListProductsResponse), args.Error(1)
}

func (m *MockProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Product), args.Error(1)
}

func (m *MockProductRepository) BatchUpdateSortOrder(ctx context.Context, products []*domain.Product) error {
	args := m.Called(ctx, products)
	return args.Error(0)
}

func (m *MockProductRepository) GetByCategoryAll(ctx context.Context, categoryID string) ([]*domain.Product, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Product), args.Error(1)
}

func (m *MockProductRepository) GetAttributeFilterOptions(ctx context.Context, categoryID string, attrNames []string) (map[string][]string, error) {
	args := m.Called(ctx, categoryID, attrNames)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]string), args.Error(1)
}

func TestPricingService_CalculatePrice_AreaBased(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	// Setup mocks
	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	// Create service
	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	// Setup test data
	category := &domain.Category{
		ID:   "cat_bedsheets",
		Name: "Bedsheets",
	}

	pricingRule := &domain.PricingRule{
		ID:           "rule_bedsheets_area",
		Name:         "Bedsheets Area Pricing",
		ScopeType:    domain.PricingRuleScopeCategory,
		ScopeID:      "cat_bedsheets",
		PricingType:  domain.PricingTypeAreaBased,
		BasePrice:    50000, // ₹500
		PricePerUnit: 35,    // ₹0.35 per sq inch
		Unit:         domain.PricingUnitSqInch,
		MaterialMultipliers: map[string]float64{
			"cotton": 1.0,
			"silk":   2.5,
		},
		AttributeSurcharges: []domain.AttributeSurcharge{
			{
				AttributeName:  "thread_count",
				AttributeValue: "600",
				SurchargeType:  domain.SurchargeTypeFixed,
				SurchargeValue: 50000, // ₹500
			},
		},
		Priority: 100,
		IsActive: true,
	}

	// Setup expectations
	categoryRepo.On("GetByID", ctx, "cat_bedsheets").Return(category, nil)

	material := "silk"
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_bedsheets", (*string)(nil), &material).Return([]*domain.PricingRule{pricingRule}, nil)
	priceQuoteRepo.On("Create", ctx, mock.AnythingOfType("*domain.PriceQuote")).Return(nil)

	// Test calculation
	req := domain.CalculatePriceRequest{
		CategoryID: "cat_bedsheets",
		Dimensions: &domain.Dimensions{
			Length: 100,
			Width:  90,
			Unit:   "inches",
		},
		Attributes: map[string]interface{}{
			"material":     "silk",
			"thread_count": "600",
		},
		Quantity: 1,
	}

	result, err := svc.CalculatePrice(ctx, req)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.QuoteID)
	assert.Equal(t, pricingRule.ID, result.PricingRuleID)

	// Verify price breakdown
	breakdown := result.PriceBreakdown
	assert.Equal(t, float64(9000), breakdown.Area) // 100 x 90
	assert.Equal(t, float64(2.5), breakdown.MaterialMultiplier)
	assert.Equal(t, 1, breakdown.Quantity)

	// Base cost = 50000 + (9000 * 35) = 50000 + 315000 = 365000
	assert.Equal(t, int64(365000), breakdown.BaseCost)

	// Material cost = 365000 * 2.5 = 912500
	assert.Equal(t, int64(912500), breakdown.MaterialCost)

	// Surcharges = 50000 (thread count 600)
	assert.Equal(t, int64(50000), breakdown.SurchargesTotal)

	// Total = 912500 + 50000 = 962500 (₹9,625)
	assert.Equal(t, int64(962500), breakdown.Total)

	pricingRuleRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	priceQuoteRepo.AssertExpectations(t)
}

func TestPricingService_CalculatePrice_NilDimensions(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	category := &domain.Category{
		ID:   "cat_bedsheets",
		Name: "Bedsheets",
	}

	categoryRepo.On("GetByID", ctx, "cat_bedsheets").Return(category, nil)

	// Test with nil dimensions
	req := domain.CalculatePriceRequest{
		CategoryID: "cat_bedsheets",
		Dimensions: nil, // nil dimensions should fail validation
		Attributes: map[string]interface{}{
			"material": "cotton",
		},
		Quantity: 1,
	}

	result, err := svc.CalculatePrice(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Dimensions are required")
}

func TestPricingService_CalculatePrice_NoPricingRule(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	category := &domain.Category{ID: "cat_orphan", Name: "Orphan"}

	categoryRepo.On("GetByID", ctx, "cat_orphan").Return(category, nil)

	cotton := "cotton"
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_orphan", (*string)(nil), &cotton).
		Return([]*domain.PricingRule{}, nil)

	req := domain.CalculatePriceRequest{
		CategoryID: "cat_orphan",
		Dimensions: &domain.Dimensions{Length: 10, Width: 10, Unit: "inches"},
		Attributes: map[string]interface{}{"material": "cotton"},
		Quantity:   1,
	}

	result, err := svc.CalculatePrice(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "No applicable pricing rule found")
}

func TestPricingService_CreateRule(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	// Setup mocks
	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	// Create service
	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	pricingRuleRepo.On("Create", ctx, mock.AnythingOfType("*domain.PricingRule")).Return(nil)

	req := domain.CreatePricingRuleRequest{
		Name:         "Test Rule",
		ScopeType:    domain.PricingRuleScopeGlobal,
		PricingType:  domain.PricingTypeFixed,
		BasePrice:    100000,
		Priority:     10,
		IsActive:     true,
	}

	rule, err := svc.CreateRule(ctx, req, "user_123")

	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, "Test Rule", rule.Name)
	assert.Equal(t, domain.PricingRuleScopeGlobal, rule.ScopeType)
	assert.Equal(t, int64(100000), rule.BasePrice)
	assert.Equal(t, "user_123", rule.CreatedBy)
	assert.NotEmpty(t, rule.ID)

	pricingRuleRepo.AssertExpectations(t)
}

func TestPricingService_BulkCalculatePrice(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	// Setup mocks
	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	// Create service
	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	category := &domain.Category{
		ID:   "cat_bedsheets",
		Name: "Bedsheets",
	}

	pricingRule := &domain.PricingRule{
		ID:           "rule_test",
		PricingType:  domain.PricingTypeAreaBased,
		BasePrice:    50000,
		PricePerUnit: 35,
		Unit:         domain.PricingUnitSqInch,
		MaterialMultipliers: map[string]float64{
			"cotton": 1.0,
			"silk":   2.5,
		},
		Priority: 100,
		IsActive: true,
	}

	// Setup expectations for multiple calculations
	categoryRepo.On("GetByID", ctx, "cat_bedsheets").Return(category, nil)

	cotton := "cotton"
	silk := "silk"
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_bedsheets", (*string)(nil), &cotton).Return([]*domain.PricingRule{pricingRule}, nil)
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_bedsheets", (*string)(nil), &silk).Return([]*domain.PricingRule{pricingRule}, nil)
	priceQuoteRepo.On("Create", ctx, mock.AnythingOfType("*domain.PriceQuote")).Return(nil)

	req := domain.BulkCalculatePriceRequest{
		CategoryID: "cat_bedsheets",
		Configurations: []domain.PriceConfiguration{
			{
				Dimensions: &domain.Dimensions{Length: 75, Width: 36, Unit: "inches"},
				Attributes: map[string]interface{}{"material": "cotton"},
				Quantity:   1,
			},
			{
				Dimensions: &domain.Dimensions{Length: 100, Width: 90, Unit: "inches"},
				Attributes: map[string]interface{}{"material": "silk"},
				Quantity:   2,
			},
		},
	}

	result, err := svc.BulkCalculatePrice(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Calculations, 2)
	assert.NotEmpty(t, result.QuoteID)
	assert.True(t, result.QuoteValidUntil.After(time.Now()))

	// First calculation (cotton, smaller area)
	assert.Empty(t, result.Calculations[0].Error)
	assert.Greater(t, result.Calculations[0].Price, int64(0))

	// Second calculation (silk, larger area, quantity 2)
	assert.Empty(t, result.Calculations[1].Error)
	assert.Greater(t, result.Calculations[1].Price, result.Calculations[0].Price)
}

func TestPricingService_BulkCalculatePrice_MaxLimit(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	// Build 11 configurations (exceeds max of 10)
	configs := make([]domain.PriceConfiguration, 11)
	for i := range configs {
		configs[i] = domain.PriceConfiguration{
			Dimensions: &domain.Dimensions{Length: 10, Width: 10, Unit: "inches"},
			Attributes: map[string]interface{}{"material": "cotton"},
			Quantity:   1,
		}
	}

	req := domain.BulkCalculatePriceRequest{
		CategoryID:     "cat_123",
		Configurations: configs,
	}

	result, err := svc.BulkCalculatePrice(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Maximum 10")
}

func TestPricingService_GetRule(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	t.Run("found", func(t *testing.T) {
		expected := &domain.PricingRule{ID: "rule_123", Name: "Test Rule"}
		pricingRuleRepo.On("GetByID", ctx, "rule_123").Return(expected, nil)

		rule, err := svc.GetRule(ctx, "rule_123")
		assert.NoError(t, err)
		assert.Equal(t, "rule_123", rule.ID)
	})

	t.Run("not found", func(t *testing.T) {
		pricingRuleRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.NotFound("Rule"))

		rule, err := svc.GetRule(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, rule)
	})
}

func TestPricingService_UpdateRule(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	t.Run("successful update", func(t *testing.T) {
		existing := &domain.PricingRule{
			ID:        "rule_123",
			Name:      "Old Name",
			BasePrice: 10000,
		}

		pricingRuleRepo.On("GetByID", ctx, "rule_123").Return(existing, nil).Once()
		pricingRuleRepo.On("Update", ctx, mock.AnythingOfType("*domain.PricingRule")).Return(nil).Once()

		newName := "New Name"
		newPrice := int64(20000)
		req := domain.UpdatePricingRuleRequest{
			Name:      &newName,
			BasePrice: &newPrice,
		}

		rule, err := svc.UpdateRule(ctx, "rule_123", req, "admin_1")

		assert.NoError(t, err)
		assert.Equal(t, "New Name", rule.Name)
		assert.Equal(t, int64(20000), rule.BasePrice)
		assert.Equal(t, "admin_1", rule.UpdatedBy)
	})

	t.Run("rule not found", func(t *testing.T) {
		pricingRuleRepo.On("GetByID", ctx, "rule_nonexistent").Return(nil, errors.NotFound("Rule"))

		rule, err := svc.UpdateRule(ctx, "rule_nonexistent", domain.UpdatePricingRuleRequest{}, "admin_1")

		assert.Error(t, err)
		assert.Nil(t, rule)
	})
}

func TestPricingService_DeleteRule(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	pricingRuleRepo.On("Delete", ctx, "rule_123").Return(nil)

	err := svc.DeleteRule(ctx, "rule_123")
	assert.NoError(t, err)
}

func TestPricingService_ListRules(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	req := domain.ListPricingRulesRequest{
		PaginationRequest: domain.PaginationRequest{Limit: 10},
	}

	expected := &domain.ListPricingRulesResponse{
		Rules: []*domain.PricingRule{
			{ID: "rule_1", Name: "Rule A"},
			{ID: "rule_2", Name: "Rule B"},
		},
		Pagination: domain.PaginationResponse{Limit: 10, HasMore: false},
	}

	pricingRuleRepo.On("List", ctx, req).Return(expected, nil)

	resp, err := svc.ListRules(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, resp.Rules, 2)
}

func TestPricingService_GetQuote(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	t.Run("valid quote", func(t *testing.T) {
		quote := &domain.PriceQuote{
			ID:              "quote_123",
			CalculatedPrice: 100000,
			ValidUntil:      time.Now().Add(1 * time.Hour),
		}

		priceQuoteRepo.On("GetByID", ctx, "quote_123").Return(quote, nil)

		result, err := svc.GetQuote(ctx, "quote_123")
		assert.NoError(t, err)
		assert.Equal(t, int64(100000), result.CalculatedPrice)
	})

	t.Run("expired quote", func(t *testing.T) {
		quote := &domain.PriceQuote{
			ID:              "quote_expired",
			CalculatedPrice: 100000,
			ValidUntil:      time.Now().Add(-1 * time.Hour), // expired
		}

		priceQuoteRepo.On("GetByID", ctx, "quote_expired").Return(quote, nil)

		result, err := svc.GetQuote(ctx, "quote_expired")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("quote not found", func(t *testing.T) {
		priceQuoteRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.NotFound("Quote"))

		result, err := svc.GetQuote(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPricingService_GetDimensionOptions(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	t.Run("returns pricing attributes with surcharges", func(t *testing.T) {
		category := &domain.Category{
			ID:   "cat_123",
			Name: "Bedsheets",
			OwnAttributes: []domain.CategoryAttribute{
				{
					Name:  "border_type",
					Label: "Border",
					Type:  "SELECT",
					Options: []domain.AttributeOption{
						{Value: "plain", Label: "Plain", Surcharge: 0},
						{Value: "embroidered", Label: "Embroidered", Surcharge: 50000},
					},
				},
				{
					Name:  "color",
					Label: "Color",
					Type:  "SELECT",
					Options: []domain.AttributeOption{
						{Value: "white", Label: "White", Surcharge: 0},
						{Value: "red", Label: "Red", Surcharge: 0},
					},
				},
			},
		}

		categoryRepo.On("GetByID", ctx, "cat_123").Return(category, nil).Once()

		resp, err := svc.GetDimensionOptions(ctx, "cat_123")

		assert.NoError(t, err)
		assert.Equal(t, "cat_123", resp.CategoryID)
		// Only border_type has surcharges, so only it should appear
		assert.Len(t, resp.PricingAttributes, 1)
		assert.Equal(t, "border_type", resp.PricingAttributes[0].Name)
	})

	t.Run("category not found", func(t *testing.T) {
		categoryRepo.On("GetByID", ctx, "cat_nonexistent").Return(nil, errors.NotFound("Category"))

		resp, err := svc.GetDimensionOptions(ctx, "cat_nonexistent")
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestPricingService_CalculatePrice_FixedPricing(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	category := &domain.Category{ID: "cat_fixed", Name: "Fixed"}
	rule := &domain.PricingRule{
		ID:          "rule_fixed",
		PricingType: domain.PricingTypeFixed,
		BasePrice:   200000, // ₹2000
		Priority:    100,
		IsActive:    true,
	}

	categoryRepo.On("GetByID", ctx, "cat_fixed").Return(category, nil)

	material := ""
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_fixed", (*string)(nil), &material).
		Return([]*domain.PricingRule{rule}, nil)
	priceQuoteRepo.On("Create", ctx, mock.AnythingOfType("*domain.PriceQuote")).Return(nil)

	req := domain.CalculatePriceRequest{
		CategoryID: "cat_fixed",
		Dimensions: &domain.Dimensions{Length: 10, Width: 10, Unit: "inches"},
		Attributes: map[string]interface{}{},
		Quantity:   3,
	}

	result, err := svc.CalculatePrice(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(200000), result.PriceBreakdown.BaseCost)
	assert.Equal(t, int64(600000), result.PriceBreakdown.Total) // 200000 * 3
}

func TestPricingService_CalculatePrice_LengthBased(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	category := &domain.Category{ID: "cat_length", Name: "By Length"}
	rule := &domain.PricingRule{
		ID:           "rule_length",
		PricingType:  domain.PricingTypeLengthBased,
		BasePrice:    10000, // ₹100
		PricePerUnit: 500,   // ₹5 per unit of length
		Priority:     100,
		IsActive:     true,
	}

	categoryRepo.On("GetByID", ctx, "cat_length").Return(category, nil)

	material := ""
	pricingRuleRepo.On("GetApplicableRules", ctx, "cat_length", (*string)(nil), &material).
		Return([]*domain.PricingRule{rule}, nil)
	priceQuoteRepo.On("Create", ctx, mock.AnythingOfType("*domain.PriceQuote")).Return(nil)

	req := domain.CalculatePriceRequest{
		CategoryID: "cat_length",
		Dimensions: &domain.Dimensions{Length: 50, Width: 10, Unit: "inches"},
		Attributes: map[string]interface{}{},
		Quantity:   1,
	}

	result, err := svc.CalculatePrice(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// BaseCost = 10000 + (50 * 500) = 10000 + 25000 = 35000
	assert.Equal(t, int64(35000), result.PriceBreakdown.BaseCost)
	assert.Equal(t, int64(35000), result.PriceBreakdown.Total)
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name     string
		paise    int64
		expected string
	}{
		{"whole rupees", 100000, "₹1000.00"},
		{"with paise", 12345, "₹123.45"},
		{"zero", 0, "₹0.00"},
		{"single paise", 1, "₹0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatPrice(tt.paise))
		})
	}
}
