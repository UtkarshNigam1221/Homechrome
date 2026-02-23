package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
)

// getUserIDFromContext retrieves the user ID from context
// Delegates to middleware to use the same context key type
func getUserIDFromContext(ctx context.Context) string {
	return middleware.GetUserIDFromContext(ctx)
}

// getUserFromContext retrieves the user from context
// Delegates to middleware to use the same context key type
func getUserFromContext(ctx context.Context) *domain.User {
	return middleware.GetUserFromContext(ctx)
}

// parsePagination parses cursor-based pagination parameters from request
func parsePagination(r *http.Request) domain.PaginationRequest {
	limit := 20
	sortBy := ""
	sortDir := "desc"

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	} else if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	cursor := r.URL.Query().Get("cursor")

	if sb := r.URL.Query().Get("sort_by"); sb != "" {
		sortBy = sb
	}

	if sd := r.URL.Query().Get("sort_order"); sd == "asc" || sd == "desc" {
		sortDir = sd
	}

	return domain.PaginationRequest{
		Limit:   limit,
		Cursor:  cursor,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
}

// parseIntParam parses an integer parameter
func parseIntParam(value string, target *int) (bool, error) {
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return false, err
	}
	*target = parsed
	return true, nil
}

// parseInt64Param parses an int64 parameter
func parseInt64Param(value string, target *int64) (bool, error) {
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, err
	}
	*target = parsed
	return true, nil
}

// parseStringPtr returns a pointer to the value, or nil if empty.
func parseStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// parseInt64Ptr parses an int64 query param and returns a pointer, or nil if empty/invalid.
func parseInt64Ptr(value string) *int64 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// parseBoolParam parses a boolean parameter
func parseBoolParam(value string) *bool {
	if value == "" {
		return nil
	}
	b := value == "true" || value == "1"
	return &b
}

// getLogger retrieves logger from context
func getLogger(ctx context.Context) *logger.Logger {
	return logger.FromContext(ctx)
}
