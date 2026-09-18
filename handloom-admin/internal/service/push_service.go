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
	repo    domain.PushSubscriptionRepository
	gateway webpush.Gateway
}

// NewPushService creates a new PushService
func NewPushService(repo domain.PushSubscriptionRepository, gateway webpush.Gateway) *PushService {
	return &PushService{repo: repo, gateway: gateway}
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
	subs, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Field-for-field identical, so a conversion keeps them in lockstep: if
	// either type gains a field, this stops compiling instead of silently
	// dropping it from the payload.
	payload := domain.PushPayload(req)

	successCount := s.fanOut(ctx, subs, payload)
	failureCount := len(subs) - successCount

	broadcast := &domain.PushBroadcast{
		ID:            "bcast_" + uuid.New().String()[:8],
		Title:         req.Title,
		Body:          req.Body,
		URL:           payloadURL(req.URL),
		Tag:           req.Tag,
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
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" {
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
