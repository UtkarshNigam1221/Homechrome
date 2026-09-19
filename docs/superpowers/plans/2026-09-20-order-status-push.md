# Order Status Push Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an admin moves an order to SHIPPED, DELIVERED or CANCELLED, every browser that customer has opted in on receives a push notification about that order.

**Architecture:** Push subscriptions gain an optional `customer_id`, written when a signed-in customer opts in and back-filled on login. A pointer item (`PK=PUSH_CUST#<customerID>`) makes a customer's devices a single Query, with no new GSI. `PushService` gains `NotifyCustomer`, and `OrderService.UpdateStatus` calls it through a narrow `domain.OrderNotifier` interface so the order package never imports the push package. Delivery is best-effort: a failed push never fails the status change.

**Tech Stack:** Go 1.25, Chi, DynamoDB (single-table, `handloom-notifications`), Google Wire, gomock, AWS CDK (Go), Web Push / VAPID via `SherClockHolmes/webpush-go`.

**Spec:** None — this plan is the design record. It was agreed in conversation on 2026-09-20 with one explicit decision: hook push **directly** into `UpdateStatus` rather than building a dispatcher that drains the `PENDING` rows `NotificationService.Send` writes. That dispatcher remains unbuilt; email and SMS still go nowhere.

## Global Constraints

- Prices are in **paise** (1 INR = 100 paise). Never format a rupee amount by hand; the notification body must not contain an amount unless it is divided by 100.
- Services return `*errors.AppError` from `pkg/errors`; handlers call `response.Error(w, err)`.
- Handler validation is `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in the handler.
- Run `make wire` after changing any constructor signature, and `make generate-mocks` after changing any interface in `internal/domain/`. Both are checked in CI.
- Import ordering is goimports with local prefix `github.com/handloom/admin`.
- golangci-lint thresholds: gocognit=30, gocyclo=25, dupl=200. Run `golangci-lint run` before every commit.
- Comments are capped at two lines. Explain *why*, never *what*.
- Never log a push endpoint's `keys` (p256dh/auth): endpoint + keys is the full credential to push to that device.
- A push failure must never fail the operation that triggered it.
- Every new test must fail before its implementation exists. A test that passes on an empty implementation is a plan failure.

## Out of scope

- `OrderService.CancelOrder` (the separate cancel path) — covered only where a cancel arrives through `UpdateStatus`.
- Email and SMS channels, and any dispatcher for the `PENDING` notification rows.
- The admin "send a custom message to one customer" surface. Task 5 leaves `NotifyCustomer` ready for it; the UI is a separate plan.
- DTDC or any carrier integration.

## Security decision, stated once

Today only the push Lambda can read `/handloom/{env}/vapid-private-key` (`infra/stacks/api.go`, the `if svc == "push"` block). Hooking push into `UpdateStatus` means the **order** Lambda signs pushes too, so Task 6 grants it the same parameter. That widens the blast radius from one Lambda to two, both admin-tier. The alternative — order Lambda invokes push Lambda via `lambdaclient.InvokeAsync` — keeps the key in one place but needs an API-Gateway-shaped event and an internal auth path, which is the larger change. If the key must stay confined later, that is the upgrade; nothing in Tasks 1-5 has to change for it, because `OrderService` only ever sees the `domain.OrderNotifier` interface.

## File Structure

| File | Responsibility |
|---|---|
| `internal/domain/push_subscription.go` (modify) | `CustomerID` on `PushSubscription` and `SubscribePushRequest`; `ListByCustomer` and `LinkCustomer` on the repository interface |
| `internal/domain/order_notifier.go` (create) | `OrderNotifier` interface — the single method `OrderService` depends on |
| `internal/repository/dynamodb/push_subscription_repository.go` (modify) | Write/read the `PUSH_CUST#` pointer item; `ListByCustomer`, `LinkCustomer` |
| `internal/service/push_service.go` (modify) | `NotifyCustomer`; capture `customer_id` in `Subscribe` |
| `internal/service/order_push_copy.go` (create) | Status → notification title/body/URL. Isolated so copy edits never touch order logic |
| `internal/service/order_service.go` (modify) | Optional `notifier` dependency; best-effort call at the end of `UpdateStatus` |
| `internal/handler/store/push_handler.go` (modify) | Read the customer from context on subscribe; new `POST /link` |
| `internal/router/store_push.go` (modify) | Mount `/link` behind `CustomerAuth` |
| `internal/wire/providers.go` + `wire_gen.go` (modify) | Inject `PushService` as the order Lambda's and monolith's `OrderNotifier` |
| `infra/stacks/api.go` (modify) | Give the order Lambda the VAPID env + SSM read |
| `homechrome-store/src/hooks/usePushNotifications.ts` (modify) | Call `/link` after login |

---

### Task 1: Record which customer a subscription belongs to

**Files:**
- Modify: `handloom-admin/internal/domain/push_subscription.go`
- Modify: `handloom-admin/internal/service/push_service.go`
- Modify: `handloom-admin/internal/middleware/customer_auth.go`
- Modify: `handloom-admin/internal/router/store_push.go`
- Test: `handloom-admin/internal/service/push_service_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `PushSubscription.CustomerID string`; `SubscribePushRequest.CustomerID` is **not** added — the customer comes from the request context, never the body, so a caller cannot claim to be someone else. `PushService.Subscribe(ctx context.Context, req domain.SubscribePushRequest, userAgent string) (*domain.PushSubscription, error)` keeps its signature and reads `middleware.GetCustomerIDFromContext(ctx)` internally.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/push_service_test.go`. That file is `package service` (not `service_test`), so constructors are called unqualified. It uses a hand-written `fakeGateway` — built by `newFakeGateway()`, with `sentCount()` and `sentTo(endpoint)` helpers — and there is **no** `MockPushGateway`. The canonical setup, copied from `TestPushService_Subscribe`, is:

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

repo := mocks.NewMockPushSubscriptionRepository(ctrl)
gw := newFakeGateway()
svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))
```

```go
func TestSubscribeRecordsTheSignedInCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return true, nil
		})

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	_, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
	}, "Mozilla/5.0")

	require.NoError(t, err)
	require.Equal(t, "cust_42", saved.CustomerID)
}

func TestSubscribeWithoutASignedInCustomerStaysAnonymous(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return true, nil
		})

	_, err := svc.Subscribe(context.Background(), domain.SubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
	}, "Mozilla/5.0")

	require.NoError(t, err)
	require.Empty(t, saved.CustomerID)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin && go test ./internal/service/ -run TestSubscribeRecords -count=1`
Expected: FAIL — `saved.CustomerID` undefined (the field does not exist yet).

- [ ] **Step 3: Add the field**

In `internal/domain/push_subscription.go`, inside `PushSubscription`, directly after the `Device` field:

```go
	// CustomerID links this device to a signed-in shopper, so an order update
	// can reach their devices and no one else's. Empty for anonymous opt-ins.
	CustomerID string `json:"customer_id,omitempty" dynamodbav:"customer_id,omitempty"`
```

- [ ] **Step 4: Populate it in the service**

In `internal/service/push_service.go`, inside `Subscribe`, where the `&domain.PushSubscription{...}` literal is built, add:

```go
		CustomerID: middleware.GetCustomerIDFromContext(ctx),
```

Add `"github.com/handloom/admin/internal/middleware"` to the imports if it is not already there.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd handloom-admin && go test ./internal/service/ -run TestSubscribe -count=1`
Expected: PASS

- [ ] **Step 6: Make the store route see the customer when there is one**

`/api/v1/store/push/*` is mounted with no auth (`internal/router/store_push.go`) so anonymous visitors can opt in — that must not change. Add an *optional* customer read so a signed-in shopper is recognised without making auth mandatory.

In `internal/middleware/customer_auth.go`, below the existing `GetCustomerIDFromContext`:

```go
// OptionalCustomer populates the customer context when a valid token is
// present and does nothing when it is absent or bad. For public routes that
// behave differently for a signed-in shopper but must still serve everyone.
func (m *CustomerAuth) OptionalCustomer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx, err := m.authenticate(r); err == nil {
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
```

Read `customer_auth.go` first. It already has the token-parsing logic that produces the context seen at line 55-62; extract it into an `authenticate(r *http.Request) (context.Context, error)` helper that the existing required-auth middleware and this new one both call. Do not duplicate the parsing.

- [ ] **Step 7: Apply it to the store push routes**

In `internal/router/store_push.go`, change `NewStorePushRouter` to take the customer auth middleware and wrap the mount:

```go
func NewStorePushRouter(r *chi.Mux, h *store.PushHandler, customerAuth *middleware.CustomerAuth) {
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(20, time.Minute))
		// Optional, not required: anonymous visitors must still be able to opt
		// in, but a signed-in shopper's devices get linked to their orders.
		r.Use(customerAuth.OptionalCustomer)
		r.Mount("/api/v1/store/push", h.Routes())
	})
}
```

Update both call sites — `cmd/api/main.go` and `cmd/lambda/push/main.go`. Find them with `grep -rn "NewStorePushRouter" cmd/`.

- [ ] **Step 8: Regenerate mocks and verify the build**

```bash
cd handloom-admin && make generate-mocks && go build ./... && golangci-lint run
```
Expected: no output from build, `0 issues` from lint.

- [ ] **Step 9: Commit**

```bash
cd handloom-admin
git add internal/domain/push_subscription.go internal/service/push_service.go \
  internal/service/push_service_test.go internal/middleware/customer_auth.go \
  internal/router/store_push.go internal/mocks/ cmd/
git commit -m "feat(push): record which customer a subscription belongs to"
```

---

### Task 2: Find a customer's devices

**Files:**
- Modify: `handloom-admin/internal/domain/push_subscription.go`
- Modify: `handloom-admin/internal/repository/dynamodb/push_subscription_repository.go`
- Test: `handloom-admin/internal/repository/dynamodb/push_subscription_repository_test.go`

**Interfaces:**
- Consumes: `PushSubscription.CustomerID` from Task 1.
- Produces: `PushSubscriptionRepository.ListByCustomer(ctx context.Context, customerID string) ([]*PushSubscription, error)` — returns only ACTIVE subscriptions, and an empty slice (never an error) when the customer has none.

**Why a pointer item and not a GSI:** GSI1 is already partitioned by subscription status, and a second GSI costs a table change plus its own write capacity for a lookup that returns two or three rows. A pointer item under `PK=PUSH_CUST#<customerID>` answers it with one Query against the base table.

- [ ] **Step 1: Write the failing test**

**Create** `internal/repository/dynamodb/push_subscription_repository_test.go` — it does not exist yet. There is no `newTestPushRepo`; use the package's real harness from `testhelper_test.go`:

```go
func newPushRepo(t *testing.T) (*PushSubscriptionRepository, *dynamodb.Client) {
	t.Helper()
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testNotificationsTable)
	t.Cleanup(func() { cleanupTestTable(t, raw, testNotificationsTable) })
	return NewPushSubscriptionRepository(wrapped), raw
}
```

Check the real constructor name with `grep -n "func NewPushSubscriptionRepository" internal/repository/dynamodb/push_subscription_repository.go` and match it. `skipIfNoLocal` skips locally and **fails** under CI, so a missing container cannot pass as a green run.

```go
func TestListByCustomer(t *testing.T) {
	repo, _ := newPushRepo(t)
	ctx := context.Background()
	mine := &domain.PushSubscription{
		Endpoint:   "https://fcm.googleapis.com/fcm/send/mine",
		Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		CustomerID: "cust_1",
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  time.Now(),
	}
	theirs := &domain.PushSubscription{
		Endpoint:   "https://fcm.googleapis.com/fcm/send/theirs",
		Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		CustomerID: "cust_2",
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  time.Now(),
	}
	anonymous := &domain.PushSubscription{
		Endpoint:  "https://fcm.googleapis.com/fcm/send/anon",
		Keys:      domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		Status:    domain.PushSubscriptionActive,
		CreatedAt: time.Now(),
	}
	for _, sub := range []*domain.PushSubscription{mine, theirs, anonymous} {
		_, err := repo.Save(ctx, sub)
		require.NoError(t, err)
	}

	t.Run("returns only that customer's devices", func(t *testing.T) {
		got, err := repo.ListByCustomer(ctx, "cust_1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, mine.Endpoint, got[0].Endpoint)
	})

	t.Run("a customer with no devices is empty, not an error", func(t *testing.T) {
		got, err := repo.ListByCustomer(ctx, "cust_nobody")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	// A device that unsubscribed must not receive an order update; leaving the
	// pointer behind while filtering on read keeps deactivation a single write.
	t.Run("skips a device that has since unsubscribed", func(t *testing.T) {
		require.NoError(t, repo.Deactivate(ctx, mine.Endpoint))
		got, err := repo.ListByCustomer(ctx, "cust_1")
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin && go test ./internal/repository/dynamodb/ -run TestListByCustomer -count=1 -v`
Expected: FAIL — `repo.ListByCustomer` undefined.

**Note:** if the output says `ok ... [no tests to run]` or the test silently skips, DynamoDB Local is not up. Run `make docker-up` first — a skipped test proves nothing.

- [ ] **Step 3: Add the interface method**

In `internal/domain/push_subscription.go`, inside `PushSubscriptionRepository`, after `ListActive`:

```go
	// ListByCustomer retrieves a customer's ACTIVE subscriptions. A customer
	// with no devices yields an empty slice, not an error.
	ListByCustomer(ctx context.Context, customerID string) ([]*PushSubscription, error)
```

- [ ] **Step 4: Write the pointer item on save**

In `internal/repository/dynamodb/push_subscription_repository.go`, add near the other key helpers:

```go
// custPointerKey addresses the row that lets a customer's devices be found
// without a second GSI on a table whose GSI1 is already spent on status.
func custPointerKey(customerID, subID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "PUSH_CUST#" + customerID},
		"SK": &types.AttributeValueMemberS{Value: "PUSH_SUB#" + subID},
	}
}
```

Then in `Save`, after the existing `PutItem` succeeds and before `return isNew, nil`:

```go
	if sub.CustomerID != "" {
		item := custPointerKey(sub.CustomerID, sub.ID)
		item["endpoint"] = &types.AttributeValueMemberS{Value: sub.Endpoint}
		item["entity_type"] = &types.AttributeValueMemberS{Value: "PUSH_SUB_CUSTOMER"}
		if _, err := r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(r.client.notificationsTable),
			Item:      item,
		}); err != nil {
			return isNew, errors.Wrap(err, "Failed to link push subscription to customer")
		}
	}
```

- [ ] **Step 5: Implement ListByCustomer**

Add to the same file, after `ListActive`:

```go
// ListByCustomer queries the pointer rows, then reads each subscription. A
// customer has a handful of devices, so the extra round trips are cheaper than
// a GSI that would have to be written on every subscribe.
func (r *PushSubscriptionRepository) ListByCustomer(
	ctx context.Context, customerID string,
) ([]*domain.PushSubscription, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.notificationsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "PUSH_CUST#" + customerID},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list a customer's push subscriptions")
	}

	subs := make([]*domain.PushSubscription, 0, len(result.Items))
	for _, item := range result.Items {
		endpoint, ok := item["endpoint"].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		sub, err := r.GetByEndpoint(ctx, endpoint.Value)
		if err != nil {
			// A pointer outliving its subscription is not this caller's problem:
			// the device simply cannot be reached.
			continue
		}
		if sub.Status == domain.PushSubscriptionActive {
			subs = append(subs, sub)
		}
	}
	return subs, nil
}
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd handloom-admin && make docker-up && go test ./internal/repository/dynamodb/ -run TestListByCustomer -count=1 -v
```
Expected: PASS, with all three subtests named in the output. If any subtest is missing from the output, it did not run.

- [ ] **Step 7: Regenerate mocks, lint**

```bash
cd handloom-admin && make generate-mocks && go build ./... && golangci-lint run
```
Expected: `0 issues`.

- [ ] **Step 8: Commit**

```bash
cd handloom-admin
git add internal/domain/push_subscription.go internal/repository/dynamodb/ internal/mocks/
git commit -m "feat(push): find a customer's subscribed devices"
```

---

### Task 3: Link devices subscribed before sign-in

**Files:**
- Modify: `handloom-admin/internal/domain/push_subscription.go`
- Modify: `handloom-admin/internal/repository/dynamodb/push_subscription_repository.go`
- Modify: `handloom-admin/internal/service/push_service.go`
- Modify: `handloom-admin/internal/handler/store/push_handler.go`
- Modify: `handloom-admin/internal/router/store_push.go`
- Test: `handloom-admin/internal/service/push_service_test.go`

**Interfaces:**
- Consumes: `CustomerID` (Task 1), `custPointerKey` (Task 2).
- Produces: `PushSubscriptionRepository.LinkCustomer(ctx context.Context, endpoint, customerID string) error`; `PushService.LinkCustomer(ctx context.Context, endpoint string) error`; `POST /api/v1/store/push/link` taking `domain.LinkPushRequest{Endpoint string}`.

**Why:** most shoppers grant notification permission before they ever sign in, so subscribe-time capture alone would leave most devices anonymous forever.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/push_service_test.go`:

```go
func TestLinkCustomerRequiresASignedInCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	// No customer in context: nothing may be written, or an anonymous caller
	// could claim another shopper's device.
	err := svc.LinkCustomer(context.Background(), "https://fcm.googleapis.com/fcm/send/abc")
	require.Error(t, err)
}

func TestLinkCustomerAttachesTheDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().
		LinkCustomer(gomock.Any(), "https://fcm.googleapis.com/fcm/send/abc", "cust_42").
		Return(nil)

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	require.NoError(t, svc.LinkCustomer(ctx, "https://fcm.googleapis.com/fcm/send/abc"))
}

func TestLinkCustomerRejectsAnUnknownPushService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	err := svc.LinkCustomer(ctx, "https://evil.example.com/hook")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin && go test ./internal/service/ -run TestLinkCustomer -count=1`
Expected: FAIL — `svc.LinkCustomer` undefined.

- [ ] **Step 3: Add the request type and interface method**

In `internal/domain/push_subscription.go`, beside `UnsubscribePushRequest`:

```go
// LinkPushRequest attaches an already-registered device to the signed-in
// shopper. The customer comes from the session, never from the body.
type LinkPushRequest struct {
	Endpoint string `json:"endpoint" validate:"required,url,max=512"`
}
```

And in `PushSubscriptionRepository`, after `ListByCustomer`:

```go
	// LinkCustomer attaches an existing subscription to a customer. Linking an
	// endpoint that is not stored is a no-op, not an error.
	LinkCustomer(ctx context.Context, endpoint, customerID string) error
```

- [ ] **Step 4: Implement the repository method**

In `internal/repository/dynamodb/push_subscription_repository.go`:

```go
// LinkCustomer sets customer_id on the subscription and writes the pointer row
// that ListByCustomer reads. A device that has gone away links to nothing,
// which is the same outcome the caller wanted.
func (r *PushSubscriptionRepository) LinkCustomer(
	ctx context.Context, endpoint, customerID string,
) error {
	sub, err := r.GetByEndpoint(ctx, endpoint)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	sub.CustomerID = customerID
	if _, err := r.Save(ctx, sub); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 5: Implement the service method**

In `internal/service/push_service.go`, after `Subscribe`:

```go
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
```

Check the exact constructor in `pkg/errors` for a 401 before writing `errors.Unauthorized` — `grep -n "func Unauthorized" pkg/errors/*.go`. Use whatever that file actually exports.

- [ ] **Step 6: Add the route**

In `internal/handler/store/push_handler.go`, inside `Routes()`:

```go
	r.With(middleware.ValidateJSONTyped[domain.LinkPushRequest](h.validation)).
		Post("/link", h.Link)
```

And the handler:

```go
// Link attaches this browser's subscription to the signed-in shopper.
// POST /api/v1/store/push/link
func (h *PushHandler) Link(w http.ResponseWriter, r *http.Request) {
	req := middleware.MustGetValidatedBody[domain.LinkPushRequest](r.Context())
	if err := h.pushService.LinkCustomer(r.Context(), req.Endpoint); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"linked": true})
}
```

The route sits inside the group Task 1 wrapped in `OptionalCustomer`, so a signed-in caller is recognised and an anonymous one gets the 401 from the service. No separate mount is needed.

- [ ] **Step 7: Run the tests to verify they pass**

```bash
cd handloom-admin && make generate-mocks && go test ./internal/service/ -run TestLinkCustomer -count=1 -v
```
Expected: PASS, all three subtests named.

- [ ] **Step 8: Call it from the storefront after login**

In `homechrome-store/src/hooks/usePushNotifications.ts`, add an exported function beside the existing ones:

```ts
/**
 * Attach this browser's existing subscription to the shopper who just signed
 * in. Most people allow notifications before logging in, so without this their
 * device is never linked to their orders.
 */
export async function linkPushSubscription(): Promise<void> {
  if (!pushSupported()) return;
  try {
    const registration = await navigator.serviceWorker.getRegistration();
    const subscription = registration ? await registration.pushManager.getSubscription() : null;
    if (!subscription) return;
    await apiClient.post(ROUTES.PUSH.LINK, { endpoint: subscription.endpoint });
  } catch {
    // Linking is an optimisation: a shopper who misses it simply gets no order
    // pushes on this device until they next re-subscribe.
  }
}
```

Add `LINK: '/api/v1/store/push/link'` to the `PUSH` group in `homechrome-store/src/lib/routes.ts` (find the file with `grep -rn "PUSH" homechrome-store/src/lib/`). Then call `linkPushSubscription()` wherever the OTP verify succeeds — find it with `grep -rn "verify" homechrome-store/src/ --include=*.ts --include=*.tsx | grep -i otp`. Call it without awaiting; login must not wait on it.

- [ ] **Step 9: Verify the storefront**

```bash
cd homechrome-store && npx tsc --noEmit && npm run lint
```
Expected: no type errors; no new lint warnings beyond the 7 that already exist on this branch.

- [ ] **Step 10: Commit**

```bash
git add handloom-admin/internal/ handloom-admin/internal/mocks/ homechrome-store/src/
git commit -m "feat(push): link a device to the shopper who signs in on it"
```

---

### Task 4: Notify one customer

**Files:**
- Modify: `handloom-admin/internal/service/push_service.go`
- Test: `handloom-admin/internal/service/push_service_test.go`

**Interfaces:**
- Consumes: `ListByCustomer` (Task 2).
- Produces: `PushService.NotifyCustomer(ctx context.Context, customerID string, payload domain.PushPayload) (delivered int, err error)`.

- [ ] **Step 1: Write the failing test**

```go
func TestNotifyCustomerReachesEveryDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().ListByCustomer(gomock.Any(), "cust_1").Return([]*domain.PushSubscription{
		{ID: "a", Endpoint: "https://fcm.googleapis.com/fcm/send/a", Status: domain.PushSubscriptionActive},
		{ID: "b", Endpoint: "https://fcm.googleapis.com/fcm/send/b", Status: domain.PushSubscriptionActive},
	}, nil)

	delivered, err := svc.NotifyCustomer(context.Background(), "cust_1", domain.PushPayload{
		Title: "Your order has shipped",
		Body:  "HL-1 is on its way.",
		URL:   "/account/orders/order_1",
	})

	require.NoError(t, err)
	require.Equal(t, 2, delivered)
	require.Equal(t, 2, gw.sentCount())
}

func TestNotifyCustomerWithNoDevicesIsNotAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().ListByCustomer(gomock.Any(), "cust_1").Return(nil, nil)

	delivered, err := svc.NotifyCustomer(context.Background(), "cust_1", domain.PushPayload{
		Title: "Your order has shipped",
		Body:  "HL-1 is on its way.",
	})

	require.NoError(t, err)
	require.Zero(t, delivered)
}

func TestNotifyCustomerRequiresACustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	// No ListByCustomer expectation: an empty id must never become a query that
	// could match the anonymous partition.
	_, err := svc.NotifyCustomer(context.Background(), "", domain.PushPayload{Title: "x", Body: "y"})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin && go test ./internal/service/ -run TestNotifyCustomer -count=1`
Expected: FAIL — `svc.NotifyCustomer` undefined.

- [ ] **Step 3: Implement it**

In `internal/service/push_service.go`, after `Broadcast`:

```go
// NotifyCustomer delivers one notification to every device a customer has
// opted in on. It reuses the broadcast fan-out, so a dead endpoint is pruned
// and a rejection is logged the same way it is for a broadcast.
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd handloom-admin && go test ./internal/service/ -run TestNotifyCustomer -count=1 -v`
Expected: PASS, all three named.

- [ ] **Step 5: Lint and commit**

```bash
cd handloom-admin && golangci-lint run && \
git add internal/service/ && \
git commit -m "feat(push): deliver one notification to a single customer"
```

---

### Task 5: Order status copy, and the notifier interface

**Files:**
- Create: `handloom-admin/internal/domain/order_notifier.go`
- Create: `handloom-admin/internal/service/order_push_copy.go`
- Test: `handloom-admin/internal/service/order_push_copy_test.go`
- Modify: `handloom-admin/Makefile` (mockgen line for the new domain file)

**Interfaces:**
- Consumes: nothing.
- Produces: `domain.OrderNotifier` with `NotifyCustomer(ctx context.Context, customerID string, payload PushPayload) (int, error)` — deliberately the exact signature `PushService.NotifyCustomer` already has, so `*PushService` satisfies it with no adapter. Also `orderStatusPush(order *domain.Order) *domain.PushPayload`, returning `nil` for a status not worth interrupting someone for.

- [ ] **Step 1: Write the failing test**

Create `internal/service/order_push_copy_test.go`:

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// Named for the status it carries: `order` alone would shadow the local
// variable of that name used throughout package service.
func orderWithStatus(status domain.OrderStatus) *domain.Order {
	return &domain.Order{ID: "order_1", OrderNumber: "HL-1042", Status: status}
}

func TestOrderStatusPushCopy(t *testing.T) {
	t.Run("shipped tells the shopper their parcel is moving", func(t *testing.T) {
		payload := orderStatusPush(orderWithStatus(domain.OrderStatusShipped))
		require.NotNil(t, payload)
		require.Contains(t, payload.Title, "HL-1042")
		require.NotEmpty(t, payload.Body)
		require.Equal(t, "/account/orders/order_1", payload.URL)
	})

	t.Run("delivered and cancelled are both worth a notification", func(t *testing.T) {
		require.NotNil(t, orderStatusPush(orderWithStatus(domain.OrderStatusDelivered)))
		require.NotNil(t, orderStatusPush(orderWithStatus(domain.OrderStatusCancelled)))
	})

	// Interrupting someone for a state they never see is how an opt-in gets
	// revoked. Only the states a shopper is actually waiting on send.
	t.Run("internal states send nothing", func(t *testing.T) {
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusPending)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusConfirmed)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusProcessing)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusReturned)))
	})

	t.Run("every notification is tagged per order so a second one replaces the first", func(t *testing.T) {
		shipped := orderStatusPush(orderWithStatus(domain.OrderStatusShipped))
		delivered := orderStatusPush(orderWithStatus(domain.OrderStatusDelivered))
		require.Equal(t, shipped.Tag, delivered.Tag)
		require.Contains(t, shipped.Tag, "order_1")
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin && go test ./internal/service/ -run TestOrderStatusPushCopy -count=1`
Expected: FAIL — `orderStatusPush` undefined.

- [ ] **Step 3: Write the copy**

Create `internal/service/order_push_copy.go`:

```go
package service

import (
	"fmt"

	"github.com/handloom/admin/internal/domain"
)

// orderStatusPush is the notification for an order reaching a status, or nil
// when the status is one a shopper is not waiting on. Kept apart from the order
// service so wording changes never touch fulfilment logic.
func orderStatusPush(order *domain.Order) *domain.PushPayload {
	var title, body string

	switch order.Status {
	case domain.OrderStatusShipped:
		title = fmt.Sprintf("%s is on its way", order.OrderNumber)
		body = "Your handloom order has left our workshop. Tap to follow it."
	case domain.OrderStatusDelivered:
		title = fmt.Sprintf("%s has arrived", order.OrderNumber)
		body = "Your order was delivered. We hope you love it."
	case domain.OrderStatusCancelled:
		title = fmt.Sprintf("%s was cancelled", order.OrderNumber)
		body = "This order will not ship. Any payment is on its way back to you."
	default:
		return nil
	}

	return &domain.PushPayload{
		Title: title,
		Body:  body,
		URL:   "/account/orders/" + order.ID,
		// One tag per order: a delivered notice replaces the shipped one rather
		// than stacking two cards about the same parcel.
		Tag: "order-" + order.ID,
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd handloom-admin && go test ./internal/service/ -run TestOrderStatusPushCopy -count=1 -v`
Expected: PASS, all four subtests named.

- [ ] **Step 5: Declare the notifier interface**

Create `internal/domain/order_notifier.go`:

```go
package domain

import "context"

// OrderNotifier is the single push capability the order service depends on.
// Declared here, and narrow, so the order service never imports the push
// service and can be handed a no-op in a Lambda that must not sign pushes.
type OrderNotifier interface {
	NotifyCustomer(ctx context.Context, customerID string, payload PushPayload) (int, error)
}
```

- [ ] **Step 6: Add it to mock generation**

In `handloom-admin/Makefile`, in the `generate-mocks` target, after the `push_subscription.go` line:

```makefile
	mockgen -source=internal/domain/order_notifier.go -destination=internal/mocks/order_notifier_mock.go -package=mocks
```

- [ ] **Step 7: Generate and verify**

```bash
cd handloom-admin && make generate-mocks && go build ./... && golangci-lint run
```
Expected: `internal/mocks/order_notifier_mock.go` exists with `NewMockOrderNotifier`; `0 issues`.

- [ ] **Step 8: Commit**

```bash
cd handloom-admin
git add internal/domain/order_notifier.go internal/service/order_push_copy.go \
  internal/service/order_push_copy_test.go Makefile internal/mocks/
git commit -m "feat(orders): notification copy for the statuses a shopper waits on"
```

---

### Task 6: Fire the notification from UpdateStatus

**Files:**
- Modify: `handloom-admin/internal/service/order_service.go`
- Modify: `handloom-admin/internal/wire/providers.go`
- Modify: `handloom-admin/infra/stacks/api.go:307`
- Test: `handloom-admin/internal/service/order_service_test.go`

**Interfaces:**
- Consumes: `domain.OrderNotifier` and `orderStatusPush` (Task 5), `PushService.NotifyCustomer` (Task 4).
- Produces: `NewOrderService` gains a final parameter `notifier domain.OrderNotifier`, which may be `nil`.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/order_service_test.go`. Reuse the construction the file already uses — read its existing `NewOrderService` call and copy it, adding the notifier as the last argument.

```go
func TestUpdateStatusNotifiesTheCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// ... build the same mocks the other tests in this file build ...
	notifier := mocks.NewMockOrderNotifier(ctrl)

	existing := &domain.Order{
		ID: "order_1", OrderNumber: "HL-1042",
		CustomerID: "cust_1", Status: domain.OrderStatusProcessing,
	}
	mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_1").Return(existing, nil)
	mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	mockPaymentRepo.EXPECT().GetByOrderID(gomock.Any(), "order_1").
		Return(&domain.Payment{Status: domain.PaymentStatusPaid}, nil).AnyTimes()

	notifier.EXPECT().
		NotifyCustomer(gomock.Any(), "cust_1", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, payload domain.PushPayload) (int, error) {
			require.Contains(t, payload.Title, "HL-1042")
			return 1, nil
		})

	svc := service.NewOrderService(/* ...the same args as the other tests... */, notifier)
	require.NoError(t, svc.UpdateStatus(context.Background(), "order_1", domain.OrderStatusShipped, "admin_1"))
}

func TestUpdateStatusSucceedsWhenThePushFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// ... same mocks ...
	notifier := mocks.NewMockOrderNotifier(ctrl)

	existing := &domain.Order{
		ID: "order_1", OrderNumber: "HL-1042",
		CustomerID: "cust_1", Status: domain.OrderStatusProcessing,
	}
	mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_1").Return(existing, nil)
	mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	mockPaymentRepo.EXPECT().GetByOrderID(gomock.Any(), "order_1").
		Return(&domain.Payment{Status: domain.PaymentStatusPaid}, nil).AnyTimes()

	// The order really did ship. A push service being down cannot undo that.
	notifier.EXPECT().NotifyCustomer(gomock.Any(), "cust_1", gomock.Any()).
		Return(0, errors.Internal("push service unreachable"))

	svc := service.NewOrderService(/* ...same args... */, notifier)
	require.NoError(t, svc.UpdateStatus(context.Background(), "order_1", domain.OrderStatusShipped, "admin_1"))
}

func TestUpdateStatusSendsNothingForAnInternalStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// ... same mocks ...
	notifier := mocks.NewMockOrderNotifier(ctrl)

	existing := &domain.Order{
		ID: "order_1", OrderNumber: "HL-1042",
		CustomerID: "cust_1", Status: domain.OrderStatusConfirmed,
	}
	mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_1").Return(existing, nil)
	mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	mockPaymentRepo.EXPECT().GetByOrderID(gomock.Any(), "order_1").
		Return(&domain.Payment{Status: domain.PaymentStatusPaid}, nil).AnyTimes()

	// No NotifyCustomer expectation: PROCESSING must not reach a shopper, and
	// gomock fails the test if it is called.
	svc := service.NewOrderService(/* ...same args... */, notifier)
	require.NoError(t, svc.UpdateStatus(context.Background(), "order_1", domain.OrderStatusProcessing, "admin_1"))
}
```

Check the real `domain.Payment` status constant name before writing `PaymentStatusPaid` — `grep -n "PaymentStatus.* = " internal/domain/entity.go`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd handloom-admin && go test ./internal/service/ -run TestUpdateStatus -count=1`
Expected: FAIL — `NewOrderService` takes the wrong number of arguments.

- [ ] **Step 3: Add the dependency**

In `internal/service/order_service.go`, add to the struct after `pricingService`:

```go
	notifier       domain.OrderNotifier
```

Add a matching final parameter `notifier domain.OrderNotifier` to `NewOrderService` and set it in the returned literal.

- [ ] **Step 4: Call it from UpdateStatus**

In `UpdateStatus`, immediately after `s.applyInventoryEffect(ctx, order, status, updatedBy)` and before the `slog.InfoContext` line:

```go
	s.notifyStatusChange(ctx, order)
```

Then add, below `UpdateStatus`:

```go
// notifyStatusChange pushes the new status to the shopper's devices. Every
// failure is swallowed: the order has already moved, and a notification that
// did not arrive must not roll that back or surface as an admin error.
func (s *OrderService) notifyStatusChange(ctx context.Context, order *domain.Order) {
	if s.notifier == nil || order.CustomerID == "" {
		return
	}
	payload := orderStatusPush(order)
	if payload == nil {
		return
	}
	if _, err := s.notifier.NotifyCustomer(ctx, order.CustomerID, *payload); err != nil {
		slog.WarnContext(ctx, "Could not push the order status to the customer",
			"error", err, "order_id", order.ID, "status", order.Status)
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd handloom-admin && go test ./internal/service/ -run TestUpdateStatus -count=1 -v`
Expected: PASS, all three named. Every other test in the package must still pass — they need the new `nil` argument added to their `NewOrderService` calls.

- [ ] **Step 6: Wire it up**

In `internal/wire/providers.go`, find the `NewOrderService` provider. Add a provider that exposes the push service as the notifier:

```go
// ProvideOrderNotifier hands the order service the push capability. *PushService
// already has the exact NotifyCustomer signature, so no adapter is needed.
func ProvideOrderNotifier(push *service.PushService) domain.OrderNotifier {
	return push
}
```

Add `ProvideOrderNotifier` to the provider sets used by the **monolith** and the **order** Lambda. Do not add it to any other Lambda's set — a Lambda that never notifies should not carry the signing key.

Then: `make wire && go build ./...`

- [ ] **Step 7: Give the order Lambda the signing key**

In `infra/stacks/api.go`, the block at line ~307 currently reads `if svc == "push"`. Change it to cover both, and say why:

```go
		// Both the push Lambda (fan-out) and the order Lambda (status updates)
		// sign pushes, so both need the key. No other Lambda gets it.
		if svc == "push" || svc == "order" {
```

The body is unchanged — it already adds `VAPID_PUBLIC_KEY`, `VAPID_SUBJECT`, `VAPID_PRIVATE_KEY_PARAM` and the scoped `ssm:GetParameter` policy.

- [ ] **Step 8: Prove the key still never enters the template**

The only route by which the private key could reach CloudFormation is a Lambda
environment variable, and nothing in `infra/` ever reads `VAPID_PRIVATE_KEY` —
the push and order Lambdas read it from SSM at runtime. So the check is that
the synthesized template names the parameter but never carries a value.

```bash
cd handloom-admin/infra && cdk synth --all -c environment=dev > /tmp/synth-dev.yaml
```

`cdk synth` needs no AWS credentials here (the account comes from
`CDK_DEFAULT_ACCOUNT`, and no stack uses a `fromLookup`). Then:

```bash
# The parameter NAME and the IAM resource may appear; both are public.
grep -c "vapid-private-key" /tmp/synth-dev.yaml          # expect 2 or more

# A VAPID_PRIVATE_KEY environment variable must NOT exist on any function.
grep -c "VAPID_PRIVATE_KEY\"" /tmp/synth-dev.yaml        # expect 0
grep -c "VAPID_PRIVATE_KEY:" /tmp/synth-dev.yaml          # expect 0

# VAPID_PRIVATE_KEY_PARAM (the pointer, not the secret) SHOULD be present,
# and on exactly the two functions that sign pushes.
grep -c "VAPID_PRIVATE_KEY_PARAM" /tmp/synth-dev.yaml     # expect 2
```

If the third command returns anything but 0, stop and report it: the private
key is being baked into CloudFormation. If the last returns a number other
than 2, report which functions carry it — only the push and order Lambdas
should.

**Do not** try to fetch the real key from SSM to grep for its value. The
shell's AWS SSO session is expired, and that check is unavailable in this
environment; the assertions above are what you run instead.

- [ ] **Step 9: Full verification**

```bash
cd handloom-admin && make docker-up && make test && golangci-lint run
```
Expected: all packages `ok`, `0 issues`. A package reporting `ok` with `[no tests to run]` means its tests skipped — check DynamoDB Local and `POSTGRES_DSN` are both up before believing a green run.

- [ ] **Step 10: Commit**

```bash
cd handloom-admin
git add internal/service/order_service.go internal/service/order_service_test.go \
  internal/wire/ infra/stacks/api.go
git commit -m "feat(orders): push the new status to the shopper's devices"
```

---

### Task 7: End-to-end check against LocalStack

**Files:**
- No production code. This task either passes or sends you back to an earlier task.

**Interfaces:**
- Consumes: everything above.

- [ ] **Step 1: Bring the stack up**

```bash
cd handloom-admin && make setup-local && make run
```
In a second terminal: `cd homechrome-store && npm run dev`

- [ ] **Step 2: Subscribe as a signed-in shopper**

Open `http://localhost:3000`, sign in with the phone OTP flow (the OTP prints to the `make run` terminal), then allow notifications from the bell.

- [ ] **Step 3: Confirm the link landed**

```bash
aws dynamodb query --endpoint-url http://localhost:4566 --region ap-south-1 \
  --table-name handloom-notifications-local \
  --key-condition-expression "PK = :pk" \
  --expression-attribute-values '{":pk":{"S":"PUSH_CUST#<the customer id>"}}'
```
Expected: one item whose `endpoint` matches the browser's subscription. Get the customer id from the `make run` log line for the OTP verify.

- [ ] **Step 4: Place an order and ship it**

Buy anything through the storefront, then in the admin console (`cd handloom-admin-frontend && npm run dev:local`) open the order and move it to SHIPPED.

- [ ] **Step 5: Confirm the notification**

Expected: a notification titled `<order number> is on its way`, clicking it opens `/account/orders/<id>`.

Then move the same order to DELIVERED. Expected: the first notification is **replaced**, not stacked — that is the shared `Tag` working.

- [ ] **Step 6: Confirm an internal status stays silent**

On a second order, move PENDING → CONFIRMED → PROCESSING. Expected: no notification at any step.

- [ ] **Step 7: Confirm a broken push does not break the order**

Stop the storefront, revoke notification permission in the browser, then ship a third order. Expected: the admin console reports the status change as successful, and the `make run` log carries `Could not push the order status to the customer`.

- [ ] **Step 8: Tear down**

```bash
cd handloom-admin && make teardown-local
```

- [ ] **Step 9: Commit nothing, report findings**

This task produces no diff. Report which steps passed and any that did not.

---

## Self-Review

**1. Coverage of the four gaps identified in conversation.**

| Gap | Task |
|---|---|
| Subscriptions are anonymous | 1 (subscribe-time) + 3 (login-time back-fill) |
| No way to find a customer's devices | 2 |
| No send-to-one-customer path | 4 |
| Order status changes notify nobody | 5 (copy + interface) + 6 (the hook) |

The agreed decision — hook directly rather than build a dispatcher — is recorded under **Spec** and is why no task touches `NotificationService`.

**2. Placeholder scan.** The `/* ...same args... */` markers in Task 6 are the one deliberate ellipsis: `NewOrderService` takes seven existing arguments whose mock variables are already built by that test file's own helpers, and spelling them out here would go stale the moment the constructor changes. Each occurrence is paired with an instruction to copy the file's existing call. Everything else is literal.

**3. Type consistency.**

- `NotifyCustomer(ctx, customerID string, payload domain.PushPayload) (int, error)` — identical in Task 4 (implementation), Task 5 (interface), Task 6 (call site and mock).
- `ListByCustomer(ctx, customerID string) ([]*PushSubscription, error)` — identical in Task 2 (interface, implementation, test) and Task 4 (mock expectation).
- `LinkCustomer` appears with two different receivers on purpose: the repository takes `(ctx, endpoint, customerID)`, the service takes `(ctx, endpoint)` and reads the customer from context. Task 3 states both.
- `orderStatusPush(order *domain.Order) *domain.PushPayload` — Task 5 defines it, Task 6 calls it with that exact signature.
- `PushPayload` field names used (`Title`, `Body`, `URL`, `Tag`) all exist on the struct in `internal/domain/push_subscription.go`.

**4. Known risks the executor should not be surprised by.**

- `NewOrderService` gains a parameter, so **every** existing call in `internal/service/order_service_test.go` needs `nil` appended. Task 6 Step 5 says so; a compile error there is expected, not a defect.
- The `PUSH_CUST#` pointer is written on every `Save`, including re-subscribes. `PutItem` makes that idempotent.
- A subscription that is linked and later unsubscribed leaves its pointer row behind. `ListByCustomer` filters on status, and Task 2's third subtest pins that. Cleaning the row up on deactivate is a deliberate non-goal: it would turn a one-write deactivate into two.
