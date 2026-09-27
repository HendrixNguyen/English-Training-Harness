---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Migration lock wait log and error always quote 30s even when the caller deadline or a cancel is the bound

## Why
`PgMigrator.Lock` sets `wait := MigrationLockWait` and uses it in both the contention log (`waiting up to 30s`) and the failure error (`another process held it for longer than 30s: …`), even when that is not the bound:
- When the caller's context already has a deadline (tests, a future bounded boot), the real bound is that deadline. The message still says 30 s.
- When the caller is cancelled (a SIGTERM during boot), the error reads `another process held it for longer than 30s: context canceled` after perhaps 200 ms. That tells the operator to hunt for a stuck migrator that may not exist.

The plan asked for "`wait` is the effective bound".

## Expected output
- `wait` is `time.Until(deadline)` when `ctx` has a deadline, otherwise `MigrationLockWait`.
- A cancellation (`errors.Is(err, context.Canceled)`) returns `store: acquiring the migration lock: cancelled while waiting: %w` rather than the "held it for longer than" wording.
- A unit or integration test covers a caller deadline shorter than `MigrationLockWait`.

## Evidence
- Plan: `harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md`, Task 2 Step 2 ("where `wait` is the effective bound").
- `backend/internal/store/postgres.go` `Lock`: `wait := MigrationLockWait`, then `log.Printf("… waiting up to %s", …, wait)` and `fmt.Errorf("… held it for longer than %s: %w", wait, err)`, independent of `ctx.Deadline()`.
- Reviewer run of `TestIntegrationConcurrentMigrateDoesNotRace`: seven `store: migration lock 727100001 is held by another process; waiting up to 30s` lines, although the whole test ran in 0.11 s.
