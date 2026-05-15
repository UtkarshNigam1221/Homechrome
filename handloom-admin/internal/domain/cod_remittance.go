package domain

import "time"

// CODRemittanceStatus is the reconciliation state of a remittance payout.
type CODRemittanceStatus string

const (
	CODRemittanceStatusReceived   CODRemittanceStatus = "RECEIVED"
	CODRemittanceStatusReconciled CODRemittanceStatus = "RECONCILED"
	CODRemittanceStatusUnmatched  CODRemittanceStatus = "UNMATCHED"
)

// CODEntry is one AWB-level row inside a CODRemittance.
type CODEntry struct {
	AWB         string `json:"awb" dynamodbav:"awb"`
	OrderID     string `json:"order_id" dynamodbav:"order_id"`
	AmountPaise int64  `json:"amount_paise" dynamodbav:"amount_paise"`
	Matched     bool   `json:"matched" dynamodbav:"matched"`
}

// CODRemittance is a daily payout from the carrier to the merchant.
type CODRemittance struct {
	PK            string              `json:"-" dynamodbav:"PK"`
	SK            string              `json:"-" dynamodbav:"SK"`
	EntityType    string              `json:"-" dynamodbav:"entity_type"`
	ID            string              `json:"id" dynamodbav:"id"`
	RemittanceRef string              `json:"remittance_ref" dynamodbav:"remittance_ref"`
	AmountPaise   int64               `json:"amount_paise" dynamodbav:"amount_paise"`
	RemittedAt    time.Time           `json:"remitted_at" dynamodbav:"remitted_at"`
	BankRef       string              `json:"bank_ref" dynamodbav:"bank_ref"`
	Status        CODRemittanceStatus `json:"status" dynamodbav:"status"`
	Entries       []CODEntry          `json:"entries" dynamodbav:"entries"`
	BaseEntity
}

// SetKeys assigns PK/SK for a CODRemittance.
func (c *CODRemittance) SetKeys() {
	c.PK = "REMIT#" + c.RemittanceRef
	c.SK = SKMetadata
	c.EntityType = EntityTypeCODRemittance
}

// TableName returns the DynamoDB table for CODRemittance.
func (c *CODRemittance) TableName() string {
	return TableShipping
}
