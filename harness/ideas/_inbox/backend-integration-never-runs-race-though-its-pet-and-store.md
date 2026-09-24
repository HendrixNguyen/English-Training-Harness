---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# backend-integration never runs -race though its pet and store tests are concurrency tests

## Why
The gofmt/race plan put `-race` on `backend-unit` only and left `backend-integration` a plain run,
on the premise that "its tests are sequential by design". They are not. Three integration tests
exist precisely to drive shared Go code from many goroutines at once, against real Postgres:

- `internal/pet/integration_test.go:53` — 8 concurrent `Service.Ensure` calls on one user.
- `internal/pet/integration_test.go:175` — 8 concurrent `PgRepo.PenaliseMiss` calls for one day
  (the exactly-once guarantee the hourly cron depends on).
- `internal/store/integration_test.go:129` — 8 concurrent `Migrate` calls
  (`TestIntegrationConcurrentMigrateDoesNotRace`).

These paths (pgx pools, `Service`, `PgMigrator`'s pinned-connection lock) are exactly where a Go-level
data race would live, and the unit suite cannot reach them — they skip without `TEST_DATABASE_URL`.
So the only tests that concurrently exercise the pet engine's and migrator's real repositories are the
ones CI never runs under the race detector. A race introduced there on a `harness/*` branch stays green.

Cost is negligible: the reviewer ran the whole integration suite under `-race` locally and it finished in
seconds (`-p 1` already serialises packages; `-race` does not change that).

## Expected output
- `.github/workflows/ci.yml` → `backend-integration`: `go test ./... -count=1 -v -run Integration -p 1 -race`
  (the PASS/SKIP counting below it is unchanged).
- `backend/Makefile` → `test-integration` carries `-race` too, so the local mirror matches.
- `harness/CODEMAP.md` → CI → `backend-integration` names `-race` and why (the three concurrency tests).
- The branch's CI run shows the integration step green with `-race` in its command line.

## Evidence
- Plan under review: `harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`
  → *Notes and open questions* → "`-race` and integration tests: unchanged … its tests are sequential by design".
- `grep -n 'go func' backend --include='*_test.go'` → the three integration sites above (plus unit ones already covered).
- Reviewer, 2026-09-24, branch `harness/2026-09-24-medium-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a`
  at `76bb6ca`, isolated stack (`COMPOSE_PROJECT_NAME=rev-ci`, Postgres 55445, Redis 56395):
  `TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./... -count=1 -v -run Integration -p 1 -race -timeout 300s`
  → exit 0, 12/12 `--- PASS: TestIntegration*`, no `DATA RACE`, every package ≤ 1.8 s.
- CI run 35959338732, `backend-integration` log: `go test ./... -count=1 -v -run Integration -p 1` (no `-race`), `12/12 integration tests ran and passed`.
