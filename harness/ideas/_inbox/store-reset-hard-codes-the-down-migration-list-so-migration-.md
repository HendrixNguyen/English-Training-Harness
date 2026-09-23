---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# store reset hard-codes the down-migration list so migration 0003 will silently not roll back

## Why
`reset()` in the store integration tests is the destructive helper every integration test calls to
get an empty database. Migration `0002` turned its single hard-coded down file into a hard-coded
*list*:

```go
for _, name := range []string{"migrations/0002_google_sync.down.sql", "migrations/0001_init.down.sql"} {
```
(`backend/internal/store/integration_test.go:51`)

Nothing derives that list from `MigrationsFS`, and nothing fails when it drifts. Migration `0003`
will be applied by `Migrate` (which *does* walk the FS) but never rolled back by `reset`, so its
table survives between tests carrying rows from the previous one. That is the classic
order-dependent-test bug: it will not fail on the commit that introduces it, it will fail later,
somewhere else, intermittently.

Two more assertions carry the same hard-coded knowledge and will need hand-editing for every future
migration: `integration_test.go:76` (`want := []string{"0001_init", "0002_google_sync"}`),
`integration_test.go:144` (`const versions = 2`) and `migrations_test.go:124`. The executor did
update all of them correctly and recorded the trap in CODEMAP ("a new migration means updating
both") — documenting a trap is better than leaving it silent, but the trap itself is avoidable.

## Expected output
`reset()` enumerates `migrations/*.down.sql` from `MigrationsFS` and executes them in reverse
filename order, so adding a migration needs no edit here. The version-count assertions derive the
expected list from `MigrationsFS` the same way `Migrate` does (a helper returning the sorted version
names), so `const versions = 2` and the two literal slices disappear. CODEMAP's "a new migration
means updating both" note is then removed rather than maintained.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), Task 1 Step 4.
- `backend/internal/store/integration_test.go:51` — the hard-coded down-file list.
- `backend/internal/store/integration_test.go:76,144` — hard-coded version list and count.
- `backend/internal/store/migrations_test.go:124` — same literal.
- `harness/CODEMAP.md` → `store` — the newly added sentence documenting the manual step.
- Related but distinct, already in the inbox: `migrate-is-only-tested-against-the-single-embedded-migration.md` (that one is about coverage of multi-version runs, which `0002` has now supplied).
