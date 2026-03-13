package domain

import "time"

// ==================== PAYMENT ENTITY ====================

// PaymentProvider defines the payment gateway provider
type PaymentProvider string

const (
	PaymentProviderPhonePe PaymentProvider = "PHONEPE"
)

// PaymentMethod defines the method of payment
type PaymentMethod string

const (
	PaymentMethodUPI        PaymentMethod = "UPI"
	PaymentMethodCard       PaymentMethod = "CARD"
	PaymentMethodNetBanking PaymentMethod = "NET_BANKING"
	PaymentMethodWallet     PaymentMethod = "WALLET"
)

// Payment represents a payment transaction
type Payment struct {
	ID                    string          `json:"id" dynamodbav:"id"`
	PK                    string          `json:"-" dynamodbav:"PK"`
	SK                    string          `json:"-" dynamodbav:"SK"`
	GSI1PK                string          `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK                string          `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK                string          `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK                string          `json:"-" dynamodbav:"GSI2SK"`
	EntityType            string          `json:"-" dynamodbav:"entity_type"`
	OrderID               string          `json:"order_id" dynamodbav:"order_id"`
	CustomerID            string          `json:"customer_id" dynamodbav:"customer_id"`
	Amount                int64           `json:"amount" dynamodbav:"amount"` // in paise
	Currency              string          `json:"currency" dynamodbav:"currency"`
	Status                PaymentStatus   `json:"status" dynamodbav:"status"`
	Provider              PaymentProvider `json:"provider" dynamodbav:"provider"`
	MerchantTransactionID string          `json:"merchant_transaction_id" dynamodbav:"merchant_transaction_id"`
	ProviderTransactionID string          `json:"provider_transaction_id,omitempty" dynamodbav:"provider_transaction_id,omitempty"`
	PaymentMethod         PaymentMethod   `json:"payment_method,omitempty" dynamodbav:"payment_method,omitempty"`
	ProviderResponse      string          `json:"provider_response,omitempty" dynamodbav:"provider_response,omitempty"`
	InitiatedAt           time.Time       `json:"initiated_at" dynamodbav:"initiated_at"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
	RefundAmount          int64           `json:"refund_amount,omitempty" dynamodbav:"refund_amount,omitempty"` // in paise
	RefundedAt            *time.Time      `json:"refunded_at,omitempty" dynamodbav:"refunded_at,omitempty"`
	BaseEntity
}

// TableName returns the DynamoDB table name for Payment
func (p *Payment) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for Payment
func (p *Payment) SetKeys() {
	p.PK = "PAYMENT#" + p.ID
	p.SK = SKMetadata
	p.GSI1PK = "ORDER#" + p.OrderID
	p.GSI1SK = "PAYMENT#" + p.InitiatedAt.Format("2006-01-02T15:04:05Z")
	p.GSI2PK = "PAYMENT_TXN"
	p.GSI2SK = p.MerchantTransactionID
	p.EntityType = "PAYMENT"
}

// ==================== PAYMENT REQUEST TYPES ====================

// InitiatePaymentRequest contains data for initiating a payment
type InitiatePaymentRequest struct {
	OrderID    string `json:"order_id" validate:"required"`
	CustomerID string `json:"customer_id" validate:"required"`
	Amount     int64  `json:"amount" validate:"required,gt=0"`
	Phone      string `json:"phone" validate:"required"`
}
