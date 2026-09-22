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

func TestHandlerReturnsTokenAndUser(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	svc := NewService(ex, newFakeRepo(), newFakeSessions(),
		NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) }))

	w := postJSON(t, newAuthRouter(svc), `{"code":"the-code","redirect_uri":"https://app/cb"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var got struct {
		Token string `json:"token"`
		User  struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			FullName    string `json:"full_name"`
			CEFRCurrent string `json:"cefr_current"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding body: %v (%s)", err, w.Body.String())
	}
	if got.Token == "" {
		t.Error("token is empty")
	}
	if got.User.ID != "id-google-1" || got.User.Email != "a@example.com" || got.User.CEFRCurrent != "A1" {
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
