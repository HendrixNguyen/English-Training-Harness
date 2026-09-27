---
plan: harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/boot-store-migrate-still-has-no-deadline-now-that-the-adviso.md, harness/ideas/_inbox/migration-lock-wait-log-and-error-always-quote-30s-even-when.md]
---
# Review — Migrate's advisory lock leaks into the pool when unlock runs on a cancelled context

**Plan:** `harness/plans/2026-09-27-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md`
**Branch/worktree:** `harness/2026-09-27-medium-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs` / `.worktrees/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs`
**Diff:** `git diff main...harness/2026-09-27-medium-migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs --stat`

## Plan vs idea
Delivered:
- **Primary idea:** a cancelled unlock no longer leaks the lock. The unlock runs on `WithoutCancel` with a 5 s bound, and hijacks and closes the connection when the unlock is unconfirmed.
- **Folded "hang forever":** the lock wait is bounded and logged when the context has no deadline.
- **Folded "only tested against the embedded migration":** `Migrate` is covered over `fstest.MapFS`.
- **New:** `pool_max_conns=1` no longer deadlocks, because statements run on the pinned connection.

The `cmd/api` boot deadline was deferred by design, and I filed it as the plan asked.

## Code vs plan
Reviewed at origin head `be70b39` in a detached reviewer worktree. The diff is 5 files, +482/−16, exactly the plan's conflict fence.
- It merges cleanly with `origin/main`.
- Against the RLS and pet-shields branches, only `harness/CODEMAP.md` conflicts. The new tests derive the migration count from `fs.Glob`, so they survive the extra `0004_*` files.

Task by task:
- Tasks 1–3 landed in one commit (justified: the same method).
- The optional pure `db()` unit test was dropped, as the plan allows. The integration test covers the pinned path.
- Task 4 was followed.
- One deviation from Task 2: the log and error quote `MigrationLockWait`, not the effective bound (low, filed).

```
go build ./... && gofmt -l . && go vet ./...         -> build-ok, gofmt silent
env -u … go test -timeout 120s ./... -count=1 -race  -> all packages ok
go test ./internal/store -run 'TestMigrate|TestMigration0001Down' -v -> 16 PASS (7 new pure MapFS tests + existing)
# rv-auth stack
go test ./internal/store -run Integration -p 1 -count=1 -v
--- PASS: TestIntegrationUnlockReleasesTheLockOnACancelledContext (0.02s)
--- PASS: TestIntegrationLockWaitIsBoundedAndLogged (1.02s)
--- PASS: TestIntegrationMigrateSucceedsWithPoolMaxConnsOne (0.10s)
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace (0.11s)  (7 "waiting" + 7 "acquired after ~55ms" lines)
go test ./... -run Integration -p 1 -count=1 -v | grep -c -- '--- PASS: TestIntegration'  -> 16
gh run list --branch <branch> --limit 1 -> completed success (run 36291533240)
```
Reproduction, re-run by the reviewer: I restored the base `postgres.go`, adding only a `MigrationLockWait` shim so the tests compile, and kept the new tests. The result:
- `pg_locks still holds the migration advisory lock after unlock() on a cancelled context (count=1)`;
- `a second Postgres could not acquire the migration lock within 3s … (the lock leaked into the pool)`;
- `Migrate with pool_max_conns=1: store: ensuring version table: context deadline exceeded`.

I then reverted, and `git status` is clean.

Runtime: `cmd/api` booted on an **empty** schema with `DATABASE_URL=…&pool_max_conns=1`. It logged `migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates]` and `/healthz` answered 200. After SIGTERM, `shutdown complete`, and `pg_locks` has 0 advisory rows for key 727100001.

## Quality
- Correctness: the timeout path's comment ("a cancelled pg_advisory_lock acquires nothing, so it is safe to return the connection") is not the whole story. The server can grant the lock just as the client cancels. In practice it is safe because pgx v5 closes a connection whose query was interrupted by its context, so the pool destroys it on `Release` and the session's lock dies with it. This is my inference from pgx's documented behaviour, not something I tested, so I filed nothing.
- The `PgMigrator` statefulness (the `conn` field) is documented as one `Migrate` per value, and every caller uses `Postgres.Migrator()` fresh. That is acceptable.
- Tests are honest: they were shown to fail without the fix, and the MapFS tests assert order, partial results and no-apply-on-error.
- Logs: the contention line always says "up to 30s" (low, filed).
- CODEMAP: the `store` paragraph is accurate. The `cmd/api` "deadline-free until the leak is fixed" clause becomes stale on merge, and is covered by the follow-up bug.

## Bugs filed
- `harness/ideas/_inbox/boot-store-migrate-still-has-no-deadline-now-that-the-adviso.md` (low): the plan's deferred follow-up for the boot deadline and the CODEMAP clause.
- `harness/ideas/_inbox/migration-lock-wait-log-and-error-always-quote-30s-even-when.md` (low): the effective bound is not reported, and a cancel is worded as contention.

## Verdict
**pass-with-bugs.** Delivered, the reproduction was verified red→green, CI is green, and there are no blockers.
