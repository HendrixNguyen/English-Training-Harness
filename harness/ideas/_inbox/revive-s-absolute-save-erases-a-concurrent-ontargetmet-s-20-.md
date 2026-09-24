---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Revive's absolute Save erases a concurrent OnTargetMet's +20 and streak

## Why
The parent plan's goal, and this plan's, is that no concurrent write to `pet_states` gets silently
erased. This plan closed the last gap for `SaveTargetMet` and made `Save`'s two verdict dates
monotonic. One lost update in the same family is still open. `Service.Revive` reads a pre-image
(`Ensure`, `service.go:77`), then reads Redis (challenge hash, daily counter), then writes
`ApplyRevive(pre-image)` through `Save` (`service.go:116`). `saveSQL` sets `health_points = $2`,
`current_streak = $4` and `stage = $3` as absolute values from that pre-image.

If a `POST /quests/progress` that crosses 1800 s lands its `OnTargetMet` inside that window
(health 0 -> 20, streak 0 -> 1, marker = today), the revive then writes 50 / streak 0 / sprout.
The +20 and the streak day are erased. Thanks to this plan's `GREATEST`, the marker survives,
and quests has already flagged `daily_progress`, so the success is never re-applied. The result
(50/0) matches neither serial order: revive then met gives 70/1, and met then revive gives 409
`pet_not_wilted` with 20/1.

Reaching it takes two in-flight requests from the same learner: the crossing progress call, and a
revive call that polls on 900 s of new study. That is plausible with two tabs, or with a client
that fires both calls after a single task. It costs one day's +20 and one streak day. Low.

## Expected output
- Revive's success write computes on the live row under a predicate, like the three verdict
  writers. For example:
  `UPDATE pet_states SET health_points = 50, current_streak = 0, stage = 'sprout', updated_at = $now, judged_through = GREATEST(judged_through, $today) WHERE user_id = $1 AND health_points = 0`,
  which reports `applied`. When it returns `applied == false`, the plant is no longer wilted: re-read
  and answer with the live state, or with 409. Either way the concurrent +20 is never overwritten.
- The fake mirrors that predicate under one lock.
- An integration section runs `SaveTargetMet` between a revive's read and its write, and asserts
  that the result is not 50/0.
- The `Repo` type comment (`repo.go:32`, "no writer takes a Go-side pre-image except Save
  (revive)") and the CODEMAP pet paragraph are updated to match. The reviewer added a note on
  the branch saying the revive's health and streak are still absolute pre-image writes.

## Evidence
- Plan under review: `harness/plans/2026-09-24-get-quests-daily-still-reads-is-target-met-from-the-volatile.md` (branch `harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile`).
- `backend/internal/pet/service.go:76-118`: `Revive` = `Ensure` (read) ... Redis reads ... `repo.Save(ctx, userID, ApplyRevive(st, …))`.
- `backend/internal/pet/repo.go:79-83`: `saveSQL` writes `health_points = $2, stage = $3, current_streak = $4` unconditionally. Only the two dates are `GREATEST`-protected.
- `backend/internal/pet/repo.go:29-35`: the `Repo` comment names this exception without calling it unsafe.
- Sibling idea `harness/ideas/_inbox/pgrepo-save-resets-last-target-met-date-to-null-on-the-reviv.md` fixed only the marker for the same window.
