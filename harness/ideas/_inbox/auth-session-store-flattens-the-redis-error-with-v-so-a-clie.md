---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# auth session store flattens the Redis error with %v so a client cancel is logged as an outage

## Why
`RedisSessionStore.Put/Get/Delete` now wrap Redis failures as `fmt.Errorf("%w: reading session: %v", ErrSessionStoreUnavailable, err)`. The `%v` drops the underlying error from the chain, so `errors.Is(err, context.Canceled)` / `context.DeadlineExceeded` is false. When a learner's browser aborts a guarded request (tab closed, navigation), go-redis returns `context.Canceled`, and `Require` logs `auth: session store unavailable for user …` and answers 503. The operator then sees a false Redis outage, which is the kind of misleading log line the plan set out to remove. The log line also repeats its prefix (`auth: session store unavailable for user X: auth: session store unavailable: reading session: …`).

## Expected output
- The three wraps keep the cause in the chain: `fmt.Errorf("%w: reading session: %w", ErrSessionStoreUnavailable, err)` (Go 1.20+ multi-`%w`).
- `Require` treats a cancelled/expired request context (`errors.Is(err, context.Canceled)` or `c.Request.Context().Err() != nil`) as a client abort: no "session store unavailable" line (or a distinct one), so there is no false outage signal.
- The log line names the failure once. Drop the duplicated sentinel text from either the prefix or the sentinel message.
- A unit test: `Get` with an already-cancelled context → `errors.Is(err, context.Canceled)` is true.

## Evidence
- Plan: `harness/plans/2026-09-25-auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md` (review 2026-09-27).
- `backend/internal/auth/session.go` — `Put`, `Get` and `Delete` use `%v` for the Redis error.
- `backend/internal/auth/middleware.go` — `case err != nil:` logs and answers 503 for any non-`ErrNoSession` error, cancellation included.
- Live on the reviewer's isolated stack (`docker compose -p rv-auth stop redis`, then `GET /api/v1/pet/status` with a valid token): the log line was `auth: session store unavailable for user 00000000-…abcd: auth: session store unavailable: reading session: dial tcp [::1]:6392: connect: connection refused`, with the prefix twice.
