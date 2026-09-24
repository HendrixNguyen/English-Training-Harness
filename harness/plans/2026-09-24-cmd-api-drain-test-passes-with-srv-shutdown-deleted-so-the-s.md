---
idea: harness/ideas/_inbox/cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md
status: done
priority: high
merged: true
amends: harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md
branch: harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so
worktree: .worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/17"
---
# cmd/api amend: make the drain test observe the drain — it must fail when `srv.Shutdown` is gone — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use the executing-plans (or subagent-driven-development) skill to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md` (BLOCKER, high)
**Amends:** `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (`done`, awaiting merge). This plan lands on that plan's branch
`harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` in its existing worktree
`.worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so` (HEAD `96d190d`). **No new branch, no new worktree.** Per the execute skill's amending rule, set `status=executing branch=<inherited> worktree=<inherited>` on *this* plan before starting.
**Review that found it:** `harness/reviews/2026-09-24-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (verdict `pass-with-bugs`, merge blocked on this one bug).

**Goal:** Rewrite `TestServeStopsOnContextCancelAndDrainsInFlightRequests` in `backend/cmd/api/server_test.go` so that it fails when `serve`'s shutdown block is deleted or replaced with a bare `ln.Close()` (or `srv.Close()`), by asserting the property the branch exists for — `serve` does not return while a request is in flight, refuses new connections meanwhile, and returns nil once the request completes. **Test-only.** `backend/cmd/api/server.go` is correct (the review's live proof shows a 1.6 s drain and exit 0) and is not touched.

**Not in scope** (same review, medium/low, still in the inbox for ordinary ranking): `log.Fatalf` on grace overrun / workers not joined; a second signal during the drain; the `GIN_MODE` init panic; the documented `set -a` leaking `TEST_*`. Do not fold any of them in.

**Run every command from `backend/` inside the worktree** (`cd .worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so/backend`) unless the step says otherwise. `rg` and `timeout` are not installed: use `grep -n`, bound tests with `go test -timeout`. Do not rebase or merge anything into the branch. Commit per task; end the commit message, after a blank line, with the `Co-Authored-By` trailer your session's attribution instructions give you.

---

## Root cause (from the idea's `## Evaluation`; reproduced by the evaluator)

The current test asserts two facts — `serve` returned nil, and the client got 200 — and each is independently true of a `serve` that just calls `ln.Close()`: the already-accepted connection keeps running, so the 300 ms handler finishes and writes 200 on its own, while `serve` returns nil immediately. The property that matters is an **ordering** — the handler finishes *before* `serve` returns — and the test never observes it: it waits on `done` first (with a 9 s allowance, so an instant return looks like a drained one) and on `code` second. The handler's `time.Sleep` also means nothing can be asserted *during* the drain window. The closing `DialTimeout` check proves only that the listener is closed, which `ln.Close()` also satisfies. Evaluator's reproduction: mutant (lines 51–59 of `serve` → `_ = ln.Close(); return nil`), `go test -count=1 -race -run Serve -v .` → 3/3 PASS.

**Fix shape:** the handler blocks on a `release` channel the test controls. After `cancel()`, assert that `serve` has **not** returned within 200 ms and that a fresh dial to the listener is refused; then close `release` and require `serve` → nil and the client → 200. A correct `serve` cannot return while the handler holds its connection active (`Shutdown` waits for idle), so the negative wait cannot fail correct code; a `serve` that skips `Shutdown` returns at once and hits the first assertion. Prototyped by the evaluator: mutant A (`ln.Close()`) and mutant B (`srv.Close()` in place of `srv.Shutdown`) both fail with `serve returned (<nil>) while a request was still in flight`; the real code passes `-count=5 -race`.

---

## Tasks

### Task 1: rewrite the drain test so it observes the drain window

**Files:**
- Modify: `backend/cmd/api/server_test.go` (only `TestServeStopsOnContextCancelAndDrainsInFlightRequests` and the import block; `listen`, `TestServeReturnsAListenerError` and `TestNewServerBoundsHeaderReadsButNotWrites` stay as they are)

- [ ] **Step 1: Reproduce the dishonest pass on a scratch mutant (the "failing test" for a test-only fix)**

Build a throwaway module outside the repo from the *current* branch files and mutate `serve` exactly as the reviewer did:

```bash
S=$(mktemp -d)/drain; mkdir -p "$S"
cp cmd/api/server.go cmd/api/server_test.go "$S"/
printf 'module scratch\n\ngo 1.24\n' > "$S/go.mod"
python3 - "$S/server.go" <<'PY'
import sys
p = sys.argv[1]; s = open(p).read()
start = s.index('\tlog.Printf("shutting down')
end = s.rindex('return nil\n}') + len('return nil\n}')
s = s[:start] + '\t_ = ln.Close()\n\treturn nil\n}' + s[end:]
s = s.replace('\t"fmt"\n', '').replace('\t"log"\n', '')
open(p, 'w').write(s)
PY
(cd "$S" && go test -count=1 -race -run Serve -v . 2>&1 | tail -8)
echo "SCRATCH=$S"
```

Expected (this is the bug): `--- PASS: TestServeStopsOnContextCancelAndDrainsInFlightRequests`, `PASS`, `ok`. Keep `$S` — Step 4 reuses it.

- [ ] **Step 2: Replace the test function**

In `backend/cmd/api/server_test.go`, add `"sync"` to the import block (between `"net/http"` and `"testing"`, gofmt order), and replace everything from the comment `// The signal arrives while a request is in flight` through the closing `}` of `TestServeStopsOnContextCancelAndDrainsInFlightRequests` with:

```go
// The signal arrives while a request is in flight: serve must stop accepting,
// keep running until that request finishes, then return nil — the shape a
// Railway SIGTERM produces. The ordering is the point: serve returning lets
// main run pg.Close()/rdb.Close() and exit, so it must not return while a
// handler is still working. The handler blocks on release so the test can
// observe the drain window instead of racing a sleep.
func TestServeStopsOnContextCancelAndDrainsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock) // a failing assertion must not leave the handler wedged
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newServer(h), ln) }()

	code := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			code <- -1
			return
		}
		_ = resp.Body.Close()
		code <- resp.StatusCode
	}()

	<-started
	cancel()

	// Drain window: the handler is still blocked, so serve must still be running…
	select {
	case err := <-done:
		t.Fatalf("serve returned (%v) while a request was still in flight", err)
	case <-time.After(200 * time.Millisecond):
	}
	// …but new connections must already be refused (Shutdown closes listeners first).
	if c, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		_ = c.Close()
		t.Fatal("listener still accepting new connections during the drain")
	}

	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want nil on a clean shutdown", err)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve did not return after the in-flight request finished")
	}
	if got := <-code; got != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200 (drained, not cut off)", got)
	}
}
```

Nothing else in the file changes. The old trailing `DialTimeout` check is gone because it now lives inside the drain window, where it is meaningful.

- [ ] **Step 3: Run it against the real `server.go`**

```bash
test -z "$(gofmt -l ./cmd)" && echo fmt-ok
go vet ./cmd/api/...
go test ./cmd/api/... -count=1 -race -timeout 120s -v 2>&1 | grep -n '^--- \|^ok\|^FAIL\|DATA RACE'
go test ./cmd/api/... -count=5 -race -timeout 120s
```

Expected: `fmt-ok`; vet silent; `--- PASS` for all three tests, `ok`, no `DATA RACE`; the `-count=5` run `ok`. The drain test should take ~0.2–0.7 s (the 200 ms window plus `Shutdown`'s idle poll), not 9 s.

- [ ] **Step 4: Prove the new test catches the mutant**

Copy the rewritten test over the scratch module from Step 1 and run it against the mutated `serve`:

```bash
cp cmd/api/server_test.go "$SCRATCH"/
(cd "$SCRATCH" && go test -count=1 -race -run Drains -v . 2>&1 | grep -n 'server_test.go\|^--- \|^ok\|^FAIL')
```

Expected: `--- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests` with `server_test.go:NN: serve returned (<nil>) while a request was still in flight`, `FAIL`. Then a second mutant — `srv.Close()` instead of `srv.Shutdown` — must fail the same way:

```bash
cp cmd/api/server.go "$SCRATCH"/server.go
sed -i '' 's/if err := srv.Shutdown(shutdownCtx); err != nil {/_ = shutdownCtx\
	if err := srv.Close(); err != nil {/' "$SCRATCH"/server.go
(cd "$SCRATCH" && go test -count=1 -race -run Drains . 2>&1 | grep -n 'server_test.go\|^ok\|^FAIL')
rm -rf "$(dirname "$SCRATCH")"
```

Expected: the same `while a request was still in flight` line and `FAIL`. Paste both mutant outputs into the execution summary — they are the evidence the reviewer asked for.

- [ ] **Step 5: Commit**

```bash
git add cmd/api/server_test.go
git diff --cached --stat   # exactly one file
git commit -m "cmd/api: drain test must observe the drain window

TestServeStopsOnContextCancelAndDrainsInFlightRequests passed with serve's
shutdown block replaced by ln.Close(): it awaited serve and the client
separately and never asserted that the handler finished before serve
returned. The handler now blocks on a channel the test releases; while it is
blocked, serve must still be running and the listener must already refuse new
connections. Fails against both an ln.Close() and an srv.Close() mutant."
```

Append the `Co-Authored-By` trailer after a blank line. Then `git push origin harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` (the branch already exists on the remote; no PR — the owner takes one per day).

---

## Verification

From `backend/` in the amended worktree, after Task 1 is committed. Items 1–3 re-run the reviewer's evidence from the review's "Test honesty (blocker)" finding, so the fix is proven rather than asserted.

1. **Reviewer's mutation, re-run against the new test (must now FAIL):** repeat Step 1's scratch build with the *committed* `server_test.go` (fresh `mktemp -d`, copy both files, apply the same Python mutation to `server.go`), then `go test -count=1 -race -run Serve -v .` → `--- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests` at the `while a request was still in flight` assertion; `TestServeReturnsAListenerError` still `PASS`; overall `FAIL`. Delete the scratch dir.
2. **Second mutant (`srv.Close()` for `srv.Shutdown`)** as in Step 4 → same failing line.
3. **Real code under `-race`:** `go test ./cmd/api/... -count=5 -race -timeout 120s` → `ok`, no `DATA RACE`; and `go test ./cmd/api/... -count=1 -race -timeout 120s -v` shows the drain test passing in well under 1 s.
4. `go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)"` → exit 0.
5. `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s` → every package `ok` (nothing outside `cmd/api` changed; this guards the branch as a whole).
6. `git diff 96d190d --stat` → exactly `backend/cmd/api/server_test.go`. `server.go`, `main.go` and everything else on the branch are byte-identical to the reviewed HEAD.
7. After the push: `gh run list --branch harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so --limit 1` → all four jobs (`frontend`, `harness-tooling`, `backend-unit`, `backend-integration`) green on the new HEAD.
8. In the main checkout, once this plan is `done`: `python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` → exit 0 (no unresolved blocker), so the owner can `/harness merge` it.

## Notes

- **Why a 200 ms negative wait is safe:** the assertion is "serve has *not* returned yet". A correct `serve` is inside `srv.Shutdown`, which only returns once every connection is idle, and the handler is holding its connection active until `release` is closed — so the wait cannot flake correct code, however slow the machine. A broken `serve` returns in microseconds and is caught. Shorter than 200 ms would still work; it is 200 ms so that a slow `-race` scheduler has room to deliver the early return before we look.
- **Why the dial check moved inside the window:** `Shutdown` closes the listeners first and *then* waits for idle connections, so "new connections refused while an old one is still running" is the observable half of graceful shutdown. Outside the window it only proved the listener was closed eventually, which `ln.Close()` also does.
- **`sync.Once` + `t.Cleanup`:** if an assertion fails before `unblock()`, `t.Fatalf` ends the test goroutine while the handler is still parked on `release`; the cleanup releases it so the test binary does not carry a wedged goroutine into the next test.
- **Not a timestamp comparison:** the idea also offered "record handler-finished and serve-returned times and assert ≤". That works but cannot distinguish "returned before the handler" from "returned during the same tick", and it still leaves the drain window unobservable; the blocking handler gives a strict before/after with no clock.

## Execution summary

**Status: done.** Worked in the inherited worktree `.worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so` on the inherited branch `harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` (no new branch/worktree created, per the amending-plan rule). Local branch was already at `origin`'s HEAD (`96d190d`) when work started — no reviewer pushes to reconcile.

One commit, exactly Task 1 as planned: `bb00d46` "cmd/api: drain test must observe the drain window" (`backend/cmd/api/server_test.go` only — `sync` import added, `TestServeStopsOnContextCancelAndDrainsInFlightRequests` rewritten to block the handler on a `release` channel and assert the drain-window ordering). No deviations from the plan.

### Verification (from `backend/`)

1. **Step 1 scratch reproduction (dishonest pass, pre-fix):** copied `server.go`/`server_test.go` (pre-fix) into a scratch module, mutated `serve` to `_ = ln.Close(); return nil`, ran `go test -count=1 -race -run Serve -v .` → all 3 tests `PASS`, `ok` — reproduced the reviewer's finding exactly.
2. **Plan Verification item 1 (reviewer's mutation vs. the new, committed test — must FAIL):** fresh scratch build, same mutation, `go test -count=1 -race -run Serve -v .`:
   ```
   === RUN   TestServeStopsOnContextCancelAndDrainsInFlightRequests
       server_test.go:60: serve returned (<nil>) while a request was still in flight
   --- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.00s)
   === RUN   TestServeReturnsAListenerError
   --- PASS: TestServeReturnsAListenerError (0.00s)
   === RUN   TestNewServerBoundsHeaderReadsButNotWrites
   --- PASS: TestNewServerBoundsHeaderReadsButNotWrites (0.00s)
   FAIL
   ```
3. **Plan Verification item 2 (`srv.Close()` mutant — must FAIL):**
   ```
   2026/09/24 22:47:17 shutting down: draining in-flight requests for up to 8s
   --- FAIL: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.00s)
       server_test.go:60: serve returned (<nil>) while a request was still in flight
   FAIL
   ```
   Scratch dirs deleted after each mutant run.
4. **Plan Verification item 3 (real code under `-race`):** `go test ./cmd/api/... -count=5 -race -timeout 120s` → 15/15 `PASS` (drain test ~0.27-0.28s each run), `ok`, no `DATA RACE`; `go test ./cmd/api/... -count=1 -race -timeout 120s -v` → 3/3 `PASS`, drain test 0.28s, `ok` in 1.773s.
5. **Plan Verification item 4:** `go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)"` → exit 0, `fmt-ok`.
6. **Plan Verification item 5 (full backend suite, clean env):** `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s` → all 11 packages `ok` (`cmd/api`, `airouter`, `auth`, `config`, `google`, `health`, `notify`, `onboarding`, `pet`, `quests`, `store`).
7. **Plan Verification item 6 (diff scope):** `git diff 96d190d --stat` → exactly `backend/cmd/api/server_test.go | 33 ++++++++++++++++++++++++++-------` — `server.go`, `main.go`, everything else byte-identical to the reviewed HEAD.
8. **Plan Verification item 7 (CI):** pushed `bb00d46`; run [36022865033](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36022865033) — `backend-unit`, `frontend`, `harness-tooling`, `backend-integration` all `success`.
9. **Plan Verification item 8 (blockers):** after this plan was set `status=done`, `python3 tools/harness/cli.py blockers --plan harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` → exit 0, no output (the idea's own `status: planned` was already correct; `scan.blockers_for` only needed the fix plan's `status=done`, which this record sets).

### Runtime proof

This is a test-only change to `backend/cmd/api/server_test.go`; `server.go`/`main.go` are untouched and already had a full boot/signal proof recorded in the amended plan's (`2026-09-23-...`) own Execution summary (SIGTERM/SIGINT drain, `/healthz`, in-flight-request drain, `GIN_MODE` switch, cleanup). That proof is not re-run here — the orchestrator's task scope for this amending plan is the plan's own Verification (mutation tests + `-race` + full suite) plus CI, which are all captured above. No process or container was started by this plan's work, so there is nothing to clean up beyond the two scratch mutation directories, both of which were `rm -rf`'d immediately after their runs (confirmed by the subsequent `ls` returning "No such file or directory").

No deviations, no scope creep: only Task 1 as written, `server.go` untouched.
