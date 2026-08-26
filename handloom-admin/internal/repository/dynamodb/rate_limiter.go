package dynamodb

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// RateLimiter implements domain.RateLimiter on the sessions table, which
// already expires items by TTL.
type RateLimiter struct {
	client *Client
	// now is a seam so tests can cross a cooldown or a window without sleeping.
	now func() time.Time
}

// NewRateLimiter creates a new RateLimiter
func NewRateLimiter(client *Client) *RateLimiter {
	return &RateLimiter{client: client, now: time.Now}
}

func rateWindowKey(key string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "RATE#" + key},
		"SK": &types.AttributeValueMemberS{Value: skMetadata},
	}
}

// Claim records one use of key against rule.
//
// Two conditional updates, because one UpdateExpression cannot branch between
// extending a live window and starting one; lazy TTL deletion would lock out.
func (r *RateLimiter) Claim(ctx context.Context, key string, rule domain.RateLimitRule) error {
	now := r.now()
	nowVal := &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)}
	oneVal := &types.AttributeValueMemberN{Value: "1"}
	names := map[string]string{"#ttl": "ttl"}

	// Live window: past the cooldown and under the cap.
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                aws.String(r.client.sessionsTable),
		Key:                      rateWindowKey(key),
		UpdateExpression:         aws.String("SET claim_count = claim_count + :one, last_claim_at = :now"),
		ConditionExpression:      aws.String("attribute_exists(PK) AND #ttl > :now AND last_claim_at <= :cutoff AND claim_count < :max"),
		ExpressionAttributeNames: names,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			valOne:    oneVal,
			":now":    nowVal,
			":cutoff": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Add(-rule.Cooldown).Unix(), 10)},
			":max":    &types.AttributeValueMemberN{Value: strconv.Itoa(rule.Max)},
		},
	})
	if err == nil {
		return nil
	}
	if !isConditionalCheckFailed(err) {
		return errors.Wrap(err, "Failed to claim rate limit")
	}

	// No window, or a spent one: start fresh.
	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                aws.String(r.client.sessionsTable),
		Key:                      rateWindowKey(key),
		UpdateExpression:         aws.String("SET claim_count = :one, last_claim_at = :now, entity_type = :entity, #ttl = :ttl"),
		ConditionExpression:      aws.String("attribute_not_exists(PK) OR #ttl <= :now"),
		ExpressionAttributeNames: names,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			valOne:    oneVal,
			":now":    nowVal,
			":entity": &types.AttributeValueMemberS{Value: "RATE_WINDOW"},
			valTTL:    &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Add(rule.Window).Unix(), 10)},
		},
	})
	if err == nil {
		return nil
	}
	if isConditionalCheckFailed(err) {
		return errors.New(errors.ErrCodeRateLimited,
			"Too many requests. Please wait before trying again.")
	}
	return errors.Wrap(err, "Failed to claim rate limit")
}

// Ensure interface compliance
var _ domain.RateLimiter = (*RateLimiter)(nil)
