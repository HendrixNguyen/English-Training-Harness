---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md
---
# TestTwoConcurrentSweepsPenaliseOnce misses its defect in 1 run in 10

## Why
`TestTwoConcurrentSweepsPenaliseOnce` is the test the plan's mutation table names for two rows
("Sweep idempotent / concurrent" and "Count is honest"), and for the first row it is the *only*
unit test that catches the mutation. But it launches two bare goroutines with no synchronisation
between them, so nothing makes the two sweeps actually overlap. When one goroutine finishes all 20
pets before the other calls `SweepCandidates`, the second sweep's own fresh-state pre-check
(`JudgedThrough >= judged -> continue`) skips every pet without ever calling `PenaliseMiss` - and
the mutation goes undetected.

Measured in review: with the `applied` guard around `penalised++` removed, the test failed in
**27 of 30** separate runs. Under a real defect it passes about once in ten. It has no false
positives (300 unmutated runs and 100 more under `-race` were clean), so it will not flake CI - but
a guard that misses one run in ten is not the guard the mutation table claims it is, and the same
scheduling luck weakens the `PenaliseMiss`-unconditional row it is also the named backstop for.

## Expected output
The overlap is forced, not hoped for:

- `fakeRepo` gains a gate the test installs - e.g. a `beforeWrite func(userID string)` hook called
  inside `PenaliseMiss` while the mutex is *not* held - and the test uses a `sync.WaitGroup` or an
  unbuffered channel so neither sweep may write until both have read their candidates;
- the assertions stay as they are (sum == 20, every pet at 70);
- the test is verified to fail in 30 of 30 runs under both mutations (the unconditional
  `penalised++`, and the unconditional `PenaliseMiss`), and to pass in 300 of 300 unmutated.

`TestIntegrationVerdictWritesAreConditional` section 2 (8 concurrent real `PenaliseMiss` calls)
stays as the SQL-level proof; this is about the unit-level guard being deterministic.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`), *Verification* mutation table, rows "Sweep idempotent /
  concurrent" and "Count is honest"; the execution summary reports 7 of 8 and attributes it to
  goroutine timing.
- `backend/internal/pet/service_test.go:369-401` - two goroutines, `sync.WaitGroup`, no barrier.
- `backend/internal/pet/service.go:160-162` - the pre-check that lets a late sweep skip every pet
  without reaching the repo.
- Measured in review: removing the `applied` guard around `penalised++`
  (`backend/internal/pet/service.go:193-195`) was detected in **27/30** single runs; unmutated
  `-count=300` and `-count=100 -race` both `ok`.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Planned today in `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md` (Also planned here).**

*Confirmed (read on this branch).* `backend/internal/pet/service_test.go:369-401` launches two bare goroutines with a `WaitGroup` for completion only; nothing forces both sweeps to hold their candidate lists before either writes, so a late sweep's fresh `JudgedThrough >= judged` pre-check skips every pet and the mutation goes undetected (reviewer: 27/30 detections).

*Fix.* `fakeRepo.afterCandidates func()` — called once per sweep after `SweepCandidates` builds its list, outside the mutex — and the test installs a two-party `sync.WaitGroup` barrier there, so both sweeps carry the same stale list into the 20 `PenaliseMiss` calls. Assertions unchanged (sum 20, every pet at 70); the plan verifies 30/30 detections for both mutations and 300/300 clean runs. Low: the SQL-level proof (`TestIntegrationVerdictWritesAreConditional` section 2) already exists.
