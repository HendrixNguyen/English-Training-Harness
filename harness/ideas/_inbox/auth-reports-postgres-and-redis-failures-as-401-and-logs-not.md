---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# auth reports Postgres and Redis failures as 401 and logs nothing

## Why
`auth` collapses every kind of failure into "your credentials are bad", and writes nothing to the log
on any of them. Two places:

- **`Handler`** turns *any* error from `Service.SignIn` into `401 {"error":"google_auth_failed"}`.
  That error may be Google rejecting the code (correctly a 401), but it may equally be Postgres being
  unreachable, the `users` upsert violating a constraint, JWT signing failing, or Redis refusing the
  session write. A database outage therefore presents to every signing-in user as "Google sign-in
  failed", and to the operator as nothing at all.
- **`Require`** treats any error from `sessions.Get` — including a Redis connection failure — as
  `401 {"error":"unauthorized"}`. A Redis blip logs the entire user base out rather than returning
  503, and again logs nothing.

There is a concrete data-shaped instance of this too. §3.2 declares `users.email VARCHAR(255) UNIQUE
NOT NULL`, but the upsert's conflict target is `(google_id)` only. A second Google account presenting
an email that already belongs to a different `google_id` — an account migration, a reclaimed Workspace
address — raises a `users_email_key` unique violation that `ON CONFLICT (google_id)` does not catch.
That user is then permanently unable to sign in, sees `google_auth_failed`, and no log line anywhere
says `duplicate key value violates unique constraint "users_email_key"`. Nobody will ever diagnose
that from the outside.

This is the first package in the codebase to handle a request that can fail for infrastructure
reasons, so the convention it sets here is the one the remaining six slices will copy.

## Expected output
- `Handler` distinguishes "Google rejected this" (401) from "we failed" (500 or 503). A sentinel
  error from the Google leg — `errors.Is(err, ErrGoogleRejected)` — is enough; everything else is a
  server error.
- `Require` distinguishes `ErrNoSession` (401, correctly) from a session-store transport error (503).
- Both log the underlying error at error level, with the user id or google id where known and
  **never** the JWT, the Google access token, the refresh token or `JWT_SECRET`.
- The response body stays opaque to the client — `invalid_request` / `unauthorized` / `internal` —
  so nothing about Google's reply or the database leaks outward. (`GoogleClient.doJSON` embeds
  Google's raw response body in its error string; that is fine for a log and must not reach a client.)
- A test that a `UserRepo` failure and a `SessionStore` failure produce a 5xx, not a 401.
- Decide what a `users_email_key` collision should do — most likely a distinct 409 with a log line —
  rather than leaving it as an unexplained permanent 401.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
- `backend/internal/auth/handler.go:26-29` — `if err != nil { c.JSON(401, gin.H{"error": "google_auth_failed"}); return }`, err discarded.
- `backend/internal/auth/middleware.go:32-36` — `if err != nil || stored != raw { abortUnauthorized(c) }`, err discarded.
- `backend/internal/auth/service.go:29-53` — Google, repo, token and session errors all returned as one undifferentiated `error`.
- `backend/internal/store/migrations/0001_init.up.sql` — `email VARCHAR(255) UNIQUE NOT NULL`, vs `ON CONFLICT (google_id)` in `backend/internal/auth/repo.go:36`.
- Observed live in the plan's worktree: `POST /api/v1/auth/google` with a bogus code returned
  `401 {"error":"google_auth_failed"}` and the server log contained no line about the failure at all.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium (top 10).** Confirmed on `main`: `auth/handler.go` maps every `SignIn` error to 401 and `middleware.go` maps every `sessions.Get` error to 401, both without logging. A Redis blip signs out every active user with no operator trace, and the `users_email_key` collision is a permanent, undiagnosable lockout for that account. Plan: `ErrGoogleRejected` sentinel → 401, everything else 5xx + `log.Printf` (never the tokens); `Require` distinguishes `ErrNoSession` from transport errors (503); decide the email-collision response (409). Folds in `service-and-require-failure-paths-are-untested-the-fakes-err.md` — its three tests are this plan's regression tests.
