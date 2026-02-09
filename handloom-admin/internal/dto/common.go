package dto

// PaginationRequest represents cursor-based pagination parameters.
type PaginationRequest struct {
	Limit  int    `json:"limit" validate:"gte=1,lte=100"`
	Cursor string `json:"cursor,omitempty"`
}

// DefaultPagination returns default pagination.
func DefaultPagination() PaginationRequest {
	return PaginationRequest{
		Limit: 20,
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
