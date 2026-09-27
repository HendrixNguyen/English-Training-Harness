---
idea: harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md
status: done
priority: low
merged: true
branch: harness/2026-09-27-low-integration-gate-tests-only-prove-the-skip-and-would-pass-if
worktree: .worktrees/integration-gate-tests-only-prove-the-skip-and-would-pass-if
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/53"
---
# Integration gate tests only prove the skip and would pass if the gate always skipped — Plan

**Idea:** `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md`
**Goal:** `integration_gate_test.go` proves both directions of `requirePostgres` / `requireRedisURL`: skip without the `TEST_*` variable, proceed with it. No live service is needed. Replacing either helper body with an unconditional `t.Skip` must fail a test.

**Root cause:** `backend/internal/store/integration_gate_test.go` has one test per helper, and each asserts only that the helper skips when `TEST_*` is unset. A helper that always skipped would pass both tests. It would also silently turn every `TestIntegration*` in the package into a skip. The CI `backend-integration` job counts `--- PASS` lines per `func TestIntegration*`, which catches that in CI only, not in `go test` locally.

**Scope / files:** `backend/internal/store/integration_gate_test.go` only. Do **not** edit `integration_test.go`: two unmerged branches (`…every-public-table-is-readable…` RLS and `…pet-streak-shield…`) edit it, and today's `…migrate-s-advisory-lock…` branch edits `postgres.go`/`migrations.go`. None of them edits the gate test file. `NewPostgres` stays a lazy, non-dialling pool on both `main` and the migrate-lock branch (`postgres.go:22-34`), which is what makes the Postgres "proceeds" case deterministic.

**Deferred (recorded, not done):** the idea's direct test of `reset`'s refusal. `reset(t *testing.T, …)` calls `t.Fatal`, and a failing subtest always fails its parent, so a direct test needs `reset` to take a `testing.TB`-shaped seam. That is a signature change in `integration_test.go`, which the branches above hold. Leave it for when they have merged.

## Tasks

### Task 1: The "proceeds" cases
Files: `backend/internal/store/integration_gate_test.go`.
1. `TestRequirePostgresProceedsWithTestDatabaseURL`:
   - `t.Setenv("TEST_DATABASE_URL", "postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2")`.
   - Run `requirePostgres(t)` inside `t.Run("gated", …)`, capturing `t.Skipped()` in a defer as the existing tests do, and capture the returned `*Postgres`.
   - Assert the subtest did **not** skip and `pg != nil && pg.Pool != nil`.
   - Never call `Ping`, `Migrate` or `reset`: the address is unreachable by design, and the pool is closed by `requirePostgres`'s own `t.Cleanup`.
   - Doc comment: "the other half of the gate — it must let a nominated database through, or every TestIntegration* here silently skips".
2. `TestRequireRedisURLProceedsWithTestRedisURL`: `t.Setenv("TEST_REDIS_URL", "redis://127.0.0.1:59999/0")`. Assert it does not skip and that the returned URL equals the value set.
3. Run `cd backend && go test ./internal/store/ -run 'RequirePostgres|RequireRedisURL' -count=1 -v`. Expect 4 PASS lines. Then run it once more with `TEST_DATABASE_URL`/`TEST_REDIS_URL` exported to real values, as in CI, and expect the same 4 PASS lines, because `t.Setenv` overrides.
4. `gofmt -l internal/store` → empty.
5. Commit: `store: gate tests pin that the integration helpers proceed when nominated`.

### Task 2: Mutation check (not committed)
1. Temporarily replace `requirePostgres`'s body in `integration_test.go` with `t.Helper(); t.Skip("mutant"); return nil`. Run `go test ./internal/store/ -run RequirePostgres -count=1`, expect FAIL, then `git checkout -- internal/store/integration_test.go`.
2. Do the same for `requireRedisURL` (`t.Skip("mutant"); return ""`).
3. Record both FAIL outputs, one line each, in the plan's execution notes or the branch's final commit message. `git diff origin/main -- internal/store/integration_test.go` must be empty afterwards.

## Verification
```bash
cd backend
go test ./internal/store/ -count=1 -v -run 'RequirePostgres|RequireRedisURL'   # 4 PASS, 0 SKIP at top level
go test ./... -count=1
gofmt -l . && go vet ./...
git diff --name-only origin/main -- internal/store   # only integration_gate_test.go
```
- Both mutants from Task 2 make at least one gate test fail (evidence recorded).
- CI `backend-unit` and `backend-integration` are green on the pushed branch. The new tests are not named `TestIntegration*`, so the integration job's PASS-count gate is unchanged.

## Execution summary

**Built:** two tests appended to `backend/internal/store/integration_gate_test.go` — `TestRequirePostgresProceedsWithTestDatabaseURL` (asserts no skip and `pg != nil && pg.Pool != nil`; never pings) and `TestRequireRedisURLProceedsWithTestRedisURL` (asserts no skip and the returned URL equals the one set). One clause added to the `store` paragraph of `harness/CODEMAP.md`. Commit `4ccdf4d` on `harness/2026-09-27-low-integration-gate-tests-only-prove-the-skip-and-would-pass-if`.

**Deviations:** none from the tasks. Runtime proof: this change is test-only (no production code touched), so the "real path" is the `go test` invocation itself; the API was not booted and no compose stack was started (nothing to clean up — `docker ps`/`pgrep` untouched). The plan file is recorded in the worktree's uncommitted copy, per the orchestrator's instruction, not in ROOT.

**Reproduction (mutation check, Task 2, not committed):**
- `requirePostgres` body → `t.Helper(); t.Skip("mutant"); return nil`: `--- FAIL: TestRequirePostgresProceedsWithTestDatabaseURL` — `integration_gate_test.go:55: requirePostgres must not skip when TEST_DATABASE_URL is set`.
- `requireRedisURL` body → `t.Helper(); t.Skip("mutant"); return ""`: `--- FAIL: TestRequireRedisURLProceedsWithTestRedisURL` — `integration_gate_test.go:74: requireRedisURL must not skip when TEST_REDIS_URL is set`.
- After `git checkout`: `git diff origin/main -- internal/store/integration_test.go` → 0 lines.

**Verification output:**
```
go test ./internal/store/ -count=1 -v -run 'RequirePostgres|RequireRedisURL'
--- PASS: TestRequirePostgresSkipsWithoutTestDatabaseURL
--- PASS: TestRequireRedisURLSkipsWithoutTestRedisURL
--- PASS: TestRequirePostgresProceedsWithTestDatabaseURL
--- PASS: TestRequireRedisURLProceedsWithTestRedisURL
ok  .../backend/internal/store
# same 4 top-level PASS with TEST_DATABASE_URL/TEST_REDIS_URL exported (t.Setenv overrides)
go build ./...                      -> ok
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
                                    -> ok for all 13 packages
gofmt -l .                          -> empty
go vet ./...                        -> ok
git diff --name-only origin/main -- internal/store -> backend/internal/store/integration_gate_test.go
```

**CI:** https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36296624644 — success (backend-unit, backend-integration, frontend, docker-images, harness-tooling all green) on `4ccdf4d`.
