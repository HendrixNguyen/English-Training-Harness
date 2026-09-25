package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// errGoogleRejected wraps ErrGoogleRejected the way the Google leg does (see
// google.go), so this file's existing 401 test still exercises the real
// mapping instead of a plain unrelated error.
var errGoogleRejected = fmt.Errorf("%w: 400 invalid_grant", ErrGoogleRejected)

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

func TestHandlerMapsEachFailureToItsStatus(t *testing.T) {
	cases := []struct {
		name       string
		ex         *fakeExchanger
		repo       *fakeRepo
		sess       *fakeSessions
		wantStatus int
		wantBody   string
	}{
		{
			name:       "google rejected",
			ex:         &fakeExchanger{err: fmt.Errorf("%w: 400", ErrGoogleRejected)},
			repo:       newFakeRepo(),
			sess:       newFakeSessions(),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"error":"google_auth_failed"`,
		},
		{
			name:       "email taken",
			ex:         &fakeExchanger{token: GoogleToken{AccessToken: "at"}, profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"}},
			repo:       &fakeRepo{err: ErrEmailTaken},
			sess:       newFakeSessions(),
			wantStatus: http.StatusConflict,
			wantBody:   `"error":"email_in_use"`,
		},
		{
			name:       "session store unavailable",
			ex:         &fakeExchanger{token: GoogleToken{AccessToken: "at"}, profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"}},
			repo:       newFakeRepo(),
			sess:       &fakeSessions{vals: map[string]string{}, ttls: map[string]time.Duration{}, err: fmt.Errorf("%w: dial", ErrSessionStoreUnavailable)},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `"error":"unavailable"`,
		},
		{
			name:       "repo transport failure",
			ex:         &fakeExchanger{token: GoogleToken{AccessToken: "at"}, profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"}},
			repo:       &fakeRepo{err: errors.New("pg down")},
			sess:       newFakeSessions(),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `"error":"internal_error"`,
		},
		{
			name:       "exchange transport failure",
			ex:         &fakeExchanger{err: errors.New("dial tcp: i/o timeout")},
			repo:       newFakeRepo(),
			sess:       newFakeSessions(),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `"error":"internal_error"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.ex, tc.repo, tc.sess, NewTokenIssuer("s", time.Now))
			w := postJSON(t, newAuthRouter(svc), `{"code":"c","redirect_uri":"https://app/cb"}`)
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body = %s, want it to contain %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestHandlerLogsTheFailureWithoutTheTokens(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at-secret", RefreshToken: "rt-secret"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"},
	}
	repo := &fakeRepo{err: errors.New("pg down")}
	svc := NewService(ex, repo, newFakeSessions(), NewTokenIssuer("super-secret-jwt-key", time.Now))

	w := postJSON(t, newAuthRouter(svc), `{"code":"c","redirect_uri":"https://app/cb"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", w.Code, w.Body.String())
	}

	got := buf.String()
	if !strings.Contains(got, "auth: sign-in failed") {
		t.Errorf("log = %q, want it to contain %q", got, "auth: sign-in failed")
	}
	if !strings.Contains(got, "google-1") {
		t.Errorf("log = %q, want it to name the google id %q", got, "google-1")
	}
	for _, secret := range []string{"at-secret", "rt-secret", "super-secret-jwt-key"} {
		if strings.Contains(got, secret) {
			t.Errorf("log = %q, must not contain the secret %q", got, secret)
		}
	}
}
