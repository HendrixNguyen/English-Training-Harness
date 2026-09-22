---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# no CI runs the integration suite so PgMigrator is never verified

## Why
The slice's test strategy is "fakes by default, live-service assertions skip without
`DATABASE_URL`/`REDIS_URL`". The skipping half is honest — it prints why it skipped and does not
pass silently. What is missing is the other half: nothing in the repo ever runs the skipped tests.
There is no `.github/workflows`, no CI config of any kind, and no hook; `make up && export … &&
make test-integration` is a manual step no document lists as required before a change.

The result is that every line of code that actually speaks to Postgres or Redis has zero automated
coverage. Measured on the default `go test ./... -count=1`:

- `PgMigrator.EnsureVersionTable`, `.AppliedVersions`, `.Apply` — 0%
- `Postgres.Ping`, `Postgres.Close`, `Postgres.Migrator` — 0%
- `Redis.Ping` — 0% (nothing in the default suite pings anything)
- `NewPostgres` — 42.9% (only the `ParseConfig` failure branch)
- package `internal/store` overall — 45.7%; `cmd/api` — 0%

The unit tests that "cover" the migration are substring checks against the embedded `.sql` text
(`migrations_test.go:21-86`, `postgres_test.go:15-21`). They never execute the SQL, so a migration
that is invalid Postgres DDL, or an inverted `CHECK`, passes them. Every later slice builds on
`store`, so this gap compounds.

## Expected output
Running the integration half stops depending on someone remembering. Either:
- a CI workflow that starts the `docker-compose` services (or GitHub Actions service containers),
  exports the URLs and runs `make test-integration` alongside `make test`; or
- a documented, single pre-merge command that does the same locally, named in `AGENTS.md`/CODEMAP as
  required before a store change is reviewed.

Either way the reviewer's re-verification step can run the live assertions rather than reading them,
and `PgMigrator` stops shipping unverified.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 9; *Verification* section runs only the no-services suite).
- `backend/internal/store/integration_test.go:14-16`, `:105-107` — the two skip gates.
- `backend/internal/store/postgres.go:37,40,43,48,53,73`; `backend/internal/store/redis.go:27` — the
  0%-coverage functions.
- Coverage measured in the plan's worktree with `go test ./... -covermode=set` and
  `DATABASE_URL`/`REDIS_URL` unset.
- Repo-wide search found no CI configuration; the only YAML under `backend/` is `docker-compose.yml`.
