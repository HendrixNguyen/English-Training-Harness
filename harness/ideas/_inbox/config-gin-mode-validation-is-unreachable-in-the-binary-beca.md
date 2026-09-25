---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# config GIN_MODE validation is unreachable in the binary because gin init panics first, and its comments claim otherwise

## Why
Task 1 added a `GIN_MODE` check to `config.Load`. Its doc comment says the check exists "so
gin.SetMode (which panics on an unknown value) is only ever given a valid one", and the plan's
Task 4 Step 3 expects `GIN_MODE=verbose` to "refuse with the `GIN_MODE must be` message". In the
real binary that message is never seen.

gin v1.12's package `init()` (`mode.go:52`) reads `GIN_MODE` itself and calls `SetMode`, which
panics on an unknown value. Package inits run before `main`, so the process dies with a
goroutine-dump panic (exit 2) before `config.Load` runs. The validation only ever executes in
`config`'s own unit test, which never imports gin. The executor disclosed this honestly. The
remaining problems are that the code comment and the test message
("gin.SetMode would panic") describe a protection that does not exist, and an operator who
fat-fingers `GIN_MODE` on Railway gets a panic trace instead of the one-line `config:` error
every other bad variable produces.

Plan Verification item 6 ("… → refuses") holds only in the weak sense: the process does not
serve. Task 4 Step 3's exact-message claim does not hold.

## Expected output
One of the following, and the plan should say which:
- A clean one-line error with a non-zero exit for an invalid `GIN_MODE`. Inferred, not tried:
  a stdlib-only leaf package under `internal/` that `main` imports and whose `init` validates
  `GIN_MODE` should initialise before gin under Go 1.21+ import-path-ordered init. That would
  need a test that execs the built binary with `GIN_MODE=verbose`.
- Or accept gin's panic, and correct the `config.go` `GinMode` comment, the test's failure
  message, `.env.example` ("an unknown value aborts boot with a gin panic") and CODEMAP, so no
  document claims `config.Load` guards `gin.SetMode`.

## Evidence
- Plan: `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` (Task 1; Task 4 Step 3; Verification 6; Execution summary → *Finding*).
- `backend/internal/config/config.go` (the `GinMode` doc comment and the `switch`);
  `$GOMODCACHE/github.com/gin-gonic/gin@v1.12.0/mode.go:52-75` (`init` → `SetMode` → `panic`).
- Reviewer run, 2026-09-24: `GIN_MODE=verbose /tmp/rev-shutdown-api` printed
  `panic: gin mode unknown: verbose (available mode: debug release test)` and `exit=2`.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.** Confirmed: gin v1.12's package `init` reads `GIN_MODE` and panics before `config.Load` runs, so the `config.go` comment and the test message describe a guard that does not exist. Decision: accept gin's panic and correct the three documents (`config.go` comment, test message, `.env.example`) — a stdlib-only leaf package that inits first is clever but adds a second place `GIN_MODE` is parsed. Plan it with `config-go-still-says-jwt-secret-is-not-in-the-1st-thinking-e.md` as one `config` messages branch.
