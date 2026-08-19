package dynamodb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
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

// InMemoryPaginate slices an already-complete result set.
//
// Correct only on the output of QueryAll, never on a single Query response —
// that combination is what silently truncated these lists at DynamoDB's 1 MB
// response cap. Use QueryPage unless the caller sorts or filters in Go, where
// paging the index would order or count only the page it returned.
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

// maxQueryPageRoundTrips bounds how hard QueryPage works to fill one page.
// A selective FilterExpression can make DynamoDB read many items and return
// few, and without a bound a rare filter would walk the whole index in one
// request. Stopping early is safe: the caller gets a short page and a cursor,
// and asking again continues from there.
const maxQueryPageRoundTrips = 10

// QueryPage runs a paginated Query and returns one page plus a real cursor.
//
// It exists because reading a single Query response and slicing it in Go — what
// every list here used to do — silently truncates. A Query returns at most 1 MB
// and reports success, so past that the list stops early and looks complete.
// That is a wrong answer, not a slow one.
//
// Limit bounds items *scanned*, not returned, so a filtered query can come back
// short. This loops until the page is full or the index is exhausted, and never
// overshoots: asking for exactly the shortfall each time means the last
// evaluated key always lines up with the last item returned, so resuming from
// the cursor cannot skip or repeat.
func QueryPage[T any](
	ctx context.Context,
	db *dynamodb.Client,
	input *dynamodb.QueryInput,
	req domain.PaginationRequest,
	failure string,
) ([]*T, domain.PaginationResponse, error) {
	limit := DefaultLimit(req.Limit)

	startKey, err := DecodeCursor(req.Cursor)
	if err != nil {
		return nil, domain.PaginationResponse{}, err
	}

	items := make([]*T, 0, limit)
	var lastKey map[string]types.AttributeValue

	for range maxQueryPageRoundTrips {
		input.ExclusiveStartKey = startKey
		// DefaultLimit clamps to [1,100], so the shortfall cannot overflow.
		//nolint:gosec // G115: bounded by DefaultLimit
		input.Limit = aws.Int32(int32(limit - len(items)))

		result, queryErr := db.Query(ctx, input)
		if queryErr != nil {
			return nil, domain.PaginationResponse{}, errors.Wrap(queryErr, failure)
		}

		var page []*T
		if unmarshalErr := attributevalue.UnmarshalListOfMaps(result.Items, &page); unmarshalErr != nil {
			return nil, domain.PaginationResponse{}, errors.Internal(failure)
		}
		items = append(items, page...)

		startKey = result.LastEvaluatedKey
		lastKey = result.LastEvaluatedKey

		// No further key means the index is exhausted, not that the page is full.
		if len(startKey) == 0 || len(items) >= limit {
			break
		}
	}

	return items, BuildPaginationResponse(limit, lastKey), nil
}

// maxQueryAllRoundTrips bounds QueryAll. Reaching it means the dataset has
// outgrown listing it in memory, which is the signal to give that endpoint real
// cursors rather than to raise this number.
const maxQueryAllRoundTrips = 50

// QueryAll reads every matching item, following LastEvaluatedKey to the end.
//
// For the lists that sort or filter in Go: a cursor can only order the page it
// returns, so switching them to QueryPage would quietly turn a global sort into
// a per-page one. Reading everything keeps the semantics and still fixes the
// truncation, which is the actual defect — the old code read one 1 MB response
// and reported success as though it had seen the whole index.
func QueryAll[T any](
	ctx context.Context,
	db *dynamodb.Client,
	input *dynamodb.QueryInput,
	failure string,
) ([]*T, error) {
	var items []*T
	var startKey map[string]types.AttributeValue

	for range maxQueryAllRoundTrips {
		input.ExclusiveStartKey = startKey

		result, err := db.Query(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, failure)
		}

		var page []*T
		if unmarshalErr := attributevalue.UnmarshalListOfMaps(result.Items, &page); unmarshalErr != nil {
			return nil, errors.Internal(failure)
		}
		items = append(items, page...)

		startKey = result.LastEvaluatedKey
		if len(startKey) == 0 {
			break
		}
	}

	if len(startKey) > 0 {
		// Returning a partial set with no error is exactly the silent wrong answer
		// these helpers exist to remove. Say so: reaching the cap is the signal
		// that this endpoint has outgrown reading to the end and needs real
		// cursors.
		slog.WarnContext(ctx, "Query stopped at the round-trip cap with more to read",
			"failure", failure, "round_trips", maxQueryAllRoundTrips, "items", len(items))
		// db_ prefix so retention classes it as operational rather than business.
		metrics.Record(ctx, "db_query_all_truncated", metrics.L{})
	}

	return items, nil
}
