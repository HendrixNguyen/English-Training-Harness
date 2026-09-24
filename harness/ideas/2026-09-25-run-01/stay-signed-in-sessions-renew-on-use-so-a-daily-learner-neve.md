---
type: feature
status: proposed
source: ideator
run: 2026-09-25-run-01
---
# Stay signed in: sessions renew on use so a daily learner never sees the Google consent screen again

## Why
The product asks for a 30-minute session **every day**. The session it issues lasts exactly 24 hours from sign-in (`store.SessionTTL = 24h`, `auth.TokenTTL`, `expires_in: 86400`), with no refresh path. So a learner who practises at 20:00 tonight and opens the app at 20:05 tomorrow is expired, the route guard (`middleware/auth.global.ts`) drops the session and sends them to `/login`, and `/login` builds a Google URL with `prompt=consent` — the full Google account-chooser and permissions screen, listing Calendar and Tasks scopes, **every single day**. For a habit product this is the worst possible place for friction: it lands precisely at the moment the reminder push brought them back. Login frustration is the most-cited reason for abandoning an app, and 30-day sessions are the accepted norm for mobile; even a daily re-tap of "Continue as …" is a drop-off point, and ours is the heavier consent flow because `prompt=consent` is required to guarantee a refresh token for Google sync.

Backend spec §7 says "JWT access token validated against Redis key (`sess:{user_id}:token`) with a 24-hour TTL" — an idle window, not a hard cap. Renewing the session while it is in use keeps that guarantee (24 h of *inactivity* still signs you out, `DEL sess:*` still revokes instantly) and removes the daily consent screen. This is a change to an existing path (`auth.Require`, the client's API wrapper), not a new surface.

## Expected output
User-visible:
- A learner who opens the app at least once every 24 hours is never asked to sign in again. Reminder push → tap → hub, no Google screen.
- Being away for more than 24 hours still signs the learner out, and the login screen says why ("Phiên đã hết hạn sau 24 giờ không hoạt động") instead of appearing unexplained.
- Signing in on another device still supersedes the first (unchanged; the inbox bug about that is not touched), and a revoked session (`DEL sess:{user_id}:token`) is still rejected on the next request.

Technical:
- `auth.Require`: when the presented token verifies, matches the stored session and has less than half of `TokenTTL` left, issue a new JWT (fresh `exp = now + TokenTTL`), `SET sess:{user_id}:token` to it with the full TTL, and return it in a response header `X-Session-Token` (plus `X-Session-Expires-In`). The old token is invalid from that moment (the session key holds only the new one — same byte-for-byte rule as today). Renewal happens at most once per half-life per user, so the Redis write cost is one `SET` a day per active learner. Add the header to the CORS `Allow-Headers`/`Expose-Headers` in `middleware` so the browser can read it cross-origin.
- Frontend `utils/apiClient.ts`: after every successful response, if `X-Session-Token` is present, call `useAuthStore().renew(token, expiresIn)` which updates `accessToken`/`expiresAt` and `localStorage['aelp.auth']` without clearing the `api-state` cache (same user). The route guard's expired branch sets a `reason=expired` query the login page turns into the copy above.
- Concurrency note for the plan: two in-flight requests may both trigger renewal; the second `SET` wins and the first new token is rejected on its next use — the client must always adopt the latest header it receives, and a `401` immediately after a renewal is retried once with the newest stored token before signing out.
- Tests: `auth` pure tests — no renewal above half-life, renewal below, old token rejected after renewal, revoked session not renewed, expired token not renewed; `middleware` test for the exposed header; frontend unit tests for `renew()` persistence and the single retry. Backend spec §7 and CODEMAP `auth`/`middleware`/`shell` updated to describe the sliding session.

## Evidence
- Backend spec §7 (24-hour TTL against `sess:{user_id}:token`), §6.1 (`expires_in: 86400` response); 1st-thinking §1 (daily retention as the goal), §5.1 step 3 ("Return JWT").
- Code: `backend/internal/store/keys.go` `SessionTTL = 24 * time.Hour`; `backend/internal/auth/token.go` `TokenTTL = store.SessionTTL`; `backend/internal/auth/scopes.go` and `frontend/utils/googleAuth.ts` (`prompt=consent` on every login); `frontend/stores/auth.ts` `isAuthenticated: expiresAt > Date.now()`; `frontend/middleware/auth.global.ts` (expired session dropped → `/login`). CODEMAP `auth` ("accepts a token only if it verifies, is unexpired, and matches the stored session byte for byte").
- Inbox context (not duplicated): `signing-in-on-a-second-device-silently-logs-the-first-one-ou.md` — single-session design stays as is.
- Login friction as the top abandonment reason and 30-day mobile session guidance: https://www.corbado.com/blog/login-friction-kills-conversion ; https://www.descope.com/learn/post/session-timeout-best-practices ; user-reported churn from aggressive session expiry: https://support.discord.com/hc/en-us/community/posts/1500000234942
