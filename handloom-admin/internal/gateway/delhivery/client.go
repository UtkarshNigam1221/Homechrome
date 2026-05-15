package delhivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// Client is the production Delhivery courier client.
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient constructs a production Delhivery Client.
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://track.delhivery.com"
	}
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Compile-time interface assertion — verifies all 12 Gateway methods are implemented.
var _ courier.Gateway = (*Client)(nil)

// doRequest performs a JSON HTTP request to the Delhivery API.
// If out is non-nil, the response body is JSON-unmarshalled into it.
// The raw response body is returned alongside for callers that need it.
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any, out any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	fullURL := c.config.BaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.config.APIToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("delhivery %s %s: status %d: %s", method, path, resp.StatusCode, string(raw))
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(raw))
		}
	}
	return raw, nil
}

// CheckPincode looks up serviceability for a single pincode.
func (c *Client) CheckPincode(ctx context.Context, pincode string) (*courier.PincodeInfo, error) {
	var resp pincodeResponse
	q := url.Values{"filter_codes": {pincode}}
	if _, err := c.doRequest(ctx, http.MethodGet, "/c/api/pin-codes/json/", q, nil, &resp); err != nil {
		return nil, err
	}
	if len(resp.DeliveryCodes) == 0 {
		return &courier.PincodeInfo{Pincode: pincode, Serviceable: false}, nil
	}
	pc := resp.DeliveryCodes[0].PostalCode
	return &courier.PincodeInfo{
		Pincode:          pc.Pincode,
		Serviceable:      true,
		Zone:             pc.Zone,
		City:             pc.District,
		State:            pc.StateCode,
		CODAvailable:     pc.COD == "Y",
		PrepaidAvailable: pc.Prepaid == "Y",
		EstimatedDays:    pc.DeliveryDays,
	}, nil
}

// FetchRateMatrix iterates over zones × weight slabs and queries Delhivery's rate API for each.
func (c *Client) FetchRateMatrix(ctx context.Context) ([]courier.RateRow, error) {
	zones := []string{"A", "B", "C", "D", "E"}
	slabs := []int{500, 1000, 2000, 5000, 10000}
	rows := make([]courier.RateRow, 0, len(zones)*len(slabs))
	for _, z := range zones {
		for _, slab := range slabs {
			prepaid, err := c.fetchRate(ctx, z, slab, "Pre-paid")
			if err != nil {
				return rows, fmt.Errorf("zone=%s slab=%d prepaid: %w", z, slab, err)
			}
			cod, err := c.fetchRate(ctx, z, slab, "COD")
			if err != nil {
				return rows, fmt.Errorf("zone=%s slab=%d cod: %w", z, slab, err)
			}
			rto, err := c.fetchRate(ctx, z, slab, "RTO")
			if err != nil {
				return rows, fmt.Errorf("zone=%s slab=%d rto: %w", z, slab, err)
			}
			rows = append(rows, courier.RateRow{
				Zone:            z,
				WeightSlabGrams: slab,
				PrepaidPaise:    prepaid,
				CODPaise:        cod,
				RTOPaise:        rto,
			})
		}
	}
	return rows, nil
}

func (c *Client) fetchRate(ctx context.Context, zone string, weightG int, mode string) (int64, error) {
	q := url.Values{
		"md":    {"E"},
		"ss":    {"Delivered"},
		"d_pin": {"110001"},
		"o_pin": {"560001"},
		"cgm":   {fmt.Sprintf("%d", weightG)},
		"pt":    {mode},
		"cl":    {c.config.ClientName},
	}
	var resp rateResponse
	if _, err := c.doRequest(ctx, http.MethodGet, "/api/kinko/v1/invoice/charges/.json", q, nil, &resp); err != nil {
		return 0, err
	}
	return int64(resp.TotalAmount * 100), nil
}

// buildShipmentRow constructs a Delhivery shipment row with given payment mode + dimensions.
// Used by both forward and reverse shipment creation.
func (c *Client) buildShipmentRow(addr courier.Address, orderRef string, totalPaise, codAmountPaise int64, paymentLabel string, weightG, lengthCm, breadthCm, heightCm float64) createShipmentRow {
	row := createShipmentRow{
		Name:           strings.TrimSpace(addr.FirstName + " " + addr.LastName),
		AddOne:         addr.AddressLine1,
		AddTwo:         addr.AddressLine2,
		City:           addr.City,
		State:          addr.State,
		Country:        addr.Country,
		Phone:          addr.Phone,
		OrderID:        orderRef,
		PaymentMode:    paymentLabel,
		TotalAmount:    float64(totalPaise) / 100.0,
		Pin:            addr.Pincode,
		Weight:         weightG,
		ShipmentWidth:  breadthCm,
		ShipmentHeight: heightCm,
		ShipmentLength: lengthCm,
		SellerName:     c.config.ClientName,
	}
	if paymentLabel == "COD" {
		row.CODAmount = float64(codAmountPaise) / 100.0
	}
	return row
}

// postCreateShipment POSTs a shipment body to /api/cmu/create.json and returns
// the first package row + raw bytes after validating success.
func (c *Client) postCreateShipment(ctx context.Context, pickupLocation string, row createShipmentRow) (waybill string, uploadWBN string, raw []byte, err error) {
	body := createShipmentBody{
		Format:    "json",
		Pickup:    pickupLocation,
		Shipments: []createShipmentRow{row},
	}
	var resp createShipmentResponse
	raw, err = c.doRequest(ctx, http.MethodPost, "/api/cmu/create.json", nil, body, &resp)
	if err != nil {
		return "", "", nil, err
	}
	if !resp.Success || len(resp.Packages) == 0 {
		return "", "", raw, fmt.Errorf("delhivery createShipment: %s", resp.Error)
	}
	pkg := resp.Packages[0]
	if pkg.Status != "Success" {
		return "", "", raw, fmt.Errorf("delhivery createShipment: package status %s: %v", pkg.Status, pkg.Remarks)
	}
	return pkg.Waybill, resp.UploadWBN, raw, nil
}

// CreateShipment registers a forward shipment with Delhivery and returns the assigned AWB.
func (c *Client) CreateShipment(ctx context.Context, req *courier.CreateShipmentRequest) (*courier.CreateShipmentResult, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("createShipment: items required")
	}
	totalAmount := int64(0)
	for _, it := range req.Items {
		totalAmount += it.UnitPaise * int64(it.Quantity)
	}
	paymentLabel := "Prepaid"
	if req.PaymentMode == courier.PaymentCOD {
		paymentLabel = "COD"
	}
	row := c.buildShipmentRow(req.Customer, req.OrderID, totalAmount, req.CODAmountPaise, paymentLabel,
		float64(req.WeightGrams), float64(req.LengthCm), float64(req.BreadthCm), float64(req.HeightCm))
	waybill, uploadWBN, raw, err := c.postCreateShipment(ctx, req.PickupLocation, row)
	if err != nil {
		return nil, err
	}
	return &courier.CreateShipmentResult{
		AWB:               waybill,
		CarrierShipmentID: waybill,
		UploadWBN:         uploadWBN,
		EstimatedDays:     0,
		RawResponse:       raw,
	}, nil
}

type labelResponse struct {
	PackagesFound []struct {
		PDFDownloadLink string `json:"pdf_download_link"`
	} `json:"packages_found"`
}

// GenerateLabel returns a URL to the shipping label PDF for a given AWB.
func (c *Client) GenerateLabel(ctx context.Context, awb string) (string, error) {
	q := url.Values{"wbns": {awb}, "pdf": {"true"}}
	var resp labelResponse
	if _, err := c.doRequest(ctx, http.MethodGet, "/api/p/packing_slip", q, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.PackagesFound) == 0 {
		return "", fmt.Errorf("delhivery generateLabel: no label for %s", awb)
	}
	return resp.PackagesFound[0].PDFDownloadLink, nil
}

// CreateManifest generates a Delhivery manifest covering the given AWBs.
func (c *Client) CreateManifest(ctx context.Context, awbs []string, pickupDate time.Time) (*courier.ManifestResult, error) {
	if len(awbs) == 0 {
		return nil, fmt.Errorf("createManifest: at least one waybill required")
	}
	body := manifestRequest{
		Waybills:       awbs,
		PickupDate:     pickupDate.Format("2006-01-02"),
		PickupTime:     pickupDate.Format("15:04"),
		PickupLocation: c.config.PickupLocation,
	}
	var resp manifestResponse
	if _, err := c.doRequest(ctx, http.MethodPost, "/api/p/manifest", nil, body, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("delhivery createManifest: %s", resp.Error)
	}
	return &courier.ManifestResult{
		ManifestID: resp.ManifestID,
		PDFURL:     resp.PDFURL,
		AWBCount:   len(awbs),
	}, nil
}

// SchedulePickup creates a pickup request bound to a manifest.
func (c *Client) SchedulePickup(ctx context.Context, manifestID, pickupLocation string, pickupDate time.Time) error {
	if pickupLocation == "" {
		pickupLocation = c.config.PickupLocation
	}
	body := pickupRequest{
		PickupTime:       pickupDate.Format("15:04"),
		PickupDate:       pickupDate.Format("2006-01-02"),
		PickupLocation:   pickupLocation,
		ExpectedPackages: 0,
	}
	var resp pickupResponse
	if _, err := c.doRequest(ctx, http.MethodPost, "/fm/request/new/", nil, body, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("delhivery schedulePickup: %s", resp.Error)
	}
	return nil
}

// TrackByAWB fetches live tracking for a single AWB.
func (c *Client) TrackByAWB(ctx context.Context, awb string) (*courier.TrackingInfo, error) {
	q := url.Values{"waybill": {awb}}
	var resp trackingResponse
	if _, err := c.doRequest(ctx, http.MethodGet, "/api/v1/packages/json/", q, nil, &resp); err != nil {
		return nil, err
	}
	if len(resp.ShipmentData) == 0 {
		return nil, fmt.Errorf("delhivery trackByAWB: no data for %s", awb)
	}
	sd := resp.ShipmentData[0].Shipment
	currentStatus := MapStatus(sd.Status.Status)
	currentTime := parseDelhiveryTime(sd.Status.StatusDateTime)
	history := make([]courier.TrackingScan, 0, len(sd.Scans))
	for _, s := range sd.Scans {
		t := parseDelhiveryTime(s.ScanDetail.ScanDateTime)
		history = append(history, courier.TrackingScan{
			Status:      MapStatus(s.ScanDetail.Scan),
			Location:    s.ScanDetail.ScanLocation,
			Time:        t,
			Description: s.ScanDetail.Instructions,
		})
	}
	return &courier.TrackingInfo{
		AWB:             sd.AWB,
		Status:          currentStatus,
		CurrentLocation: sd.Status.StatusLocation,
		LastUpdate:      currentTime,
		History:         history,
	}, nil
}

// ReAttemptDelivery sends an NDR action (re-attempt, RTO, defer) to Delhivery.
func (c *Client) ReAttemptDelivery(ctx context.Context, awb string, action courier.NDRAction) error {
	act := "RE-ATTEMPT"
	switch action {
	case courier.NDRActionRTO:
		act = "RTO"
	case courier.NDRActionDefer:
		act = "DEFER"
	}
	body := ndrActionRequest{Waybill: awb, Act: act}
	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	if _, err := c.doRequest(ctx, http.MethodPost, "/api/p/update", nil, body, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("delhivery reAttempt: %s", resp.Error)
	}
	return nil
}

// CreateReversePickup creates a reverse-direction Delhivery shipment from customer to warehouse.
func (c *Client) CreateReversePickup(ctx context.Context, req *courier.ReversePickupRequest) (*courier.ReversePickupResult, error) {
	totalAmount := int64(0)
	for _, it := range req.Items {
		totalAmount += it.UnitPaise * int64(it.Quantity)
	}
	row := c.buildShipmentRow(req.Customer, "REV-"+req.OriginalOrderID, totalAmount, 0, "Pickup",
		500, 30, 25, 5)
	waybill, _, _, err := c.postCreateShipment(ctx, req.PickupLocation, row)
	if err != nil {
		return nil, err
	}
	return &courier.ReversePickupResult{
		ReverseAWB:        waybill,
		CarrierShipmentID: waybill,
		EstimatedDays:     0,
	}, nil
}

// FetchCODRemittances pulls COD payout entries for the given date range.
func (c *Client) FetchCODRemittances(ctx context.Context, from, to time.Time) ([]courier.RemittanceRow, error) {
	q := url.Values{
		"start_date": {from.Format("2006-01-02")},
		"end_date":   {to.Format("2006-01-02")},
	}
	var resp codRemittanceResponse
	if _, err := c.doRequest(ctx, http.MethodGet, "/api/cmu/get_invoice", q, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]courier.RemittanceRow, 0, len(resp.Data))
	for _, d := range resp.Data {
		t := parseDelhiveryTime(d.RemittedAt)
		out = append(out, courier.RemittanceRow{
			AWB:         d.AWB,
			OrderRef:    d.OrderID,
			AmountPaise: int64(d.AmountPaid * 100),
			UTR:         d.UTR,
			RemittedAt:  t,
		})
	}
	return out, nil
}

// VerifyWebhookSignature checks the HMAC-SHA256 signature on a Delhivery webhook payload.
// Returns a typed AppError with ErrCodeUnauthorized so the HTTP layer responds 401
// instead of 500.
func (c *Client) VerifyWebhookSignature(headers http.Header, body []byte) error {
	sig := headers.Get("X-Delhivery-Signature")
	if sig == "" {
		return errors.New(errors.ErrCodeUnauthorized, "Delhivery webhook: missing signature header")
	}
	mac := hmac.New(sha256.New, []byte(c.config.WebhookToken))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return errors.New(errors.ErrCodeUnauthorized, "Delhivery webhook: signature mismatch")
	}
	return nil
}

// ParseWebhook decodes a Delhivery webhook payload into a canonical WebhookEvent.
func (c *Client) ParseWebhook(body []byte) (*courier.WebhookEvent, error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("delhivery parseWebhook: %w", err)
	}
	t := parseDelhiveryTime(p.Timestamp)
	return &courier.WebhookEvent{
		AWB:       p.Waybill,
		Status:    MapStatus(p.Status),
		Location:  p.Location,
		Timestamp: t,
		NDRReason: p.NDRReason,
		IsReverse: p.IsReverse,
	}, nil
}
