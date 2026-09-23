---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# CI never runs go test -race so background goroutine races go undetected

## Why
`backend-unit` runs `go test ./... -count=1` with no `-race`. The backend now starts two
long-lived background goroutines in the API process (`pet.RunHourly`, `notify.RunWorker`), and
concurrency bugs in that shape are exactly what the race detector exists to find and exactly
what a non-race test run will never surface.

This is not theoretical for this repo: the notify executor ran `-race` locally, it flagged a
real read/write race between `RunWorker`'s goroutine and the test goroutine, and the executor
fixed it — while noting CI would have stayed green either way. The next such race will not be
caught, because nothing in the pipeline looks.

(For the record, the race the executor found was in the test fake, not in `Service`. I
re-ran `go test ./internal/notify/... -race -count=1` on the branch: it passes. `Service`'s
fields are set once in `NewService` and only read in `Tick`, and `RunWorker` is the only
goroutine calling `Tick`, so the production worker has no race today. The gap is the CI
pipeline, not this slice's code.)

## Expected output
`backend-unit` (or a new `backend-race` job) runs the unit suite under the race detector —
`go test ./... -count=1 -race` — and is red when a race is reported. If the whole suite under
`-race` is too slow for every push, scope it to the packages with goroutines
(`./internal/notify/... ./internal/pet/...`) or run it on `main` plus a nightly schedule while
keeping the fast non-race run on every branch push. Either way the result is that a data race
introduced on a `harness/*` branch fails a check before merge instead of after.

## Evidence
- `.github/workflows/ci.yml:43-44` — `- name: Test without services` / `run: go test ./... -count=1`. No `-race` anywhere in the file (`grep -n race .github/workflows/ci.yml` → no match).
- `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md` → *Execution summary* → *Deviations* item 3: "`-race` did complain (a real read/write race …); CI's `backend-unit` job runs `go test ./... -count=1` without `-race` so this would not have failed CI".
- `backend/cmd/api/main.go:93` (`go pet.RunHourly(ctx, petSvc)`) and `main.go:118` (`go notify.RunWorker(ctx, notifySvc, notify.PollInterval)`) — two background goroutines now in the production binary.
- Reviewer re-ran `env -u DATABASE_URL -u REDIS_URL go test ./internal/notify/... -race -count=1 -timeout 180s` on the branch → `ok … 1.957s`.
- Related CI-quality findings already in the inbox: `ci-never-runs-on-harness-branches-so-it-gates-nothing-before.md`, `ci-jobs-have-no-timeout-minutes-and-the-harness-job-floats-p.md`, `codemap-ci-section-overstates-the-unit-job-and-prescribes-a-.md`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed: `grep -n race .github/workflows/ci.yml` → no match, and the production binary will run two long-lived goroutines once notify merges (pet cron + notify worker). Developer-workflow bug: the one class of defect those goroutines can have is invisible to CI. Plan together with `ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md` — same job in the same file; one branch, one merge. Scope `-race` to the packages with goroutines if the full suite is slow.
