package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
)

// This handler is the seam between the provider and the money: it decides which
// service call an event becomes. Getting the routing wrong settles the wrong
// refund, or silently settles none.
type recordingRefundService struct {
	completed []string
	failed    []string
	failErr   error
}

func (r *recordingRefundService) Create(context.Context, string, domain.CreateRefundRequest, string) (*domain.Refund, error) {
	return nil, nil
}
func (r *recordingRefundService) ListByOrder(context.Context, string) ([]*domain.Refund, error) {
	return nil, nil
}
func (r *recordingRefundService) RecheckStatus(context.Context, string) (*domain.Refund, error) {
	return nil, nil
}

func (r *recordingRefundService) HandleRefundCompleted(_ context.Context, providerRefundID string) error {
	r.completed = append(r.completed, providerRefundID)
	return r.failErr
}

func (r *recordingRefundService) HandleRefundFailed(_ context.Context, providerRefundID, _, _ string) error {
	r.failed = append(r.failed, providerRefundID)
	return r.failErr
}

// The payment half must stay untouched by refund events, and vice versa.
type recordingPaymentService struct{ calls int }

func (p *recordingPaymentService) InitiatePayment(context.Context, domain.InitiatePaymentRequest) (*domain.PaymentResponse, error) {
	return nil, nil
}
func (p *recordingPaymentService) HandlePaymentSuccess(context.Context, domain.PaymentWebhookEvent) error {
	p.calls++
	return nil
}
func (p *recordingPaymentService) HandlePaymentFailure(context.Context, domain.PaymentWebhookEvent) error {
	p.calls++
	return nil
}
func (p *recordingPaymentService) HandlePaymentPending(context.Context, domain.PaymentWebhookEvent) error {
	p.calls++
	return nil
}
func (p *recordingPaymentService) GetByOrderID(context.Context, string) (*domain.Payment, error) {
	return nil, nil
}
func (p *recordingPaymentService) GetByMerchantTxnID(context.Context, string) (*domain.Payment, error) {
	return nil, nil
}
func (p *recordingPaymentService) CheckProviderStatus(context.Context, string) (*domain.ProviderPaymentStatus, error) {
	return nil, nil
}

func postWebhook(t *testing.T, h *WebhookHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/phonepe", strings.NewReader(body))
	req.Header.Set("Authorization", "anything")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func newWebhookHandler(t *testing.T) (*WebhookHandler, *recordingRefundService, *recordingPaymentService) {
	t.Helper()
	refunds := &recordingRefundService{}
	payments := &recordingPaymentService{}
	// DevClient accepts any signature, which is what lets this test drive the
	// handler rather than the credential check.
	return NewWebhookHandler(payments, refunds, phonepe.NewDevClient(""), "u", "p"), refunds, payments
}

func TestPhonePeWebhook_RoutesRefundEvents(t *testing.T) {
	t.Run("settles the refund a completed event names", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"pg.refund.completed","payload":{"refundId":"OMR_9"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []string{"OMR_9"}, refunds.completed)
		require.Empty(t, refunds.failed)
		require.Zero(t, payments.calls, "a refund event must not touch the payment path")
	})

	t.Run("fails the refund a failed event names", func(t *testing.T) {
		h, refunds, _ := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"pg.refund.failed","payload":{"refundId":"OMR_9","errorCode":"E","detailedErrorCode":"D"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []string{"OMR_9"}, refunds.failed)
		require.Empty(t, refunds.completed)
	})

	// PhonePe adds events without warning; an unrecognized one must be
	// acknowledged and ignored, never guessed at.
	t.Run("ignores an event it does not recognize", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"pg.something.new","payload":{"refundId":"OMR_9"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, refunds.completed)
		require.Empty(t, refunds.failed)
		require.Zero(t, payments.calls)
	})

	// PhonePe retries anything that is not a 200, so a service failure must still
	// be acknowledged — the settlement is guarded by its own conditional write.
	t.Run("acknowledges the delivery even when settlement fails", func(t *testing.T) {
		h, refunds, _ := newWebhookHandler(t)
		refunds.failErr = context.DeadlineExceeded

		rec := postWebhook(t, h, `{"event":"pg.refund.completed","payload":{"refundId":"OMR_9"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []string{"OMR_9"}, refunds.completed)
	})

	t.Run("still routes payment events", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"checkout.order.completed","payload":{"merchantOrderId":"txn_1","state":"COMPLETED"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 1, payments.calls)
		require.Empty(t, refunds.completed, "a payment event must not touch the refund path")
	})
}
