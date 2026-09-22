---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
plan: harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md
---

# A rejected POST /quests/progress still writes Redis and daily_progress

## Why
`Service.RecordProgress` writes both stores **before** it validates that the exercise belongs to the
caller (`backend/internal/quests/service.go:75-90`): `counter.Add` (INCRBY + EXPIRE), then
`progress.Upsert` into `daily_progress`, and only then `quests.MarkComplete`, which is the call that
returns `ErrExerciseNotFound`. When that last step fails the handler answers `404
{"error":"exercise_not_found"}` — but the counter has already been raised and the durable
`daily_progress` row has already been rewritten from it.

So a request the API *rejects* still moves the retention metric the whole product is built around.
The reviewer reproduced it against the real binary and real services (port 18123, scratch Postgres
5433 / Redis 6381):

```
POST /quests/progress {"exercise_id":"<another user's exercise>","duration_seconds":999}
  -> 404 {"error":"exercise_not_found"}

GET /quests/daily     -> accumulated_seconds 1800 -> 2799,  is_target_met true
psql: SELECT minutes_spent, is_target_met FROM daily_progress ...
  -> 46 | t          (46 = 2799/60 — written by the request that was answered 404)
```

Ownership itself is enforced correctly (`markCompleteSQL` is scoped by `roadmap_id`, and 404 rather
than 403 keeps ids unprobeable) — the defect is purely that enforcement happens after the writes.
The same shape means any authenticated user can inflate `daily_progress.is_target_met` without
holding a single valid exercise id, and the pet and notify slices both key off that column.

The plan's §5.2 ordering argument does not cover this: §5.2 step 1 is "Complete Task", i.e. the task
is established before the INCRBY of step 2. The call-log test
(`TestRecordProgressIncrementsRedisBeforeWritingPostgres`) pins Redis-before-Postgres, which a fix
can keep — validation is a read, not a write.

## Expected output
`RecordProgress` resolves and authorises the exercise before it touches either store — e.g. a
`QuestRepo.ExerciseInRoadmap(ctx, roadmapID, exerciseID)` read (or folding ownership into the same
query that already fetches the active roadmap) that returns `ErrExerciseNotFound` up front. The
INCRBY / upsert / `MarkComplete` / hook sequence then runs only for a request that will succeed, so
the §5.2 order is unchanged. A `404` leaves `daily:accumulated:*` and `daily_progress` byte for byte
as they were, asserted by extending the existing call-log test:
`TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap` gains `if len(h.log.calls) != 0 { ... }`,
exactly as `TestRecordProgressRejectsNonPositiveSeconds` already does.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`
- `backend/internal/quests/service.go:75-90` — `counter.Add` -> `progress.Upsert` -> `quests.MarkComplete`.
- `backend/internal/quests/repo.go:76-79,135-144` — `MarkComplete` is the only ownership check.
- `backend/internal/quests/handler.go:67-69` — maps `ErrExerciseNotFound` to 404 after the writes landed.
- `backend/internal/quests/service_test.go:203-211` — asserts the error only, never the call log.
- Runtime proof above, re-run by the reviewer on branch `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` at `ee99b1a`.

## Evaluation

**Verdict: select, `high` (blocker).** Confirmed by the reviewer against the real binary (review
`harness/reviews/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`, *Runtime proof re-run*); not
re-litigated here. The *Why* is real for this product: `daily_progress.is_target_met` is the retention metric the
pet and notify slices key off, and today any authenticated user can flip it without holding one valid exercise id.

**Root cause** (read in the worktree at `ef4b9fa`): `Service.RecordProgress` (`backend/internal/quests/service.go:75-90`)
runs `counter.Add` → `progress.Upsert` → `quests.MarkComplete`; `MarkComplete` (`repo.go:135-144`) is the only place
ownership is checked, via `RowsAffected() == 0` on an `UPDATE … WHERE id = $1 AND roadmap_id = $2`. Ownership is
therefore established by the *last write*, after the two writes that matter have committed. Nothing in §5.2 requires
that: §5.2 step 1 is "Complete Task", so the task is established before step 2's INCRBY.

**Fix shape (owner's call):** a read-only lookup — `QuestRepo.CheckExercise(ctx, roadmapID, exerciseID, day)` —
that returns `ErrExerciseNotFound` unless the exercise sits on the caller's active roadmap **and** on today's
`day_number`, run before any write. The write sequence INCRBY → upsert → `MarkComplete` → hook is unchanged, so the
§5.2 ordering test keeps passing. The two "rejects an unknown exercise" tests gain the `len(h.log.calls) != 0` assertion
that `TestRecordProgressRejectsNonPositiveSeconds` already has — that missing line is why this shipped green
(`the-progress-rejection-tests-assert-only-the-error-and-the-f.md`, folded into the same plan).

**Dependencies:** none beyond the branch. **Plan:** one amending plan shared with the second blocker and the DST bug,
`amends: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`, landing on the same branch.
