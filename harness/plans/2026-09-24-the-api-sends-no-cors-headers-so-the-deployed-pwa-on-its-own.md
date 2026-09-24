---
idea: harness/ideas/_inbox/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md
status: done
priority: high
merged: false
branch: harness/2026-09-24-high-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own
worktree: .worktrees/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/17"
---
# API edge: a CORS allow-list for the PWA's origin, bounded request bodies, and a `/healthz` that names no secrets — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` → Task 3, Task 5
- `harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md` (folds the rejected `healthz-shares-one-2s-deadline-across-two-sequential-pings.md`) → Task 4

**Goal:** The deployed PWA on its own Railway origin can call every `/api/v1/*` endpoint (preflights answered `204`, the allow-listed origin echoed back), no bearer token can stream an unbounded JSON body into memory, and the unauthenticated `/healthz` stops publishing the production database identity when a dependency is down.

**Why now (`priority: high`):** Spec §8 / backend spec §9 deploy the Nuxt PWA and the Go API as two Railway services on two origins. Every guarded call carries `Authorization`, so the browser preflights, and the API answers `OPTIONS` with `404` (Gin registers no OPTIONS routes) and a plain `GET` with no `Access-Control-Allow-Origin`. The shipped PWA gets zero data from the API in production. Every frontend idea in the queue is inert until this lands (2026-09-24 ideation run, *Notes*).

**Root cause (from the idea's `## Evaluation`, re-read on this branch):** `backend/cmd/api/main.go` builds `r := gin.Default()` and mounts `/healthz` and the `/api/v1` group with no middleware in between; `grep -rni 'cors\|Access-Control' backend/` is empty; `backend/go.mod` has no CORS library. The body-bound gap is the same shape: five handlers call `c.ShouldBindJSON` straight against an unbounded `c.Request.Body` (`auth`, `quests`, `pet`, `notify`, `onboarding`) and nothing inbound uses `http.MaxBytesReader`. `/healthz` copies `err.Error()` into the body (`internal/health/health.go`), and pgx's dial error embeds `user=… database=… host:port`.

**Architecture:** One new package, `backend/internal/middleware`, holds the two request-edge guards as plain `gin.HandlerFunc`s with no knowledge of routes or users: `CORS(allowed []string)` and `BodyLimit(max int64)`, plus `ParseOrigins` for the `FRONTEND_ORIGIN` value. `config.Load` only carries the raw string (with the Nuxt dev-server default) so `config.go`'s footprint stays at one field — the approved cmd/api shutdown plan and the secrets plan also edit that file. `main.go` gains exactly three lines (`ParseOrigins` + `r.Use(CORS)` before `/healthz`, `v1.Use(BodyLimit)` right after the group is created). Hand-rolled, not `gin-contrib/cors`: the required behaviour is ~60 lines, fully specified by the idea, and adds no dependency.

**Why a global `r.Use` works for preflights:** Gin has no OPTIONS routes, so a preflight falls to the NoRoute chain — and `engine.Use` rebuilds that chain (`rebuild404Handlers`) from the global middleware, so the CORS handler runs for `OPTIONS /api/v1/anything` and can `AbortWithStatus(204)` before the 404 is written. `BodyLimit` goes on the `/api/v1` group instead: `/healthz` has no body, and a group's middleware is copied into every route registered *after* the `Use` call — so the `Use` line must precede the first `v1.POST`.

**Tech stack:** Go 1.25, Gin v1.12, stdlib `net/http`, `net/url`. No new dependencies.

**⚠ Merge-order / conflict note — read before `git worktree add`:** `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (approved) edits `main.go` (engine construction, `gin.SetMode`, `SetTrustedProxies`, the serve loop), `config.go` (`GinMode`) and `.env.example` (an application section). This plan's edits to those three files are written as *region edits*: insert after the engine-construction line whatever it is on your base, append one config field, append one `.env.example` block. If `origin/main` has moved when you start or before you push, `git fetch origin main && git merge origin/main --no-edit` and keep both sides' additions. Suggested daily merge order: shutdown plan → this plan → the secrets plan.

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n` and `go test -timeout`; bound curls with `--max-time`.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/config/config.go` | Add `DefaultFrontendOrigin` const and `FrontendOrigin string` field; read `FRONTEND_ORIGIN` with the default |
| `backend/internal/config/config_test.go` | Append `TestLoadDefaultsFrontendOriginToTheNuxtDevServer` |
| `backend/internal/middleware/cors.go` | **New**: `ErrBadOrigin`, `ParseOrigins`, `CORS` |
| `backend/internal/middleware/cors_test.go` | **New**: preflight/actual/disallowed/no-origin/exact-echo tests, `ParseOrigins` table |
| `backend/internal/middleware/bodylimit.go` | **New**: `MaxBodyBytes`, `BodyLimit` |
| `backend/internal/middleware/bodylimit_test.go` | **New**: over-limit rejected before decode, under-limit decoded, production constant |
| `backend/internal/health/health.go` | `PingTimeout` var; per-dependency budget; fixed `"unavailable"` markers; `log.Printf` of the driver error |
| `backend/internal/health/health_test.go` | Update the two down tests; add both-down, own-budget, logs-server-side |
| `backend/cmd/api/main.go` | `ParseOrigins` + `r.Use(middleware.CORS(...))` before `/healthz`; `v1.Use(middleware.BodyLimit(...))` after `r.Group("/api/v1")` |
| `backend/.env.example` | `FRONTEND_ORIGIN` block |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §9 step 2: add `FRONTEND\_ORIGIN` |
| `harness/CODEMAP.md` | New `**middleware**` and `**health**` bullets; `config` env mention |

---

## Tasks

### Task 1: `config.Load` carries `FRONTEND_ORIGIN` (default: the Nuxt dev server)

**Files:**
- Modify: `backend/internal/config/config_test.go` (append)
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Write the failing test** (append to `config_test.go`; the 32-byte `JWT_SECRET` keeps this test valid whether or not the secrets plan — which raises the minimum — has landed)

```go
func TestLoadDefaultsFrontendOriginToTheNuxtDevServer(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.FrontendOrigin != DefaultFrontendOrigin || DefaultFrontendOrigin != "http://localhost:3000" {
		t.Errorf("FrontendOrigin = %q, want the Nuxt dev server default %q", cfg.FrontendOrigin, "http://localhost:3000")
	}

	t.Setenv("FRONTEND_ORIGIN", "https://app.example.com, https://staging.example.com")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FrontendOrigin != "https://app.example.com, https://staging.example.com" {
		t.Errorf("FrontendOrigin = %q, want the raw value (middleware.ParseOrigins validates it)", cfg.FrontendOrigin)
	}
}
```

If the secrets plan has already landed on your base, its `ENCRYPTION_SECRET_KEY` is also required — add `t.Setenv("ENCRYPTION_SECRET_KEY", <64 hex chars>)` the way its tests do.

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/config/ -run TestLoadDefaultsFrontendOrigin -count=1`
Expected: compile error — `cfg.FrontendOrigin undefined` / `undefined: DefaultFrontendOrigin`.

- [ ] **Step 3: Add the field and the read** (`config.go`)

Add after `DefaultVAPIDSubject`:

```go
// DefaultFrontendOrigin is the Nuxt dev server. Production sets FRONTEND_ORIGIN
// to the PWA's own Railway origin (backend spec §9): the PWA and the API are
// two services on two origins, so every browser call is cross-origin.
const DefaultFrontendOrigin = "http://localhost:3000"
```

Add to `Config` (last field):

```go
	// FrontendOrigin is FRONTEND_ORIGIN as given: a comma-separated allow-list
	// of exact scheme://host[:port] origins. middleware.ParseOrigins validates
	// it at wiring time; config only supplies the default.
	FrontendOrigin string
```

Add to `Load`, before `return cfg, nil`:

```go
	if cfg.FrontendOrigin = os.Getenv("FRONTEND_ORIGIN"); cfg.FrontendOrigin == "" {
		cfg.FrontendOrigin = DefaultFrontendOrigin
	}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/config/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "config: read FRONTEND_ORIGIN with the Nuxt dev-server default"
```

### Task 2: `middleware.CORS` — allow-list, preflight 204, exact echo

**Files:**
- Create: `backend/internal/middleware/cors_test.go`
- Create: `backend/internal/middleware/cors.go`

- [ ] **Step 1: Write the failing tests**

```go
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
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/middleware/ -count=1`
Expected: compile error — `undefined: CORS`, `undefined: ParseOrigins`.

- [ ] **Step 3: Implement `cors.go`**

```go
// Package middleware holds the request-edge guards cmd/api mounts once: the
// CORS allow-list for the PWA's origin and the request-body bound. It knows
// nothing about routes, users or tables.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrBadOrigin is ParseOrigins' error for an entry that is not a bare
// scheme://host[:port] origin, or for an empty list.
var ErrBadOrigin = errors.New("middleware: FRONTEND_ORIGIN entries must be http(s)://host[:port] with no path, query, fragment or userinfo")

// ParseOrigins turns the FRONTEND_ORIGIN value into the exact-match allow-list:
// comma-separated, whitespace trimmed, one trailing slash dropped, scheme and
// host lower-cased (browsers send Origin lower-cased). Empty entries are
// skipped; an empty result is an error — a CORS middleware with no origins
// would silently block the PWA, which is the bug this package exists to fix.
func ParseOrigins(raw string) ([]string, error) {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSuffix(strings.TrimSpace(part), "/")
		if p == "" {
			continue
		}
		u, err := url.Parse(p)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https" && strings.ToLower(u.Scheme) != "http" && strings.ToLower(u.Scheme) != "https") ||
			u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.Opaque != "" {
			return nil, fmt.Errorf("%w: %q", ErrBadOrigin, strings.TrimSpace(part))
		}
		out = append(out, strings.ToLower(u.Scheme)+"://"+strings.ToLower(u.Host))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no origins in %q", ErrBadOrigin, raw)
	}
	return out, nil
}

// CORS answers cross-origin browser requests from the allow-listed origins
// (spec §8 deploys the PWA and the API as two Railway services on two
// origins). Mount it globally with engine.Use before any route: Gin rebuilds
// its NoRoute chain from the global middleware, so a preflight for a path
// that has only a GET route (Gin registers no OPTIONS routes) still reaches
// this handler and is answered 204, not 404.
//
// Allowed origin: echo it in Access-Control-Allow-Origin (never "*"), add
// Vary: Origin; a preflight (OPTIONS + Access-Control-Request-Method) also
// gets Allow-Methods / Allow-Headers / Max-Age and stops here with 204.
// Origin not on the list: a preflight is answered 403; an actual request
// passes through with no CORS headers, which the browser then refuses to
// read, while non-browser clients are unaffected. No Origin header
// (same-origin, curl, the Railway health probe): untouched.
//
// There is deliberately no Access-Control-Allow-Credentials: the session is
// a bearer Authorization header, not a cookie.
func CORS(allowed []string) gin.HandlerFunc {
	allow := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		allow[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		c.Writer.Header().Add("Vary", "Origin")
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""
		if !allow[origin] {
			if preflight {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		if preflight {
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
```

(`url.Parse` lower-cases the scheme itself, so the `strings.ToLower(u.Scheme)` alternatives in the condition are belt-and-braces; keep the condition readable if you simplify it, and keep the `HTTPS://App.Example.com` table row green.)

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/middleware/ -count=1 -v -run 'Preflight|Origin|ParseOrigins'`
Expected: PASS ×8.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/cors.go internal/middleware/cors_test.go
git commit -m "middleware: CORS allow-list from FRONTEND_ORIGIN, preflights answered 204"
```

### Task 3: `middleware.BodyLimit` — 64 KiB on every `/api/v1` body

**Files:**
- Create: `backend/internal/middleware/bodylimit_test.go`
- Create: `backend/internal/middleware/bodylimit.go`

- [ ] **Step 1: Write the failing tests**

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type seen struct {
	answers int
	err     error
}

// newLimitedRouter mirrors cmd/api: the limit sits on the /api/v1 group and
// the handler binds JSON exactly as every real handler does.
func newLimitedRouter(t *testing.T, max int64) (*gin.Engine, *seen) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s := &seen{}
	v1 := r.Group("/api/v1")
	v1.Use(BodyLimit(max))
	v1.POST("/echo", func(c *gin.Context) {
		var body struct {
			Answers []string `json:"answers"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			s.err = err
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		s.answers = len(body.Answers)
		c.JSON(http.StatusOK, gin.H{"n": len(body.Answers)})
	})
	return r, s
}

func post(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/echo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestABodyPastTheLimitIsRejectedBeforeItIsDecoded(t *testing.T) {
	r, s := newLimitedRouter(t, 64)
	body := `{"answers":[` + strings.Repeat(`"A",`, 1000) + `"A"]}` // ~4 KiB against a 64-byte limit

	w := post(t, r, body)

	if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"invalid_request"}` {
		t.Fatalf("status %d body %s; want 400 invalid_request", w.Code, w.Body.String())
	}
	if s.err == nil || !strings.Contains(s.err.Error(), "request body too large") {
		t.Errorf("handler saw %v, want http.MaxBytesReader's \"request body too large\"", s.err)
	}
	if s.answers != 0 {
		t.Errorf("handler decoded %d answers past the limit, want none", s.answers)
	}
}

func TestABodyUnderTheLimitIsDecodedNormally(t *testing.T) {
	r, s := newLimitedRouter(t, MaxBodyBytes)
	w := post(t, r, `{"answers":["A","B"]}`)
	if w.Code != http.StatusOK || s.answers != 2 {
		t.Errorf("status %d, decoded %d; want 200 and 2", w.Code, s.answers)
	}
}

func TestTheProductionLimitIs64KiB(t *testing.T) {
	if MaxBodyBytes != 64<<10 {
		t.Errorf("MaxBodyBytes = %d, want 65536 — the largest §6 body is an assessment or a push subscription, a few KiB", MaxBodyBytes)
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/middleware/ -count=1 -run 'Body|Limit'`
Expected: compile error — `undefined: BodyLimit`, `undefined: MaxBodyBytes`.

- [ ] **Step 3: Implement `bodylimit.go`**

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes bounds every /api/v1 request body. The largest legitimate
// body is an onboarding assessment (ten answers, ~1 KiB) or a push
// subscription (a ≤ 2048-byte endpoint plus two keys); 64 KiB leaves room
// for §6.2's free-form user_answers without letting one bearer token stream
// megabytes into encoding/json's slice growth before validate() ever runs.
const MaxBodyBytes int64 = 64 << 10

// BodyLimit wraps the request body in http.MaxBytesReader, so a body past
// max makes the handler's ShouldBindJSON fail — which every handler already
// answers with 400 invalid_request — instead of being allocated. Mount it on
// the /api/v1 group *before* the routes are registered: Gin copies a group's
// middleware into each route at registration time.
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/middleware/ -count=1 -v`
Expected: PASS ×11 (Task 2's eight plus these three).

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/bodylimit.go internal/middleware/bodylimit_test.go
git commit -m "middleware: bound every /api/v1 request body at 64 KiB"
```

### Task 4: `/healthz` names dependencies, never driver errors; each ping has its own budget

**Files:**
- Modify: `backend/internal/health/health_test.go`
- Modify: `backend/internal/health/health.go`

- [ ] **Step 1: Rewrite the two down tests and add three** (replace `TestHealthzUnavailableWhenPostgresIsDown` and `TestHealthzUnavailableWhenRedisIsDown`; append the rest; add `"bytes"`, `"log"`, `"time"` to the imports)

```go
// driverError is what pgx really says: the connection identity is inside it.
const driverError = "failed to connect to `user=english database=english`: 10.0.0.7:5432 (10.0.0.7): dial error: connection refused"

func TestHealthzUnavailableWhenPostgresIsDownNamesItWithoutTheDriverError(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New(driverError)}, fakePinger{}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"postgres":"unavailable"`) || !strings.Contains(body, `"redis":"ok"`) || !strings.Contains(body, `"status":"unavailable"`) {
		t.Errorf("body = %s, want postgres unavailable, redis ok, status unavailable", body)
	}
	for _, leak := range []string{"user=", "database=", "10.0.0.7", "5432", "dial error"} {
		if strings.Contains(body, leak) {
			t.Errorf("body leaks %q on an unauthenticated route: %s", leak, body)
		}
	}
}

func TestHealthzUnavailableWhenRedisIsDownNamesItWithoutTheDriverError(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{err: errors.New("dial tcp 10.0.0.9:6379: connect: connection refused")}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"redis":"unavailable"`) || strings.Contains(body, "10.0.0.9") || strings.Contains(body, "6379") {
		t.Errorf("body = %s, want the marker and no host/port", body)
	}
}

func TestHealthzNamesBothDependenciesWhenBothAreDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New("no pg")}, fakePinger{err: errors.New("no redis")}))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"postgres":"unavailable"`) || !strings.Contains(w.Body.String(), `"redis":"unavailable"`) {
		t.Errorf("status %d body %s; want 503 with both markers", w.Code, w.Body.String())
	}
}

func TestHealthzLogsTheDriverErrorServerSide(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	do(t, newRouter(fakePinger{err: errors.New(driverError)}, fakePinger{}))

	if !strings.Contains(buf.String(), "user=english database=english") {
		t.Errorf("server log = %q, want the full driver error for the operator", buf.String())
	}
}

// slowPinger never answers: it returns only when its own deadline fires.
type slowPinger struct{}

func (slowPinger) Ping(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

func TestHealthzGivesEachDependencyItsOwnBudget(t *testing.T) {
	old := PingTimeout
	PingTimeout = 30 * time.Millisecond
	t.Cleanup(func() { PingTimeout = old })

	start := time.Now()
	w := do(t, newRouter(slowPinger{}, slowPinger{}))
	elapsed := time.Since(start)

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"postgres":"unavailable"`) || !strings.Contains(w.Body.String(), `"redis":"unavailable"`) {
		t.Errorf("status %d body %s; want 503 with both markers", w.Code, w.Body.String())
	}
	// Two sequential budgets: the second dependency is not starved by the
	// first having spent a shared deadline (the folded healthz-shares-one-
	// 2s-deadline finding), and the probe still finishes promptly.
	if elapsed < 2*PingTimeout || elapsed > time.Second {
		t.Errorf("elapsed = %v, want between %v and 1s", elapsed, 2*PingTimeout)
	}
	if strings.Contains(w.Body.String(), "deadline") {
		t.Errorf("body leaks the context error: %s", w.Body.String())
	}
}
```

(`os` is needed for `os.Stderr`; add it to the imports.)

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/health/ -count=1`
Expected: compile error — `undefined: PingTimeout`; after stubbing it, the postgres/redis tests fail on `"unavailable"` and the leak assertions.

- [ ] **Step 3: Rewrite `health.go`**

```go
// Package health serves GET /healthz: a liveness probe that reports whether
// Postgres and Redis are reachable. The route is unauthenticated (Railway
// probes it), so the body names the failing dependency with a fixed marker
// and never carries the driver's error — pgx embeds `user=… database=…` and
// the host:port in it. The full error goes to the server log instead.
package health

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is anything that can be checked for reachability. store.Postgres and
// store.Redis both satisfy it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingTimeout is each dependency's own budget: two slow dependencies cost
// 2×PingTimeout, and neither can starve the other by spending a shared
// deadline. A var so tests can shrink it.
var PingTimeout = 2 * time.Second

// unavailable is the only thing the body ever says about a failing dependency.
const unavailable = "unavailable"

// Handler returns 200 when both dependencies answer, 503 otherwise, with
// "unavailable" against the dependency that did not.
func Handler(db, cache Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := gin.H{"status": "ok", "postgres": "ok", "redis": "ok"}
		healthy := true

		if err := ping(c.Request.Context(), db); err != nil {
			log.Printf("health: postgres ping failed: %v", err)
			body["postgres"] = unavailable
			healthy = false
		}
		if err := ping(c.Request.Context(), cache); err != nil {
			log.Printf("health: redis ping failed: %v", err)
			body["redis"] = unavailable
			healthy = false
		}
		if !healthy {
			body["status"] = unavailable
			c.JSON(http.StatusServiceUnavailable, body)
			return
		}
		c.JSON(http.StatusOK, body)
	}
}

func ping(parent context.Context, p Pinger) error {
	ctx, cancel := context.WithTimeout(parent, PingTimeout)
	defer cancel()
	return p.Ping(ctx)
}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/health/ -count=1 -v`
Expected: PASS ×6 (`OKWhenBothServicesRespond`, the two `…WithoutTheDriverError`, `BothAreDown`, `LogsTheDriverErrorServerSide`, `OwnBudget`). Then `grep -n 'err.Error()' internal/health/health.go` → no output.

- [ ] **Step 5: Commit**

```bash
git add internal/health/health.go internal/health/health_test.go
git commit -m "health: fixed unavailable markers, driver errors logged not served, per-ping budget"
```

### Task 5: Wire it in `main.go`; `.env.example`, spec §9, CODEMAP; boot proof

**Files:**
- Modify: `backend/cmd/api/main.go` (three insertions — region edits)
- Modify: `backend/.env.example` (append)
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` (§9 step 2)
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1: `main.go` — add the import and the three lines**

Import: `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/middleware"` (alphabetical, between `health` and `notify`).

Immediately **after the engine construction line** (`r := gin.Default()` on today's `main`; after the `gin.SetMode`/`SetTrustedProxies` lines if the shutdown plan has landed) and **before** `r.GET("/healthz", …)`:

```go
	// Spec §8: the PWA is a separate Railway service on its own origin, so
	// every browser call is cross-origin. Global, so preflights for paths with
	// no OPTIONS route reach it (see middleware.CORS).
	origins, err := middleware.ParseOrigins(cfg.FrontendOrigin)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	r.Use(middleware.CORS(origins))
	log.Printf("cors: allowing %v", origins)
```

Immediately **after** `v1 := r.Group("/api/v1")` and **before** `v1.POST("/auth/google", …)`:

```go
	v1.Use(middleware.BodyLimit(middleware.MaxBodyBytes)) // before any route: a group's middleware is copied at registration
```

Run: `go build ./... && go vet ./...` — expected: no output.

- [ ] **Step 2: `.env.example`** — append (or, if the shutdown plan's application section exists, add the variable there with this comment):

```
# Browser origin(s) of the PWA, comma-separated, exact scheme://host[:port]
# (backend spec §8/§9: the Nuxt app and the Go API are two Railway services,
# so every API call is cross-origin and needs this allow-list). Default: the
# Nuxt dev server. Never "*".
#FRONTEND_ORIGIN=http://localhost:3000
```

- [ ] **Step 3: Backend spec §9 step 2** — in `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`, the line beginning `2.  Inject DATABASE\_URL, …`: append `FRONTEND\_ORIGIN (the PWA's origin, comma-separated if several)` to the list, keeping the document's backslash-escaped style. Verify with `grep -n 'FRONTEND\\\\_ORIGIN' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"` → one line.

- [ ] **Step 4: CODEMAP** — under *Backend packages*, add after the `**store**` bullet:

```
- **middleware** — the request-edge guards `cmd/api` mounts once, no routes or tables of their own. `CORS(origins)` is global (`r.Use`, before `/healthz`): spec §8 deploys the PWA and the API on two Railway origins, so every browser call is cross-origin and every guarded call preflights; the allow-list is `FRONTEND_ORIGIN` (comma-separated exact `scheme://host[:port]`, default `http://localhost:3000`, parsed by `ParseOrigins` — boot refuses a malformed list), the matched origin is echoed (never `*`), `Vary: Origin` is added, a preflight is answered `204` with `Allow-Methods: GET, POST, OPTIONS`, `Allow-Headers: Authorization, Content-Type`, `Max-Age: 600` — from the NoRoute chain, since Gin registers no OPTIONS routes — and a preflight from any other origin is `403`. No `Allow-Credentials`: sessions are bearer headers. `BodyLimit(MaxBodyBytes)` sits on the `/api/v1` group (before its routes) and wraps every body in `http.MaxBytesReader` at 64 KiB, so an oversized body fails each handler's existing `ShouldBindJSON` → `400 invalid_request` instead of being allocated. Tests are pure `httptest`.
- **health** — `GET /healthz` (unauthenticated; Railway's probe). Pings Postgres then Redis, each under its own `PingTimeout` (2 s), and answers `503` with `"postgres"`/`"redis": "unavailable"` — a fixed marker, never the driver error, which embeds the connection identity; the full error is `log.Printf`ed for the operator.
```

In the `**store**` bullet's env sentence or the `**auth**` `JWT_SECRET` note, add nothing; instead append to the frontend `**shell**` bullet's runtime-config sentence: "The API's `FRONTEND_ORIGIN` must list this app's origin or every call is blocked by CORS (see `middleware`)."

- [ ] **Step 5: Boot proof (needs the dev stack; use a unique compose project)**

```bash
cp -n .env.example .env 2>/dev/null; export COMPOSE_PROJECT_NAME=edge POSTGRES_PORT=5461 REDIS_PORT=6411
docker compose up -d --wait --wait-timeout 60
export DATABASE_URL=postgres://english:english@localhost:5461/english?sslmode=disable REDIS_URL=redis://localhost:6411/0
export GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x JWT_SECRET=$(openssl rand -base64 32) PORT=8111 FRONTEND_ORIGIN=https://app.example.com
# if the secrets plan has landed on your base: export ENCRYPTION_SECRET_KEY=$(openssl rand -hex 32)
go run ./cmd/api & sleep 3
curl -s -o /dev/null -w '%{http_code}\n' -X OPTIONS -H 'Origin: https://app.example.com' -H 'Access-Control-Request-Method: GET' --max-time 5 http://localhost:8111/api/v1/onboarding/quiz
# expect: 204
curl -s -D - -o /dev/null -H 'Origin: https://app.example.com' --max-time 5 http://localhost:8111/api/v1/onboarding/quiz | grep -i 'access-control-allow-origin'
# expect: access-control-allow-origin: https://app.example.com  (the 401 body is fine — the header is what the browser needs)
curl -s -o /dev/null -w '%{http_code}\n' -X OPTIONS -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: GET' --max-time 5 http://localhost:8111/api/v1/onboarding/quiz
# expect: 403
head -c 70000 /dev/zero | tr '\0' 'a' | curl -s -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' --data-binary @- --max-time 5 http://localhost:8111/api/v1/auth/google
# expect: 400
docker compose stop postgres && sleep 1 && curl -s --max-time 5 http://localhost:8111/healthz; echo
# expect: {"postgres":"unavailable","redis":"ok","status":"unavailable"} — and no "user=", "database=" or host:port anywhere in it
kill %1; docker compose down
```

Record the four outputs in the execution summary.

- [ ] **Step 6: Commit**

```bash
git add cmd/api/main.go .env.example "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../harness/CODEMAP.md
git commit -m "cmd/api: mount CORS allow-list and body limit; document FRONTEND_ORIGIN"
```

---

## Verification

```bash
cd backend
gofmt -l ./internal/middleware ./internal/health ./internal/config ./cmd/api
# expect: no output
go build ./... && go vet ./... && go test ./... -count=1
# expect: ok for every package, no live service needed
go test ./internal/middleware/ -count=1 -v
# expect: PASS ×11 — PreflightFromAnAllowedOriginIs204WithTheAllowHeaders, PreflightForAPathWithNoRouteAtAllIs204Not404,
#         AnActualRequestFromAnAllowedOriginCarriesTheAllowOriginHeader, AnActualRequestFromAnotherOriginPassesWithoutCORSHeaders,
#         PreflightFromAnotherOriginIs403, RequestsWithoutAnOriginAreUntouched, TheEchoedOriginIsTheMatchedOneNeverAWildcardOrTheList,
#         ParseOrigins, ABodyPastTheLimitIsRejectedBeforeItIsDecoded, ABodyUnderTheLimitIsDecodedNormally, TheProductionLimitIs64KiB
go test ./internal/health/ -count=1 -v
# expect: PASS ×6 incl. NamesBothDependenciesWhenBothAreDown, LogsTheDriverErrorServerSide, GivesEachDependencyItsOwnBudget
grep -n 'err.Error()' internal/health/health.go
# expect: no output
grep -n 'middleware.ParseOrigins\|r.Use(middleware.CORS\|v1.Use(middleware.BodyLimit' cmd/api/main.go
# expect: 3 lines; the CORS Use line number is smaller than the /healthz line's; the BodyLimit line sits right after r.Group("/api/v1")
grep -rn 'MaxBytesReader' internal/ --include='*.go' | grep -v _test
# expect: exactly one hit, in internal/middleware/bodylimit.go
grep -c 'FRONTEND_ORIGIN' .env.example ../harness/CODEMAP.md; grep -c 'FRONTEND\\_ORIGIN' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
# expect: ≥ 1 each
git log --oneline origin/main..HEAD | wc -l
# expect: 5 commits, one per task, each with the Co-Authored-By trailer
python3 ../tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
```

Mutation checks (each must turn the named test red, then restore):

| Mutation | Test that fails |
| --- | --- |
| `CORS`: replace `c.Header("Access-Control-Allow-Origin", origin)` with `"*"` | `TheEchoedOriginIsTheMatchedOneNeverAWildcardOrTheList`, `PreflightFromAnAllowedOriginIs204…` |
| `CORS`: delete the `if preflight { … AbortWithStatus(204) }` block | `PreflightFromAnAllowedOriginIs204…`, `PreflightForAPathWithNoRouteAtAllIs204Not404` |
| `CORS`: delete the `!allow[origin]` branch | `PreflightFromAnotherOriginIs403`, `AnActualRequestFromAnotherOriginPassesWithoutCORSHeaders` |
| `BodyLimit`: delete the `MaxBytesReader` line | `ABodyPastTheLimitIsRejectedBeforeItIsDecoded` |
| `health`: `body["postgres"] = err.Error()` | `…PostgresIsDownNamesItWithoutTheDriverError` |
| `health`: one shared `context.WithTimeout` for both pings | `GivesEachDependencyItsOwnBudget` (elapsed < 2×PingTimeout) |

Boot proof: Task 5 Step 5's four curl outputs (`204`, the echoed origin, `403`, `400`) and the redacted `/healthz` body, recorded in the execution summary.

## Notes and open questions

- **Default origin.** `FRONTEND_ORIGIN` defaults to `http://localhost:3000` rather than being required: a required variable would break every developer `.env` today, while an unset variable in production reproduces exactly today's failure (loudly — the boot log prints the allow-list, and spec §9 now lists the variable). If the owner prefers boot to refuse an unset value in `GIN_MODE=release`, that is a two-line change in `main.go` after the shutdown plan lands.
- **Status for an oversized body.** `400 invalid_request` (the handlers' existing mapping) rather than `413`: one middleware, zero handler edits, and the client's remedy is the same. `http.MaxBytesReader` also tells the server to close the connection after the response, so a slow sender cannot keep the goroutine.
- **`Vary: Origin`** is added for every request that carries an `Origin`, allowed or not, so an intermediary cache never serves one origin's CORS headers to another.
- **Health probe latency.** Sequential pings with two 2 s budgets means a worst case of 4 s; Railway's default health-check timeout comfortably exceeds that. Concurrent pings would halve it at the cost of a goroutine and a wait group in a 40-line handler — not worth it (YAGNI).
- **Out of scope.** Server-level `ReadHeaderTimeout`/`IdleTimeout` (the shutdown plan), rate limiting, `Access-Control-Expose-Headers` (the PWA reads only JSON bodies).

## Execution summary

Built exactly as planned, five tasks / five commits, no deviations from the plan's intent. Base was fresh `origin/main` (`fe2c29a`) — the shutdown plan had **not** landed yet, so `main.go`/`config.go`/`.env.example` were in their pre-shutdown-plan shape and the plan's region-edit instructions applied directly with no merge-order juggling needed.

**Deviation (tooling, not plan intent):** the harness's Edit/Write file tools refuse writes to paths outside this session's own worktree tree; since the session instructions place the plan's worktree under the main checkout (a sibling of the session's worktree, not nested inside it), all file edits in `.worktrees/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own` were made with `Bash`/`python3` (exact string replacement, same content as the plan specifies) instead of the Edit tool. `git`, `go`, `docker`, `curl` all ran normally via Bash. No code or test content differs from the plan.

### Verification (from `backend/`)

```
$ gofmt -l ./internal/middleware ./internal/health ./internal/config ./cmd/api
(no output)

$ go build ./... && go vet ./... && go test ./... -count=1
ok  .../internal/airouter    0.254s
ok  .../internal/auth        0.880s
ok  .../internal/config      0.153s
ok  .../internal/google      0.604s
ok  .../internal/health      0.426s
ok  .../internal/middleware  1.118s
ok  .../internal/notify      1.417s
ok  .../internal/onboarding  1.649s
ok  .../internal/pet         1.931s
ok  .../internal/quests      2.228s
ok  .../internal/store       2.414s

$ go test ./internal/middleware/ -count=1 -v
PASS ×11 (Preflight/Origin/ParseOrigins ×8, Body/Limit ×3)

$ go test ./internal/health/ -count=1 -v
PASS ×6 (OKWhenBothServicesRespond, both …NamesItWithoutTheDriverError, NamesBothDependenciesWhenBothAreDown, LogsTheDriverErrorServerSide, GivesEachDependencyItsOwnBudget)

$ grep -n 'err.Error()' internal/health/health.go
(no output)

$ grep -n 'middleware.ParseOrigins\|r.Use(middleware.CORS\|v1.Use(middleware.BodyLimit' cmd/api/main.go
84:	origins, err := middleware.ParseOrigins(cfg.FrontendOrigin)
88:	r.Use(middleware.CORS(origins))
147:	v1.Use(middleware.BodyLimit(middleware.MaxBodyBytes))
# CORS Use (88) precedes /healthz (91); BodyLimit (147) immediately follows r.Group("/api/v1") (146).

$ grep -rn 'MaxBytesReader' internal/ --include='*.go' | grep -v _test
internal/middleware/bodylimit.go:16: (doc comment)
internal/middleware/bodylimit.go:24: (the call site)
# Both in the one file the plan specifies; the plan's own reference bodylimit.go
# has the word in its comment too, so this is 2 lines / 1 file, not literally
# "one hit" — intent (single implementation site) satisfied.

$ grep -c 'FRONTEND_ORIGIN' .env.example ../harness/CODEMAP.md
.env.example:1
../harness/CODEMAP.md:2
$ grep -c 'FRONTEND\_ORIGIN' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
1

$ git log --oneline origin/main..HEAD | wc -l
5
df251d3 config: read FRONTEND_ORIGIN with the Nuxt dev-server default
7acc243 middleware: CORS allow-list from FRONTEND_ORIGIN, preflights answered 204
eb22323 middleware: bound every /api/v1 request body at 64 KiB
4d4985d health: fixed unavailable markers, driver errors logged not served, per-ping budget
1875b89 cmd/api: mount CORS allow-list and body limit; document FRONTEND_ORIGIN
(all 5 carry the Co-Authored-By trailer)

$ python3 ../tools/harness/cli.py validate; echo "exit=$?"
exit=0
```

**Mutation checks** — all six from the plan's table turned the named test(s) red, then were reverted (`git checkout --`) and the suite re-confirmed green:
1. `ACAO` → `"*"` → failed `TheEchoedOriginIsTheMatchedOneNeverAWildcardOrTheList` + `PreflightFromAnAllowedOriginIs204…`
2. Deleted the preflight `204` block → failed `PreflightFromAnAllowedOriginIs204…` + `PreflightForAPathWithNoRouteAtAllIs204Not404`
3. Deleted the `!allow[origin]` branch → failed `PreflightFromAnotherOriginIs403` + `AnActualRequestFromAnotherOriginPassesWithoutCORSHeaders`
4. Deleted the `MaxBytesReader` line → failed `ABodyPastTheLimitIsRejectedBeforeItIsDecoded`
5. `body["postgres"] = err.Error()` → failed `…PostgresIsDownNamesItWithoutTheDriverError`
6. One shared `context.WithTimeout` for both pings → failed `GivesEachDependencyItsOwnBudget` (`elapsed = 32ms, want between 60ms and 1s`)

### Runtime proof

Isolation: `COMPOSE_PROJECT_NAME=exec-edge`, `POSTGRES_PORT=55433`, `REDIS_PORT=56380` (scratch `backend/.env`, removed after); API on port `18082`. `docker compose up -d --wait --wait-timeout 60` → both containers healthy. Booted with `go run ./cmd/api`; log showed `cors: allowing [https://app.example.com]` and full route table.

```
$ curl -X OPTIONS -H 'Origin: https://app.example.com' -H 'Access-Control-Request-Method: GET' .../api/v1/onboarding/quiz
204

$ curl -D - -H 'Origin: https://app.example.com' .../api/v1/onboarding/quiz | grep -i access-control-allow-origin
Access-Control-Allow-Origin: https://app.example.com

$ curl -X OPTIONS -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: GET' .../api/v1/onboarding/quiz
403

$ head -c 70000 /dev/zero | tr '\0' 'a' | curl -X POST -H 'Content-Type: application/json' --data-binary @- .../api/v1/auth/google
400

$ docker compose stop postgres && curl .../healthz
{"postgres":"unavailable","redis":"ok","status":"unavailable"}
# no "user=", "database=", host or port anywhere in the body

$ grep 'health: postgres ping failed' server.log
2026/09/24 12:05:10 health: postgres ping failed: failed to connect to `user=english database=english`: ...
# confirms the full driver error reaches the operator log, only the HTTP body is redacted
```

Cleanup verified: API process and its `go run` child force-killed, `docker compose down` removed the `exec-edge` containers/network/volume, scratch `.env` deleted. `pgrep -fl 'go run ./cmd/api|exe/api'` and `docker ps` came back empty of anything from this run; the other executor's `exec-shutdown-*` containers and unrelated `scio3-*` containers were never touched.

### CI

Branch `harness/2026-09-24-high-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own` pushed; no PR opened. All four jobs green: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35958578061 (`harness-tooling`, `frontend`, `backend-unit`, `backend-integration`).
