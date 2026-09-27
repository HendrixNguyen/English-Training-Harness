---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# The drain overrun log line reads server: server: drain grace exceeded

## Why
On a drain overrun the operator's one signal in Railway's deploy log is `server: server: drain grace exceeded; remaining connections were closed: context deadline exceeded`. `ErrDrainTimedOut`'s text already starts with `server:` and `main.go` logs it with another `server: %v`. Cosmetic, but this line is the whole point of decision 1 ("the log line is the operator's signal"), and a doubled prefix reads as a wrapping bug to anyone grepping logs.

## Expected output
The overrun logs once with a single prefix, e.g. `ErrDrainTimedOut = errors.New("drain grace exceeded; remaining connections were closed")` and `main` keeps `log.Printf("server: %v", err)` (or the reverse). CODEMAP/execution-summary quotes of the line updated to match.

## Evidence
- Plan: `harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md` (Task 1 Step 3, Task 2 Step 4; the execution summary's live proof shows the doubled prefix too).
- Branch @ `ae207d3`: `backend/cmd/api/server.go` `var ErrDrainTimedOut = errors.New("server: drain grace exceeded; …")`, `backend/cmd/api/main.go:216` `log.Printf("server: %v", err)`.
- Reviewer live run 2026-09-27 (`go build ./cmd/api`, held POST, single SIGTERM): `2026/09/27 20:31:37 server: server: drain grace exceeded; remaining connections were closed: context deadline exceeded`.
