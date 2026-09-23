---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# main.go installs a signal handler with no server shutdown so the API now ignores SIGINT and SIGTERM

## Why
The pet slice needed a cancellable context for its cron goroutine and wired one through
`signal.NotifyContext`:

```go
// backend/cmd/api/main.go (added by harness/2026-09-22-high-pet-health-streak-…)
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
...
go pet.RunHourly(ctx, petSvc)
...
if err := r.Run(":" + cfg.Port); err != nil {   // still blocks forever
```

`signal.NotifyContext` does not merely observe the signals — it **registers a handler for them**,
which replaces Go's default disposition of terminating the process. Nothing then unblocks
`gin.Engine.Run`, so the only effect of SIGINT/SIGTERM is that the pet cron stops. The process
keeps listening and serving, indefinitely.

On `main` this file has `ctx := context.Background()` and no `os/signal` import at all
(`git show main:backend/cmd/api/main.go`), so before this branch SIGTERM killed the API instantly.
This branch converts "ungraceful stop" into "no stop at all", which is strictly worse in two
places the owner touches daily:

- **Local development.** `Ctrl-C` on the running binary does nothing, and a second `Ctrl-C` does
  nothing either — the handler stays registered. The port is held until the developer finds the
  pid and sends `SIGKILL`. The executor hit exactly this and recorded it under *Runtime proof*:
  "`SIGTERM` alone did not stop the running API binary … the process kept running until
  `SIGKILL`."
- **Deployment.** Spec §9 deploys to Railway, which stops a revision with SIGTERM and kills after
  a grace period. Every deploy now burns the full grace window, and during it the replica is
  still accepting traffic while its cron is already dead — the worst of both states.

This is the other half of the existing low-priority bug
`cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md`, whose *Expected output* asks for
`signal.NotifyContext` **and** `http.Server` + `srv.Shutdown(ctx)` together. Half of it landed.
Filing separately because the severity changed: that bug described unreachable `defer`s (cosmetic);
this one describes a process that cannot be stopped by a signal (operational).

## Expected output
`cmd/api` can be stopped by a signal again, and stops cleanly:

- the router is served through `srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}` started in a
  goroutine, treating `http.ErrServerClosed` as a normal exit;
- on `<-ctx.Done()`, `main` calls `srv.Shutdown` with a bounded grace period (a named constant next
  to `health.pingTimeout`), logs a "shutting down" line, and **returns** rather than
  `log.Fatalf`, so the existing `defer pg.Close()` / `defer rdb.Close()` finally run and the pet
  cron's cancellation is joined rather than raced;
- a runtime proof in the amending plan: start the binary, `curl /healthz`, send `SIGTERM`, observe
  the shutdown line and a **zero exit status within the grace period** — no `SIGKILL`;
- the CI/dev docs stop needing the "this binary ignores SIGTERM" caveat.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 9 —
  the `main.go` block; the plan's *Execution summary* → *Runtime proof* records the symptom and
  correctly scopes it out of the task list).
- `backend/cmd/api/main.go` — `signal.NotifyContext(...)` added; `r.Run(...)` unchanged; no
  `http.Server`, no `Shutdown`.
- `git show main:backend/cmd/api/main.go` — `ctx := context.Background()`, no `signal` import:
  the regression is introduced by this branch.
- Reviewer runtime proof, 2026-09-23: the API on port 8097 required `kill -9`.
- Spec §9 — Railway container deployment; SIGTERM is how the platform stops a revision.
- Supersedes in severity: `harness/ideas/_inbox/cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md`
  (priority low) — the two should be fixed by one amendment.
