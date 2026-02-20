package domain

import (
	"time"
)

// ==================== AUDIT LOG ENTITY ====================

// AuditAction defines the type of audit action
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionLogin  AuditAction = "LOGIN"
	AuditActionLogout AuditAction = "LOGOUT"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string                 `json:"id" dynamodbav:"id"`
	PK           string                 `json:"-" dynamodbav:"PK"`
	SK           string                 `json:"-" dynamodbav:"SK"`
	GSI1PK       string                 `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK       string                 `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK       string                 `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK       string                 `json:"-" dynamodbav:"GSI2SK"`
	EntityType   string                 `json:"-" dynamodbav:"entity_type"`
	TTL          int64                  `json:"-" dynamodbav:"ttl"` // 90 days

	// User info
	UserID       string                 `json:"user_id" dynamodbav:"user_id"`
	UserEmail    string                 `json:"user_email" dynamodbav:"user_email"`
	UserRole     UserRole               `json:"user_role" dynamodbav:"user_role"`

	// Action info
	Action       AuditAction            `json:"action" dynamodbav:"action"`
	EntityTypeAudit string              `json:"entity_type_audit" dynamodbav:"entity_type_audit"` // e.g., ORDER, PRODUCT
	EntityID     string                 `json:"entity_id" dynamodbav:"entity_id"`

	// Changes
	Changes      map[string]FieldChange `json:"changes,omitempty" dynamodbav:"changes,omitempty"`
	OldValues    map[string]interface{} `json:"old_values,omitempty" dynamodbav:"old_values,omitempty"`
	NewValues    map[string]interface{} `json:"new_values,omitempty" dynamodbav:"new_values,omitempty"`

	// Request info
	IPAddress    string                 `json:"ip_address,omitempty" dynamodbav:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty" dynamodbav:"user_agent,omitempty"`
	RequestID    string                 `json:"request_id,omitempty" dynamodbav:"request_id,omitempty"`

	CreatedAt    time.Time              `json:"created_at" dynamodbav:"created_at"`
}

// TableName returns the DynamoDB table name for AuditLog
func (a *AuditLog) TableName() string {
	return "handloom-audit"
}

// SetKeys sets the DynamoDB keys for AuditLog
func (a *AuditLog) SetKeys() {
	a.PK = "AUDIT#" + a.CreatedAt.Format("2006-01-02")
	a.SK = a.CreatedAt.Format("15:04:05.000Z") + "#" + a.ID
	a.GSI1PK = a.EntityTypeAudit + "#" + a.EntityID
	a.GSI1SK = a.CreatedAt.Format("2006-01-02T15:04:05Z")
	a.GSI2PK = "USER#" + a.UserID
	a.GSI2SK = a.CreatedAt.Format("2006-01-02T15:04:05Z")
	a.EntityType = "AUDIT_LOG"
	a.TTL = a.CreatedAt.Add(90 * 24 * time.Hour).Unix()
}

// FieldChange represents a change in a field
type FieldChange struct {
	OldValue interface{} `json:"old_value" dynamodbav:"old_value"`
	NewValue interface{} `json:"new_value" dynamodbav:"new_value"`
}
