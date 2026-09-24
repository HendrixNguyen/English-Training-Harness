package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newGuardedRouter(iss *TokenIssuer, sess SessionStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/guarded", Require(iss, sess), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": UserID(c)})
	})
	return r
}

func get(t *testing.T, r *gin.Engine, header string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAllowsAValidTokenAndExposesTheUserID(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeSessions()
	if err := sess.Put(context.Background(), "user-1", tok, TokenTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}

	w := get(t, newGuardedRouter(iss, sess), "Bearer "+tok)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if want := `"user_id":"user-1"`; !strings.Contains(w.Body.String(), want) {
		t.Errorf("body = %s, want %s", w.Body.String(), want)
	}
}

func TestRequireRejectsAMissingOrMalformedHeader(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	r := newGuardedRouter(iss, newFakeSessions())

	for _, header := range []string{"", "Bearer", "Basic abc", "Bearer not-a-jwt", "Bearer a.b.c"} {
		if w := get(t, r, header); w.Code != http.StatusUnauthorized {
			t.Errorf("header %q → status %d, want 401", header, w.Code)
		}
	}
}

func TestRequireRejectsAnExpiredToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok, err := NewTokenIssuer("secret", func() time.Time { return now }).Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	sess := newFakeSessions()
	_ = sess.Put(context.Background(), "user-1", tok, TokenTTL)

	later := NewTokenIssuer("secret", func() time.Time { return now.Add(TokenTTL + time.Minute) })

	if w := get(t, newGuardedRouter(later, sess), "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for an expired token", w.Code)
	}
}

func TestRequireRejectsARevokedSession(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	tok, _ := iss.Issue("user-1")
	sess := newFakeSessions()
	_ = sess.Put(context.Background(), "user-1", tok, TokenTTL)
	// Revocation = delete the Redis key while the JWT is still within exp.
	_ = sess.Delete(context.Background(), "user-1")

	if w := get(t, newGuardedRouter(iss, sess), "Bearer "+tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a revoked session", w.Code)
	}
}

func TestRequireRejectsASupersededToken(t *testing.T) {
	// Signing in again replaces the stored token; the old one must stop working.
	iss := NewTokenIssuer("secret", time.Now)
	old, _ := iss.Issue("user-1")
	sess := newFakeSessions()
	_ = sess.Put(context.Background(), "user-1", "a-newer-token", TokenTTL)

	if w := get(t, newGuardedRouter(iss, sess), "Bearer "+old); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a superseded token", w.Code)
	}
}

func TestRequireAcceptsACaseInsensitiveBearerScheme(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	tok, _ := iss.Issue("user-1")
	sess := newFakeSessions()
	_ = sess.Put(context.Background(), "user-1", tok, TokenTTL)
	r := newGuardedRouter(iss, sess)

	for _, header := range []string{"bearer " + tok, "BEARER " + tok, "Bearer " + tok} {
		if w := get(t, r, header); w.Code != http.StatusOK {
			t.Errorf("header %q → status %d, want 200 (RFC 7235: the scheme is case-insensitive)", header[:6], w.Code)
		}
	}
}
