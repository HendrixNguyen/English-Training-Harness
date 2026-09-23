---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
---
# Sweep reads updated_at then writes unconditionally so two sweeps in one local hour double the miss penalty

## Why
`Service.Sweep` makes the §8 miss penalty idempotent with a **read-then-write** check, not an
atomic one:

```go
// backend/internal/pet/service.go:152-165
midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
if !c.State.UpdatedAt.Before(midnight) {
        continue // already handled this local day
}
...
if err := s.repo.Save(ctx, c.UserID, ApplyMiss(c.State, now)); err != nil {
```

`c.State.UpdatedAt` came from the `SweepCandidates` SELECT; the `Save` that follows is
`UPDATE pet_states SET … WHERE user_id = $1` (`backend/internal/pet/repo.go:51-54`) with no
predicate on `updated_at`. Nothing prevents a second reader from seeing the same pre-midnight
`updated_at` and applying `ApplyMiss` to the same pre-image. Two overlapping sweeps therefore take
a user from 100 to 40 in one night instead of 70 — and from 30 to 0 (`wilted`) instead of 0… i.e.
a plant can wilt a full day early.

Today the cron is in-process and single-instance (`go pet.RunHourly(ctx, petSvc)` in
`backend/cmd/api/main.go`), so the window only opens when the app runs more than one replica —
which is exactly what spec §9's Railway deployment makes a one-click change — or when an operator
triggers a sweep manually alongside the timer. The plan's *Notes* claim the `updated_at` guard
makes a re-run idempotent; that is true for a *sequential* re-run (proved by
`TestSweepIsIdempotentWithinTheSameLocalDay`) and untrue for a concurrent one, and nothing in the
code or the CODEMAP records the single-instance assumption.

The same read-then-write shape also loses a concurrent `OnTargetMet`: a user who crosses 1800 s at
00:00:30 while the sweep is mid-flight can have the `+20` overwritten by the sweep's `-30` computed
from the stale pre-image.

## Expected output
The penalty is applied by a conditional write, so a second sweeper in the same local day is a
no-op no matter how many processes are running:

- `Repo.Save` (or a new `Repo.ApplyMiss`) takes the local midnight and writes
  `UPDATE pet_states SET health_points = …, stage = …, current_streak = 0, updated_at = $n
   WHERE user_id = $1 AND updated_at < $midnight`, computing the new health in SQL
  (`GREATEST(0, health_points - 30)`) so the pre-image is read and written in one statement;
- `Sweep` counts a pet as penalised only when `RowsAffected() == 1`, so the returned count stays
  honest;
- a test starts two `Sweep` calls concurrently against the same fake/real row and asserts the
  health fell by exactly 30 and the count summed to 1;
- CODEMAP's pet bullet says the sweep is safe to run from more than one process, instead of the
  current "a re-run in the same hour is idempotent", which only holds sequentially.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 5,
  `Service.Sweep`; *Notes* — "`pet_states.updated_at` is set by Go, not `now()`, so the idempotency
  guard and the tests share one clock").
- `backend/internal/pet/service.go:152-165` — the guard and the unconditional `Save`.
- `backend/internal/pet/repo.go:51-54` — `saveSQL`, `WHERE user_id = $1` only.
- `backend/internal/pet/service_test.go:255-269` — `TestSweepIsIdempotentWithinTheSameLocalDay`
  runs the two sweeps sequentially, so it passes and proves nothing about concurrency.
- Backend spec §8 — `Health = Max(0, Health - 30)` is once per missed day, not once per sweeper.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Only opens with more than one replica or a manual sweep; the app is single-instance today. The pet "durable day judgement" plan should make the miss a conditional `UPDATE … WHERE updated_at < $midnight` while it is in `Sweep`, which closes this for free.

**Planned (2026-09-23):** `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — decision 3, Tasks 3 and 6 (`PenaliseMiss` is one conditional `UPDATE`; the count follows `RowsAffected`).
