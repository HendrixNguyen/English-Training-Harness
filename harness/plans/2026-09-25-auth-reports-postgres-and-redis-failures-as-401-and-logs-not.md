---
idea: harness/ideas/_inbox/auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md
status: approved
priority: medium
merged: false
---
# auth: a Google rejection is 401, an outage is 5xx, and every failure is logged — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md`
(`harness/ideas/_inbox/service-and-require-failure-paths-are-untested-the-fakes-err.md` was rejected as its duplicate on 2026-09-23; its three tests are Tasks 2–4 here.)

**Goal:** A learner is told "sign in again" only when Google actually rejected them or their session is genuinely gone. A Postgres or Redis failure answers 5xx, keeps the existing sessions valid, and leaves one log line an operator can act on — never a token.

**Why now (`priority: medium`):** Confirmed on `origin/main` today: `backend/internal/auth/handler.go` maps every `SignIn` error to `401 google_auth_failed`; `backend/internal/auth/middleware.go` `Require` does `if err != nil || stored != raw { abortUnauthorized(c) }`; `grep -n 'log\.' backend/internal/auth/*.go` (non-test) prints nothing. A Redis blip therefore answers `401` to every guarded call, and the PWA's API client treats a `401` as "signed out" (CODEMAP `shell`: `401 → sign out + /login`) — one Redis hiccup logs the whole user base out, with `prompt=consent` on the way back in, and the operator sees only `[GIN] 401` lines. The `users_email_key` collision is a permanent, silent lockout for that account. This is the highest-impact item left in the inbox (AGENTS.md standing priority: happy-path breakage outranks tidiness).

**Root cause:** the auth slice (2026-09-22) returned one undifferentiated `error` from every leg of `SignIn` and from `SessionStore.Get`, and both call sites chose the client-facing answer that fits the *expected* failure (Google said no; the key is gone) without a way to tell the *unexpected* ones apart.

**Design decisions:**
1. **Sentinels, not error-string matching.** `ErrGoogleRejected` is raised by the `Exchanger` leg (any 4xx from Google, or a 2xx with no access token / no `sub`), `ErrEmailTaken` by the repo (`users_email_key`), and `ErrSessionStoreUnavailable` by `RedisSessionStore` for any Redis error that is not `redis.Nil`. Every wrap keeps `%w`, so `errors.Is` works at the handler.
2. **Status mapping** (`POST /api/v1/auth/google`): `ErrGoogleRejected` → `401 google_auth_failed` (unchanged wire behaviour for the one case it was right about); `ErrEmailTaken` → `409 email_in_use`; `ErrSessionStoreUnavailable` → `503 unavailable`; anything else (repo transport, JWT signing, a Google 5xx or network failure) → `500 internal_error` (the code the other packages already use — `grep -rn '"internal_error"' backend/internal` finds 7). `Require`: `ErrNoSession`, a missing/bad token, or a mismatch → `401 unauthorized` (unchanged); any other `Get` error → `503 unavailable` and **the session stays valid**.
3. **One log line per failure, at the handler.** `log.Printf("auth: sign-in failed: %v", err)` / `log.Printf("auth: session store unavailable for user %s: %v", userID, err)`. The error chain carries the Google id (the service wraps the repo error with it) but never the JWT, the Google access/refresh tokens or `JWT_SECRET` — `GoogleClient.doJSON` embeds Google's raw *error* body (no tokens in an error body) and nothing else in the chain formats a token. Task 4 pins this with a captured log.
4. **The client sees nothing new that leaks.** Bodies stay opaque codes; the frontend already renders `StateBlock state="error"` with a retry for any non-401 `ApiError` (CODEMAP `shell`), so a `503` from `Require` is a retryable error screen, not a sign-out. No frontend change.
5. **Email collision is a 409, not an update.** Re-pointing an email at a different `google_id` is an account merge nobody designed; `409 email_in_use` plus the log line makes it diagnosable. The frontend's `/login` shows its generic failure copy for a 409 today — acceptable for this branch; a dedicated message is a frontend follow-up, noted in *Notes*.

**Tech stack:** Go 1.25, Gin, pgx/v5 (`github.com/jackc/pgx/v5/pgconn` for `*pgconn.PgError`, already a transitive import of the module), go-redis/v9, stdlib `log`. No new dependencies.

**Run every command from the worktree root** unless a step says otherwise; `backend/` commands say so. `rg` is not installed — use `grep -n`. Unit tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front so nothing touches a live service.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/auth/errors.go` | **new** — `ErrGoogleRejected`, `ErrEmailTaken`, `ErrSessionStoreUnavailable` with doc comments |
| `backend/internal/auth/google.go` | `doJSON` wraps a 4xx with `ErrGoogleRejected`; `Exchange`/`UserInfo` wrap their "missing field" errors with it |
| `backend/internal/auth/google_test.go` | 4xx → `errors.Is(ErrGoogleRejected)`; 5xx → not |
| `backend/internal/auth/session.go` | `RedisSessionStore.Get`/`Put`/`Delete` wrap non-`redis.Nil` errors with `ErrSessionStoreUnavailable` |
| `backend/internal/auth/session_test.go` | `fakeSessions` honours `err` on `Delete` too (consistency, from the folded finding) |
| `backend/internal/auth/repo.go` | `UpsertByGoogleID` maps `23505` on `users_email_key` to `ErrEmailTaken` |
| `backend/internal/auth/integration_test.go` | new `TestIntegrationUpsertRejectsAnEmailOwnedByAnotherGoogleAccount` |
| `backend/internal/auth/service.go` | wraps the repo error with the google id; no behaviour change |
| `backend/internal/auth/service_test.go` | repo failure → error, no session written; `Put` failure → error, no partial state |
| `backend/internal/auth/handler.go` | the status mapping of decision 2; the log line of decision 3 |
| `backend/internal/auth/handler_test.go` | 401 / 409 / 503 / 500 table; captured log contains no token |
| `backend/internal/auth/middleware.go` | `Require` distinguishes `ErrNoSession` from a transport error |
| `backend/internal/auth/middleware_test.go` | transport error → 503, session untouched |
| `harness/CODEMAP.md` | `auth` paragraph: the four outcomes and the logging rule |

---

## Tasks

### Task 1: Sentinels and the Google leg

**Files:**
- Create: `backend/internal/auth/errors.go`
- Modify: `backend/internal/auth/google.go`, `backend/internal/auth/google_test.go`

- [ ] **Step 1: Write the failing tests** in `google_test.go`. Look at the existing four tests for the `httptest.Server` shape and add:
  - `TestExchangeWrapsAGoogle4xxAsRejected`: server answers `400 {"error":"invalid_grant"}` → `err != nil && errors.Is(err, ErrGoogleRejected)`; the error string still contains `invalid_grant` (the operator needs Google's reason).
  - `TestExchangeDoesNotCallAGoogle5xxRejected`: server answers `502` → `err != nil && !errors.Is(err, ErrGoogleRejected)`.
  - `TestUserInfoWithoutASubIsRejected`: `200 {"email":"a@example.com"}` → `errors.Is(err, ErrGoogleRejected)`.
- [ ] **Step 2: Run them red:** from `backend/`: `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/auth/ -run 'TestExchange|TestUserInfo' -count=1` → the three new tests fail to compile (`ErrGoogleRejected` undefined).
- [ ] **Step 3: Create `errors.go`:**

```go
package auth

import "errors"

// ErrGoogleRejected means Google refused the code or the access token (any
// 4xx from the token or userinfo endpoint, or a 2xx without the fields sign-in
// needs). The only sensible client response is to restart the consent flow,
// so Handler answers 401. Every other failure is ours and answers 5xx.
var ErrGoogleRejected = errors.New("auth: google rejected the sign-in")

// ErrEmailTaken means users.email already belongs to a different google_id
// (§3.2: email is UNIQUE; the upsert conflicts on google_id only). Nothing
// merges accounts, so Handler answers 409 and logs it.
var ErrEmailTaken = errors.New("auth: email already belongs to another google account")

// ErrSessionStoreUnavailable wraps a Redis failure that is not "key absent".
// Require answers 503 for it and leaves the session alone: a Redis blip must
// not sign anyone out.
var ErrSessionStoreUnavailable = errors.New("auth: session store unavailable")
```

- [ ] **Step 4: Make the tests pass** in `google.go`. In `doJSON`, replace the non-2xx branch with:

```go
	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		return fmt.Errorf("%w: %s returned %d: %s", ErrGoogleRejected, req.URL.Host, resp.StatusCode, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("auth: %s returned %d: %s", req.URL.Host, resp.StatusCode, string(body))
	}
```

  and wrap the two "missing field" errors: `fmt.Errorf("%w: Google returned no access token", ErrGoogleRejected)` and `fmt.Errorf("%w: Google profile is missing sub or email", ErrGoogleRejected)`. `Exchange`/`UserInfo` already return `doJSON`'s error unchanged, so `%w` survives.
- [ ] **Step 5: Run green:** the same command → `ok`. Then the whole package: `env -u … go test ./internal/auth/ -count=1` → `ok` (the existing `TestExchange…` tests only assert `err != nil`, so they stay green).
- [ ] **Step 6: Commit:** `git add backend/internal/auth/errors.go backend/internal/auth/google.go backend/internal/auth/google_test.go && git commit -m "auth: ErrGoogleRejected marks a 4xx or a malformed reply from Google"`.

### Task 2: The session store names an outage

**Files:**
- Modify: `backend/internal/auth/session.go`, `backend/internal/auth/session_test.go`

- [ ] **Step 1: Write the failing test.** `session_test.go` builds a `RedisSessionStore` over a real client in `TestRedisSessionStore…` (gated, skips locally) and defines `fakeSessions`. Add a pure test using a `redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})` (nothing listens on port 1; the dial fails at once) —`TestGetOnAnUnreachableRedisIsUnavailableNotNoSession`: `_, err := s.Get(ctx, "u1")` → `errors.Is(err, ErrSessionStoreUnavailable) && !errors.Is(err, ErrNoSession)`. Use a `context.WithTimeout(…, 2*time.Second)` so a firewall that drops instead of refusing cannot hang the suite.
- [ ] **Step 2: Run red:** `env -u … go test ./internal/auth/ -run TestGetOnAnUnreachableRedis -count=1` → fails (`errors.Is` false: today the error is `auth: reading session: …`).
- [ ] **Step 3: Make it pass** in `session.go`: in `Get`, the `if err != nil` branch returns `fmt.Errorf("%w: reading session: %v", ErrSessionStoreUnavailable, err)`; `Put` → `"%w: storing session: %v"`; `Delete` → `"%w: deleting session: %v"`. `redis.Nil` → `ErrNoSession` is unchanged.
- [ ] **Step 4: Make `fakeSessions.Delete` honour `err`** (the folded finding): `if f.err != nil { return f.err }` before the map delete, matching `Put`/`Get`.
- [ ] **Step 5: Run green:** `env -u … go test ./internal/auth/ -count=1` → `ok`.
- [ ] **Step 6: Commit:** `git commit -am "auth: session store wraps Redis failures as ErrSessionStoreUnavailable"`.

### Task 3: The repo names an email collision; the service carries the google id

**Files:**
- Modify: `backend/internal/auth/repo.go`, `backend/internal/auth/integration_test.go`, `backend/internal/auth/service.go`, `backend/internal/auth/service_test.go`

- [ ] **Step 1: Write the failing integration test** in `integration_test.go`, next to `TestIntegrationUpsertCreatesThenPreservesTheLearnerState` (copy its setup: gate, `store.Migrate`, the `secrets.Box`, the truncate/cleanup it uses):
  `TestIntegrationUpsertRejectsAnEmailOwnedByAnotherGoogleAccount`: upsert `("google-a", "shared@example.com", …)` → ok; upsert `("google-b", "shared@example.com", …)` → `errors.Is(err, ErrEmailTaken)` and the error string contains `google-b`; a third upsert of `google-a` still succeeds (the collision changed nothing).
  It skips without `TEST_DATABASE_URL`, like its sibling; CI's `backend-integration` counts `TestIntegration*` functions and will require it to pass (13 after this branch, was 12).
- [ ] **Step 2: Write the failing unit tests** in `service_test.go` (the folded finding's first two): `TestSignInFailsAndWritesNoSessionWhenTheRepoFails` — `fakeRepo{err: errors.New("pg down")}` → `SignIn` returns an error wrapping it, `sess.Get(ctx, anyID)` finds nothing; `TestSignInFailsWhenTheSessionWriteFails` — `fakeSessions{err: …}` → error; `errors.Is(err, fakeErr)`.
- [ ] **Step 3: Run red:** `env -u … go test ./internal/auth/ -run 'TestSignInFails' -count=1` → the two unit tests fail or fail to compile (`fakeRepo`'s `err` field exists but `SignIn`'s wrapping is what the `errors.Is` checks; if they pass already, keep them — they are the regression tests the folded idea asked for and were missing).
- [ ] **Step 4: Make them pass.** In `repo.go`, after the `QueryRow(...).Scan` error:

```go
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return User{}, fmt.Errorf("%w: google_id %s", ErrEmailTaken, googleID)
		}
		return User{}, fmt.Errorf("auth: upserting user: %w", err)
	}
```

  (imports: `errors`, `github.com/jackc/pgx/v5/pgconn`). In `service.go`, wrap the repo error so the operator sees who: `return SignInResult{}, fmt.Errorf("auth: upserting google_id %s: %w", profile.Sub, err)`; leave the Google-leg wraps as they are (they already `%w`).
- [ ] **Step 5: Run green, unit and integration.** Unit: `env -u … go test ./internal/auth/ -count=1` → `ok`. Integration, from `backend/` with an isolated stack (AGENTS.md worktree rule):

```bash
COMPOSE_PROJECT_NAME=auth5xx POSTGRES_PORT=55501 REDIS_PORT=56501 docker compose up -d --wait
TEST_DATABASE_URL='postgres://english:english@localhost:55501/english?sslmode=disable' TEST_REDIS_URL='redis://localhost:56501/0' go test ./internal/auth/ -run Integration -count=1 -v -p 1
COMPOSE_PROJECT_NAME=auth5xx docker compose down
```

  Expected: `--- PASS` for both `TestIntegration*` in `auth`, no `--- SKIP`.
- [ ] **Step 6: Commit:** `git commit -am "auth: an email owned by another google_id is ErrEmailTaken; service names the google id"`.

### Task 4: Handler and Require map and log

**Files:**
- Modify: `backend/internal/auth/handler.go`, `backend/internal/auth/handler_test.go`, `backend/internal/auth/middleware.go`, `backend/internal/auth/middleware_test.go`

- [ ] **Step 1: Write the failing handler tests** in `handler_test.go`, using the `newAuthRouter`/`postJSON` helpers and `fakeExchanger`/`fakeRepo`/`fakeSessions`:
  - a table `TestHandlerMapsEachFailureToItsStatus` with rows: exchanger `err: fmt.Errorf("%w: 400", ErrGoogleRejected)` → `401 {"error":"google_auth_failed"}`; repo `err: ErrEmailTaken` → `409 {"error":"email_in_use"}`; sessions `err: fmt.Errorf("%w: dial", ErrSessionStoreUnavailable)` → `503 {"error":"unavailable"}`; repo `err: errors.New("pg down")` → `500 {"error":"internal_error"}`; exchanger `err: errors.New("dial tcp: i/o timeout")` (transport, not rejected) → `500`.
  - `TestHandlerLogsTheFailureWithoutTheTokens`: `var buf bytes.Buffer; log.SetOutput(&buf); t.Cleanup(func() { log.SetOutput(os.Stderr) })`; drive the `500` row with `fakeExchanger{token: GoogleToken{AccessToken: "at-secret", RefreshToken: "rt-secret"}, …}` and a failing repo; assert `buf.String()` contains `auth: sign-in failed` and `google-1` (the sub) and does **not** contain `at-secret`, `rt-secret` or the JWT secret string passed to `NewTokenIssuer`.
- [ ] **Step 2: Write the failing middleware tests** in `middleware_test.go`: `TestRequireAnswers503WhenTheSessionStoreIsDown` — `fakeSessions{err: fmt.Errorf("%w: dial", ErrSessionStoreUnavailable)}` with a valid token → `503 {"error":"unavailable"}`; and `TestRequireStillAnswers401ForAMissingSession` — `fakeSessions` with no entry → `401` (pins that `ErrNoSession` did not move).
- [ ] **Step 3: Run red:** `env -u … go test ./internal/auth/ -run 'TestHandlerMaps|TestHandlerLogs|TestRequireAnswers503|TestRequireStillAnswers401' -count=1` → the 409/503/500 rows and the 503 middleware test fail (everything is 401 today).
- [ ] **Step 4: Make them pass.** `handler.go`:

```go
		out, err := svc.SignIn(c.Request.Context(), req.Code, req.RedirectURI)
		if err != nil {
			log.Printf("auth: sign-in failed: %v", err) // never a token: see errors.go and the handler test
			switch {
			case errors.Is(err, ErrGoogleRejected):
				c.JSON(http.StatusUnauthorized, gin.H{"error": "google_auth_failed"})
			case errors.Is(err, ErrEmailTaken):
				c.JSON(http.StatusConflict, gin.H{"error": "email_in_use"})
			case errors.Is(err, ErrSessionStoreUnavailable):
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
			return
		}
```

  and rewrite the `Handler` doc comment: "Google rejected → 401; the email belongs to another account → 409; the session store is down → 503; anything else → 500. Every failure is logged once." `middleware.go`:

```go
		stored, err := sessions.Get(c.Request.Context(), userID)
		switch {
		case errors.Is(err, ErrNoSession):
			abortUnauthorized(c)
			return
		case err != nil:
			log.Printf("auth: session store unavailable for user %s: %v", userID, err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
			return
		case stored != raw:
			abortUnauthorized(c)
			return
		}
```

  Extend the `Require` doc comment with the (d) clause: "a session-store failure that is not 'absent' is 503, not 401 — a Redis blip must not sign anyone out."
- [ ] **Step 5: Run green:** `env -u … go test ./internal/auth/ -count=1` → `ok`; then `cd backend && make check` → fmt-check silent, vet silent, every package `ok` under `-race`.
- [ ] **Step 6: Commit:** `git commit -am "auth: 401 only for a Google rejection or a missing session; outages are 5xx and logged"`.

### Task 5: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`auth` paragraph only)

- [ ] **Step 1:** In the `auth` bullet, after the sentence that ends "answers the backend spec §6.1 body … (contract fix 2026-09-23)", add: "Failure mapping (2026-09-25): `ErrGoogleRejected` (any Google 4xx, or a 2xx missing the access token / `sub`) → `401 google_auth_failed`; `ErrEmailTaken` (`users_email_key` — §3.2 makes email UNIQUE but the upsert conflicts on `google_id`) → `409 email_in_use`; `ErrSessionStoreUnavailable` → `503 unavailable`; anything else → `500 internal_error`; each failure is logged once with the google id and never a token." In the `Require` sentence, after "matches the stored session byte for byte", add: "— and a session-store error that is not `ErrNoSession` answers `503 unavailable` with the session left intact, so a Redis blip never signs anyone out (the PWA treats only `401` as sign-out)".
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → exit 0. Commit: `git commit -am "harness: CODEMAP auth failure mapping"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/auth/ -count=1 -race -v 2>&1 | grep -c '^--- PASS'
# expect: ≥ 40 (32 `func Test` today of which the two gated ones skip, plus the 10 new pure tests: 3 google, 1 session, 2 service, 2 handler, 2 middleware); 0 FAIL
grep -rn '"error": *"' internal/auth/*.go | grep -v _test | grep -o '"[a-z_]*"' | sort -u
# expect: "email_in_use" "google_auth_failed" "internal_error" "invalid_request" "unauthorized" "unavailable"
grep -n 'log\.Printf' internal/auth/handler.go internal/auth/middleware.go
# expect: exactly one line in each
grep -n 'Token\|Secret' internal/auth/handler.go | grep 'log\.'
# expect: no output (no token formatted into a log call)
grep -c '^func TestIntegration' internal/auth/integration_test.go
# expect: 2
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
cd ..
grep -n 'email_in_use\|ErrSessionStoreUnavailable' harness/CODEMAP.md
# expect: both in the auth paragraph
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output (the branch touches no other harness/ file)
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0

# The outer loop: after `git push -u origin <branch>`,
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration (13 TestIntegration* pass, 0 skip), harness-tooling, frontend all green
```

Live proof (optional, record in the summary if run): boot the API against the isolated stack, stop Redis (`docker compose -p auth5xx stop redis`), call `GET /api/v1/quests/daily` with a valid token → `503 {"error":"unavailable"}` and one `auth: session store unavailable for user …` log line; start Redis again → the same token answers `200`/`404` as before, no re-login.

## Notes and open questions

- **Why 500 and not 503 for a repo failure?** The two codes are already distinguished by the client only as "not 401"; the split is for the operator reading `[GIN]` lines: 503 means "a dependency we poll in `/healthz` is down", 500 means "look at the log line above". Keep it.
- **Frontend copy for `409 email_in_use`:** `/login` shows its generic "Không đăng nhập được. Thử lại." for any non-2xx today. A distinct message ("Email này đã gắn với một tài khoản Google khác") is a one-line frontend follow-up; not this branch — file it with the reviewer if it matters.
- **The 4xx rule includes 429.** A Google rate limit is "rejected" under decision 1 and answers 401; it is rare enough on a per-user consent exchange that a dedicated branch is not worth its test. Say so in the `doJSON` comment.
- **Not touched:** the single-session model (`signing-in-on-a-second-device…` is planned next on this package; it edits `session.go`'s key shape, so land this first).
- **Out of scope:** Google's raw error body in the log is fine (no tokens in an OAuth error body); scrubbing it is not needed.
