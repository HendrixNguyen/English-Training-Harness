---
idea: harness/ideas/2026-09-25-run-01/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md
status: done
priority: high
merged: false
design: harness/designs/stay-signed-in.md
branch: harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve
worktree: .worktrees/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve
---
# Stay signed in: `auth.Require` renews a session below half-life and the PWA adopts the new token silently; an expired session says why on `/login` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F1** of 2026-09-26. **Estimate:** 4 h. **Branch:** `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve`.

**Design:** `harness/designs/stay-signed-in.md` — frontend tasks cite its sections; `## Verification` repeats its §8 acceptance list.
**Idea:** `harness/ideas/2026-09-25-run-01/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`

**Goal:** A learner who opens the app at least once every 24 hours never sees Google's consent screen again: every authenticated request whose token has less than half of `TokenTTL` left gets a fresh token in `X-Session-Token` (+ `X-Session-Expires-In`), the client adopts it without any visible change, and a learner away for more than 24 h lands on `/login?reason=expired` with one sentence explaining why. Revocation (`DEL sess:*`) and single-session supersession are unchanged.

**Architecture:** Backend — `auth` gains `renew.go` (one function, one call site in `Require` after the byte-for-byte check) and `TokenIssuer.Claims` (subject + expiry); `middleware.CORS` exposes the two headers. Frontend — `utils/apiClient.ts` (`onRenew`, one retry), `stores/auth.ts` (`renew`), `composables/useApi.ts` wiring, `middleware/auth.global.ts` (`reason=expired`), `utils/loginReason.ts` (pure), `pages/login.vue` (one `AppCard`), `service-worker/sw.ts` (`cacheWillUpdate` strips the headers from `api-state`). Backend spec §7 wins for the session semantics ("24-hour TTL" read as an idle window — stated there).

**⚠ Conflict note:** `origin/harness/2026-09-25-medium-auth-reports-postgres-and-redis-failures-as-401-and-logs-not` (done, unmerged) also edits `auth.Require` (a 503 branch on the session read). Keep this plan's `Require` change to **one call** — `maybeRenew(c, tokens, sessions, raw, userID)` placed after `stored != raw` passes — and all logic in `renew.go`, so the merge is a one-line conflict. If `origin/main` has that branch when you start: `git fetch origin main && git merge origin/main --no-edit`, keep both sides (the renewal call goes after the 503 branch).

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`, npm from `frontend/`. `rg`/`timeout` not installed: `grep -n`, `go test -timeout 60s`.
- The session key holds **one** token (byte-for-byte rule unchanged): after renewal the old token is rejected by the next request. Never widen a caller's deadline; never renew a token that failed to verify, is expired, or does not match the stored session.
- A failed `sessions.Put` during renewal is logged and the request proceeds with the old token (no header) — renewal must never turn a good request into a 5xx.
- `X-Session-Expires-In` is `TokenTTL` in whole seconds (`86400`), the same number `POST /auth/google` returns as `expires_in`.
- Frontend: `renew` never calls `clearApiCache()` (design §5); `StateBlock` is not used for the notice (§6); no page other than `login.vue` changes; `gofmt`, `npm run lint`, `npm run typecheck` clean.

## Review Focus
1. `Require` renews only when `exp - now < TokenTTL/2` (test at exactly half → no renewal; one second under → renewal); after renewal `sessions.Get` returns the new token and a request with the **old** token is `401`.
2. A revoked session (`Get` returns not found) and an expired JWT are both `401` and never renewed (no `Put` call).
3. `Access-Control-Expose-Headers: X-Session-Token, X-Session-Expires-In` on every allowed-origin non-preflight response; absent on the preflight and on foreign origins.
4. `apiClient`: header adopted only on `res.ok` and only when it differs from `getToken()`; the 401 retry happens exactly once and only when the token changed mid-flight (design §5).
5. The SW's `api-state` cache never stores the two headers (design §4 "Offline").
6. `/login` without `reason` renders byte-for-byte as today (e2e heading assertion passes).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/auth/token.go` | `Claims(token) (userID string, exp time.Time, err error)`; `Verify` wraps it |
| `backend/internal/auth/renew.go` | `RenewBelow = TokenTTL / 2`; `maybeRenew(...)`; header names as constants |
| `backend/internal/auth/middleware.go` | one call to `maybeRenew` after the session check |
| `backend/internal/auth/renew_test.go` | the six renewal tests (fake `SessionStore`, controllable `now`) |
| `backend/internal/middleware/cors.go` + `cors_test.go` | `Access-Control-Expose-Headers` |
| `frontend/utils/apiClient.ts` + `tests/unit/apiClient.test.ts` | `onRenew`, one retry |
| `frontend/stores/auth.ts` + `tests/unit/authStore.test.ts` | `renew()` |
| `frontend/composables/useApi.ts` | wires `onRenew` |
| `frontend/middleware/auth.global.ts` + `tests/unit/authMiddleware.test.ts` | `reason=expired` |
| `frontend/utils/loginReason.ts` + `tests/unit/loginReason.test.ts` | `loginNotice()` |
| `frontend/pages/login.vue` + `tests/e2e/login.spec.ts` | the expired `AppCard` |
| `frontend/service-worker/sw.ts` + `tests/unit/push.test.ts` or a new `swApiState.test.ts` | `cacheWillUpdate` strip |
| Backend spec §7; `harness/CODEMAP.md` (`auth`, `middleware`, `shell`) | sliding-session sentence |

## Tasks

### Task 1: `TokenIssuer.Claims` and `maybeRenew`

**Files:** `backend/internal/auth/token.go`, `renew.go`, `renew_test.go`, `middleware.go`.

- [ ] **Step 1 (tests first), `renew_test.go`:** build a `TokenIssuer` with a settable `now`, a fake `SessionStore` (map + `putCalls`), a Gin engine with `Require` and a `/ping` route returning the `UserID`. Issue a token at `t0`, `Put` it. Tests:
  - `TestRequireRenewsBelowHalfLife`: `now = t0 + TokenTTL/2 + 1s` → `200`, `X-Session-Token` present and ≠ old, `X-Session-Expires-In` = `"86400"`, store holds the new token, `putCalls == 1`; a second request with the **new** token at `now + 1s` → `200` and **no** header (`putCalls` still 1).
  - `TestRequireDoesNotRenewAtOrAboveHalfLife`: `now = t0 + TokenTTL/2 - 1s` → `200`, no header, `putCalls == 0`.
  - `TestRequireRejectsTheOldTokenAfterRenewal`: after a renewal, the old token → `401`.
  - `TestRequireDoesNotRenewARevokedSession`: store empty → `401`, `putCalls == 0`.
  - `TestRequireDoesNotRenewAnExpiredToken`: `now = t0 + TokenTTL + 1s` → `401`, `putCalls == 0`.
  - `TestRequireServesWhenRenewalPutFails`: fake `Put` returns an error → `200`, no header, one log line (capture with `log.SetOutput`).
- [ ] **Step 2:** `token.go`: add `func (t *TokenIssuer) Claims(token string) (string, time.Time, error)` (the existing parse; returns `claims.Subject, claims.ExpiresAt.Time`); `Verify` becomes `userID, _, err := t.Claims(token)`.
- [ ] **Step 3:** `renew.go`:
```go
// RenewBelow is how much of TokenTTL may remain before Require issues a
// fresh token: half. A learner who shows up daily therefore never expires;
// 24 h of absence still does (backend spec §7, read as an idle window).
const RenewBelow = TokenTTL / 2

const (
	HeaderSessionToken     = "X-Session-Token"
	HeaderSessionExpiresIn = "X-Session-Expires-In"
)

// maybeRenew runs after the presented token has verified and matched the
// stored session. Below RenewBelow it issues a new token, stores it with the
// full TTL (the old one is invalid from that moment — the key holds one
// token) and returns it in the response headers. A store failure is logged
// and the request proceeds on the old token.
func maybeRenew(c *gin.Context, tokens *TokenIssuer, sessions SessionStore, userID string, exp time.Time) { … }
```
  `Require`: replace `userID, err := tokens.Verify(raw)` with `userID, exp, err := tokens.Claims(raw)` and, after the `stored != raw` check, call `maybeRenew(c, tokens, sessions, userID, exp)`. Nothing else in `Require` changes.
- [ ] **Step 4:** `go test -timeout 60s ./internal/auth -run 'TestRequire' -v` green; existing auth tests green. Commit: `auth: Require renews a session below half-life (X-Session-Token); old token invalid at once`.

### Task 2: CORS exposes the headers

**Files:** `backend/internal/middleware/cors.go`, `cors_test.go`.

- [ ] **Step 1 (test first):** `TestCORSExposesTheSessionHeaders`: allowed origin, `GET` → `Access-Control-Expose-Headers: X-Session-Token, X-Session-Expires-In`; preflight → header absent; foreign origin → absent.
- [ ] **Step 2:** In the allowed non-preflight branch add `c.Header("Access-Control-Expose-Headers", "X-Session-Token, X-Session-Expires-In")` (string literal here — `middleware` must not import `auth`; a comment names `auth.HeaderSessionToken`).
- [ ] **Step 3:** `go test ./internal/middleware -v`; commit: `middleware: CORS exposes X-Session-Token and X-Session-Expires-In`.

### Task 3: Silent renewal in the PWA — design §5 "Silent renewal", "One retry", §4 "Offline"

**Files:** `frontend/utils/apiClient.ts`, `stores/auth.ts`, `composables/useApi.ts`, `service-worker/sw.ts`, tests.

- [ ] **Step 1 (tests first):** `tests/unit/apiClient.test.ts` — `calls onRenew with X-Session-Token and X-Session-Expires-In on a 2xx`; `ignores X-Session-Token on an error response and when it matches the current token`; `retries a 401 once with the newest token when a renewal landed since the request was sent`; `does not retry a 401 when no renewal happened`. `tests/unit/authStore.test.ts` — `renew updates accessToken, expiresAt and aelp.auth without touching the api-state cache` (spy on `clearApiCache`/`caches.delete`); `renew ignores a non-positive expires_in and a missing user`.
- [ ] **Step 2:** `apiClient.ts`: `ApiClientOptions.onRenew?: (token: string, expiresIn: number) => void`; in `request`, remember `sent = token`; after `doFetch`: if `res.ok`, read `X-Session-Token`; if present and `!== opts.getToken()`, `opts.onRenew?.(tok, Number(res.headers.get('X-Session-Expires-In')))`. On `401`: if `opts.getToken() !== sent` and this is the first attempt, re-send once with the newest token; otherwise `onUnauthorized()` + throw as today.
- [ ] **Step 3:** `stores/auth.ts`: action `renew(token: string, expiresIn: number, now = Date.now())` per design §5 (guards: finite positive `expiresIn`, `user !== null`); persists `{accessToken, expiresAt, user}` to `AUTH_STORAGE_KEY`; never calls `clearApiCache`.
- [ ] **Step 4:** `composables/useApi.ts`: pass `onRenew: (t, e) => auth.renew(t, e)`.
- [ ] **Step 5:** `service-worker/sw.ts`: on the `api-state` `NetworkFirst` strategy add a `cacheWillUpdate` plugin that returns a copy of the response with `X-Session-Token` and `X-Session-Expires-In` removed (`new Response(res.body, { status, statusText, headers: stripped })`); unit test (`tests/unit/swApiState.test.ts`, using the existing `fakeCaches.ts` pattern): a cached `quests/daily` response carries no `X-Session-Token`.
- [ ] **Step 6:** `npm run lint && npm run typecheck && npm run test:unit`. Commit: `web: adopt X-Session-Token silently, retry a 401 once after a renewal, never cache the header`.

### Task 4: The expired title screen — design §2 "Expired session", §3, §4, §6, §7

**Files:** `frontend/middleware/auth.global.ts`, `utils/loginReason.ts`, `pages/login.vue`, tests.

- [ ] **Step 1 (tests first):** `tests/unit/authMiddleware.test.ts` — `redirects an expired session to /login?reason=expired`; `redirects a missing session to /login without a reason`. `tests/unit/loginReason.test.ts` — `maps expired to the 24-hour sentence and everything else to null` (cases: `'expired'`, `'foo'`, `''`, `['expired']`, `undefined`). `tests/e2e/login.spec.ts` — `/login?reason=expired explains the 24-hour expiry before the Google button`; `/login without a reason shows no notice`.
- [ ] **Step 2:** `auth.global.ts` guarded branch: `accessToken !== null && !isAuthenticated` → `signOut()` then `navigateTo({ path: '/login', query: { reason: 'expired' } }, { replace: true })`; `accessToken === null` → `/login` no query. The `/login`-direct branch, `AppHeader` sign-out and `useApi` 401 path stay plain `/login` (design §2).
- [ ] **Step 3:** `utils/loginReason.ts`: `export function loginNotice(reason: unknown): string | null` → `'expired'` → `'Phiên đã hết hạn sau 24 giờ không hoạt động. Đăng nhập lại để tiếp tục.'`, else `null`.
- [ ] **Step 4:** `pages/login.vue`: between the tagline and the Google `AppButton`, `<AppCard v-if="notice" data-testid="login-expired" class="text-left text-sm text-ink"><span aria-hidden="true">⏳</span> {{ notice }}</AppCard>` with `const notice = computed(() => loginNotice(route.query.reason))`. No `aria-live`, no `role="alert"` (design §4). Everything else unchanged.
- [ ] **Step 5:** `npm run lint && npm run typecheck && npm run test:unit && npx playwright test tests/e2e/login.spec.ts`. Commit: `web: /login?reason=expired explains the 24-hour idle expiry`.

### Task 5: Spec and CODEMAP

- [ ] **Step 1:** Backend spec §7 (`grep -n '24-hour' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"`): add "The 24-hour TTL is an idle window: `auth.Require` renews a token with less than 12 h left and returns the new one in `X-Session-Token` / `X-Session-Expires-In`; the Redis key always holds exactly one token, so `DEL` still revokes at once." CODEMAP `auth` (renewal rule, headers, `Claims`), `middleware` (`Expose-Headers`), `shell` (`onRenew`, one retry, `reason=expired`, SW header strip).
- [ ] **Step 2:** Commit: `docs: sliding session in the backend spec and CODEMAP`.

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
cd ../frontend && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build && npx playwright test tests/e2e/login.spec.ts
git push -u origin harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve   # CI green: backend-unit, backend-integration, frontend, docker-images
```
Design acceptance (`harness/designs/stay-signed-in.md` §8):
- [ ] A renewal (`X-Session-Token` on any `2xx`) changes `auth.accessToken`, `auth.expiresAt` and `localStorage['aelp.auth']` and nothing visible; the `api-state` cache still holds its entries afterwards.
- [ ] The header is ignored on non-`2xx` responses and when it equals the current token.
- [ ] A `401` after a renewal (token sent ≠ token now) is retried exactly once with the newest token; a `401` with no renewal signs out immediately.
- [ ] A guarded route with an expired stored token redirects to `/login?reason=expired` (replace); with no token to `/login` with no query.
- [ ] `/login?reason=expired` shows the `⏳` card with the exact §7 sentence between the tagline and the Google button, not as a live region; `/login`, `/login?reason=foo`, `/login?reason=` render as today.
- [ ] After "Đăng xuất" and after a `401` sign-out, `/login` shows no notice.
- [ ] Offline, a cache-served `quests/daily` / `pet/status` response carries no `X-Session-Token`.
- [ ] The Vitest/Playwright tests named in §8 exist and pass.
Manual (reviewer, live or local): sign in, set `expiresAt` in `aelp.auth` to now + 11 h, load `/`, see `X-Session-Token` in the network tab and no re-render; set `expiresAt` in the past, reload `/`, read the sentence; "Đăng xuất" → gone.

## Execution summary

**Built.** All five tasks landed as separate commits on `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve`, each preceded by a failing test:

1. `auth: Require renews a session below half-life (X-Session-Token); old token invalid at once` — `token.go` gained `Claims` (subject + expiry; `Verify` now wraps it), `renew.go` (`RenewBelow = TokenTTL/2`, `maybeRenew`, `HeaderSessionToken`/`HeaderSessionExpiresIn`), `middleware.go`'s `Require` calls `maybeRenew` once after the existing `stored != raw` check. Six new tests in `renew_test.go` (half-life boundary at exactly half and one second under, old-token rejection after renewal, no renewal on a revoked session or an expired token, and a failed `Put` serving the request on the old token with one log line and no header).
2. `middleware: CORS exposes X-Session-Token and X-Session-Expires-In` — one `c.Header` call in the allowed non-preflight branch (a string literal, per the plan's note that `middleware` must not import `auth`), one new test asserting present/absent across allowed, preflight and foreign-origin cases.
3. `web: adopt X-Session-Token silently, retry a 401 once after a renewal, never cache the header` — `apiClient.ts` gained `onRenew` and the one-retry-on-401 logic; `stores/auth.ts` gained `renew()` (guarded against non-positive `expiresIn`/no user, never calls `clearApiCache`); `useApi.ts` wires it; `service-worker/sw.ts`'s `api-state` `NetworkFirst` strategy gained a `cacheWillUpdate` plugin backed by a new pure `service-worker/apiStateCache.ts::stripSessionHeaders`.
4. `web: /login?reason=expired explains the 24-hour idle expiry` — `middleware/auth.global.ts` now redirects with `?reason=expired` only when a *stored* token was present but expired (a missing token still gets a bare `/login`); new `utils/loginReason.ts::loginNotice`; `pages/login.vue` renders the `⏳` `AppCard` between the tagline and the Google button.
5. `docs: sliding session in the backend spec and CODEMAP` — backend spec §7 gained the idle-window sentence; CODEMAP `auth`, `middleware` and `shell` entries updated.

**Deviations:**
- `service-worker/apiStateCache.ts` is a new file the plan's file-structure table didn't list (only `sw.ts` + a new test were named). Extracted `stripSessionHeaders` into it — mirroring the existing `push.ts` split — because `sw.ts` runs top-level `self.__WB_MANIFEST` service-worker setup that a Vitest/happy-dom environment can't boot; a plain exported function is what `tests/unit/swApiState.test.ts` (named in the plan) actually needed to import.
- `TestRequireDoesNotRenewAtOrAboveHalfLife` covers both "one second above half" (named in the plan) and "exactly half" (named only in Review Focus #1) as subtests, so both the step-1 bullet and the review checklist are satisfied by one test function.
- `renew_test.go` uses its own `fakeRenewSessions` (map + `putErr` + `putCalls`) rather than `session_test.go`'s shared `fakeSessions`, whose single `err` field would also fail the `Get` that `Require` does just before `maybeRenew` — needed to isolate a failing `Put` from a failing `Get`.

**Verification (all green, from a clean shell):**
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
  → ok for all 13 packages (cmd/api, airouter, auth, config, google, health, middleware, notify, onboarding, pet, quests, secrets, store); gofmt -l empty; go vet clean.
cd backend && make test-integration (TEST_DATABASE_URL/TEST_REDIS_URL against the isolated stack below)
  → 12 Integration tests pass, including auth's own TestIntegrationUpsertCreatesThenPreservesTheLearnerState (unaffected).
cd frontend && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build && npx playwright test tests/e2e/login.spec.ts
  → ESLint: no issues; typecheck: clean; 18 test files / 92 tests pass (incl. the new apiClient, authStore, loginReason, swApiState, and the updated authMiddleware suites); npm run build succeeds (including the sw.ts workbox build); 5/5 Playwright tests pass, including the two new login.spec.ts cases.
```

**Runtime proof (COMPOSE_PROJECT_NAME=stay-signed-in, Postgres 55613 / Redis 56613 / API 8613):**
- Booted `docker compose -p stay-signed-in up -d --wait` (Postgres+Redis healthy), then `go run ./cmd/api` on PORT=8613 with a scratch `.env` (JWT_SECRET, ENCRYPTION_SECRET_KEY, dummy Google client id/secret, FRONTEND_ORIGIN=http://localhost:3613) — migrations applied, API listening.
- Real user path exercised end to end: issued a JWT (via a throwaway `cmd/devtoken` helper using the running server's own `JWT_SECRET`, deleted afterwards — never committed) whose remaining life was one second under `TokenTTL/2`, seeded it into Redis as `sess:dev-user-1:token`, then `curl`ed a guarded endpoint with `Origin: http://localhost:3613`:
  - Response carried `X-Session-Token` (a fresh, different JWT), `X-Session-Expires-In: 86400`, and `Access-Control-Expose-Headers: X-Session-Token, X-Session-Expires-In`.
  - A second request with the **old** token → `401`.
  - A request with the **new** token → succeeds at the auth layer with no further renewal header (not yet below half-life).
  - An `OPTIONS` preflight to the same origin → `204` with no `Access-Control-Expose-Headers` (present only on the actual request, as designed).
- `npx playwright test tests/e2e/login.spec.ts` additionally boots the real built Nuxt server (`node .output/server/index.mjs`) and drives it with a real Chromium browser — the `/login?reason=expired` and `/login` (no reason) cases render through the actual page, not a mock.
- Cleanup verified: `docker compose -p stay-signed-in down` (containers + network removed, `docker ps` empty for the project), the API process on port 8613 killed (`lsof -i :8613` empty), the scratch `backend/.env` and the throwaway `backend/cmd/devtoken/` deleted, `git status --short` clean before pushing.

**CI:** pushed `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve`; run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36227582887 — all five jobs green (`docker-images`, `backend-integration`, `harness-tooling`, `backend-unit`, `frontend`).

No PR opened (owner takes one PR per day per AGENTS.md).
