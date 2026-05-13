package phonepe

import (
	"context"
	"fmt"
)

// Gateway defines the methods that PaymentService uses from the PhonePe client.
// Both Client (real) and DevClient (local dev) implement this interface.
type Gateway interface {
	InitiatePayment(ctx context.Context, merchantTxnID, customerID string, amount int64, orderID string) (string, error)
	CheckPaymentStatus(ctx context.Context, merchantTxnID string) (*StatusResponse, error)
	VerifyWebhookSignature(username, password, authHeader string) bool
}

// Ensure both clients satisfy the Gateway interface.
var (
	_ Gateway = (*Client)(nil)
	_ Gateway = (*DevClient)(nil)
)

// DevClient is a stub PhonePe client for local development.
// It skips real payment and returns a redirect to a local success page.
type DevClient struct {
	redirectURL string
}

// NewDevClient creates a dev PhonePe client.
func NewDevClient(redirectURL string) *DevClient {
	return &DevClient{redirectURL: redirectURL}
}

func (d *DevClient) InitiatePayment(_ context.Context, merchantTxnID, _ string, amount int64, orderID string) (string, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV PHONEPE: payment %s  ║\n", merchantTxnID)
	fmt.Printf("║  Amount: ₹%.2f                                  ║\n", float64(amount)/100)
	fmt.Printf("╚══════════════════════════════════════════════════╝\n\n")

	redirect := d.redirectURL
	if redirect == "" {
		redirect = "http://localhost:3000/checkout/confirmation"
	}
	return fmt.Sprintf("%s?order_id=%s&dev_payment=%s", redirect, orderID, merchantTxnID), nil
}

func (d *DevClient) CheckPaymentStatus(_ context.Context, merchantTxnID string) (*StatusResponse, error) {
	return &StatusResponse{
		OrderID: merchantTxnID,
		State:   StateCompleted,
		PaymentDetails: []PaymentDetail{
			{
				TransactionID: "DEV-TXN-" + merchantTxnID,
				PaymentMode:   "UPI_INTENT",
				State:         StateCompleted,
			},
		},
	}, nil
}

func (d *DevClient) VerifyWebhookSignature(_, _, _ string) bool {
	return true
}
