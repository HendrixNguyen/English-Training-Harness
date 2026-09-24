---
plan: harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md, harness/ideas/_inbox/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md, harness/ideas/_inbox/the-documented-set-a-env-export-also-exports-test-database-u.md, harness/ideas/_inbox/config-gin-mode-validation-is-unreachable-in-the-binary-beca.md, harness/ideas/_inbox/a-second-sigint-or-sigterm-during-the-8-s-shutdown-drain-is-.md]
---
# Review — cmd/api: serve through an http.Server that drains on SIGINT/SIGTERM, and harden the wiring file once

**Plan:** `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md`
**Branch/worktree:** `harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so` / `.worktrees/main-go-installs-a-signal-handler-with-no-server-shutdown-so`
**Diff:** `git diff origin/main...harness/2026-09-24-high-main-go-installs-a-signal-handler-with-no-server-shutdown-so --stat`

## Plan vs idea
The idea's central ask is delivered and proven live. `cmd/api` stops on a signal again: SIGTERM
and SIGINT each produce `shutting down…` and then `shutdown complete`, with exit 0 inside the
grace window and no SIGKILL. It is served through an `http.Server` that treats
`http.ErrServerClosed` as a normal exit, with a named `ShutdownGrace` constant. No
"ignores SIGTERM" caveat remains in the docs (`grep` of backend/, .github/, AGENTS.md, README,
CODEMAP and .agents/ found none).

Two clauses of the idea's Expected output bullet 2 were not carried into the plan:
- "**returns** rather than `log.Fatalf`". This holds on the fast path only. On a grace overrun,
  `main.go:184` `log.Fatalf`s, exits 1 and skips the deferred Closes.
- "the pet cron's cancellation is joined rather than raced". Neither `pet.RunHourly` nor
  `notify.RunWorker` is awaited.

Both are filed as one medium bug. The idea's core regression, a process that could not be
stopped, is fixed, so this is not a `fail`.

The folded findings are delivered as planned: `GIN_MODE` defaulting to release,
`SetTrustedProxies(nil)`, embedded tzdata plus the boot check, the header comment, the single
`config:` prefix, and the `.env.example` application section. The one exception is that the
`GIN_MODE` validation never runs in the real binary (next section).

## Code vs plan
- **Task 1 (config GIN_MODE)**: followed exactly. The tests pass. However, the validation is
  unreachable in the built binary: gin v1.12's own `init()` (`mode.go:52`) reads `GIN_MODE`
  and panics before `main`. **Judgement on the executor's claim:** it is accurate and
  honestly disclosed. Verification item 6 ("→ refuses") holds only literally: exit 2, a panic
  trace and no `listening` line. Task 4 Step 3's "refuse with the `GIN_MODE must be` message"
  does **not** hold. The executor's statement that a fix is "impossible" is overstated. The bug
  describes a possible leaf-package init check; that is the reviewer's inference and untried.
  This is not a gate failure, because it was disclosed rather than claimed. It is filed as a
  low bug for the misleading comments and the panic UX.
- **Task 2 (server.go)**: code followed exactly. The test file was also followed exactly, but
  the drain test is dishonest (see Quality). **Blocker.**
- **Task 3 (main.go)**: followed exactly. `ctx` is reused, `defer stop()` is kept, the
  `store.Migrate` deadline is deliberately left out, and the notify wiring is untouched.
- **Task 4 (.env.example, Makefile, CODEMAP)**: followed exactly. The new documented
  `set -a; . ./.env` loop leaks `TEST_DATABASE_URL`/`TEST_REDIS_URL` (which point at the dev DB)
  into the shell. Filed as medium.
- No deviations from the file structure. `git merge-tree` against `origin/main` (2 commits
  ahead) is clean.

Re-run evidence (reviewer, 2026-09-24; worktree `backend/`; compose project `rev-shutdown` on
55442/56392; app on 18091; the scratch `.env` omits TEST_*):
```
go build ./... && go vet ./... && test -z "$(gofmt -l ./cmd ./internal/config)"   -> fmt-ok
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s
  -> 11/11 packages ok (cmd/api, airouter, auth, config, google, health, notify, onboarding, pet, quests, store)
go test ./cmd/api/... -count=1 -race -timeout 120s -v -> 3 PASS, ok, no DATA RACE
gh run 35958446539 @ 96d190d (= HEAD): frontend, harness-tooling, backend-unit, backend-integration all success

signal proof, built binary:
  SIGTERM: healthz {"postgres":"ok","redis":"ok","status":"ok"}; exit=0 after 0.00s; GIN-debug lines 0;
           "listening on [::]:18091 (GIN_MODE=release)", "shutting down: draining … up to 8s", "shutdown complete"
  SIGINT:  identical, exit=0 after 0.01s
in-flight drain (POST /api/v1/auth/google, body streamed over 2 s via `curl -T -`, kill at +0.5 s):
  SIGTERM: exit=0 after 1.62s; inflight code=400 {"error":"invalid_request"}; both shutdown lines
  SIGINT:  exit=0 after 1.57s; inflight code=400
grace overrun (body streamed over 12 s; SIGTERM, then SIGINT 1 s later):
  still alive after second signal; "server: shutdown: context deadline exceeded"; exit=1 after 8.02s
env -u DATABASE_URL -u REDIS_URL → "config: DATABASE_URL is required" (single prefix), exit=1
GIN_MODE=verbose → "panic: gin mode unknown: verbose (available mode: debug release test)", exit=2
GIN_MODE=debug   → 12 [GIN-debug] lines, SIGTERM exit=0
cleanup: pkill; `docker compose -p rev-shutdown down -v`; scratch .env and /tmp files removed;
         pgrep and `docker ps -a --filter name=rev-shutdown` empty; worktree `git status` clean
```
Note: the plan's Verification step 5 idiom (`curl … -d '{}' &` and then an immediate kill) can
race ahead of the connection. The reviewer's first attempt with `--data-binary @-` got `000`
because curl buffers stdin before it connects. A streamed body (`-T -`) is what reliably puts a
request in flight.

## Quality
- **Test honesty (blocker).** `TestServeStopsOnContextCancelAndDrainsInFlightRequests` passes
  when `serve`'s whole shutdown block is replaced by `_ = ln.Close()`. The reviewer reproduced
  this in a scratch module: 3/3 PASS under `-race`, and the-validator found the same
  independently. The test awaits `done` and `code` separately and never asserts that the handler
  finished before `serve` returned, so CI would stay green if `Shutdown` were deleted. This is
  the plan's only automated guard for its own regression.
- **Error paths.** The `srv.Shutdown`-timeout branch (`server.go:52-55`) is untested because
  `ShutdownGrace` is a const, and in `main` that branch leads to `log.Fatalf`. A second signal
  during the drain is swallowed (`stop` is only deferred). Filed as medium and low.
- **Design.** Splitting out `newServer`/`serve` is clean and testable. Keeping `WriteTimeout`
  unset is well argued and pinned by a test. `SetTrustedProxies(nil)` is correct while nothing
  reads `ClientIP()`. The tzdata boot check is belt and braces but harmless.
- **Conventions.** It matches the codebase and is gofmt-clean in touched paths.
  `internal/quests/{handler_test.go,repo.go}` are unformatted, but they are not in this diff
  (pre-existing; owned by the gofmt CI plan).
- **Boundaries.** No cross-package access was added, and `config` stays free of the gin import.
- **CODEMAP.** The new `cmd/api` bullet is accurate for what exists. No `config` bullet exists,
  so the "add GIN_MODE to its env list" step correctly did not apply. No correction was
  committed. The GIN_MODE-panic nuance is carried by the low bug, whose fix updates CODEMAP.
- **Documentation.** The `.env.example` credentials and ports match `docker-compose.yml`.

## Bugs filed
- **BLOCKER** (high, `blocks` this plan): `harness/ideas/_inbox/cmd-api-drain-test-passes-with-srv-shutdown-deleted-so-the-s.md`. The drain test passes with `srv.Shutdown` removed.
- medium: `harness/ideas/_inbox/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md`. On grace overrun, `log.Fatalf` exits 1 and skips the deferred Closes; the workers are not joined (both are idea Expected-output items).
- medium: `harness/ideas/_inbox/the-documented-set-a-env-export-also-exports-test-database-u.md`. The documented `set -a; . ./.env` exports TEST_* (the dev DB), so a later `make test` drops its tables.
- low: `harness/ideas/_inbox/config-gin-mode-validation-is-unreachable-in-the-binary-beca.md`. gin's `init` panics before `config.Load`, and the comments overclaim.
- low: `harness/ideas/_inbox/a-second-sigint-or-sigterm-during-the-8-s-shutdown-drain-is-.md`. A second signal during the 8 s drain is ignored.

## Verdict
**pass-with-bugs, merge blocked.** The idea's core regression is fixed and verified live: the API
drains and exits 0 on SIGTERM/SIGINT. The build, the full suite, `-race` and CI are all green, and
the executor's evidence reproduces, so there is no gate failure. The branch must not merge until the
blocker's test-only amend lands on it. `python3 tools/harness/cli.py blockers --plan <plan>`
currently exits 1.
