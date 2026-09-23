package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var errGoogleRejected = errors.New("google rejected the code")

func newAuthRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/google", Handler(svc))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlerReturnsTheSpecSignInBody(t *testing.T) {
	clock := func() time.Time { return time.Unix(1_800_000_000, 0) }
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	svc := NewService(ex, newFakeRepo(), newFakeSessions(), NewTokenIssuer("secret", clock))

	w := postJSON(t, newAuthRouter(svc), `{"code":"the-code","redirect_uri":"https://app/cb"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Backend spec §6.1: exactly {access_token, token_type, expires_in, user}.
	// Decode to a map first so the assertion is about the wire keys, not a Go struct.
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &keys); err != nil {
		t.Fatalf("decoding body: %v (%s)", err, w.Body.String())
	}
	if _, ok := keys["token"]; ok {
		t.Errorf("body has a \"token\" key; spec §6.1 names it access_token: %s", w.Body.String())
	}
	for _, k := range []string{"access_token", "token_type", "expires_in", "user"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("body is missing %q: %s", k, w.Body.String())
		}
	}
	if len(keys) != 4 {
		t.Errorf("body has %d top-level keys, want 4: %s", len(keys), w.Body.String())
	}

	var got struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		User        struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			FullName    string `json:"full_name"`
			CEFRCurrent string `json:"cefr_current"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding body: %v (%s)", err, w.Body.String())
	}
	if got.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want %q", got.TokenType, "Bearer")
	}
	if got.ExpiresIn != 86400 {
		t.Errorf("expires_in = %d, want 86400 (spec §7: 24-hour session)", got.ExpiresIn)
	}
	// access_token is the session JWT: it must verify under the same secret and
	// name the upserted user.
	sub, err := NewTokenIssuer("secret", clock).Verify(got.AccessToken)
	if err != nil {
		t.Fatalf("access_token does not verify as the session JWT: %v", err)
	}
	if sub != "id-google-1" {
		t.Errorf("access_token subject = %q, want id-google-1", sub)
	}
	if got.User.ID != "id-google-1" || got.User.Email != "a@example.com" ||
		got.User.FullName != "A Person" || got.User.CEFRCurrent != "A1" {
		t.Errorf("user = %+v", got.User)
	}
}

func TestHandlerRejectsAMissingCode(t *testing.T) {
	svc := NewService(&fakeExchanger{}, newFakeRepo(), newFakeSessions(), NewTokenIssuer("s", time.Now))

	for _, body := range []string{`{}`, `{"code":"","redirect_uri":"u"}`, `{"code":"c"}`, `not json`} {
		w := postJSON(t, newAuthRouter(svc), body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q → status %d, want 400", body, w.Code)
		}
	}
}

func TestHandlerReturns401WhenGoogleRejectsTheCode(t *testing.T) {
	ex := &fakeExchanger{err: errGoogleRejected}
	svc := NewService(ex, newFakeRepo(), newFakeSessions(), NewTokenIssuer("s", time.Now))

	w := postJSON(t, newAuthRouter(svc), `{"code":"bad","redirect_uri":"https://app/cb"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", w.Code, w.Body.String())
	}
}
