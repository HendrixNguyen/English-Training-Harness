---
idea: harness/ideas/_inbox/main-go-installs-a-signal-handler-with-no-server-shutdown-so.md
status: approved
priority: high
merged: false
---
# cmd/api: serve through an http.Server that drains on SIGINT/SIGTERM, and harden the wiring file once — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/main-go-installs-a-signal-handler-with-no-server-shutdown-so.md`
**Goal:** Make the API stop on SIGINT/SIGTERM again — draining in-flight requests within a bounded grace and reaching the deferred `pg.Close()`/`rdb.Close()` — and, because `backend/cmd/api/main.go` is the file the owner keeps merging by hand, land every other pending change to it in the same branch: Gin release mode + trusted proxies, server timeouts, embedded tzdata, the stale header comment, the doubled `config:` prefix, and the `.env.example` application section.

**Why now (`priority: high`):** `main.go` on `main` installs `signal.NotifyContext(…, os.Interrupt, syscall.SIGTERM)` (the pet cron needed a cancellable context) and still blocks in `r.Run`. `NotifyContext` *registers* a handler, replacing Go's default terminate-on-signal, and nothing unblocks `Run` — so the process now ignores Ctrl-C and SIGTERM: the reviewer needed `kill -9`, the notify executor's test server would not die, and Railway (spec §9) stops a revision with SIGTERM, so every deploy burns the full grace window serving traffic with a dead cron. Found by a reviewer and hit independently by an executor.

**⚠ Merge-order precondition — read before `git worktree add`:** the unmerged branch `harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue` also edits `main.go` (adds the `notify` wiring and `go notify.RunWorker(ctx, …)`). The owner should merge notify **first**, then this. If this plan is executed before notify lands, the executor must `git fetch origin main && git merge origin/main --no-edit` into the worktree after notify merges and resolve `main.go` by hand — every snippet below is written as an *edit to a region*, not a whole-file replacement, so it applies on top of either version. Do not touch the notify wiring block.

**Root cause (from the idea's `## Evaluation`):** `signal.NotifyContext` was added for `pet.RunHourly` without the `http.Server`/`Shutdown` half that `cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md` (now rejected as a duplicate of this) had already asked for. Half a fix is worse than none: "ungraceful stop" became "no stop".

**Folded findings (each rejected in the inbox with a pointer here — their fix is a task below):**
- `gin-default-ships-debug-mode-and-all-proxies-trusted-to-prod.md` → Task 1 + Task 3 (`GIN_MODE`, `SetTrustedProxies(nil)`).
- `timezone-handling-depends-on-system-tzdata-with-no-time-tzda.md` → Task 3 (`_ "time/tzdata"` + boot check).
- `stale-main-go-header-comment-and-a-double-prefixed-config-er.md` → Task 3.
- `backend-env-example-omits-the-app-s-own-database-url-redis-u.md` → Task 4.
- `no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` is **not** folded: only its server-timeout half is taken here (Task 2); the body-limit middleware is a separate selected idea whose `main.go` diff will be one `Use(...)` line.

**Deliberately out of scope:** a deadline on the boot migration. `migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md` (selected, medium) shows that a cancelled/expired context during `Migrate` leaks the advisory lock into the pool; adding a deadline before that fix lands would make the leak live. Leave `store.Migrate(ctx, …)` exactly as it is.

**Architecture:** one new file `backend/cmd/api/server.go` holds the server construction and the serve/shutdown loop as two small functions (`newServer`, `serve`) so they are unit-testable with a real `net.Listener` on `127.0.0.1:0` — `main.go` itself stays untestable wiring and shrinks by a few lines. `config.Load` gains `GinMode` (env `GIN_MODE`, default `release`, validated) so `main` never reads `os.Getenv` for it; Gin's own `GIN_MODE` handling is bypassed by calling `gin.SetMode(cfg.GinMode)` before `gin.Default()`. `WriteTimeout` stays **unset** on purpose — `google.SyncTimeout` is 60 s and an onboarding assessment can legitimately run several `airouter.Route` calls of up to `len(FallbackOrder) × ProviderTimeout` each; a server-wide write deadline would cut those responses off. `ReadHeaderTimeout` and `IdleTimeout` are set (slowloris / idle keep-alives); body size is a separate middleware (other idea).

**Tech stack:** Go 1.25, Gin v1.12, stdlib `net/http`, `os/signal`, `time/tzdata`. No new dependencies.

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n` and `go test -timeout`. Bound the boot proof with `curl --max-time` and a `wait` on the pid.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/config/config.go` | Add `GinMode` field; read/validate `GIN_MODE` (default `release`) |
| `backend/internal/config/config_test.go` | Add `TestLoadDefaultsGinModeToRelease`, `TestLoadRejectsAnUnknownGinMode` |
| `backend/cmd/api/server.go` | **New**: `ShutdownGrace`, `ReadHeaderTimeout`, `IdleTimeout`, `newServer`, `serve` |
| `backend/cmd/api/server_test.go` | **New**: drain-on-cancel, listener-error, timeout-fields tests |
| `backend/cmd/api/main.go` | Header comment; `_ "time/tzdata"` + boot check; `gin.SetMode` + `SetTrustedProxies(nil)`; `log.Fatal(err)` for config; replace `r.Run` with `net.Listen` + `serve` |
| `backend/.env.example` | Add the application section (`DATABASE_URL`, `REDIS_URL`, `PORT`, `GIN_MODE`, `JWT_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`) + "Go does not read this file" note |
| `backend/Makefile` | Comment on `run`: how to export `.env` |
| `harness/CODEMAP.md` | New `**cmd/api**` bullet; `config` env list mention of `GIN_MODE`; CI section unchanged |

---

## Tasks

### Task 1: `config.Load` reads and validates `GIN_MODE`

**Files:**
- Modify: `backend/internal/config/config_test.go` (append)
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Write the failing tests**

Append to `backend/internal/config/config_test.go` (the file already imports only `testing`):

```go
func TestLoadDefaultsGinModeToRelease(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("GIN_MODE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// gin.Default() alone would run in debug mode; production must not.
	if cfg.GinMode != "release" {
		t.Fatalf("GinMode = %q, want release", cfg.GinMode)
	}
}

func TestLoadRejectsAnUnknownGinMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("JWT_SECRET", "s3cret")

	for _, mode := range []string{"debug", "release", "test"} {
		t.Setenv("GIN_MODE", mode)
		if cfg, err := Load(); err != nil || cfg.GinMode != mode {
			t.Fatalf("GIN_MODE=%s: cfg=%+v err=%v", mode, cfg, err)
		}
	}
	t.Setenv("GIN_MODE", "verbose")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for GIN_MODE=verbose (gin.SetMode would panic), got nil")
	}
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `go test ./internal/config/... -count=1 -timeout 60s -run 'GinMode' -v`
Expected: both `FAIL` — `cfg.GinMode undefined` (compile error) is the expected first failure.

- [ ] **Step 3: Add the field and the validation**

In `backend/internal/config/config.go`, add to `Config` after `JWTSecret`:

```go
	// GinMode is Gin's run mode: "release" (default), "debug" or "test". Read
	// from GIN_MODE and validated here so the deployed binary never runs Gin's
	// debug logging by accident — gin.Default() alone defaults to debug — and
	// so gin.SetMode (which panics on an unknown value) is only ever given a
	// valid one. Not in spec §8; documented in backend/.env.example.
	GinMode string
```

In `Load`, before `return cfg, nil`:

```go
	cfg.GinMode = os.Getenv("GIN_MODE")
	if cfg.GinMode == "" {
		cfg.GinMode = "release"
	}
	switch cfg.GinMode {
	case "debug", "release", "test":
	default:
		return Config{}, fmt.Errorf("config: GIN_MODE must be debug, release or test, got %q", cfg.GinMode)
	}
```

(The literals equal `gin.DebugMode`/`gin.ReleaseMode`/`gin.TestMode`; `config` stays free of the gin import.)

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/config/... -count=1 -timeout 60s -v`
Expected: all `PASS`, including the five pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "config: read GIN_MODE, default release, reject unknown modes"
```

---

### Task 2: `cmd/api/server.go` — a server that drains on context cancellation

**Files:**
- Create: `backend/cmd/api/server_test.go`
- Create: `backend/cmd/api/server.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/cmd/api/server_test.go`:

```go
package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// The signal arrives while a request is in flight: serve must let it finish,
// close the listener, and return nil — the shape a Railway SIGTERM produces.
func TestServeStopsOnContextCancelAndDrainsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond) // well under ShutdownGrace
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

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want nil on a clean shutdown", err)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
	if got := <-code; got != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200 (drained, not cut off)", got)
	}
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		t.Fatal("listener still accepting connections after shutdown")
	}
}

func TestServeReturnsAListenerError(t *testing.T) {
	ln := listen(t)
	_ = ln.Close() // Serve on a closed listener fails at once
	if err := serve(context.Background(), newServer(http.NotFoundHandler()), ln); err == nil {
		t.Fatal("serve returned nil for a dead listener")
	}
}

// WriteTimeout must stay unset: google.SyncTimeout (60 s) and onboarding's AI
// calls legitimately outlive any sane server-wide write deadline.
func TestNewServerBoundsHeaderReadsButNotWrites(t *testing.T) {
	srv := newServer(http.NotFoundHandler())
	if srv.ReadHeaderTimeout <= 0 || srv.IdleTimeout <= 0 {
		t.Fatalf("ReadHeaderTimeout/IdleTimeout unset: %+v", srv)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout = %s, want 0 (see newServer doc)", srv.WriteTimeout)
	}
}
```

- [ ] **Step 2: Run and confirm they fail to compile**

Run: `go test ./cmd/api/... -count=1 -timeout 60s -run 'Serve|NewServer' -v`
Expected: `undefined: serve`, `undefined: newServer`, `undefined: ShutdownGrace`.

- [ ] **Step 3: Create `server.go`**

Create `backend/cmd/api/server.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	// ShutdownGrace bounds srv.Shutdown once SIGINT/SIGTERM arrives: in-flight
	// requests get this long to finish before the process exits. Railway sends
	// SIGTERM and kills after its own grace window, so this stays under 10 s.
	ShutdownGrace = 8 * time.Second
	// ReadHeaderTimeout caps how long a client may take to send request
	// headers (slowloris). Request bodies are bounded per handler, not here.
	ReadHeaderTimeout = 10 * time.Second
	// IdleTimeout closes keep-alive connections that sit idle.
	IdleTimeout = 120 * time.Second
)

// newServer wraps the router in an http.Server that main can shut down.
// WriteTimeout is deliberately unset: POST /integrations/google/sync runs up
// to google.SyncTimeout (60 s) and an onboarding assessment may spend several
// airouter.Route calls of up to len(FallbackOrder) × ProviderTimeout each; a
// server-wide write deadline would cut those responses off mid-flight.
func newServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: ReadHeaderTimeout, IdleTimeout: IdleTimeout}
}

// serve runs srv on ln until ctx is done, then drains it within ShutdownGrace.
// It returns nil on a clean shutdown (Serve's own http.ErrServerClosed is the
// normal exit, not an error) and the listener error otherwise, so main can
// return — letting its deferred pg/rdb Close calls run — instead of Fatalf-ing.
func serve(ctx context.Context, srv *http.Server, ln net.Listener) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Printf("shutting down: draining in-flight requests for up to %s", ShutdownGrace)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // grace exceeded: close what is left so the process still exits
		return fmt.Errorf("shutdown: %w", err)
	}
	<-errc // Serve has returned http.ErrServerClosed
	return nil
}
```

- [ ] **Step 4: Run the tests, including under the race detector**

Run: `go test ./cmd/api/... -count=1 -timeout 120s -race -v`
Expected: three `PASS`; no `WARNING: DATA RACE`.

- [ ] **Step 5: Commit**

```bash
git add cmd/api/server.go cmd/api/server_test.go
git commit -m "cmd/api: http.Server with bounded Shutdown on context cancel"
```

---

### Task 3: Rewire `main.go` — mode, proxies, tzdata, header, prefix, serve

**Files:**
- Modify: `backend/cmd/api/main.go` (regions named below; leave every package wiring block untouched)

- [ ] **Step 1: Header comment (lines 1-2)**

Replace the two-line package comment with:

```go
// Command api is the single Go process of spec §2.1: it mounts every package's
// routes under /api/v1 (the list lives in harness/CODEMAP.md, not here) and
// runs the in-process cron workers. It serves through an http.Server that
// drains on SIGINT/SIGTERM (server.go).
```

- [ ] **Step 2: Imports**

Add `"net"` to the stdlib group and, as its own blank import with a comment, embedded tzdata:

```go
	_ "time/tzdata" // embed the zone database so timezone math never depends on the container image
```

- [ ] **Step 3: Config error prefix**

Change `log.Fatalf("config: %v", err)` to `log.Fatal(err)` — `config.Load` errors already begin `config: `. Leave the `postgres:`/`redis:`/`migrate:` prefixes: those disambiguate three `store:`-prefixed calls.

- [ ] **Step 4: tzdata boot check**

Immediately after the config block (before `store.NewPostgres`):

```go
	if _, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err != nil {
		log.Fatalf("tzdata: %v (time/tzdata is embedded; this should be impossible)", err)
	}
```

- [ ] **Step 5: Gin mode and trusted proxies**

Replace `r := gin.Default()` with:

```go
	gin.SetMode(cfg.GinMode) // validated by config.Load; release unless GIN_MODE says otherwise
	r := gin.Default()
	// Nothing reads c.ClientIP() yet. Trust no proxy headers until something
	// does and the platform's proxy range is known — gin.Default() trusts all.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("gin: %v", err)
	}
```

- [ ] **Step 6: Replace `r.Run`**

Replace the final block

```go
	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
```

with

```go
	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("listening on %s (GIN_MODE=%s)", ln.Addr(), cfg.GinMode)
	if err := serve(ctx, newServer(r), ln); err != nil {
		log.Fatalf("server: %v", err)
	}
	log.Printf("shutdown complete") // main returns: deferred pg.Close / rdb.Close run
```

(`ctx` is the existing `signal.NotifyContext` context; do not create a second one. `defer stop()` stays.)

- [ ] **Step 7: Build, vet, format, whole suite**

Run:
```bash
go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)" && echo fmt-ok
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s
```
Expected: `fmt-ok`; every package `ok`. (Three files elsewhere are known-unformatted on `main` — that is `ci-never-runs-gofmt-…`'s job, not this plan's; the `gofmt -l` above is scoped to what this plan touches.)

- [ ] **Step 8: Commit**

```bash
git add cmd/api/main.go
git commit -m "cmd/api: stop on SIGINT/SIGTERM via serve(); release mode, no trusted proxies, embedded tzdata"
```

---

### Task 4: `.env.example`, Makefile, CODEMAP

**Files:**
- Modify: `backend/.env.example` (append a section after the `REDIS_PORT` block, before the TEST_* block)
- Modify: `backend/Makefile` (`run` target comment)
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1: `.env.example` application section**

Insert after the `POSTGRES_PORT`/`REDIS_PORT` lines:

```
# Application variables (spec §8 + JWT_SECRET, which §8 omits). The Go process
# does NOT read this file — Docker Compose does. Export them before `make run`
# (`set -a; . ./.env; set +a`) or set them on the Railway service. The URLs
# below point at the dev stack on POSTGRES_PORT / REDIS_PORT above.
DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=8080
# release (default) | debug (per-request Gin logging, local only) | test
GIN_MODE=release
# HS256 session key: at least 32 random bytes, e.g. `openssl rand -base64 32`
JWT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

Comments stay on their own lines (Compose's `.env` parser and `. ./.env` both tolerate that; trailing inline comments are not portable).

- [ ] **Step 2: Makefile `run` comment**

Above the `run:` target:

```make
# The Go process does not read .env; export it first:
#   set -a; . ./.env; set +a; make run
```

- [ ] **Step 3: Prove the documented loop, then the safe failure**

From `backend/` with a scratch `.env` copied from `.env.example` (fill `JWT_SECRET`, `GOOGLE_CLIENT_*` with dummies; set `COMPOSE_PROJECT_NAME=<slug>` and spare ports per AGENTS.md):
```bash
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose -p <slug> up -d --wait --wait-timeout 120
set -a; . ./.env; set +a; go run ./cmd/api &   # note the pid
```
Expected boot log: `migrations applied …` (first run), `listening on [::]:8080 (GIN_MODE=release)`, and **no** line containing `[GIN-debug]`. Then `env -u DATABASE_URL -u REDIS_URL go run ./cmd/api` must print exactly `config: DATABASE_URL is required` (single prefix) and exit 1; `GIN_MODE=verbose … go run ./cmd/api` must refuse with the `GIN_MODE must be` message.

- [ ] **Step 4: CODEMAP**

Add to `harness/CODEMAP.md` under *Backend packages*, as the last bullet before the frontend section:

> - **cmd/api** — the one binary (spec §2.1). `main.go` is wiring only: `config.Load` → Postgres/Redis → `store.Migrate` (deadline-free on purpose until the advisory-lock unlock leak is fixed) → `gin.SetMode(cfg.GinMode)` (`GIN_MODE`, default `release`) + `SetTrustedProxies(nil)` → route groups → in-process crons → `serve()`. `server.go` owns `newServer` (`ReadHeaderTimeout` 10 s, `IdleTimeout` 120 s, **no `WriteTimeout`** — `google.SyncTimeout` and onboarding's AI calls outlive any sane one) and `serve(ctx, srv, ln)`, which drains on the `signal.NotifyContext` context within `ShutdownGrace` (8 s) and returns so the deferred `Close` calls run; `cmd/api` tests use a real listener on `127.0.0.1:0`. `_ "time/tzdata"` is imported so zone math never depends on the image. `backend/.env.example` documents the application variables; the Go process does not read `.env` (`set -a; . ./.env; set +a` before `make run`).

And in the `store` bullet's mention of `.env.example`, nothing changes. If a `config` bullet exists by the time this executes (`codemap-does-not-document-the-config-and-health-packages-and.md`), add `GIN_MODE` to its env list; otherwise the `cmd/api` bullet above is sufficient.

- [ ] **Step 5: Commit**

```bash
git add .env.example Makefile ../harness/CODEMAP.md
git commit -m "backend: document the application env in .env.example; CODEMAP cmd/api entry"
```

---

## Verification

From `backend/` in the worktree, dev stack up on non-default ports with a unique compose project (AGENTS.md), all `TEST_*` vars unset:

1. `go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)"` → exit 0.
2. `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s` → every package `ok`.
3. `go test ./cmd/api/... -count=1 -race -timeout 120s` → `ok`, no race report.
4. **The signal proof (the point of the plan):**
   ```bash
   go build -o /tmp/<slug>-api ./cmd/api
   set -a; . ./.env; set +a; PORT=8097 /tmp/<slug>-api > /tmp/<slug>-api.log 2>&1 & PID=$!
   sleep 2; curl --max-time 5 -s http://127.0.0.1:8097/healthz; echo
   kill -TERM $PID; wait $PID; echo "exit=$?"
   grep -c 'GIN-debug' /tmp/<slug>-api.log; grep -n 'shutting down\|shutdown complete' /tmp/<slug>-api.log
   ```
   Expected: `{"postgres":"ok","redis":"ok","status":"ok"}`; `exit=0` within `ShutdownGrace`; `0` `GIN-debug` lines; both shutdown lines present. Repeat with `kill -INT` → same. **Paste the real output into the execution summary — this is the regression the idea reports.**
5. In-flight drain, live: start the binary, in one shell `curl --max-time 20 -s -X POST http://127.0.0.1:8097/api/v1/auth/google -d '{}' -H 'Content-Type: application/json' &` immediately followed by `kill -TERM $PID` in another; the curl must still receive its `400 invalid_request`, and the process exits 0.
6. `env -u DATABASE_URL -u REDIS_URL /tmp/<slug>-api` → prints `config: DATABASE_URL is required` once, exit 1. `GIN_MODE=verbose … /tmp/<slug>-api` → refuses.
7. `GIN_MODE=debug` boot prints `[GIN-debug]` route lines (the switch works both ways) — one line of evidence is enough.
8. Clean up: `kill` anything left, `docker compose -p <slug> down`, delete the scratch `.env`; `pgrep -fl <slug>-api` and `docker ps` show nothing of yours.
9. Push; `gh run watch` — all four CI jobs green on the branch.

## Notes and open questions

- **Why `ShutdownGrace = 8 s`:** Railway's default stop grace is 10 s (configurable per service); staying under it means a clean exit is observed as such rather than racing the kill. If the owner raises Railway's grace, raise this constant with it.
- **`SetTrustedProxies(nil)`** makes `c.ClientIP()` return the TCP peer (the platform proxy) rather than a spoofable `X-Forwarded-For`. When a per-IP limiter appears, replace `nil` with the platform's proxy CIDR — that is a config value, not a code change.
- **Notify branch** — see the precondition at the top. Its `main.go` adds a `notify` import, a VAPID block and `go notify.RunWorker(ctx, …)` between the pet cron and the route groups; none of this plan's edits overlap those lines, but git will still need a human to confirm the merge if both land close together.
- **Not here:** body-size middleware (`no-post-handler-bounds…`, one `Use` line later), migration deadline (`migrate-s-advisory-lock-leaks…` first), `JWT_SECRET` length validation (`jwt-secret-is-accepted…`, also `config.go` — schedule after this so `config` merges cleanly).
