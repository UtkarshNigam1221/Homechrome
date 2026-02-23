package phonepe

import (
	"context"
	"fmt"
)

// Gateway defines the methods that PaymentService uses from the PhonePe client.
// Both Client (real) and DevClient (local dev) implement this interface.
type Gateway interface {
	InitiatePayment(ctx context.Context, merchantTxnID, customerID string, amount int64) (string, error)
	CheckPaymentStatus(ctx context.Context, merchantTxnID string) (*StatusResponse, error)
	VerifyWebhookSignature(responseBase64, xVerifyHeader string) bool
	DecodeWebhookResponse(responseBase64 string) (*StatusResponse, error)
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

func (d *DevClient) InitiatePayment(_ context.Context, merchantTxnID, _ string, amount int64) (string, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV PHONEPE: payment %s  ║\n", merchantTxnID)
	fmt.Printf("║  Amount: ₹%.2f                                  ║\n", float64(amount)/100)
	fmt.Printf("╚══════════════════════════════════════════════════╝\n\n")

	// Return the redirect URL with the merchant txn ID so the frontend
	// can poll for status. The dev webhook handler will auto-complete it.
	redirect := d.redirectURL
	if redirect == "" {
		redirect = "http://localhost:3000"
	}
	return fmt.Sprintf("%s/account/orders?dev_payment=%s", redirect, merchantTxnID), nil
}

func (d *DevClient) CheckPaymentStatus(_ context.Context, merchantTxnID string) (*StatusResponse, error) {
	// In dev mode, always return COMPLETED
	resp := &StatusResponse{
		Success: true,
		Code:    "PAYMENT_SUCCESS",
		Message: "Dev payment completed",
	}
	resp.Data.MerchantTransactionID = merchantTxnID
	resp.Data.TransactionID = "DEV-TXN-" + merchantTxnID
	resp.Data.State = "COMPLETED"
	resp.Data.ResponseCode = "SUCCESS"
	resp.Data.PaymentInstrument.Type = "UPI"
	return resp, nil
}

func (d *DevClient) VerifyWebhookSignature(_, _ string) bool {
	return true
}

func (d *DevClient) DecodeWebhookResponse(_ string) (*StatusResponse, error) {
	return &StatusResponse{
		Success: true,
		Code:    "PAYMENT_SUCCESS",
		Message: "Dev payment completed",
	}, nil
}
