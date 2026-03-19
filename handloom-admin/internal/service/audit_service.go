// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
)

// AuditService implements domain.AuditService
type AuditService struct {
	auditRepo domain.AuditRepository
}

// NewAuditService creates a new AuditService
func NewAuditService(auditRepo domain.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

// Log creates an audit log entry
func (s *AuditService) Log(ctx context.Context, action string, entityType string, entityID string, userID string, changes []domain.FieldChange, metadata map[string]interface{}) error {
	now := time.Now()

	// Convert changes slice to map
	changesMap := make(map[string]domain.FieldChange)
	for i, change := range changes {
		changesMap[fmt.Sprintf("change_%d", i)] = change
	}

	log := &domain.AuditLog{
		ID:              "audit_" + uuid.New().String()[:8],
		Action:          domain.AuditAction(action),
		EntityTypeAudit: entityType,
		EntityID:        entityID,
		UserID:          userID,
		Changes:         changesMap,
		CreatedAt:       now,
	}
	log.SetKeys()

	if err := s.auditRepo.Create(ctx, log); err != nil {
		slog.ErrorContext(ctx, "Failed to create audit log", "error", err)
		return err
	}

	return nil
}

// GetByID retrieves an audit log by ID
func (s *AuditService) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	return s.auditRepo.GetByID(ctx, id)
}

// List retrieves audit logs with filters
func (s *AuditService) List(ctx context.Context, req domain.ListAuditLogsRequest) (*domain.ListAuditLogsResponse, error) {
	return s.auditRepo.List(ctx, req)
}

// GetByEntity retrieves audit logs for a specific entity
func (s *AuditService) GetByEntity(ctx context.Context, entityType string, entityID string, pagination domain.PaginationRequest) (*domain.ListAuditLogsResponse, error) {
	return s.auditRepo.GetByEntity(ctx, entityType, entityID, pagination)
}

// GetByUser retrieves audit logs for a specific user
func (s *AuditService) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListAuditLogsResponse, error) {
	return s.auditRepo.GetByUser(ctx, userID, pagination)
}

// Ensure interface compliance
var _ domain.AuditService = (*AuditService)(nil)
