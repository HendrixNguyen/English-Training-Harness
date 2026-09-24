---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md
---
# GET quests/daily still reads is_target_met from the volatile counter

## Why
The plan made `daily_progress.is_target_met` the durable record of the day and taught
`RecordProgress` to prefer it over the 48-hour Redis counter
(`targetMet := total >= TargetSeconds || alreadyMet`). `Service.Daily` - the handler behind
`GET /api/v1/quests/daily`, the screen the learner actually looks at - was not changed and still
computes `IsTargetMet: total >= TargetSeconds` from the counter alone.

So after a Redis eviction, restart or `FLUSHDB` mid-day, the two endpoints disagree about the same
day: `POST /quests/progress` answers `is_target_met: true` (durable row), `GET /quests/daily`
answers `false`. The learner's daily screen tells them they have not hit 30 minutes on a day the
pet has already been paid for and the durable row already records - which is the precise symptom
the plan exists to remove, left in place on the one endpoint the user reads most.

The plan's *File structure* table asks for this: the `internal/quests/service.go` row reads
"`IsTargetMet` = counter *or* durable flag". Only `RecordProgress` got it; `Daily` is not mentioned
in any task body, so the gap is the plan's as much as the execution's.

## Expected output
`GET /api/v1/quests/daily` reports the same `is_target_met` as `POST /quests/progress` for the same
local day:

- `ProgressRepo` gains a read - `TargetMet(ctx, userID, localDate string) (bool, error)` over
  `SELECT COALESCE(is_target_met, FALSE) FROM daily_progress WHERE user_id = $1 AND date = $2::date`
  (no row -> false);
- `Service.Daily` sets `IsTargetMet: total >= TargetSeconds || flagged`, mirroring `RecordProgress`;
- `AccumulatedSeconds` keeps reading the counter - it is a live progress bar, and the spec section
  6.2 shape does not change;
- a unit test drives the fake counter to 0 with the fake row flagged and asserts `Daily` still
  answers `is_target_met: true`; the existing `TestIntegrationDailyAndProgressAgainstRealServices`
  gains a `Daily` call after its `DEL` of the counter.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`), *File structure*, `backend/internal/quests/service.go` row.
- `backend/internal/quests/service.go:247` - `IsTargetMet: total >= TargetSeconds` (counter only).
- `backend/internal/quests/service.go:121` - `targetMet := total >= TargetSeconds || alreadyMet`
  (the durable-aware version, `RecordProgress` only).
- `backend/internal/quests/repo.go:56-70` - `ProgressRepo` has no read method, which is why the
  fix needs one.
- Backend spec section 6.2 gives `is_target_met` on both endpoints with no hint that they may differ.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Planned today as the head of `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md`.**

*Is the Why real?* Yes: `GET /quests/daily` is the screen the learner reads, and after a Redis eviction, restart or `FLUSHDB` it tells them the day is unmet while `POST /quests/progress` (and the pet) say it is met — the exact contradiction the 2026-09-23 verdict plan set out to remove, left on the most-read endpoint.

*Root cause (read on this branch).* `backend/internal/quests/service.go` `Daily` builds `IsTargetMet: total >= TargetSeconds` from `Counter.Total` alone, while `RecordProgress` computes `targetMet := total >= TargetSeconds || alreadyMet` from `ProgressRepo.Upsert`'s returned flag. `ProgressRepo` (`repo.go`) has `Upsert` and `MarkTargetMet` and no read, which is why `Daily` had nothing to consult. The parent plan's file-structure row asked for the durable-aware value on the service and only `RecordProgress` received it.

*Fix.* `ProgressRepo.TargetMet(ctx, userID, localDate) (bool, error)` over `SELECT COALESCE(is_target_met, FALSE) FROM daily_progress WHERE user_id = $1 AND date = $2::date` (no row → false); `Daily` sets `IsTargetMet: total >= TargetSeconds || flagged`; `AccumulatedSeconds` keeps the counter (a live progress bar; the §6.2 shape is unchanged). Unit test: reach 1800 s, lose the counter, `Daily` still answers true. Integration: a `Daily` call after the existing `DEL`.

*Also planned in the same branch* — the other 2026-09-23 review follow-ups on the same two packages' verdict writes, each a small conditional-SQL or test fix: `ontargetmet-s-read-then-write-erases-a-concurrent-sweep-s-mi.md`, `pgrepo-save-resets-last-target-met-date-to-null-on-the-reviv.md`, `marktargetmet-ignores-rowsaffected-so-a-missing-row-silently.md`, `fakeprogressrepo-upsert-s-monotonic-max-is-not-covered-by-an.md`, `testtwoconcurrentsweepspenaliseonce-misses-its-defect-in-1-r.md`. Medium: user-visible on the happy path, but only after a Redis loss.
