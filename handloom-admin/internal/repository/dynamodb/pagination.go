package dynamodb

import (
	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// DecodeCursor decodes a base64-encoded cursor string into a DynamoDB
// ExclusiveStartKey map. Returns nil if the cursor is empty (first page).
func DecodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errors.Validation("invalid pagination cursor")
	}

	var raw map[string]map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.Validation("invalid pagination cursor format")
	}

	result := make(map[string]types.AttributeValue, len(raw))
	for key, typeMap := range raw {
		if val, ok := typeMap["S"]; ok {
			result[key] = &types.AttributeValueMemberS{Value: val}
		} else if val, ok := typeMap["N"]; ok {
			result[key] = &types.AttributeValueMemberN{Value: val}
		}
	}

	return result, nil
}

// EncodeCursor encodes a DynamoDB LastEvaluatedKey map into a base64 string.
// Returns empty string if the key is nil (last page).
func EncodeCursor(lastKey map[string]types.AttributeValue) string {
	if len(lastKey) == 0 {
		return ""
	}

	raw := make(map[string]map[string]string, len(lastKey))
	for key, av := range lastKey {
		switch v := av.(type) {
		case *types.AttributeValueMemberS:
			raw[key] = map[string]string{"S": v.Value}
		case *types.AttributeValueMemberN:
			raw[key] = map[string]string{"N": v.Value}
		}
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(data)
}

// BuildPaginationResponse creates a PaginationResponse from a DynamoDB LastEvaluatedKey.
func BuildPaginationResponse(limit int, lastKey map[string]types.AttributeValue) domain.PaginationResponse {
	nextCursor := EncodeCursor(lastKey)
	return domain.PaginationResponse{
		Limit:      limit,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
}

// DefaultLimit returns a sanitized limit value, clamped to [1, 100].
func DefaultLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// inMemoryOffset is the cursor format used by inMemoryPaginate.
type inMemoryOffset struct {
	Offset int `json:"o"`
}

// decodeOffsetCursor decodes an offset-based cursor. Returns 0 for empty cursor.
func decodeOffsetCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	var o inMemoryOffset
	if err := json.Unmarshal(data, &o); err != nil {
		return 0
	}
	return o.Offset
}

// encodeOffsetCursor encodes an offset into a cursor string.
func encodeOffsetCursor(offset int) string {
	data, _ := json.Marshal(inMemoryOffset{Offset: offset})
	return base64.URLEncoding.EncodeToString(data)
}

// InMemoryPaginate applies cursor-based pagination to an in-memory slice.
// TODO: migrate callers to real DynamoDB cursor-based pagination
func InMemoryPaginate[T any](items []T, req domain.PaginationRequest) ([]T, domain.PaginationResponse) {
	limit := DefaultLimit(req.Limit)
	offset := decodeOffsetCursor(req.Cursor)

	if offset >= len(items) {
		return nil, domain.PaginationResponse{Limit: limit}
	}

	end := offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}

	var nextCursor string
	if hasMore {
		nextCursor = encodeOffsetCursor(end)
	}

	return items[offset:end], domain.PaginationResponse{
		Limit:      limit,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
