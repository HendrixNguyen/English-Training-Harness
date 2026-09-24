---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md
plan: harness/plans/2026-09-24-cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md
---
# cmd/api drain test passes with srv.Shutdown deleted so the shutdown regression is unguarded

## Why
`TestServeStopsOnContextCancelAndDrainsInFlightRequests` is the only automated guard for the
regression this branch fixes: that the API stops cleanly on SIGINT/SIGTERM and lets in-flight
requests finish first. It does not actually guard it. If the whole graceful-shutdown block in
`serve` is replaced with a bare `ln.Close()`, meaning no `srv.Shutdown`, no `ShutdownGrace` and no
waiting, **all three `cmd/api` tests still pass under `-race`**.

The test passes by coincidence. `ln.Close()` stops new `Accept`s but does not touch
connections already accepted, so the one 300 ms request finishes by itself. The test waits on
`done` and `code` separately and never checks their order. So nothing asserts that `serve`
returned *after* the handler finished. That ordering is the whole point: in the real binary
`serve` returning lets `main` return, which runs `pg.Close()`/`rdb.Close()` under a live
handler and then exits the process, killing it mid-request. A future refactor could delete
`Shutdown` and CI would stay green. This is a dishonest test on an unmerged branch, which is why
it blocks the merge.

## Expected output
- The drain test fails against the `ln.Close()` mutant. For example, the handler blocks on a
  `release` channel. After `cancel()`, the test asserts that `serve` has **not** returned within
  some interval while the handler is still blocked. Then it closes `release` and asserts that
  `serve` returns nil and the client got 200. Alternatively, record the handler-finished time
  and the `serve`-returned time and assert handler-finished ≤ serve-returned.
- Optional, same file: a new connection attempted during the drain window is refused. This
  proves `Shutdown` stopped accepting while the old request was still running.
- The fix is test-only. `server.go` behaves correctly: the live proof below shows it draining.

## Evidence
- Plan: `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (Task 2 test; branch `harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so`).
- `backend/cmd/api/server_test.go:22-62`: `done` and `code` are awaited independently, and no
  ordering between them is asserted.
- Mutation, run by the reviewer on 2026-09-24: `server.go` and `server_test.go` were copied to a
  scratch module, and everything from `log.Printf("shutting down…` down to the final
  `return nil` was replaced with `_ = ln.Close()`. Result of
  `go test -count=1 -race -run Serve -v .`:
  `--- PASS: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.30s)`,
  `--- PASS: TestServeReturnsAListenerError`, `ok`. The-validator reproduced the same result
  independently.
- For contrast, the real implementation does drain. In the live binary, a request whose body was
  still uploading at `kill -TERM` held shutdown open for 1.6 s, got its `400 invalid_request`,
  and the process exited 0.

## Evaluation

**Verdict: select, `priority: high` (blocker — it holds `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` off `main`).**

**Is the Why real?** Yes, and reproduced here rather than taken on trust. `server.go` and
`server_test.go` were copied off the blocked branch (HEAD `96d190d`) into a scratch module and
lines 51–59 of `serve` (the `log.Printf` … `<-errc` block) replaced with `_ = ln.Close()`;
`go test -count=1 -race -run Serve -v .` → `--- PASS: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.30s)`,
3/3 PASS, `ok`. The reviewer's evidence is exact.

**Root cause** (systematic-debugging, read-only): the test asserts two facts —
`serve` returned nil, and the client got 200 — but each is true of the mutant on its own. After
`ln.Close()` the already-accepted connection keeps running, so the 300 ms handler finishes and
writes 200 regardless; and `serve` returns nil at once. The property that matters — *`serve` must
not return while a handler is running* — is an ordering between `done` and the handler's
completion, and the test never observes it because it waits on `done` first (with a 9 s
allowance, so an immediate return is indistinguishable from a drained one) and on `code` second.
The handler's `time.Sleep` also makes the drain window unobservable from the test: nothing can be
asserted *during* it. The dial check at the end proves only that the listener is closed, which
`ln.Close()` satisfies too. `server.go` itself is correct — the review's live proof shows a
1.6 s drain and exit 0 — so this is a test-only fix.

**Fix, prototyped in the same scratch module and proven against two mutants:** the handler
blocks on a `release` channel instead of sleeping. After `cancel()`, the test asserts that
`serve` has *not* returned within 200 ms (the handler is still blocked, so a correct `serve` is
still inside `srv.Shutdown`), and that a new `net.DialTimeout` to the listener is refused
(`Shutdown` closes listeners first). Only then does it close `release` and require `serve` →
nil and the client → 200. Results: mutant A (`ln.Close()` for the whole block) →
`server_test.go: serve returned (<nil>) while a request was still in flight`, FAIL in 0.00 s;
mutant B (`srv.Close()` in place of `srv.Shutdown(shutdownCtx)`) → same line, FAIL. Real
`server.go` → `go vet` clean, `go test -count=5 -race -run . .` → `ok`. The 200 ms negative
wait can only produce a false *pass* if `serve` is slow to return wrongly — never a false
failure of correct code, because a correct `serve` cannot return while the handler holds the
connection active.

**Achievable in one plan?** Trivially: one test function rewritten in one existing file on the
existing branch, no `server.go` change, no new files. **Dependencies:** none beyond the branch
itself; it must land as an amend (`amends:` the blocked plan) in worktree
`.worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so`.

**Scope held to the blocker.** The sibling bugs from the same review (`log.Fatalf` on grace
overrun / workers not joined; second signal swallowed; `GIN_MODE` init panic; `set -a` leaking
`TEST_*`) are medium/low, do not block the merge, and stay in the inbox for ordinary ranking.
