package notify

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// These tests prove the RFC 8291/8292 request shape, not the dial guard, so
// they run over an unguarded client that trusts the httptest CA. The guard
// has its own tests in endpoint_test.go and below.
//
// Send's URL layer refuses IP literals, so the server is addressed as
// "localhost:<port>"; httptest's certificate covers 127.0.0.1, ::1,
// example.com and *.example.com — not localhost (net/http/internal/testcert)
// — hence the ServerName override.
func testTLSClient(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	tr := srv.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.ServerName = "example.com"
	return &http.Client{Transport: tr, Timeout: 5 * time.Second}
}

func localhostURL(t *testing.T, srv *httptest.Server, path string) string {
	t.Helper()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	if err != nil {
		t.Fatal(err)
	}
	return "https://localhost:" + port + path
}

func TestWebPushSenderPostsAnEncryptedVAPIDSignedRequest(t *testing.T) {
	var gotAuth, gotEncoding, gotTTL string
	var gotBody []byte
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotEncoding = r.Header.Get("Content-Encoding")
		gotTTL = r.Header.Get("TTL")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	s := newTestSender(t)
	s.HTTPClient = testTLSClient(t, srv)
	err := s.Send(context.Background(), browserSubscription(t, localhostURL(t, srv, "/push/abc")), Payload{Title: "t", Body: "b", URL: "/"})
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
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		s := newTestSender(t)
		s.HTTPClient = testTLSClient(t, srv)
		err := s.Send(context.Background(), browserSubscription(t, localhostURL(t, srv, "/")), Payload{Title: "t"})
		srv.Close()
		if !errors.Is(err, ErrSubscriptionGone) {
			t.Errorf("%d: err = %v, want ErrSubscriptionGone", code, err)
		}
	}
}

func TestWebPushSenderReportsOtherFailuresAsErrors(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	s := newTestSender(t)
	s.HTTPClient = testTLSClient(t, srv)
	err := s.Send(context.Background(), browserSubscription(t, localhostURL(t, srv, "/")), Payload{Title: "t"})
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

// Send-level: the sender built by NewWebPushSender must refuse before dialling.
func TestWebPushSenderRefusesAForbiddenEndpointBeforeDialling(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))

	s := newTestSender(t) // default HTTPClient: the guarded one
	// `want` pins WHICH layer answered: the URL-layer rows would also be
	// refused by the dial guard (loopback), so the message text is what
	// proves Send validated before dialling.
	for name, tc := range map[string]struct{ endpoint, want string }{
		"ip literal, URL layer": {srv.URL + "/x", "not a public address"}, // https://127.0.0.1:port
		"plain http, URL layer": {"http://127.0.0.1:" + port + "/x", "want https"},
		"dns name, dial layer":  {"https://localhost:" + port + "/x", "refusing to dial"},
	} {
		err := s.Send(context.Background(), browserSubscription(t, tc.endpoint), Payload{Title: "t"})
		if !errors.Is(err, ErrForbiddenEndpoint) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want ErrForbiddenEndpoint containing %q", name, err, tc.want)
		}
		if errors.Is(err, ErrSubscriptionGone) {
			t.Errorf("%s: forbidden must not masquerade as gone", name)
		}
	}
	if hits.Load() != 0 {
		t.Errorf("push server handled %d request(s); nothing may be sent to a forbidden endpoint", hits.Load())
	}
}

func TestWebPushSenderReturnsARedirectAsAFailureNotAFollow(t *testing.T) {
	var redirected atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/redirected", http.StatusFound) })
	mux.HandleFunc("/redirected", func(w http.ResponseWriter, _ *http.Request) { redirected.Add(1); w.WriteHeader(http.StatusCreated) })
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	s := newTestSender(t)
	// The URL layer must pass the endpoint, so dial by hostname, not srv.URL's
	// IP literal. httptest's certificate covers 127.0.0.1, ::1, example.com
	// and *.example.com — NOT localhost (checked: net/http/internal/testcert)
	// — so verify it under the example.com name. Keep CheckRedirect; swap
	// only the transport (unguarded, trusts the test CA).
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	tr := srv.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.ServerName = "example.com"
	client := newPushHTTPClient()
	client.Transport = tr
	s.HTTPClient = client
	sub := browserSubscription(t, "https://localhost:"+port+"/push")

	err := s.Send(context.Background(), sub, Payload{Title: "t"})
	if err == nil || errors.Is(err, ErrSubscriptionGone) || !strings.Contains(err.Error(), "302") {
		t.Errorf("err = %v, want a non-gone failure mentioning 302", err)
	}
	if redirected.Load() != 0 {
		t.Errorf("/redirected was hit %d time(s)", redirected.Load())
	}
}
