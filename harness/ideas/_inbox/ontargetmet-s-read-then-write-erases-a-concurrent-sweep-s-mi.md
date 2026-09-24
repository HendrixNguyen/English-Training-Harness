---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# OnTargetMet's read-then-write erases a concurrent sweep's miss penalty

## Why
The plan that introduced the verdict markers made the *once-per-day* guarantee a SQL predicate, but
left the *health value* on a read-then-write path. `Service.OnTargetMet` reads the whole state
through `Ensure`, computes `health + 20` in Go, and writes it back as an absolute
`health_points = $2`. `PgRepo.PenaliseMiss` does its arithmetic in SQL on the live row. When the
hourly sweep lands between the read and the write, the sweep's -30 is **silently erased**: the
user keeps a full 100 health on a day the plant should have decayed, and the pet's whole retention
signal (health decays when you miss) quietly stops being true for that day.

The plan's *Notes and open questions* documents this race but states the wrong outcomes:
"miss-then-met gives 90/1, met-then-miss gives 70/0 - the difference is one streak day and 20
health". The real third outcome is **100/1** - the entire penalty, not 20 health. A maintainer
reading that note will believe the damage is bounded when it is not.

Proven live against Postgres 16 during the review of the plan below (health 100 -> PenaliseMiss ->
70 -> SaveTargetMet with the stale pre-image -> **100**).

## Expected output
The success write stops depending on a pre-image it read over the network:

- `saveTargetMetSQL` computes health in SQL the way `penaliseMissSQL` already does -
  `health_points = LEAST(100, COALESCE(health_points, 100) + $n)`, `current_streak =
  COALESCE(current_streak, 0) + 1`, `stage` from the same `CASE`, with the existing
  `WHERE ... last_target_met_date IS NULL OR last_target_met_date < $d` predicate unchanged.
  Pre-image and write become one statement, matching `PenaliseMiss`;
- `ApplyTargetMet` stays as the Go reference the fake and an integration test hold the SQL to,
  exactly as `ApplyMiss` is held to `penaliseMissSQL` today;
- `TestIntegrationVerdictWritesAreConditional` gains a section that interleaves the two writers -
  read the pre-image, run `PenaliseMiss`, then `SaveTargetMet` from the stale image - and asserts
  the health is 90, not 100;
- the plan's / CODEMAP's residual-race note is corrected, or deleted once the race is gone.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`), *Notes and open questions* -> "Residual race, documented".
- `backend/internal/pet/service.go:57-64` - `OnTargetMet`: `Ensure` (read) then
  `SaveTargetMet(ctx, userID, ApplyTargetMet(st, ...))` (write of an absolute value).
- `backend/internal/pet/repo.go:75-79` - `saveTargetMetSQL` sets `health_points = $2`,
  `current_streak = $4` from the caller's state; contrast `penaliseMissSQL`
  (`backend/internal/pet/repo.go:85-92`), which reads and writes in one statement.
- Reproduced against real Postgres in review: pre-image read at health 100 -> `PenaliseMiss`
  for 2026-09-22 applied=true -> health 70 -> `SaveTargetMet` for 2026-09-23 from the stale
  pre-image applied=true -> **health 100, streak 1**. The -30 is gone.
- Pre-existing in class (`main`'s `OnTargetMet` also read-then-wrote via `Save`), so this is not a
  regression introduced by the branch - but the branch's own design decision 2 ("one mechanism, in
  the database") is only half applied.
