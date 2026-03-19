// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// PricingService implements domain.PricingService
type PricingService struct {
	pricingRuleRepo  domain.PricingRuleRepository
	priceQuoteRepo   domain.PriceQuoteRepository
	categoryRepo     domain.CategoryRepository
	productRepo      domain.ProductRepository
	quoteValidityHrs int
}

// NewPricingService creates a new PricingService
func NewPricingService(
	pricingRuleRepo domain.PricingRuleRepository,
	priceQuoteRepo domain.PriceQuoteRepository,
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	quoteValidityHrs int,
) *PricingService {
	return &PricingService{
		pricingRuleRepo:  pricingRuleRepo,
		priceQuoteRepo:   priceQuoteRepo,
		categoryRepo:     categoryRepo,
		productRepo:      productRepo,
		quoteValidityHrs: quoteValidityHrs,
	}
}

// CreateRule creates a new pricing rule
func (s *PricingService) CreateRule(ctx context.Context, req domain.CreatePricingRuleRequest, createdBy string) (*domain.PricingRule, error) {
	rule := &domain.PricingRule{
		ID:                  "rule_" + uuid.New().String()[:8],
		Name:                req.Name,
		Description:         req.Description,
		ScopeType:           req.ScopeType,
		ScopeID:             req.ScopeID,
		CategoryID:          req.CategoryID,
		MaterialName:        req.MaterialName,
		PricingType:         req.PricingType,
		BasePrice:           req.BasePrice,
		PricePerUnit:        req.PricePerUnit,
		Unit:                req.Unit,
		MaterialMultipliers: req.MaterialMultipliers,
		AttributeSurcharges: req.AttributeSurcharges,
		Tiers:               req.Tiers,
		Formula:             req.Formula,
		MinArea:             req.MinArea,
		MaxArea:             req.MaxArea,
		MinOrderValue:       req.MinOrderValue,
		Priority:            req.Priority,
		IsActive:            req.IsActive,
		ValidFrom:           req.ValidFrom,
		ValidUntil:          req.ValidUntil,
	}
	rule.CreatedBy = createdBy

	if err := s.pricingRuleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Created pricing rule", "rule_id", rule.ID)
	return rule, nil
}

// GetRule retrieves a pricing rule by ID
func (s *PricingService) GetRule(ctx context.Context, id string) (*domain.PricingRule, error) {
	return s.pricingRuleRepo.GetByID(ctx, id)
}

// UpdateRule updates an existing pricing rule
func (s *PricingService) UpdateRule(ctx context.Context, id string, req domain.UpdatePricingRuleRequest, updatedBy string) (*domain.PricingRule, error) {
	rule, err := s.pricingRuleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.BasePrice != nil {
		rule.BasePrice = *req.BasePrice
	}
	if req.PricePerUnit != nil {
		rule.PricePerUnit = *req.PricePerUnit
	}
	if req.MaterialMultipliers != nil {
		rule.MaterialMultipliers = req.MaterialMultipliers
	}
	if req.AttributeSurcharges != nil {
		rule.AttributeSurcharges = req.AttributeSurcharges
	}
	if req.Tiers != nil {
		rule.Tiers = req.Tiers
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.ValidFrom != nil {
		rule.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		rule.ValidUntil = req.ValidUntil
	}
	rule.UpdatedBy = updatedBy

	if err := s.pricingRuleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// DeleteRule deletes a pricing rule
func (s *PricingService) DeleteRule(ctx context.Context, id string) error {
	return s.pricingRuleRepo.Delete(ctx, id)
}

// ListRules retrieves pricing rules with filters
func (s *PricingService) ListRules(ctx context.Context, req domain.ListPricingRulesRequest) (*domain.ListPricingRulesResponse, error) {
	return s.pricingRuleRepo.List(ctx, req)
}

// GetRulesForCategory retrieves all applicable rules for a category
func (s *PricingService) GetRulesForCategory(ctx context.Context, categoryID string) (*domain.CategoryPricingRulesResponse, error) {
	// Verify category exists
	if _, err := s.categoryRepo.GetByID(ctx, categoryID); err != nil {
		return nil, err
	}

	response := &domain.CategoryPricingRulesResponse{
		CategoryRules: []*domain.PricingRuleSummary{},
		ParentRules:   []*domain.PricingRuleSummary{},
		GlobalRules:   []*domain.PricingRuleSummary{},
	}

	// Get category-specific rules
	categoryRules, err := s.pricingRuleRepo.GetByScope(ctx, domain.PricingRuleScopeCategory, categoryID)
	if err != nil {
		return nil, err
	}
	for _, rule := range categoryRules {
		response.CategoryRules = append(response.CategoryRules, &domain.PricingRuleSummary{
			ID:       rule.ID,
			Name:     rule.Name,
			Priority: rule.Priority,
			IsActive: rule.IsActive,
		})
	}

	// Get global rules
	globalRules, err := s.pricingRuleRepo.GetByScope(ctx, domain.PricingRuleScopeGlobal, "")
	if err != nil {
		return nil, err
	}
	for _, rule := range globalRules {
		response.GlobalRules = append(response.GlobalRules, &domain.PricingRuleSummary{
			ID:       rule.ID,
			Name:     rule.Name,
			Priority: rule.Priority,
			IsActive: rule.IsActive,
		})
	}

	// Determine effective rule
	effectiveRule := s.findEffectiveRule(categoryRules, globalRules)
	if effectiveRule != nil {
		response.EffectiveRule = &domain.PricingRuleSummary{
			ID:       effectiveRule.ID,
			Name:     effectiveRule.Name,
			Priority: effectiveRule.Priority,
			IsActive: effectiveRule.IsActive,
			Reason:   "Highest priority active rule",
		}
	}

	return response, nil
}

// findEffectiveRule finds the highest priority active rule
func (s *PricingService) findEffectiveRule(categoryRules, globalRules []*domain.PricingRule) *domain.PricingRule {
	allRules := append(categoryRules, globalRules...)

	var activeRules []*domain.PricingRule
	now := time.Now()
	for _, rule := range allRules {
		if !rule.IsActive {
			continue
		}
		if rule.ValidFrom != nil && rule.ValidFrom.After(now) {
			continue
		}
		if rule.ValidUntil != nil && rule.ValidUntil.Before(now) {
			continue
		}
		activeRules = append(activeRules, rule)
	}

	if len(activeRules) == 0 {
		return nil
	}

	// Sort by priority (descending)
	sort.Slice(activeRules, func(i, j int) bool {
		return activeRules[i].Priority > activeRules[j].Priority
	})

	return activeRules[0]
}

// CalculatePrice calculates price for given dimensions and attributes
func (s *PricingService) CalculatePrice(ctx context.Context, req domain.CalculatePriceRequest) (*domain.CalculatePriceResponse, error) {
	// Validate and set defaults
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	// Get category
	var categoryID string
	var category *domain.Category

	if req.ProductID != nil {
		product, err := s.productRepo.GetByID(ctx, *req.ProductID)
		if err != nil {
			return nil, err
		}
		categoryID = product.CategoryID
	} else {
		categoryID = req.CategoryID
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	// Validate dimensions
	if dimErr := s.validateDimensions(category, req.Dimensions); dimErr != nil {
		return nil, dimErr
	}

	// Get applicable pricing rule
	material, _ := req.Attributes["material"].(string)
	rules, err := s.pricingRuleRepo.GetApplicableRules(ctx, categoryID, req.ProductID, &material)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, errors.New(errors.ErrCodePricingRuleNotFound, "No applicable pricing rule found")
	}

	// Sort by priority and get highest
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	rule := rules[0]

	// Calculate price
	breakdown, err := s.calculatePriceBreakdown(rule, req.Dimensions, req.Attributes, req.Quantity)
	if err != nil {
		return nil, err
	}

	// Check minimum order value
	if rule.MinOrderValue > 0 && breakdown.Total < rule.MinOrderValue {
		return nil, errors.New(errors.ErrCodeMinOrderValue, fmt.Sprintf("Minimum order value is %s", formatPrice(rule.MinOrderValue))).
			WithDetails(map[string]interface{}{
				"calculated_price": breakdown.Total,
				"min_required":     rule.MinOrderValue,
			})
	}

	// Create quote
	quoteID := "quote_" + uuid.New().String()[:12]
	validUntil := time.Now().Add(time.Duration(s.quoteValidityHrs) * time.Hour)

	quote := &domain.PriceQuote{
		ID:              quoteID,
		CategoryID:      categoryID,
		ProductID:       req.ProductID,
		Dimensions:      req.Dimensions,
		Attributes:      req.Attributes,
		Quantity:        req.Quantity,
		PricingRuleID:   rule.ID,
		CalculatedPrice: breakdown.Total,
		PriceBreakdown:  breakdown,
		ValidUntil:      validUntil,
		CreatedAt:       time.Now(),
	}

	if err := s.priceQuoteRepo.Create(ctx, quote); err != nil {
		slog.ErrorContext(ctx, "Failed to save price quote", "error", err)
		// Continue even if quote save fails
	}

	return &domain.CalculatePriceResponse{
		PriceBreakdown: breakdown,
		FormattedPrice: &domain.FormattedPrice{
			Subtotal: formatPrice(breakdown.SubtotalPerUnit * int64(breakdown.Quantity)),
			Total:    formatPrice(breakdown.Total),
			Currency: "INR",
		},
		PricingRuleID:   rule.ID,
		QuoteID:         quoteID,
		QuoteValidUntil: validUntil,
	}, nil
}

// validateDimensions validates that dimensions are provided
func (s *PricingService) validateDimensions(category *domain.Category, dimensions *domain.Dimensions) error {
	if dimensions == nil {
		return errors.Validation("Dimensions are required")
	}

	return nil
}

// calculatePriceBreakdown calculates the detailed price breakdown
func (s *PricingService) calculatePriceBreakdown(rule *domain.PricingRule, dimensions *domain.Dimensions, attributes map[string]interface{}, quantity int) (*domain.PriceBreakdown, error) {
	breakdown := &domain.PriceBreakdown{
		Quantity: quantity,
	}

	var baseCost int64

	switch rule.PricingType {
	case domain.PricingTypeFixed:
		baseCost = rule.BasePrice

	case domain.PricingTypeAreaBased:
		area := dimensions.Length * dimensions.Width
		breakdown.Area = area
		breakdown.AreaUnit = "sq_" + dimensions.Unit

		// Convert to pricing unit if needed
		areaInUnit := s.convertArea(area, dimensions.Unit, rule.Unit)
		baseCost = rule.BasePrice + int64(areaInUnit*float64(rule.PricePerUnit))

	case domain.PricingTypeLengthBased:
		baseCost = rule.BasePrice + int64(dimensions.Length*float64(rule.PricePerUnit))

	case domain.PricingTypeTiered:
		area := dimensions.Length * dimensions.Width
		breakdown.Area = area
		breakdown.AreaUnit = "sq_" + dimensions.Unit

		// Find applicable tier
		for _, tier := range rule.Tiers {
			if area >= tier.MinValue && area <= tier.MaxValue {
				baseCost = rule.BasePrice + int64(area*float64(tier.PricePerUnit))
				break
			}
		}
		if baseCost == 0 {
			baseCost = rule.BasePrice
		}

	default:
		baseCost = rule.BasePrice
	}

	breakdown.BaseCost = baseCost

	// Apply material multiplier
	materialCost := baseCost
	if material, ok := attributes["material"].(string); ok && len(rule.MaterialMultipliers) > 0 {
		if multiplier, exists := rule.MaterialMultipliers[material]; exists {
			breakdown.MaterialMultiplier = multiplier
			materialCost = int64(float64(baseCost) * multiplier)
		}
	}
	breakdown.MaterialCost = materialCost

	// Apply attribute surcharges
	var totalSurcharges int64
	for _, surcharge := range rule.AttributeSurcharges {
		if attrValue, ok := attributes[surcharge.AttributeName]; ok {
			attrStr, _ := attrValue.(string)
			if attrStr == surcharge.AttributeValue {
				var amount int64
				if surcharge.SurchargeType == domain.SurchargeTypeFixed {
					amount = surcharge.SurchargeValue
				} else {
					// Percentage
					amount = int64(float64(materialCost) * float64(surcharge.SurchargeValue) / 10000)
				}
				breakdown.Surcharges = append(breakdown.Surcharges, domain.SurchargeDetail{
					Attribute: surcharge.AttributeName,
					Value:     surcharge.AttributeValue,
					Amount:    amount,
				})
				totalSurcharges += amount
			}
		}
	}
	breakdown.SurchargesTotal = totalSurcharges

	// Calculate totals
	breakdown.SubtotalPerUnit = materialCost + totalSurcharges
	breakdown.Total = breakdown.SubtotalPerUnit * int64(quantity)

	return breakdown, nil
}

// convertArea converts area from one unit to another
func (s *PricingService) convertArea(area float64, fromUnit string, toUnit domain.PricingUnit) float64 {
	// First convert to square inches
	var sqInches float64
	switch fromUnit {
	case "inches", "inch":
		sqInches = area
	case "cm":
		sqInches = area / 6.4516 // 1 sq inch = 6.4516 sq cm
	case "feet", "foot":
		sqInches = area * 144 // 1 sq foot = 144 sq inches
	case "meters", "meter":
		sqInches = area * 1550.0031 // 1 sq meter = 1550 sq inches
	default:
		sqInches = area
	}

	// Then convert to target unit
	switch toUnit {
	case domain.PricingUnitSqInch:
		return sqInches
	case domain.PricingUnitSqFoot:
		return sqInches / 144
	case domain.PricingUnitSqCm:
		return sqInches * 6.4516
	case domain.PricingUnitSqMeter:
		return sqInches / 1550.0031
	default:
		return sqInches
	}
}

// GetDimensionOptions retrieves dimension options for a category
func (s *PricingService) GetDimensionOptions(ctx context.Context, categoryID string) (*domain.DimensionOptionsResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	response := &domain.DimensionOptionsResponse{
		CategoryID:   category.ID,
		CategoryName: category.Name,
	}

	// Get pricing attributes from category (attributes with surcharges on their options)
	for _, attr := range category.OwnAttributes {
		pricingAttr := domain.PricingAttribute{
			Name:  attr.Name,
			Label: attr.Label,
			Type:  attr.Type,
		}
		hasSurcharges := false
		for _, opt := range attr.Options {
			pricingAttr.Options = append(pricingAttr.Options, domain.PricingAttributeOption{
				Value:     opt.Value,
				Label:     opt.Label,
				Surcharge: opt.Surcharge,
			})
			if opt.Surcharge > 0 {
				hasSurcharges = true
			}
		}
		if hasSurcharges {
			response.PricingAttributes = append(response.PricingAttributes, pricingAttr)
		}
	}

	return response, nil
}

// BulkCalculatePrice calculates prices for multiple configurations
func (s *PricingService) BulkCalculatePrice(ctx context.Context, req domain.BulkCalculatePriceRequest) (*domain.BulkCalculatePriceResponse, error) {
	if len(req.Configurations) > 10 {
		return nil, errors.Validation("Maximum 10 configurations allowed")
	}

	response := &domain.BulkCalculatePriceResponse{
		Calculations:    make([]domain.BulkCalculationResult, len(req.Configurations)),
		QuoteID:         "quote_bulk_" + uuid.New().String()[:8],
		QuoteValidUntil: time.Now().Add(time.Duration(s.quoteValidityHrs) * time.Hour),
	}

	for i, config := range req.Configurations {
		result := domain.BulkCalculationResult{
			ConfigurationIndex: i,
			Dimensions:         config.Dimensions,
			Attributes:         config.Attributes,
			Quantity:           config.Quantity,
		}

		if config.Quantity < 1 {
			config.Quantity = 1
		}

		calcReq := domain.CalculatePriceRequest{
			CategoryID: req.CategoryID,
			Dimensions: config.Dimensions,
			Attributes: config.Attributes,
			Quantity:   config.Quantity,
		}

		calcResp, err := s.CalculatePrice(ctx, calcReq)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Price = calcResp.PriceBreakdown.Total
			result.FormattedPrice = calcResp.FormattedPrice.Total
			result.Quantity = config.Quantity
		}

		response.Calculations[i] = result
	}

	return response, nil
}

// GetQuote retrieves a price quote by ID
func (s *PricingService) GetQuote(ctx context.Context, quoteID string) (*domain.PriceQuote, error) {
	quote, err := s.priceQuoteRepo.GetByID(ctx, quoteID)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(quote.ValidUntil) {
		return nil, errors.New(errors.ErrCodeQuoteExpired, "Price quote has expired")
	}

	return quote, nil
}

// formatPrice formats price in paise to INR string
func formatPrice(paise int64) string {
	rupees := float64(paise) / 100
	return fmt.Sprintf("₹%.2f", math.Round(rupees*100)/100)
}

// Ensure interface compliance
var _ domain.PricingService = (*PricingService)(nil)
