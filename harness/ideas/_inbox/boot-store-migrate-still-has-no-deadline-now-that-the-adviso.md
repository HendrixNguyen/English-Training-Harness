---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Boot store.Migrate still has no deadline now that the advisory-lock leak is fixed

## Why
The advisory-lock plan made `PgMigrator.Lock` wait at most `MigrationLockWait` (30 s) and made unlock safe on a cancelled context. The reason `cmd/api/main.go` runs its boot `store.Migrate` with no deadline, the leaked lock, is now gone. The plan deferred the boot deadline to keep `cmd/api` out of its conflict fence and asked the reviewer to file it. Today a boot whose DDL hangs, for example a migration blocked on another session's table lock (which `MigrationLockWait` does not bound), still waits forever with no log. The CODEMAP `cmd/api` sentence ("deadline-free on purpose until the advisory-lock unlock leak is fixed") is stale once the lock branch merges.

## Expected output
- `cmd/api/main.go` wraps the boot `store.Migrate` in `context.WithTimeout(ctx, <bound>)`, for example 2 minutes, generous for DDL on Supabase's pooler. On timeout it exits non-zero with a message naming the migration step.
- The CODEMAP `cmd/api` sentence describes the bound, and the "deadline-free … until the leak is fixed" clause is removed.

## Evidence
- Plan: `harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md`, *Global Constraints* ("The boot-migration deadline in `cmd/api/main.go` … is deferred … for the reviewer to file") and the Execution summary's *Follow-ups*.
- `backend/cmd/api/main.go:81` — `applied, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` on the signal context, with no deadline.
- `harness/CODEMAP.md:21` — "`store.Migrate` (deadline-free on purpose until …".
