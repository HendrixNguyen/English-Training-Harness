---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Integration gate tests only prove the skip and would pass if the gate always skipped

## Why
`backend/internal/store/integration_gate_test.go` is the automated proof for the safety property
introduced by `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md`:
that `go test ./...` can never drop tables just because the production `DATABASE_URL` is exported.
Both of its tests assert only one direction. They set the `TEST_*` variable to `""` and check that
`requirePostgres` / `requireRedisURL` skip:

```
11:	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2")
12:	t.Setenv("TEST_DATABASE_URL", "")
...
20:	if !skipped {
21:		t.Fatal("requirePostgres must skip when TEST_DATABASE_URL is unset")
```

Nothing asserts the other direction, so the suite cannot distinguish a working gate from a broken
one that skips unconditionally. Replace either helper body with a bare `t.Skip("…")` and the whole
default suite stays green — and so does `make test-integration`, which would then report three
`SKIP`s and `ok`, exactly as it does for a correctly-gated run with the variables unset. The three
integration tests would be silently dead, and the two assertions this repo has about the §3.2
migration actually executing (`TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent`,
`TestIntegrationPetStatesRejectsASecondRowForTheSameUser`) would never run again without anyone
noticing.

That is the classic vacuous-pass shape: a test that only ever observes the skip branch. It matters
more than usual here because the gate's whole job is to decide whether the destructive tests run, it
is the pattern every later slice will copy for its own live-service tests, and — per the already
filed `no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md` — no CI runs the positive
path either. The reviewer had to start the dev stack by hand and observe three `--- PASS` lines to
confirm the gate still lets a nominated run through.

## Expected output
`integration_gate_test.go` pins both directions, still with no live service required:

- The existing skip cases stay as they are.
- Two new cases set `TEST_DATABASE_URL` / `TEST_REDIS_URL` to a syntactically valid but unreachable
  address (the same `127.0.0.1:59999` shape already used in the file, with a short
  `connect_timeout`) and assert the helper **does not skip** — i.e. it proceeds past the gate. The
  Postgres case reaches `NewPostgres`, which parses and returns a lazy pool without dialling
  (`postgres.go:22-34`), so the subtest observes "not skipped" quickly and deterministically; the
  Redis case simply asserts the returned URL equals what was set. Neither case may call `reset` or
  touch a database.
- A mutation check is recorded in the amending plan's verification: replacing either helper body
  with an unconditional `t.Skip` makes at least one test fail.
- The `reset` refusal at `integration_test.go:46-48` gains its own direct test — currently it is
  reachable only through `requirePostgres`, so nothing proves the second line of defence works on
  its own.

## Evidence
- Plan under review: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
- Fix plan that introduced the gate and these tests:
  `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md` (Task 1).
- `backend/internal/store/integration_gate_test.go:8-23` and `:26-39` — both tests assert only
  `t.Skipped()`.
- `backend/internal/store/integration_test.go:14-37` — the helpers under test; `:46-48` — the
  untested `reset` refusal.
- Reviewer verification: the positive direction had to be proved manually —
  `POSTGRES_PORT=5433 REDIS_PORT=6380 make up`, then `make test-integration` with
  `TEST_DATABASE_URL`/`TEST_REDIS_URL` exported, gave `--- PASS` for all three integration tests.
  Nothing in the repo performs that check automatically.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Test-only, but the gate is the thing that keeps `go test` from dropping a production database, so a positive-direction test is worth having. CI's `backend-integration` job now fails on any `--- SKIP`, which is the outer guard the reviewer wanted; the unit-level assertion is the remaining half. Batch with the 0003 migration plan (same package, same test files).
