---
plan: harness/plans/2026-09-24-cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md
verdict: pass
bugs: []
---
# Review — cmd/api amend: make the drain test observe the drain — it must fail when `srv.Shutdown` is gone

**Plan:** `harness/plans/2026-09-24-cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md`
**Branch/worktree:** `harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` / `.worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so`
**Diff:** `git diff main...harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so --stat`

**Scope:** the amend commit only — `git diff 96d190d..bb00d46` → `backend/cmd/api/server_test.go | 33 ++++++++++++++++++++++++++-------` (26+/7-). `server.go`, `main.go` and every other file on the branch are byte-identical to the previously reviewed HEAD `96d190d`.

## Plan vs idea
Delivered. The idea's Expected output asked for (1) a drain test that fails against the `ln.Close()` mutant, using a `release`-blocked handler and a "serve has not returned" assertion, then `serve` → nil and client → 200; (2) optionally, a new connection refused during the drain window; (3) a test-only fix. All three are present: the handler blocks on `release`, the test asserts no return within 200 ms after `cancel()`, asserts `DialTimeout` is refused inside the window, then releases and requires nil + 200. `server.go` is untouched.

## Code vs plan
Task 1 **followed** exactly: `"sync"` added in gofmt order; the test body matches the plan's Step 2 block verbatim; the old trailing `DialTimeout` moved inside the drain window; the other two tests and `listen` unchanged. No deviations.

Verification re-run by the reviewer (all from `backend/` in the worktree unless noted; mutants on a scratch module under the session scratchpad, never the worktree, deleted after):

- **Mutant A** (`log.Printf("shutting down…` through final `return nil` → `_ = ln.Close(); return nil`), `go test -count=1 -race -run Serve -v .`:
  ```
  server_test.go:60: serve returned (<nil>) while a request was still in flight
  --- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.00s)
  --- PASS: TestServeReturnsAListenerError (0.00s)
  --- PASS: TestNewServerBoundsHeaderReadsButNotWrites (0.00s)
  FAIL	scratch	1.297s
  ```
- **Mutant B** (`srv.Shutdown(shutdownCtx)` → `srv.Close()`), same command:
  ```
  server_test.go:60: serve returned (<nil>) while a request was still in flight
  --- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.00s)
  FAIL	scratch	1.184s
  ```
- **Flakiness:** `go test ./cmd/api/... -count=20 -race -timeout 300s` run twice (once `-v`) → 20/20 drain-test `--- PASS` per run, `ok ... 6.896s`, no `DATA RACE`. 40/40 clean.
- `go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)"` → `fmt-ok`.
- Full suite, `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s` → all 11 packages `ok`.
- **CI:** `gh run list --branch harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` → run 36022865033, `headSha bb00d46b…`, `success`; jobs `backend-unit`, `backend-integration`, `harness-tooling`, `frontend` all `success`.
- **Blocker:** `python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` → exit 0, no output.
- Worktree `git status --short` clean after all runs.

The executor's claims all reproduce; no gate failure.

## Quality
- **Test honesty:** now honest for the regression it guards. The ordering (handler finishes before `serve` returns) is observed directly rather than inferred, and the dial check sits where it actually distinguishes `Shutdown` from "closed eventually". The 200 ms negative wait cannot flake correct code — `Shutdown` cannot return while the handler holds its connection active — which the 40-run `-race` sweep bears out. The test takes ~0.3 s, down from a 9 s worst-case allowance.
- **Cleanup:** `sync.Once` + `t.Cleanup(unblock)` correctly prevents a wedged handler goroutine when an assertion fails before `unblock()`.
- **Residual, by design (not filed):** a contrived mutant that closes the listener, sleeps 300 ms and returns nil passes (`--- PASS (0.30s)` ×3), because any bounded "has not returned" window can be outlived by a fixed delay. That is inherent to a black-box timing test, the plan's Notes accept it, and no realistic refactor produces that shape. Not a defect.
- **Minor hygiene (not filed):** `<-started` (pre-existing) and the final `<-code` are unbounded receives. They can only block on already-broken code, and `go test -timeout` turns that into a stack-traced failure, so nothing is lost but fail-fast latency.
- Comments are accurate (`Shutdown` does close listeners before waiting for idle). Boundaries and CODEMAP are unaffected by a test-only change.

## Bugs filed
None.

## Verdict
**pass.** The amend delivers the blocker idea exactly: the drain test now fails against both the `ln.Close()` and the `srv.Close()` mutants, passes 40/40 under `-race`, the full suite and CI are green on `bb00d46`, and `blockers --plan harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` exits 0, so that plan is clear for `/harness merge`.
