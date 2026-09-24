---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Durable-flag and live-row success tests leave updated_at and the Daily error path unasserted

## Why
The plan's tests are honest: I re-ran every mutation in its table, and each one turns its named
test red (details in the review). They still leave three cheap gaps that a later change could
slip through.

1. `saveTargetMetSQL` sets `updated_at = $5`. The sweep's first-contact rule reads
   `LocalDate(updated_at)`, so this column matters. Section 4b's mirror loop
   (`integration_test.go:242-260`) compares health, streak and stage, and checks only that
   `LastPracticedAt != nil`. Two mutations pass the whole suite (`make test-integration`, verified in review):
   `updated_at = $5` changed to `updated_at = updated_at`, and `last_practiced_at = $5` changed to
   `COALESCE(last_practiced_at, $5)` (stale whenever the row already has one). The loop also never checks `LastTargetMetDate == d` for the four mirror cases.
2. `Service.Daily`'s new `TargetMet` read (`quests/service.go:237`) has an error branch that no
   unit test drives. `fakeProgressRepo.err` already exists, so a two-line test would pin
   "a flag-read failure fails `Daily`, and does not report unmet".
3. `TestFakeSaveKeepsAMarkerItWasNotGiven` discards the second `Save`'s error
   (`pet/service_test.go:616`, `_ = h.repo.Save(…)`). A fake that starts refusing the write would
   still pass the assertion that follows.

## Expected output
- The mirror loop asserts `got.UpdatedAt.Equal(now)`, `got.LastPracticedAt.Equal(now)` and
  `*got.LastTargetMetDate == d` (watch the timestamp precision round-trip, as section 1 does).
- A `TestDailyFailsWhenTheFlagReadFails` unit test.
- The second `Save` in `TestFakeSaveKeepsAMarkerItWasNotGiven` checks its error.

## Evidence
- Plan under review: `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md` (branch `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`).
- `backend/internal/pet/integration_test.go:242-260`: the mirror loop's comparison.
- `backend/internal/quests/service.go:237-240`: the `TargetMet` error branch.
- `backend/internal/pet/service_test.go:616`.
