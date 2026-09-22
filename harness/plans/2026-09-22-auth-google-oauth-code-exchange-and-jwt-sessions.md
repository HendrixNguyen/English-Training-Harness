---
idea: harness/ideas/2026-09-22-run-02/auth-google-oauth-code-exchange-and-jwt-sessions.md
status: done
priority: high
merged: true
order: 2
branch: harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions
worktree: .worktrees/auth-google-oauth-code-exchange-and-jwt-sessions
---
# Auth: Google OAuth code exchange and JWT sessions — Plan

**Idea:** `harness/ideas/2026-09-22-run-02/auth-google-oauth-code-exchange-and-jwt-sessions.md`
**Goal:** Add `backend/internal/auth` — `POST /api/v1/auth/google` (code exchange → user upsert → JWT + Redis session) and the `auth.Require()` Gin middleware every later slice mounts behind — with a test suite that needs neither Google nor live Postgres/Redis.

**Architecture:** Four collaborators behind interfaces, wired by a `Service`: `GoogleClient` (HTTP, endpoint URLs injectable so tests point at `httptest`), `UserRepo` (the single `ON CONFLICT (google_id)` upsert), `SessionStore` (Redis `sess:{user_id}:token` using `store.SessionKey`/`store.SessionTTL`), and `TokenIssuer` (HS256 JWT). The handler and the middleware are thin. Every test in the default `go test ./...` run uses fakes; the one SQL statement is additionally covered by a `DATABASE_URL`-gated test that skips, matching slice 1's convention.

**Tech stack:** Go 1.22, Gin, `github.com/golang-jwt/jwt/v5`, `net/http` + `net/http/httptest`. No `golang.org/x/oauth2` — the exchange is one form POST and one GET, and hand-rolling it keeps the outgoing parameters assertable.

**Depends on:** slice 1 (`store`). Do not start until `backend/go.mod`, `internal/store` and `internal/config` exist on `main`.

**Run every command from `backend/`** unless the step says otherwise. `rg` is not installed — use `grep -n`.

## File structure

| Path | Responsibility |
| --- | --- |
| `backend/internal/config/config.go` `_test.go` | + `GoogleClientID`, `GoogleClientSecret`, `JWTSecret` |
| `backend/internal/auth/scopes.go` `scopes_test.go` | the scope list + `access_type=offline`, shared with frontend-shell |
| `backend/internal/auth/token.go` `token_test.go` | HS256 issue/verify |
| `backend/internal/auth/google.go` `google_test.go` | code exchange + userinfo over injectable endpoints |
| `backend/internal/auth/repo.go` `repo_test.go` | `User` model, `UserRepo` interface, `PgUserRepo` upsert SQL |
| `backend/internal/auth/session.go` `session_test.go` | `SessionStore` interface + `RedisSessionStore` |
| `backend/internal/auth/service.go` `service_test.go` | orchestration: exchange → upsert → issue → store |
| `backend/internal/auth/handler.go` `handler_test.go` | `POST /api/v1/auth/google` |
| `backend/internal/auth/middleware.go` `middleware_test.go` | `Require()` |
| `backend/internal/auth/integration_test.go` | upsert against a real database, skipped without `DATABASE_URL` |
| `backend/cmd/api/main.go` | mount the route |
| `harness/CODEMAP.md` | `auth` paragraph |

---

## Tasks

### Task 1: Config gains the auth environment

**Files:**
- Modify: `backend/internal/config/config.go`
- Test: `backend/internal/config/config_test.go` (append)

- [ ] **Step 1: Write the failing test** (append)

```go
func TestLoadRequiresGoogleAndJWTSecrets(t *testing.T) {
	base := func(t *testing.T) {
		t.Helper()
		t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("GOOGLE_CLIENT_ID", "cid")
		t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
		t.Setenv("JWT_SECRET", "s3cret")
	}

	for _, missing := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "JWT_SECRET"} {
		t.Run("missing "+missing, func(t *testing.T) {
			base(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error when %s is unset, got nil", missing)
			}
		})
	}

	t.Run("all present", func(t *testing.T) {
		base(t)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() = %v, want nil error", err)
		}
		if cfg.GoogleClientID != "cid" || cfg.GoogleClientSecret != "csecret" || cfg.JWTSecret != "s3cret" {
			t.Errorf("got %+v", cfg)
		}
	})
}
```

The existing `config_test.go` tests set only `DATABASE_URL`/`REDIS_URL`; add the three new variables to
each of them via `t.Setenv` so they keep passing.

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/config/...
```
Expected: FAIL — `cfg.GoogleClientID undefined`.

- [ ] **Step 3: Implement**

Add to `Config`:
```go
	GoogleClientID     string
	GoogleClientSecret string
	// JWTSecret signs session tokens. NOTE: JWT_SECRET is NOT in the spec §8
	// environment list — see the CODEMAP auth paragraph and the plan's open questions.
	JWTSecret string
```
and in `Load`, after the existing reads:
```go
	cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	cfg.JWTSecret = os.Getenv("JWT_SECRET")

	for name, v := range map[string]string{
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"JWT_SECRET":           cfg.JWTSecret,
	} {
		if v == "" {
			return Config{}, fmt.Errorf("config: %s is required", name)
		}
	}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/config/... -v
```
Expected: all `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: config reads GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, JWT_SECRET"
```

---

### Task 2: The OAuth scope list

**Files:**
- Create: `backend/internal/auth/scopes.go`
- Test: `backend/internal/auth/scopes_test.go`

This is the cross-slice contract from `_run.md`: slice 6 (google) needs these scopes and slice 8
(frontend-shell) builds the consent URL from the same list. Getting it wrong means every early user
re-consents later.

- [ ] **Step 1: Write the failing test**

`backend/internal/auth/scopes_test.go`:
```go
package auth

import (
	"strings"
	"testing"
)

func TestScopesCoverOpenIDCalendarAndTasks(t *testing.T) {
	want := []string{
		"openid",
		"email",
		"profile",
		"https://www.googleapis.com/auth/calendar.events",
		"https://www.googleapis.com/auth/tasks",
	}
	if len(Scopes) != len(want) {
		t.Fatalf("Scopes = %v, want %d entries", Scopes, len(want))
	}
	for i, w := range want {
		if Scopes[i] != w {
			t.Errorf("Scopes[%d] = %q, want %q", i, Scopes[i], w)
		}
	}
}

func TestScopeStringIsSpaceSeparated(t *testing.T) {
	got := ScopeString()
	if strings.Count(got, " ") != len(Scopes)-1 {
		t.Errorf("ScopeString() = %q, want space-separated", got)
	}
	if !strings.Contains(got, "auth/tasks") {
		t.Errorf("ScopeString() = %q, missing the tasks scope", got)
	}
}

func TestAuthCodeURLRequestsOfflineAccess(t *testing.T) {
	u := AuthCodeURL("cid", "https://app.example.com/callback", "state-123")

	for _, want := range []string{
		"access_type=offline",
		"prompt=consent",
		"response_type=code",
		"client_id=cid",
		"state=state-123",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("AuthCodeURL() = %q, missing %q", u, want)
		}
	}
	if !strings.HasPrefix(u, "https://accounts.google.com/o/oauth2/v2/auth?") {
		t.Errorf("AuthCodeURL() = %q, wrong endpoint", u)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
mkdir -p internal/auth && go test ./internal/auth/...
```
Expected: build failure, `undefined: Scopes`.

- [ ] **Step 3: Implement**

`backend/internal/auth/scopes.go`:
```go
// Package auth owns Google sign-in, session tokens and the Gin middleware that
// every per-user route in spec §7 sits behind.
package auth

import (
	"net/url"
	"strings"
)

// AuthEndpoint is Google's consent screen. The frontend builds its login link
// from AuthCodeURL so the scope list never diverges from the one the backend
// exchanges against.
const AuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"

// Scopes is the full consent set. calendar.events and tasks are requested at
// first sign-in — not later, when the google slice lands — so that no existing
// user has to re-consent (spec §5.1 steps 6-7).
var Scopes = []string{
	"openid",
	"email",
	"profile",
	"https://www.googleapis.com/auth/calendar.events",
	"https://www.googleapis.com/auth/tasks",
}

// ScopeString joins Scopes the way Google's `scope` parameter expects.
func ScopeString() string { return strings.Join(Scopes, " ") }

// AuthCodeURL builds the consent URL. access_type=offline plus prompt=consent
// is what makes Google return a refresh token (users.google_refresh_token).
func AuthCodeURL(clientID, redirectURI, state string) string {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {ScopeString()},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return AuthEndpoint + "?" + q.Encode()
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/auth/... -v
```
Expected: three `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: Google scope list with offline access"
```

---

### Task 3: JWT issue and verify

**Files:**
- Create: `backend/internal/auth/token.go`
- Test: `backend/internal/auth/token_test.go`

- [ ] **Step 1: Add the dependency**

```sh
go get github.com/golang-jwt/jwt/v5@latest
```

- [ ] **Step 2: Write the failing test**

`backend/internal/auth/token_test.go`:
```go
package auth

import (
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestIssueThenVerifyRoundTrips(t *testing.T) {
	iss := NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) })

	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}
	userID, err := iss.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}
}

func TestTokenExpiryMatchesTheRedisSessionTTL(t *testing.T) {
	if TokenTTL != store.SessionTTL {
		t.Fatalf("TokenTTL = %v, store.SessionTTL = %v — they must be one value", TokenTTL, store.SessionTTL)
	}
	if TokenTTL != 24*time.Hour {
		t.Errorf("TokenTTL = %v, want 24h (spec §4)", TokenTTL)
	}
}

func TestVerifyRejectsAnExpiredToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	iss := NewTokenIssuer("secret", func() time.Time { return now })
	tok, err := iss.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}

	later := NewTokenIssuer("secret", func() time.Time { return now.Add(TokenTTL + time.Minute) })
	if _, err := later.Verify(tok); err == nil {
		t.Fatal("expected an error for an expired token, got nil")
	}
}

func TestVerifyRejectsAnotherSecret(t *testing.T) {
	tok, err := NewTokenIssuer("secret", time.Now).Issue("user-1")
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}
	if _, err := NewTokenIssuer("other-secret", time.Now).Verify(tok); err == nil {
		t.Fatal("expected an error for a token signed with another secret, got nil")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)

	for _, bad := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := iss.Verify(bad); err == nil {
			t.Errorf("Verify(%q) = nil error, want an error", bad)
		}
	}
}

func TestVerifyRejectsTheNoneAlgorithm(t *testing.T) {
	// alg=none with {"sub":"user-1"} and an empty signature.
	const none = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyLTEifQ."
	if _, err := NewTokenIssuer("secret", time.Now).Verify(none); err == nil {
		t.Fatal("expected an error for alg=none, got nil")
	}
}
```

- [ ] **Step 3: Run and confirm it fails**

```sh
go test ./internal/auth/... -run Token
```
Expected: build failure, `undefined: NewTokenIssuer`.

- [ ] **Step 4: Implement**

`backend/internal/auth/token.go`:
```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TokenTTL is both the JWT exp window and the Redis session TTL, so the two can
// never drift (spec §4: sess:{user_id}:token, 24 hours).
const TokenTTL = store.SessionTTL

// TokenIssuer signs and verifies session JWTs (HS256).
type TokenIssuer struct {
	secret []byte
	now    func() time.Time
}

// NewTokenIssuer builds an issuer. now is injectable so expiry is testable.
func NewTokenIssuer(secret string, now func() time.Time) *TokenIssuer {
	if now == nil {
		now = time.Now
	}
	return &TokenIssuer{secret: []byte(secret), now: now}
}

// Issue returns a signed token whose subject is userID.
func (t *TokenIssuer) Issue(userID string) (string, error) {
	now := t.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing token: %w", err)
	}
	return signed, nil
}

// Verify checks the signature and expiry and returns the subject.
func (t *TokenIssuer) Verify(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(tok *jwt.Token) (any, error) {
			if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("auth: unexpected signing method %v", tok.Header["alg"])
			}
			return t.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return "", fmt.Errorf("auth: verifying token: %w", err)
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", fmt.Errorf("auth: token has no subject")
	}
	return claims.Subject, nil
}
```

- [ ] **Step 5: Run and confirm it passes**

```sh
go mod tidy && go test ./internal/auth/... -run Token -v
```
Expected: six `--- PASS` lines.

- [ ] **Step 6: Commit**

```sh
cd .. && git add backend && git commit -m "auth: HS256 session tokens sharing the §4 24h TTL"
```

---

### Task 4: Google code exchange and profile fetch

**Files:**
- Create: `backend/internal/auth/google.go`
- Test: `backend/internal/auth/google_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/auth/google_test.go`:
```go
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeSendsTheAuthorizationCodeGrant(t *testing.T) {
	var gotForm map[string][]string

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3599,"token_type":"Bearer"}`))
	}))
	defer tokenSrv.Close()

	c := &GoogleClient{
		ClientID:      "cid",
		ClientSecret:  "csecret",
		TokenURL:      tokenSrv.URL,
		UserInfoURL:   "unused",
		HTTPClient:    tokenSrv.Client(),
	}

	tok, err := c.Exchange(context.Background(), "the-code", "https://app.example.com/callback")
	if err != nil {
		t.Fatalf("Exchange() = %v", err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" {
		t.Errorf("token = %+v", tok)
	}

	want := map[string]string{
		"grant_type":    "authorization_code",
		"code":          "the-code",
		"redirect_uri":  "https://app.example.com/callback",
		"client_id":     "cid",
		"client_secret": "csecret",
	}
	for k, v := range want {
		if got := gotForm.Get(k); got != v {
			t.Errorf("form[%s] = %q, want %q", k, got, v)
		}
	}
}

func TestExchangeReportsAGoogleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{TokenURL: srv.URL, HTTPClient: srv.Client()}

	if _, err := c.Exchange(context.Background(), "bad", "uri"); err == nil {
		t.Fatal("expected an error for a 400 from Google, got nil")
	}
}

func TestUserInfoSendsTheBearerTokenAndParsesTheProfile(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"google-123","email":"a@example.com","name":"A Person"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{UserInfoURL: srv.URL, HTTPClient: srv.Client()}

	p, err := c.UserInfo(context.Background(), "at")
	if err != nil {
		t.Fatalf("UserInfo() = %v", err)
	}
	if gotAuth != "Bearer at" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer at")
	}
	if p.Sub != "google-123" || p.Email != "a@example.com" || p.Name != "A Person" {
		t.Errorf("profile = %+v", p)
	}
}

func TestUserInfoRejectsAProfileWithoutSubOrEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"No IDs"}`))
	}))
	defer srv.Close()

	c := &GoogleClient{UserInfoURL: srv.URL, HTTPClient: srv.Client()}

	if _, err := c.UserInfo(context.Background(), "at"); err == nil {
		t.Fatal("expected an error when sub/email are missing, got nil")
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/auth/... -run 'Exchange|UserInfo'
```
Expected: build failure, `undefined: GoogleClient`.

- [ ] **Step 3: Implement**

`backend/internal/auth/google.go`:
```go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default Google endpoints. They are fields on GoogleClient so tests can point
// at an httptest.Server — this slice never calls Google in `go test ./...`.
const (
	DefaultTokenURL    = "https://oauth2.googleapis.com/token"
	DefaultUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
)

// GoogleToken is the subset of Google's token response this product uses.
// RefreshToken is empty on re-consent unless prompt=consent was requested — see
// AuthCodeURL in scopes.go.
type GoogleToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GoogleProfile is the OpenID userinfo payload (spec §5.1 step 2).
type GoogleProfile struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Exchanger is the Google side of sign-in, so Service can be tested with a fake.
type Exchanger interface {
	Exchange(ctx context.Context, code, redirectURI string) (GoogleToken, error)
	UserInfo(ctx context.Context, accessToken string) (GoogleProfile, error)
}

// GoogleClient is the real Exchanger.
type GoogleClient struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
	HTTPClient   *http.Client
}

// NewGoogleClient builds a client against the production endpoints.
func NewGoogleClient(clientID, clientSecret string) *GoogleClient {
	return &GoogleClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     DefaultTokenURL,
		UserInfoURL:  DefaultUserInfoURL,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *GoogleClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Exchange swaps an authorization code for tokens (spec §5.1 step 2).
func (c *GoogleClient) Exchange(ctx context.Context, code, redirectURI string) (GoogleToken, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return GoogleToken{}, fmt.Errorf("auth: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tok GoogleToken
	if err := c.doJSON(req, &tok); err != nil {
		return GoogleToken{}, err
	}
	if tok.AccessToken == "" {
		return GoogleToken{}, fmt.Errorf("auth: Google returned no access token")
	}
	return tok, nil
}

// UserInfo fetches sub/email/name for an access token.
func (c *GoogleClient) UserInfo(ctx context.Context, accessToken string) (GoogleProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return GoogleProfile{}, fmt.Errorf("auth: building userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var p GoogleProfile
	if err := c.doJSON(req, &p); err != nil {
		return GoogleProfile{}, err
	}
	if p.Sub == "" || p.Email == "" {
		return GoogleProfile{}, fmt.Errorf("auth: Google profile is missing sub or email")
	}
	return p, nil
}

func (c *GoogleClient) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("auth: calling %s: %w", req.URL.Host, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("auth: reading %s response: %w", req.URL.Host, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("auth: %s returned %d: %s", req.URL.Host, resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("auth: decoding %s response: %w", req.URL.Host, err)
	}
	return nil
}

var _ Exchanger = (*GoogleClient)(nil)
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/auth/... -run 'Exchange|UserInfo' -v
```
Expected: four `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: Google code exchange and profile fetch"
```

---

### Task 5: User upsert

**Files:**
- Create: `backend/internal/auth/repo.go`
- Test: `backend/internal/auth/repo_test.go`

The idea's contract: upsert on `google_id`; set `email`, `full_name`, `google_refresh_token`; **keep**
`cefr_current` and `target_goal` on re-login; insert `target_goal = ''` for a new user because §3.2 has it
`NOT NULL` with no default.

- [ ] **Step 1: Write the failing test**

`backend/internal/auth/repo_test.go`:
```go
package auth

import (
	"strings"
	"testing"
)

func TestUpsertSQLMatchesTheContract(t *testing.T) {
	sql := upsertUserSQL

	for _, want := range []string{
		"INSERT INTO users",
		"ON CONFLICT (google_id) DO UPDATE",
		"email = EXCLUDED.email",
		"full_name = EXCLUDED.full_name",
		"RETURNING id, email, full_name, cefr_current",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("upsertUserSQL is missing %q:\n%s", want, sql)
		}
	}

	// cefr_current and target_goal must never appear on the UPDATE side:
	// re-login must not reset a user's level or goal.
	update := sql[strings.Index(sql, "DO UPDATE"):]
	for _, forbidden := range []string{"cefr_current =", "target_goal ="} {
		if strings.Contains(update, forbidden) {
			t.Errorf("the DO UPDATE clause must not set %q:\n%s", forbidden, update)
		}
	}

	// An empty refresh token from Google must not wipe the stored one.
	if !strings.Contains(update, "COALESCE(NULLIF(EXCLUDED.google_refresh_token, ''), users.google_refresh_token)") {
		t.Errorf("re-login with no refresh_token must keep the stored one:\n%s", update)
	}
}

func TestNewUserGetsAnEmptyTargetGoal(t *testing.T) {
	// users.target_goal is NOT NULL with no default in spec §3.2, and the goal
	// is only known after onboarding.
	if !strings.Contains(upsertUserSQL, "target_goal") {
		t.Fatal("the INSERT must name target_goal explicitly")
	}
	if defaultTargetGoal != "" {
		t.Errorf("defaultTargetGoal = %q, want the empty string", defaultTargetGoal)
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/auth/... -run 'Upsert|NewUser'
```
Expected: build failure, `undefined: upsertUserSQL`.

- [ ] **Step 3: Implement**

`backend/internal/auth/repo.go`:
```go
package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User is the slice of spec §3.2 `users` that sign-in reads back.
type User struct {
	ID          string
	Email       string
	FullName    string
	CEFRCurrent string
}

// defaultTargetGoal is what a brand-new user gets. §3.2 makes target_goal
// NOT NULL with no default, but the goal is only known after onboarding, so
// sign-in writes the empty string and onboarding fills it in later.
const defaultTargetGoal = ""

// UserRepo is the Postgres side of sign-in, so Service can be tested with a fake.
type UserRepo interface {
	// UpsertByGoogleID creates or refreshes the user and returns the stored row.
	UpsertByGoogleID(ctx context.Context, googleID, email, fullName, refreshToken string) (User, error)
}

// upsertUserSQL creates the user on first sign-in and refreshes the Google-owned
// fields afterwards. cefr_current and target_goal are deliberately absent from
// the UPDATE: re-login must not reset a learner's level or goal. An empty
// refresh token (Google omits it on silent re-consent) keeps the stored value.
const upsertUserSQL = `
INSERT INTO users (email, full_name, google_id, google_refresh_token, target_goal)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (google_id) DO UPDATE SET
    email = EXCLUDED.email,
    full_name = EXCLUDED.full_name,
    google_refresh_token = COALESCE(NULLIF(EXCLUDED.google_refresh_token, ''), users.google_refresh_token)
RETURNING id, email, full_name, cefr_current`

// PgUserRepo is the real UserRepo.
type PgUserRepo struct{ Pool *pgxpool.Pool }

// NewPgUserRepo builds a repo over an existing pool.
func NewPgUserRepo(pool *pgxpool.Pool) *PgUserRepo { return &PgUserRepo{Pool: pool} }

func (r *PgUserRepo) UpsertByGoogleID(ctx context.Context, googleID, email, fullName, refreshToken string) (User, error) {
	// full_name and cefr_current are nullable in §3.2, so scan through pointers
	// and flatten NULL to the zero value.
	var (
		u        User
		nullName *string
		nullCEFR *string
	)
	err := r.Pool.QueryRow(ctx, upsertUserSQL,
		email, fullName, googleID, refreshToken, defaultTargetGoal,
	).Scan(&u.ID, &u.Email, &nullName, &nullCEFR)
	if err != nil {
		return User{}, fmt.Errorf("auth: upserting user: %w", err)
	}
	if nullName != nil {
		u.FullName = *nullName
	}
	if nullCEFR != nil {
		u.CEFRCurrent = *nullCEFR
	}
	return u, nil
}

var _ UserRepo = (*PgUserRepo)(nil)
```

> **Implementation note:** the argument order in `QueryRow` must match `$1..$5` in `upsertUserSQL`
> (`email, full_name, google_id, google_refresh_token, target_goal`). A mismatch here is the single most
> likely bug in this task — re-read both lines side by side before running the tests. Note that the Go
> parameter order (`googleID` first) deliberately differs from the SQL placeholder order.

- [ ] **Step 4: Run and confirm it passes**

```sh
go build ./... && go test ./internal/auth/... -run 'Upsert|NewUser' -v
```
Expected: two `--- PASS` lines, and `go build` clean (the placeholder resolved).

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: users upsert on google_id preserving cefr_current and target_goal"
```

---

### Task 6: Redis session store

**Files:**
- Create: `backend/internal/auth/session.go`
- Test: `backend/internal/auth/session_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/auth/session_test.go`:
```go
package auth

import (
	"context"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// fakeSessions is the in-memory SessionStore used across this package's tests.
type fakeSessions struct {
	vals map[string]string
	ttls map[string]time.Duration
	err  error
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{vals: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (f *fakeSessions) Put(_ context.Context, userID, token string, ttl time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.vals[userID] = token
	f.ttls[userID] = ttl
	return nil
}

func (f *fakeSessions) Get(_ context.Context, userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.vals[userID]
	if !ok {
		return "", ErrNoSession
	}
	return v, nil
}

func (f *fakeSessions) Delete(_ context.Context, userID string) error {
	delete(f.vals, userID)
	return nil
}

func TestSessionKeyComesFromStore(t *testing.T) {
	// The store package owns the §4 key topology; auth must not build key
	// strings of its own.
	if got, want := store.SessionKey("u1"), "sess:u1:token"; got != want {
		t.Fatalf("store.SessionKey = %q, want %q", got, want)
	}
}

func TestFakeSessionsSatisfiesSessionStore(t *testing.T) {
	var s SessionStore = newFakeSessions()

	ctx := context.Background()
	if err := s.Put(ctx, "u1", "tok", store.SessionTTL); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, "u1")
	if err != nil || got != "tok" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if err := s.Delete(ctx, "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "u1"); err == nil {
		t.Fatal("Get after Delete = nil error, want ErrNoSession")
	}
}
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/auth/... -run Session
```
Expected: build failure, `undefined: SessionStore`.

- [ ] **Step 3: Implement**

`backend/internal/auth/session.go`:
```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// ErrNoSession means the sess:{user_id}:token key is absent — the session was
// revoked or has expired.
var ErrNoSession = errors.New("auth: no active session")

// SessionStore is the Redis side of sign-in. Deleting a user's key revokes
// their session even though the JWT itself is still within its exp window.
type SessionStore interface {
	Put(ctx context.Context, userID, token string, ttl time.Duration) error
	Get(ctx context.Context, userID string) (string, error)
	Delete(ctx context.Context, userID string) error
}

// RedisSessionStore is the real SessionStore, keyed by store.SessionKey (§4).
type RedisSessionStore struct{ Client *redis.Client }

// NewRedisSessionStore builds a store over an existing client.
func NewRedisSessionStore(r *store.Redis) *RedisSessionStore {
	return &RedisSessionStore{Client: r.Client}
}

func (s *RedisSessionStore) Put(ctx context.Context, userID, token string, ttl time.Duration) error {
	if err := s.Client.Set(ctx, store.SessionKey(userID), token, ttl).Err(); err != nil {
		return fmt.Errorf("auth: storing session: %w", err)
	}
	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, userID string) (string, error) {
	v, err := s.Client.Get(ctx, store.SessionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNoSession
	}
	if err != nil {
		return "", fmt.Errorf("auth: reading session: %w", err)
	}
	return v, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.SessionKey(userID)).Err(); err != nil {
		return fmt.Errorf("auth: deleting session: %w", err)
	}
	return nil
}

var _ SessionStore = (*RedisSessionStore)(nil)
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go mod tidy && go test ./internal/auth/... -run Session -v
```
Expected: two `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: Redis session store over the §4 sess key"
```

---

### Task 7: Sign-in service and the `POST /api/v1/auth/google` handler

**Files:**
- Create: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/handler.go`
- Test: `backend/internal/auth/service_test.go`
- Test: `backend/internal/auth/handler_test.go`

- [ ] **Step 1: Write the failing tests**

`backend/internal/auth/service_test.go`:
```go
package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExchanger struct {
	token   GoogleToken
	profile GoogleProfile
	err     error

	gotCode, gotRedirect, gotAccessToken string
}

func (f *fakeExchanger) Exchange(_ context.Context, code, redirectURI string) (GoogleToken, error) {
	f.gotCode, f.gotRedirect = code, redirectURI
	return f.token, f.err
}

func (f *fakeExchanger) UserInfo(_ context.Context, accessToken string) (GoogleProfile, error) {
	f.gotAccessToken = accessToken
	return f.profile, f.err
}

type fakeRepo struct {
	users map[string]User // keyed by google_id
	last  struct{ googleID, email, fullName, refreshToken string }
	err   error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{users: map[string]User{}} }

func (f *fakeRepo) UpsertByGoogleID(_ context.Context, googleID, email, fullName, refreshToken string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	f.last.googleID, f.last.email, f.last.fullName, f.last.refreshToken = googleID, email, fullName, refreshToken

	u, ok := f.users[googleID]
	if !ok {
		u = User{ID: "id-" + googleID, CEFRCurrent: "A1"} // a fresh row's §3.2 default
	}
	u.Email, u.FullName = email, fullName
	f.users[googleID] = u
	return u, nil
}

func newTestService(ex *fakeExchanger, repo *fakeRepo, sess *fakeSessions) *Service {
	return NewService(ex, repo, sess, NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) }))
}

func TestSignInCreatesANewUserAndASession(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	repo, sess := newFakeRepo(), newFakeSessions()
	svc := newTestService(ex, repo, sess)

	out, err := svc.SignIn(context.Background(), "the-code", "https://app/cb")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}

	if ex.gotCode != "the-code" || ex.gotRedirect != "https://app/cb" {
		t.Errorf("Exchange got (%q, %q)", ex.gotCode, ex.gotRedirect)
	}
	if ex.gotAccessToken != "at" {
		t.Errorf("UserInfo got access token %q, want %q", ex.gotAccessToken, "at")
	}
	if repo.last.googleID != "google-1" || repo.last.refreshToken != "rt" || repo.last.email != "a@example.com" {
		t.Errorf("upsert got %+v", repo.last)
	}
	if out.User.ID != "id-google-1" || out.User.CEFRCurrent != "A1" {
		t.Errorf("user = %+v", out.User)
	}

	stored, err := sess.Get(context.Background(), out.User.ID)
	if err != nil {
		t.Fatalf("no session stored: %v", err)
	}
	if stored != out.Token {
		t.Errorf("stored session = %q, want the issued token %q", stored, out.Token)
	}
	if sess.ttls[out.User.ID] != TokenTTL {
		t.Errorf("session TTL = %v, want %v", sess.ttls[out.User.ID], TokenTTL)
	}
}

func TestSignInKeepsAReturningUsersLevel(t *testing.T) {
	repo := newFakeRepo()
	repo.users["google-1"] = User{ID: "id-google-1", CEFRCurrent: "B2"}

	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at", RefreshToken: "rt2"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com", Name: "A Person"},
	}
	svc := newTestService(ex, repo, newFakeSessions())

	out, err := svc.SignIn(context.Background(), "code", "uri")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}
	if out.User.CEFRCurrent != "B2" {
		t.Errorf("CEFRCurrent = %q, want B2 preserved on re-login", out.User.CEFRCurrent)
	}
}

func TestSignInIssuesAVerifiableToken(t *testing.T) {
	ex := &fakeExchanger{
		token:   GoogleToken{AccessToken: "at"},
		profile: GoogleProfile{Sub: "google-1", Email: "a@example.com"},
	}
	iss := NewTokenIssuer("secret", func() time.Time { return time.Unix(1_800_000_000, 0) })
	svc := NewService(ex, newFakeRepo(), newFakeSessions(), iss)

	out, err := svc.SignIn(context.Background(), "code", "uri")
	if err != nil {
		t.Fatalf("SignIn() = %v", err)
	}
	sub, err := iss.Verify(out.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if sub != out.User.ID {
		t.Errorf("token subject = %q, want %q", sub, out.User.ID)
	}
}

func TestSignInPropagatesAnExchangeFailure(t *testing.T) {
	ex := &fakeExchanger{err: errors.New("invalid_grant")}
	svc := newTestService(ex, newFakeRepo(), newFakeSessions())

	if _, err := svc.SignIn(context.Background(), "bad", "uri"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}
```

`backend/internal/auth/handler_test.go`:
```go
package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

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
```

`errGoogleRejected` is test-local: add `var errGoogleRejected = errors.New("google rejected the code")`
at the top of `handler_test.go` and import `errors`. The handler maps *any* exchange failure to 401, so
the specific error value does not matter.

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/auth/...
```
Expected: build failure, `undefined: NewService`.

- [ ] **Step 3: Implement**

`backend/internal/auth/service.go`:
```go
package auth

import (
	"context"
	"fmt"
)

// SignInResult is what the handler serialises.
type SignInResult struct {
	Token string
	User  User
}

// Service performs spec §5.1 steps 1-3: exchange the code, store the refresh
// token, issue a JWT.
type Service struct {
	google   Exchanger
	users    UserRepo
	sessions SessionStore
	tokens   *TokenIssuer
}

// NewService wires the four collaborators.
func NewService(google Exchanger, users UserRepo, sessions SessionStore, tokens *TokenIssuer) *Service {
	return &Service{google: google, users: users, sessions: sessions, tokens: tokens}
}

// SignIn exchanges code for tokens, upserts the user and opens a session.
func (s *Service) SignIn(ctx context.Context, code, redirectURI string) (SignInResult, error) {
	tok, err := s.google.Exchange(ctx, code, redirectURI)
	if err != nil {
		return SignInResult{}, fmt.Errorf("auth: exchanging code: %w", err)
	}
	profile, err := s.google.UserInfo(ctx, tok.AccessToken)
	if err != nil {
		return SignInResult{}, fmt.Errorf("auth: fetching profile: %w", err)
	}

	user, err := s.users.UpsertByGoogleID(ctx, profile.Sub, profile.Email, profile.Name, tok.RefreshToken)
	if err != nil {
		return SignInResult{}, err
	}

	jwtToken, err := s.tokens.Issue(user.ID)
	if err != nil {
		return SignInResult{}, err
	}
	// The Redis key is written last: a failure here must not leave a user
	// holding a token with no session behind it.
	if err := s.sessions.Put(ctx, user.ID, jwtToken, TokenTTL); err != nil {
		return SignInResult{}, err
	}
	return SignInResult{Token: jwtToken, User: user}, nil
}
```

`backend/internal/auth/handler.go`:
```go
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// signInRequest is the §7 POST /api/v1/auth/google body.
type signInRequest struct {
	Code        string `json:"code" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// Handler serves POST /api/v1/auth/google. Any failure on Google's side is a
// 401: the client's only sensible response is to restart the consent flow.
func Handler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signInRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		out, err := svc.SignIn(c.Request.Context(), req.Code, req.RedirectURI)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "google_auth_failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": out.Token,
			"user": gin.H{
				"id":           out.User.ID,
				"email":        out.User.Email,
				"full_name":    out.User.FullName,
				"cefr_current": out.User.CEFRCurrent,
			},
		})
	}
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/auth/... -v
```
Expected: every test in the package `--- PASS`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: sign-in service and POST /api/v1/auth/google"
```

---

### Task 8: `auth.Require()` middleware

**Files:**
- Create: `backend/internal/auth/middleware.go`
- Test: `backend/internal/auth/middleware_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/auth/middleware_test.go`:
```go
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
```

- [ ] **Step 2: Run and confirm it fails**

```sh
go test ./internal/auth/... -run Require
```
Expected: build failure, `undefined: Require`.

- [ ] **Step 3: Implement**

`backend/internal/auth/middleware.go`:
```go
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextUserID is the Gin context key holding the authenticated user's UUID.
const ContextUserID = "user_id"

// Require verifies the bearer token and that the matching Redis session is
// still present. Every per-user route in spec §7 mounts behind it.
//
// A token is accepted only when it (a) verifies against JWT_SECRET, (b) is
// within its exp window, and (c) is byte-for-byte the token stored at
// sess:{user_id}:token. (c) is what makes DEL a revocation and what makes a new
// sign-in supersede the previous token.
func Require(tokens *TokenIssuer, sessions SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			abortUnauthorized(c)
			return
		}
		userID, err := tokens.Verify(raw)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		stored, err := sessions.Get(c.Request.Context(), userID)
		if err != nil || stored != raw {
			abortUnauthorized(c)
			return
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}

// UserID returns the authenticated user set by Require, or "" outside it.
func UserID(c *gin.Context) string {
	v, ok := c.Get(ContextUserID)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
```

- [ ] **Step 4: Run and confirm it passes**

```sh
go test ./internal/auth/... -run Require -v
```
Expected: five `--- PASS` lines.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: Require() middleware with Redis-backed revocation"
```

---

### Task 9: Integration test for the upsert, and route wiring

**Files:**
- Create: `backend/internal/auth/integration_test.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Write the integration test**

`backend/internal/auth/integration_test.go`:
```go
package auth

import (
	"context"
	"os"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// These are the only tests in this package that need a database, and they skip
// without DATABASE_URL — `go test ./...` stays green with no services.
func TestIntegrationUpsertCreatesThenPreservesTheLearnerState(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()

	pg, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	if _, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	repo := NewPgUserRepo(pg.Pool)
	const gid = "google-integration-1"
	t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })
	_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)

	first, err := repo.UpsertByGoogleID(ctx, gid, "a@example.com", "A Person", "rt-1")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if first.ID == "" {
		t.Fatal("first upsert returned no id")
	}
	if first.CEFRCurrent != "A1" {
		t.Errorf("CEFRCurrent = %q, want the §3.2 default A1", first.CEFRCurrent)
	}

	// The learner progresses and sets a goal.
	if _, err := pg.Pool.Exec(ctx,
		`UPDATE users SET cefr_current = 'B2', target_goal = 'IELTS 7.0' WHERE id = $1`, first.ID); err != nil {
		t.Fatalf("simulating onboarding: %v", err)
	}

	// Re-login with no refresh token (Google omits it on silent consent).
	second, err := repo.UpsertByGoogleID(ctx, gid, "a@example.com", "Renamed Person", "")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("re-login created a new row: %q vs %q", second.ID, first.ID)
	}
	if second.CEFRCurrent != "B2" {
		t.Errorf("CEFRCurrent = %q, want B2 preserved", second.CEFRCurrent)
	}
	if second.FullName != "Renamed Person" {
		t.Errorf("FullName = %q, want the Google value refreshed", second.FullName)
	}

	var goal, refresh string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT target_goal, google_refresh_token FROM users WHERE id = $1`, first.ID).Scan(&goal, &refresh); err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if goal != "IELTS 7.0" {
		t.Errorf("target_goal = %q, want it preserved across re-login", goal)
	}
	if refresh != "rt-1" {
		t.Errorf("google_refresh_token = %q, want the stored token kept when Google sends none", refresh)
	}
}
```

- [ ] **Step 2: Confirm it skips with no database**

```sh
env -u DATABASE_URL go test ./internal/auth/... -run Integration -v
```
Expected: `--- SKIP` and `ok`.

- [ ] **Step 3: Wire the route in `cmd/api/main.go`**

After the existing Redis/Postgres setup and before `r.Run`, add:
```go
	authSvc := auth.NewService(
		auth.NewGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		auth.NewPgUserRepo(pg.Pool),
		auth.NewRedisSessionStore(rdb),
		auth.NewTokenIssuer(cfg.JWTSecret, time.Now),
	)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/google", auth.Handler(authSvc))

	// Later slices mount their routes on this group:
	//   guarded := v1.Group("", auth.Require(tokens, sessions))
```
with imports `time` and `.../internal/auth`. Keep the `TokenIssuer` and `SessionStore` in local
variables so the next slice can pass them to `auth.Require()` without re-constructing them.

- [ ] **Step 4: Confirm the whole suite is green**

```sh
go build ./... && go vet ./... && go test ./...
```
Expected: `ok` for `internal/auth`, `internal/config`, `internal/health`, `internal/store`; no `FAIL`.

- [ ] **Step 5: Commit**

```sh
cd .. && git add backend && git commit -m "auth: mount POST /api/v1/auth/google and add the upsert integration test"
```

---

### Task 10: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (the `**auth**` bullet)

- [ ] **Step 1: Replace the bullet**

```
- **auth** — Google sign-in and sessions. `POST /api/v1/auth/google` ({code, redirect_uri}) exchanges the code, fetches the OpenID profile, upserts `users` on `google_id` (refreshes email/full_name/refresh token; never resets `cefr_current` or `target_goal`; a new user gets `target_goal = ''` because §3.2 makes it NOT NULL with no default), issues an HS256 JWT and writes `sess:{user_id}:token` for 24h. `auth.Scopes` is the single consent list (`openid email profile` + `calendar.events` + `tasks`, with `access_type=offline&prompt=consent`) — the frontend must build its login URL from `auth.AuthCodeURL` so no user has to re-consent when the google slice lands. `auth.Require(tokens, sessions)` guards every per-user route: it accepts a token only if it verifies, is unexpired, and matches the stored session byte for byte, so `DEL sess:{user_id}:token` revokes and a new sign-in supersedes the old token; `auth.UserID(c)` reads the authenticated id. Needs `JWT_SECRET`, which is **absent from the spec §8 env list**. Tests are pure (httptest fakes for Google, in-memory `UserRepo`/`SessionStore`); the upsert also has a `DATABASE_URL`-gated integration test that skips.
```

- [ ] **Step 2: Verify**

```sh
grep -n 'auth.Require' harness/CODEMAP.md
```
Expected: one hit.

- [ ] **Step 3: Commit**

```sh
git add harness/CODEMAP.md && git commit -m "codemap: auth — sign-in, scopes, Require() middleware"
```

---

## Verification

Run from the worktree root.

```sh
cd backend && go build ./... && go vet ./...
# expect: no output

go test ./...
# expect: ok for internal/auth, internal/config, internal/health, internal/store; no FAIL

env -u DATABASE_URL -u REDIS_URL -u GOOGLE_CLIENT_ID -u GOOGLE_CLIENT_SECRET -u JWT_SECRET go test ./... -count=1
# expect: still ok — no test needs live services or real Google credentials

go test ./internal/auth/... -v -count=1 2>&1 | grep -c -- '--- PASS'
# expect: at least 25 (the package's pure tests)

go test ./internal/auth/... -v -count=1 2>&1 | grep -- '--- SKIP'
# expect: one line, the DATABASE_URL-gated upsert test

grep -n 'access_type' internal/auth/scopes.go
# expect one hit: "access_type":   {"offline"},

grep -n 'calendar.events\|auth/tasks' internal/auth/scopes.go
# expect two hits — the google slice's scopes are requested at first consent

grep -n 'cefr_current =\|target_goal =' internal/auth/repo.go
# expect: no hits inside the DO UPDATE clause (re-login must not reset them)

grep -n 'store.SessionKey\|store.SessionTTL' internal/auth/*.go
# expect hits in session.go and token.go — auth never hand-builds a §4 key

cd .. && python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0

git log --oneline main..HEAD
# expect 10 commits, one per task, each with the Co-Authored-By trailer

git status --short
# expect: clean
```

## Notes and open questions

- **`JWT_SECRET` is not in spec §8.** This plan makes it a required env var and says so in CODEMAP. A follow-up spec bug should add it to the §8 checklist; the plan does not edit the spec.
- **The session value is the JWT string**, not a metadata blob. §4 calls the key "Active JWT session context **and user metadata**", so a later slice may want to store JSON there. `SessionStore.Get` returning a string keeps that change small, but it is a change — flagged rather than pre-built (YAGNI).
- **No logout endpoint.** Revocation works (`SessionStore.Delete`), but §7 lists no logout route and this plan adds no API beyond the one §7 names.
- **`prompt=consent` on every sign-in** guarantees a refresh token but shows the consent screen each time. If that proves annoying, the fix is to drop `prompt=consent` and rely on the upsert's `COALESCE` to keep the stored refresh token — which this plan already implements and tests.
- **`redirect_uri` is supplied by the client.** Google validates it against the registered list, so this is safe, but it does mean the backend cannot enforce a single origin. If the human wants that, it becomes a config value and a 400 on mismatch — a small, separate change.

## Execution summary

Built exactly as the plan describes: `backend/internal/auth` (scopes, token, google, repo, session, service, handler, middleware) plus config additions, integration test, route wiring in `cmd/api/main.go`, and the CODEMAP `auth` paragraph. 13 commits on `harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions` (10 plan tasks + 3 fix commits, two of which surfaced only in CI — see below).

**Starting state:** this worktree/branch already existed with Tasks 1–2 committed and Task 3's `go get` staged (a prior partial run). Continued from there rather than re-doing that work.

**Deviations from the plan, with reasons:**
1. **Task 4 test bug fixed.** `google_test.go`'s `gotForm` was declared `map[string][]string` but the test calls `.Get()` on it, which only exists on `url.Values` (what `r.PostForm` actually is). Declared `gotForm` as `url.Values` instead — same assertions, compiles.
2. **Task 9 integration test gate fixed — the important one.** The plan's `integration_test.go` as written reads `os.Getenv("DATABASE_URL")`, the *production* variable. `internal/store`'s own integration tests (and this repo's `AGENTS.md`/CODEMAP contract) establish `TEST_DATABASE_URL`/`TEST_REDIS_URL` as the only allowed gate, specifically because these tests DROP data and because CI's `backend-integration` job only exports `TEST_DATABASE_URL`/`TEST_REDIS_URL` — never `DATABASE_URL`. As written, the test would have silently `SKIP`ped in CI and failed the "no skips" gate in `.github/workflows/ci.yml`. Changed the gate to `TEST_DATABASE_URL`, matching `internal/store/integration_test.go`'s established pattern; updated the CODEMAP `auth` paragraph to say `TEST_DATABASE_URL` instead of the plan's suggested `DATABASE_URL`-gated wording. Committed separately (`54ee414`) so the mistake and the fix are both visible in history.

**Plan documentation inaccuracy (no code change):** Task 3's verification command `go test ./internal/auth/... -run Token -v` is expected to print "six `--- PASS` lines", but Go's `-run` is a substring regex match against the test name — only `TestTokenExpiryMatchesTheRedisSessionTTL` and `TestVerifyRejectsAnExpiredToken` contain the literal substring "Token". All 6 tests in that task pass; verified with the full package run instead (`go test ./internal/auth/... -v`).

### Plan's Verification section — output

```
$ go build ./... && go vet ./...
(no output)

$ go test ./...
ok  	.../backend/internal/auth	0.744s
ok  	.../backend/internal/config	(cached)
ok  	.../backend/internal/health	(cached)
ok  	.../backend/internal/store	(cached)

$ env -u DATABASE_URL -u REDIS_URL -u GOOGLE_CLIENT_ID -u GOOGLE_CLIENT_SECRET -u JWT_SECRET go test ./... -count=1
ok  	.../backend/internal/auth	2.663s
ok  	.../backend/internal/config	1.338s
ok  	.../backend/internal/health	2.005s
ok  	.../backend/internal/store	0.724s

$ go test ./internal/auth/... -v -count=1 2>&1 | grep -c -- '--- PASS'
29   (>= 25 required)

$ go test ./internal/auth/... -v -count=1 2>&1 | grep -- '--- SKIP'
--- SKIP: TestIntegrationUpsertCreatesThenPreservesTheLearnerState (0.00s)

$ grep -n 'access_type' internal/auth/scopes.go
29:// AuthCodeURL builds the consent URL. access_type=offline plus prompt=consent
37:		"access_type":   {"offline"},

$ grep -n 'calendar.events\|auth/tasks' internal/auth/scopes.go
22:	"https://www.googleapis.com/auth/calendar.events",
23:	"https://www.googleapis.com/auth/tasks",

$ grep -n 'cefr_current =\|target_goal =' internal/auth/repo.go
(no hits — DO UPDATE clause never resets them)

$ grep -n 'store.SessionKey\|store.SessionTTL' internal/auth/*.go
hits in session.go and token.go (auth never hand-builds a §4 key)

$ python3 tools/harness/cli.py validate; echo exit=$?
exit=0

$ git log --oneline main..HEAD
13 commits (10 tasks + 3 fixes: the TEST_DATABASE_URL gate, the Migrate Locker, and CI's -p 1), each with the Co-Authored-By trailer

$ git status --short
(clean)
```

### Runtime proof (step 8)

- **Build:** `go build ./...` and `go vet ./...` clean, no warnings.
- **Whole suite, clean shell:** `env -u DATABASE_URL -u REDIS_URL -u GOOGLE_CLIENT_ID -u GOOGLE_CLIENT_SECRET -u JWT_SECRET go test ./... -count=1` → all four packages `ok`, no `FAIL`, no live services.
- **Boot + real paths:** built `cmd/api` to a binary, brought up the dev stack via a scratch `backend/.env` (`POSTGRES_PORT=5433 REDIS_PORT=6380`, so the owner's `scio3-redis-1` on 6379 was never touched — confirmed `docker ps` before and after), ran `make up`, waited for `pg_isready`/`redis-cli ping`, then started the binary with `DATABASE_URL`/`REDIS_URL` pointed at those ports plus fake `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`JWT_SECRET`. Both routes registered (`GET /healthz`, `POST /api/v1/auth/google`).
  - `GET /healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`
  - `POST /api/v1/auth/google` with `{}` → `400 {"error":"invalid_request"}`
  - `POST /api/v1/auth/google` with a bogus code → `401 {"error":"google_auth_failed"}` (the service actually reached Google's real token endpoint over the network — `GoogleClient`'s endpoints are hard-wired to `DefaultTokenURL`/`DefaultUserInfoURL` in `main.go`'s wiring, not configurable at the binary level, so this is the closest real-path exercise available without editing the plan's design; the injectable-endpoint design is exercised by the package's own `httptest` unit tests instead) and the handler correctly mapped Google's rejection to 401.
  - Confirmed the safe-failure mode: running the binary with no env vars exported → `config: DATABASE_URL is required`, exit 1, no crash, no partial state.
  - `make test-integration` against the live scratch stack: `TestIntegrationUpsertCreatesThenPreservesTheLearnerState` — `--- PASS`, alongside `internal/store`'s three integration tests, all `--- PASS`.
  - Tore down: `make down`, deleted the scratch `backend/.env`; `docker ps` afterward shows only the pre-existing `scio3-redis-1`/`scio3-mongo-1`, confirming no disturbance.

### CI

Pushed `harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions`. Two CI-only failures surfaced and were fixed, both out of the plan's named file list but a direct, necessary consequence of this plan adding the *second* package (`auth`, after `store`) with its own `TestIntegration*`:

1. **Run [35740944612](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35740944612) — FAIL.** `backend-integration` failed: `ensuring version table: ERROR: duplicate key value violates unique constraint "pg_type_typname_nsp_index"`. Root cause: `internal/store`'s and `internal/auth`'s integration-test binaries run in parallel against one shared `TEST_DATABASE_URL`/database (CI's single Postgres service container), and both call `store.Migrate`; Postgres's `CREATE TABLE/TYPE IF NOT EXISTS` is check-then-act, not atomic, under concurrency. Fixed in `4c704ba` (`store: serialize Migrate against concurrent callers on the same database`): an additive `store.Locker` interface, taken via type assertion in `Migrate` (the existing `Migrator` interface and store's fakes are unchanged), and `PgMigrator.Lock`, a Postgres advisory lock on one pinned connection for the whole migration run. Covered by new unit tests (`fakeLockingMigrator`) and `TestIntegrationConcurrentMigrateDoesNotRace` (8 concurrent callers against a real database).
2. **Run [35742036352](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35742036352) — FAIL, after fix 1.** `backend-integration` failed differently: `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` got `first run applied [], want [0001_init]`. The Locker fix stops concurrent *Migrate* calls from racing each other, but `internal/store`'s own tests also destructively `DROP`/recreate the schema around each of their own tests (`reset()`), uncoordinated with any other package — so `auth`'s test running concurrently with `store`'s `reset()` produced order-dependent results. Fixed in `75b56d3` (`ci: run integration tests with -p 1, one package at a time`): added `-p 1` to both the CI workflow's and `make test-integration`'s `go test ... -run Integration` invocation, so packages' test binaries run one at a time against the shared database (intra-package test order/parallelism is unaffected). Verified stable across three local runs against a live Postgres before pushing. (This commit's message lost a `make test-integration` reference to an unintended shell backtick expansion while writing it — the diff itself is unaffected; not amended per the repo's no-amend rule.)
3. **Run [35742567690](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35742567690) — SUCCESS.** All three jobs (`backend-unit`, `backend-integration`, `harness-tooling`) green at `75b56d3` (branch HEAD). `gh pr create` was attempted once and failed with "must be a collaborator" (the authenticated `gh` account has no write access to this repo), as expected — recorded here and skipped; the branch is pushed and ready for a human to open the PR.
