---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 2
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
