package domain

import (
	"context"
)

// ==================== UTM LINK ENTITY ====================

// UTMDestType is where a tagged link points on the storefront. A closed set rather
// than a free-form path, so a saved link can't outlive a route rename silently.
type UTMDestType string

const (
	UTMDestHome     UTMDestType = "HOME"
	UTMDestCategory UTMDestType = "CATEGORY"
	UTMDestProduct  UTMDestType = "PRODUCT"
)

// MaxUTMValueLen caps each utm_* value. Matches middleware.MaxUTMLen and the
// storefront's UTM_MAX_LEN: a longer value would be truncated on capture, so the
// saved link would stop matching its own analytics rows.
const MaxUTMValueLen = 32

// utmSortTimeLayout is the GSI1SK timestamp format. The fraction is fixed-width on
// purpose: at second precision two links saved in the same second sorted by their
// random id suffix instead of by time, so "newest first" was wrong for bulk entry.
const utmSortTimeLayout = "2006-01-02T15:04:05.000000000Z"

// UTMLink is a saved campaign link for the storefront. URL is stored rather than
// rebuilt on read so the admin list and whatever was pasted into an ad stay identical.
type UTMLink struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Name string `json:"name" dynamodbav:"name"`

	DestType UTMDestType `json:"dest_type" dynamodbav:"dest_type"`
	DestSlug string      `json:"dest_slug,omitempty" dynamodbav:"dest_slug,omitempty"`

	UTMSource   string `json:"utm_source" dynamodbav:"utm_source"`
	UTMMedium   string `json:"utm_medium" dynamodbav:"utm_medium"`
	UTMCampaign string `json:"utm_campaign" dynamodbav:"utm_campaign"`

	URL string `json:"url" dynamodbav:"url"`

	BaseEntity
}

// TableName returns the DynamoDB table name for UTMLink
func (l *UTMLink) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for UTMLink. GSI1SK embeds CreatedAt so a single
// descending GSI1 query returns newest-first; callers must set CreatedAt first.
func (l *UTMLink) SetKeys() {
	l.PK = "UTM_LINK#" + l.ID
	l.SK = SKMetadata
	l.GSI1PK = "UTM_LINK#ALL"
	l.GSI1SK = l.CreatedAt.UTC().Format(utmSortTimeLayout) + "#" + l.ID
	l.EntityType = "UTM_LINK"
}

// ==================== UTM LINK REPOSITORY ====================

// UTMLinkRepository defines the interface for UTM link data access
type UTMLinkRepository interface {
	Create(ctx context.Context, link *UTMLink) error
	GetByID(ctx context.Context, id string) (*UTMLink, error)
	Update(ctx context.Context, link *UTMLink) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, req ListUTMLinksRequest) (*ListUTMLinksResponse, error)
}

// ListUTMLinksRequest contains parameters for listing UTM links
type ListUTMLinksRequest struct {
	PaginationRequest
	Search string `json:"search,omitempty"`
}

// ListUTMLinksResponse contains the list of UTM links
type ListUTMLinksResponse struct {
	Links      []*UTMLink         `json:"links"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== UTM LINK SERVICE ====================

// UTMLinkService defines the interface for UTM link operations
type UTMLinkService interface {
	Create(ctx context.Context, req CreateUTMLinkRequest, createdBy string) (*UTMLink, error)
	GetByID(ctx context.Context, id string) (*UTMLink, error)
	Update(ctx context.Context, id string, req UpdateUTMLinkRequest, updatedBy string) (*UTMLink, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, req ListUTMLinksRequest) (*ListUTMLinksResponse, error)
}

// CreateUTMLinkRequest contains data for creating a UTM link
type CreateUTMLinkRequest struct {
	Name        string      `json:"name" validate:"required"`
	DestType    UTMDestType `json:"dest_type" validate:"required,oneof=HOME CATEGORY PRODUCT"`
	DestSlug    string      `json:"dest_slug,omitempty"`
	UTMSource   string      `json:"utm_source" validate:"required"`
	UTMMedium   string      `json:"utm_medium" validate:"required"`
	UTMCampaign string      `json:"utm_campaign" validate:"required"`
}

// UpdateUTMLinkRequest contains data for updating a UTM link. Every field is
// optional; the URL is rebuilt from whatever the merge produces.
type UpdateUTMLinkRequest struct {
	Name        *string      `json:"name,omitempty"`
	DestType    *UTMDestType `json:"dest_type,omitempty" validate:"omitempty,oneof=HOME CATEGORY PRODUCT"`
	DestSlug    *string      `json:"dest_slug,omitempty"`
	UTMSource   *string      `json:"utm_source,omitempty"`
	UTMMedium   *string      `json:"utm_medium,omitempty"`
	UTMCampaign *string      `json:"utm_campaign,omitempty"`
}
