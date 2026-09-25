package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

func TestLookalikeOriginsGetNoCORSAndA403Preflight(t *testing.T) {
	for _, origin := range []string{
		"null",                             // opaque origin (sandboxed iframe, file://)
		"https://app.example.com.evil.com", // suffix
		"https://evilapp.example.com",      // prefix
		"http://app.example.com",           // scheme downgrade
		"https://app.example.com:443",      // explicit default port
		"https://app.example.com/",         // trailing slash
		"HTTPS://APP.EXAMPLE.COM",          // not what a browser sends; must not match either
	} {
		t.Run(origin, func(t *testing.T) {
			r := newCORSRouter(t, app)
			pre := do(t, r, http.MethodOptions, "/api/v1/quests/daily", map[string]string{"Origin": origin, "Access-Control-Request-Method": "GET"})
			if pre.Code != http.StatusForbidden || pre.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("preflight from %q: status %d, allow-origin %q; want 403 and none", origin, pre.Code, pre.Header().Get("Access-Control-Allow-Origin"))
			}
			act := do(t, r, http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": origin})
			if act.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("actual request from %q carried Allow-Origin %q", origin, act.Header().Get("Access-Control-Allow-Origin"))
			}
		})
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
		{"https://app.example.com:443", nil, true},          // browsers omit the default port from Origin
		{"http://localhost:80", nil, true},
		{"https://*.up.railway.app", nil, true},             // no wildcards: list each preview origin
		{"https://app.example.com.", nil, true},             // trailing-dot FQDN
		{"https://app.example.com:8443", []string{"https://app.example.com:8443"}, false},
		{"http://localhost:443", []string{"http://localhost:443"}, false}, // 443 is not http's default
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

func TestParseOriginsSaysWhy(t *testing.T) {
	for _, tc := range []struct {
		in     string
		substr string
	}{
		{"https://app.example.com:443", "default port"},
		{"https://*.up.railway.app", "wildcard"},
		{"https://app.example.com.", "trailing dot"},
	} {
		_, err := ParseOrigins(tc.in)
		if err == nil {
			t.Fatalf("ParseOrigins(%q): want error, got nil", tc.in)
		}
		if !errors.Is(err, ErrBadOrigin) {
			t.Errorf("ParseOrigins(%q): err = %v, want errors.Is(err, ErrBadOrigin)", tc.in, err)
		}
		if !strings.Contains(err.Error(), tc.substr) {
			t.Errorf("ParseOrigins(%q): err = %q, want it to contain %q", tc.in, err.Error(), tc.substr)
		}
	}
}
