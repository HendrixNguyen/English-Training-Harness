---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md
---
# fakeProgressRepo.Upsert's monotonic max is not covered by any unit test

## Why
`daily_progress.minutes_spent` never lowering is one of the two guarantees the plan added to quests,
and `fakeProgressRepo.Upsert` mirrors it with `row.minutes = max(row.minutes, minutes)`. Nothing
tests that line. Deleting it - `row.minutes = minutes` - leaves the **entire `internal/quests` unit
suite green** (verified in review). The behaviour is pinned only by
`TestIntegrationDailyAndProgressAgainstRealServices`, which is gated on `TEST_DATABASE_URL` and so
skips on every developer machine without a stack up.

The executor found and reported this honestly, with the right diagnosis: the named test
(`TestTheHookFiresFromTheDurableFlagNotTheCounterEdge`) re-adds the *same* 1800s twice, so the "new"
minutes value equals the old one and no decrease ever occurs. CI does run `backend-integration`, so
the branch is protected - but a test double whose fidelity to the SQL is itself untested will drift,
and the unit suite is the loop developers actually run.

## Expected output
The monotonic upsert is covered by the unit suite as well as the integration suite:

- a service test reaches 1800s, loses the counter (fake `Total` restarts from a *smaller* value),
  reports a short session, and asserts the fake row's `minutes` did not drop - the value must
  actually decrease for the assertion to have teeth;
- deleting `max(...)` from `fakeProgressRepo.Upsert` turns that test red;
- optionally the same shape for `MarkTargetMet`'s "never returns to FALSE".

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`), *Verification* mutation table, row "`daily_progress`
  monotonic"; the execution summary reports the row as not failing its named test.
- `backend/internal/quests/fakes_test.go:118-128` - `row.minutes = max(row.minutes, minutes)`.
- Verified in review: replacing it with `row.minutes = minutes` leaves
  `go test ./internal/quests/ -count=1` **ok** (whole package). The equivalent SQL mutation
  (`minutes_spent = EXCLUDED.minutes_spent` in `backend/internal/quests/repo.go:106`) does fail
  `TestIntegrationDailyAndProgressAgainstRealServices` - `daily_progress = (1, true) ... want
  (41, true)`.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Planned today in `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md` (Also planned here).**

*Confirmed (read on this branch).* `backend/internal/quests/fakes_test.go` `Upsert`: `row.minutes = max(row.minutes, minutes)`; `TestTheHookFiresFromTheDurableFlagNotTheCounterEdge` re-adds the same 1800 s after the counter loss, so the value never decreases and the `max` is never exercised; only the `TEST_DATABASE_URL`-gated integration test pins the SQL.

*Fix.* A service test that reaches 1800 s (row 30 min), loses the counter, records 60 s (total 60 → `Upsert(…, 1)`) and asserts the fake row is still 30 minutes — replacing `max(...)` with the new value turns it red. Low: the branch is protected by CI's integration job; this is the developer loop.
