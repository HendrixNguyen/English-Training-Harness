package notify

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// browserSubscription fabricates what PushManager.subscribe() hands the PWA:
// an uncompressed P-256 public key and a 16-byte auth secret, base64url.
func browserSubscription(t *testing.T, endpoint string) Subscription {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	return Subscription{
		ID:       "sub-1",
		Endpoint: endpoint,
		P256dh:   base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(auth),
	}
}

func newTestSender(t *testing.T) *WebPushSender {
	t.Helper()
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewWebPushSender(pub, priv, "mailto:test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestWebPushSenderPostsAnEncryptedVAPIDSignedRequest(t *testing.T) {
	var gotAuth, gotEncoding, gotTTL string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotEncoding = r.Header.Get("Content-Encoding")
		gotTTL = r.Header.Get("TTL")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	s := newTestSender(t)
	err := s.Send(context.Background(), browserSubscription(t, srv.URL+"/push/abc"), Payload{Title: "t", Body: "b", URL: "/"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(strings.ToLower(gotAuth), "vapid ") {
		t.Errorf("Authorization = %q, want a VAPID header", gotAuth)
	}
	if gotEncoding != "aes128gcm" {
		t.Errorf("Content-Encoding = %q, want aes128gcm (RFC 8291)", gotEncoding)
	}
	if gotTTL != "3600" {
		t.Errorf("TTL = %q, want 3600", gotTTL)
	}
	if len(gotBody) == 0 || strings.Contains(string(gotBody), `"title"`) {
		t.Errorf("body (%d bytes) must be the encrypted payload, never plaintext JSON", len(gotBody))
	}
}

func TestWebPushSenderReportsGoneOn404And410(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusGone} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		s := newTestSender(t)
		err := s.Send(context.Background(), browserSubscription(t, srv.URL), Payload{Title: "t"})
		srv.Close()
		if !errors.Is(err, ErrSubscriptionGone) {
			t.Errorf("%d: err = %v, want ErrSubscriptionGone", code, err)
		}
	}
}

func TestWebPushSenderReportsOtherFailuresAsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	s := newTestSender(t)
	err := s.Send(context.Background(), browserSubscription(t, srv.URL), Payload{Title: "t"})
	if err == nil || errors.Is(err, ErrSubscriptionGone) {
		t.Errorf("429: err = %v, want a non-gone error", err)
	}
}

func TestNewWebPushSenderRequiresBothKeys(t *testing.T) {
	if _, err := NewWebPushSender("", "priv", "mailto:x@example.com"); err == nil {
		t.Error("missing public key accepted")
	}
	if _, err := NewWebPushSender("pub", "", "mailto:x@example.com"); err == nil {
		t.Error("missing private key accepted")
	}
}
