package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
)

// broadcastConcurrency bounds simultaneous in-flight pushes. Every send is one
// HTTPS round trip to a push service, so an unbounded fan-out over a large
// subscriber list would open thousands of sockets at once and trip the
// Lambda's file-descriptor ceiling long before the push services rate-limit us.
const broadcastConcurrency = 32

// defaultBroadcastHistory is how many past broadcasts the admin console shows.
const defaultBroadcastHistory int32 = 20

// defaultPushIcon is the storefront asset shown when a payload names no icon.
const defaultPushIcon = "/icon.png"

// maxPushEndpointLen caps the stored endpoint. Real ones run to ~500 chars.
const maxPushEndpointLen = 512

// pushEndpointHosts are the push services we accept subscriptions for, matched
// on the host or any subdomain of it.
var pushEndpointHosts = []string{
	"fcm.googleapis.com",        // Chrome, Edge, Brave, Opera
	"android.googleapis.com",    // legacy GCM endpoints still issued by old Chrome
	"push.services.mozilla.com", // Firefox
	"notify.windows.com",        // Edge / WNS
	"web.push.apple.com",        // Safari
}

// PushService implements Web Push subscription and delivery operations.
type PushService struct {
	repo           domain.PushSubscriptionRepository
	gateway        webpush.Gateway
	assetFinalizer domain.AssetFinalizer
}

// NewPushService creates a new PushService
func NewPushService(
	repo domain.PushSubscriptionRepository,
	gateway webpush.Gateway,
	assetFinalizer domain.AssetFinalizer,
) *PushService {
	return &PushService{repo: repo, gateway: gateway, assetFinalizer: assetFinalizer}
}

// PublicKey returns the VAPID application server key for the storefront.
// Empty means push is not configured; the storefront then hides its opt-in UI.
func (s *PushService) PublicKey() string {
	return s.gateway.PublicKey()
}

// Subscribe registers (or refreshes) a browser's push endpoint.
func (s *PushService) Subscribe(
	ctx context.Context,
	req domain.SubscribePushRequest,
	userAgent string,
) (*domain.PushSubscription, error) {
	if err := validatePushEndpoint(req.Endpoint); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sub := &domain.PushSubscription{
		Endpoint:   req.Endpoint,
		Keys:       req.Keys,
		UserAgent:  userAgent,
		Device:     req.Device,
		CustomerID: middleware.GetCustomerIDFromContext(ctx),
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  now,
		LastSeenAt: now,
	}

	isNew, err := s.repo.Save(ctx, sub)
	if err != nil {
		return nil, err
	}

	if isNew {
		// Best-effort welcome push: a failure here must not fail the
		// subscription that already succeeded.
		s.sendOne(ctx, sub, domain.PushPayload{
			Title: "Welcome to Homechrome",
			Body:  "First look at new weaver collections and private offers.",
			URL:   "/",
			Tag:   "homechrome-welcome",
		})
	}

	slog.InfoContext(ctx, "Saved push subscription", "subscription_id", sub.ID, "is_new", isNew)
	return sub, nil
}

// LinkCustomer attaches a device that opted in before sign-in. Most shoppers
// grant permission first and sign in later, so without this their devices stay
// anonymous and never receive an order update.
func (s *PushService) LinkCustomer(ctx context.Context, endpoint string) error {
	customerID := middleware.GetCustomerIDFromContext(ctx)
	if customerID == "" {
		return errors.Unauthorized("Sign in to link this device")
	}
	if err := validatePushEndpoint(endpoint); err != nil {
		return err
	}
	return s.repo.LinkCustomer(ctx, endpoint, customerID)
}

// Unsubscribe retires an endpoint.
func (s *PushService) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.Deactivate(ctx, endpoint)
}

// SendTest delivers a notification to one endpoint the caller is already
// subscribed to. Scoped to a single endpoint on purpose — the storefront's
// "send test" must never be able to reach another visitor's device.
func (s *PushService) SendTest(ctx context.Context, endpoint string) error {
	sub, err := s.repo.GetByEndpoint(ctx, endpoint)
	if err != nil {
		return err
	}
	if sub.Status != domain.PushSubscriptionActive {
		return errors.Validation("This device is not subscribed to notifications")
	}

	if err := s.deliver(ctx, sub, domain.PushPayload{
		Title: "Notifications are working",
		Body:  "This is exactly how a Homechrome alert will look.",
		URL:   "/",
		Tag:   "homechrome-test",
	}); err != nil {
		return errors.Wrap(err, "Failed to deliver test notification")
	}
	return nil
}

// Broadcast fans a notification out to every active subscriber and records the
// outcome. Endpoints the push services report as gone are deactivated as we go,
// so the next broadcast does not retry them.
func (s *PushService) Broadcast(
	ctx context.Context,
	req domain.BroadcastPushRequest,
	sentBy string,
) (*domain.BroadcastPushResponse, error) {
	if s.assetFinalizer == nil {
		return nil, errors.Internal("This service cannot broadcast")
	}

	subs, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Field-for-field identical, so a conversion keeps them in lockstep: if
	// either type gains a field, this stops compiling instead of silently
	// dropping it from the payload.
	payload := domain.PushPayload(req)

	// tmp/ is deleted after a day, so an unfinalized banner would break in
	// history and on any notification still sitting on a device.
	if payload.Image, err = s.assetFinalizer.FinalizeIfTemp(ctx, payload.Image); err != nil {
		return nil, errors.Wrap(err, "Failed to store the broadcast image")
	}
	if payload.Icon, err = s.assetFinalizer.FinalizeIfTemp(ctx, payload.Icon); err != nil {
		return nil, errors.Wrap(err, "Failed to store the broadcast icon")
	}

	broadcastID := "bcast_" + uuid.New().String()

	// Chrome silently drops everything past the second action; trim rather than
	// let a sender believe a third button shipped.
	if len(payload.Actions) > domain.MaxPushActions {
		payload.Actions = payload.Actions[:domain.MaxPushActions]
	}

	// A shared tag makes each notification replace the last one on the device,
	// so an unread broadcast disappears when the next goes out. Default to one
	// tag per broadcast; an explicit tag opts back in to replacing.
	if payload.Tag == "" {
		payload.Tag = broadcastID
	}

	successCount := s.fanOut(ctx, subs, payload)
	failureCount := len(subs) - successCount

	broadcast := &domain.PushBroadcast{
		ID:            broadcastID,
		Title:         req.Title,
		Body:          req.Body,
		URL:           payloadURL(req.URL),
		Tag:           req.Tag,
		Image:         payload.Image,
		Icon:          payload.Icon,
		Actions:       req.Actions,
		TotalTargeted: len(subs),
		SuccessCount:  successCount,
		FailureCount:  failureCount,
		Status:        broadcastStatus(len(subs), successCount, failureCount),
		SentAt:        time.Now().UTC(),
		SentBy:        sentBy,
	}

	if err := s.repo.SaveBroadcast(ctx, broadcast); err != nil {
		// The pushes already went out — losing the audit row must not report
		// the broadcast as failed, so log and return the real delivery counts.
		slog.ErrorContext(ctx, "Failed to record push broadcast", "error", err, "broadcast_id", broadcast.ID)
	}

	slog.InfoContext(ctx, "Push broadcast complete",
		"broadcast_id", broadcast.ID,
		"targeted", broadcast.TotalTargeted,
		"success", successCount,
		"failure", failureCount)

	return &domain.BroadcastPushResponse{
		TotalTargeted: broadcast.TotalTargeted,
		SuccessCount:  successCount,
		FailureCount:  failureCount,
		Status:        broadcast.Status,
		Broadcast:     broadcast,
	}, nil
}

// NotifyCustomer notifies every device a customer opted in on. It reuses the
// broadcast fan-out, so dead endpoints are pruned and rejections logged.
func (s *PushService) NotifyCustomer(
	ctx context.Context, customerID string, payload domain.PushPayload,
) (int, error) {
	if customerID == "" {
		return 0, errors.BadRequest("A customer is required to notify")
	}

	subs, err := s.repo.ListByCustomer(ctx, customerID)
	if err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}

	delivered := s.fanOut(ctx, subs, payload)
	slog.InfoContext(ctx, "Notified a customer",
		"customer_id", customerID, "devices", len(subs), "delivered", delivered)
	return delivered, nil
}

// fanOut delivers payload to every subscription, bounded by broadcastConcurrency,
// and returns how many succeeded.
func (s *PushService) fanOut(ctx context.Context, subs []*domain.PushSubscription, payload domain.PushPayload) int {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		sem       = make(chan struct{}, broadcastConcurrency)
	)

	for _, sub := range subs {
		wg.Add(1)
		go func(sub *domain.PushSubscription) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := s.deliver(ctx, sub, payload); err != nil {
				// Without this a PARTIAL broadcast is a bare count with no way
				// to learn which endpoint rejected it, or why.
				slog.WarnContext(ctx, "Push broadcast delivery failed",
					"error", err, "subscription_id", sub.ID, "endpoint", sub.Endpoint)
				return
			}
			mu.Lock()
			succeeded++
			mu.Unlock()
		}(sub)
	}

	wg.Wait()
	return succeeded
}

// deliver encodes and sends one push, deactivating the subscription when the
// push service reports the endpoint is permanently gone.
func (s *PushService) deliver(ctx context.Context, sub *domain.PushSubscription, payload domain.PushPayload) error {
	if payload.URL == "" {
		payload.URL = "/"
	}
	if payload.Icon == "" {
		payload.Icon = defaultPushIcon
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Internal("Failed to encode push payload")
	}

	err = s.gateway.Send(ctx, webpush.Subscription{
		Endpoint: sub.Endpoint,
		P256dh:   sub.Keys.P256dh,
		Auth:     sub.Keys.Auth,
	}, body)
	if err == nil {
		return nil
	}

	if stderrors.Is(err, webpush.ErrSubscriptionGone) {
		if deactivateErr := s.repo.Deactivate(ctx, sub.Endpoint); deactivateErr != nil {
			slog.WarnContext(ctx, "Failed to deactivate dead push subscription",
				"error", deactivateErr, "subscription_id", sub.ID)
		}
	}
	return err
}

// sendOne delivers a push and swallows the error, for deliveries whose failure
// must not fail the caller (the welcome push).
func (s *PushService) sendOne(ctx context.Context, sub *domain.PushSubscription, payload domain.PushPayload) {
	if err := s.deliver(ctx, sub, payload); err != nil {
		slog.WarnContext(ctx, "Best-effort push delivery failed",
			"error", err, "subscription_id", sub.ID)
	}
}

// ListSubscriptions returns a page of subscriptions for the admin console.
func (s *PushService) ListSubscriptions(
	ctx context.Context,
	status domain.PushSubscriptionStatus,
	pagination domain.PaginationRequest,
) (*domain.ListPushSubscriptionsResponse, error) {
	if status == "" {
		status = domain.PushSubscriptionActive
	}
	return s.repo.List(ctx, status, pagination)
}

// ListBroadcasts returns recent broadcast history for the admin console.
func (s *PushService) ListBroadcasts(ctx context.Context, limit int32) ([]*domain.PushBroadcast, error) {
	if limit <= 0 {
		limit = defaultBroadcastHistory
	}
	return s.repo.ListBroadcasts(ctx, limit)
}

// validatePushEndpoint keeps the public subscribe route from pointing the
// backend's VAPID-signed sends at an arbitrary host of the caller's choosing.
func validatePushEndpoint(endpoint string) error {
	// Userinfo and port are rejected, not ignored: Go drops userinfo when
	// dialing, so a@host and b@host reach one device under two row keys.
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" ||
		len(endpoint) > maxPushEndpointLen {
		return errors.Validation("Unsupported push endpoint")
	}

	host := strings.ToLower(u.Hostname())
	for _, allowed := range pushEndpointHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return errors.Validation("Unsupported push endpoint")
}

// payloadURL normalizes an empty click-through target to the storefront root.
func payloadURL(url string) string {
	if url == "" {
		return "/"
	}
	return url
}

// broadcastStatus classifies a fan-out result. A broadcast with no subscribers
// is a success: there was nothing to fail.
func broadcastStatus(targeted, success, failure int) domain.PushBroadcastStatus {
	switch {
	case targeted == 0 || failure == 0:
		return domain.PushBroadcastSuccess
	case success > 0:
		return domain.PushBroadcastPartial
	default:
		return domain.PushBroadcastFailed
	}
}
