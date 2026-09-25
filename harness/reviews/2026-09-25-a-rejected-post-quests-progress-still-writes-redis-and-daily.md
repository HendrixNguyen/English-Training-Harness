---
plan: harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-non-uuid-exercise-id-on-post-quests-progress-answers-500-i.md]
---
# Review — Quests amend: validate before writing, bound duration_seconds, DST-safe day_number

**Plan:** `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md`
**Branch/worktree:** `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` / `.worktrees/quests-daily-quest-suite-and-progress-recording` — both gone; the plan is already merged.
**Reviewed on:** a detached worktree of `origin/main` at `3f4242d` (post-hoc review, 2026-09-25 daily run). The five plan commits are on main: `3c760b3`, `4f39f85`, `551770a`, `89dd878`, `4d4b00c`.

## Plan vs idea
The plan folds four ideas; each Expected output is met on current main.
- **Blocker 1 (write before validate):** `RecordProgress` runs `QuestRepo.CheckExercise(roadmap, exercise, today's day_number)` before any write (`service.go:96`). The three rejection tests assert `len(h.log.calls) == 0`.
- **Blocker 2 (unbounded duration):** `MaxDurationSeconds = 3600` per call and `MaxDailySeconds = 86400` per local day, both checked before the INCRBY and mapped to `400 invalid_request` through `ErrInvalidDuration`. It rejects rather than clamps, as the plan argued. The post-INCRBY failure case is now unreachable by magnitude, and `TestAnUpsertFailureAfterTheIncrbyLeavesTheCounterUsable` proves the counter stays usable.
- **DST day loss:** `DayNumber` subtracts UTC calendar dates, and there is a 16-case DST table test.
- **Test honesty (medium):** the call-log assertions were added, and both formerly dead fake fields have tests. The idea also asked that "a replay after clearing the error fires `OnTargetMet` exactly once". The plan deferred that openly to the separate hook-loss bug. A later commit (`267ab95`) redid the hook to fire from the durable `is_target_met` flag, so the deferral is superseded rather than missing.

## Code vs plan
All five tasks were followed, with no deviations. Later work touched the files but did not undo the fix: `267ab95` added `MarkTargetMet` and moved `MarkComplete` after the hook.

Verification re-run on `origin/main`:
```
go build ./... && go vet ./...                                  # clean
env -u TEST_* -u DATABASE_URL -u REDIS_URL go test ./... -count=1  # 13 packages ok
DSTTransitions                                                  => 17 PASS  (expect 17)
RejectsAnExerciseOutsideTheActiveRoadmap|…FromAnotherDay|Returns404ForAnUnknownExercise => 3 (expect 3)
OutOfRangeDuration|MaximumPerCallDuration|DailyCeiling|RejectsABadBody => 4 (expect 4)
IncrementsRedisBeforeWritingPostgres|CrossingExactly1800|FurtherProgress => 3 (expect 3)
RedisFailureWritesNothing|UpsertFailureAfterTheIncrby          => 2 (expect 2)
grep -c 'len(h.log.calls) != 0'                                 # service_test.go:5 handler_test.go:2
grep -n 'Hours()/24' day.go                                     # no hits
grep -n 'markErr' *_test.go                                     # 4 hits — NOT a regression: a new
                                                                #   fakeProgressRepo.markErr (MarkTargetMet)
                                                                #   from 267ab95; the removed QuestRepo field is gone
```
Integration on scratch Postgres/Redis (`COMPOSE_PROJECT_NAME=rev0925olds`, ports 5447/6397, `make up` / `make down`): `make test-integration` gave 12 × `--- PASS`, including `TestIntegrationDailyAndProgressAgainstRealServices` (the review's two reproductions), with no SKIP or FAIL. I did not repeat the HTTP runtime proof with a throwaway `cmd/devtoken`. The integration test covers the same reproductions at service level, and CI's `backend-integration` job has been green on every main push since then.

## Quality
- **Boundaries:** the ownership check stays inside `quests` through the `QuestRepo` interface. §5.2's Redis-first order is unchanged: Total (read) → INCRBY → upsert → hook → MarkComplete.
- **Non-UUID `exercise_id` → 500.** The plan's Notes deferred this to a low inbox bug, which was later rejected as "overtaken", but that bug never covered the UUID case. Nothing tracks it now, so I filed it (low).
- **Comment arithmetic nit, not filed:** `day.go` says the INT limit stays "~10^6 concurrent max-size requests away". The real figure is (2³¹−1)·60 / 3600 ≈ 3.6·10⁷. The comment errs on the safe side, so it is harmless.
- **Accepted trade-off, as the plan noted:** a 500 after the INCRBY keeps the increment, so a client that retries double-counts that report. The daily ceiling bounds it.

## Bugs filed
- `harness/ideas/_inbox/a-non-uuid-exercise-id-on-post-quests-progress-answers-500-i.md` (low): a malformed `exercise_id` gets 500 from the Postgres uuid cast in `CheckExercise`.

## Verdict
**pass-with-bugs.** Both blockers and the DST bug are fixed and still hold on current main. The one new finding is low. The plan is already merged, so nothing here blocks.
