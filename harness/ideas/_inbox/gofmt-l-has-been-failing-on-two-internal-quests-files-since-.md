---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Duplicate of the selected ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md, planned today in harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md: same CI step, and its gofmt -w covers both quests files."
---
# gofmt -l has been failing on two internal/quests files since before this branch

## Why
`gofmt -l .` in `backend/` reports `internal/quests/handler_test.go` and `internal/quests/repo.go`.
Both are unformatted on `main` as well (verified by running `gofmt` over `git show main:...` copies),
so this is not the pet day-judgement branch's doing - but nothing in CI runs `gofmt`, so it has
gone unnoticed and will keep spreading. `repo.go`'s hit is a single comment-alignment on
`Exercise.TaskType`; `handler_test.go` is the same class.

Every executor's *Definition of done* claims a clean `gofmt` for the files it touched, which means
each one has to stop and re-verify that these two pre-existing failures are not theirs. That check
has now been paid for at least twice.

## Expected output
`gofmt -l` is clean and stays clean:

- both files are reformatted (`gofmt -w internal/quests/handler_test.go internal/quests/repo.go`);
- `.github/workflows/ci.yml`'s `backend-unit` job gains a step that fails when `gofmt -l .` prints
  anything, so the next drift is caught by the outer loop rather than by each executor's manual
  comparison against `main`.

## Evidence
- Observed while reviewing `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`: `cd backend && gofmt -l .` prints
  `internal/quests/handler_test.go` and `internal/quests/repo.go`.
- Pre-existing: `gofmt -l` over `git show main:backend/internal/quests/repo.go` and
  `.../handler_test.go` reports both, so neither is this branch's change. That branch's execution
  summary flags them for the same reason.
- `gofmt -d internal/quests/repo.go` - one hunk, the `TaskType string // vocabulary | reading |
  practice` comment alignment.
- `.github/workflows/ci.yml` - `backend-unit` runs build/vet/test, no format check.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Reject — duplicate.** The selected `ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md` (medium) asks for the same CI step and the same `gofmt -w`, and is planned today in `harness/plans/2026-09-24-ci-never-runs-gofmt-so-three-files-on-main-are-unformatted-a.md`. `gofmt -l .` on this branch lists exactly the two files named here (`internal/quests/handler_test.go`, `internal/quests/repo.go`; `internal/google/fakes_test.go` has since been formatted), and that plan formats whatever `gofmt -l .` prints at execution time.
