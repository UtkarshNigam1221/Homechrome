// Package delhivery implements the courier.Gateway interface for Delhivery.
package delhivery

import "time"

// Config holds Delhivery client configuration.
type Config struct {
	APIToken       string
	BaseURL        string
	ClientName     string
	WebhookToken   string
	PickupLocation string
}

// --- Pincode serviceability ---

type pincodeResponse struct {
	DeliveryCodes []struct {
		PostalCode struct {
			Pincode      string `json:"pin"`
			District     string `json:"district"`
			StateCode    string `json:"state_code"`
			COD          string `json:"cod"`      // "Y" | "N"
			Prepaid      string `json:"pre_paid"` // "Y" | "N"
			MaxAmount    int    `json:"max_amount"`
			DeliveryDays int    `json:"delivery_days"`
			Zone         string `json:"remarks"` // zone returned in remarks (Delhivery convention)
		} `json:"postal_code"`
	} `json:"delivery_codes"`
}

// --- Create shipment ---

type createShipmentBody struct {
	Format    string              `json:"format"`
	Data      string              `json:"data"`
	Pickup    string              `json:"pickup_location"`
	Shipments []createShipmentRow `json:"shipments"`
}

type createShipmentRow struct {
	Name           string  `json:"name"`
	AddOne         string  `json:"add"`
	AddTwo         string  `json:"add2"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Country        string  `json:"country"`
	Phone          string  `json:"phone"`
	OrderID        string  `json:"order"`
	PaymentMode    string  `json:"payment_mode"` // "Prepaid" | "COD"
	CODAmount      float64 `json:"cod_amount"`
	TotalAmount    float64 `json:"total_amount"`
	Pin            string  `json:"pin"`
	Weight         float64 `json:"weight"` // grams
	ShipmentWidth  float64 `json:"shipment_width"`
	ShipmentHeight float64 `json:"shipment_height"`
	ShipmentLength float64 `json:"shipment_length"`
	HSNCode        string  `json:"hsn_code,omitempty"`
	SellerName     string  `json:"seller_name,omitempty"`
}

type createShipmentResponse struct {
	Packages []struct {
		Waybill string   `json:"waybill"`
		Refnum  string   `json:"refnum"`
		Status  string   `json:"status"`
		Remarks []string `json:"remarks"`
	} `json:"packages"`
	UploadWBN string `json:"upload_wbn"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// --- Manifest + Pickup ---

type manifestRequest struct {
	Waybills       []string `json:"waybills"`
	PickupDate     string   `json:"pickup_date"`
	PickupTime     string   `json:"pickup_time"`
	PickupLocation string   `json:"pickup_location"`
}

type manifestResponse struct {
	ManifestID string `json:"manifest_id"`
	PDFURL     string `json:"pdf_url"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

type pickupRequest struct {
	PickupTime       string `json:"pickup_time"`
	PickupDate       string `json:"pickup_date"`
	PickupLocation   string `json:"pickup_location"`
	ExpectedPackages int    `json:"expected_package_count"`
}

type pickupResponse struct {
	PickupID string `json:"pickup_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// --- Tracking ---

type trackingResponse struct {
	ShipmentData []struct {
		Shipment struct {
			AWB    string `json:"AWB"`
			Status struct {
				Status         string `json:"Status"`
				StatusLocation string `json:"StatusLocation"`
				StatusDateTime string `json:"StatusDateTime"`
				StatusType     string `json:"StatusType"`
				Instructions   string `json:"Instructions"`
			} `json:"Status"`
			Scans []struct {
				ScanDetail struct {
					Scan         string `json:"Scan"`
					ScanLocation string `json:"ScanLocation"`
					ScanDateTime string `json:"ScanDateTime"`
					Instructions string `json:"Instructions"`
				} `json:"ScanDetail"`
			} `json:"Scans"`
		} `json:"Shipment"`
	} `json:"ShipmentData"`
}

// --- NDR action ---

type ndrActionRequest struct {
	Waybill string `json:"waybill"`
	Act     string `json:"act"` // RE-ATTEMPT | RTO | DEFER
}

// --- COD remittance ---

type codRemittanceResponse struct {
	Data []struct {
		AWB        string  `json:"awb"`
		OrderID    string  `json:"order_id"`
		AmountPaid float64 `json:"amount_paid"` // rupees
		UTR        string  `json:"utr"`
		RemittedAt string  `json:"paid_date"` // RFC3339
	} `json:"data"`
}

// --- Rate calc ---

type rateResponse struct {
	TotalAmount float64 `json:"total_amount"` // rupees
	GrossAmount float64 `json:"gross_amount"`
	Zone        string  `json:"zone"`
}

// --- Webhook ---

type webhookPayload struct {
	Waybill   string `json:"waybill"`
	Status    string `json:"status"`
	Location  string `json:"location"`
	Timestamp string `json:"timestamp"`
	NDRReason string `json:"ndr_reason,omitempty"`
	IsReverse bool   `json:"is_reverse,omitempty"`
}

// parseDelhiveryTime parses Delhivery timestamp formats. Returns zero time on failure.
func parseDelhiveryTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
