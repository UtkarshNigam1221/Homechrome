package webpush

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	wp "github.com/SherClockHolmes/webpush-go"

	metricsmw "github.com/handloom/admin/pkg/metrics/middleware"
)

// ErrSubscriptionGone reports that the push service has permanently retired the
// endpoint (404/410). Callers should stop sending to it rather than retry.
var ErrSubscriptionGone = errors.New("push subscription is gone")

// defaultTTL is how long a push service holds an undelivered message. A day is
// long enough for a laptop to be reopened, short enough that a "new arrivals"
// alert is not stale when it lands.
const defaultTTL = 24 * 60 * 60

// maxErrorBody caps how much of a rejection we quote. Push services answer with
// a short reason ({"reason":"BadJwtToken"}); anything longer is not for us.
const maxErrorBody = 256

// Client sends Web Push messages using VAPID-signed requests.
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a Web Push client using the supplied VAPID credentials.
func NewClient(config Config) *Client {
	if config.TTLSeconds <= 0 {
		config.TTLSeconds = defaultTTL
	}
	httpClient := metricsmw.NewInstrumentedClient(10*time.Second, "webpush")
	// A signed push is for one endpoint; never replay it at a redirect target.
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{config: config, httpClient: httpClient}
}

// PublicKey returns the VAPID application server key.
func (c *Client) PublicKey() string {
	return c.config.PublicKey
}

// Send encrypts payload for sub and posts it to the browser's push service.
func (c *Client) Send(ctx context.Context, sub Subscription, payload []byte) error {
	resp, err := wp.SendNotificationWithContext(ctx, payload, &wp.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     wp.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &wp.Options{
		HTTPClient:      c.httpClient,
		Subscriber:      c.config.Subject,
		VAPIDPublicKey:  c.config.PublicKey,
		VAPIDPrivateKey: c.config.PrivateKey,
		TTL:             c.config.TTLSeconds,
		Urgency:         wp.UrgencyNormal,
	})
	if err != nil {
		return fmt.Errorf("web push request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// 404/410 are the push services' way of saying the endpoint is retired —
	// the browser cleared site data, or the subscription was replaced.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return ErrSubscriptionGone
	}
	if resp.StatusCode >= 300 {
		// The status alone cannot separate a malformed VAPID subject from a key
		// the endpoint was not subscribed with — both are 403. The body names it.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		reason := strings.TrimSpace(string(body))
		if reason == "" {
			return fmt.Errorf("web push rejected with status %d", resp.StatusCode)
		}
		return fmt.Errorf("web push rejected with status %d: %s", resp.StatusCode, reason)
	}
	return nil
}

// DevClient logs pushes to stdout instead of delivering them. Selected when
// VAPID keys are not configured, so local development works without them.
type DevClient struct{}

// NewDevClient creates a Web Push client that prints payloads to the console.
func NewDevClient() *DevClient {
	return &DevClient{}
}

// PublicKey returns an empty key — a dev client cannot be subscribed to, and
// the storefront hides the opt-in UI when the key is empty.
func (d *DevClient) PublicKey() string { return "" }

// Send logs the payload instead of delivering it.
func (d *DevClient) Send(ctx context.Context, sub Subscription, payload []byte) error {
	slog.InfoContext(ctx, "DEV web push (not delivered)",
		"endpoint", sub.Endpoint, "payload", string(payload))
	return nil
}

// Ensure interface compliance
var (
	_ Gateway = (*Client)(nil)
	_ Gateway = (*DevClient)(nil)
)
