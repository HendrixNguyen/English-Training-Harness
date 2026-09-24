---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# PgRepo.Save resets last_target_met_date to NULL on the revive path

## Why
`saveSQL` protects `judged_through` with `GREATEST(judged_through, $8::date)` but writes
`last_target_met_date = $7::date` unconditionally from the caller's in-memory state. `Save` is used
only by `Service.Revive`, which builds that state from a pre-image read at the top of the call. If
an `OnTargetMet` lands between the read and the `Save` - a learner finishing a quest while their
client polls `POST /pet/revive` - the revive write rolls `last_target_met_date` back to its stale
value, usually NULL. The pet's own once-per-day marker for that day is then gone.

The blast radius today is small: quests will not re-fire the hook (its `daily_progress` flag is
already TRUE), and the sweep still spares the day via `judged_through`, which `ApplyRevive` sets.
But the marker is the pet package's stated source of truth for "this day's +20 was applied", and a
write path that can erase it is a trap for the next change that relies on it.

It is also invisible to the suite: `TestIntegrationVerdictWritesAreConditional` section 5 passes a
`State` with a nil `LastTargetMetDate` into `Save` - the exact wipe - and asserts only on
`judged_through`.

## Expected output
No write path can move `last_target_met_date` backwards:

- `saveSQL` protects it the same way it protects `judged_through` -
  `last_target_met_date = GREATEST(last_target_met_date, $7::date)`, which in Postgres ignores NULL
  on either side, so a revive carrying no marker leaves the stored one alone;
- `fakeRepo.Save` mirrors that (it already mirrors the `judged_through` GREATEST at
  `backend/internal/pet/fakes_test.go:77-81`);
- `TestIntegrationVerdictWritesAreConditional` section 5 gains one assertion: after the `Save` that
  passes a nil marker, `last_target_met_date` is still the value section 1 left there.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`).
- `backend/internal/pet/repo.go:69-73` - `saveSQL`: `judged_through = GREATEST(judged_through,
  $8::date)` but `last_target_met_date = $7::date`.
- `backend/internal/pet/service.go:76-118` - `Revive`: `Ensure` (read) ... `Save` (write), the only
  caller of `Save`.
- `backend/internal/pet/fakes_test.go:65-84` - the fake faithfully mirrors the unconditional write,
  so the unit suite cannot see it either.
- `backend/internal/pet/integration_test.go`, `TestIntegrationVerdictWritesAreConditional` section
  5 - passes `State{HealthPoints: 50, Stage: StageSprout, JudgedThrough: &earlier}` (nil marker)
  and asserts only `*st.JudgedThrough`.
