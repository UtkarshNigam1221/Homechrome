package delhivery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/gateway/courier"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient(Config{APIToken: "tok-123", BaseURL: srv.URL, ClientName: "test", PickupLocation: "Primary"})
	return c, srv
}

func TestCheckPincode(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/c/api/pin-codes/json/", r.URL.Path)
		assert.Equal(t, "Token tok-123", r.Header.Get("Authorization"))
		assert.Equal(t, "560001", r.URL.Query().Get("filter_codes"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"delivery_codes": []map[string]any{
				{"postal_code": map[string]any{
					"pin": "560001", "district": "Bangalore", "state_code": "KA",
					"cod": "Y", "pre_paid": "Y", "max_amount": 50000, "delivery_days": 3, "remarks": "A",
				}},
			},
		})
	})
	info, err := c.CheckPincode(context.Background(), "560001")
	require.NoError(t, err)
	assert.True(t, info.Serviceable)
	assert.Equal(t, "560001", info.Pincode)
	assert.Equal(t, "A", info.Zone)
	assert.Equal(t, "Bangalore", info.City)
	assert.Equal(t, "KA", info.State)
	assert.True(t, info.CODAvailable)
	assert.True(t, info.PrepaidAvailable)
	assert.Equal(t, 3, info.EstimatedDays)
}

func TestFetchRateMatrix(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/kinko/v1/invoice/charges/.json", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"total_amount": 75.50, "gross_amount": 75.50, "zone": "A"})
	})
	rows, err := c.FetchRateMatrix(context.Background())
	require.NoError(t, err)
	// 5 zones × 5 slabs = 25 rows
	assert.Len(t, rows, 25)
	assert.Equal(t, int64(7550), rows[0].PrepaidPaise)
}

func TestCreateShipment(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/cmu/create.json", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"packages":   []map[string]any{{"waybill": "AWB123", "refnum": "ORDER-1", "status": "Success"}},
			"upload_wbn": "WBN-9",
			"success":    true,
		})
	})
	req := &courier.CreateShipmentRequest{
		OrderID:        "ORDER-1",
		PickupLocation: "Primary",
		Customer: courier.Address{
			FirstName: "Jane", LastName: "Doe", Phone: "9999999999",
			AddressLine1: "1 MG Road", City: "Bangalore", State: "Karnataka",
			Pincode: "560001", Country: "India",
		},
		Items:              []courier.ShipmentItem{{Name: "Saree", SKU: "SK1", Quantity: 1, UnitPaise: 250000}},
		PaymentMode:        courier.PaymentPrepaid,
		WeightGrams:        500,
		LengthCm:           30,
		BreadthCm:          25,
		HeightCm:           5,
		DeclaredValuePaise: 250000,
	}
	res, err := c.CreateShipment(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "AWB123", res.AWB)
	assert.Equal(t, "WBN-9", res.UploadWBN)
}

func TestGenerateLabel(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/p/packing_slip", r.URL.Path)
		assert.Equal(t, "AWB123", r.URL.Query().Get("wbns"))
		_, _ = w.Write([]byte(`{"packages_found":[{"pdf_download_link":"https://delhivery.com/labels/AWB123.pdf"}]}`))
	})
	url, err := c.GenerateLabel(context.Background(), "AWB123")
	require.NoError(t, err)
	assert.Equal(t, "https://delhivery.com/labels/AWB123.pdf", url)
}

func TestCreateManifest(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/p/manifest", r.URL.Path)
		var body manifestRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, []string{"AWB1", "AWB2"}, body.Waybills)
		assert.Equal(t, "Primary", body.PickupLocation)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"manifest_id": "MAN-1", "pdf_url": "https://delhivery.com/manifests/MAN-1.pdf", "success": true,
		})
	})
	res, err := c.CreateManifest(context.Background(), []string{"AWB1", "AWB2"}, time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, "MAN-1", res.ManifestID)
	assert.Equal(t, 2, res.AWBCount)
}

func TestSchedulePickup(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fm/request/new/", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"pickup_id": "PCK-1", "success": true})
	})
	err := c.SchedulePickup(context.Background(), "MAN-1", "Primary", time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC))
	require.NoError(t, err)
}

func TestTrackByAWB(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/packages/json/", r.URL.Path)
		assert.Equal(t, "AWB123", r.URL.Query().Get("waybill"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ShipmentData": []map[string]any{{
				"Shipment": map[string]any{
					"AWB": "AWB123",
					"Status": map[string]any{
						"Status":         "In Transit",
						"StatusLocation": "Mumbai Hub",
						"StatusDateTime": "2026-05-16T10:30:00",
					},
					"Scans": []map[string]any{
						{"ScanDetail": map[string]any{
							"Scan": "Manifested", "ScanLocation": "Origin",
							"ScanDateTime": "2026-05-15T09:00:00", "Instructions": "Manifested",
						}},
						{"ScanDetail": map[string]any{
							"Scan": "In Transit", "ScanLocation": "Mumbai Hub",
							"ScanDateTime": "2026-05-16T10:30:00", "Instructions": "In Transit",
						}},
					},
				},
			}},
		})
	})
	info, err := c.TrackByAWB(context.Background(), "AWB123")
	require.NoError(t, err)
	assert.Equal(t, "AWB123", info.AWB)
	assert.Equal(t, courier.EventInTransit, info.Status)
	assert.Equal(t, "Mumbai Hub", info.CurrentLocation)
	assert.Len(t, info.History, 2)
}

func TestReAttemptDelivery(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/p/update", r.URL.Path)
		var body ndrActionRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "AWB123", body.Waybill)
		assert.Equal(t, "RE-ATTEMPT", body.Act)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	err := c.ReAttemptDelivery(context.Background(), "AWB123", courier.NDRActionReAttempt)
	require.NoError(t, err)
}

func TestCreateReversePickup(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/cmu/create.json", r.URL.Path)
		var body createShipmentBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		require.Len(t, body.Shipments, 1)
		assert.Equal(t, "REV-ORDER-1", body.Shipments[0].OrderID)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"packages":   []map[string]any{{"waybill": "REVAWB1", "refnum": "REV-ORDER-1", "status": "Success"}},
			"upload_wbn": "REVWBN",
			"success":    true,
		})
	})
	req := &courier.ReversePickupRequest{
		OriginalOrderID: "ORDER-1",
		OriginalAWB:     "AWB-1",
		Customer: courier.Address{
			FirstName: "Jane", LastName: "Doe", Phone: "9999999999",
			AddressLine1: "1 MG Road", City: "Bangalore", State: "Karnataka",
			Pincode: "560001", Country: "India",
		},
		PickupLocation: "Primary",
		Items:          []courier.ShipmentItem{{Name: "Saree", SKU: "SK1", Quantity: 1, UnitPaise: 250000}},
		Reason:         "Defective",
	}
	res, err := c.CreateReversePickup(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "REVAWB1", res.ReverseAWB)
}

func TestFetchCODRemittances(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/cmu/get_invoice", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"awb": "AWB1", "order_id": "ORDER-1", "amount_paid": 1200.50, "utr": "UTR-1", "paid_date": "2026-05-15T10:00:00Z"},
			},
		})
	})
	rows, err := c.FetchCODRemittances(context.Background(), time.Now().Add(-24*time.Hour), time.Now())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "AWB1", rows[0].AWB)
	assert.Equal(t, int64(120050), rows[0].AmountPaise)
}

func TestVerifyWebhookSignature(t *testing.T) {
	c := NewClient(Config{WebhookToken: "secret"})
	body := []byte(`{"waybill":"AWB1"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	headers := http.Header{}
	headers.Set("X-Delhivery-Signature", sig)
	err := c.VerifyWebhookSignature(headers, body)
	require.NoError(t, err)

	headers.Set("X-Delhivery-Signature", "wrong")
	err = c.VerifyWebhookSignature(headers, body)
	require.Error(t, err)
}

func TestParseWebhook(t *testing.T) {
	c := NewClient(Config{})
	body := []byte(`{"waybill":"AWB1","status":"Delivered","location":"Bangalore","timestamp":"2026-05-16T10:00:00Z"}`)
	ev, err := c.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, "AWB1", ev.AWB)
	assert.Equal(t, courier.EventDelivered, ev.Status)
}
