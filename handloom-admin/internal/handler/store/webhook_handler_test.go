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

// This handler is the seam between the provider and the money: it decides which service
// call an event becomes. Wrong routing settles the wrong refund, or silently none.
type recordingRefundService struct {
	completed []string
	failed    []string
	failErr   error
}

func (r *recordingRefundService) Create(context.Context, string, domain.CreateRefundRequest, string) (*domain.Refund, error) {
	return nil, nil
}
func (r *recordingRefundService) Preview(context.Context, string, domain.PreviewRefundRequest) (*domain.RefundPreview, error) {
	return nil, nil
}
func (r *recordingRefundService) ListByOrder(context.Context, string) ([]*domain.Refund, error) {
	return nil, nil
}
func (r *recordingRefundService) RecheckStatus(context.Context, string, string) (*domain.Refund, error) {
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
type recordingPaymentService struct {
	calls   int
	failErr error
}

func (p *recordingPaymentService) InitiatePayment(context.Context, domain.InitiatePaymentRequest) (*domain.PaymentResponse, error) {
	return nil, nil
}
func (p *recordingPaymentService) HandlePaymentSuccess(context.Context, domain.PaymentWebhookEvent) error {
	p.calls++
	return p.failErr
}
func (p *recordingPaymentService) HandlePaymentFailure(context.Context, domain.PaymentWebhookEvent) error {
	p.calls++
	return p.failErr
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

	// PhonePe documents exactly four events, and a pending one is not among them. An
	// event we do not model must be acknowledged and dropped, not routed anywhere.
	t.Run("acknowledges an event it does not model without touching either path", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"checkout.order.pending","payload":{"orderId":"OMO_1"}}`)

		require.Equal(t, http.StatusOK, rec.Code, "answering 200 stops PhonePe retrying forever")
		require.Zero(t, payments.calls)
		require.Empty(t, refunds.completed)
		require.Empty(t, refunds.failed)
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

	// This asserted 200 until dev produced the case it was reasoning about
	// backwards: two orders paid at PhonePe sat PENDING for five hours because the
	// only thing that settles them had failed and been acknowledged. The
	// conditional write makes a retry *safe*; it does nothing about never
	// retrying. PhonePe's retries are bounded, an unsettled paid order is not.
	t.Run("asks for redelivery when a valid delivery fails to process", func(t *testing.T) {
		h, refunds, _ := newWebhookHandler(t)
		refunds.failErr = context.DeadlineExceeded

		rec := postWebhook(t, h, `{"event":"pg.refund.completed","payload":{"refundId":"OMR_9"}}`)

		require.Equal(t, http.StatusInternalServerError, rec.Code,
			"a 200 here is a settlement silently dropped")
		require.Equal(t, []string{"OMR_9"}, refunds.completed)
	})

	t.Run("asks for redelivery when a payment fails to settle", func(t *testing.T) {
		h, _, payments := newWebhookHandler(t)
		payments.failErr = context.DeadlineExceeded

		rec := postWebhook(t, h, `{"event":"checkout.order.completed","payload":{"merchantOrderId":"txn_1","state":"COMPLETED"}}`)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, 1, payments.calls)
	})

	// A delivery we cannot read is not one a retry fixes, so it stays
	// acknowledged — otherwise PhonePe redelivers the same broken body.
	t.Run("acknowledges a body it cannot parse", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `not json`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Zero(t, payments.calls)
		require.Empty(t, refunds.completed)
	})

	t.Run("still routes payment events", func(t *testing.T) {
		h, refunds, payments := newWebhookHandler(t)

		rec := postWebhook(t, h, `{"event":"checkout.order.completed","payload":{"merchantOrderId":"txn_1","state":"COMPLETED"}}`)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 1, payments.calls)
		require.Empty(t, refunds.completed, "a payment event must not touch the refund path")
	})
}
