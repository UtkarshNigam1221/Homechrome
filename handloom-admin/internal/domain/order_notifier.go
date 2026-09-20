package domain

import "context"

// OrderNotifier is the only push capability the order service depends on, kept
// narrow so that service never imports the push service.
type OrderNotifier interface {
	NotifyCustomer(ctx context.Context, customerID string, payload PushPayload) (int, error)
}
