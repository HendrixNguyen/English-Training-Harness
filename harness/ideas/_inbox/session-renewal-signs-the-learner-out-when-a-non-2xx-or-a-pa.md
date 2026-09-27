---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md
---
# Session renewal signs the learner out when a non-2xx or a parallel request races the rotated token

## Why
The sliding session rotates the stored token (`sess:{user_id}:token` gets the new JWT and the old one is rejected at once), but the client does not always receive or adopt the new one. Every such case ends in `401` → `onUnauthorized()` → sign-out → `/login` with the full Google consent screen. That is the exact outcome this feature exists to remove, and it now happens **12 hours** after sign-in instead of 24. Two ways it happens, both reproduced:

1. **A non-2xx response rotates the session, but the client ignores the header on non-2xx (deterministic).** `maybeRenew` runs in `Require` before the handler, so `X-Session-Token` rides on whatever status the handler answers: 404 `no_active_roadmap`, 409, 422, 429, 503/504 from AI routes, 500. `apiClient.ts` adopts the header only when `res.ok` (design §5, pinned by the unit test `ignores X-Session-Token on an error response…`). The client keeps the old token, which Redis no longer holds, so the next request is 401 with `sent === getToken()`, there is no retry, and the learner is signed out.
2. **Parallel requests race the renewal (intermittent, on every hub load).** `pages/index.vue:11` runs `Promise.all([pet.load(), quest.load()])`, so the first hub load past half-life always sends two requests with the same old token. If one request's `Put` lands before the other's `Get`, the second gets 401. If that 401 returns before the renewing response (it usually does, because a 401 skips the handler), the client still holds the token it sent, the one-retry rule does not fire, and the learner is signed out. If both renew in different seconds, the two responses carry different tokens. The client adopts whichever *arrives* last, which may not be the one Redis kept, and the next request signs out.

## Expected output
- The server and the client agree on when a session rotates. Either the client adopts `X-Session-Token` from any response except 401, or the server only emits and stores a renewal on responses it knows are 2xx (e.g. a response-writer wrapper). The first option is simpler.
- Concurrent requests with the same pre-renewal token never produce a 401. For example: renewal is a compare-and-set (Lua: `SET` only if the value still equals the presented token; otherwise read the current value and return *that* in the header). And for a short grace window (≈60 s) `Require` accepts the immediately previous token and answers with the current one in `X-Session-Token`, instead of 401. Revocation (`DEL`) still rejects at once, with no grace.
- Tests:
  - a backend test with two concurrent requests with one below-half-life token, asserting no 401 and the same token in both headers;
  - a backend test that a 404 handler response carrying a renewal is adopted by the client;
  - the frontend unit test that currently asserts "ignored on an error response" is replaced;
  - an apiClient test where a sibling's 401 arrives before the renewing response, asserting no sign-out.

## Evidence
- Plan: `harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`. Idea concurrency note: "a `401` immediately after a renewal is retried once with the newest stored token before signing out". Design §5 "Silent renewal" and "One retry".
- `backend/internal/auth/middleware.go`: `maybeRenew(c, tokens, sessions, userID, exp)` runs before `c.Next()`. `backend/internal/auth/renew.go`: `sessions.Put(...)`, then `c.Header(HeaderSessionToken, fresh)` regardless of the eventual status.
- `frontend/utils/apiClient.ts`: `if (res.ok) { const renewed = res.headers.get('X-Session-Token') … }`, and on 401 `if (!isRetry && opts.getToken() !== sent)`.
- `frontend/pages/index.vue:11`: `void Promise.all([pet.load(), quest.load()])`.
- Reviewer, isolated stack (`rv-auth`, API on 18343, token minted with `RenewBelow + 1m` elapsed):
  - `non2xx.sh`: `GET /api/v1/quests/daily` → `HTTP/1.1 404`, `X-Session-Token: eyJ…`, `{"error":"no_active_roadmap"}`. The same old token on `GET /api/v1/pet/status` then → `401`.
  - `race.sh` (two concurrent curls, 15 trials): `trial 1: pet=[401 0.0054] quests=[404 0.0110]`. The 401 returned in 5 ms, before the renewing response (11 ms).
  - Real browser (built Nuxt on :3614 against that API, `aelp.auth` seeded with the below-half-life token, then load `/`): network `GET …/quests/daily → 401`, `GET …/pet/status → 500` (aborted by the navigation). The page ends on `/login` (no notice) and `aelp.auth` is cleared. The learner is signed out on the first hub load after 12 h.
