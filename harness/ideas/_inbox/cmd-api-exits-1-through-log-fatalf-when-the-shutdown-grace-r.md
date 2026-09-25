---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md
---
# cmd/api exits 1 through log.Fatalf when the shutdown grace runs out, skipping the deferred pg and rdb Close

## Why
The idea this branch delivers (`main-go-installs-a-signal-handler-with-no-server-shutdown-so.md`)
asks that on shutdown `main` "**returns** rather than `log.Fatalf`, so the existing
`defer pg.Close()` / `defer rdb.Close()` finally run and the pet cron's cancellation is joined
rather than raced". That only happens on the fast path.

When an in-flight request outlives `ShutdownGrace` (8 s), `serve` calls `srv.Close()` and returns
`shutdown: context deadline exceeded`. Then `main.go:184` does
`log.Fatalf("server: %v", err)`, which calls `os.Exit(1)`, so the deferred Close calls never run.
`serve`'s own doc comment says it exists "so main can return … instead of Fatalf-ing".

This path is routine, not exotic. The same file documents that `POST /integrations/google/sync`
runs up to `google.SyncTimeout` (60 s) and onboarding makes multi-provider AI calls. So any
Railway redeploy during one of those requests logs a fatal error and exits 1. That reads as a
crash in deploy logs and alerting, not a planned stop.

The idea's other clause, joining the background workers, is not implemented on either path.
`go pet.RunHourly(ctx, …)` and `go notify.RunWorker(ctx, …)` are never awaited before the
deferred Closes. Inferred, not run: both loops exit on `ctx.Done()`, and the notify `Tick`
reschedules before sending, so the practical cost today is log noise such as "client is closed",
not lost work. It is still an Expected-output item the plan did not carry.

## Expected output
- On grace overrun, `serve` or `main` logs the overrun (one line, not fatal), force-closes the
  remaining connections, and `main` **returns**, so `pg.Close()`/`rdb.Close()` run. Whether the
  exit status is 0 or a deliberate non-zero is a decision for the plan to state, but it must not
  be `log.Fatalf`.
- `main` waits for the pet cron and notify worker goroutines to return, for example with a
  `sync.WaitGroup` bounded by the remaining grace, before the deferred Closes run.
- A unit test covers the overrun branch. This needs the grace to be injectable: today
  `ShutdownGrace` is a const, so the `srv.Close()` and error branch in `server.go:52-55` has no
  test.

## Evidence
- Plan: `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (Task 3 Step 6; the idea's Expected output bullet 2).
- `backend/cmd/api/main.go:184-186`, `backend/cmd/api/server.go:35-37` (doc) and `:52-55`.
- Live repro, run by the reviewer on 2026-09-24 against the built binary with a request whose body
  trickles over 12 s: `kill -TERM`, then the log shows
  `server: shutdown: context deadline exceeded` and `exit=1 after 8.02s`.
- The code-review skill independently flagged `server.go:55` / `main.go:184` (medium).

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium, planned today (head of the cmd/api group).** Confirmed on `main`: `backend/cmd/api/server.go` `serve` returns `fmt.Errorf("shutdown: %w", err)` after `srv.Close()`, and `main.go` answers any non-nil `serve` error with `log.Fatalf` — so the one shutdown path that needs the deferred `pg.Close()`/`rdb.Close()` most (connections still open) is the one that skips them, and a Railway redeploy during a 60 s Google sync is logged as a crash. `go pet.RunHourly` and `go notify.RunWorker` are never joined (both do return on `ctx.Done()`, so the cost is log noise, but the idea's clause is unmet). Root cause: `serve` folds "the drain timed out and I force-closed" into the same error class as "the listener died", and `main` treats every error as fatal. Decision: `serve` returns a sentinel `ErrDrainTimedOut` for the overrun; `main` logs it and returns normally (exit 0 — the stop was planned, the log line is the signal); a listener error stays fatal. The grace becomes a parameter so the overrun branch gets a unit test with a 100 ms grace. The two goroutines are joined through a `sync.WaitGroup` bounded by the grace. Two sibling findings on the same two files are folded in: the second-signal swallow (`a-second-sigint-or-sigterm-during-the-8-s-shutdown-drain-is-.md`) and the missing `ReadTimeout` (`a-slow-request-body-still-holds-its-goroutine-indefinitely-n.md`). Plan: `harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md`.
