package dto

import "github.com/handloom/admin/internal/domain"

// AddStockRequest represents the add stock request.
type AddStockRequest struct {
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Reason      string `json:"reason" validate:"required"`
	ReferenceID string `json:"reference_id,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *AddStockRequest) ToDomain() domain.AddStockRequest {
	return domain.AddStockRequest{
		Quantity:    r.Quantity,
		Reason:      r.Reason,
		ReferenceID: r.ReferenceID,
	}
}

// RemoveStockRequest represents the remove stock request.
type RemoveStockRequest struct {
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Reason      string `json:"reason" validate:"required"`
	ReferenceID string `json:"reference_id,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *RemoveStockRequest) ToDomain() domain.RemoveStockRequest {
	return domain.RemoveStockRequest{
		Quantity:    r.Quantity,
		Reason:      r.Reason,
		ReferenceID: r.ReferenceID,
	}
}

// AdjustStockRequest represents the adjust stock request.
type AdjustStockRequest struct {
	NewQuantity int    `json:"new_quantity" validate:"gte=0"`
	Reason      string `json:"reason" validate:"required"`
}

// ToDomain converts DTO to domain request.
func (r *AdjustStockRequest) ToDomain() domain.AdjustStockRequest {
	return domain.AdjustStockRequest{
		NewQuantity: r.NewQuantity,
		Reason:      r.Reason,
	}
}
