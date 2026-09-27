---
idea: harness/ideas/_inbox/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md
status: approved
priority: medium
merged: false
---
# Migrate's advisory lock leaks into the pool when unlock runs on a cancelled context — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B4** of 2026-09-27. **Estimate:** 3 h. **Branch:** `harness/2026-09-27-medium-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs`.

**Idea:** `harness/ideas/_inbox/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md` (primary). Folded: `harness/ideas/_inbox/migrate-is-only-tested-against-the-single-embedded-migration.md` (Task 4) and the already-rejected-into-this-one `migrate-s-advisory-lock-can-hang-boot-forever-with-no-bound-.md` (Tasks 2–3). Backend only; **no design doc**.

**Goal:** `PgMigrator` never returns a connection that still holds the migration advisory lock to the pool, never blocks a boot forever without saying why, and cannot deadlock itself on a one-connection pool — proven by live-Postgres regression tests in a new file, with `Migrate`'s own logic covered over an in-memory `fs.FS`.

**Root cause (confirmed on `origin/main`):** `backend/internal/store/postgres.go:67-70` — the `unlock` closure runs `SELECT pg_advisory_unlock($1)` on the *caller's* context, discards the result, then `conn.Release()`. `migrations.go:56` defers it, and `cmd/api/main.go:81` passes the `signal.NotifyContext` context, so a SIGTERM during a boot migration cancels the unlock; pgx v5.11.0 `pgxpool/conn.go:32` destroys a released connection only when it is closed, busy or in a transaction, so the session goes back into the pool with the lock held until `MaxConnLifetime` (1 h). Every later `Migrate` against that database then blocks in `pg_advisory_lock` (blocking variant, no bound, no log — the folded finding), and `EnsureVersionTable`/`AppliedVersions`/`Apply` need a *second* pooled connection while `Lock` pins one, so `pool_max_conns=1` deadlocks outright.

**Architecture:** all three fixes stay inside `PgMigrator` (`postgres.go`); `Migrate` and the `Migrator`/`Locker` interfaces do not change shape. A `pgQuerier` interface (`Exec`, `Query`, `Begin`) is satisfied by both `*pgxpool.Pool` and `*pgxpool.Conn`, so the migrator's statements run on the pinned connection while locked and on the pool otherwise. Timeouts are package-level `var`s (`MigrationLockWait`, `migrationUnlockTimeout`) so tests can shorten them. Logging uses the standard `log` package like `pet/cron.go` and `main.go`.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` are not installed — use `grep -n`, `go test -timeout`; integration tests via `COMPOSE_PROJECT_NAME=<slug> POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait` … `make down` with the same project.
- **Conflict fence (the 2026-09-26 review run never ran, 21 `done` plans are unmerged):** touch only `backend/internal/store/postgres.go`, `backend/internal/store/migrations.go`, the two **new** test files `backend/internal/store/migrate_lock_test.go` and `backend/internal/store/migrate_fs_test.go`, and the `store` paragraph of `harness/CODEMAP.md`. **Never edit** `migrations_test.go`, `integration_test.go` (both rewritten on the unmerged pet-shields and RLS branches), anything under `migrations/` (no migration today), or `cmd/api/`. If a fix seems to need one of those files, stop and report instead.
- New test files define their **own** fakes with distinct names (`fsMigrator`, not `fakeMigrator`) and only *call* the existing helpers `requirePostgres(t)` / `reset(t, pg)` from `integration_test.go`; they never hard-code the number of embedded migrations (both unmerged branches add a `0004_*`) — derive expectations from `fs.Glob(MigrationsFS, "migrations/*.up.sql")`.
- Postgres-backed tests follow the existing gate exactly: named `TestIntegration…`, first line `pg := requirePostgres(t)` (skips without `TEST_DATABASE_URL`, never reads `DATABASE_URL`); CI's `backend-integration` counts every `func TestIntegration*` and fails on a skip, so they must pass there.
- The boot-migration deadline in `cmd/api/main.go` (and the CODEMAP `cmd/api` sentence "deadline-free on purpose until the advisory-lock unlock leak is fixed") is **deferred** — `cmd/api` is unsafe today. List it under *Follow-ups* in the Execution summary for the reviewer to file.
- `gofmt -l internal/store` empty; `go vet ./...` clean; the unit suite must pass under `-race` with no services (`TEST_*` unset).

## Review Focus
1. After `cancel(); unlock()` no advisory lock with `objid = 727100001` remains in `pg_locks`, and a second `PgMigrator` from a *different* pool acquires the lock within 3 s.
2. `unlock` never `Release()`s a connection whose unlock failed or returned `false` — that connection is hijacked and closed, and the error is logged, never discarded.
3. `Lock` on a context with no deadline is bounded by `MigrationLockWait`; contention is logged once before waiting and once on acquisition; the timeout error names the lock and the wait.
4. While locked, `EnsureVersionTable`, `AppliedVersions` and `Apply` run on the pinned connection — `Migrate` succeeds on `pool_max_conns=1` with a 10 s deadline.
5. `Migrate` over an `fstest.MapFS`: filename order, applied-vs-pending, wrapped errors that apply nothing, partial `applied` on a mid-loop failure, malformed names refused.
6. Nothing outside the conflict fence changed (`git diff --stat origin/main..HEAD` lists only the five allowed paths).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/postgres.go` | `Lock`: try-lock + bounded blocking wait + contention log; unlock on an uncancellable bounded context, hijack+close on failure; `pgQuerier` + pinned-connection statements |
| `backend/internal/store/migrations.go` | `versionOf(name)` guard (malformed migration names refused); doc comment on `Locker` updated |
| `backend/internal/store/migrate_lock_test.go` (new) | `TestIntegrationUnlockReleasesTheLockOnACancelledContext`, `TestIntegrationLockWaitIsBoundedAndLogged`, `TestIntegrationMigrateSucceedsWithPoolMaxConnsOne`, plus a pure unit test for the `pgQuerier` selection |
| `backend/internal/store/migrate_fs_test.go` (new) | `Migrate` over `fstest.MapFS` with an `fsMigrator` recording fake; 0001 down-file ordering |
| `harness/CODEMAP.md` → `store` paragraph | lock semantics, bounded wait, pinned-connection statements, the new test files |

## Tasks

### Task 1: Unlock releases the lock regardless of the caller's context

**Files:** `backend/internal/store/postgres.go`, `backend/internal/store/migrate_lock_test.go` (new).

- [ ] **Step 1 (tests first):** create `migrate_lock_test.go` (package `store`, imports `context`, `os`, `testing`, `time`). Write `TestIntegrationUnlockReleasesTheLockOnACancelledContext`: `pg := requirePostgres(t)`; `m := pg.Migrator().(*PgMigrator)`; `ctx, cancel := context.WithCancel(context.Background())`; `unlock, err := m.Lock(ctx)` (fatal on err); `cancel()`; `unlock()`. Assert (a) `SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND classid = 0 AND objid = $1 AND granted` with `int64(migrationLockKey)` on `pg.Pool` is `0` (the key is < 2³², so `classid` is 0 and `objid` is the key); (b) a second `NewPostgres(ctx2, os.Getenv("TEST_DATABASE_URL"))` (own pool, closed via `t.Cleanup`) can `Lock` under `context.WithTimeout(…, 3*time.Second)` and its `unlock()` runs clean. Run `go test ./internal/store -run TestIntegrationUnlockReleases -count=1 -v` against the compose database — it must **fail** on (a)/(b) before the fix (this is the reviewer's reproduction).
- [ ] **Step 2:** in `postgres.go` add `var migrationUnlockTimeout = 5 * time.Second` and rewrite the closure returned by `Lock`:
  ```go
  return func() {
      uctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), migrationUnlockTimeout)
      defer cancel()
      var released bool
      err := conn.QueryRow(uctx, `SELECT pg_advisory_unlock($1)`, int64(migrationLockKey)).Scan(&released)
      if err == nil && released {
          conn.Release()
          return
      }
      // The session may still hold the lock: end the session instead of handing it back to the pool.
      log.Printf("store: releasing the migration lock failed (released=%v, err=%v); closing the connection", released, err)
      pc := conn.Hijack()
      _ = pc.Close(uctx)
  }, nil
  ```
  (`context.WithoutCancel` — Go 1.21+, module is Go 1.25.) Add the `log` and `time` imports. Keep the doc comment on `Lock` and extend it with one sentence on why unlock ignores the caller's cancellation.
- [ ] **Step 3:** `go test ./internal/store -run TestIntegrationUnlockReleases -count=1 -v` passes; `go test ./internal/store -count=1 -race` (no `TEST_*` exported) still passes and the new test **skips**. Commit: `store: release the migration advisory lock even on a cancelled context`.

### Task 2: Bounded lock wait with a contention log

**Files:** `backend/internal/store/postgres.go`, `backend/internal/store/migrate_lock_test.go`.

- [ ] **Step 1 (tests first):** add `TestIntegrationLockWaitIsBoundedAndLogged`: shorten `MigrationLockWait` to 1 s for the test (save/restore with `t.Cleanup`); holder pool `Lock`s and keeps the lock; a second `NewPostgres` calls `Lock(context.Background())` — must return an error within ~2 s (`time.Since(start) < 3*time.Second`) whose message contains `migration lock` and `1s`; capture `log` output via `log.SetOutput(&buf)` (restore in `t.Cleanup`) and assert it contains `waiting`. Then the holder's `unlock()` runs and a third `Lock` with a 3 s deadline succeeds and is **not** logged as contended (buffer unchanged since reset).
- [ ] **Step 2:** in `postgres.go` add `var MigrationLockWait = 30 * time.Second` (exported, documented: "how long Lock waits for another process's migration before failing when the caller's context has no deadline"). In `Lock` after `Acquire`: `SELECT pg_try_advisory_lock($1)` → `Scan(&got)`; if `got` return the unlock closure at once. Otherwise `log.Printf("store: migration lock %d is held by another process; waiting up to %s", migrationLockKey, wait)`, build `wctx := ctx` and, if `ctx.Deadline()` reports none, `wctx, cancel = context.WithTimeout(ctx, MigrationLockWait)` (`defer cancel()`), then `conn.Exec(wctx, `SELECT pg_advisory_lock($1)`, …)`; on error `conn.Release()` (the session holds nothing yet — a cancelled `pg_advisory_lock` acquires nothing) and return `fmt.Errorf("store: acquiring the migration lock: another process held it for longer than %s: %w", wait, err)` where `wait` is the effective bound; on success `log.Printf("store: migration lock acquired after %s", time.Since(start).Round(time.Millisecond))`. Any error from the try-lock itself releases the conn and returns as today.
- [ ] **Step 3:** `go test ./internal/store -run 'TestIntegrationLockWait|TestIntegrationUnlockReleases|TestIntegrationConcurrentMigrate' -count=1 -v` (the existing concurrent test must still pass: 8 callers, contention logged, all finish well inside 30 s). Commit: `store: bound the migration lock wait and log contention`.

### Task 3: Migration statements run on the pinned connection

**Files:** `backend/internal/store/postgres.go`, `backend/internal/store/migrate_lock_test.go`.

- [ ] **Step 1 (tests first):** add `TestIntegrationMigrateSucceedsWithPoolMaxConnsOne`: `pg := requirePostgres(t)`; `reset(t, pg)` + `t.Cleanup(func() { reset(t, pg) })`; open a second `Postgres` from `TEST_DATABASE_URL` with `pool_max_conns=1` appended (`sep := "?"; if strings.Contains(url, "?") { sep = "&" }`); `Migrate` under a 10 s deadline must return nil and `len(applied)` must equal the number of `migrations/*.up.sql` in `MigrationsFS` (derive it with `fs.Glob`, never a literal). Before the fix this fails with `store: ensuring version table: context deadline exceeded` (the reviewer's reproduction). Also a pure unit test `TestPgMigratorUsesThePinnedConnectionWhileLocked`: a zero `PgMigrator{pool: p}` with `conn == nil` returns `p` from `db()`; with a non-nil `conn` field it returns that conn — assert by type switch (`*pgxpool.Pool` vs `*pgxpool.Conn`); construct the pool with `pgxpool.NewWithConfig` from `pgxpool.ParseConfig("postgres://localhost/x")` (lazy, no dial) and a `*pgxpool.Conn` literal `&pgxpool.Conn{}` (never used, only its type). If a `&pgxpool.Conn{}` literal is not constructible, drop the unit test and rely on the integration test — say so in the Execution summary.
- [ ] **Step 2:** in `postgres.go` add
  ```go
  // pgQuerier is what PgMigrator needs from either the pool or one pinned
  // connection: *pgxpool.Pool and *pgxpool.Conn both satisfy it.
  type pgQuerier interface {
      Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
      Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
      Begin(ctx context.Context) (pgx.Tx, error)
  }
  ```
  give `PgMigrator` a `conn *pgxpool.Conn` field, `func (m *PgMigrator) db() pgQuerier { if m.conn != nil { return m.conn }; return m.pool }`, set `m.conn = conn` after the lock is acquired and `m.conn = nil` at the top of the unlock closure (before release/hijack), and switch `EnsureVersionTable`, `AppliedVersions`, `Apply` from `m.pool.` to `m.db().`. Document on `PgMigrator` that one value serves one `Migrate` at a time (`Postgres.Migrator()` returns a fresh one per call, which every caller uses). Imports: `github.com/jackc/pgx/v5`, `github.com/jackc/pgx/v5/pgconn`.
- [ ] **Step 3:** `go test ./internal/store -run 'Integration|PgMigrator' -count=1 -v` green against compose; `go vet ./internal/store`. Commit: `store: run migration statements on the pinned lock connection so pool_max_conns=1 cannot deadlock`.

### Task 4: `Migrate` covered over an in-memory FS; malformed names refused (folded idea)

**Files:** `backend/internal/store/migrate_fs_test.go` (new), `backend/internal/store/migrations.go`, `harness/CODEMAP.md` (`store` paragraph only).

- [ ] **Step 1 (tests first):** create `migrate_fs_test.go` with `type fsMigrator struct { ensureErr, appliedErr error; done map[string]bool; failOn string; order []string }` implementing `Migrator` (records `Apply` versions in `order`). Helper `mapFS(names ...string) fstest.MapFS` building `migrations/<name>.up.sql` entries with body `-- <name>`. Tests, no database:
  - `TestMigrateAppliesInFilenameOrder`: entries added as `0010_ten`, `0002_two`, `0001_one` → `order == [0001_one 0002_two 0010_ten]` and the returned slice equals it.
  - `TestMigrateSkipsAppliedVersions`: `done = {0001_one}` → only `0002_two` applied.
  - `TestMigrateEnsureVersionTableErrorAppliesNothing` and `TestMigrateAppliedVersionsErrorAppliesNothing`: `errors.Is` unwraps to the sentinel; messages start `store: ensuring version table:` / `store: reading applied versions:`; `order` empty.
  - `TestMigrateReturnsPartialAppliedWhenReadFails`: `0002_two` stored as a directory entry (`&fstest.MapFile{Mode: fs.ModeDir}`) so `fs.Glob` lists it and `fs.ReadFile` fails → error mentions `reading migrations/0002_two.up.sql`, returned slice `== [0001_one]`.
  - `TestMigrateReturnsPartialAppliedWhenApplyFails`: `failOn = "0002_two"` with three files → error mentions `applying 0002_two`, returned slice `== [0001_one]`, `order == [0001_one]`.
  - `TestMigrateRefusesAMalformedMigrationName`: file `migrations/README.up.sql` beside `0001_one` → error `store: malformed migration name "migrations/README.up.sql"`, nothing applied (validation happens before the loop applies anything).
  - `TestMigration0001DownDropsTypesAfterTables`: read `migrations/0001_init.down.sql` from `MigrationsFS`; every `DROP TYPE` index is greater than the index of `DROP TABLE IF EXISTS users`.
- [ ] **Step 2:** in `migrations.go` add `func versionOf(name string) (string, error)` — trims `migrations/` and `.up.sql`, requires the form `^\d{4}_[a-z0-9_]+$` (`regexp.MustCompile` at package level), errors `store: malformed migration name %q`. In `Migrate`, after `sort.Strings(names)`, resolve every name to a version **first** (return the error before applying anything), then loop. The existing `MigrationsFS` files all match; the doc comment on `MigrationsFS` gains the naming rule.
- [ ] **Step 3:** CODEMAP `store` paragraph: replace the `PgMigrator also implements store.Locker (…)` clause with the new semantics — try-lock then bounded blocking wait (`MigrationLockWait`, 30 s, only when the caller's context has no deadline) with contention logged; unlock runs on an uncancellable 5 s context and hijacks+closes the connection if the lock cannot be released, so a cancelled boot never leaks the lock into the pool; statements run on the pinned connection while locked so `pool_max_conns=1` works; migration names must match `NNNN_name`; tests `migrate_lock_test.go` (integration) and `migrate_fs_test.go` (pure). Do not touch the `cmd/api` paragraph.
- [ ] **Step 4:** `go test ./internal/store -count=1 -race` (unit, no services) and `gofmt -l internal/store` empty. Commit: `store: test Migrate over an in-memory FS and refuse malformed migration names; CODEMAP`.

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
COMPOSE_PROJECT_NAME=<slug> POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait && TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable' TEST_REDIS_URL=redis://localhost:6380/0 go test -timeout 300s ./... -run Integration -p 1 -count=1 -v 2>&1 | grep -c -- '--- PASS: TestIntegration' ; COMPOSE_PROJECT_NAME=<slug> docker compose down
grep -n 'WithoutCancel\|Hijack\|pg_try_advisory_lock\|MigrationLockWait\|func (m \*PgMigrator) db()' internal/store/postgres.go
grep -n 'versionOf\|malformed migration name' internal/store/migrations.go
grep -n 'migrate_lock_test\|migrate_fs_test\|MigrationLockWait' ../harness/CODEMAP.md
git diff --stat origin/main..HEAD   # only postgres.go, migrations.go, migrate_lock_test.go, migrate_fs_test.go, harness/CODEMAP.md
git push -u origin harness/2026-09-27-medium-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs
```
(`english`/`english`/`english` are `backend/docker-compose.yml`'s defaults; the ports are the overrides set above.)
