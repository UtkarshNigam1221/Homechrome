package domain

import (
	"context"
	"time"
)

//go:generate mockgen -source=refund.go -destination=../mocks/refund_mock.go -package=mocks

// RefundStatus tracks a refund from initiation to a terminal outcome. PhonePe
// has no intermediate "accepted" state: a refund goes PENDING then settles.
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusCompleted RefundStatus = "COMPLETED"
	RefundStatusFailed    RefundStatus = "FAILED"
)

// RefundReason is bounded so it can label a metric without unbounded
// cardinality. Anything that needs explaining belongs in an order note.
type RefundReason string

const (
	RefundReasonOutOfStock      RefundReason = "OUT_OF_STOCK"
	RefundReasonDamaged         RefundReason = "DAMAGED"
	RefundReasonCustomerRequest RefundReason = "CUSTOMER_REQUEST"
	RefundReasonPricingError    RefundReason = "PRICING_ERROR"
	RefundReasonOther           RefundReason = "OTHER"
)

// IsValid reports whether the reason is one this system recognizes.
func (r RefundReason) IsValid() bool {
	switch r {
	case RefundReasonOutOfStock, RefundReasonDamaged, RefundReasonCustomerRequest,
		RefundReasonPricingError, RefundReasonOther:
		return true
	default:
		return false
	}
}

// RefundItem is one order line going back, in whole units.
type RefundItem struct {
	OrderItemID string `json:"order_item_id" dynamodbav:"order_item_id"`
	ProductID   string `json:"product_id" dynamodbav:"product_id"`
	ProductName string `json:"product_name" dynamodbav:"product_name"`
	Quantity    int    `json:"quantity" dynamodbav:"quantity"`

	// Amount is this line's share of the refund: its own value less its
	// prorated share of the order discount and tax.
	Amount int64 `json:"amount" dynamodbav:"amount"`

	// Restock returns the units to sale. False writes them off, which is the
	// default: "cannot serve" usually means the goods are not there.
	Restock bool `json:"restock" dynamodbav:"restock"`
}

// Refund is one attempt to send money back for part or all of an order. Separate from
// Payment because an order can be refunded line by line, several times over.
type Refund struct {
	refundKeys

	ID         string `json:"id" dynamodbav:"id"`
	OrderID    string `json:"order_id" dynamodbav:"order_id"`
	PaymentID  string `json:"payment_id" dynamodbav:"payment_id"`
	CustomerID string `json:"customer_id" dynamodbav:"customer_id"`

	Amount int64        `json:"amount" dynamodbav:"amount"` // paise, derived server-side
	Status RefundStatus `json:"status" dynamodbav:"status"`
	Reason RefundReason `json:"reason" dynamodbav:"reason"`
	Items  []RefundItem `json:"items" dynamodbav:"items"`

	// MerchantRefundID is ours, unique per attempt, and what the status endpoint takes.
	// ProviderRefundID is PhonePe's, empty until initiation returns; webhooks use it.
	MerchantRefundID string `json:"merchant_refund_id" dynamodbav:"merchant_refund_id"`
	ProviderRefundID string `json:"provider_refund_id,omitempty" dynamodbav:"provider_refund_id,omitempty"`

	ErrorCode         string `json:"error_code,omitempty" dynamodbav:"error_code,omitempty"`
	DetailedErrorCode string `json:"detailed_error_code,omitempty" dynamodbav:"detailed_error_code,omitempty"`

	InitiatedAt time.Time  `json:"initiated_at" dynamodbav:"initiated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
	CreatedBy   string     `json:"created_by" dynamodbav:"created_by"`

	// CreatedByName is resolved when the list is read, not stored: created_by
	// holds an opaque user id, which is no use to whoever reads back who sent
	// money out. Empty for a refund no admin raised.
	CreatedByName string `json:"created_by_name,omitempty" dynamodbav:"-"`
}

// DynamoDB keys; refunds live in the orders table. GSI2 carries two partition shapes
// so the re-check (our id) and the webhook (PhonePe's) share one index, unhotly.
type refundKeys struct {
	PK     string `json:"-" dynamodbav:"PK"`
	SK     string `json:"-" dynamodbav:"SK"`
	GSI1PK string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK string `json:"-" dynamodbav:"GSI1SK"`

	// Omitted until initiation returns: DynamoDB rejects an empty indexed key, so a
	// half-written refund carries no GSI2 keys rather than blank ones.
	GSI2PK string `json:"-" dynamodbav:"GSI2PK,omitempty"`
	GSI2SK string `json:"-" dynamodbav:"GSI2SK,omitempty"`

	EntityType string `json:"-" dynamodbav:"entity_type"`
}

// SetKeys fills the DynamoDB keys for a refund.
func (r *Refund) SetKeys() {
	r.PK = "REFUND#" + r.ID
	r.SK = SKMetadata
	r.GSI1PK = "ORDER#" + r.OrderID
	r.GSI1SK = "REFUND#" + r.InitiatedAt.Format("2006-01-02T15:04:05Z")
	r.EntityType = "REFUND"

	if r.ProviderRefundID != "" {
		r.GSI2PK = "REFUND_PROVIDER#" + r.ProviderRefundID
		r.GSI2SK = SKMetadata
	}
}

// IsTerminal reports whether the refund has settled either way.
func (r *Refund) IsTerminal() bool {
	return r.Status == RefundStatusCompleted || r.Status == RefundStatusFailed
}

// CreateRefundItemRequest is one requested line. Quantity only — the server
// derives the money.
type CreateRefundItemRequest struct {
	OrderItemID string `json:"order_item_id" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`
	Restock     bool   `json:"restock"`
}

// CreateRefundRequest carries what to refund, never how much. A client-supplied
// amount is not accepted: money is not a client input.
type CreateRefundRequest struct {
	Reason RefundReason              `json:"reason" validate:"required"`
	Items  []CreateRefundItemRequest `json:"items" validate:"required,min=1,dive"`
}

// RefundRepository persists refunds.
type RefundRepository interface {
	Create(ctx context.Context, refund *Refund) error
	GetByID(ctx context.Context, id string) (*Refund, error)

	// ListByOrder returns an order's refunds, oldest first.
	ListByOrder(ctx context.Context, orderID string) ([]*Refund, error)

	// GetByProviderRefundID finds the refund a webhook is about. Webhooks carry only
	// PhonePe's id, and an order can have several refunds.
	GetByProviderRefundID(ctx context.Context, providerRefundID string) (*Refund, error)

	// SetProviderRefundID records PhonePe's id once initiation returns.
	SetProviderRefundID(ctx context.Context, id, providerRefundID string) error

	// Settle moves a refund to a terminal state, only from PENDING. That condition is
	// the gate: of two concurrent deliveries one wins, and only it runs the effects.
	Settle(ctx context.Context, id string, status RefundStatus, completedAt time.Time, errorCode, detailedErrorCode string) error
}

// RefundService owns the refund lifecycle.
type RefundService interface {
	Create(ctx context.Context, orderID string, req CreateRefundRequest, createdBy string) (*Refund, error)
	ListByOrder(ctx context.Context, orderID string) ([]*Refund, error)

	// HandleRefundCompleted and HandleRefundFailed settle a refund from a
	// provider webhook, keyed by PhonePe's refund id.
	HandleRefundCompleted(ctx context.Context, providerRefundID string) error
	HandleRefundFailed(ctx context.Context, providerRefundID, errorCode, detailedErrorCode string) error

	// RecheckStatus asks the provider directly: the escape hatch for a webhook that
	// never came, and the only recovery when no provider id was ever stored.
	//
	// orderID is the one in the route: the refund must belong to it, or any refund
	// would be reachable through any order's URL.
	RecheckStatus(ctx context.Context, orderID, refundID string) (*Refund, error)
}
