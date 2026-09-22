---
plan: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/two-recordprogress-error-branches-are-uncovered-and-the-redi.md, harness/ideas/_inbox/two-comments-in-the-new-progress-validation-misstate-the-cod.md]
---
# Review 2 (focused re-review after the blocker fixes) — Quests: daily quest suite and progress recording

**Plan:** `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`
**Fix plan:** `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md` (`amends` the above, same branch)
**Branch/worktree:** `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` / `.worktrees/quests-daily-quest-suite-and-progress-recording` @ `4d4b00c`
**Scope:** `git diff ef4b9fa..4d4b00c` and what it touches. The first review
(`harness/reviews/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`, verdict `fail`) is not redone;
its findings 1-7 stand except where these five commits change them.
**Diff:** 10 files, +374/-21 — 9 in `backend/internal/quests/`, plus `harness/CODEMAP.md`. Production code is 4 files,
+89/-7; the rest is tests.

## Plan vs idea

Unchanged from review 1 — the idea's *Expected output* was already delivered there and these commits add no
scope. Two of that review's gaps against the idea are now closed: `day_number` genuinely counts calendar days
(the idea's "timezone boundary picks the right date" row now holds in DST zones, not only fixed-offset ones),
and the table list is still `roadmaps`/`exercises`/`daily_progress` plus the pre-existing direct `users` read —
`CheckExercise` adds a fourth query but no fifth table.

## Code vs plan

All five tasks of the fix plan landed as written, one commit each, no deviations:

```
4d4b00c quests: integration-test the review's reproductions; CODEMAP
89dd878 quests: test the Redis-down and upsert-failure paths
551770a quests: bound duration_seconds per call (3600) and per day (86400)
4f39f85 quests: check exercise ownership and day before any progress write
3c760b3 quests: DayNumber counts calendar days, not elapsed hours
```

**Verification re-run** (worktree `4d4b00c`; scratch stack `POSTGRES_PORT=5436 REDIS_PORT=6384` under an isolated
compose project, API on 18317 — the owner's 6379/6380 containers and other agents' 5433-5435/6381-6383/8099/18201
untouched; scratch `backend/.env`, `backend/cmd/devtoken/` and the built binaries deleted, `git status --short`
clean and `docker compose down -v` run afterwards):

```
go build ./... && go vet ./...                                        -> no output
env -u TEST_DATABASE_URL -u TEST_REDIS_URL -u DATABASE_URL -u REDIS_URL go test ./... -count=1
  ok auth 2.235s | config 1.368s | health 0.770s | quests 3.521s | store 2.615s
go test ./internal/quests/... -v            -> 0 '--- FAIL'

-run DSTTransitions                                                            -> 17 '--- PASS'
-run RejectsAnExerciseOutsideTheActiveRoadmap|RejectsAnExerciseFromAnotherDay|Returns404ForAnUnknownExercise -> 3
-run OutOfRangeDuration|MaximumPerCallDuration|DailyCeiling|RejectsABadBody     -> 4
-run IncrementsRedisBeforeWritingPostgres|CrossingExactly1800|FurtherProgress   -> 3
-run RedisFailureWritesNothing|UpsertFailureAfterTheIncrby                      -> 2

TEST_DATABASE_URL=… TEST_REDIS_URL=… make test-integration
  --- PASS TestIntegrationUpsertCreatesThenPreservesTheLearnerState
  --- PASS TestIntegrationDailyAndProgressAgainstRealServices
  --- PASS TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
  --- PASS TestIntegrationConcurrentMigrateDoesNotRace
  --- PASS TestIntegrationPetStatesRejectsASecondRowForTheSameUser
  --- PASS TestIntegrationRedisRoundTrip            (no --- SKIP, no FAIL)

grep MaxDurationSeconds = 3600 / MaxDailySeconds = 86400  day.go   -> :19, :29
grep -c CheckExercise            repo.go / service.go              -> 3 / 3
grep -c ErrInvalidDuration       handler.go                        -> 1
grep -c 'len(h.log.calls) != 0'  service_test.go / handler_test.go -> 5 / 2
grep -c markErr                  *_test.go                         -> 0
grep -c 'Hours()/24'             day.go                            -> 0
python3 tools/harness/cli.py validate                              -> exit=0
git status --short (worktree)                                      -> clean
```

Everything the executor's summary claims reproduces. **No executor gate failure.**
CI on the pushed branch is green: run `35755768316` (`backend-unit` 19s, `backend-integration` 33s,
`harness-tooling` 6s, conclusion `success`) on commit `4d4b00c`.

## Quality

### 1. Both blockers are genuinely fixed — live, and the tests bite

**Runtime proof, real binary + real Postgres/Redis, two seeded users:**

```
POST /quests/progress  user A's own day-1 exercise, 1800s
  -> 200 {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":0}

BLOCKER 1 — POST /quests/progress  user B's real exercise id, 999s
  -> 404 {"error":"exercise_not_found"}
  GET /quests/daily            -> "accumulated_seconds":1800      (review 1 saw 2799)
  redis GET daily:accumulated:<A>:<day> -> 1800
  psql daily_progress          -> 30|t                            (review 1 saw 46|t)
  psql exercises (user B's row) -> is_completed = f

BLOCKER 2 — duration_seconds 3601 / 1e9 / 1e14
  -> 400 {"error":"invalid_request"}  x3                          (review 1 saw 200, 200, 500)
  redis GET …                  -> 1800 still;  psql daily_progress -> 30|t still
  POST … 60s (the ordinary next call)
  -> 200 {"daily_seconds_spent":1860,"daily_minutes_spent":31,…}  (review 1 saw 500 — the day was bricked)
  POST … 3600s (the inclusive bound) -> 200, 5460s/91m
```

**The tests would fail if the fix were reverted.** Confirmed in a throwaway copy of `backend/` outside the repo
(nothing committed, worktree verified clean afterwards):

- Moving `CheckExercise` back to after `counter.Add`:
  ```
  handler_test.go:174: a 404 touched Redis/Postgres: [INCRBY u1|2026-09-22 600 EXPIRE u1|2026-09-22]
  --- FAIL: TestProgressHandlerReturns404ForAnUnknownExercise
  service_test.go:214: a rejected call touched Redis/Postgres: [INCRBY … EXPIRE …]
  --- FAIL: TestRecordProgressRejectsAnExerciseOutsideTheActiveRoadmap
  service_test.go:229: a rejected call touched Redis/Postgres: [INCRBY … EXPIRE …]
  --- FAIL: TestRecordProgressRejectsAnExerciseFromAnotherDay
  ```
- Dropping the per-call cap and the daily ceiling: `--- FAIL` on `TestProgressHandlerRejectsABadBody`,
  `TestRecordProgressRejectsAnOutOfRangeDuration`, `TestRecordProgressRejectsCrossingTheDailyCeiling`.
- Restoring `int(today.Sub(start).Hours()/24) + 1`: `--- FAIL` on `TestDayNumberCountsCalendarDaysAcrossDSTTransitions`
  and on exactly the four spring-forward subtests annotated `// was N`.

The zero-writes assertions are load-bearing, not decoration. `h.log.calls` is a genuine write log — `CheckExercise`,
`Profile`, `ActiveRoadmap` and `Counter.Total` are reads and deliberately do not append to it, so `len(calls) == 0`
means *no write happened*, which is exactly the claim.

### 2. The DST fix is correct, and UTC / Asia/Ho_Chi_Minh did not regress

`DayNumber` now re-expresses both local midnights as UTC dates before subtracting
(`day.go:64-66`), so the length of the local day is irrelevant. Checked independently against the real 2026
transitions rather than trusting the table: Europe/London 03-29 and 10-25, America/New_York 03-08 and 11-01,
Australia/Sydney 10-04 (forward, southern hemisphere) and 04-05 (back). All sixteen expected day numbers are the
correct calendar-day counts.

**Fall-back is covered in both hemispheres** — London 10-20 -> 10-25/10-26 (6/7), New York 10-28 -> 11-01/11-02
(5/6), Sydney 04-01 -> 04-05/04-06 (5/6). Those six cases pass under *both* the old and the new formula, which is
right: a 25-hour day never lost a day, so they are regression guards, and the test does not pretend otherwise (only
the four spring-forward rows carry `// was N`). The parent test also pins 01:30 UTC on 2026-03-29 — inside the hour
London skips — at day 5.

No regression on the pre-existing zones: `day_test.go`'s UTC and Asia/Ho_Chi_Minh cases and
`TestDayNumberCrossesTheBoundaryInTheUsersTimezone` still pass, as does `service_test.go`'s timezone-boundary test.
The guard `if loc == time.UTC { t.Fatalf("tzdata missing") }` is a good touch — without it the whole table would
silently pass as UTC on a machine with no zoneinfo.

One residual, unchanged by this fix and not worth filing: `startOfDay` uses `time.Date` with a local midnight that
does not exist in zones whose transition is *at* midnight; Go normalises it forward to 01:00 on the same calendar
day, so `.Day()` is still right and `DayNumber` is unaffected.

### 3. The §5.2 ordering contract still holds, and the race window is bounded as claimed

The writes are still three straight-line statements in the required order — `counter.Add` (INCRBY+EXPIRE in one
`TxPipeline`) -> `progress.Upsert` -> `MarkComplete` -> hook -> `State` (`service.go:101-140`). What moved in front
of them is four *reads*: `Profile`, `ActiveRoadmap`, `CheckExercise` and `Counter.Total`. That is not a weakening of
§5.2 — as review 1 argued, §5.2's own step 1 establishes the task before step 2's INCRBY, so validating first *is*
the contract. `TestRecordProgressIncrementsRedisBeforeWritingPostgres` still compares the entire call sequence with
`reflect.DeepEqual` and still passes; the ordering is enforced by the code, with the test as the regression guard.

**The race window is bounded, and the fix plan's arithmetic is right.** `Counter.Total` then `counter.Add` is a
read-then-write with no CAS, so N concurrent calls can each pass a ceiling check that the others invalidate; the
overshoot is at most `N x MaxDurationSeconds` above 86400. Overflowing `minutes_spent INT` (2147483647 minutes =
1.288e11 s) needs `(1.288e11 - 86400)/3600 ~= 3.6e7` simultaneous max-size requests — the figure in the plan's
*Notes*. The race is also self-limiting: once the counter is over the ceiling every further call is rejected, so the
overshoot happens once and does not compound. `day.go`'s own comment puts this at `~10^6` instead, disagreeing with
the plan by 36x (bug 2 below) — wrong number, right conclusion.

Two consequences worth stating rather than filing. A rejection at the ceiling costs one extra Redis `GET` per
progress call and locks the learner out of recording for the remainder of that *local date* (the key is per-date, so
tomorrow is fresh) — vastly better than the 48-hour brick it replaces, and only reachable by 24 deliberate
max-size reports. And `Counter.Total` failing now aborts before `Add`, so a Redis outage still writes nothing.

### 4. What the fix introduced

Nothing dead, nothing leaked — but two things are weaker than they look, and both are in the tests:

- **`counter.Add`'s error branch is now uncovered, and `TestARedisFailureWritesNothing`'s comment says otherwise.**
  The comment reads "Counter.Total and Counter.Add both fail"; `Total` is called first and short-circuits, so `Add`
  never runs. Deleting the `if f.err != nil` guard from `fakeCounter.Add` in a scratch copy leaves the suite green.
  The behaviour is fine (both abort before any Postgres write); the claim is not. Bug 1 below.
- **`MarkComplete`'s error branch lost its only test hook.** `4f39f85` deleted `fakeQuestRepo.markErr`. The
  production branch is kept on purpose — `service.go:116-119` documents it as the guard for the row vanishing
  between `CheckExercise` and the update — but replacing it with `_ = s.quests.MarkComplete(...)` in a scratch copy
  breaks no test. Kept-on-purpose code with no coverage. Bug 1 below.

No error shape leaks: `ErrInvalidDuration`'s wrapped message carries the running total and the ceiling
(`service.go:98`), and the handler maps it to a bare `{"error":"invalid_request"}` — confirmed on the wire for all
four rejection cases. `ErrExerciseNotFound` stays a 404 rather than a 403, so ids remain unprobeable, and
`CheckExercise` deliberately cannot distinguish "unknown" from "someone else's" (`repo.go:49-52`). Returning 404 for
another day's *own* exercise is the same honest choice.

Two comments misstate their code (the `~10^6` figure above, and `handler.go:65-68`'s "the request is malformed, not
the state" — false for the daily-ceiling branch, where a well-formed 200s report is rejected because of state).
Bug 2 below. The behaviour of collapsing both into `invalid_request` is the plan's explicit, reasoned decision, so
the behaviour is not the finding; the comment asserting the opposite of half its branch is.

Incidental improvement, unfiled: a non-UUID `exercise_id` still yields 500 (Postgres type error, pre-existing low
bug 10), but it now happens in `CheckExercise` *before* any write instead of in `MarkComplete` after two, so that
path no longer mutates state either.

### 5. §6.2 wire contract unchanged

Verified by the plan's greps **and** on the wire against the running binary:

```
grep '"duration_seconds"|"user_answers"'                         handler.go  -> :39, :44
grep '"daily_seconds_spent"|"daily_minutes_spent"|"pet_health"|"streak_count"'  service.go -> :19,:20,:22,:23
grep '"day_number"|"total_minutes_required"|"accumulated_seconds"|"tasks"'      service.go -> :169,:170,:171,:173
grep -r '"total_seconds"|"target_met"|"newly_met"|"exercises"|"seconds"' (non-test) -> no hits

GET  /api/v1/quests/daily  -> keys ['accumulated_seconds','date','day_number','is_target_met','tasks','total_minutes_required']
                              tasks[0] keys ['content_json','duration_minutes','id','is_completed','task_type','title']
POST /api/v1/quests/progress -> {"daily_seconds_spent","daily_minutes_spent","is_target_met","pet_health","streak_count"}
GET  /api/v1/quests/daily  (no Authorization) -> 401 {"error":"unauthorized"}
```

Exactly the §6.2 field sets, no additions, no renames. `NewlyMet` is still `json:"-"`. The new
`ErrInvalidDuration` reuses the existing 400 code rather than inventing one, so the frontend sees nothing new.
**The contract review 1 cleared is untouched — the frontend can still build on it.**

### 6. CODEMAP

The executor's update is accurate and needed no correction. It records the validate-before-write ordering, both
constants with their values, the `CheckExercise` day scoping, the DST-safe day count with the zones covered, and
that the rejection tests assert an empty call log. The "self-heals" caveat review 1 corrected at `ef4b9fa` is
preserved verbatim.

## Which of review 1's ordinary bugs these commits resolve

For the evaluator — do not treat these as new work:

| # | Inbox bug | Now |
|---|---|---|
| 3 | `daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md` [high] | **Resolved.** `3c760b3` + 16-case DST table; reverting the formula fails exactly the four spring-forward rows. |
| 7 | `the-progress-rejection-tests-assert-only-the-error-and-the-f.md` [medium] | **Resolved as written.** Both rejection tests (and 5 more) assert `len(h.log.calls) == 0`; `fakeProgressRepo.err` and `fakeCounter.err` are both set by real tests; neither field is dead. The residue — `Add`'s branch still unreached, and the newly deleted `markErr` — is *new*, and is filed separately as bug 1 rather than left on this idea. |
| 10 | `progress-request-validation-is-incomplete-outside-the-gin-bi.md` [low] | **Resolved.** Both halves: the service now returns typed `ErrInvalidDuration` mapped to 400 (tests assert `errors.Is`, not just `err != nil`), and `CheckExercise` scopes to the resolved `day_number` with `TestRecordProgressRejectsAnExerciseFromAnotherDay` proving another day's exercise is a 404. The non-UUID 500 the idea did not name persists, but no longer writes first. |

Still open, unchanged by these commits — `quests-reads-the-users-table-directly-for-the-timezone.md` [medium]
(`repo.go:66,109` still `SELECT … FROM users`), `ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md`
[medium] (explicitly not folded in; narrower now that invalid input cannot reach the INCRBY, but the upsert/
`MarkComplete` failure path is unchanged, and `TestAnUpsertFailureAfterTheIncrby…` deliberately does not assert
recovery), `a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` [medium] (`service.go:136` still
`pet = PetState{}`), `seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md` [low] (`seed.go` not touched),
`no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md` [low] — and slightly larger in scope, since
`CheckExercise` adds a fourth per-request Postgres round trip (it filters on the `exercises` primary key, so it is
an index lookup, but `roadmaps(user_id, is_active)` and `exercises(roadmap_id, day_number)` are still unindexed).

## Bugs filed

Ordinary inbox bugs (no blockers; neither stops the merge):
1. `harness/ideas/_inbox/two-recordprogress-error-branches-are-uncovered-and-the-redi.md` [medium] — `counter.Add`'s
   and `MarkComplete`'s error branches have no coverage, and `TestARedisFailureWritesNothing`'s comment claims one
   of them does; both proved by deleting the branch in a scratch copy and watching the suite stay green.
2. `harness/ideas/_inbox/two-comments-in-the-new-progress-validation-misstate-the-cod.md` [low] — `MaxDailySeconds`'s
   race-headroom figure is 36x off and contradicts its own plan; the `ErrInvalidDuration` handler comment is false
   for the daily-ceiling branch.

## Verdict

**pass-with-bugs.** Both blockers are genuinely fixed, not merely asserted: reproduced live against a real binary,
real Postgres and real Redis — the 404 leaves the counter at 1800, `daily_progress` at `30|t` and the other user's
exercise uncompleted; 3601/1e9/1e14 all return `400 invalid_request` with the stores untouched, and the following
ordinary 60s report returns 200. The tests that guard them fail when the fix is reverted, so they are real
regression guards rather than assertions written to match the code. The DST fix is correct on all three tabled
zones in both directions and regresses neither UTC nor Asia/Ho_Chi_Minh. The §5.2 write ordering is unchanged —
only reads moved in front of it — and the ceiling's read-then-write race is bounded roughly 3.6e7 concurrent
max-size requests away from the INT overflow it exists to prevent. §6.2 is untouched field for field, on the wire
and by grep. CI is green on `4d4b00c`.

The two new findings are quality, not correctness: uncovered error branches with a test comment that overstates
them, and two comments that misdescribe their own code. Both are cheap and neither justifies holding the branch.
With `cli.py blockers --plan …` now exiting 0, this slice is mergeable.

No PR (`pr: skipped-not-a-collaborator`), so the skill's `gh pr comment` / `gh pr ready` steps are skipped.
Merge command for the human: `/harness merge harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`.
