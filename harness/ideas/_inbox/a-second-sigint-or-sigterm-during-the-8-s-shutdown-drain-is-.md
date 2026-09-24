---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A second SIGINT or SIGTERM during the 8 s shutdown drain is swallowed because stop is only deferred

## Why
`main` gets its context from `signal.NotifyContext(…, os.Interrupt, syscall.SIGTERM)` and only
`defer stop()`s it. The handler therefore stays registered through the entire drain. A developer
who presses Ctrl-C and then Ctrl-C again, because a slow request is holding shutdown open, gets
nothing from the second press and waits the full `ShutdownGrace` (8 s). The same applies to an
operator's second `kill`. The standard idiom, used in the `signal.NotifyContext` docs, is to call
`stop()` as soon as the context is done so a second signal gets Go's default disposition and
terminates immediately. Before this branch the process never stopped at all, so this only became
visible now.

## Expected output
- Once shutdown begins, `stop()` is called before draining (for example in `main` right after
  `serve` observes `ctx.Done()`, or by passing `stop` into `serve`). A second SIGINT/SIGTERM
  then ends the process at once.
- One line in the shutdown log or CODEMAP: "press Ctrl-C again to force".

## Evidence
- Plan: `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (Task 3 Step 6: "`defer stop()` stays").
- `backend/cmd/api/main.go:48-49`.
- Reviewer run, 2026-09-24: while a 12 s request was draining, `kill -TERM`, then 1 s later
  `kill -INT`. The process was still alive after the second signal and exited only at the grace
  deadline (`exit=1 after 8.02s`).
- The code-review skill independently flagged `main.go:48` (low).
