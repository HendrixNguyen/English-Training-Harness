---
idea: harness/ideas/_inbox/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md
status: approved
priority: medium
merged: false
---
# cmd/api: a drain overrun returns instead of Fatalf-ing, workers are joined, a second signal forces, and requests get a `ReadTimeout` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/a-second-sigint-or-sigterm-during-the-8-s-shutdown-drain-is-.md` → Task 3
- `harness/ideas/_inbox/a-slow-request-body-still-holds-its-goroutine-indefinitely-n.md` → Task 4

**Goal:** Every way the process stops runs the deferred `pg.Close()` / `rdb.Close()` and joins the two background workers; a redeploy during a 60 s Google sync logs one "drain grace exceeded" line and exits 0, not a fatal; a second Ctrl-C / SIGTERM ends the process at once; and a client that sends headers and then trickles a body cannot hold a goroutine for longer than 30 s.

**Why now (`priority: medium`):** All three confirmed on `origin/main` today. `backend/cmd/api/server.go` `serve` returns `fmt.Errorf("shutdown: %w", err)` after force-closing and `main.go` answers any `serve` error with `log.Fatalf` — the overrun path is the only one that *needs* the deferred Closes and the only one that skips them, and it reads as a crash in Railway's deploy log. `main.go` only `defer stop()`s the `signal.NotifyContext` cancel, so a second signal during the 8 s drain is swallowed (reviewer: `kill -TERM`, `kill -INT` 1 s later, `exit=1 after 8.02s`). `newServer` sets `ReadHeaderTimeout` and `IdleTimeout` only; `POST /api/v1/auth/google` is unauthenticated, so anyone can open connections and trickle bodies indefinitely — and the CORS plan's note that `http.MaxBytesReader` closes the connection is false through gin's writer (`$GOROOT/src/net/http/request.go:1266` on go1.27.1 checks an unexported interface that `gin.ResponseWriter` does not satisfy). Three findings, two files, one branch.

**Root cause:** the shutdown plan (2026-09-23) built the fast path — drain, return, deferred Closes — and left the slow path (`Shutdown` times out), the signal handler's lifetime, and the request-read deadline as they were.

**Design decisions:**
1. **A drain overrun is a sentinel, not a fatal.** `serve` returns `ErrDrainTimedOut` (wrapping the context error) after `srv.Close()`; `main` logs it and continues to its normal return — **exit status 0**: the stop was planned, and the log line is the operator's signal; a non-zero exit would make Railway treat a routine redeploy as a crash. A listener error stays fatal (nothing was serving).
2. **The grace is a parameter of `serve`,** so the overrun branch gets a unit test at 100 ms instead of 8 s. `ShutdownGrace` stays the production value `main` passes.
3. **Workers are joined with a `sync.WaitGroup` bounded by the grace.** `pet.RunHourly` and `notify.RunWorker` both return on `ctx.Done()` (read today), so the join is normally instant; the bound exists so a worker stuck in a Postgres call cannot hold the process past Railway's kill. A timeout logs which wait ran out and proceeds to the Closes.
4. **`stop()` runs as soon as the context is done** (`go func() { <-ctx.Done(); stop() }()`), the `signal.NotifyContext` idiom: after the first signal the handler is deregistered and a second SIGINT/SIGTERM gets Go's default disposition (immediate exit 130/143). The drain log line says "press Ctrl-C again to force".
5. **`ReadTimeout = 30 s`** on the server: headers plus body for one request must arrive within it. It bounds *reads* only; the handler's response can still take `google.SyncTimeout` (60 s) or several AI calls, so the no-`WriteTimeout` rationale is untouched. A 64 KiB JSON body needs milliseconds; 30 s is generous for a phone on a bad link and short enough that an attacker's connections turn over. `IdleTimeout` is set explicitly, so keep-alive waits are unaffected.
6. **The connection-close claim is corrected, not worked around.** A gin middleware cannot hand `MaxBytesReader` the underlying `*http.response`; with a read deadline the connection is bounded anyway. `bodylimit.go`'s comment and CODEMAP say so; the CORS plan's note is corrected in the execution summary only (a plan branch never edits `harness/plans/*`).

**Tech stack:** Go 1.25 stdlib (`net/http`, `os/signal`, `sync.WaitGroup.Go`). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` is not installed — use `grep -n`. Tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/cmd/api/server.go` | `ReadTimeout` const + field; `ErrDrainTimedOut`; `serve(ctx, srv, ln, grace)`; `waitWithin(wg, d)` |
| `backend/cmd/api/server_test.go` | overrun test; slow-body test; `waitWithin` tests; existing tests pass `ShutdownGrace`; `newServer` test asserts `ReadTimeout` |
| `backend/cmd/api/main.go` | early `stop()`; `WaitGroup` around the two workers; overrun → log + return; join before the deferred Closes |
| `backend/internal/middleware/bodylimit.go` | comment: the close hook does not fire through gin's writer; `ReadTimeout` bounds the connection |
| `harness/CODEMAP.md` | `cmd/api` and `middleware` paragraphs |

---

## Tasks

### Task 1: The drain overrun is a sentinel with an injectable grace

**Files:**
- Modify: `backend/cmd/api/server.go`, `backend/cmd/api/server_test.go`

- [ ] **Step 1: Write the failing test** in `server_test.go`, modelled on `TestServeStopsOnContextCancelAndDrainsInFlightRequests` (same `started`/`release` handler, `listen(t)`, `http.Get` in a goroutine):

```go
func TestServeReturnsErrDrainTimedOutAndClosesTheStragglerWhenTheGraceRunsOut(t *testing.T) {
	// handler blocks on release; client GET in flight; ctx cancelled; grace 100 ms
	…
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, ErrDrainTimedOut) {
			t.Fatalf("serve returned %v, want ErrDrainTimedOut", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after the grace ran out")
	}
	// The straggler was force-closed: the client sees an error, not a 200.
	if got := <-code; got != -1 {
		t.Fatalf("in-flight request got %d, want a transport error after srv.Close()", got)
	}
	unblock()
}
```

  Update the two existing `serve(...)` calls to `serve(ctx, newServer(h), ln, ShutdownGrace)` so the file compiles.
- [ ] **Step 2: Run red:** from `backend/`: `env -u … go test ./cmd/api/ -run TestServeReturnsErrDrainTimedOut -count=1` → compile error (`serve` arity / `ErrDrainTimedOut` undefined).
- [ ] **Step 3: Make it pass** in `server.go`:

```go
// ErrDrainTimedOut is serve's answer when in-flight requests outlive the grace:
// the remaining connections were force-closed. main logs it and returns
// normally (exit 0 — the stop was planned; the log line is the signal), so the
// deferred pg/rdb Close calls run, unlike a log.Fatalf.
var ErrDrainTimedOut = errors.New("server: drain grace exceeded; remaining connections were closed")

// serve runs srv on ln until ctx is done, then drains it within grace …
func serve(ctx context.Context, srv *http.Server, ln net.Listener, grace time.Duration) error {
	…
	log.Printf("shutting down: draining in-flight requests for up to %s (press Ctrl-C again to force)", grace)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // grace exceeded: close what is left so the process still exits
		<-errc          // Serve has returned; do not leak its goroutine
		return fmt.Errorf("%w: %v", ErrDrainTimedOut, err)
	}
	<-errc
	return nil
}
```

  Keep the doc comment's promise ("returns … so main can return") and add the overrun sentence.
- [ ] **Step 4: Run green:** `env -u … go test ./cmd/api/ -count=1 -race` → `ok` (the drain test and the listener-error test unchanged in behaviour).
- [ ] **Step 5: Commit:** `git commit -am "cmd/api: a drain overrun is ErrDrainTimedOut with an injectable grace, not a fatal"`.

### Task 2: `main` returns on an overrun and joins the workers

**Files:**
- Modify: `backend/cmd/api/server.go` (`waitWithin`), `backend/cmd/api/server_test.go`, `backend/cmd/api/main.go`

- [ ] **Step 1: Write the failing tests** for the helper: `TestWaitWithinReturnsTrueWhenTheGroupFinishes` (a `WaitGroup` with one goroutine that returns → `waitWithin(&wg, time.Second) == true` well under a second) and `TestWaitWithinReturnsFalseWhenItDoesNot` (a goroutine blocked on a channel → `waitWithin(&wg, 50*time.Millisecond) == false`; close the channel in `t.Cleanup`).
- [ ] **Step 2: Run red:** `env -u … go test ./cmd/api/ -run TestWaitWithin -count=1` → undefined.
- [ ] **Step 3: Add to `server.go`:**

```go
// waitWithin waits for wg up to d and reports whether it finished. main uses it
// to join the background workers after the drain without letting a stuck
// worker hold the process past Railway's kill window.
func waitWithin(wg *sync.WaitGroup, d time.Duration) bool {
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}
```

- [ ] **Step 4: Wire `main.go`.** Replace the two bare `go …` worker starts and the tail:

```go
	var workers sync.WaitGroup
	// Spec §8 hourly cron, in-process (§2.1). Sweeps at every :00 UTC.
	workers.Go(func() { pet.RunHourly(ctx, petSvc) })
	…
	if pushSender != nil {
		workers.Go(func() { notify.RunWorker(ctx, notifySvc, notify.PollInterval) })
	} else { … unchanged log … }
	…
	err = serve(ctx, newServer(r), ln, ShutdownGrace)
	switch {
	case errors.Is(err, ErrDrainTimedOut):
		log.Printf("server: %v", err) // planned stop that ran long: not fatal, exit 0 (see ErrDrainTimedOut)
	case err != nil:
		log.Fatalf("server: %v", err) // the listener died: nothing was serving
	}
	// Both workers return on ctx.Done(); bound the join so a worker stuck in a
	// store call cannot hold the process past Railway's kill window.
	if !waitWithin(&workers, ShutdownGrace) {
		log.Printf("shutdown: background workers did not stop within %s; closing stores anyway", ShutdownGrace)
	}
	log.Printf("shutdown complete") // main returns: deferred pg.Close / rdb.Close run
```

  (`sync.WaitGroup.Go` exists since Go 1.25; `go.mod` says `go 1.25`. Keep the `ctx`/`stop` lines for Task 3.)
- [ ] **Step 5: Run green:** `env -u … go test ./cmd/api/ -count=1 -race` → `ok`; `go vet ./...` → silent; `go build ./...` → ok.
- [ ] **Step 6: Commit:** `git commit -am "cmd/api: main returns on a drain overrun and joins the pet and notify workers"`.

### Task 3: A second signal forces

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1:** Right after `ctx, stop := signal.NotifyContext(...)` / `defer stop()`:

```go
	// signal.NotifyContext keeps the handler installed until stop() runs. Call
	// it the moment the context is done, so a second SIGINT/SIGTERM during the
	// drain gets Go's default disposition and ends the process at once
	// ("press Ctrl-C again to force"). defer stop() stays for the error paths.
	go func() { <-ctx.Done(); stop() }()
```

- [ ] **Step 2: Live proof** (record the output in the execution summary). From `backend/`, with an isolated stack (`COMPOSE_PROJECT_NAME=cmdapi5 POSTGRES_PORT=55502 REDIS_PORT=56502 docker compose up -d --wait`) and the vars exported in a subshell only:

```bash
go build -o /tmp/api-drain ./cmd/api
(set -a; . ./.env; set +a; DATABASE_URL=postgres://english:english@localhost:55502/english?sslmode=disable REDIS_URL=redis://localhost:56502/0 PORT=18094 /tmp/api-drain > /tmp/api-drain.log 2>&1 & echo $! > /tmp/api-drain.pid)
sleep 2
# hold a request open: headers now, body never
(printf 'POST /api/v1/auth/google HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{'; sleep 30) | nc 127.0.0.1 18094 > /dev/null &
sleep 1; start=$(date +%s); kill -TERM "$(cat /tmp/api-drain.pid)"; sleep 1; kill -INT "$(cat /tmp/api-drain.pid)"
wait "$(cat /tmp/api-drain.pid)" 2>/dev/null; echo "exit=$? after $(( $(date +%s) - start ))s"
tail -3 /tmp/api-drain.log
```

  Expected: the process is gone within ~1 s of the second signal (`after 1s`, exit 130 or 143 — the default disposition, no "shutdown complete" line); the log shows the "draining … (press Ctrl-C again to force)" line. Then repeat **without** the second `kill`: expect `shutdown: … drain grace exceeded` after ~8 s, then `shutdown complete`, `exit=0`. `COMPOSE_PROJECT_NAME=cmdapi5 docker compose down` after.
- [ ] **Step 3: Commit:** `git commit -am "cmd/api: a second SIGINT/SIGTERM during the drain ends the process at once"`.

### Task 4: `ReadTimeout`, and the truth about `MaxBytesReader`

**Files:**
- Modify: `backend/cmd/api/server.go`, `backend/cmd/api/server_test.go`, `backend/internal/middleware/bodylimit.go`

- [ ] **Step 1: Write the failing tests.** Extend `TestNewServerBoundsHeaderReadsButNotWrites` to also require `srv.ReadTimeout == ReadTimeout && ReadTimeout > 0` (rename to `…BoundsReadsButNotWrites`). Add:

```go
func TestASlowBodyIsCutOffAtReadTimeout(t *testing.T) {
	// The handler tries to read the body, as every JSON handler does.
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.ReadAll(r.Body); w.WriteHeader(http.StatusOK) })
	ln := listen(t)
	srv := newServer(h)
	srv.ReadTimeout = 200 * time.Millisecond // production is 30 s; the deadline is what is under test
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = serve(ctx, srv, ln, ShutdownGrace) }()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	fmt.Fprint(conn, "POST /x HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\n{") // headers + 1 byte, then nothing
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1)
	start := time.Now()
	_, err = conn.Read(buf) // the server must close the connection, not answer 200
	if err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("read = %v after %s; want the server to close the connection at ReadTimeout", err, time.Since(start))
	}
}
```

- [ ] **Step 2: Run red:** `env -u … go test ./cmd/api/ -run 'TestNewServer|TestASlowBody' -count=1` → the `ReadTimeout` assertion fails; the slow-body test times out at 3 s with `err == nil`? — no: with no deadline the read blocks until the client deadline, so it fails with "i/o timeout after 3 s > 2 s".
- [ ] **Step 3: Make them pass** in `server.go`:

```go
	// ReadTimeout bounds one whole request read — headers and body — so a
	// client that sends headers promptly and then trickles a body cannot hold a
	// goroutine and a connection indefinitely (POST /auth/google needs no
	// token). It bounds reads only: handler responses may still take
	// google.SyncTimeout or several AI calls (see newServer on WriteTimeout).
	ReadTimeout = 30 * time.Second
```

  and `newServer` sets `ReadTimeout: ReadTimeout` next to the two existing fields; extend its doc comment with the read-deadline sentence.
- [ ] **Step 4: Correct `bodylimit.go`'s comment** (the behaviour is unchanged): add "MaxBytesReader's connection-close hook never fires here — net/http checks an unexported interface that gin's ResponseWriter does not satisfy — so an over-limit request answers 400 on a connection that stays open; cmd/api's ReadTimeout is what bounds a slow sender."
- [ ] **Step 5: Run green:** `env -u … go test ./cmd/api/ ./internal/middleware/ -count=1 -race` → `ok`; `cd backend && make check` → all `ok`.
- [ ] **Step 6: Commit:** `git commit -am "cmd/api: ReadTimeout 30 s bounds a trickling request body; bodylimit comment corrected"`.

### Task 5: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** `cmd/api` paragraph: replace "`server.go` owns `newServer` (`ReadHeaderTimeout` 10 s, `IdleTimeout` 120 s, **no `WriteTimeout`** — …) and `serve(ctx, srv, ln)`, which drains on the `signal.NotifyContext` context within `ShutdownGrace` (8 s) and returns so the deferred `Close` calls run" with: "`server.go` owns `newServer` (`ReadHeaderTimeout` 10 s, `ReadTimeout` 30 s — headers **and** body, so a trickling sender is bounded — `IdleTimeout` 120 s, **no `WriteTimeout`** — …) and `serve(ctx, srv, ln, grace)`, which drains on the `signal.NotifyContext` context within the grace (`ShutdownGrace`, 8 s) and returns; when the drain runs out it force-closes and returns `ErrDrainTimedOut`, which `main` logs and treats as a planned stop (exit 0), so the deferred `Close` calls run on every path; `main` joins `pet.RunHourly` and `notify.RunWorker` through a `WaitGroup` bounded by the same grace, and calls `stop()` the moment the context is done so a second SIGINT/SIGTERM ends the process at once ('press Ctrl-C again to force')". `middleware` paragraph: after "instead of being allocated" add "(the connection stays open — `MaxBytesReader`'s close hook does not fire through gin's writer; `cmd/api`'s `ReadTimeout` is what bounds a slow sender)".
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — cmd/api shutdown paths, ReadTimeout, bodylimit truth"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./cmd/api/ -count=1 -race -v 2>&1 | grep -c '^--- PASS'
# expect: 7 (3 today + overrun + 2 waitWithin + slow body)
grep -n 'ReadTimeout' cmd/api/server.go
# expect: the const, the newServer field, and the doc sentence
grep -n 'log.Fatalf("server' cmd/api/main.go
# expect: exactly 1 (the listener-error branch)
grep -n 'workers.Go\|waitWithin(&workers' cmd/api/main.go
# expect: 2 Go calls + 1 waitWithin
grep -n 'go pet.RunHourly\|go notify.RunWorker' cmd/api/main.go
# expect: no output (both are now WaitGroup.Go)
grep -n 'close hook\|ReadTimeout' internal/middleware/bodylimit.go
# expect: the corrected comment
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
cd ..
grep -n 'ErrDrainTimedOut\|ReadTimeout' harness/CODEMAP.md
# expect: cmd/api paragraph (both), middleware paragraph (ReadTimeout)
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: all four jobs green
```

Live proofs from Task 3 Step 2 (both runs: second-signal exit within ~1 s; single-signal overrun → `exit=0` after ~8 s with "drain grace exceeded" then "shutdown complete") go in the execution summary verbatim.

## Notes and open questions

- **Exit 0 on an overrun** is decision 1. If the owner prefers a non-zero status to make overruns visible in Railway's restart stats, it is one line in `main`; the log line is the same either way.
- **Why not make `ShutdownGrace` a flag/env?** Nothing needs it configurable; the test injects it through `serve`'s parameter.
- **`ReadTimeout` and keep-alive:** Go arms the read deadline per request when it starts reading the next request line; with `IdleTimeout` set explicitly, an idle keep-alive connection is governed by `IdleTimeout`, not `ReadTimeout`. Verified by reading `net/http/server.go` `conn.serve` (go1.27.1); the slow-body test covers the case that matters.
- **The CORS plan's note** ("`http.MaxBytesReader` also tells the server to close the connection") is wrong for gin; record the correction in this plan's execution summary and in CODEMAP — do not edit `harness/plans/2026-09-24-the-api-sends-no-cors-headers…` on the branch.
- **Out of scope:** a per-handler read deadline shorter than 30 s; `http.TimeoutHandler` (would cut the AI/Google routes).
