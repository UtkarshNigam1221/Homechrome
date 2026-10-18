package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
)

// broadcastConcurrency bounds in-flight pushes: each is an HTTPS round trip,
// and an unbounded fan-out would hit the Lambda's file-descriptor ceiling.
const broadcastConcurrency = 32

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

	owner, err := s.subscribeOwner(ctx, req.Endpoint)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sub := &domain.PushSubscription{
		Endpoint:   req.Endpoint,
		Keys:       req.Keys,
		UserAgent:  userAgent,
		Device:     req.Device,
		CustomerID: owner,
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  now,
		LastSeenAt: now,
	}

	isNew, err := s.repo.Save(ctx, sub)
	if err != nil {
		return nil, err
	}

	if isNew {
		// Best-effort: a failed welcome must not fail the subscription that succeeded.
		welcome := domain.PushPayload{
			Title: "Welcome to Homechrome",
			Body:  "First look at new weaver collections and private offers.",
			URL:   "/",
			Tag:   "homechrome-welcome",
		}
		if err := s.deliver(ctx, sub, welcome); err != nil {
			slog.WarnContext(ctx, "Welcome push failed", "error", err, "subscription_id", sub.ID)
		}
	}

	slog.InfoContext(ctx, "Saved push subscription", "subscription_id", sub.ID, "is_new", isNew)
	return sub, nil
}

// subscribeOwner decides who a re-subscribe leaves the device belonging to. No
// identity may only mean a lapsed token, so /unlink alone gives a device up.
func (s *PushService) subscribeOwner(ctx context.Context, endpoint string) (string, error) {
	if customerID := middleware.GetCustomerIDFromContext(ctx); customerID != "" {
		return customerID, nil
	}

	existing, err := s.repo.GetByEndpoint(ctx, endpoint)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return existing.CustomerID, nil
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

// UnlinkCustomer detaches a device from whoever owned it, on sign-out. Without
// it a shared tablet keeps showing one shopper's order numbers to the next.
func (s *PushService) UnlinkCustomer(ctx context.Context, endpoint string) error {
	if middleware.GetCustomerIDFromContext(ctx) == "" {
		return errors.Unauthorized("Sign in to unlink this device")
	}
	if err := validatePushEndpoint(endpoint); err != nil {
		return err
	}
	return s.repo.UnlinkCustomer(ctx, endpoint)
}

// Unsubscribe retires an endpoint.
func (s *PushService) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.Deactivate(ctx, endpoint)
}

// SendTest delivers a notification to the one endpoint the caller owns,
// so the storefront's "send test" can never reach another visitor's device.
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

// Broadcast fans a notification out to every active subscriber and records
// the outcome, deactivating endpoints the push services report as gone.
func (s *PushService) Broadcast(
	ctx context.Context,
	req domain.BroadcastPushRequest,
	sentBy string,
) (*domain.PushBroadcast, error) {
	if s.assetFinalizer == nil {
		return nil, errors.Internal("This service cannot broadcast")
	}

	subs, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Field-for-field identical, so the conversion stops compiling if either
	// type gains a field instead of silently dropping it from the payload.
	payload := domain.PushPayload(req)
	if payload.URL == "" {
		payload.URL = "/"
	}

	// tmp/ is deleted after a day, so an unfinalized banner would break in
	// history and on any notification still sitting on a device.
	if payload.Image, err = s.assetFinalizer.FinalizeIfTemp(ctx, payload.Image); err != nil {
		return nil, errors.Wrap(err, "Failed to store the broadcast image").WithDetails(map[string]string{"field": "image"})
	}
	if payload.Icon, err = s.assetFinalizer.FinalizeIfTemp(ctx, payload.Icon); err != nil {
		return nil, errors.Wrap(err, "Failed to store the broadcast icon").WithDetails(map[string]string{"field": "icon"})
	}

	broadcastID := "bcast_" + uuid.New().String()

	// Chrome silently drops everything past the second action; trim rather than
	// let a sender believe a third button shipped.
	if len(payload.Actions) > domain.MaxPushActions {
		payload.Actions = payload.Actions[:domain.MaxPushActions]
	}

	// A shared tag makes each notification replace the last on the device;
	// default to one tag per broadcast, an explicit tag opts into replacing.
	if payload.Tag == "" {
		payload.Tag = broadcastID
	}

	successCount := s.fanOut(ctx, subs, payload)
	failureCount := len(subs) - successCount

	broadcast := &domain.PushBroadcast{
		ID:            broadcastID,
		Title:         req.Title,
		Body:          req.Body,
		URL:           payload.URL,
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

	return broadcast, nil
}

// NotifyCustomer notifies every device a customer opted in on. It reuses the
// broadcast fan-out, so dead endpoints are pruned and rejections logged.
func (s *PushService) NotifyCustomer(
	ctx context.Context, customerID string, payload domain.PushPayload,
) (int, error) {
	if customerID == "" {
		return 0, errors.BadRequest("A customer is required to notify")
	}

	// The dev gateway stands in when VAPID is unconfigured and its Send always
	// succeeds, so an unset key would otherwise log as delivered=N.
	if s.gateway.PublicKey() == "" {
		slog.ErrorContext(ctx, "Cannot notify a customer: no VAPID key is configured",
			"customer_id", customerID)
		return 0, errors.Internal("Push notifications are not configured")
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
	var succeeded atomic.Int32
	g := new(errgroup.Group)
	g.SetLimit(broadcastConcurrency)

	for _, sub := range subs {
		g.Go(func() error {
			if err := s.deliver(ctx, sub, payload); err != nil {
				// Without this a PARTIAL broadcast is a bare count with no way
				// to learn which endpoint rejected it, or why.
				slog.WarnContext(ctx, "Push broadcast delivery failed",
					"error", err, "subscription_id", sub.ID, "endpoint", sub.Endpoint)
				return nil
			}
			succeeded.Add(1)
			return nil
		})
	}

	_ = g.Wait()
	return int(succeeded.Load())
}

// deliver encodes and sends one push, deactivating the subscription when the
// push service reports the endpoint is permanently gone.
func (s *PushService) deliver(ctx context.Context, sub *domain.PushSubscription, payload domain.PushPayload) error {
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
		s.retire(ctx, sub)
	}
	return err
}

// retire takes a permanently gone endpoint out of service. Its pointer goes
// too, or ListByCustomer pays a GetItem for a dead device on every update.
func (s *PushService) retire(ctx context.Context, sub *domain.PushSubscription) {
	if err := s.repo.Deactivate(ctx, sub.Endpoint); err != nil {
		slog.WarnContext(ctx, "Failed to deactivate dead push subscription",
			"error", err, "subscription_id", sub.ID)
	}
	if sub.CustomerID == "" {
		return
	}
	if err := s.repo.UnlinkCustomer(ctx, sub.Endpoint); err != nil {
		slog.WarnContext(ctx, "Failed to drop a dead device's customer pointer",
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
func (s *PushService) ListBroadcasts(ctx context.Context, limit int) ([]*domain.PushBroadcast, error) {
	return s.repo.ListBroadcasts(ctx, limit)
}

// validatePushEndpoint keeps the public subscribe route from pointing the
// backend's VAPID-signed sends at an arbitrary host of the caller's choosing.
func validatePushEndpoint(endpoint string) error {
	// Userinfo and port are rejected, not ignored: Go drops userinfo when
	// dialing, so a@host and b@host reach one device under two row keys.
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
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
