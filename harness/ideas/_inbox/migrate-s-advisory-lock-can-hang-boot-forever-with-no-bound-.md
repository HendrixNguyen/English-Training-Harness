---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: medium
rejected_reason: "Folded into migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md (survivor): one PgMigrator plan makes the unlock context-independent, bounds the lock wait (pg_try_advisory_lock loop or lock_timeout), logs contention, and runs the migration statements on the pinned connection so pool_max_conns=1 cannot deadlock."
---
# Migrate's advisory lock can hang boot forever with no bound and no log

## Why
The new `PgMigrator.Lock` uses `pg_advisory_lock`, the **blocking** variant, and `cmd/api/main.go`
calls `store.Migrate` with a deadline-free `context.Background()`. Two consequences, both of which
turn a loud failure into a silent one:

1. **A stuck migration now hangs every other replica.** Before this change, two replicas booting
   onto a fresh database raced and the loser crash-looped with a visible
   `duplicate key value violates unique constraint` — ugly, but diagnosable in the platform logs.
   Now the loser blocks inside `pg_advisory_lock` indefinitely, printing nothing between
   "connected" and whatever comes after the migration. Spec §8 deploys to Railway, where a replica
   that never finishes booting and never logs why is materially harder to debug than one that
   crash-loops.

2. **`pool_max_conns=1` deadlocks the boot outright.** `Lock` pins one connection from `m.pool` for
   the whole migration run, while `EnsureVersionTable`, `AppliedVersions` and `Apply` each need a
   *second* connection from the same pool. pgx parses `pool_max_conns` out of the connection string,
   so a `DATABASE_URL` tuned down to one connection — a plausible thing to do on a small Railway
   plan — makes `Migrate` wait forever for a connection it is itself holding. Confirmed against a
   live database: with `pool_max_conns=1` and a 5s deadline, `Migrate` returned
   `store: ensuring version table: context deadline exceeded`; with `main.go`'s `context.Background()`
   it would simply never return.

## Expected output
Migration acquires its lock in bounded time and says what it is doing:

- use `pg_try_advisory_lock` in a retry loop with an overall budget, or `SET lock_timeout` on the
  pinned connection, so waiting for the lock cannot exceed a known bound;
- log once when the lock is contended ("another instance is migrating; waiting…") and again on
  acquisition, so a stuck boot is visible in the platform logs;
- give the boot migration in `cmd/api/main.go` a deadline (`context.WithTimeout`), which the
  existing inbox idea already asks for — but see the sibling bug about `unlock` leaking the lock on
  a cancelled context, which must be fixed first or the deadline makes things worse;
- run the migration statements on the same pinned connection as the lock, or document a minimum
  pool size, so `pool_max_conns=1` cannot deadlock.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` (third deviation).
- `backend/internal/store/postgres.go:59-75` — `m.pool.Acquire` then `SELECT pg_advisory_lock($1)`, blocking, no timeout.
- `backend/internal/store/migrations.go:38-51` — lock held across `EnsureVersionTable` / `AppliedVersions` / `Apply`, all of which use `m.pool`.
- `backend/cmd/api/main.go:19` (`ctx := context.Background()`), `:38-41` (`store.Migrate(ctx, ...)` then `log.Fatalf`).
- Reproduced in the plan's worktree (temporary probe test, since removed): `NewPostgres(ctx, url+"&pool_max_conns=1")`
  then `Migrate` with a 5s deadline → `store: ensuring version table: context deadline exceeded`.
- Spec §8 "Deploy compiled Go binary as lightweight Docker container on Railway."
- Overlaps `harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md` (its secondary ask).

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — folded.** Same struct, same plan as the unlock-leak finding; the two must land together (a bounded wait without the leak fix makes the leak live). Scope recorded in the survivor's Evaluation.
