package domain

import (
	"strconv"
	"time"
)

// RateSource indicates how a ShippingRate row was populated.
type RateSource string

const (
	RateSourceDelhiveryAPI   RateSource = "delhivery_api"
	RateSourceManualOverride RateSource = "manual_override"
)

// ShippingRate is one entry in the rate matrix (zone × weight slab).
type ShippingRate struct {
	PK              string     `json:"-" dynamodbav:"PK"`
	SK              string     `json:"-" dynamodbav:"SK"`
	EntityType      string     `json:"-" dynamodbav:"entity_type"`
	Zone            string     `json:"zone" dynamodbav:"zone"`
	WeightSlabGrams int        `json:"weight_slab_grams" dynamodbav:"weight_slab_grams"`
	PrepaidPaise    int64      `json:"prepaid_paise" dynamodbav:"prepaid_paise"`
	CODPaise        int64      `json:"cod_paise" dynamodbav:"cod_paise"`
	RTOPaise        int64      `json:"rto_paise" dynamodbav:"rto_paise"`
	RefreshedAt     time.Time  `json:"refreshed_at" dynamodbav:"refreshed_at"`
	Source          RateSource `json:"source" dynamodbav:"source"`
	BaseEntity
}

// SetKeys assigns PK/SK for a ShippingRate.
func (r *ShippingRate) SetKeys() {
	r.PK = "RATE#" + r.Zone + "#" + strconv.Itoa(r.WeightSlabGrams)
	r.SK = SKMetadata
	r.EntityType = EntityTypeShipping
}

// TableName returns the DynamoDB table for ShippingRate.
func (r *ShippingRate) TableName() string {
	return TableShipping
}

// PincodeZone maps an Indian pincode to a carrier zone with cached metadata.
type PincodeZone struct {
	PK               string    `json:"-" dynamodbav:"PK"`
	SK               string    `json:"-" dynamodbav:"SK"`
	EntityType       string    `json:"-" dynamodbav:"entity_type"`
	Pincode          string    `json:"pincode" dynamodbav:"pincode"`
	Zone             string    `json:"zone" dynamodbav:"zone"`
	City             string    `json:"city" dynamodbav:"city"`
	State            string    `json:"state" dynamodbav:"state"`
	Serviceable      bool      `json:"serviceable" dynamodbav:"serviceable"`
	CODAvailable     bool      `json:"cod_available" dynamodbav:"cod_available"`
	PrepaidAvailable bool      `json:"prepaid_available" dynamodbav:"prepaid_available"`
	RefreshedAt      time.Time `json:"refreshed_at" dynamodbav:"refreshed_at"`
	TTL              int64     `json:"-" dynamodbav:"ttl,omitempty"` // unix seconds, 7-day cache
	BaseEntity
}

// SetKeys assigns PK/SK for a PincodeZone.
func (p *PincodeZone) SetKeys() {
	p.PK = "PIN#" + p.Pincode
	p.SK = SKMetadata
	p.EntityType = EntityTypePincodeZone
}

// TableName returns the DynamoDB table for PincodeZone.
func (p *PincodeZone) TableName() string {
	return TableShipping
}
