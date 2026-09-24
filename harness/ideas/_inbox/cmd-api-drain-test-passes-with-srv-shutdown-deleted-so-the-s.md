---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md
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
