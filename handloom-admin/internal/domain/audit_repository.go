package domain

import (
	"context"
)

//go:generate mockgen -source=audit_repository.go -destination=../mocks/audit_repository_mock.go -package=mocks

// AuditRepository defines the interface for audit log data access
type AuditRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, log *AuditLog) error

	// GetByID retrieves an audit log by ID
	GetByID(ctx context.Context, id string) (*AuditLog, error)

	// List retrieves audit logs with filters
	List(ctx context.Context, req ListAuditLogsRequest) (*ListAuditLogsResponse, error)

	// GetByEntity retrieves audit logs for a specific entity
	GetByEntity(ctx context.Context, entityType string, entityID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)

	// GetByUser retrieves audit logs for a specific user
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)
}

// ListAuditLogsRequest contains parameters for listing audit logs
type ListAuditLogsRequest struct {
	PaginationRequest
	Action     *string `json:"action,omitempty"`
	EntityType *string `json:"entity_type,omitempty"`
	EntityID   *string `json:"entity_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
	StartDate  *string `json:"start_date,omitempty"`
	EndDate    *string `json:"end_date,omitempty"`
}

// ListAuditLogsResponse contains the list of audit logs
type ListAuditLogsResponse struct {
	Logs       []*AuditLog        `json:"logs"`
	Pagination PaginationResponse `json:"pagination"`
}

// AuditService defines the interface for audit operations
type AuditService interface {
	// Log creates an audit log entry
	Log(ctx context.Context, action string, entityType string, entityID string, userID string, changes []FieldChange, metadata map[string]interface{}) error

	// GetByID retrieves an audit log by ID
	GetByID(ctx context.Context, id string) (*AuditLog, error)

	// List retrieves audit logs with filters
	List(ctx context.Context, req ListAuditLogsRequest) (*ListAuditLogsResponse, error)

	// GetByEntity retrieves audit logs for a specific entity
	GetByEntity(ctx context.Context, entityType string, entityID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)

	// GetByUser retrieves audit logs for a specific user
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)
}
