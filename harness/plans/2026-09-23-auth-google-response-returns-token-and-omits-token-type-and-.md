---
idea: harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md
status: approved
priority: high
merged: false
---
# auth/google response: answer the backend spec §6.1 sign-in body — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md`
**Goal:** Make `POST /api/v1/auth/google` return exactly the backend spec §6.1 200 body — `{access_token, token_type: "Bearer", expires_in: 86400, user: {id, email, full_name, cefr_current}}` — instead of `{token, user}`, with a test that pins the wire keys, so the approved frontend-shell slice (MVP order 9) can sign in against `main`.

**Why now (MVP-enabling, `priority: high`):** `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` reads `access_token` / `expires_in` with no fallback to `token`; its header records that it cannot merge until this lands on `main`. Ordinary plan — not a blocker of an unmerged branch (no branch touches `handler.go`; `git log --all -- backend/internal/auth/handler.go` shows only the original commit). Own worktree, own `harness/*` branch.

**Root cause (from the idea's `## Evaluation`):** the auth slice was planned against the 1st-thinking doc §7, which names the endpoint without a body; the executor chose `token` and the review checked code against that plan. The backend spec, which is the REST contract (AGENTS.md → *Reading the spec*), came later. Contract drift only — `SignInResult{Token, User}` (`backend/internal/auth/service.go:9-12`) already carries everything the §6.1 body needs; `TokenTTL` (`token.go:14`, `= store.SessionTTL = 24h`) supplies `expires_in`.

**The contract being implemented** — backend spec §6.1, `Response (200 OK)` under `POST /api/v1/auth/google` (`project-base/Adaptive English Learning Platform - Backend Technical Specification.md:251`):

```json
{"access_token": "eyJhbGciOiJKV1QiLC...", "token_type": "Bearer", "expires_in": 86400, "user": {"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "email": "user@example.com", "full_name": "Nguyen Hendrix", "cefr_current": "B1"}}
```

Rules: exactly those four top-level keys; **no `token` alias** (one contract, one shape); `expires_in` is an integer number of seconds derived from `TokenTTL` so it can never disagree with the JWT `exp` or the Redis session TTL; the test asserts the spec literal `86400` so a TTL change forces a spec conversation rather than silently changing the wire. Error bodies (`400 invalid_request`, `401 google_auth_failed`) and the request body `{code, redirect_uri}` are already to spec and do not change.

**Architecture:** replace the `gin.H` literal in `auth.Handler` with a typed `signInResponse` struct (the same pattern the later `pet` handler uses — `backend/internal/pet/handler.go:16,42`), so the contract is readable in one place. Nothing outside `handler.go` / `handler_test.go` changes; `Service`, `TokenIssuer`, middleware and sessions are untouched.

**Tech stack:** Go 1.25, Gin, `encoding/json`, `httptest`; existing in-package fakes (`fakeExchanger`, `newFakeRepo`, `newFakeSessions` in `service_test.go`).

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n` and `go test -timeout`. Sibling checks are not needed: no other endpoint in the auth slice exists (`backend/cmd/api/main.go:95` is the only auth route) and the only other `access_token`/`expires_in`/`token_type` keys in Go are Google's own token DTO in `google.go:25-28`, which is correct.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/auth/handler.go` | Modify lines 15-41: add `signInResponse` / `signInUser` structs, emit them; import `time` |
| `backend/internal/auth/handler_test.go` | Modify lines 33-64: replace `TestHandlerReturnsTokenAndUser` with `TestHandlerReturnsTheSpecSignInBody` (wire-key assertions + JWT verification) |
| `harness/CODEMAP.md` | Modify line 10 (`**auth**` bullet): name the §6.1 response fields |

---

## Tasks

### Task 1: Pin the §6.1 body in the handler test, then make the handler emit it

**Files:**
- Modify: `backend/internal/auth/handler_test.go:33-64`
- Modify: `backend/internal/auth/handler.go:15-41`

- [ ] **Step 1: Replace the shape-pinning test**

In `backend/internal/auth/handler_test.go`, delete `TestHandlerReturnsTokenAndUser` (lines 33-64) and put this in its place. The imports already present (`encoding/json`, `errors`, `net/http`, `net/http/httptest`, `strings`, `testing`, `time`, gin) are sufficient — add none.

```go
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
```

- [ ] **Step 2: Run it and confirm it fails for the right reason**

Run: `go test ./internal/auth/... -count=1 -timeout 60s -run TestHandlerReturnsTheSpecSignInBody -v`
Expected: `--- FAIL: TestHandlerReturnsTheSpecSignInBody` with (at least) `body has a "token" key`, `body is missing "access_token"`, `body is missing "token_type"`, `body is missing "expires_in"`, `body has 2 top-level keys, want 4`, and `access_token does not verify` (empty string). If it passes, stop — you are not on the code this plan was written against.

- [ ] **Step 3: Emit the §6.1 body from the handler**

Replace the whole of `backend/internal/auth/handler.go` from line 15 (`// Handler serves …`) to the end with:

```go
// signInResponse is the backend spec §6.1 200 body for POST /api/v1/auth/google.
// expires_in is derived from TokenTTL so it can never disagree with the JWT exp
// or the Redis session TTL; token_type is always "Bearer" — the value auth.Require
// expects in the Authorization header.
type signInResponse struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   int        `json:"expires_in"`
	User        signInUser `json:"user"`
}

type signInUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	FullName    string `json:"full_name"`
	CEFRCurrent string `json:"cefr_current"`
}

// Handler serves POST /api/v1/auth/google (backend spec §6.1). Any failure on
// Google's side is a 401: the client's only sensible response is to restart the
// consent flow.
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

		c.JSON(http.StatusOK, signInResponse{
			AccessToken: out.Token,
			TokenType:   "Bearer",
			ExpiresIn:   int(TokenTTL / time.Second),
			User: signInUser{
				ID:          out.User.ID,
				Email:       out.User.Email,
				FullName:    out.User.FullName,
				CEFRCurrent: out.User.CEFRCurrent,
			},
		})
	}
}
```

and add `"time"` to the import block (keep it in the stdlib group, before the blank line and the gin import):

```go
import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)
```

Lines 1-13 (`package auth`, `signInRequest`) are unchanged — the request body already matches §6.1.

- [ ] **Step 4: Run the auth package and confirm green**

Run: `gofmt -l ./internal/auth && go vet ./internal/auth/... && go test ./internal/auth/... -count=1 -timeout 120s -v 2>&1 | grep -E '^(--- |ok|FAIL|PASS)'`
Expected: `gofmt -l` prints nothing; `--- PASS: TestHandlerReturnsTheSpecSignInBody`, `--- PASS: TestHandlerRejectsAMissingCode`, `--- PASS: TestHandlerReturns401WhenGoogleRejectsTheCode`, every other auth test `--- PASS` (the `TestIntegration*` one prints `--- SKIP` locally without `TEST_DATABASE_URL`; that is expected), and a final `ok`.

- [ ] **Step 5: Confirm `token` is gone from the wire and the whole module still builds**

Run: `grep -n '"token"' internal/auth/handler.go internal/auth/handler_test.go; go build ./... && go test ./... -count=1 -timeout 300s 2>&1 | grep -E '^(ok|FAIL|---)'`
Expected: the `grep` prints exactly one line — the test's negative assertion (`keys["token"]`) in `handler_test.go` — and nothing from `handler.go`; every package prints `ok`, no `FAIL`.

- [ ] **Step 6: Commit**

```bash
git add internal/auth/handler.go internal/auth/handler_test.go
git commit -m "auth: POST /api/v1/auth/google answers the spec §6.1 body (access_token, token_type, expires_in)"
```

### Task 2: Name the response fields in CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md:10` (the `**auth**` bullet)

- [ ] **Step 1: Confirm the anchor sentence exists exactly once** (run from the worktree root)

Run: `grep -c 'issues an HS256 JWT and writes `sess:{user_id}:token` for 24h\.' harness/CODEMAP.md`
Expected: `1`

- [ ] **Step 2: Extend the sentence** (worktree root; exact-string replace, no regex)

```bash
python3 - <<'PY'
from pathlib import Path
p = Path("harness/CODEMAP.md")
s = p.read_text()
old = "issues an HS256 JWT and writes `sess:{user_id}:token` for 24h."
new = ("issues an HS256 JWT, writes `sess:{user_id}:token` for 24h, and answers the backend spec §6.1 body "
       "`{access_token, token_type: \"Bearer\", expires_in: 86400, user: {id, email, full_name, cefr_current}}` "
       "— `expires_in` is `int(TokenTTL / time.Second)`, and there is no `token` key (contract fix 2026-09-23).")
assert s.count(old) == 1, s.count(old)
p.write_text(s.replace(old, new))
PY
```

- [ ] **Step 3: Verify**

Run: `grep -n 'access_token, token_type' harness/CODEMAP.md && git diff --stat -- harness/CODEMAP.md`
Expected: one hit on line 10; `1 file changed, 1 insertion(+), 1 deletion(-)`.

- [ ] **Step 4: Commit**

```bash
git add harness/CODEMAP.md
git commit -m "codemap: auth — name the §6.1 sign-in response fields"
```

---

## Verification

From `backend/` in the worktree:

```sh
gofmt -l ./internal/auth                                   # expect: no output
go build ./... && go vet ./...                             # expect: exit 0
go test ./internal/auth/... -count=1 -timeout 120s -run 'TestHandler' -v 2>&1 | grep -c -- '--- PASS'
# expect: 3   (TheSpecSignInBody, RejectsAMissingCode, Returns401WhenGoogleRejectsTheCode)
go test ./... -count=1 -timeout 300s 2>&1 | grep -c '^FAIL'
# expect: 0
grep -n '"token"' internal/auth/handler.go                 # expect: no output
grep -n 'json:"access_token"\|json:"token_type"\|json:"expires_in"' internal/auth/handler.go
# expect: the three signInResponse fields
```

From the worktree root:

```sh
grep -n 'access_token, token_type' harness/CODEMAP.md     # expect: one hit on the **auth** bullet
git diff --stat main -- backend harness/CODEMAP.md
# expect: 3 files changed (handler.go, handler_test.go, CODEMAP.md); nothing else
python3 tools/harness/cli.py validate; echo exit=$?         # expect: exit=0
```

CI: after pushing, `gh run list --branch <branch>` must show `backend-unit`, `backend-integration` and `harness-tooling` green — this change adds no `TestIntegration*`, so the integration job's PASS count is unchanged.

**Follow-up for the reviewer / orchestrator, not for this executor:** once this merges to `main`, the *Merge blocker* paragraph at the top of `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` and its matrix row ("live on `main`, wrong shape") are stale — that plan's row should read "live on `main`" before frontend-shell is reviewed.

## Execution summary
