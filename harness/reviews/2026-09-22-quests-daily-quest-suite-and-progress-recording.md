---
plan: harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md
verdict: fail
bugs: [harness/ideas/_inbox/a-rejected-post-quests-progress-still-writes-redis-and-daily.md, harness/ideas/_inbox/duration-seconds-is-unbounded-so-one-request-bricks-a-user-s.md, harness/ideas/_inbox/daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md, harness/ideas/_inbox/quests-reads-the-users-table-directly-for-the-timezone.md, harness/ideas/_inbox/ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md, harness/ideas/_inbox/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md, harness/ideas/_inbox/the-progress-rejection-tests-assert-only-the-error-and-the-f.md, harness/ideas/_inbox/seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md, harness/ideas/_inbox/no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md, harness/ideas/_inbox/progress-request-validation-is-incomplete-outside-the-gin-bi.md]
---
# Review — Quests: daily quest suite and progress recording

**Plan:** `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`
**Branch/worktree:** `harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording` / `.worktrees/quests-daily-quest-suite-and-progress-recording`
**Diff:** `git diff main...harness/2026-09-22-high-quests-daily-quest-suite-and-progress-recording --stat`

## Plan vs idea

The idea's *Expected output* is delivered, with the substitutions its own *Reconciliation note*
authorised. `GET /api/v1/quests/daily` and `POST /api/v1/quests/progress` exist behind
`auth.Require()`; the roadmap/`day_number`/clamp arithmetic, the Redis-first ordering, the once-only
hook (renamed `TargetMetListener` -> `quests.Pet`, now `OnTargetMet` + `State`), the 404
`no_active_roadmap`, and the demo-roadmap fixture (`store.SeedDemoRoadmap`) are all present. Every
test the idea named exists and asserts what it claims:

| Idea's required test | Where | Honest? |
|---|---|---|
| INCRBY precedes the Postgres upsert | `service_test.go:38` call-log | yes — full sequence compared with `reflect.DeepEqual` |
| crossing exactly 1800s fires the listener once | `service_test.go:78` | yes |
| a second call the same day does not re-fire | `service_test.go:115` | yes |
| `day_number` clamps at 28 | `day_test.go:44`, `service_test.go:302` | yes |
| timezone boundary picks the right date | `day_test.go:56`, `service_test.go:159` | yes for fixed-offset zones; **no DST case** (bug 3) |

The one substantive gap against the idea is its table list: *Expected output* says "Tables:
`roadmaps`, `exercises` (read; `is_completed` write), `daily_progress` (upsert)", and the code also
reads `users` directly (bug 4).

## Code vs plan

All nine tasks followed, no deviations. The executor's "every file matched the plan's listing
verbatim, including the §6.2 field names" reproduces: 16 files, +1599/-6, nine commits each carrying
the `Co-Authored-By: Claude Fable 5.1` trailer, `git status --short` clean.

**Verification re-run** (worktree, `ee99b1a`, scratch stack on `POSTGRES_PORT=5433 REDIS_PORT=6381`,
API on port 18123 — the owner's 6379/6380 containers untouched, scratch `backend/.env` and
`cmd/devtoken` deleted afterwards):

```
go build ./... && go vet ./...                                        -> no output
env -u TEST_DATABASE_URL -u TEST_REDIS_URL -u DATABASE_URL -u REDIS_URL go test ./... -count=1
  ok auth 2.896s | config 1.907s | health 0.911s | quests 4.477s | store 3.407s
go test ./... -count=1 -v -run Integration -p 1   (TEST_* exported)
  --- PASS TestIntegrationUpsertCreatesThenPreservesTheLearnerState
  --- PASS TestIntegrationDailyAndProgressAgainstRealServices
  --- PASS TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
  --- PASS TestIntegrationConcurrentMigrateDoesNotRace
  --- PASS TestIntegrationPetStatesRejectsASecondRowForTheSameUser
  --- PASS TestIntegrationRedisRoundTrip          (no --- SKIP, no FAIL)

grep '"accumulated_seconds"|"total_minutes_required"|"tasks"'  service.go  -> 3 hits (139,140,142)
grep '"daily_seconds_spent"|"daily_minutes_spent"|"pet_health"|"streak_count"' service.go -> 4 hits (14,15,17,18)
grep '"duration_seconds"|"user_answers"' handler.go            -> 2 hits (39,44)
grep -r '"total_seconds"|"target_met"|"newly_met"|"exercises"|"seconds"' (non-test) -> no hits
grep 'store.DailyAccumulatedKey|store.DailyAccumulatedTTL' counter.go -> 2 hits (30,46)
grep 'TargetSeconds = 1800' day.go                             -> 1 hit
grep 'TEMPORARY' internal/store/seed.go                        -> 1 hit
grep -r 'Getenv("DATABASE_URL")|Getenv("REDIS_URL")' quests/ seed*.go -> no hits
grep -c '^func TestIntegration' quests/integration_test.go     -> 1
grep 'auth.Require(tokens, sessions)' cmd/api/main.go          -> 1 hit (:71)
python3 tools/harness/cli.py validate                          -> exit=0
```

**Runtime proof re-run** — reproduces the executor's block exactly, plus two probes it did not make:

```
GET  /healthz                                    -> 200
GET  /api/v1/quests/daily   (no Authorization)   -> 401 {"error":"unauthorized"}
GET  /api/v1/quests/daily   (real HS256 bearer + Redis session)
  -> 200 {"date":"2026-09-22","day_number":1,"total_minutes_required":30,
          "accumulated_seconds":0,"is_target_met":false,"tasks":[3 × vocabulary/reading/practice,
          "title":"Day 1 vocabulary","duration_minutes":10,"is_completed":false,"content_json":{…}]}
POST /api/v1/quests/progress {"exercise_id":"<vocab>","duration_seconds":600,"user_answers":{"q1":"A"}}
  -> 200 {"daily_seconds_spent":600,"daily_minutes_spent":10,"is_target_met":false,"pet_health":100,"streak_count":0}
POST … {"duration_seconds":1200}   (crosses 1800)
  -> 200 {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":0}
GET  /api/v1/quests/daily -> accumulated_seconds 1800, is_target_met true, vocabulary completed, other two false

NEW: POST … {"exercise_id":"<another user's exercise>","duration_seconds":999}
  -> 404 {"error":"exercise_not_found"}   …and accumulated_seconds 1800 -> 2799,
     psql daily_progress -> minutes_spent 46, is_target_met t      ← written by the rejected request
NEW: POST … {"duration_seconds":100000000000000}
  -> 500; the next ordinary POST {"duration_seconds":60} -> 500 as well; the day is bricked for 48h
```

Everything the executor claimed reproduces. There is **no executor gate failure** — the plan was
correctly marked `done`; the findings below are quality findings the gate does not cover.

CI on the pushed branch is green (`gh run list`: run 35749993289, `success`, 38s).

## Quality

**1. Wire contract vs backend spec §6.2 — correct, field for field.** Both DTOs match the §6.2 JSON
blocks exactly, in name, type and nullability: `{date:string, day_number:int, total_minutes_required:int,
accumulated_seconds:int64, is_target_met:bool, tasks[{id, task_type, title, duration_minutes,
is_completed, content_json}]}` and `{daily_seconds_spent, daily_minutes_spent, is_target_met,
pet_health, streak_count}`; the request is `{exercise_id, duration_seconds, user_answers?}` with
`user_answers` genuinely optional. Verified on the wire, not just by grep. `tasks` is
`make([]Task, 0, …)` so an empty day serialises `[]`, not `null` — the frontend can iterate
unconditionally. `NewlyMet` is `json:"-"` and never appears. The pre-§6.2 names are gone, and
`{"seconds":600}` is rejected 400 rather than aliased. `total_minutes_required` is derived
(`TargetSeconds/60`), not a literal 30. Error shapes (`{"error":"<code>"}`, 400 `invalid_request`,
401 `unauthorized`, 404 `no_active_roadmap`/`exercise_not_found`, 500 `internal_error`) are §6.2-silent
and consistent with the merged auth slice. **Nothing here blocks the frontend.**

**2. The §5.2 ordering contract — enforced by the code, but validation sits on the wrong side of it.**
The order is real: `service.go:75-98` executes `counter.Add` (INCRBY+EXPIRE in one `TxPipeline`) ->
`progress.Upsert` -> `MarkComplete` -> hook -> `State`, as straight-line statements. The call-log fake
is a regression guard on top of that, not the only thing holding it — the answer to "code or test?"
is *code*. `Daily`/`RecordProgress` never read `daily_progress` back to decide anything, so Redis is
genuinely the single source of truth for the day, as the idea wanted.

What the ordering does *not* survive is failure. Postgres failing after the INCRBY returns 500 with
the counter already raised; the plan's "self-heals within the 48h TTL" is true of `minutes_spent`
(the next call re-derives it) but false of the hook — see finding 3 — and it also invites the
double-counting retry that the plan elsewhere argues against for `Pet.State`. Worse, `MarkComplete`
is both the *ownership check* and the third write, so a request that ends 404 has already moved both
stores (**blocker 1**, reproduced above). §5.2's own step 1 is "Complete Task", i.e. the task is
established before the INCRBY of step 2 — validating first is not a deviation from the contract, it
is the contract.

**3. Once-only `newly_met` — arithmetically right, operationally fragile.**
`total >= 1800 && total-seconds < 1800` is correct for a delta that jumps from far below to far above
(5000 from 1000: fires), for replays (each replay increments, only the crossing call fires), and for
`<= 0` (rejected before any write, and the test proves the call log stays empty). `LocalDate` is
correct at 23:59 local and across DST, because it formats in the location — 23:59 local lands on the
local date, which is what both the §4 key and `daily_progress.date` use.

Two real problems. `DayNumber` divides elapsed hours by 24 instead of counting calendar days, so it
**loses a day permanently at every spring-forward transition** — reproduced for Europe/London,
America/New_York and Australia/Sydney (bug 3); the existing tests use UTC and Asia/Ho_Chi_Minh,
neither of which observes DST. And because `newly_met` is an inferred edge on a counter that is
raised before the durable writes, a failure on the crossing call means `OnTargetMet` never fires for
that user that day — "exactly once" is really "at most once" (bug 5). Nothing consumes the hook yet,
so this is medium today and a lost streak the moment the pet slice lands.

Idempotency is absent by design (no request id), so a client retry double-counts. That is inherent to
the spec's client-asserted model and is only worth noting because the 500 paths make retries likely.

**4. Trust boundary — ownership is enforced, magnitude is not.** `markCompleteSQL` matches
`id = $1 AND roadmap_id = $2` against the caller's own active roadmap, so a user cannot complete
another user's exercise; confirmed at runtime (404 for a second user's exercise id). Returning 404
rather than 403 keeps ids unprobeable, and an inactive roadmap is excluded because `ActiveRoadmap`
filters `is_active = TRUE`. Good.

`duration_seconds` has no ceiling: 10^9 is accepted with a 200 and `daily_minutes_spent: 16666713`;
10^14 overflows `minutes_spent INT` and leaves the user's counter poisoned and every later progress
call a 500 for the rest of the 48h TTL (**blocker 2**). The plan's "trusts the client's
`duration_seconds`" note covers the *trust*; it does not cover the overflow or the brick. A per-call
and per-day cap closes both. `MarkComplete` also does not scope to today's `day_number`, so a learner
can tick day 28's tasks on day 1 (bug 10).

**5. `NopPet` and the `quests.Pet` seam — well shaped, one bad failure value.** The interface takes a
`context.Context`, returns an `error`, and is small enough that the pet slice can implement it without
importing anything from quests but the two types. `OnTargetMet` carries the `localDate` the bump
belongs to, which the pet slice needs for §8's once-per-day arithmetic; the doc comment records that
it fires after the commit, best-effort, and warns about the §8 cron double-bump. `State` being read
*after* the hook is proven by the fake's 80->100 / 4->5 assertion — a genuinely good test. `NopPet`
returning the §3.2 defaults (100, 0) is honest: that is a freshly onboarded pet, not a fabricated
error state.

Two gaps. The interface does not say what `State` must return for a user with **no `pet_states` row**
— the pet slice will have to guess between an error, a zero value and the §3.2 defaults; the doc
comment should say (the reviewer's reading: not-found is not an error, it is 100/0). And the plan's
choice of 200-with-zeros on a `State` failure is wrong in this domain: §8 makes 0 health a *dead
plant* with a revive challenge, so a transient read error tells the client the learner's plant just
died on the very call that recorded a successful session (bug 6). Keeping the 200 is right; the zeros
are not — omit the fields instead.

**6. `SeedDemoRoadmap` — genuinely temporary, and the keys match.** One file, one unambiguous
`TEMPORARY … DELETE this file rather than extending it` comment, exported from `store`, called from
nothing but the gated integration test — a real single deletion point (plus its `seed_test.go` and
the one call site). It writes `{"title":"Day N <type>","duration_minutes":10,…}`, exactly the keys
`toTask` reads, confirmed on the wire (`"title":"Day 1 vocabulary","duration_minutes":10`). It is not
transactional and never deactivates an existing roadmap, so a mid-loop failure leaves an *active*
roadmap with missing days that `GET /daily` serves as `"tasks": []` (bug 8).

**7. Boundaries, tests, CODEMAP, idiom.**
- *Boundaries:* quests never touches `pet_states` — the only mentions are doc comments and a test's
  error string, and the `Pet` interface is the sole channel. It **does** read `users` directly
  (`repo.go:61`), which is `auth`'s table and is absent from the idea's table list (bug 4). The
  interface/fake split is otherwise exemplary: the whole package tests with no live service, and
  `counter.go` never hand-builds a §4 key.
- *Test honesty:* mostly high. The call-log test compares the entire sequence, the `content_json`
  mapping and 400 table are real, and the gated integration test genuinely round-trips Postgres and
  Redis. Two exceptions: both "rejects an unknown exercise" tests assert only the error and never the
  call log — which is exactly why blocker 1 shipped green, and the sibling non-positive-seconds test
  *does* make that assertion — and `fakeCounter.err` / `fakeProgressRepo.err` are dead fields no test
  sets, so Redis-down and upsert-failed have no coverage at all (bug 7). `store/seed_test.go` asserts
  two constants against their own literals.
- *Performance:* `Daily` costs three Postgres round trips plus one Redis; `RecordProgress` four plus
  one pipeline. `Profile` could fold into the roadmap query. More concretely, neither
  `roadmaps(user_id, is_active)` nor `exercises(roadmap_id, day_number)` has an index, and both run
  per request (bug 9) — the DDL must stay §3.2-verbatim, so that needs a `0002_` migration.
- *CODEMAP:* accurate and unusually thorough — the `quests` paragraph names the DTOs, the ordering,
  the once-only derivation, the `Pet` seam and the gating, and the `store` paragraph already records
  `SeedDemoRoadmap` as temporary scaffolding. One sentence was materially incomplete (the
  "self-heals" claim does not hold for the hook); the reviewer corrected it on the branch at
  `ef4b9fa`.
- *Go idiom:* good. Sentinel errors with `errors.Is`, `%w` wrapping everywhere, interfaces defined at
  the consumer, `var _ Counter = (*RedisCounter)(nil)` assertions, an injected clock, `rows.Err()`
  checked, `defer rows.Close()`. `counterKey`'s panic on a malformed date is defensible as an
  invariant (`gin.Default()` has Recovery, so it is a 500) but a returned error would be more
  idiomatic in a request path. `toTask` deliberately swallowing a `json.Unmarshal` error is
  documented and correct — one bad row should not 500 the day.

## Bugs filed

Blockers (`blocks` this plan; `cli.py blockers --plan …` exits 1):
1. `harness/ideas/_inbox/a-rejected-post-quests-progress-still-writes-redis-and-daily.md` [high] — a 404 `exercise_not_found` still increments the counter and rewrites `daily_progress`; validate before the writes.
2. `harness/ideas/_inbox/duration-seconds-is-unbounded-so-one-request-bricks-a-user-s.md` [high] — no cap on `duration_seconds`; overflowing `minutes_spent INT` bricks the user's progress endpoint for 48h.

Ordinary inbox bugs:
3. `harness/ideas/_inbox/daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md` [high] — `DayNumber` divides elapsed hours by 24; every DST zone loses a day permanently.
4. `harness/ideas/_inbox/quests-reads-the-users-table-directly-for-the-timezone.md` [medium] — cross-package table access on `users`.
5. `harness/ideas/_inbox/ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md` [medium] — the once-only hook is unrecoverable after a failed write on the crossing call.
6. `harness/ideas/_inbox/a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` [medium] — 0 health is "dead plant", not "unknown".
7. `harness/ideas/_inbox/the-progress-rejection-tests-assert-only-the-error-and-the-f.md` [medium] — rejection tests skip the call-log assertion; two fake error fields are dead.
8. `harness/ideas/_inbox/seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md` [low] — 85 statements, no transaction, no deactivation of prior roadmaps.
9. `harness/ideas/_inbox/no-index-supports-the-quests-lookups-on-roadmaps-and-exercis.md` [low] — no index for either per-request lookup.
10. `harness/ideas/_inbox/progress-request-validation-is-incomplete-outside-the-gin-bi.md` [low] — service-level duration guard yields 500; `MarkComplete` not scoped to today.

## Verdict

**fail** — not because anything the executor claimed is false (it all reproduces, and CI is green),
but because two findings must not merge as they stand: a rejected request mutates both stores, and an
unbounded `duration_seconds` can take a user's write path offline for 48 hours while corrupting that
day's `is_target_met`. Both are cheap fixes on this branch — validate the exercise before the INCRBY,
and cap the duration per call and per day — and neither touches the §6.2 wire contract, which is
correct and safe for the frontend to build on now.

The slice is otherwise strong work: the DTOs are exact, the ordering is enforced in code rather than
only in a test, the `Pet` seam is well shaped for slice 4, and the seed is honestly temporary. The
DST day-loss (bug 3) is the highest-value non-blocking fix and is worth folding into the amend plan
while the package is open.

No PR (`pr: skipped-not-a-collaborator`), so steps 8's `gh pr comment` / `gh pr ready` are skipped.
The branch stays unmerged: `cli.py` refuses `merged=true` while blockers 1 and 2 are unresolved.
