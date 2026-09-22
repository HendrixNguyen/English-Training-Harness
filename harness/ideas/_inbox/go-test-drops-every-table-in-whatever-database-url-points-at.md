---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
---
# go test drops every table in whatever DATABASE_URL points at

## Why
`backend/internal/store/integration_test.go` gates its live-database tests only on the presence of
`DATABASE_URL` — the exact variable name spec §8 uses for the Railway production database. The first
thing each of those tests does is `reset(t, pg)` (integration_test.go:27-40), which executes
`0001_init.down.sql` (`DROP TABLE users; …`) and then `DROP TABLE IF EXISTS schema_migrations`
against that connection. There is no build tag, no `-short` guard, no separate `TEST_DATABASE_URL`,
and no "is this a throwaway database?" check.

That makes the project's own documented workflow destructive. `harness/CODEMAP.md` tells the reader
to `make up`, export `DATABASE_URL`/`REDIS_URL`, then `make test-integration`. Those exports stay in
the shell. The very next `make test` — described in the same CODEMAP sentence as needing "no live
services" — silently wipes the schema. Point the same shell at a real database (to inspect it, to run
a one-off query, to reproduce a bug) and `go test ./...` destroys it.

`internal/health` is unaffected; this is purely the store integration file, but it runs as part of
the default `./...` package set.

## Expected output
The destructive tests cannot run against a database the developer did not explicitly nominate as
disposable. Any of these is acceptable, cheapest first:
- Gate on a dedicated variable (`TEST_DATABASE_URL`), so `DATABASE_URL` alone never triggers a drop.
- Or put the file behind `//go:build integration` so `go test ./...` cannot compile it in.
- Plus a belt-and-braces refusal in `reset()` when the target database already contains rows in
  `users`, or when the host is not localhost.

`make test` must stay non-destructive no matter what is exported. The CODEMAP sentence claiming
`make test` "needs no live services" must then be true unconditionally.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 9 Step 2 — the file is the plan's verbatim text, so this is a plan defect, not an execution one).
- `backend/internal/store/integration_test.go:11-23` (`requirePostgres`, skips only on empty `DATABASE_URL`),
  `:27-40` (`reset`, the drops), `:44`, `:80` (called before any other assertion).
- `harness/CODEMAP.md` store bullet — instructs exporting `DATABASE_URL` and calls `make test`
  "no live services needed" in the same sentence.
- Reproduced by the reviewer without a live database (the tests run instead of skipping; only the
  dial fails):
```
$ DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    env -u REDIS_URL go test ./internal/store/ -count=1 -run TestIntegration -v
=== RUN   TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
    integration_test.go:44: down migration: failed to connect to `user=u database=db`: …
--- FAIL: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.00s)
```
  With a reachable database that `down migration` succeeds and the schema is gone.
