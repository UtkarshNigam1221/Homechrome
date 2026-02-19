package domain

import (
	"context"
	"time"
)

// ==================== ARTISAN ENTITY ====================

// ArtisanStatus defines the status of an artisan
type ArtisanStatus string

const (
	ArtisanStatusActive   ArtisanStatus = "ACTIVE"
	ArtisanStatusInactive ArtisanStatus = "INACTIVE"
	ArtisanStatusPending  ArtisanStatus = "PENDING"
)

// Artisan represents an artisan/craftsperson
type Artisan struct {
	ID              string        `json:"id" dynamodbav:"id"`
	PK              string        `json:"-" dynamodbav:"PK"`
	SK              string        `json:"-" dynamodbav:"SK"`
	GSI1PK          string        `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK          string        `json:"-" dynamodbav:"GSI1SK"`
	EntityType      string        `json:"-" dynamodbav:"entity_type"`

	// Basic Info
	Name            string        `json:"name" dynamodbav:"name"`
	Email           string        `json:"email,omitempty" dynamodbav:"email,omitempty"`
	Phone           string        `json:"phone" dynamodbav:"phone"`
	ProfileImage    string        `json:"profile_image,omitempty" dynamodbav:"profile_image,omitempty"`
	Bio             string        `json:"bio,omitempty" dynamodbav:"bio,omitempty"`

	// Location
	Address         *Address      `json:"address,omitempty" dynamodbav:"address,omitempty"`
	Location        string        `json:"location" dynamodbav:"location"` // e.g., "Jaipur, Rajasthan"

	// Craft Info
	CraftTypes      []string      `json:"craft_types" dynamodbav:"craft_types"` // e.g., ["Block Printing", "Handloom"]
	Specializations []string      `json:"specializations,omitempty" dynamodbav:"specializations,omitempty"`
	Experience      int           `json:"experience" dynamodbav:"experience"` // in years

	// Bank Details (for payments)
	BankDetails     *BankDetails  `json:"bank_details,omitempty" dynamodbav:"bank_details,omitempty"`

	// Commission
	CommissionRate  float64       `json:"commission_rate" dynamodbav:"commission_rate"` // percentage

	// Stats
	ProductCount    int           `json:"product_count" dynamodbav:"product_count"`
	TotalSales      int64         `json:"total_sales" dynamodbav:"total_sales"`
	TotalEarnings   int64         `json:"total_earnings" dynamodbav:"total_earnings"`
	PendingPayout   int64         `json:"pending_payout" dynamodbav:"pending_payout"`

	Status          ArtisanStatus `json:"status" dynamodbav:"status"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Artisan
func (a *Artisan) TableName() string {
	return "handloom-core"
}

// SetKeys sets the DynamoDB keys for Artisan
func (a *Artisan) SetKeys() {
	a.PK = "ARTISAN#" + a.ID
	a.SK = "METADATA"
	a.GSI1PK = "ARTISAN_STATUS"
	a.GSI1SK = string(a.Status) + "#" + a.ID
	a.EntityType = "ARTISAN"
}

// BankDetails represents bank account details
type BankDetails struct {
	AccountHolderName string `json:"account_holder_name" dynamodbav:"account_holder_name"`
	AccountNumber     string `json:"account_number" dynamodbav:"account_number"`
	IFSCCode          string `json:"ifsc_code" dynamodbav:"ifsc_code"`
	BankName          string `json:"bank_name" dynamodbav:"bank_name"`
	BranchName        string `json:"branch_name,omitempty" dynamodbav:"branch_name,omitempty"`
}

// ArtisanPayout represents a payout to an artisan
type ArtisanPayout struct {
	ID            string    `json:"id" dynamodbav:"id"`
	PK            string    `json:"-" dynamodbav:"PK"`
	SK            string    `json:"-" dynamodbav:"SK"`
	EntityType    string    `json:"-" dynamodbav:"entity_type"`

	ArtisanID     string    `json:"artisan_id" dynamodbav:"artisan_id"`
	Amount        int64     `json:"amount" dynamodbav:"amount"`
	Status        string    `json:"status" dynamodbav:"status"` // PENDING, PROCESSING, COMPLETED, FAILED
	PaymentMethod string    `json:"payment_method" dynamodbav:"payment_method"`
	TransactionID string    `json:"transaction_id,omitempty" dynamodbav:"transaction_id,omitempty"`

	// Period covered
	PeriodStart   time.Time `json:"period_start" dynamodbav:"period_start"`
	PeriodEnd     time.Time `json:"period_end" dynamodbav:"period_end"`

	// Order details
	OrderIDs      []string  `json:"order_ids" dynamodbav:"order_ids"`
	OrderCount    int       `json:"order_count" dynamodbav:"order_count"`

	ProcessedAt   *time.Time `json:"processed_at,omitempty" dynamodbav:"processed_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty" dynamodbav:"failure_reason,omitempty"`

	CreatedAt     time.Time `json:"created_at" dynamodbav:"created_at"`
	CreatedBy     string    `json:"created_by" dynamodbav:"created_by"`
}

// TableName returns the DynamoDB table name for ArtisanPayout
func (p *ArtisanPayout) TableName() string {
	return "handloom-core"
}

// SetKeys sets the DynamoDB keys for ArtisanPayout
func (p *ArtisanPayout) SetKeys() {
	p.PK = "ARTISAN#" + p.ArtisanID
	p.SK = "PAYOUT#" + p.CreatedAt.Format("2006-01-02T15:04:05Z")
	p.EntityType = "ARTISAN_PAYOUT"
}

// ==================== ARTISAN REPOSITORY ====================

// ArtisanRepository defines the interface for artisan data access
type ArtisanRepository interface {
	// Create creates a new artisan
	Create(ctx context.Context, artisan *Artisan) error

	// GetByID retrieves an artisan by ID
	GetByID(ctx context.Context, id string) (*Artisan, error)

	// Update updates an existing artisan
	Update(ctx context.Context, artisan *Artisan) error

	// Delete deletes an artisan by ID
	Delete(ctx context.Context, id string) error

	// List retrieves artisans with filters
	List(ctx context.Context, req ListArtisansRequest) (*ListArtisansResponse, error)

	// UpdateStats updates artisan statistics
	UpdateStats(ctx context.Context, id string, productCount int, totalSales int64, totalEarnings int64) error

	// CreatePayout creates a new payout record
	CreatePayout(ctx context.Context, payout *ArtisanPayout) error

	// GetPayouts retrieves payouts for an artisan
	GetPayouts(ctx context.Context, artisanID string, pagination PaginationRequest) (*ListArtisanPayoutsResponse, error)

	// GetProducts retrieves products for an artisan
	GetProducts(ctx context.Context, artisanID string, pagination PaginationRequest) (*ListProductsResponse, error)

	// Search searches artisans by query
	Search(ctx context.Context, query string, pagination PaginationRequest) (*ListArtisansResponse, error)
}

// ListArtisansRequest contains parameters for listing artisans
type ListArtisansRequest struct {
	PaginationRequest
	Status    *ArtisanStatus `json:"status,omitempty"`
	CraftType string         `json:"craft_type,omitempty"`
	Location  string         `json:"location,omitempty"`
	Search    string         `json:"search,omitempty"`
}

// ListArtisansResponse contains the list of artisans
type ListArtisansResponse struct {
	Artisans   []*Artisan         `json:"artisans"`
	Pagination PaginationResponse `json:"pagination"`
}

// ListArtisanPayoutsResponse contains the list of artisan payouts
type ListArtisanPayoutsResponse struct {
	Payouts    []*ArtisanPayout   `json:"payouts"`
	Pagination PaginationResponse `json:"pagination"`
}

// ==================== ARTISAN SERVICE ====================

// ArtisanService defines the interface for artisan operations
type ArtisanService interface {
	// Create creates a new artisan
	Create(ctx context.Context, req CreateArtisanRequest, createdBy string) (*Artisan, error)

	// GetByID retrieves an artisan by ID
	GetByID(ctx context.Context, id string) (*Artisan, error)

	// Update updates an existing artisan
	Update(ctx context.Context, id string, req UpdateArtisanRequest, updatedBy string) (*Artisan, error)

	// Delete deletes an artisan by ID
	Delete(ctx context.Context, id string) error

	// List retrieves artisans with filters
	List(ctx context.Context, req ListArtisansRequest) (*ListArtisansResponse, error)

	// UpdateStatus updates artisan status
	UpdateStatus(ctx context.Context, id string, status ArtisanStatus, updatedBy string) error

	// GetPayouts retrieves payouts for an artisan
	GetPayouts(ctx context.Context, artisanID string, pagination PaginationRequest) (*ListArtisanPayoutsResponse, error)

	// CreatePayout creates a payout for an artisan
	CreatePayout(ctx context.Context, req CreatePayoutRequest, createdBy string) (*ArtisanPayout, error)
}

// CreateArtisanRequest contains data for creating an artisan
type CreateArtisanRequest struct {
	Name            string       `json:"name" validate:"required"`
	Email           string       `json:"email,omitempty"`
	Phone           string       `json:"phone" validate:"required"`
	ProfileImage    string       `json:"profile_image,omitempty"`
	Bio             string       `json:"bio,omitempty"`
	Address         *Address     `json:"address,omitempty"`
	Location        string       `json:"location" validate:"required"`
	CraftTypes      []string     `json:"craft_types" validate:"required,min=1"`
	Specializations []string     `json:"specializations,omitempty"`
	Experience      int          `json:"experience"`
	BankDetails     *BankDetails `json:"bank_details,omitempty"`
	CommissionRate  float64      `json:"commission_rate"`
}

// UpdateArtisanRequest contains data for updating an artisan
type UpdateArtisanRequest struct {
	Name            *string      `json:"name,omitempty"`
	Email           *string      `json:"email,omitempty"`
	Phone           *string      `json:"phone,omitempty"`
	ProfileImage    *string      `json:"profile_image,omitempty"`
	Bio             *string      `json:"bio,omitempty"`
	Address         *Address     `json:"address,omitempty"`
	Location        *string      `json:"location,omitempty"`
	CraftTypes      []string     `json:"craft_types,omitempty"`
	Specializations []string     `json:"specializations,omitempty"`
	Experience      *int         `json:"experience,omitempty"`
	BankDetails     *BankDetails `json:"bank_details,omitempty"`
	CommissionRate  *float64     `json:"commission_rate,omitempty"`
	Status          *ArtisanStatus `json:"status,omitempty"`
}

// CreatePayoutRequest contains data for creating an artisan payout
type CreatePayoutRequest struct {
	ArtisanID     string    `json:"artisan_id" validate:"required"`
	Amount        int64     `json:"amount" validate:"required,gt=0"`
	PaymentMethod string    `json:"payment_method" validate:"required"`
	PeriodStart   time.Time `json:"period_start" validate:"required"`
	PeriodEnd     time.Time `json:"period_end" validate:"required"`
	OrderIDs      []string  `json:"order_ids,omitempty"`
}
