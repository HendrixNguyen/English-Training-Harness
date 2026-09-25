---
plan: harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md
verdict: pass
bugs: []
---
# Review — go test drops every table in whatever DATABASE_URL points at

**Plan:** `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md`
**Branch/worktree:** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` / `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001` — both gone; the plan is already merged.
**Reviewed on:** a detached worktree of `origin/main` at `3f4242d` (post-hoc review, 2026-09-25 daily run).

## Plan vs idea
The idea asked that destructive tests never run against a database the developer did not nominate, and that `make test` stay non-destructive whatever is exported. **Delivered.** The plan took the cheapest acceptable option: gate on `TEST_DATABASE_URL` / `TEST_REDIS_URL`, plus a belt-and-braces refusal inside `reset()`. It skipped the optional "refuse if `users` has rows / host not localhost" guard and gave reasons (Notes). That is acceptable because the idea lists the options as alternatives. The fix still holds on current main, and **every package added since** follows the same rule (`auth`, `google`, `pet`, `onboarding`, `notify`, `quests`, `airouter`): no `_test.go` in the backend reads `DATABASE_URL` or `REDIS_URL`.

## Code vs plan
- **Task 1 (gate tests first):** `internal/store/integration_gate_test.go` exists, and both tests pass.
- **Task 2 (gate and `reset()` refusal):** `requirePostgres` and `requireRedisURL` read only the `TEST_*` variables, and `reset()` calls `t.Fatal` without `TEST_DATABASE_URL` (`integration_test.go:47`). Followed.
- **Task 3 (Makefile and CODEMAP):** the Makefile comment above `test-integration` names the `TEST_*` variables, and the CODEMAP describes the gate and the CI guard. Followed.

Verification re-run:
```
env -u TEST_* -u REDIS_URL DATABASE_URL=postgres://…:59999/… go test ./internal/store/ -run Integration -v
  # 4 × --- SKIP (the 3 from the plan + TestIntegrationConcurrentMigrateDoesNotRace added later), PASS, no "down migration:" line
go test ./internal/store/ -run SkipsWithout -v   # 2 × --- PASS
env -u TEST_* DATABASE_URL=…:59999 REDIS_URL=…:59999 make test   # 13 packages ok
grep -n 'os.Getenv' internal/store/integration_test.go            # all four read TEST_* names
```
**Live destructive-gate proof** (beyond the plan): on a scratch Postgres (`rev0925olds`, port 5447) I created a `canary` table. I then ran `go test ./... -run Integration -p 1` with `DATABASE_URL` / `REDIS_URL` pointed at that real, reachable database and the `TEST_*` variables unset. All 12 integration tests printed `--- SKIP`, and `canary` still held its row. With `TEST_*` set to the same database, `make test-integration` gave 12 × `--- PASS`. `canary` survived that too, because `reset()` drops only the tables the migrations own.

## Quality
- **Residual hazard, already tracked:** `.env.example` sets `TEST_DATABASE_URL` equal to the dev `DATABASE_URL`, and the documented `set -a; . ./.env` puts it in the shell. After that, a plain `go test ./...` drops the dev database's tables. This is the planned medium bug `the-documented-set-a-env-export-also-exports-test-database-u.md`, so no duplicate.
- `reset()` runs only the 0002 and 0001 down migrations. A comment explains why 0003 is skipped: its columns go with 0001's `DROP TABLE`. A future migration that creates a new table would need adding here. That is worth remembering, but nothing is broken today.

## Bugs filed
None.

## Verdict
**pass.** Delivered, still true on current main, and adopted by every later integration test.
