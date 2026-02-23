package shiprocket

import (
	"context"
	"fmt"
)

// Gateway defines the methods that ShippingService uses from the Shiprocket client.
// Both Client (real) and DevClient (local dev) implement this interface.
type Gateway interface {
	CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightKG float64) (*CourierServiceabilityResponse, error)
	CreateOrder(ctx context.Context, order *CreateOrderRequest) (*CreateOrderResponse, error)
	AssignAWB(ctx context.Context, shipmentID, courierID int) (*AssignAWBResponse, error)
	GenerateLabel(ctx context.Context, shipmentID int) (string, error)
	TrackByAWB(ctx context.Context, awb string) (*TrackingResponse, error)
}

// Ensure both clients satisfy the Gateway interface.
var (
	_ Gateway = (*Client)(nil)
	_ Gateway = (*DevClient)(nil)
)

// DevClient is a stub Shiprocket client for local development.
// It returns mock courier data instead of calling the real Shiprocket API.
type DevClient struct{}

// NewDevClient creates a dev Shiprocket client.
func NewDevClient() *DevClient {
	return &DevClient{}
}

func (d *DevClient) CheckServiceability(_ context.Context, pickupPincode, deliveryPincode string, _ float64) (*CourierServiceabilityResponse, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV SHIPROCKET: serviceability %s → %s  ║\n", pickupPincode, deliveryPincode)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")

	return &CourierServiceabilityResponse{
		Status: 200,
		Data: struct {
			AvailableCourierCompanies []CourierCompany `json:"available_courier_companies"`
		}{
			AvailableCourierCompanies: []CourierCompany{
				{
					CourierCompanyID: 1,
					CourierName:      "Dev Express",
					Rate:             50.0,
					ETD:              "3-5",
					EstimatedDays:    4,
				},
			},
		},
	}, nil
}

func (d *DevClient) CreateOrder(_ context.Context, order *CreateOrderRequest) (*CreateOrderResponse, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV SHIPROCKET: create order %s            ║\n", order.OrderID)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")

	return &CreateOrderResponse{
		OrderID:    99999,
		ShipmentID: 88888,
		Status:     "NEW",
		StatusCode: 1,
	}, nil
}

func (d *DevClient) AssignAWB(_ context.Context, shipmentID, _ int) (*AssignAWBResponse, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV SHIPROCKET: assign AWB for shipment %d ║\n", shipmentID)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")

	resp := &AssignAWBResponse{}
	resp.Response.Data.AWBCode = "DEV000000001"
	resp.Response.Data.CourierName = "Dev Express"
	resp.Response.Data.AppliedWeight = 0.5
	return resp, nil
}

func (d *DevClient) GenerateLabel(_ context.Context, shipmentID int) (string, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV SHIPROCKET: generate label for %d      ║\n", shipmentID)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")

	return "https://dev-label.example.com/label.pdf", nil
}

func (d *DevClient) TrackByAWB(_ context.Context, awb string) (*TrackingResponse, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV SHIPROCKET: track AWB %s               ║\n", awb)
	fmt.Printf("╚══════════════════════════════════════════════╝\n\n")

	return &TrackingResponse{
		TrackingData: struct {
			TrackStatus    int    `json:"track_status"`
			ShipmentStatus int    `json:"shipment_status"`
			CurrentStatus  string `json:"current_status"`
		}{
			TrackStatus:    1,
			ShipmentStatus: 7,
			CurrentStatus:  "DELIVERED",
		},
	}, nil
}
