package webpush

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wp "github.com/SherClockHolmes/webpush-go"
)

// A real subscription from Chrome; the keys only need to be well-formed for
// the payload encryption to run.
const (
	testP256dh = "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM"
	testAuth   = "tBHItJI5svbpez7KI4CCXg"
)

func sendTo(t *testing.T, srv *httptest.Server) error {
	t.Helper()

	private, public, err := wp.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	client := NewClient(Config{
		PublicKey:  public,
		PrivateKey: private,
		Subject:    "mailto:info@homechrome.in",
	})
	return client.Send(context.Background(), Subscription{
		Endpoint: srv.URL,
		P256dh:   testP256dh,
		Auth:     testAuth,
	}, []byte(`{"title":"hi"}`))
}

func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSendClassifiesRejections(t *testing.T) {
	t.Run("a retired endpoint is reported as gone so the caller can prune it", func(t *testing.T) {
		if err := sendTo(t, serve(t, http.StatusGone, "")); !errors.Is(err, ErrSubscriptionGone) {
			t.Fatalf("want ErrSubscriptionGone, got %v", err)
		}
	})

	t.Run("a rejection carries the service's reason, not just the status", func(t *testing.T) {
		err := sendTo(t, serve(t, http.StatusForbidden, `{"reason":"BadJwtToken"}`))
		if err == nil {
			t.Fatal("want an error")
		}
		// The status alone cannot separate a bad VAPID subject from a key the
		// endpoint was not subscribed with — both are 403.
		if !strings.Contains(err.Error(), "BadJwtToken") {
			t.Fatalf("want the reason quoted, got %q", err)
		}
		if errors.Is(err, ErrSubscriptionGone) {
			t.Fatal("403 is not terminal: a bad deploy would retire every subscriber")
		}
	})

	t.Run("a rejection with no body still names the status", func(t *testing.T) {
		err := sendTo(t, serve(t, http.StatusRequestEntityTooLarge, ""))
		if err == nil || !strings.Contains(err.Error(), "413") {
			t.Fatalf("want the status quoted, got %v", err)
		}
	})

	t.Run("a delivered push is not an error", func(t *testing.T) {
		if err := sendTo(t, serve(t, http.StatusCreated, "")); err != nil {
			t.Fatalf("want success, got %v", err)
		}
	})
}
