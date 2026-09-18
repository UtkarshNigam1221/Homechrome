package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// ==================== WEB PUSH ENTITIES ====================

// PushSubscriptionStatus is the delivery state of a browser push subscription.
type PushSubscriptionStatus string

const (
	// PushSubscriptionActive means the endpoint still accepts deliveries.
	PushSubscriptionActive PushSubscriptionStatus = "ACTIVE"
	// PushSubscriptionInactive means the browser unsubscribed, or the push
	// service answered 404/410 and the endpoint is permanently gone.
	PushSubscriptionInactive PushSubscriptionStatus = "INACTIVE"
)

// PushSubscriptionKeys holds the ECDH public key and auth secret the browser
// generated for this subscription. Together with the endpoint they are the
// full credential needed to deliver a push, so they are never returned to
// unauthenticated callers.
type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh" dynamodbav:"p256dh" validate:"required"`
	Auth   string `json:"auth" dynamodbav:"auth" validate:"required"`
}

// PushDeviceInfo is the coarse browser/OS label shown in the admin console.
type PushDeviceInfo struct {
	Browser  string `json:"browser,omitempty" dynamodbav:"browser,omitempty"`
	OS       string `json:"os,omitempty" dynamodbav:"os,omitempty"`
	IsMobile bool   `json:"is_mobile" dynamodbav:"is_mobile"`
}

// PushSubscription is one browser's Web Push registration.
type PushSubscription struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Endpoint string               `json:"endpoint" dynamodbav:"endpoint"`
	Keys     PushSubscriptionKeys `json:"-" dynamodbav:"keys"`

	UserAgent string          `json:"user_agent,omitempty" dynamodbav:"user_agent,omitempty"`
	Device    *PushDeviceInfo `json:"device,omitempty" dynamodbav:"device,omitempty"`

	Status PushSubscriptionStatus `json:"status" dynamodbav:"status"`

	CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at" dynamodbav:"last_seen_at"`
}

// PushEndpointID derives the stable item ID for an endpoint URL. Endpoints run
// to ~500 chars and vary by push service, so they are hashed rather than used
// as a key directly — this also makes re-subscribing the same browser an
// idempotent overwrite instead of a duplicate row.
func PushEndpointID(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])
}

// TableName returns the DynamoDB table name for PushSubscription
func (p *PushSubscription) TableName() string {
	return TableNotifications
}

// SetKeys sets the DynamoDB keys for PushSubscription.
// GSI1 is partitioned by status so a broadcast can query only live endpoints
// without scanning and filtering out the dead ones.
func (p *PushSubscription) SetKeys() {
	p.PK = "PUSH_SUB#" + p.ID
	p.SK = SKMetadata
	p.GSI1PK = "PUSH_SUB#" + string(p.Status)
	p.GSI1SK = p.CreatedAt.UTC().Format(time.RFC3339)
	p.EntityType = "PUSH_SUBSCRIPTION"
}

// PushBroadcastStatus is the aggregate outcome of a fan-out.
type PushBroadcastStatus string

const (
	// PushBroadcastSuccess means every targeted endpoint accepted the push.
	PushBroadcastSuccess PushBroadcastStatus = "SUCCESS"
	// PushBroadcastPartial means some endpoints accepted and some failed.
	PushBroadcastPartial PushBroadcastStatus = "PARTIAL"
	// PushBroadcastFailed means no endpoint accepted the push.
	PushBroadcastFailed PushBroadcastStatus = "FAILED"
)

// PushBroadcast is the audit record of one admin-triggered fan-out.
type PushBroadcast struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Title string `json:"title" dynamodbav:"title"`
	Body  string `json:"body" dynamodbav:"body"`
	URL   string `json:"url" dynamodbav:"url"`
	Tag   string `json:"tag,omitempty" dynamodbav:"tag,omitempty"`
	Image string `json:"image,omitempty" dynamodbav:"image,omitempty"`

	TotalTargeted int                 `json:"total_targeted" dynamodbav:"total_targeted"`
	SuccessCount  int                 `json:"success_count" dynamodbav:"success_count"`
	FailureCount  int                 `json:"failure_count" dynamodbav:"failure_count"`
	Status        PushBroadcastStatus `json:"status" dynamodbav:"status"`

	SentAt time.Time `json:"sent_at" dynamodbav:"sent_at"`
	SentBy string    `json:"sent_by" dynamodbav:"sent_by"`
}

// TableName returns the DynamoDB table name for PushBroadcast
func (b *PushBroadcast) TableName() string {
	return TableNotifications
}

// SetKeys sets the DynamoDB keys for PushBroadcast
func (b *PushBroadcast) SetKeys() {
	b.PK = "PUSH_BROADCAST#" + b.ID
	b.SK = SKMetadata
	b.GSI1PK = "PUSH_BROADCAST"
	b.GSI1SK = b.SentAt.UTC().Format(time.RFC3339Nano)
	b.EntityType = "PUSH_BROADCAST"
}

// ==================== WEB PUSH REPOSITORY ====================

// PushSubscriptionRepository defines data access for Web Push state.
type PushSubscriptionRepository interface {
	// Save upserts a subscription, reporting whether the endpoint was new.
	Save(ctx context.Context, sub *PushSubscription) (isNew bool, err error)

	// GetByEndpoint retrieves a single subscription by its endpoint URL.
	GetByEndpoint(ctx context.Context, endpoint string) (*PushSubscription, error)

	// Deactivate marks an endpoint INACTIVE. Deactivating an endpoint that is
	// already gone is a no-op, not an error.
	Deactivate(ctx context.Context, endpoint string) error

	// ListActive retrieves every ACTIVE subscription, for broadcast fan-out.
	ListActive(ctx context.Context) ([]*PushSubscription, error)

	// List retrieves subscriptions of one status, newest first, for the admin console.
	List(ctx context.Context, status PushSubscriptionStatus, pagination PaginationRequest) (*ListPushSubscriptionsResponse, error)

	// SaveBroadcast records the outcome of a fan-out.
	SaveBroadcast(ctx context.Context, broadcast *PushBroadcast) error

	// ListBroadcasts retrieves broadcast history, newest first.
	ListBroadcasts(ctx context.Context, limit int32) ([]*PushBroadcast, error)
}

// ListPushSubscriptionsResponse is a page of subscriptions.
type ListPushSubscriptionsResponse struct {
	Subscriptions []*PushSubscription `json:"subscriptions"`
	Pagination    PaginationResponse  `json:"pagination"`
}

// ==================== WEB PUSH SERVICE REQUESTS ====================

// SubscribePushRequest is the storefront's registration payload. It mirrors the
// browser's PushSubscription.toJSON() shape plus a device label.
type SubscribePushRequest struct {
	Endpoint string               `json:"endpoint" validate:"required,url,max=512"`
	Keys     PushSubscriptionKeys `json:"keys" validate:"required"`
	Device   *PushDeviceInfo      `json:"device,omitempty"`
}

// UnsubscribePushRequest identifies the endpoint to retire.
type UnsubscribePushRequest struct {
	Endpoint string `json:"endpoint" validate:"required,url,max=512"`
}

// PushPayload is the notification the service worker renders.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Icon  string `json:"icon,omitempty"`
	// Image is the wide banner Android shows when the notification is expanded.
	Image string `json:"image,omitempty"`
}

// TestPushRequest asks for a delivery to one endpoint the caller already owns.
// Endpoint-scoped on purpose: a storefront visitor testing their own
// notifications must never be able to reach anyone else's device.
type TestPushRequest struct {
	Endpoint string `json:"endpoint" validate:"required,url,max=512"`
}

// BroadcastPushRequest is the admin fan-out payload.
type BroadcastPushRequest struct {
	Title string `json:"title" validate:"required,max=120"`
	Body  string `json:"body" validate:"required,max=300"`
	URL   string `json:"url,omitempty" validate:"omitempty,max=512,startswith=/"`
	Tag   string `json:"tag,omitempty" validate:"omitempty,max=64"`
	Icon  string `json:"icon,omitempty" validate:"omitempty,max=512"`
	Image string `json:"image,omitempty" validate:"omitempty,url,max=512"`
}

// BroadcastPushResponse reports the fan-out result to the admin console.
type BroadcastPushResponse struct {
	TotalTargeted int                 `json:"total_targeted"`
	SuccessCount  int                 `json:"success_count"`
	FailureCount  int                 `json:"failure_count"`
	Status        PushBroadcastStatus `json:"status"`
	Broadcast     *PushBroadcast      `json:"broadcast"`
}

// PushStatsResponse is the admin console's subscriber summary.
type PushStatsResponse struct {
	ActiveCount   int `json:"active_count"`
	InactiveCount int `json:"inactive_count"`
}
