package shiprocket

import "time"

// Config holds Shiprocket configuration
type Config struct {
	Email         string
	Password      string
	BaseURL       string
	PickupPincode string
}

// AuthResponse is the response from Shiprocket auth
type AuthResponse struct {
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

// CourierServiceabilityResponse is the response from courier serviceability check
type CourierServiceabilityResponse struct {
	Status int `json:"status"`
	Data   struct {
		AvailableCourierCompanies []CourierCompany `json:"available_courier_companies"`
	} `json:"data"`
}

// CourierCompany represents a courier option
type CourierCompany struct {
	CourierCompanyID int     `json:"courier_company_id"`
	CourierName      string  `json:"courier_name"`
	Rate             float64 `json:"rate"`
	ETD              string  `json:"etd"` // estimated transit days e.g. "3-5"
	EstimatedDays    int     `json:"estimated_delivery_days"`
}

// CreateOrderRequest is the request for creating a shipment order
type CreateOrderRequest struct {
	OrderID           string      `json:"order_id"`
	OrderDate         string      `json:"order_date"`
	PickupLocation    string      `json:"pickup_location"`
	BillingCustomer   string      `json:"billing_customer_name"`
	BillingLastName   string      `json:"billing_last_name"`
	BillingAddress    string      `json:"billing_address"`
	BillingCity       string      `json:"billing_city"`
	BillingPincode    string      `json:"billing_pincode"`
	BillingState      string      `json:"billing_state"`
	BillingCountry    string      `json:"billing_country"`
	BillingEmail      string      `json:"billing_email"`
	BillingPhone      string      `json:"billing_phone"`
	ShippingIsBilling bool        `json:"shipping_is_billing"`
	OrderItems        []OrderItem `json:"order_items"`
	PaymentMethod     string      `json:"payment_method"` // "Prepaid" or "COD"
	SubTotal          float64     `json:"sub_total"`
	Length            float64     `json:"length"`
	Breadth           float64     `json:"breadth"`
	Height            float64     `json:"height"`
	Weight            float64     `json:"weight"` // in kg
}

// OrderItem is an item in a Shiprocket order
type OrderItem struct {
	Name         string  `json:"name"`
	SKU          string  `json:"sku"`
	Units        int     `json:"units"`
	SellingPrice float64 `json:"selling_price"`
}

// CreateOrderResponse is the response from creating a shipment order
type CreateOrderResponse struct {
	OrderID    int    `json:"order_id"`
	ShipmentID int    `json:"shipment_id"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}

// AssignAWBResponse is the response from assigning an AWB
type AssignAWBResponse struct {
	Response struct {
		Data struct {
			AWBCode       string  `json:"awb_code"`
			CourierName   string  `json:"courier_name"`
			AppliedWeight float64 `json:"applied_weight"`
		} `json:"data"`
	} `json:"response"`
}

// GenerateLabelResponse is the response from generating a label
type GenerateLabelResponse struct {
	LabelURL string `json:"label_url"`
}

// TrackingResponse is the response from tracking
type TrackingResponse struct {
	TrackingData struct {
		TrackStatus    int    `json:"track_status"`
		ShipmentStatus int    `json:"shipment_status"`
		CurrentStatus  string `json:"current_status"`
	} `json:"tracking_data"`
}

// tokenCache holds a cached auth token with expiry
type tokenCache struct {
	token     string
	expiresAt time.Time
}
