package domain

import "time"

// EntityTypeManifest tags Manifest rows on the orders table.
const EntityTypeManifest = "MANIFEST"

// Manifest records a single pickup batch handed to the carrier. One Manifest
// row is written per call to ManifestService.CreatePerOrderManifest and
// RunDailyBatch, so operators can audit batch history and reconcile pickups.
type Manifest struct {
	ID                string    `json:"id" dynamodbav:"id"`
	PK                string    `json:"-" dynamodbav:"PK"`
	SK                string    `json:"-" dynamodbav:"SK"`
	GSI1PK            string    `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK            string    `json:"-" dynamodbav:"GSI1SK"`
	EntityType        string    `json:"-" dynamodbav:"entity_type"`
	ManifestID        string    `json:"manifest_id" dynamodbav:"manifest_id"` // carrier-assigned
	ShipmentCount     int       `json:"shipment_count" dynamodbav:"shipment_count"`
	AWBs              []string  `json:"awbs" dynamodbav:"awbs"`
	ShipmentMarkedIDs []string  `json:"shipment_marked_ids,omitempty" dynamodbav:"shipment_marked_ids,omitempty"`
	FailedShipmentIDs []string  `json:"failed_shipment_ids,omitempty" dynamodbav:"failed_shipment_ids,omitempty"`
	PickupDate        time.Time `json:"pickup_date" dynamodbav:"pickup_date"`
	PickupLocation    string    `json:"pickup_location" dynamodbav:"pickup_location"`
	Mode              string    `json:"mode" dynamodbav:"mode"` // "PER_ORDER" | "DAILY_BATCH"
	BaseEntity
}

// TableName returns the DynamoDB table name for Manifest.
func (m *Manifest) TableName() string {
	return TableOrders
}

// SetKeys assigns DynamoDB keys. GSI1 partitions all manifests under a
// single "MANIFEST" bucket sorted by created_at desc for list queries.
func (m *Manifest) SetKeys() {
	m.PK = "MANIFEST#" + m.ID
	m.SK = SKMetadata
	m.GSI1PK = "MANIFEST"
	m.GSI1SK = m.CreatedAt.Format("2006-01-02T15:04:05Z") + "#" + m.ID
	m.EntityType = EntityTypeManifest
}
