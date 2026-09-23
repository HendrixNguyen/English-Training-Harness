---
idea: harness/ideas/_inbox/auth-google-response-returns-token-and-omits-token-type-and-.md
status: done
priority: high
merged: true
branch: harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-
worktree: .worktrees/auth-google-response-returns-token-and-omits-token-type-and-
pr: none
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

**Status: done.** Executed exactly as written, both tasks, no scope changes, no plan deviations.

Branch: `harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-`
Worktree: `.worktrees/auth-google-response-returns-token-and-omits-token-type-and-`
Commits: `67b1597` (auth: spec §6.1 body), `6566205` (codemap update).

### Task 1 — handler + test

Replaced `TestHandlerReturnsTokenAndUser` with `TestHandlerReturnsTheSpecSignInBody` exactly as specified. Ran it before the handler change and confirmed it failed for the documented reasons — `token` key present, `access_token`/`token_type`/`expires_in` missing, 2 top-level keys not 4, and (unfiltered `go test -v` output, since the `rtk` proxy hook condenses `go test`'s default output) `access_token does not verify as the session JWT: ... token contains an invalid number of segments`. Then replaced the handler body with the typed `signInResponse`/`signInUser` structs per the plan, added the `time` import. `TokenTTL` (`internal/auth/token.go:14`, `= store.SessionTTL = 24h`) was used as specified — not hard-coded.

### Task 2 — CODEMAP

Extended the anchor sentence exactly as specified; `git diff --stat` showed `1 file changed, 1 insertion(+), 1 deletion(-)` as expected.

### Verification (plan's section, `backend/`)

```
$ gofmt -l ./internal/auth                                   # (no output)
$ go build ./... && go vet ./...                             # exit=0
$ go test ./internal/auth/... -count=1 -timeout 120s -run 'TestHandler' -v 2>&1 | grep -c -- '--- PASS'
3
$ go test ./... -count=1 -timeout 300s 2>&1 | grep -c '^FAIL'
0
$ grep -n '"token"' internal/auth/handler.go        # (no output)
$ grep -n 'json:"access_token"\|json:"token_type"\|json:"expires_in"' internal/auth/handler.go
21:	AccessToken string     `json:"access_token"`
22:	TokenType   string     `json:"token_type"`
23:	ExpiresIn   int        `json:"expires_in"`
```

Worktree root:

```
$ grep -n 'access_token, token_type' harness/CODEMAP.md     # one hit, the auth bullet
$ git diff --stat main -- backend harness/CODEMAP.md
 backend/internal/auth/handler.go      | 40 +++++++++++++++++++++-------
 backend/internal/auth/handler_test.go | 50 +++++++++++++++++++++++++++++------
 harness/CODEMAP.md                    |  2 +-
 3 files changed, 74 insertions(+), 18 deletions(-)
$ python3 tools/harness/cli.py validate; echo exit=$?
exit=0
```

### Runtime proof (executor role's Definition of done)

1. **Builds** — `go build ./...` exit 0 (above).
2. **Whole suite** — unit suite: `go test ./... -count=1 -timeout 300s` → all 7 packages `ok`, 0 `FAIL` (above). Integration suite: brought up a scratch Postgres/Redis stack (`COMPOSE_PROJECT_NAME=authfix`, `POSTGRES_PORT=5443`, `REDIS_PORT=6391`, scratch `backend/.env`), ran `make test-integration` (which is `go test ./... -count=1 -v -run Integration -p 1`) against it:
   ```
   --- PASS: TestIntegrationRateLimiterAllowsFiveThenBlocks
   --- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState   (auth)
   --- PASS: TestIntegrationEnsureCreatesExactlyOnePetRow
   --- PASS: TestIntegrationDailyAndProgressAgainstRealServices
   --- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
   --- PASS: TestIntegrationConcurrentMigrateDoesNotRace
   --- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser
   --- PASS: TestIntegrationRedisRoundTrip
   ```
   All PASS, none skipped. Then `docker compose -p authfix down` and deleted the scratch `.env`; `docker ps` afterward showed only pre-existing, unrelated containers (`scio3-redis-1`, `scio3-mongo-1`).
3. **Boots and answers a real request** — the full `cmd/api` binary needs live Postgres/Redis plus real Google OAuth credentials (`GOOGLE_CLIENT_ID`/`SECRET`), none available here, and `SignIn` calls Google for real, so it can't demonstrate a *successful* sign-in body. Instead built and ran a temporary, uncommitted program (`backend/cmd/verifyauth_tmp/main.go`, deleted before finishing) that wires the real, unmodified `auth.Handler`/`auth.NewService`/`auth.NewTokenIssuer` into a real `gin` server bound to `:18080`, with in-memory fakes for `Exchanger`/`UserRepo`/`SessionStore` (same shape as the test fakes, but exported types so they compile from outside the package). Started it as a background process, then ran a real `curl` against it:
   ```
   $ curl -sS --max-time 10 -X POST http://localhost:18080/api/v1/auth/google \
       -H 'Content-Type: application/json' \
       -d '{"code":"the-code","redirect_uri":"https://app/cb"}'
   HTTP/1.1 200 OK
   {"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJpZC1nb29nbGUtMSIsImV4cCI6MTc5MDIxOTg0NiwiaWF0IjoxNzkwMTMzNDQ2fQ.suV6TxNxxu626UvVp_Js8sL9JVThBP603GoqXb9xjsI","token_type":"Bearer","expires_in":86400,"user":{"id":"id-google-1","email":"a@example.com","full_name":"A Person","cefr_current":"A1"}}
   ```
   Decoded the JWT payload: `{"sub": "id-google-1", "exp": 1790219846, "iat": 1790133446}` — subject matches `user.id`. Exactly four top-level keys, no `token`. Killed the process, deleted the temp binary, log files, and the `backend/cmd/verifyauth_tmp/` directory; `git status --short` was clean afterward and `pgrep -fl verifyauth_tmp_bin` found nothing.
4. **Every documented command works as documented** — every command above (`gofmt`, `go build`, `go vet`, `go test`, `make test-integration`, `docker compose up -d --wait`/`down`) was run exactly as written and behaved as documented.
5. **Reverted-handler check** — copied the pre-fix `handler.go` (commit `f37175a`) back in, ran `TestHandlerReturnsTheSpecSignInBody`: it failed with the same six diagnostics as the original TDD red run (`FAIL`, exit 1). Confirms the test is load-bearing on wire keys and the JWT subject, not just Go struct tags. Restored the fixed `handler.go` afterward; `git status --short` showed no diff and the test passed again.
6. **No leftover process or container** — confirmed via `pgrep -fl verifyauth_tmp_bin` (empty) and `docker ps` (only pre-existing `scio3-*` containers, no `authfix-*`).

### PR

`gh pr create` failed as anticipated: `pull request create failed: GraphQL: must be a collaborator (createPullRequest)` — the `gh` CLI here is authenticated as `hendrixnguyen-optisigns` (a work account), not a collaborator on this personal GitHub repo. Attempted once, noted, moved on; branch is pushed so a human with repo access can open the PR.

### CI

Pushed `harness/2026-09-23-high-auth-google-response-returns-token-and-omits-token-type-and-`; watched the triggered run to completion:

Run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35813839556 — **all green**: `backend-unit` (19s), `backend-integration` (46s), `harness-tooling` (6s).

### Deviations from the plan

- The plan's Verification section did not include a live-request boot check or `make test-integration`; both were added to satisfy the executor role's Definition of done (items 2 and 3), using a temporary uncommitted verification program since the real `cmd/api` binary cannot complete a Google sign-in without live external credentials. No repo files besides the two named in the plan were touched.
- No PR opened (see above) — branch pushed only.
