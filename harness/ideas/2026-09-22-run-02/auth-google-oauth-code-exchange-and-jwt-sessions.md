---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 2
priority: high
plan: harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md
---
# Auth: Google OAuth code exchange and JWT sessions

## Why
Google sign-in is the only door into the product (§5.1 steps 1–3) and the refresh token captured here is the one thing the Calendar/Tasks integration (§5.1 steps 6–7, a stated core objective) cannot work without. Every other endpoint in §7 is per-user, so a JWT + Redis session middleware is a precondition for quests, pet, notify and google alike. Getting the OAuth scopes right in this slice avoids forcing every early user to re-consent when the google slice lands.

## Expected output
Delivers (Go package `backend/internal/auth`):
- `POST /api/v1/auth/google` — body `{code, redirect_uri}`; exchanges the code with Google using `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`, fetches the profile (`sub`, `email`, `name`), upserts `users` on `google_id` (sets `email`, `full_name`, `google_refresh_token`; keeps existing `cefr_current`/`target_goal` on re-login), issues a JWT (24h) and writes `sess:{user_id}:token` with the §4 24h TTL. Response `{token, user:{id,email,full_name,cefr_current}}`.
- The OAuth request must include the scopes the google slice needs (`openid email profile`, `https://www.googleapis.com/auth/calendar.events`, `https://www.googleapis.com/auth/tasks`) with `access_type=offline` so a refresh token is returned; the frontend-shell login screen uses the same scope list.
- Gin middleware `auth.Require()` that verifies the JWT, checks `sess:{user_id}:token` still exists in Redis (deleting the key revokes the session), and puts `user_id` into the request context. All later slices mount their routes behind it.
- `users.target_goal` is `NOT NULL` in §3.2 but is only known after onboarding; this slice inserts an empty string and CODEMAP notes it (see Notes in `_run.md`).
- Tests: code exchange and profile fetch against `httptest` fakes; new user vs returning user upsert; middleware rejects missing, malformed, expired, and revoked (Redis key deleted) tokens.
- Tables: `users` (write). Redis keys: `sess:{user_id}:token`. Screens: none (the login screen is in frontend-shell).

Depends on: store (1).

## Evidence
- Spec §5.1 (lines 270–302) steps 1–3: OAuth callback → exchange token → store refresh token → return JWT.
- Spec §7 (line 670) `POST /api/v1/auth/google`: "OAuth code token swap & JWT issuance".
- Spec §3.2 `users` (lines 152–174): `google_id UNIQUE NOT NULL`, `google_refresh_token TEXT`, `target_goal VARCHAR(255) NOT NULL`.
- Spec §4 (line 262) `sess:{user_id}:token` String, 24h.
- Spec §8 (line 696) `GOOGLE_CLIENT_ID` & `GOOGLE_CLIENT_SECRET`.
- `harness/CODEMAP.md` → `auth`, `google` (needs the refresh token stored here).

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** Yes. §5.1 steps 1–3 make Google the only door into the product, and §7 lists eight endpoints of which seven are per-user — so without `auth.Require()` the remaining six slices have nowhere to mount their routes. The sharper argument in the *Why* is the scope list: `google_refresh_token` is a column in §3.2 and §5.1 step 2 says to store it, but a refresh token only comes back if the *first* consent carries `access_type=offline` and the Calendar/Tasks scopes. If slice 2 ships with `openid email profile` alone, every user created before slice 6 has to re-consent. That is a real, cheap-now/expensive-later decision, and it belongs here.

**Is the *Expected output* achievable in one plan?** Yes, and it is materially smaller than slice 1: one endpoint, one middleware, one upsert. The only external dependency is Google's token and userinfo endpoints, and both are plain HTTPS JSON calls that an `httptest.Server` can stand in for — so this slice needs no live Google credentials to be verified, exactly as the *Expected output* asks.

**Dependencies:** `store` (slice 1) for the module, the `users` table and `SessionKey`/`SessionTTL`. Queued at `order: 1` ahead of this one, so this is not a blocking gap. Nothing else.

**Priority rationale:** `high` — it blocks MVP order. Slices 3, 4, 6 and 7 all state "routes behind `auth.Require()`".

**Test strategy (no live services).** Everything is behind an interface:
- The Google exchange is an `http.Client` pointed at configurable endpoint URLs, so tests aim it at `httptest.Server` handlers and assert the *outgoing* form values — `grant_type=authorization_code`, `code`, `redirect_uri`, `client_id`, `client_secret` — as well as the parsed response.
- The scope list is a package-level constant tested directly (all five scopes present, `access_type=offline`, `prompt=consent`), because the frontend-shell slice has to reuse the same list and a constant is what it will import.
- The user upsert sits behind a `UserRepo` interface with an in-memory fake, so "new user" vs "returning user keeps `cefr_current`/`target_goal`" is a pure test. The real SQL is one `INSERT … ON CONFLICT (google_id) DO UPDATE` statement, asserted as text by a separate test and exercised for real only by a `DATABASE_URL`-gated integration test that skips like slice 1's.
- The session store sits behind a `SessionStore` interface (`Put`/`Exists`/`Delete`) with a fake, so the four middleware rejection cases — missing, malformed, expired, revoked — are pure `httptest` assertions.

**Design decisions taken in the plan that the idea left open:**
- `github.com/golang-jwt/jwt/v5`, HS256, `JWT_SECRET` from env. §8 does not list `JWT_SECRET` (the `_run.md` Notes flag this); the plan adds it to `config.Config` as **required** and records the gap in CODEMAP rather than inventing a spec edit.
- The Redis session value is the JWT string itself, keyed by `sess:{user_id}:token` (§4 calls it "Active JWT session context"). `auth.Require()` compares the presented token to the stored one, so issuing a new token supersedes the old one and `DEL` revokes — which is what the idea means by "deleting the key revokes the session". Recorded as an open question below since §4's "and user metadata" hints at a richer value.
- Token expiry is 24h to match `SessionTTL`; the JWT `exp` and the Redis TTL are set from one constant so they cannot drift.
- `users.target_goal` is inserted as `''` (the `_run.md` Notes contract). The plan asserts this explicitly in a test so a later onboarding slice cannot quietly regress it.

**Open questions recorded for the human (not blocking):**
1. §4 describes `sess:{user_id}:token` as "Active JWT session context **and user metadata**". The plan stores only the token string, which is enough for revocation. If per-request user metadata without a Postgres round-trip is wanted, the value should become a JSON blob — a small, backward-compatible change a later slice can make.
2. `JWT_SECRET` is missing from the §8 deployment checklist. Worth a follow-up spec bug so the Railway environment list is complete.
3. §7 says `POST /api/v1/auth/google` performs the "OAuth code token swap", but the spec never specifies the logout endpoint that would `DEL` the session key. The plan ships the revocation mechanism (delete the key) without an endpoint; adding `POST /api/v1/auth/logout` would be a new API and is therefore out of scope here.
