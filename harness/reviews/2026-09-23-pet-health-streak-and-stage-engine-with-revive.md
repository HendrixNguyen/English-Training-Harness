---
plan: harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/sweep-reads-updated-at-then-writes-unconditionally-so-two-sw.md, harness/ideas/_inbox/zones-that-skip-local-midnight-on-spring-forward-are-never-s.md, harness/ideas/_inbox/the-hourly-sweep-loads-every-pet-in-a-zone-into-memory-and-r.md, harness/ideas/_inbox/service-ontargetmet-ignores-localdate-so-pet-has-no-idempote.md, harness/ideas/_inbox/meeting-the-daily-target-during-a-revive-challenge-answers-4.md, harness/ideas/_inbox/a-passed-revival-is-knocked-from-50-back-to-20-by-the-same-l.md, harness/ideas/_inbox/main-go-installs-a-signal-handler-with-no-server-shutdown-so.md, harness/ideas/_inbox/pet-sweep-tests-cannot-fail-on-the-branches-they-are-named-f.md, harness/ideas/_inbox/the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md]
---
# Review — Pet: health, streak and stage engine with revive

**Plan:** `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md`
**Branch/worktree:** `harness/2026-09-22-high-pet-health-streak-and-stage-engine-with-revive` / `.worktrees/pet-health-streak-and-stage-engine-with-revive`
**Diff:** `git diff main...harness/2026-09-22-high-pet-health-streak-and-stage-engine-with-revive --stat` → 17 files, +1553 −5. HEAD `f6e27b1`, 10 commits, tree clean.
**No blocker filed.** Nothing found that double-bumps or wrongly penalises a plant under ordinary operation, no data loss, and every §6.3 field verified on the wire.

## Plan vs idea

The idea's *Expected output* is delivered, with the three substitutions its own `## Evaluation` section authorised and one it did not:

| Idea said | Shipped | Verdict |
| --- | --- | --- |
| `GET /pet/status` + idempotent `Ensure` | `Service.Ensure` = `INSERT … ON CONFLICT (user_id) DO NOTHING` then `Get` | delivered |
| revive → health 20 | health **50** (§6.3) | authorised deviation (evaluation) |
| miss → −20 | **−30** (§8) | authorised deviation (evaluation) |
| `quests.TargetMetListener` | `quests.Pet` via `pet.QuestHook` | interface renamed by slice 3; correct |
| decay reads `daily_progress.is_target_met` | reads the **Redis** `daily:accumulated` counter through `StudyCounter` | **unauthorised** — see below |
| stage thresholds picked + in CODEMAP | 0–2/3–6/7–13/14+, in CODEMAP | delivered |
| tests: cap at 100, misses → wilted, 409, pass at 900 s, concurrent `Ensure` | all five present | delivered |

The one substitution neither the idea nor the evaluation sanctioned is the decay predicate. The idea lists `daily_progress` as a read table; the plan swapped it for the Redis counter to avoid pet touching quests' table. The boundary argument is good, but `RedisCounter.Total` maps a missing key to `0, nil`, so any counter loss is indistinguishable from "did not study" and the sweep takes 30 points off a user who met the target. Filed medium (`the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`) — this is the finding most likely to deserve promotion, because the durable answer already exists in `daily_progress`.

## Code vs plan

Ten tasks, ten commits, one per task, file-for-file matching the plan's *File structure* table. Nothing missing, nothing extra.

| Task | Commit | Files | Verdict |
| --- | --- | --- | --- |
| 1 store key | `f09527e` | `keys.go` `keys_test.go` | followed |
| 2 engine | `07c217c` | `engine.go` `engine_test.go` | followed |
| 3 repo/revive | `fd030d2` | `repo.go` `revive.go` | followed |
| 4 fakes | `6999697` | `fakes_test.go` | followed |
| 5 service/hook | `51de820` | `service.go` `questhook.go` `service_test.go` | followed |
| 6 cron | `0ff846b` | `cron.go` `cron_test.go` | followed |
| 7 handlers | `3a3d399` | `handler.go` `handler_test.go` | followed |
| 8 integration | `873615f` | `integration_test.go` | followed |
| 9 main.go | `d598cd8` | `cmd/api/main.go` | followed verbatim |
| 10 CODEMAP | `f6e27b1` | `harness/CODEMAP.md` | followed verbatim (diffed byte for byte against the plan's Task 10 blocks — identical) |

### Re-run verification (worktree, clean shell)

```
$ go build ./... && go vet ./... && echo BUILD_VET_OK
BUILD_VET_OK

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 120s
ok  internal/airouter 0.783s   ok  internal/auth 1.540s   ok  internal/config 0.896s
ok  internal/health 2.073s     ok  internal/pet 2.683s    ok  internal/quests 3.877s
ok  internal/store 3.205s
```

Every grep assertion in the plan's *Verification* section reproduced, including the three that matter for the split: `ApplyTargetMet` has exactly one non-test call site (`service.go:57`, inside `OnTargetMet`, never in `Sweep`); `daily:accumulated`/`DailyAccumulatedKey` has **0** hits in `internal/pet/`; `NopPet` has **0** hits in `cmd/api/main.go`.

Integration suite against a scratch stack (`COMPOSE_PROJECT_NAME=petrev`, ports 5440/6388; the owner's 6379/6380 containers untouched, confirmed with `docker ps` before and after):

```
$ TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./... -count=1 -v -run Integration -p 1 -timeout 300s
--- PASS: TestIntegrationRateLimiterAllowsFiveThenBlocks
--- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState
--- PASS: TestIntegrationEnsureCreatesExactlyOnePetRow
--- PASS: TestIntegrationDailyAndProgressAgainstRealServices
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser
--- PASS: TestIntegrationRedisRoundTrip
SKIP count: 0
```

### Runtime proof re-run (live Postgres + Redis, real JWT + `sess:` session, API on 8097)

Two real users, HS256 tokens minted against the running `JWT_SECRET`, sessions written as `auth.RedisSessionStore.Put` writes them. Full transcript:

```
1. GET /pet/status twice on a fresh user
   both -> 200 {"plant_name":"My Green Buddy","stage":"sprout","health_points":100,"current_streak":0,"last_practiced_at":null}
   SELECT count(*) FROM pet_states -> 1
   updated_at after two GETs = 2026-09-23 02:16:20.383915+00   (the INSERT default — unchanged by either read)

2. POST /quests/progress 900 / 1800 / 2700 s
   900s  -> {"daily_seconds_spent":900,"daily_minutes_spent":15,"is_target_met":false,"pet_health":100,"streak_count":0}
   1800s -> {"daily_seconds_spent":1800,"daily_minutes_spent":30,"is_target_met":true,"pet_health":100,"streak_count":1}
   2700s -> {"daily_seconds_spent":2700,"daily_minutes_spent":45,"is_target_met":true,"pet_health":100,"streak_count":1}   <- no second bump
   GET /pet/status -> {"plant_name":"My Green Buddy","stage":"sprout","health_points":100,"current_streak":1,"last_practiced_at":"2026-09-23T02:16:20.750202Z"}
   updated_at now = 2026-09-23 02:16:20.750202+00

3. POST /pet/revive while health 100 -> 409 {"error":"pet_not_wilted"}

4. wilted user: revive #1 -> 200 {"revival_passed":false,"pet_state":{"health_points":0,"stage":"wilted","current_streak":0}}
   899 s recorded, revive #2 -> 200 {"revival_passed":false,…}        <- boundary, not in the executor's proof
   900 s recorded, revive #3 -> 200 {"revival_passed":true,"pet_state":{"health_points":50,"stage":"sprout","current_streak":0}}
   db -> 50|sprout|0

5. bad bearer token -> 401 {"error":"unauthorized"}
```

Everything the executor claimed reproduces; the 899 s boundary and the "third progress call does not re-bump" case are additions of mine and both behave. **No executor gate failure.** Cleanup verified: `pgrep -fl exe/api` → none, `docker compose -p petrev down -v`, no `petrev` volumes, scratch `.env` removed, worktree `git status --short` clean.

### Recovery check — the three untrailered commits

Confirmed and benign.

```
f09527e trailer=1   07c217c trailer=1   fd030d2 trailer=1   6999697 trailer=1
51de820 trailer=1   0ff846b trailer=1   3a3d399 trailer=1
873615f trailer=0   d598cd8 trailer=0   f6e27b1 trailer=0
```

Content is coherent with Tasks 8–10 and **nothing was lost**: `873615f` carries exactly `integration_test.go` (Task 8), `d598cd8` exactly `cmd/api/main.go` (Task 9) whose diff is the plan's Task 9 code block verbatim, `f6e27b1` exactly `harness/CODEMAP.md` (Task 10) whose pet bullet and store sentence are byte-identical to the plan's prescribed text (machine-diffed). The only defect is the missing trailer, which the executor documented and correctly declined to fix by rewriting history.

## Quality

**1. §8 arithmetic and the §6.2-vs-§8 split — correct, but the once-ness is borrowed.** `ApplyTargetMet` appears in exactly one non-test call site and it is not in `Sweep`; the cron applies only `ApplyMiss`. There is no hook-plus-cron path that bumps twice, and a user who met the target is spared by the `total >= TargetSeconds` check (verified on the wire: the 2700 s call did not re-bump). But `Service.OnTargetMet` takes `localDate` and discards it — the once-per-day property lives entirely in quests inferring a rising edge on a 48-hour Redis counter. Lose that counter mid-day and the next 1800 s bumps a second time, +40 and two streak days for one calendar day. `LastPracticedAt` is already stamped and `localDate` is already passed; a three-line guard would make pet self-idempotent. Filed medium.

The `updated_at` guard is honest about the two cases in the question: a **second cron run in the same local hour** is a no-op *sequentially* (`Save` stamps `updated_at = now ≥ midnight`, proved by `TestSweepIsIdempotentWithinTheSameLocalDay`) but **not concurrently** — the guard is a read from `SweepCandidates` and the `UPDATE` has no `updated_at` predicate, so two sweepers apply −60. Filed medium. A **`GET /pet/status` earlier in the day does not touch `updated_at`**: `Ensure` is `ON CONFLICT DO NOTHING` and `Get` is a `SELECT`, confirmed empirically above (`updated_at` after two GETs was still the INSERT timestamp). That path is clean.

**2. Cron correctness at scale and at boundaries.**
- *Half-hour zones are fine, and the plan's DST note is wrong.* I enumerated the local hour for all 24 UTC `:00` samples per zone: Kolkata (+5:30), Kathmandu (+5:45), Chatham and Lord Howe each get exactly one hour-0 hit per day, up to 45 minutes late as the plan says. Havana's fall-back gives **two** hits and the second is correctly suppressed. But **America/Santiago on 2026-09-06 gets zero hits** — that zone springs forward *at midnight*, so local hour 0 never exists and the whole zone is skipped for the day. The plan asserts "On spring-forward there is still an hour 0". Filed medium.
- *Yesterday is the user's yesterday.* `midnight.AddDate(0,0,-1).Format("2006-01-02")` in the user's `*time.Location`, matching `store.DailyAccumulatedKey`'s formatting. Correct, including across transitions.
- *Invalid/empty timezones are safe.* `COALESCE(timezone,'UTC')` handles NULL; `quests.Location` maps `""` and any unloadable name to UTC; the selector and `SweepCandidates` filter on the same raw string, so they cannot disagree.
- *A user created mid-day is penalised* for the fraction of the day they were present (sign up 23:50, −30 at 00:00). Spec-literal, so not filed on its own, but no test covers it and the fake cannot express it (below).
- *The sweep is not bounded.* `SweepCandidates` has no `LIMIT` and materialises every pet in the zone — at `00:00 UTC` that is the whole user base, since `'UTC'` is the DDL default. `COALESCE(u.timezone,'UTC') = ANY($1)` is non-sargable and there is no index on `users.timezone`, so both sweep queries full-scan `users ⋈ pet_states` every hour regardless of whether any zone is at midnight. And `quests.Location` is called *inside* the candidate loop while Go's `time.LoadLocation` is uncached — measured 12.2 µs and one fresh `*time.Location` per call (50 000 calls in 609 ms, `l == l2` false). Filed medium.
- *A single user's error does not abort the sweep.* Errors are collected into `errs` and joined; `RunHourly` logs and continues. Correct — though the test that claims to prove it cannot (below).

**3. Revive semantics.** `start_seconds` is read from the same counter and the same local date the pass check reads (`study.Total(ctx, userID, today)` at start, `c.LocalDate` at check, and the branch guarantees they are equal) — correct. Two concurrent calls are safe in both directions: on the start path the last `HSET` wins and both return `passed:false`; on the pass path `ApplyRevive` is absolute rather than additive, so a double apply is idempotent at 50. Redis down → 500 from `challenges.Get` or `study.Total`, which is right. Key expiry mid-challenge loses the baseline and the user restarts from their *current* total — so a user who has already banked 1200 s that day needs 2100 s in total; undocumented, but bounded and benign.

Two real gaps. **Meeting the normal 30-minute target while a challenge is open answers 409 and orphans the key for its full 24 h** — reproduced live: `revive #1` starts, 1800 s recorded, `pet_health` goes to 20 via the hook, `revive #2` → `409 pet_not_wilted`, `EXISTS pet:revive:…` → 1, `TTL` → 86400. A polling client never terminates. Filed medium. And **a passed revival is undone the same night**: 900 s of challenge is less than the 1800 s the sweep requires, so health 50 becomes 20 at local midnight — spec-literal under §8, but it cancels §6.3's "resets health to 50%", and the outcome differs by up to 24 h of health depending on whether the user revived before or after that hour's sweep. Filed medium; the owner may want to promote it, since it is the one finding a real user would notice.

**4. `quests.Pet` contract.** The previous review's question is answered: `QuestHook.State` calls `Service.Ensure`, so a user with no `pet_states` row gets one created and reads back the §3.2 defaults (100/0) — the same values `NopPet` reported, so the §6.2 `pet_health`/`streak_count` shape is continuous across the swap, and `State` is read after the hook so the response carries post-bump values. Verified on the wire. Worth noting in passing that `State` is a read-shaped method with a write side effect (a `GET /quests/daily` for a brand-new user inserts a `pet_states` row); it is harmless and arguably the right place for it, but the interface's doc comment does not say so.

**5. §6.3 wire contract — verified on the wire, not by grep.** Both bodies are byte-identical to the spec's examples. `GET /pet/status` → `plant_name`, `stage`, `health_points`, `current_streak`, `last_practiced_at` (explicitly `null`, not omitted, on a fresh pet — `handler_test.go` asserts the literal `"last_practiced_at":null`). `POST /pet/revive` → `revival_passed` plus nested `pet_state{health_points, stage, current_streak}`, 200 on both start and pass, **409** `pet_not_wilted` above 0 health, 400 `invalid_request` on malformed JSON, 401 `unauthorized` with no session. `answers` is accepted and ignored, which §6.3 permits since it never says how answers are graded. Nothing here that a frontend would build on is wrong.

**6. Boundaries.** `pet` owns `pet_states` alone — `grep` for `daily:accumulated`/`DailyAccumulatedKey` in `internal/pet/` returns nothing, the counter is reached only through the `StudyCounter` interface, and the revive key is built only via `store.PetReviveKey`/`store.PetReviveTTL` (3 hits in `revive.go`, no literal anywhere). `pet` imports `quests`; `quests` never imports `pet`. The one violation is the expected one: `Repo.Timezone`/`Timezones`/`SweepCandidates` read `users.timezone` directly — three more queries against `auth`'s table, widening the already-open `harness/ideas/_inbox/quests-reads-the-users-table-directly-for-the-timezone.md` (medium). I did not file a duplicate; that bug's fix (an `auth.ProfileReader` both packages consume) now has two callers and should be re-priced accordingly. The plan's *Notes* call this out honestly.

**7. Tests, CODEMAP, idiom.** The suite is above this repo's average: `handler_test.go` compares whole §6.3 bodies byte for byte rather than spot-checking fields, the clock is injected through `NewService`, and `TestIntegrationEnsureCreatesExactlyOnePetRow` is a real test — eight concurrent `Ensure` calls against live Postgres, then a round trip through `Save` that exercises the `pet_stage` cast down to `wilted`, then `Timezones`/`SweepCandidates`. It would fail if `ON CONFLICT DO NOTHING` were dropped.

Four honest gaps, all in the sweep: `TestSweepSkipsAUnreadableCounterAndContinues` seeds **one** candidate, so the `continue` it is named for is never exercised — delete it and the test stays green; `fakeRepo.saveErr` exists and is never set, so the `Save` error branch is uncovered; `fakeRepo.Ensure` stamps a **zero** `UpdatedAt` where the DDL stamps `CURRENT_TIMESTAMP`, despite a comment claiming it "mirrors the §3.2 column defaults", which makes the newly-created-pet sweep case inexpressible; and only whole-hour zones are tested. Plus the handlers' 401 branch and `Repo.Timezone`'s error path are unreachable from the tests. Filed low.

CODEMAP is accurate and unusually specific — I checked its every factual claim against the code and found one imprecision: "a pet whose `updated_at` is already past that local midnight is skipped, which makes a re-run in the same hour idempotent" is true only for a sequential re-run (see finding 1). "(also called by onboarding)" describes a slice that does not exist yet. Go idiom is clean throughout: `min`/`max` builtins, `errors.Join` for the per-user errors, `var _ Repo = (*PgRepo)(nil)` assertions on all three implementations, `pgx.ErrNoRows` translated to a package error, `fmt.Errorf("pet: …: %w", err)` wrapping consistent with `quests`. Handlers answer 500 without logging, which is silent but matches `internal/quests/handler.go` exactly, so it is a codebase-wide gap rather than this slice's.

**Outside the pet package.** Task 9 added `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` to give the cron a cancellable context but left `r.Run()` blocking with no `http.Server`/`Shutdown`. Registering the handler *replaces* Go's default terminate-on-signal, so the API now ignores SIGINT and SIGTERM entirely — `Ctrl-C` in dev does nothing and every Railway deploy burns the full kill grace period while the replica still serves traffic with a dead cron. `main` has no signal handling at all, so this is a regression introduced by this branch, not the pre-existing condition; it is the other half of the low-priority `cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md`, whose *Expected output* asked for both pieces together. Filed medium. The executor hit it during the runtime proof and recorded it, but scoped it as pre-existing, which is not quite right.

## Bugs filed

All nine are ordinary inbox bugs. None blocks the merge.

| Priority | Bug |
| --- | --- |
| medium | `sweep-reads-updated-at-then-writes-unconditionally-so-two-sw.md` — the idempotency guard is read-then-write, so concurrent sweepers apply −60 |
| medium | `zones-that-skip-local-midnight-on-spring-forward-are-never-s.md` — `Hour()==0` never matches in midnight-DST zones; the plan's note is wrong |
| medium | `the-hourly-sweep-loads-every-pet-in-a-zone-into-memory-and-r.md` — unbounded read, non-sargable predicate, no index, uncached `LoadLocation` per user |
| medium | `service-ontargetmet-ignores-localdate-so-pet-has-no-idempote.md` — once-per-day lives only in a volatile Redis counter edge |
| medium | `meeting-the-daily-target-during-a-revive-challenge-answers-4.md` — 409 instead of a pass, and the Redis key leaks 24 h |
| medium | `a-passed-revival-is-knocked-from-50-back-to-20-by-the-same-l.md` — §6.3's 50 % and §8's −30 cancel within one local day |
| medium | `the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — the durable `daily_progress.is_target_met` is ignored; a lost counter penalises a user who met the target |
| medium | `main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` — the API now ignores SIGINT/SIGTERM (regression from Task 9) |
| low | `pet-sweep-tests-cannot-fail-on-the-branches-they-are-named-f.md` — the "continues" test has one candidate; `saveErr` unused; fake `updated_at` diverges from the DDL |

## Verdict

**pass-with-bugs.** The plan is delivered task for task, the §6.2-vs-§8 split is implemented exactly as decided and holds under every path I could construct, both §6.3 contracts are byte-correct on the wire, the boundaries hold (`pet` owns `pet_states`, reaches the counter only through `StudyCounter`, never builds a Redis key by hand), and the executor's Runtime proof reproduces in full with two boundary cases added. The three untrailered commits carry exactly the Task 8–10 content, byte-identical to the plan where the plan prescribed text; nothing was lost.

Nine bugs, none blocking. The three worth the evaluator's attention first are the volatile-Redis decay predicate and the concurrent-sweep race (both can take health off a user who earned it, under conditions the code does not defend against), and the SIGTERM regression (it is outside the pet package but the branch introduced it, and the fix is a ten-line amendment to a file every later slice extends).

Merge command: `/harness merge harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md`
