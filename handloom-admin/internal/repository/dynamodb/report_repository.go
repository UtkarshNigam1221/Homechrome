package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// ReportRepository implements domain.ReportRepository
type ReportRepository struct {
	client *Client
}

// NewReportRepository creates a new ReportRepository
func NewReportRepository(client *Client) *ReportRepository {
	return &ReportRepository{
		client: client,
	}
}

// Create creates a new report
func (r *ReportRepository) Create(ctx context.Context, report *domain.Report) error {
	now := time.Now()
	report.CreatedAt = now
	report.UpdatedAt = now
	report.SetKeys()

	av, err := attributevalue.MarshalMap(report)
	if err != nil {
		return errors.Internal("Failed to marshal report")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Report already exists")
		}
		return errors.Wrap(err, "Failed to create report")
	}

	return nil
}

// GetByID retrieves a report by ID
func (r *ReportRepository) GetByID(ctx context.Context, id string) (*domain.Report, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REPORT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get report")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Report")
	}

	var report domain.Report
	if err := attributevalue.UnmarshalMap(result.Item, &report); err != nil {
		return nil, errors.Internal("Failed to unmarshal report")
	}

	return &report, nil
}

// Update updates a report
func (r *ReportRepository) Update(ctx context.Context, report *domain.Report) error {
	report.UpdatedAt = time.Now()
	report.SetKeys()

	av, err := attributevalue.MarshalMap(report)
	if err != nil {
		return errors.Internal("Failed to marshal report")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Report")
		}
		return errors.Wrap(err, "Failed to update report")
	}

	return nil
}

// Delete deletes a report
func (r *ReportRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REPORT#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Report")
		}
		return errors.Wrap(err, "Failed to delete report")
	}

	return nil
}

// List lists reports with filters
func (r *ReportRepository) List(ctx context.Context, req domain.ListReportsRequest) (*domain.ListReportsResponse, error) {
	// TODO: Implement with DynamoDB scan/query
	return &domain.ListReportsResponse{
		Reports:    []*domain.Report{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// GetByUser retrieves reports for a user
func (r *ReportRepository) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListReportsResponse, error) {
	// TODO: Implement with GSI query
	return &domain.ListReportsResponse{
		Reports:    []*domain.Report{},
		Pagination: domain.PaginationResponse{},
	}, nil
}

// Ensure interface compliance
var _ domain.ReportRepository = (*ReportRepository)(nil)
