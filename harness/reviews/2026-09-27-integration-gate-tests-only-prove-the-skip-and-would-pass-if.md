---
plan: harness/plans/2026-09-27-integration-gate-tests-only-prove-the-skip-and-would-pass-if.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/store-reset-refusal-still-has-no-direct-test-deferred-by-the.md]
---
# Review — Integration gate tests only prove the skip and would pass if the gate always skipped

**Plan:** `harness/plans/2026-09-27-integration-gate-tests-only-prove-the-skip-and-would-pass-if.md`
**Branch/worktree:** `harness/2026-09-27-low-integration-gate-tests-only-prove-the-skip-and-would-pass-if` / `.worktrees/integration-gate-tests-only-prove-the-skip-and-would-pass-if`
**Diff:** `git diff main...harness/2026-09-27-low-integration-gate-tests-only-prove-the-skip-and-would-pass-if --stat`

## Plan vs idea
Mostly delivered. Of the idea's four Expected-output bullets, three exist: skip cases unchanged, two "proceeds" cases on an unreachable `127.0.0.1:59999` that never dial, and a recorded mutation check. The fourth — a direct test of `reset`'s refusal (`integration_test.go:46-48`) — is explicitly deferred by the plan, because it needs a `testing.TB` seam in `integration_test.go`, which unmerged branches hold. The deferral is reasonable. It was recorded only in the plan, so I filed it as a low follow-up so it isn't lost.

## Code vs plan
Reviewed at head `4ccdf4d` (detached scratch worktree; 23 behind main). CI run 36296624644 `success`.
- Task 1: followed. Two tests appended to `integration_gate_test.go`, same `t.Run` + deferred `t.Skipped()` pattern as the existing ones. The Postgres case asserts `pg != nil && pg.Pool != nil` and never pings. The pool is closed by the helper's own cleanup on the subtest.
- Task 2 (mutation, not committed): reproduced by the reviewer.
  - `t.Skip("mutant")` at the top of `requirePostgres` → `--- FAIL: TestRequirePostgresProceedsWithTestDatabaseURL … integration_gate_test.go:55: requirePostgres must not skip when TEST_DATABASE_URL is set`.
  - Same in `requireRedisURL` → `--- FAIL: TestRequireRedisURLProceedsWithTestRedisURL … integration_gate_test.go:74`.
  - `integration_test.go` restored; `git status` clean.
- Scope: only `integration_gate_test.go` + one CODEMAP clause. `integration_test.go` untouched.

Verification (reviewer, `backend/`):
```
go build ./... ; gofmt -l . ; go vet ./...        -> clean
env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1   -> all packages ok
go test ./internal/store/ -count=1 -v -run 'RequirePostgres|RequireRedisURL'
  --- PASS: TestRequirePostgresSkipsWithoutTestDatabaseURL
  --- PASS: TestRequireRedisURLSkipsWithoutTestRedisURL
  --- PASS: TestRequirePostgresProceedsWithTestDatabaseURL
  --- PASS: TestRequireRedisURLProceedsWithTestRedisURL
  ok  .../internal/store 1.183s
```

## Quality
- Test honesty: both new tests are killed by the obvious mutant. Neither name starts with `TestIntegration*`, so the CI PASS-count gate is unchanged. `t.Setenv` overrides a CI-exported `TEST_*`, so the tests are deterministic in both jobs.
- Merges cleanly with `origin/main` + the google-403 branch; `go test ./internal/store` ok on that merge.
- CODEMAP: the added clause in the `store` bullet is accurate.

## Bugs filed
- `harness/ideas/_inbox/store-reset-refusal-still-has-no-direct-test-deferred-by-the.md` — low: the deferred direct test of `reset`'s refusal, to do after the branches holding `integration_test.go` merge.

## Verdict
pass-with-bugs — what the plan scoped is delivered and proven by mutation. One idea bullet is deferred (filed as a low follow-up). Mergeable.
