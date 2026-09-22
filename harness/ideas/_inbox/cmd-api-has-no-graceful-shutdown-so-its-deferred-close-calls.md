---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# cmd/api has no graceful shutdown so its deferred Close calls are unreachable

## Why
`backend/cmd/api/main.go` registers two cleanups that can never execute:

```
28:	defer pg.Close()
34:	defer func() { _ = rdb.Close() }()
...
48:	if err := r.Run(":" + cfg.Port); err != nil {
49:		log.Fatalf("server: %v", err)
50:	}
```

`gin.Engine.Run` blocks until the listener fails. There are only two ways out of `main`:

- the server errors, and `log.Fatalf` calls `os.Exit(1)`, which by definition does not run deferred
  functions; or
- the process receives a signal, and with no `os/signal` handling anywhere
  (`grep -rn 'signal\|Shutdown\|http.Server' backend/cmd backend/internal` → no matches) the runtime
  terminates without unwinding.

So both `defer`s are decorative. That is harmless today — the OS reclaims the sockets, and
`/healthz` holds no state — but it is misleading code in the file every later slice extends, and it
comes with a real consequence once there are writes: spec §8 deploys to Railway, which stops a
container with `SIGTERM` followed by a kill. Under the current shape an in-flight request is cut
mid-handler on every deploy, and the pgx pool is torn down by process death rather than drained.
Slice 3 (quests) writes `daily_progress` on `POST /quests/progress`; that is the point at which this
stops being cosmetic, and the fix is much cheaper to make now, in a 51-line wiring file, than after
several route groups depend on its shape.

## Expected output
`cmd/api` runs the server through an `http.Server` it can shut down, and the deferred cleanups
actually run:

- an `http.Server{Addr: ":" + cfg.Port, Handler: r}` started in a goroutine, with
  `ListenAndServe`'s `http.ErrServerClosed` treated as a normal exit rather than an error;
- `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` in `main`, and on cancellation
  `srv.Shutdown(ctx)` with a bounded grace period (a few seconds, named as a constant next to
  `health.pingTimeout`);
- the postgres and redis `Close` calls reached on the normal exit path — i.e. `main` returns instead
  of calling `log.Fatalf` after the server is up;
- the boot log gains a "shutting down" line so an operator can see the difference between a graceful
  stop and a crash.

A manual proof belongs in the amending plan's runtime section: start the binary, `curl /healthz`,
send `SIGTERM`, and observe the shutdown line and a zero exit status.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 8 — `main.go` is the plan's verbatim code block).
- `backend/cmd/api/main.go:28`, `:34` — the unreachable `defer`s; `:48-50` — the blocking `Run` and
  the `log.Fatalf` that skips them.
- `grep -rn 'signal\|Shutdown\|http.Server' backend/cmd backend/internal` → no matches.
- Spec §8 — Railway container deployment; `SIGTERM` is how the platform stops a revision.
