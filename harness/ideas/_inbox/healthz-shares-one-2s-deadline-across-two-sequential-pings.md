---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into healthz-leaks-postgres-and-redis-driver-error-strings-public.md (survivor): same handler, one plan gives each ping its own pingTimeout budget and adds the both-down and slow-dependency tests."
---
# healthz shares one 2s deadline across two sequential pings

## Why
`health.Handler` derives one `context.WithTimeout(…, 2*time.Second)` and then pings Postgres and
Redis sequentially on it (`backend/internal/health/health.go:25-38`). The budget is shared, not
per-dependency. If Postgres is slow and consumes the whole 2s, the Redis ping gets an
already-expired context and fails immediately with `context deadline exceeded` — so a healthy Redis
is reported as down. The operator reading the 503 body sees two failures and starts debugging the
wrong one. Worst case the handler blocks a request goroutine for the full 2s per probe while the
platform's own probe timeout is shorter.

The timeout is also untested. `fakePinger.Ping` (`health_test.go:16`) ignores its context entirely,
so deleting the `WithTimeout` wrapping, or setting `pingTimeout` to zero or 24h, would leave every
existing test green. There is no test for both dependencies being down at once either, so the path
that sets `healthy = false` twice and reports both errors in one body is unverified.

## Expected output
- Each dependency gets its own `pingTimeout` budget (a fresh `context.WithTimeout` per ping), or the
  two pings run concurrently under the shared deadline. Either way a slow Postgres cannot mark a
  healthy Redis down.
- A test with a `Pinger` that blocks until its context is cancelled proves the deadline is enforced
  and proves the other dependency is still checked correctly.
- A test for both dependencies failing at once, asserting 503 and both error markers in one body.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` (Task 7).
- `backend/internal/health/health.go:19` (`pingTimeout`), `:25-26` (single shared context),
  `:31-38` (the two sequential pings).
- `backend/internal/health/health_test.go:16` (fake ignores ctx), `:32-62` (three cases, none with
  both down, none with a slow dependency).

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — folded.** Correct but low; the health-leak plan rewrites the same handler and takes the per-ping deadline and the missing tests with it.
