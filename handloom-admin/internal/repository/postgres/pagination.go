package postgres

import (
	"encoding/base64"
	"strconv"

	"github.com/handloom/admin/internal/domain"
)

const defaultPageSize = 20
const maxPageSize = 100

// decodeCursor decodes a base64-encoded cursor into an integer offset.
// Returns 0 if the cursor is empty or invalid.
func decodeCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	offset, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	return offset
}

// encodeCursor encodes an integer offset into a base64 cursor.
func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// pageParams extracts limit and offset from a PaginationRequest.
func pageParams(req domain.PaginationRequest) (limit, offset int) {
	limit = req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	offset = decodeCursor(req.Cursor)
	return
}

// buildPaginationResponse creates a PaginationResponse from result metadata.
func buildPaginationResponse(limit, offset, fetched int) domain.PaginationResponse {
	resp := domain.PaginationResponse{
		Limit:   limit,
		HasMore: fetched > limit,
	}
	if resp.HasMore {
		resp.NextCursor = encodeCursor(offset + limit)
	}
	return resp
}
