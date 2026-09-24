---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
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
