package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

// When an order has admin-set tracking but no Shipment record (the manual-
// fulfillment path after the Shiprocket removal), TrackOrder must surface the
// order-level tracking number/carrier/status in the response.
func TestTrackOrder_FallsBackToOrderTracking(t *testing.T) {
	ctrl := gomock.NewController(t)

	orderRepo := mocks.NewMockOrderRepository(ctrl)
	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

	order := &domain.Order{
		ID:              "order-1",
		OrderNumber:     "HC-123",
		Status:          domain.OrderStatusShipped,
		TrackingNumber:  "TRK123",
		ShippingCarrier: "BlueDart",
	}
	orderRepo.EXPECT().GetByOrderNumber(gomock.Any(), "HC-123").Return(order, nil)
	shipmentRepo.EXPECT().GetByOrderID(gomock.Any(), "order-1").Return(nil, errors.NotFound("Shipment"))

	srv := httptest.NewServer(NewTrackingHandler(orderRepo, shipmentRepo).Routes())
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/HC-123", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Success bool             `json:"success"`
		Data    TrackingResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.True(t, body.Success)
	require.NotNil(t, body.Data.Shipment)
	assert.Equal(t, "TRK123", body.Data.Shipment.AWBNumber)
	assert.Equal(t, "BlueDart", body.Data.Shipment.CourierName)
	assert.Equal(t, "SHIPPED", body.Data.Shipment.Status)
}
