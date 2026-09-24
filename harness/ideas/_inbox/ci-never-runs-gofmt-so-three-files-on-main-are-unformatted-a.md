---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md
---
# CI never runs gofmt so three files on main are unformatted and nothing fails

## Why
`gofmt -l .` from `backend/` on `main` (2026-09-23) lists three files —
`internal/google/fakes_test.go`, `internal/quests/handler_test.go`, `internal/quests/repo.go` — and CI is
green. The `backend-unit` job runs `go build`, `go vet` and `go test` but never `gofmt`, so formatting
drift lands silently and every later edit to those files produces a noisy diff. CODEMAP's CI section
already says there is no linter; this is the cheapest check to add and the one that keeps
executor-written Go uniform.

Filed by the evaluator at the orchestrator's request (2026-09-23) while triaging the inbox; `source:
reviewer` because it is a conformance finding, like `google-refresh-token-is-stored-in-plaintext-…`.

## Expected output
- `backend-unit` gains a step before `go vet`: `test -z "$(gofmt -l .)"` (printing the list on failure),
  so an unformatted file is red on the branch that introduces it.
- The three files are formatted in the same change (`gofmt -w`), so the check passes on `main`.
- `harness/CODEMAP.md` → CI → `backend-unit` names the step.
- Plan together with `ci-never-runs-go-test-race-so-background-goroutine-races-go-.md` — both edit
  `.github/workflows/ci.yml`'s `backend-unit` job and the owner merges one branch at a time.

## Evidence
- `cd backend && gofmt -l .` → the three paths above (evaluator, 2026-09-23, `main` at `ca9a0fc`).
- `.github/workflows/ci.yml` → `backend-unit`: `go build ./...`, `go vet ./...`, `go test ./... -count=1`;
  `grep -n gofmt .github/workflows/ci.yml` → no match.
- `harness/CODEMAP.md` → CI: "No linter yet".

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage._

**Select — medium.** Developer-workflow bug with a live symptom (three unformatted files on `main`, green CI). One `ci.yml` step plus `gofmt -w` on three files. Plan together with `ci-never-runs-go-test-race-so-background-goroutine-races-go-.md` (same job).

_Evaluator, 2026-09-24 — daily evaluate._

**Planned today as the head of `harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`.** On this branch `gofmt -l .` lists two files (`internal/quests/handler_test.go`, `internal/quests/repo.go`; `internal/google/fakes_test.go` has been formatted since 2026-09-23) — the plan formats whatever the command prints at execution time and adds the gate before `go vet`. The duplicate inbox filing `gofmt-l-has-been-failing-on-two-internal-quests-files-since-.md` is rejected pointing here.
