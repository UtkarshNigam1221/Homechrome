package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/pkg/errors"
)

// DynamicUpdate holds the components of a DynamoDB UpdateItem expression.
type DynamicUpdate struct {
	Expression string
	AttrNames  map[string]string
	AttrValues map[string]types.AttributeValue
}

// buildDynamicUpdate constructs a SET expression that sets status + updated_at,
// plus any additional key-value pairs from the updates map.
func buildDynamicUpdate(status string, updates map[string]interface{}) (*DynamicUpdate, error) {
	now := time.Now()
	expr := "SET #status = :status, updated_at = :now"
	names := map[string]string{nameStatus: attrStatus}
	values := map[string]types.AttributeValue{
		valStatus: &types.AttributeValueMemberS{Value: status},
		exprNow:   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	i := 0
	for key, val := range updates {
		placeholder := fmt.Sprintf(":val%d", i)
		nameAlias := fmt.Sprintf("#field%d", i)
		expr += fmt.Sprintf(", %s = %s", nameAlias, placeholder)
		names[nameAlias] = key

		av, err := attributevalue.Marshal(val)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to marshal update value for key %s", key))
		}
		values[placeholder] = av
		i++
	}

	return &DynamicUpdate{Expression: expr, AttrNames: names, AttrValues: values}, nil
}

// batchDeleteKeys deletes items from a DynamoDB table in chunks of 25.
// keys is a slice of DynamoDB key maps (PK/SK pairs).
func batchDeleteKeys(ctx context.Context, db *dynamodb.Client, tableName string, keys []map[string]types.AttributeValue) error {
	const batchSize = 25
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		writeRequests := make([]types.WriteRequest, 0, end-i)
		for _, key := range keys[i:end] {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{Key: key},
			})
		}

		_, err := db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: writeRequests,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
