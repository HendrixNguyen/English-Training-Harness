---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# MarkTargetMet ignores RowsAffected so a missing row silently succeeds

## Why
`PgRepo.MarkTargetMet` runs `UPDATE daily_progress SET is_target_met = TRUE WHERE user_id = $1 AND
date = $2::date` and returns nil whatever the update touched. If the row is not there, the call
reports success and the flag is never set. Every sibling verdict writer in this change deliberately
reports what it did - `SaveTargetMet`, `PenaliseMiss` and `MarkJudged` all return
`tag.RowsAffected() == 1`, and `PgRepo.Save` returns `ErrNoPet` on zero - so this is the one write
in the new design that cannot tell "done" from "nothing was there".

It is unreachable today: `Upsert` creates the row on the same call, a few lines earlier. But the
consequence if it ever becomes reachable is a day that silently re-fires the pet hook on every
subsequent progress call for the rest of the day, with a log line that says the flag was written.

## Expected output
The write reports what it did, like its three siblings:

- `MarkTargetMet` returns `ErrNoProgressRow` (or a bool) when `RowsAffected() == 0`, so
  `RecordProgress`'s existing `else if err := s.progress.MarkTargetMet(...)` branch logs a real
  failure instead of a silent one;
- the fake mirrors it (today `fakeProgressRepo.MarkTargetMet` creates the row if absent, which also
  hides the case);
- a unit test calls `MarkTargetMet` for a date with no row and asserts the error.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`).
- `backend/internal/quests/repo.go:190-195` - `MarkTargetMet` discards the `pgconn.CommandTag`.
- Contrast `backend/internal/pet/repo.go:156-181` (`SaveTargetMet`, `PenaliseMiss`, `MarkJudged`
  all return `tag.RowsAffected() == 1`) and `backend/internal/pet/repo.go:145-154` (`Save` returns
  `ErrNoPet` on zero).
- `backend/internal/quests/fakes_test.go:130-140` - the fake creates the row, so the unit suite
  cannot see the case either.
