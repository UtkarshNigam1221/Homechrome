package delhivery

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/gateway/courier"
)

// DevClient is a stub Delhivery client for local development.
// Returns deterministic fake responses without calling the real API.
type DevClient struct{}

// NewDevClient returns a dev Delhivery client.
func NewDevClient() *DevClient { return &DevClient{} }

// Compile-time interface assertion.
var _ courier.Gateway = (*DevClient)(nil)

func (d *DevClient) CheckPincode(ctx context.Context, pincode string) (*courier.PincodeInfo, error) {
	return &courier.PincodeInfo{
		Pincode:          pincode,
		Serviceable:      true,
		Zone:             "A",
		City:             "Bangalore",
		State:            "Karnataka",
		CODAvailable:     true,
		PrepaidAvailable: true,
		EstimatedDays:    3,
	}, nil
}

func (d *DevClient) FetchRateMatrix(ctx context.Context) ([]courier.RateRow, error) {
	zones := []string{"A", "B", "C", "D", "E"}
	slabs := []int{500, 1000, 2000, 5000, 10000}
	base := int64(5000) // 50 INR base
	rows := make([]courier.RateRow, 0, len(zones)*len(slabs))
	for zi, z := range zones {
		for si, s := range slabs {
			rate := base + int64(zi)*1000 + int64(si)*2000
			rows = append(rows, courier.RateRow{
				Zone:            z,
				WeightSlabGrams: s,
				PrepaidPaise:    rate,
				CODPaise:        rate + 3500,
				RTOPaise:        rate + 5000,
			})
		}
	}
	return rows, nil
}

func (d *DevClient) CreateShipment(ctx context.Context, req *courier.CreateShipmentRequest) (*courier.CreateShipmentResult, error) {
	return &courier.CreateShipmentResult{
		AWB:               "DEVAWB" + uuid.New().String()[:8],
		CarrierShipmentID: "devship-" + uuid.New().String()[:8],
		UploadWBN:         "devwbn-" + uuid.New().String()[:8],
		EstimatedDays:     4,
		RawResponse:       []byte(`{"dev":true}`),
	}, nil
}

func (d *DevClient) GenerateLabel(ctx context.Context, awb string) (string, error) {
	return fmt.Sprintf("https://dev.local/labels/%s.pdf", awb), nil
}

func (d *DevClient) CreateManifest(ctx context.Context, awbs []string, pickupDate time.Time) (*courier.ManifestResult, error) {
	return &courier.ManifestResult{
		ManifestID: "devmanifest-" + uuid.New().String()[:8],
		PDFURL:     "https://dev.local/manifests/devmanifest.pdf",
		AWBCount:   len(awbs),
	}, nil
}

func (d *DevClient) SchedulePickup(ctx context.Context, manifestID, pickupLocation string, pickupDate time.Time) error {
	return nil
}

func (d *DevClient) TrackByAWB(ctx context.Context, awb string) (*courier.TrackingInfo, error) {
	now := time.Now().UTC()
	return &courier.TrackingInfo{
		AWB:             awb,
		Status:          courier.EventInTransit,
		CurrentLocation: "Bangalore Hub",
		LastUpdate:      now,
		History: []courier.TrackingScan{
			{Status: courier.EventManifested, Location: "Origin", Time: now.Add(-24 * time.Hour), Description: "Manifested"},
			{Status: courier.EventPickedUp, Location: "Origin Hub", Time: now.Add(-20 * time.Hour), Description: "Picked Up"},
			{Status: courier.EventInTransit, Location: "Bangalore Hub", Time: now, Description: "In Transit"},
		},
	}, nil
}

func (d *DevClient) ReAttemptDelivery(ctx context.Context, awb string, action courier.NDRAction) error {
	return nil
}

func (d *DevClient) CreateReversePickup(ctx context.Context, req *courier.ReversePickupRequest) (*courier.ReversePickupResult, error) {
	return &courier.ReversePickupResult{
		ReverseAWB:        "DEVREV" + uuid.New().String()[:8],
		CarrierShipmentID: "devrev-" + uuid.New().String()[:8],
		EstimatedDays:     4,
	}, nil
}

func (d *DevClient) FetchCODRemittances(ctx context.Context, from, to time.Time) ([]courier.RemittanceRow, error) {
	return []courier.RemittanceRow{}, nil
}

func (d *DevClient) VerifyWebhookSignature(headers http.Header, body []byte) error {
	return nil
}

func (d *DevClient) ParseWebhook(body []byte) (*courier.WebhookEvent, error) {
	return &courier.WebhookEvent{
		AWB:       "DEVAWB",
		Status:    courier.EventDelivered,
		Location:  "Bangalore",
		Timestamp: time.Now().UTC(),
	}, nil
}
