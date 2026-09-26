package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

// newCORSRouter mirrors cmd/api: the middleware is global, the routes are
// GET/POST only, and nothing registers OPTIONS.
func newCORSRouter(t *testing.T, origins ...string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.GET("/api/v1/onboarding/quiz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func do(t *testing.T, r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const app = "https://app.example.com"

func TestPreflightFromAnAllowedOriginIs204WithTheAllowHeaders(t *testing.T) {
	w := do(t, newCORSRouter(t, app), http.MethodOptions, "/api/v1/onboarding/quiz", map[string]string{
		"Origin":                         app,
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "authorization",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", w.Code, w.Body.String())
	}
	for k, want := range map[string]string{
		"Access-Control-Allow-Origin":  app,
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Authorization, Content-Type",
		"Access-Control-Max-Age":       "600",
		"Vary":                         "Origin",
	} {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("Allow-Credentials must not be set: the session is a bearer header, not a cookie")
	}
}

func TestPreflightForAPathWithNoRouteAtAllIs204Not404(t *testing.T) {
	// cmd/api registers POST /api/v1/quests/progress; this router does not
	// register it at all — the NoRoute chain must still reach the middleware.
	w := do(t, newCORSRouter(t, app), http.MethodOptions, "/api/v1/quests/progress", map[string]string{
		"Origin": app, "Access-Control-Request-Method": "POST",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (was 404 before the middleware existed)", w.Code)
	}
}

func TestAnActualRequestFromAnAllowedOriginCarriesTheAllowOriginHeader(t *testing.T) {
	w := do(t, newCORSRouter(t, app), http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": app})
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != app || w.Header().Get("Vary") != "Origin" {
		t.Errorf("status %d, ACAO %q, Vary %q; want 200, %q, Origin", w.Code, w.Header().Get("Access-Control-Allow-Origin"), w.Header().Get("Vary"), app)
	}
}

func TestAnActualRequestFromAnotherOriginPassesWithoutCORSHeaders(t *testing.T) {
	w := do(t, newCORSRouter(t, app), http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": "https://evil.example"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — non-browser clients are unaffected; the browser refuses to read it", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("ACAO = %q for a disallowed origin, want none", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestPreflightFromAnotherOriginIs403(t *testing.T) {
	w := do(t, newCORSRouter(t, app), http.MethodOptions, "/api/v1/onboarding/quiz", map[string]string{
		"Origin": "https://evil.example", "Access-Control-Request-Method": "GET",
	})
	if w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("status %d, ACAO %q; want 403 and no ACAO", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRequestsWithoutAnOriginAreUntouched(t *testing.T) {
	w := do(t, newCORSRouter(t, app), http.MethodGet, "/api/v1/onboarding/quiz", nil)
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Vary") != "" {
		t.Errorf("same-origin/curl request got status %d, ACAO %q, Vary %q; want 200 and neither header", w.Code, w.Header().Get("Access-Control-Allow-Origin"), w.Header().Get("Vary"))
	}
}

func TestTheEchoedOriginIsTheMatchedOneNeverAWildcardOrTheList(t *testing.T) {
	const staging = "https://staging.example.com"
	w := do(t, newCORSRouter(t, app, staging), http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": staging})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != staging {
		t.Errorf("ACAO = %q, want exactly %q", got, staging)
	}
}

func TestCORSExposesTheSessionHeaders(t *testing.T) {
	const want = "X-Session-Token, X-Session-Expires-In"

	get := do(t, newCORSRouter(t, app), http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": app})
	if got := get.Header().Get("Access-Control-Expose-Headers"); got != want {
		t.Errorf("allowed origin, actual request: Expose-Headers = %q, want %q", got, want)
	}

	preflight := do(t, newCORSRouter(t, app), http.MethodOptions, "/api/v1/onboarding/quiz", map[string]string{
		"Origin": app, "Access-Control-Request-Method": "GET",
	})
	if got := preflight.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("preflight: Expose-Headers = %q, want none", got)
	}

	foreign := do(t, newCORSRouter(t, app), http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": "https://evil.example"})
	if got := foreign.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("foreign origin: Expose-Headers = %q, want none", got)
	}
}

func TestParseOrigins(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
		err  bool
	}{
		{"https://app.example.com", []string{"https://app.example.com"}, false},
		{" https://app.example.com/ , http://localhost:3000", []string{"https://app.example.com", "http://localhost:3000"}, false},
		{"HTTPS://App.Example.com", []string{"https://app.example.com"}, false},
		{"app.example.com", nil, true},
		{"https://app.example.com/login", nil, true},
		{"https://app.example.com/?x=1", nil, true},
		{"https://user@app.example.com", nil, true},
		{"ftp://app.example.com", nil, true},
		{"", nil, true},
		{" , ", nil, true},
	} {
		got, err := ParseOrigins(tc.in)
		if (err != nil) != tc.err {
			t.Errorf("ParseOrigins(%q) err = %v, want error %t", tc.in, err, tc.err)
			continue
		}
		if !tc.err && !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParseOrigins(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
