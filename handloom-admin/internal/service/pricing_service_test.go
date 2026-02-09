package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/handloom/admin/internal/domain"
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

func (m *MockProductRepository) CreateWithAttributeIndexes(ctx context.Context, product *domain.Product, searchableAttrs map[string][]string) error {
	args := m.Called(ctx, product, searchableAttrs)
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

func (m *MockProductRepository) UpdateWithAttributeIndexes(ctx context.Context, product *domain.Product, oldSearchableAttrs, newSearchableAttrs map[string][]string) error {
	args := m.Called(ctx, product, oldSearchableAttrs, newSearchableAttrs)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteWithAttributeIndexes(ctx context.Context, id string, sku string, searchableAttrs map[string][]string) error {
	args := m.Called(ctx, id, sku, searchableAttrs)
	return args.Error(0)
}

func (m *MockProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListProductsResponse), args.Error(1)
}

func (m *MockProductRepository) GetByCategory(ctx context.Context, categoryID string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	args := m.Called(ctx, categoryID, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListProductsResponse), args.Error(1)
}

func (m *MockProductRepository) FilterByAttributes(ctx context.Context, categoryID string, filters map[string][]string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	args := m.Called(ctx, categoryID, filters, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ListProductsResponse), args.Error(1)
}

func (m *MockProductRepository) UpdateInventory(ctx context.Context, id string, quantity, reservedQty, availableQty int) error {
	args := m.Called(ctx, id, quantity, reservedQty, availableQty)
	return args.Error(0)
}

func (m *MockProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Product), args.Error(1)
}

func (m *MockProductRepository) AddAttributeValues(ctx context.Context, categoryID string, attrValues map[string][]string) error {
	args := m.Called(ctx, categoryID, attrValues)
	return args.Error(0)
}

func (m *MockProductRepository) GetAttributeValues(ctx context.Context, categoryID string) (map[string][]string, error) {
	args := m.Called(ctx, categoryID)
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

func TestPricingService_CalculatePrice_DimensionValidation(t *testing.T) {
	ctx := context.Background()
	log := logger.New(true)

	// Setup mocks
	pricingRuleRepo := new(MockPricingRuleRepository)
	priceQuoteRepo := new(MockPriceQuoteRepository)
	categoryRepo := new(MockCategoryRepository)
	productRepo := new(MockProductRepository)

	// Create service
	svc := NewPricingService(pricingRuleRepo, priceQuoteRepo, categoryRepo, productRepo, log, 24)

	// Setup test data with dimension constraints
	category := &domain.Category{
		ID:   "cat_bedsheets",
		Name: "Bedsheets",
	}

	categoryRepo.On("GetByID", ctx, "cat_bedsheets").Return(category, nil)

	// Test with length out of range
	req := domain.CalculatePriceRequest{
		CategoryID: "cat_bedsheets",
		Dimensions: &domain.Dimensions{
			Length: 150, // Exceeds max of 120
			Width:  90,
			Unit:   "inches",
		},
		Attributes: map[string]interface{}{
			"material": "cotton",
		},
		Quantity: 1,
	}

	result, err := svc.CalculatePrice(ctx, req)

	// Should fail validation
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "DIMENSION_OUT_OF_RANGE")
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
