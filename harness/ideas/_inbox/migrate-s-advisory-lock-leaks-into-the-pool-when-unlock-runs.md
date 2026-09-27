---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md
---
# Migrate's advisory lock leaks into the pool when unlock runs on a cancelled context

## Why
`PgMigrator.Lock` returns an `unlock` closure that runs `SELECT pg_advisory_unlock($1)` **on the
caller's context** and discards the result (`_, _ = conn.Exec(ctx, ...)`), then calls
`conn.Release()`. If that context is already cancelled or past its deadline when `Migrate` returns,
the unlock statement never reaches Postgres, the error is swallowed, and the connection goes back
into the pool still holding a session-level advisory lock.

pgxpool does not reset session state on release: `pgxpool.Conn.Release` only destroys a connection
that is closed, busy, or inside a transaction, and a session holding an advisory lock is none of
those. So the lock stays held until `MaxConnLifetime` (pgx default: 1 hour) recycles that pooled
connection, or the process exits. Every subsequent `store.Migrate` against that database — in this
process or any other — blocks in `pg_advisory_lock` for up to an hour with no log line.

Nothing on this branch passes a cancellable context to `Migrate` today (`cmd/api/main.go` uses
`context.Background()`), so this is latent rather than live. It stops being latent the moment the
open inbox idea `migrations-run-on-every-boot-with-no-advisory-lock.md` gets its secondary ask —
"`main.go` should not use a deadline-free `context.Background()` for the migration" — because a
migration that exceeds its new deadline will poison the pool on the way out and the next boot will
hang instead of failing fast. The two changes are individually reasonable and jointly a trap.

## Expected output
`unlock` releases the advisory lock regardless of the caller's context, and never returns a still-locked
connection to the pool:

- run the unlock on `context.WithoutCancel(ctx)` (Go 1.21+) or a fresh short-deadline context, not on `ctx`;
- if the unlock still errors, destroy the connection instead of releasing it (`conn.Conn().Close(...)`,
  or `conn.Hijack()` then close) so the session — and with it the lock — ends;
- log the unlock error rather than discarding it, since a silently-held migration lock is the single
  worst failure mode this code has.

A regression test: take the lock with a cancellable context, cancel it, call `unlock()`, then assert a
second caller acquires the lock within a short deadline (or that
`SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND objid=<key>)` is false).

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md` (third deviation — `internal/store` was outside the plan's file list).
- `backend/internal/store/postgres.go:69-72` — `return func() { _, _ = conn.Exec(ctx, ...); conn.Release() }`.
- `backend/internal/store/migrations.go:38-44` — `defer unlock()` inside `Migrate`.
- pgx v5.11.0 `pgxpool/conn.go:32` — `Release` destroys only on `IsClosed() || IsBusy() || TxStatus() != 'I'`;
  no `DISCARD ALL`, so advisory locks survive release.
- Reproduced in the plan's worktree against a live Postgres (a temporary probe test, since removed):
  after `cancel(); unlock()`, `pg_locks` still showed the advisory lock held, and a second
  `PgMigrator.Lock` with a 3s deadline returned
  `store: acquiring the migration lock: timeout: context deadline exceeded`.
- Related open idea: `harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium (was high).** No longer latent: `main.go` now passes the `signal.NotifyContext` context to `store.Migrate`, so a SIGTERM during a boot migration runs `unlock` on a cancelled context and leaves the advisory lock in the pool for up to an hour — though only for that narrow window, which is why medium rather than high. Plan scope (folding `migrate-s-advisory-lock-can-hang-boot-forever-with-no-bound-.md`): unlock on `context.WithoutCancel`, destroy the connection if unlock fails, log it; bounded lock wait with a contention log line; statements on the pinned connection; regression test against a live Postgres. **Ordering:** the cmd/api hardening plan deliberately does *not* add a migration deadline; add it here, after the leak is fixed.

_Evaluator, 2026-09-27 — daily decide (bug queue, planned today as **B4**)._

**Confirmed on `origin/main`, still medium.** Root cause is unchanged: `backend/internal/store/postgres.go:67-70` — the `unlock` closure runs `SELECT pg_advisory_unlock($1)` on the *caller's* `ctx`, discards the error, then `conn.Release()`; `migrations.go:56` defers it, and `cmd/api/main.go:81` now passes the `signal.NotifyContext` context, so a SIGTERM during a boot migration is the live path. pgx v5.11.0 `pgxpool/conn.go:32` destroys a released connection only when it is closed, busy or inside a transaction — a session holding an advisory lock is none of those, so the lock rides back into the pool until `MaxConnLifetime` (1 h) or process exit, and every later `Migrate` against that database blocks in `pg_advisory_lock`. Checked: the three store test files, the eight `store.Migrate` call sites (all pass `context.Background()` except `main.go`), and that nothing sets `pool_max_conns` today (so the folded `pool_max_conns=1` deadlock is latent). **Fix decision (smallest correct):** (1) unlock on `context.WithTimeout(context.WithoutCancel(ctx), 5 s)` and scan the boolean; on error or `false`, log it and `Hijack()`+`Close` the connection so the session — and the lock — ends instead of returning to the pool; (2) the folded bounded-wait ask: `pg_try_advisory_lock` first, and only on contention log once and block in `pg_advisory_lock` under a `MigrationLockWait` (30 s) deadline when the caller's context has none; (3) the folded `pool_max_conns=1` ask: `PgMigrator` runs `EnsureVersionTable`/`AppliedVersions`/`Apply` on the pinned connection while locked. Regression tests live in a **new** `migrate_lock_test.go` gated exactly like `integration_test.go` (`requirePostgres`, `TestIntegration…` names) because the pool behaviour needs a live Postgres; nothing in `migrations_test.go`/`integration_test.go` is edited (both are rewritten on the unmerged 2026-09-26 pet-shields and RLS branches). **Deferred:** the boot-migration deadline in `cmd/api/main.go` (and the CODEMAP `cmd/api` sentence that says it waits for this fix) — `cmd/api` is edited by unmerged branches; the executor lists it as a follow-up for the reviewer to file once the 2026-09-26 daily PR merges. Plan: `harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md`, approved (bug auto-approve).
