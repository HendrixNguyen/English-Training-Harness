---
plan: harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/the-worker-join-after-a-drain-overrun-waits-a-fresh-8-s-grac.md, harness/ideas/_inbox/the-drain-overrun-log-line-reads-server-server-drain-grace-e.md]
---
# Review — cmd/api: a drain overrun returns instead of Fatalf-ing, workers are joined, a second signal forces, and requests get a `ReadTimeout`

**Plan:** `harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md`
**Branch/worktree:** `harness/2026-09-25-medium-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r` / `.worktrees/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r`
**Diff:** `git diff main...harness/2026-09-25-medium-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r --stat`

## Plan vs idea
Delivered for all three folded ideas. Head idea: the overrun path no longer `log.Fatalf`s — `serve` returns `ErrDrainTimedOut`, `main` logs it, joins the workers and returns (exit 0), so the deferred `pg.Close()`/`rdb.Close()` run; the grace is injectable and the overrun branch has a unit test. The workers are joined through a `WaitGroup` — but bounded by a fresh `ShutdownGrace`, not the *remaining* grace the idea asked for (bug 1, low). Second-signal idea: `stop()` runs on `ctx.Done()`, a second SIGINT/SIGTERM gets the default disposition. Slow-body idea: `ReadTimeout = 30 s` frees a handler blocked on a trickling body; the `bodylimit.go` comment now tells the truth about `MaxBytesReader`'s close hook under gin.

## Code vs plan
Reviewed at origin head `ae207d3` in a detached scratch worktree (5 files, +201/-17).
- Task 1: followed (sentinel, `grace` parameter, `<-errc` after `Close` so Serve's goroutine is not leaked, overrun test at 100 ms).
- Task 2: followed (`waitWithin` + two tests, `workers.Go`, `switch` in `main`, exactly one `log.Fatalf("server`).
- Task 3: followed (`go func() { <-ctx.Done(); stop() }()`).
- Task 4: followed in production code; the test's assertion deviates, justified — `ReadTimeout` is a read deadline only, so a handler that still writes gets its response out; the test now asserts the handler's `r.Body` read unblocks, which is the actual goroutine-leak fix and fails without `ReadTimeout` (blocks past the 2 s guard). Executor's SIGINT-vs-SIGTERM methodology note is an artefact of its shell; my run below sends the plan's literal SIGTERM-then-SIGINT from Python and it works.
- Task 5: CODEMAP `cmd/api` and `middleware` paragraphs updated and accurate.

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed success harness: CODEMAP — cmd/api shutdown paths, ReadTimeout, bodylimit truth  CI ... 36094035787
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check
go vet ./... ; go test ./... -count=1 -race  -> ok for all 13 packages
$ go test ./cmd/api/ -count=1 -race -v | grep '^--- '
--- PASS: TestServeStopsOnContextCancelAndDrainsInFlightRequests (0.28s)
--- PASS: TestServeReturnsAListenerError (0.00s)
--- PASS: TestServeReturnsErrDrainTimedOutAndClosesTheStragglerWhenTheGraceRunsOut (0.10s)
--- PASS: TestNewServerBoundsReadsButNotWrites (0.00s)
--- PASS: TestASlowBodyIsCutOffAtReadTimeout (0.21s)
--- PASS: TestWaitWithinReturnsTrueWhenTheGroupFinishes (0.00s)
--- PASS: TestWaitWithinReturnsFalseWhenItDoesNot (0.05s)            (7, as the plan expects)
$ grep -n 'log.Fatalf("server' cmd/api/main.go     -> 218 (exactly one)
$ grep -n 'go pet.RunHourly\|go notify.RunWorker' cmd/api/main.go   -> (no output)

Live proof — built ./cmd/api, isolated stack (compose project rv-backend, pg 5443, redis 6393),
POST /api/v1/auth/google with headers + 1 body byte held open, then:
single SIGTERM:
  exit=0 after 8.06s
  ... shutting down: draining in-flight requests for up to 8s (press Ctrl-C again to force)
  ... server: server: drain grace exceeded; remaining connections were closed: context deadline exceeded
  ... shutdown complete
SIGTERM, then SIGINT 1 s later:
  exit=-2 after 1.01s      (killed by SIGINT's default disposition; no "shutdown complete")
```
Integration suite not re-run locally (no store/package change); CI `backend-integration` green on the head.

Also checked, not a bug: whether `ReadTimeout` cancels the request context of a long handler (the 180 s roadmap route, the 60 s Google sync). Scratch repro on go1.27.1 with `ReadTimeout: 300ms` and a 1.5 s handler (POST with a consumed body, and GET): the handler finished normally and the client got 200 — `connReader.startBackgroundRead` clears the read deadline (`net/http/server.go:741`). `WriteTimeout` stays unset.

## Quality
- Correctness: the overrun branch now drains `errc` after `Close`, fixing a goroutine leak the old code had. Sentinel wraps with `%w`, context error with `%v` — `errors.Is` works as intended.
- Join bound (bug 1): after an overrun the join gets another full 8 s, so a stuck worker makes worst-case shutdown 16 s, past the < 10 s budget `server.go:15-18` itself documents; the idea asked for the remaining grace.
- Log text (bug 2): the operator-facing line is `server: server: drain grace exceeded…`.
- Tests are honest: the overrun test asserts both the sentinel and that the straggler got a transport error; the slow-body test fails without the fix. `main`'s `switch` has no unit test (it is `main`), covered by the live proof.
- `bodylimit.go`'s claim checked: `MaxBytesReader` type-asserts an unexported `requestTooLarger` interface; gin's writer embeds `http.ResponseWriter` as an interface field, which does not promote it. Correct.
- Merge note: `git merge-tree origin/main <branch>` reports conflicts in `backend/cmd/api/server.go` (the `newServer` doc comment, which `main`'s `61c3cee` reworded for `airouter.TaskTimeout`) and `harness/CODEMAP.md`. Both are comment/prose conflicts; keep main's wording plus this branch's `ReadTimeout` sentence and field.

## Bugs filed
- `harness/ideas/_inbox/the-worker-join-after-a-drain-overrun-waits-a-fresh-8-s-grac.md` — low.
- `harness/ideas/_inbox/the-drain-overrun-log-line-reads-server-server-drain-grace-e.md` — low.

## Verdict
pass-with-bugs — all three ideas delivered and reproduced live; two low findings. Integration merge must resolve a comment conflict in `backend/cmd/api/server.go` and CODEMAP.
