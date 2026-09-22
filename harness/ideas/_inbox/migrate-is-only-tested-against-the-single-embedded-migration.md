---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Migrate is only tested against the single embedded migration

## Why
`Migrate` exists to apply *several* migrations in order and skip the ones already applied. Every test
of it passes the real `MigrationsFS`, which contains exactly one file, so the behaviour that matters
is never exercised:

- `sort.Strings(names)` (`migrations.go:46`) runs on a one-element slice — ordering across
  `0002_…`, `0010_…` is unverified.
- The `if done[version] { continue }` branch (`migrations.go:51-53`) is only hit in the degenerate
  case where the *only* migration is already applied (`TestMigrateIsIdempotent`). The realistic
  upgrade case — one applied, one pending, apply exactly the pending one — has no test.
- Four of the five error paths are dead: `EnsureVersionTable` failure (`:35-36`), `AppliedVersions`
  failure (`:39-40`), `fs.Glob` failure (`:44-45`) and `fs.ReadFile` failure (`:56-57`). The fake
  migrator always returns `nil` from the first two (`migrations_test.go:95-106`) and every call site
  passes the real embedded FS.
- `TestMigrateReportsApplyFailure` (`migrations_test.go:163-169`) asserts only `err != nil`; the
  partial `applied` slice returned on a mid-loop failure (`migrations.go:59`) is never checked.

Slice 2 will add `0002_…`. The first time ordering or partial application matters, it will be in
production with no test behind it. Related: `TestMigration0001DownDropsEverything`
(`migrations_test.go:82-85`) checks only that `exercises` is dropped before `roadmaps`; reordering
`DROP TYPE cefr_level` above `DROP TABLE users` would still pass, because the assertions are
substring presence, not execution.

## Expected output
`Migrate` is tested against an in-memory `fs.FS` (`fstest.MapFS`) the tests control, covering at
minimum:
- two or more migrations applied in filename order, asserted by the order of `Apply` calls;
- one already in `AppliedVersions` and one pending — only the pending one is applied;
- `EnsureVersionTable` and `AppliedVersions` returning errors — `Migrate` wraps and returns them, and
  applies nothing;
- `ReadFile` failing mid-loop — the returned `applied` slice contains the versions that did succeed;
- the down file's `DROP TYPE` statements asserted to come after `DROP TABLE users`.

`Migrate` already takes `fs.FS`, so none of this needs a database or a change to production code.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` (Task 4).
- `backend/internal/store/migrations.go:33-63`.
- `backend/internal/store/migrations_test.go:117-169` — all four `Migrate` tests pass `MigrationsFS`.
- Coverage profile in the plan's worktree shows zero hits on `migrations.go:35-36,39-40,44-45,56-57`.
