package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeRenewSessions is the in-memory SessionStore used by the renewal tests.
// It tracks Put calls separately from session_test.go's fakeSessions (whose
// single err field would also break the Get that Require does just before
// maybeRenew) so a failing Put can be exercised on its own.
type fakeRenewSessions struct {
	vals     map[string]string
	putErr   error
	putCalls int
}

func newFakeRenewSessions() *fakeRenewSessions {
	return &fakeRenewSessions{vals: map[string]string{}}
}

func (f *fakeRenewSessions) Put(_ context.Context, userID, token string, _ time.Duration) error {
	f.putCalls++
	if f.putErr != nil {
		return f.putErr
	}
	f.vals[userID] = token
	return nil
}

func (f *fakeRenewSessions) Get(_ context.Context, userID string) (string, error) {
	v, ok := f.vals[userID]
	if !ok {
		return "", ErrNoSession
	}
	return v, nil
}

func (f *fakeRenewSessions) Delete(_ context.Context, userID string) error {
	delete(f.vals, userID)
	return nil
}

var _ SessionStore = (*fakeRenewSessions)(nil)

func TestRequireRenewsBelowHalfLife(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	now := t0
	iss := NewTokenIssuer("secret", func() time.Time { return now })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeRenewSessions()
	if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	sess.putCalls = 0 // reset after the setup Put above
	r := newGuardedRouter(iss, sess)

	now = t0.Add(TokenTTL/2 + time.Second) // one second under half-life remaining
	w := get(t, r, "Bearer "+tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	newTok := w.Header().Get(HeaderSessionToken)
	if newTok == "" || newTok == tok {
		t.Fatalf("X-Session-Token = %q, want a fresh, different token", newTok)
	}
	if got := w.Header().Get(HeaderSessionExpiresIn); got != "86400" {
		t.Fatalf("X-Session-Expires-In = %q, want %q", got, "86400")
	}
	if sess.putCalls != 1 {
		t.Fatalf("putCalls = %d, want 1", sess.putCalls)
	}
	if got := sess.vals["user-1"]; got != newTok {
		t.Fatalf("stored token = %q, want the new token %q", got, newTok)
	}

	// A second request with the new token, still below half-life: no further
	// renewal.
	now = now.Add(time.Second)
	w2 := get(t, r, "Bearer "+newTok)
	if w2.Code != http.StatusOK {
		t.Fatalf("second request status = %d, want 200", w2.Code)
	}
	if got := w2.Header().Get(HeaderSessionToken); got != "" {
		t.Fatalf("second request X-Session-Token = %q, want empty", got)
	}
	if sess.putCalls != 1 {
		t.Fatalf("putCalls after second request = %d, want 1", sess.putCalls)
	}
}

func TestRequireDoesNotRenewAtOrAboveHalfLife(t *testing.T) {
	cases := map[string]time.Duration{
		"above half":   TokenTTL/2 - time.Second,
		"exactly half": TokenTTL / 2,
	}
	for name, delta := range cases {
		t.Run(name, func(t *testing.T) {
			t0 := time.Unix(1_800_000_000, 0)
			now := t0
			iss := NewTokenIssuer("secret", func() time.Time { return now })
			tok, err := iss.Issue("user-1")
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			sess := newFakeRenewSessions()
			if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
				t.Fatalf("Put: %v", err)
			}
			sess.putCalls = 0
			r := newGuardedRouter(iss, sess)

			now = t0.Add(delta)
			w := get(t, r, "Bearer "+tok)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if got := w.Header().Get(HeaderSessionToken); got != "" {
				t.Fatalf("X-Session-Token = %q, want empty (not below half-life)", got)
			}
			if sess.putCalls != 0 {
				t.Fatalf("putCalls = %d, want 0", sess.putCalls)
			}
		})
	}
}

func TestRequireRejectsTheOldTokenAfterRenewal(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	now := t0
	iss := NewTokenIssuer("secret", func() time.Time { return now })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeRenewSessions()
	if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	r := newGuardedRouter(iss, sess)

	now = t0.Add(TokenTTL/2 + time.Second)
	if w := get(t, r, "Bearer "+tok); w.Code != http.StatusOK {
		t.Fatalf("renewal request status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	now = now.Add(time.Second)
	if w := get(t, r, "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("old token after renewal: status = %d, want 401", w.Code)
	}
}

func TestRequireDoesNotRenewARevokedSession(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeRenewSessions() // nothing stored: a revoked/never-signed-in session
	r := newGuardedRouter(iss, sess)

	w := get(t, r, "Bearer "+tok)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if sess.putCalls != 0 {
		t.Fatalf("putCalls = %d, want 0", sess.putCalls)
	}
}

func TestRequireDoesNotRenewAnExpiredToken(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	iss := NewTokenIssuer("secret", func() time.Time { return t0 })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeRenewSessions()
	if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	sess.putCalls = 0

	later := NewTokenIssuer("secret", func() time.Time { return t0.Add(TokenTTL + time.Second) })
	r := newGuardedRouter(later, sess)

	w := get(t, r, "Bearer "+tok)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if sess.putCalls != 0 {
		t.Fatalf("putCalls = %d, want 0", sess.putCalls)
	}
}

func TestRequireServesWhenRenewalPutFails(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	now := t0
	iss := NewTokenIssuer("secret", func() time.Time { return now })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeRenewSessions()
	if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	sess.putCalls = 0
	sess.putErr = errors.New("redis: connection refused")
	r := newGuardedRouter(iss, sess)

	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	now = t0.Add(TokenTTL/2 + time.Second)
	w := get(t, r, "Bearer "+tok)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get(HeaderSessionToken); got != "" {
		t.Fatalf("X-Session-Token = %q, want empty when the store Put fails", got)
	}
	if sess.putCalls != 1 {
		t.Fatalf("putCalls = %d, want 1", sess.putCalls)
	}
	if logBuf.Len() == 0 {
		t.Fatal("expected a log line for the failed renewal, got none")
	}
	if strings.Count(strings.TrimRight(logBuf.String(), "\n"), "\n") != 0 {
		t.Fatalf("expected exactly one log line, got: %q", logBuf.String())
	}
}
