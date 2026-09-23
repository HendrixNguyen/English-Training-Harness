---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# PgRefreshTokenSource has no sentinel for a missing user, so a deleted user gets 500 not 409

## Why
`PgRefreshTokenSource.RefreshToken` runs `SELECT COALESCE(google_refresh_token, '') FROM users
WHERE id = $1` and wraps *any* scan error opaquely (`token.go:36-44`). When the row does not exist,
pgx returns `pgx.ErrNoRows`, which becomes `google: reading refresh token: no rows in result set` —
not `ErrNoRefreshToken`. `Sync` only special-cases `ErrNoRefreshToken` (`service.go:40-42`), so the
error falls through to `service.go:43-45`, then to `handler.go:40-41`, and the caller gets
`500 {"error":"internal_error"}`.

Its two siblings in the same package get this right: `ActiveRoadmap` (`repo.go:108-110`) and
`SyncState` (`repo.go:145-147`) both translate `pgx.ErrNoRows` into a package sentinel. `token.go`
is the odd one out, and it is the one on the authentication path.

Reachable whenever a valid session outlives its `users` row — the session lives in Redis with a 24 h
TTL (`sess:{user_id}:token`) and nothing deletes it when a user is removed, so an account deletion
or a restored-from-backup database turns this route into a 500 instead of the 409 that tells the
client to re-authenticate. It is a small hole, but it produces exactly the wrong signal: an
unauthenticated user is told the server is broken.

## Expected output
`RefreshToken` maps `pgx.ErrNoRows` to `ErrNoRefreshToken` (or a distinct `ErrNoUser` that `Sync`
also folds into `ErrReauthRequired`), so a missing user yields `409 reauth_required` like an empty
column does. A unit or integration test queries an id that is not in `users` and asserts the
sentinel; `integration_test.go` currently only inserts a user with a non-empty token
(`integration_test.go:35-37`), so neither the empty-column branch (`token.go:41-43`) nor the
missing-row branch has any coverage.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23).
- `backend/internal/google/token.go:34-44` — the query and the opaque wrap, with no `pgx.ErrNoRows` case.
- `backend/internal/google/repo.go:108-110` and `:145-147` — the sentinel-mapping convention this file departs from.
- `backend/internal/google/service.go:39-45` — only `ErrNoRefreshToken` becomes `ErrReauthRequired`.
- `backend/internal/google/handler.go:40-41` — everything else becomes `500 internal_error`.
- `backend/internal/google/integration_test.go:35-37,54-56` — the only exercise of this code, happy path only.
