---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Two RecordProgress error branches are uncovered and the Redis-down test's comment overstates it

## Why
The blocker fixes (`ef4b9fa..4d4b00c`) closed the rejection-test gap that inbox bug
`the-progress-rejection-tests-assert-only-the-error-and-the-f.md` named, but they left two error
branches in `RecordProgress` with no coverage at all, and one of the new tests claims coverage it
does not have.

**1. `counter.Add`'s error branch is never exercised, and the test says it is.**
`TestARedisFailureWritesNothing` (`service_test.go:427-440`) sets `h.counter.err` and its comment
reads *"Counter.Total and Counter.Add both fail"*. Both fakes do honour `f.err`
(`fakes_test.go:31-37,40-45`), but the pre-write ceiling read now calls `Total` first
(`service.go:93-96`), so `Add` is never reached and its `if err != nil { return }`
(`service.go:103-106`) is dead in the suite. Proved by deleting the `if f.err != nil` guard from
`fakeCounter.Add` in a scratch copy of the package: `go test ./internal/quests/...` still passes.
The comment is the problem as much as the gap — it is exactly the "test that passes because it
asserts nothing meaningful" class the previous review flagged, one layer down.

**2. `MarkComplete`'s error branch is now untestable.**
Commit `4f39f85` deleted `fakeQuestRepo.markErr`, which was the only injection point for a
`MarkComplete` failure. The production branch it fed is deliberately retained — `service.go:116-119`
carries the comment *"MarkComplete keeps its own ErrExerciseNotFound for the race where the row
vanished between CheckExercise and here; the handler still maps it to 404"* — so there is code kept
on purpose for a documented race, with no way left to test it. Proved the same way: replacing
`if err := s.quests.MarkComplete(...); err != nil { return ... }` with `_ = s.quests.MarkComplete(...)`
in a scratch copy breaks no test.

Neither is a correctness defect today: `Total` and `Add` both fail before any Postgres write, and a
`MarkComplete` failure already returns cleanly. The cost is that the §5.2 sequence's two remaining
failure modes are unguarded against regression on a branch whose whole amend was about failure
ordering, and one test's comment misdescribes what it proves.

## Expected output
- A test in which `Counter.Total` succeeds and `Counter.Add` fails — the error surfaces, `h.log.calls`
  contains no `UPSERT`/`MARK COMPLETE`, and the fake needs separate `totalErr` / `addErr` fields (or an
  `err` honoured only by `Add`) so the two paths can be driven independently.
- `TestARedisFailureWritesNothing`'s comment corrected to say which call actually fails, or the test
  split into a Total-fails case and an Add-fails case.
- `fakeQuestRepo` regains a `markErr` field, and a test asserts that a `MarkComplete` failure after a
  successful INCRBY + upsert surfaces the error, does not fire `Pet.OnTargetMet`, and leaves the
  counter at its incremented value — i.e. pins the race behaviour the comment promises.
- Each new test fails if its production branch is removed (the check above, run as part of the fix).

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`
  (amended by `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md`, Task 4).
- `backend/internal/quests/service_test.go:427-440` (the overstated comment);
  `backend/internal/quests/service.go:93-96,103-106,116-119`;
  `backend/internal/quests/fakes_test.go:31-37,40-45,87-91` (no `markErr`).
- Reproduction: in a copy of `backend/`, drop `if f.err != nil { return 0, f.err }` from
  `fakeCounter.Add`, then `go test ./internal/quests/... -count=1` -> `ok`. Separately, replace the
  `MarkComplete` error check with `_ = s.quests.MarkComplete(...)`, then `go test ./internal/quests/... -count=1` -> `ok`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Test-only; the code is right. Take `addErr`/`markErr` and the corrected comment in the pet day-judgement plan, which changes `RecordProgress`'s hook derivation and must re-cover these branches anyway.
