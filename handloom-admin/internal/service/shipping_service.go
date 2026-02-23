package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/shiprocket"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// ShippingService implements domain.ShippingService
type ShippingService struct {
	shipmentRepo  domain.ShipmentRepository
	orderRepo     domain.OrderRepository
	shiprocket    shiprocket.Gateway
	pickupPincode string
	logger        *logger.Logger
}

// NewShippingService creates a new ShippingService
func NewShippingService(
	shipmentRepo domain.ShipmentRepository,
	orderRepo domain.OrderRepository,
	shiprocketClient shiprocket.Gateway,
	pickupPincode string,
	logger *logger.Logger,
) *ShippingService {
	return &ShippingService{
		shipmentRepo:  shipmentRepo,
		orderRepo:     orderRepo,
		shiprocket:    shiprocketClient,
		pickupPincode: pickupPincode,
		logger:        logger,
	}
}

// CheckServiceability checks courier availability for a given route
func (s *ShippingService) CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightGrams int) (*domain.ServiceabilityResult, error) {
	// Convert grams to kilograms for Shiprocket API
	weightKG := float64(weightGrams) / 1000.0

	resp, err := s.shiprocket.CheckServiceability(ctx, pickupPincode, deliveryPincode, weightKG)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to check serviceability for pincodes %s -> %s", pickupPincode, deliveryPincode)
		return nil, errors.Wrap(err, "Failed to check shipping serviceability")
	}

	couriers := resp.Data.AvailableCourierCompanies
	if len(couriers) == 0 {
		return &domain.ServiceabilityResult{
			Serviceable: false,
			Couriers:    nil,
		}, nil
	}

	courierOptions := make([]domain.CourierOption, 0, len(couriers))
	for _, c := range couriers {
		courierOptions = append(courierOptions, domain.CourierOption{
			ID:            c.CourierCompanyID,
			Name:          c.CourierName,
			Rate:          int64(math.Round(c.Rate * 100)), // convert rupees to paise
			EstimatedDays: c.EstimatedDays,
		})
	}

	return &domain.ServiceabilityResult{
		Serviceable: true,
		Couriers:    courierOptions,
	}, nil
}

// CreateShipment creates a shipment for an order via Shiprocket
func (s *ShippingService) CreateShipment(ctx context.Context, order *domain.Order) (*domain.Shipment, error) {
	if order.ShippingAddress == nil {
		return nil, errors.Validation("Order has no shipping address")
	}

	// Build Shiprocket order items
	srItems := make([]shiprocket.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		srItems = append(srItems, shiprocket.OrderItem{
			Name:         item.ProductName,
			SKU:          item.ProductSKU,
			Units:        item.Quantity,
			SellingPrice: float64(item.UnitPrice) / 100.0, // paise to rupees
		})
	}

	// Determine payment method
	paymentMethod := "Prepaid"
	if order.PaymentMethod == "COD" {
		paymentMethod = "COD"
	}

	// Calculate total weight in kg (default to 0.5 kg if not available)
	weightKG := 0.5

	// Build the Shiprocket create order request
	addr := order.ShippingAddress
	createReq := &shiprocket.CreateOrderRequest{
		OrderID:           order.ID,
		OrderDate:         order.CreatedAt.Format("2006-01-02 15:04"),
		PickupLocation:    "Primary",
		BillingCustomer:   addr.FirstName,
		BillingLastName:   addr.LastName,
		BillingAddress:    addr.AddressLine1,
		BillingCity:       addr.City,
		BillingPincode:    addr.PostalCode,
		BillingState:      addr.State,
		BillingCountry:    addr.Country,
		BillingEmail:      order.CustomerEmail,
		BillingPhone:      addr.Phone,
		ShippingIsBilling: true,
		OrderItems:        srItems,
		PaymentMethod:     paymentMethod,
		SubTotal:          float64(order.Subtotal) / 100.0, // paise to rupees
		Length:            30,
		Breadth:           25,
		Height:            5,
		Weight:            weightKG,
	}

	// Create order in Shiprocket
	createResp, err := s.shiprocket.CreateOrder(ctx, createReq)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to create Shiprocket order for order %s", order.ID)
		return nil, errors.Wrap(err, "Failed to create shipping order")
	}

	// Assign AWB (use first available courier)
	awbResp, err := s.shiprocket.AssignAWB(ctx, createResp.ShipmentID, 0)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to assign AWB for shipment %d", createResp.ShipmentID)
		return nil, errors.Wrap(err, "Failed to assign AWB number")
	}

	// Generate shipping label
	labelURL, err := s.shiprocket.GenerateLabel(ctx, createResp.ShipmentID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to generate label for shipment %d", createResp.ShipmentID)
		return nil, errors.Wrap(err, "Failed to generate shipping label")
	}

	// Create domain shipment
	now := time.Now()
	shipment := &domain.Shipment{
		ID:                 uuid.New().String(),
		OrderID:            order.ID,
		Provider:           "shiprocket",
		ProviderOrderID:    strconv.Itoa(createResp.OrderID),
		ProviderShipmentID: strconv.Itoa(createResp.ShipmentID),
		AWBNumber:          awbResp.Response.Data.AWBCode,
		CourierName:        awbResp.Response.Data.CourierName,
		Status:             domain.ShipmentStatusCreated,
		LabelURL:           labelURL,
		WeightGrams:        int(weightKG * 1000),
		ShippedAt:          &now,
	}

	// Save shipment to repository
	if err := s.shipmentRepo.Create(ctx, shipment); err != nil {
		s.logger.WithError(err).Errorf("Failed to save shipment for order %s", order.ID)
		return nil, errors.Wrap(err, "Failed to save shipment")
	}

	// Update order status to SHIPPED with tracking info
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatusShipped, "system"); err != nil {
		s.logger.WithError(err).Errorf("Failed to update order status to SHIPPED for order %s", order.ID)
	}

	if err := s.orderRepo.UpdateTracking(ctx, order.ID, shipment.AWBNumber, shipment.CourierName); err != nil {
		s.logger.WithError(err).Errorf("Failed to update tracking info for order %s", order.ID)
	}

	return shipment, nil
}

// TrackShipment tracks a shipment by order ID
func (s *ShippingService) TrackShipment(ctx context.Context, orderID string) (*domain.Shipment, error) {
	shipment, err := s.shipmentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// If we have an AWB number, fetch live tracking from Shiprocket
	if shipment.AWBNumber != "" {
		trackResp, err := s.shiprocket.TrackByAWB(ctx, shipment.AWBNumber)
		if err != nil {
			s.logger.WithError(err).Errorf("Failed to track AWB %s for order %s", shipment.AWBNumber, orderID)
			// Return cached shipment data even if live tracking fails
			return shipment, nil
		}

		newStatus := mapShiprocketStatus(trackResp.TrackingData.CurrentStatus)
		if newStatus != shipment.Status {
			updates := map[string]interface{}{}
			if newStatus == domain.ShipmentStatusDelivered {
				now := time.Now()
				updates["delivered_at"] = now.Format(time.RFC3339)
			}
			if err := s.shipmentRepo.UpdateStatus(ctx, orderID, shipment.ID, newStatus, updates); err != nil {
				s.logger.WithError(err).Errorf("Failed to update shipment status for order %s", orderID)
			}
			shipment.Status = newStatus

			// Update corresponding order status if delivered
			if newStatus == domain.ShipmentStatusDelivered {
				if err := s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusDelivered, "system"); err != nil {
					s.logger.WithError(err).Errorf("Failed to update order status to DELIVERED for order %s", orderID)
				}
			}
		}
	}

	return shipment, nil
}

// shiprocketWebhookPayload represents the Shiprocket webhook payload
type shiprocketWebhookPayload struct {
	AWB           string `json:"awb"`
	CurrentStatus string `json:"current_status"`
	ShipmentID    int    `json:"shipment_id"`
	OrderID       string `json:"order_id"`
	ETDL          string `json:"etd"`
}

// HandleWebhook handles incoming Shiprocket webhook events
func (s *ShippingService) HandleWebhook(ctx context.Context, payload []byte, token string) error {
	var webhook shiprocketWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return errors.BadRequest("Invalid webhook payload")
	}

	if webhook.OrderID == "" {
		return errors.BadRequest("Missing order_id in webhook payload")
	}

	// Retrieve the shipment for this order
	shipment, err := s.shipmentRepo.GetByOrderID(ctx, webhook.OrderID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to find shipment for webhook order %s", webhook.OrderID)
		return errors.Wrap(err, fmt.Sprintf("Failed to find shipment for order %s", webhook.OrderID))
	}

	// Map the webhook status to domain status
	newStatus := mapShiprocketStatus(webhook.CurrentStatus)
	if newStatus == shipment.Status {
		// No status change, nothing to do
		return nil
	}

	updates := map[string]interface{}{}
	if newStatus == domain.ShipmentStatusDelivered {
		now := time.Now()
		updates["delivered_at"] = now.Format(time.RFC3339)
	}
	if webhook.AWB != "" && shipment.AWBNumber == "" {
		updates["awb_number"] = webhook.AWB
	}

	// Update shipment status
	if err := s.shipmentRepo.UpdateStatus(ctx, shipment.OrderID, shipment.ID, newStatus, updates); err != nil {
		s.logger.WithError(err).Errorf("Failed to update shipment status via webhook for order %s", webhook.OrderID)
		return errors.Wrap(err, "Failed to update shipment status")
	}

	// Update order status for terminal statuses
	switch newStatus {
	case domain.ShipmentStatusDelivered:
		if err := s.orderRepo.UpdateStatus(ctx, shipment.OrderID, domain.OrderStatusDelivered, "system"); err != nil {
			s.logger.WithError(err).Errorf("Failed to update order to DELIVERED for order %s", shipment.OrderID)
		}
	case domain.ShipmentStatusRTO:
		if err := s.orderRepo.UpdateStatus(ctx, shipment.OrderID, domain.OrderStatusReturned, "system"); err != nil {
			s.logger.WithError(err).Errorf("Failed to update order to RETURNED for order %s", shipment.OrderID)
		}
	}

	return nil
}

// mapShiprocketStatus maps Shiprocket status strings to domain ShipmentStatus
func mapShiprocketStatus(status string) domain.ShipmentStatus {
	switch status {
	case "PICKED UP", "Picked Up":
		return domain.ShipmentStatusPickedUp
	case "IN TRANSIT", "In Transit":
		return domain.ShipmentStatusInTransit
	case "OUT FOR DELIVERY", "Out For Delivery":
		return domain.ShipmentStatusOutForDelivery
	case "DELIVERED", "Delivered":
		return domain.ShipmentStatusDelivered
	case "RTO", "RTO Initiated", "RTO Delivered":
		return domain.ShipmentStatusRTO
	default:
		return domain.ShipmentStatusCreated
	}
}

// Ensure interface compliance
var _ domain.ShippingService = (*ShippingService)(nil)
