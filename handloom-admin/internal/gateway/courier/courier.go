// Package courier defines a carrier-agnostic shipping gateway interface.
package courier

import (
	"context"
	"net/http"
	"time"
)

// Gateway is the carrier-agnostic shipping interface.
// Implementations: delhivery.Client (production), delhivery.DevClient (local dev).
type Gateway interface {
	// Serviceability + rates
	CheckPincode(ctx context.Context, pincode string) (*PincodeInfo, error)
	FetchRateMatrix(ctx context.Context) ([]RateRow, error)

	// Forward shipment
	CreateShipment(ctx context.Context, req *CreateShipmentRequest) (*CreateShipmentResult, error)
	GenerateLabel(ctx context.Context, awb string) (string, error)

	// Manifest + pickup
	CreateManifest(ctx context.Context, awbs []string, pickupDate time.Time) (*ManifestResult, error)
	SchedulePickup(ctx context.Context, manifestID, pickupLocation string, pickupDate time.Time) error

	// Tracking
	TrackByAWB(ctx context.Context, awb string) (*TrackingInfo, error)

	// NDR
	ReAttemptDelivery(ctx context.Context, awb string, action NDRAction) error

	// Returns
	CreateReversePickup(ctx context.Context, req *ReversePickupRequest) (*ReversePickupResult, error)

	// COD
	FetchCODRemittances(ctx context.Context, from, to time.Time) ([]RemittanceRow, error)

	// Webhook
	VerifyWebhookSignature(headers http.Header, body []byte) error
	ParseWebhook(body []byte) (*WebhookEvent, error)
}

// NDRAction is the action to take on a Non-Delivery Report.
type NDRAction string

const (
	NDRActionReAttempt NDRAction = "REATTEMPT"
	NDRActionRTO       NDRAction = "RTO"
	NDRActionDefer     NDRAction = "DEFER"
)
