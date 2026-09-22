---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---

# The progress rejection tests assert only the error and the fakes error paths are dead

## Why
Two tests are named for a rejection but assert only the returned error, so they pass while the
rejected request has already written to Redis and Postgres:

- `TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap` (`service_test.go:203-211`) — sets
  `markErr` and checks `errors.Is(err, ErrExerciseNotFound)`. Nothing looks at `h.log.calls`, which
  at that point contains `INCRBY`, `EXPIRE` and `UPSERT daily_progress`.
- `TestProgressHandlerReturns404ForAnUnknownExercise` (`handler_test.go:151-163`) — same, one layer up.

The suite proves it knows how to make this assertion: `TestRecordProgressRejectsNonPositiveSeconds`
(`service_test.go:213-225`) ends with `if len(h.log.calls) != 0 { t.Errorf("a rejected call touched
Redis/Postgres: ...") }`. The two exercise-rejection tests omit exactly that line, which is why the
write-before-validate defect shipped green.

Alongside it, the fakes carry error fields that no test ever sets, so two failure paths through the
§5.2 sequence have no coverage at all:
- `fakeCounter.err` (`fakes_test.go:22,32-34`) — Redis down; never set.
- `fakeProgressRepo.err` (`fakes_test.go:96,107-109`) — the `daily_progress` upsert failing after the
  INCRBY committed; never set. This is the path in which the once-only hook is lost for good.

`fakePet.hookErr` and `fakePet.stateErr` *are* exercised, so the omission looks like an oversight
rather than a policy.

Everything else in the suite is honest: the call-log ordering test, the 80->100 / 4->5 proof that
`State` is read after the hook, the `content_json` mapping test, the 400 table including the
pre-§6.2 `seconds` name, and the live `TestIntegrationDailyAndProgressAgainstRealServices` (which the
reviewer re-ran green against real services) all assert something real.

## Expected output
- Both exercise-rejection tests assert that nothing was written — `len(h.log.calls) == 0` at the
  service level, and the same on the handler test's harness — so they fail until the ordering bug is
  fixed and fail again if it regresses.
- A test per unused error field: `counter.err` set -> `RecordProgress` returns the error and neither
  Postgres fake was called; `progress.err` set on the crossing call -> the error surfaces, the counter
  state is inspected, and a replay after clearing the error fires `OnTargetMet` exactly once.
- The two currently-unused fields stop being dead code either way.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 4 — "shared test fakes with an ordering call log"; Task 5).
- `backend/internal/quests/service_test.go:203-211` vs `:213-225`.
- `backend/internal/quests/handler_test.go:151-163`.
- `backend/internal/quests/fakes_test.go:22,96`.
