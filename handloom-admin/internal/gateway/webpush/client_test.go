package webpush

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

// decodeJWTSub pulls the "sub" claim out of the VAPID Authorization header
// without verifying it — the signature is not what this asserts.
func decodeJWTSub(t *testing.T, authorization string) string {
	t.Helper()

	token := strings.TrimPrefix(authorization, "vapid t=")
	token, _, _ = strings.Cut(token, ",")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", authorization)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("parse JWT claims: %v", err)
	}
	return claims.Sub
}

// Apple validates the VAPID "sub" claim and answers 403 BadJwtToken when it is
// not a bare mailto: or https: URL. FCM does not look at it, so a malformed
// subject delivers on Android and fails on every Apple device.
func TestSendSignsAWellFormedSubject(t *testing.T) {
	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	private, public, err := wp.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}
	client := NewClient(Config{
		PublicKey:  public,
		PrivateKey: private,
		Subject:    "mailto:info@homechrome.in",
	})
	if err := client.Send(context.Background(), Subscription{
		Endpoint: srv.URL,
		P256dh:   testP256dh,
		Auth:     testAuth,
	}, []byte(`{"title":"hi"}`)); err != nil {
		t.Fatalf("send: %v", err)
	}

	if got := decodeJWTSub(t, authorization); got != "mailto:info@homechrome.in" {
		t.Fatalf("sub claim is %q, want %q", got, "mailto:info@homechrome.in")
	}
}

func TestVapidSubscriber(t *testing.T) {
	cases := map[string]string{
		"mailto:info@homechrome.in":  "info@homechrome.in",
		"info@homechrome.in":         "info@homechrome.in",
		"https://homechrome.in/push": "https://homechrome.in/push",
	}
	// webpush-go restores the scheme for everything that is not an https: URL,
	// so all three must end up as a well-formed subject.
	for subject, want := range cases {
		if got := vapidSubscriber(subject); got != want {
			t.Errorf("vapidSubscriber(%q) = %q, want %q", subject, got, want)
		}
	}
}
