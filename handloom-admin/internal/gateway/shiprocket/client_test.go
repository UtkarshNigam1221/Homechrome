package shiprocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(AuthResponse{Token: "test-token-123"})

		case "/courier/serviceability/":
			assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(CourierServiceabilityResponse{
				Status: 200,
				Data: struct {
					AvailableCourierCompanies []CourierCompany `json:"available_courier_companies"`
				}{
					AvailableCourierCompanies: []CourierCompany{
						{CourierCompanyID: 1, CourierName: "Delhivery", Rate: 75.0, EstimatedDays: 3},
						{CourierCompanyID: 2, CourierName: "BlueDart", Rate: 120.0, EstimatedDays: 2},
					},
				},
			})

		case "/orders/create/adhoc":
			assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(CreateOrderResponse{
				OrderID:    12345,
				ShipmentID: 67890,
				Status:     "NEW",
				StatusCode: 1,
			})

		case "/courier/assign/awb":
			assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(AssignAWBResponse{
				Response: struct {
					Data struct {
						AWBCode       string  `json:"awb_code"`
						CourierName   string  `json:"courier_name"`
						AppliedWeight float64 `json:"applied_weight"`
					} `json:"data"`
				}{
					Data: struct {
						AWBCode       string  `json:"awb_code"`
						CourierName   string  `json:"courier_name"`
						AppliedWeight float64 `json:"applied_weight"`
					}{
						AWBCode:     "AWB123456789",
						CourierName: "Delhivery",
					},
				},
			})

		case "/courier/track/awb/AWB123":
			assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TrackingResponse{
				TrackingData: struct {
					TrackStatus    int    `json:"track_status"`
					ShipmentStatus int    `json:"shipment_status"`
					CurrentStatus  string `json:"current_status"`
				}{
					TrackStatus:    1,
					ShipmentStatus: 7,
					CurrentStatus:  "DELIVERED",
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestClient_Authenticate(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	client := NewClient(Config{
		Email:    "test@example.com",
		Password: "password",
		BaseURL:  server.URL,
	})

	token, err := client.Authenticate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-token-123", token)

	// Second call should use cache
	token2, err := client.Authenticate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, token, token2)
}

func TestClient_CheckServiceability(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	client := NewClient(Config{Email: "test@example.com", Password: "pass", BaseURL: server.URL})
	result, err := client.CheckServiceability(context.Background(), "110001", "400001", 0.5)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result.Data.AvailableCourierCompanies))
	assert.Equal(t, "Delhivery", result.Data.AvailableCourierCompanies[0].CourierName)
}

func TestClient_CreateOrder(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	client := NewClient(Config{Email: "test@example.com", Password: "pass", BaseURL: server.URL})
	result, err := client.CreateOrder(context.Background(), &CreateOrderRequest{
		OrderID:       "ORD-001",
		PaymentMethod: "Prepaid",
		SubTotal:      999.00,
	})
	require.NoError(t, err)
	assert.Equal(t, 12345, result.OrderID)
	assert.Equal(t, 67890, result.ShipmentID)
}

func TestClient_AssignAWB(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	client := NewClient(Config{Email: "test@example.com", Password: "pass", BaseURL: server.URL})
	result, err := client.AssignAWB(context.Background(), 67890, 1)
	require.NoError(t, err)
	assert.Equal(t, "AWB123456789", result.Response.Data.AWBCode)
}

func TestClient_TrackByAWB(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	client := NewClient(Config{Email: "test@example.com", Password: "pass", BaseURL: server.URL})
	result, err := client.TrackByAWB(context.Background(), "AWB123")
	require.NoError(t, err)
	assert.Equal(t, "DELIVERED", result.TrackingData.CurrentStatus)
}

func TestClient_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(Config{Email: "bad", Password: "bad", BaseURL: server.URL})
	_, err := client.Authenticate(context.Background())
	assert.Error(t, err)
}
