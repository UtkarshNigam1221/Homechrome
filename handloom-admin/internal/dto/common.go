package dto

// PaginationRequest represents pagination parameters.
type PaginationRequest struct {
	Page    int `json:"page" validate:"gte=1"`
	PerPage int `json:"per_page" validate:"gte=1,lte=100"`
}

// DefaultPagination returns default pagination.
func DefaultPagination() PaginationRequest {
	return PaginationRequest{
		Page:    1,
		PerPage: 10,
	}
}

// IDRequest represents a request with just an ID.
type IDRequest struct {
	ID string `json:"id" validate:"required"`
}

// StatusRequest represents a generic status update request.
type StatusRequest struct {
	Status string `json:"status" validate:"required"`
}

// SearchRequest represents a generic search request.
type SearchRequest struct {
	Query string `json:"query" validate:"required,min=1"`
	Limit int    `json:"limit" validate:"gte=1,lte=100"`
}

// DateRangeRequest represents a date range filter.
type DateRangeRequest struct {
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}
