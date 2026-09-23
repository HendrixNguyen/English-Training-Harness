---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# healthz leaks Postgres and Redis driver error strings publicly

## Why
`GET /healthz` is unauthenticated (it is the platform's probe endpoint) and, on failure, copies the
raw driver error into the JSON body: `body["postgres"] = err.Error()` and the same for Redis
(`backend/internal/health/health.go:31-38`). pgx errors embed the connection identity. The reviewer
saw this shape in ordinary local output:

```
failed to connect to `user=u database=db`: 127.0.0.1:59999 (127.0.0.1): dial error: …
```

Against the Railway deployment that becomes `user=<db user> database=<db name>` plus the internal
host and port, served to anyone who curls the URL while the database is down — i.e. exactly when an
attacker is most interested. go-redis dial errors disclose the Redis host and port the same way.

This is information disclosure, not a crash: the endpoint still behaves correctly.

## Expected output
`/healthz` returns a fixed, non-descriptive marker per dependency — `"postgres": "unavailable"`,
`"redis": "unavailable"` — with HTTP 503 as today, and logs the full error server-side
(`log.Printf`/structured logger) where the operator can read it. No connection string, host, port,
user or database name reaches the response body. The existing tests that assert on the error text
(`internal/health/health_test.go:49`, `:60`) are updated to assert the marker plus the 503.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 7 — the handler is the plan's verbatim text).
- `backend/internal/health/health.go:31-38` — `body["postgres"] = err.Error()`, `body["redis"] = err.Error()`.
- `backend/cmd/api/main.go:45` — the route is mounted with no auth middleware, as it must be.
- Spec §8 — `DATABASE_URL` / `REDIS_URL` are injected by Railway, so the leaked identity is the
  production one.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed at `health.go:31-38`: `body["postgres"] = err.Error()` on an unauthenticated route, disclosing the production DB user/name/host when it is down. Fix: fixed `"unavailable"` markers, log the error server-side, per-dependency deadlines. Folds in `healthz-shares-one-2s-deadline-across-two-sequential-pings.md` — same 20-line handler, same tests.
