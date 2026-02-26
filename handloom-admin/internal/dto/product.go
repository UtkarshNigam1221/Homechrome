package dto

import "github.com/handloom/admin/internal/domain"

// CreateProductRequest represents the product creation request.
type CreateProductRequest struct {
	Name                  string                 `json:"name" validate:"required"`
	SKU                   string                 `json:"sku" validate:"required"`
	CategoryID            string                 `json:"category_id" validate:"required"`
	Description           string                 `json:"description,omitempty"`
	BasePrice             int64                  `json:"base_price" validate:"required,gt=0"`
	SellingPrice          int64                  `json:"selling_price" validate:"required,gt=0"`
	CostPrice             int64                  `json:"cost_price,omitempty"`
	Dimensions            *domain.Dimensions     `json:"dimensions,omitempty"`
	Weight                int                    `json:"weight,omitempty"`
	AllowCustomDimensions bool                   `json:"allow_custom_dimensions"`
	PricingRuleID         *string                `json:"pricing_rule_id,omitempty"`
	Attributes            map[string]interface{} `json:"attributes,omitempty"`
	Material              string                 `json:"material,omitempty"`
	Color                 string                 `json:"color,omitempty"`
	WeaveType             string                 `json:"weave_type,omitempty"`
	Origin                string                 `json:"origin,omitempty"`
	CraftType             string                 `json:"craft_type,omitempty"`
	Images                []domain.ProductImage  `json:"images,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	InitialStock          int                    `json:"initial_stock"`
	LowStockThreshold     int                    `json:"low_stock_threshold"`
}

// ToDomain converts DTO to domain request.
func (r *CreateProductRequest) ToDomain() domain.CreateProductRequest {
	return domain.CreateProductRequest{
		Name:                  r.Name,
		SKU:                   r.SKU,
		CategoryID:            r.CategoryID,
		Description:           r.Description,
		BasePrice:             r.BasePrice,
		SellingPrice:          r.SellingPrice,
		CostPrice:             r.CostPrice,
		Dimensions:            r.Dimensions,
		Weight:                r.Weight,
		AllowCustomDimensions: r.AllowCustomDimensions,
		PricingRuleID:         r.PricingRuleID,
		Attributes:            r.Attributes,
		Material:              r.Material,
		Color:                 r.Color,
		WeaveType:             r.WeaveType,
		Origin:                r.Origin,
		CraftType:             r.CraftType,
		Images:                r.Images,
		Tags:                  r.Tags,
		InitialStock:          r.InitialStock,
		LowStockThreshold:     r.LowStockThreshold,
	}
}

// UpdateProductRequest represents the product update request.
type UpdateProductRequest struct {
	Name                  *string                `json:"name,omitempty"`
	Description           *string                `json:"description,omitempty"`
	BasePrice             *int64                 `json:"base_price,omitempty" validate:"omitempty,gt=0"`
	SellingPrice          *int64                 `json:"selling_price,omitempty" validate:"omitempty,gt=0"`
	CostPrice             *int64                 `json:"cost_price,omitempty"`
	Dimensions            *domain.Dimensions     `json:"dimensions,omitempty"`
	Weight                *int                   `json:"weight,omitempty"`
	AllowCustomDimensions *bool                  `json:"allow_custom_dimensions,omitempty"`
	PricingRuleID         *string                `json:"pricing_rule_id,omitempty"`
	Attributes            map[string]interface{} `json:"attributes,omitempty"`
	Material              *string                `json:"material,omitempty"`
	Color                 *string                `json:"color,omitempty"`
	WeaveType             *string                `json:"weave_type,omitempty"`
	Origin                *string                `json:"origin,omitempty"`
	CraftType             *string                `json:"craft_type,omitempty"`
	Images                []domain.ProductImage  `json:"images,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	LowStockThreshold     *int                   `json:"low_stock_threshold,omitempty"`
	Status                *domain.ProductStatus  `json:"status,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *UpdateProductRequest) ToDomain() domain.UpdateProductRequest {
	return domain.UpdateProductRequest{
		Name:                  r.Name,
		Description:           r.Description,
		BasePrice:             r.BasePrice,
		SellingPrice:          r.SellingPrice,
		CostPrice:             r.CostPrice,
		Dimensions:            r.Dimensions,
		Weight:                r.Weight,
		AllowCustomDimensions: r.AllowCustomDimensions,
		PricingRuleID:         r.PricingRuleID,
		Attributes:            r.Attributes,
		Material:              r.Material,
		Color:                 r.Color,
		WeaveType:             r.WeaveType,
		Origin:                r.Origin,
		CraftType:             r.CraftType,
		Images:                r.Images,
		Tags:                  r.Tags,
		LowStockThreshold:     r.LowStockThreshold,
		Status:                r.Status,
	}
}
