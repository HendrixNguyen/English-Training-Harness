---
plan: harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/ontargetmet-s-read-then-write-erases-a-concurrent-sweep-s-mi.md, harness/ideas/_inbox/a-multi-day-sweep-outage-collapses-every-missed-day-into-one.md, harness/ideas/_inbox/get-quests-daily-still-reads-is-target-met-from-the-volatile.md, harness/ideas/_inbox/pgrepo-save-resets-last-target-met-date-to-null-on-the-reviv.md, harness/ideas/_inbox/testtwoconcurrentsweepspenaliseonce-misses-its-defect-in-1-r.md, harness/ideas/_inbox/fakeprogressrepo-upsert-s-monotonic-max-is-not-covered-by-an.md, harness/ideas/_inbox/marktargetmet-ignores-rowsaffected-so-a-missing-row-silently.md, harness/ideas/_inbox/gofmt-l-has-been-failing-on-two-internal-quests-files-since-.md]
---
# Review — pet: own the day's verdict — durable once-per-day success, a civil-date sweep, and a revival that resolves its day

**Plan:** `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`
**Branch/worktree:** `harness/2026-09-23-medium-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore` / `.worktrees/the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore`
**Diff:** `git diff main...harness/2026-09-23-medium-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore --stat`

## Plan vs idea

Seven ideas were planned here; all seven *Expected output* sections are delivered.

| Idea | Delivered |
| --- | --- |
| `the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore` | Yes, by a **better mechanism than the idea proposed**. The idea asked pet to read `daily_progress` through a new `quests`-implemented interface; the plan's design decision 1 rejected that (CODEMAP boundary + `cmd/api/main.go` would have to be touched) and gave pet its own `pet_states.last_target_met_date` / `judged_through`. The deviation is argued in the plan, and `grep -rn daily_progress internal/pet/` is empty — the boundary holds. |
| `service-ontargetmet-ignores-localdate-so-pet-has-no-idempote` | Yes, and more strictly. The idea asked for a Go-side early return on `LastPracticedAt`'s local date; the code makes it a SQL predicate (`last_target_met_date IS NULL OR last_target_met_date < $d`) with no Go pre-check at all. Both the "bumps once" and the "cannot be satisfied by never bumping" tests exist, plus a third for an *earlier* date. |
| `ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail` | Yes, via the idea's first option — `Upsert` `RETURNING` the durable flag, hook fired on that, flag set only after the hook returns nil. The asked-for replay test exists (`TestAFailedUpsertOnTheCrossingCallIsRetriedByTheNextCall`, plus two siblings for `MarkComplete` and the hook). |
| `a-passed-revival-is-knocked-from-50-back-to-20-by-the-same-l` | Yes. The idea explicitly left the product question to the owner; the owner ruled, and the code implements the ruling (see *The revival rule* below). |
| `sweep-reads-updated-at-then-writes-unconditionally-so-two-sw` | Yes. `PenaliseMiss` is one conditional `UPDATE` with the arithmetic in SQL, the count follows `RowsAffected() == 1`, and both a fake-level concurrent test and a real-Postgres 8-writer test exist. |
| `zones-that-skip-local-midnight-on-spring-forward-are-never-s` | Yes, by the idea's first shape (a DATE column, selection on the local **date**). The asked-for table test runs 72 hourly ticks across all four named zones plus three more. |
| `pet-sweep-tests-cannot-fail-on-the-branches-they-are-named-f` | Yes — per-user `errFor`/`saveErrFor`, a clock in `fakeRepo.Ensure`, the first-contact pair, the 7-zone table test, and the `newPetRouter` no-user variant with both 401 assertions. |

## Code vs plan

**Scope.** `git diff main..HEAD --stat -- backend/cmd/api/main.go` is **empty**. Confirmed.

**Tasks 1-8 plus one unplanned commit — all followed.** 9 commits, one per task plus
`store: update integration test expectations for migration 0003`.

Two deviations, both reported by the executor and both correct:

1. *Task 7's monotonicity check moved to the end of `TestIntegrationDailyAndProgressAgainstRealServices`,
   expecting 41 minutes not the plan's illustrative 40.* Correct — the plan's literal insertion point sits
   above the test's own "Review reproduction" sections, which assert `total==1800` and
   `daily_progress==(30,true)` are untouched. Placing it there would have broken them. 41 is right for the
   1860s+600s starting state.
2. *`backend/internal/store/integration_test.go` (not in the plan's file table) hardcoded the migration list
   and count.* The fix is correct and minimal — two constants, plus a comment explaining why `0003`'s down
   is **not** added to `reset()`. I verified that reasoning: `0001_init.down.sql:6` is
   `DROP TABLE IF EXISTS pet_states`, which takes `0003`'s columns with it, and `0003`'s `ALTER TABLE` has
   no `IF EXISTS` on the table, so calling it against an already-empty database would error. The hardcoding
   was load-bearing for nothing else — `grep` finds no other hardcoded migration list.

### Verification re-run (my own, in the worktree)

```
go build ./...                                            -> BUILD OK
go vet ./...                                              -> VET OK
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL \
  -u TEST_REDIS_URL go test ./... -count=1 -race -timeout 300s
  -> ok for all 10 packages (airouter auth config google health notify
     onboarding pet quests store), no race reports
```

Named groups, all PASS:
* `-run OnTargetMet` — 5/5 incl. `TwiceForTheSameLocalDateBumpsOnce`, `ForTheNextLocalDateBumpsAgain`, `ForAnEarlierLocalDateIsIgnored`
* `-run 'Sweep|TwoConcurrent'` — 11 tests + 7 zone subtests (UTC, Ho_Chi_Minh, Kolkata, Kathmandu, Chatham, Havana, Santiago)
* `-run 'Hook|Flag|Crossing|Refire|Retried|MarkComplete'` — 10/10

Every grep assertion in the plan's *Verification* holds: no `Hour() == 0`, no
`time.Date(local.Year()`, **no `time.Date` anywhere in non-test `internal/pet`**, no
`total-seconds < TargetSeconds`, conditional-write count `3`, `GREATEST` in both repos, no
`daily_progress` in `internal/pet/`, no `Timezones`, spec/migration `ADD COLUMN judged_through DATE;`
`1` and `1`.

**Integration, live stack** (`COMPOSE_PROJECT_NAME=petrev`, Postgres 5455, Redis 6403 — torn down with
`docker compose down -v`, containers *and* volume confirmed gone, my scratch `backend/.env` deleted,
worktree left clean):

```
make test-integration -> ok for every package, no SKIPs
  --- PASS: TestIntegrationVerdictWritesAreConditional
  --- PASS: TestIntegrationEnsureCreatesExactlyOnePetRow
  --- PASS: TestIntegrationDailyAndProgressAgainstRealServices
  --- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
  --- PASS: TestIntegrationConcurrentMigrateDoesNotRace
```

**CI.** `gh run view 35851715543` — `completed / success` on head SHA `64950f1`, which is the branch head
and matches `@{u}`. `frontend`, `harness-tooling`, `backend-unit`, `backend-integration` all `success`.
Not a blocker.

**`gofmt -l`** flags `internal/quests/handler_test.go` and `internal/quests/repo.go`. I verified against
`git show main:...` copies that **both are already unformatted on `main`** — the executor's claim is
accurate. Filed as a separate low bug since nothing in CI catches it.

## Quality

### The revival rule — matches the owner's decision exactly

`ApplyRevive` (`backend/internal/pet/engine.go:102-109`) sets health 50, streak 0, sprout, and
`JudgedThrough = laterDate(cur, localDate)` — and **does not touch** `LastTargetMetDate` or
`LastPracticedAt`. `Sweep` skips any day `<= judged_through`. `OnTargetMet`'s predicate is on
`last_target_met_date`, which the revival left alone, so the +20 is still available.

The plan's Notes described a one-line reversal (drop the `JudgedThrough` line, flip the test to expect 20).
**The code did not take it** — the line is present at `engine.go:106`.

`TestRevivePassResolvesTheLocalDayItWasPassedOn` pins all three halves of the ruling in one test:
`judged_through == 2026-09-22`, `LastTargetMetDate`/`LastPracticedAt` still nil, and then
`OnTargetMet` for the same day takes health 50 -> **70** with streak 1.
`TestSweepDoesNotPenaliseADayResolvedByARevive` covers both the 20:00 revive and the 00:10-next-morning
mirror case. This is exactly what the owner ruled.

### Concurrency — the idempotence claim holds; a second, narrower race does not

**Idempotence is genuinely a SQL predicate, not a Go pre-check.** `Service.OnTargetMet` has no guard at
all (`service.go:57-64`) — the whole once-ness is `saveTargetMetSQL`'s
`WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < $7::date)`.
`PenaliseMiss` and `MarkJudged` are the same shape on `judged_through`. `Sweep` has a Go pre-check
(`judged_through >= judged -> continue`) but it is a *read-ahead optimisation on top of* the predicate,
not a substitute — removing the repo's predicate is still caught (see the mutation section). The fakes
mirror the predicates exactly (`dateBefore`, `fakes_test.go:86-135`), including `SaveTargetMet` preserving
`cur.JudgedThrough` the way `saveTargetMetSQL`'s column list does.

All concurrent tests re-run under `-race`: clean. `TestTwoConcurrentSweepsPenaliseOnce` at
`-count=300` and `-count=100 -race`: clean, no false positives.

**What I did find** (filed, not blocking): `OnTargetMet` is still `Ensure` (read) -> `SaveTargetMet`
(write of an *absolute* `health_points = $2`). When the sweep's SQL-side `PenaliseMiss` lands between the
two, the -30 is **completely erased**. Proven against real Postgres:

```
pre-image read by OnTargetMet: health=100
PenaliseMiss 2026-09-22 applied=true -> health=70
SaveTargetMet(stale pre-image, 2026-09-23) applied=true -> health=100, streak=1
```

The plan's "Residual race, documented" note says the outcomes are 90/1 or 70/0. The real third outcome is
100/1 — the whole penalty, not 20 health. The once-per-day guarantee is unaffected (the date predicate still
holds and `judged_through` is still correct); it is the *value* that is lost. Same class on the revive path:
`saveSQL` writes `last_target_met_date = $7` unconditionally while protecting `judged_through` with
`GREATEST`, so a concurrent `OnTargetMet` can be rolled back to NULL. Both filed.

### DST and civil dates — correct, including the zones the plan did not name

**No midnight instant is constructed anywhere.** `grep -n 'time.Date' internal/pet/*.go` (non-test) is
empty; `PreviousDate` is pure string calendar arithmetic (`time.Parse` -> `AddDate(0,0,-1)` -> `Format`),
and `quests.LocalDate` is `now.In(loc).Format(...)`. Nothing can hit Go's undefined resolution of a
non-existent local midnight.

The 7-zone table test uses an **independent oracle** (`len(seen)-1`, the number of local dates the ticks
actually entered) rather than a constant, so a future tzdata change cannot rot it. It covers every zone the
lead asked about: Havana (hour 0 twice, 2026-11-01), Kolkata (+05:30), Kathmandu (+05:45), Chatham (+12:45),
Santiago (no hour 0, 2026-09-06), plus UTC and Ho Chi Minh.

**A timezone that changes between ticks** — not covered by the plan; I probed it. Moving
Kiritimati (+14) -> Midway (-11) makes the local date jump backwards a day; the monotonic
`judged_through >= judged -> continue` absorbs it with no double penalty. Correct. The opposite move skips
forward a day, which is the same forgiveness as the outage case below.

**A tick the process slept through** is caught up *within the same local day* — confirmed by
`TestSweepIsIdempotentAcrossTicksAndHours` and by the 01:00 Santiago case. **It is not caught up across
days**, and that is the one real gap: `Sweep` judges exactly one day per pet per tick and then sets
`judged_through = judged` outright, so every intervening day is marked judged without being judged. Probed:
`judged_through` seeded to `2026-09-18`, one sweep at `2026-09-23T00:00Z` -> `penalised=1, health=70,
judged_through=2026-09-22`. Four missed days, one -30, and the plant does not wilt. The doc comment at
`service.go:130-132` and the CODEMAP pet bullet both say the catch-up is general. Filed (medium); I left
CODEMAP alone because the wording should follow the owner's decision on whether to catch up or forgive,
not pre-empt it. Strictly more lenient than `main`, so not a regression.

**First contact** is right in both directions. `judged_through IS NULL && LocalDate(updated_at, loc) > judged`
-> `MarkJudged` only, no penalty; otherwise the day is judged normally.
`TestSweepFirstContactJudgesOnlyDaysThePetExisted` covers both sides (created 00:10 after the tick -> spared
and initialised; created 23:50 before it -> penalised). A pet cannot be retroactively penalised for days
before it existed, and a genuine miss on its first day is not skipped. The one-time migration artefact
(every pre-`0003` row has `last_target_met_date` NULL, so its first judged day is spared only by the Redis
counter fallback) is bounded by the counter's 48h TTL and self-heals after one sweep.

### Monotonicity in quests — the ordering claim and its converse both hold

`RecordProgress` (`quests/service.go:120-131`): `newlyMet := total >= TargetSeconds && !alreadyMet`, then
`if err := OnTargetMet(...); err != nil { log } else if err := MarkTargetMet(...); err != nil { log }`.

* **Failed hook leaves the day unflagged** — yes, `MarkTargetMet` is in the `else` branch and is
  unreachable on a hook error. `TestAPetHookFailureLeavesTheDayUnflaggedSoTheNextCallRetries` pins it.
* **The converse — can the flag be set without the hook succeeding?** No. `MarkTargetMet` has exactly one
  call site, reachable only when `OnTargetMet` returned nil. A day's reward cannot be silently lost.
  (`OnTargetMet` returning nil on `applied == false` is the intended no-op — pet already has the day.)
* `MarkComplete` moved *below* the hook, so a vanished exercise (404) no longer costs the +20. A 404 now
  carries a silently-applied pet bonus — deliberate, documented in the code comment, and the minutes were
  already counted by the INCRBY either way.
* `GREATEST` on `minutes_spent` and a flag that only ever goes TRUE: both correct in SQL. But
  `Service.Daily` was **not** updated and still derives `is_target_met` from the counter alone
  (`quests/service.go:247`), while `RecordProgress` uses `counter || durable flag` (`:121`). After a lost
  counter the two endpoints disagree about the same day — the plan's *File structure* row asked for
  "counter *or* durable flag" and only half of it landed. Filed (medium).

### Migration 0003

Exercised live against Postgres 16 on a **populated** `pet_states`:
* **Up on a populated table** — adds both columns as NULL, existing `health_points`/`current_streak`
  preserved. Nullable-column add, so no table rewrite.
* **Down is a true inverse** — drops exactly the two columns, leaves the other 8 and their data intact,
  and is idempotent (`IF EXISTS` -> `NOTICE`, no error). Running it twice is safe.
* Up is not re-runnable (`column ... already exists`), which matches `0001`/`0002`'s style and is fine —
  `schema_migrations` gates it.

**Spec DDL.** I diffed the spec's §3.2 `sql` block against `0001 + 0002 + 0003` concatenated,
statement-for-statement (comments and whitespace normalised): **identical**. The whole spec diff is
`+5 -0` — the `ALTER TABLE` and its comment, nothing else in the file moved.

### Test honesty — the three flagged mutation rows

All three re-run by me. The executor's reasoning is correct in every case; my judgement on whether the
coverage is real differs on one.

1. **Unconditional `PenaliseMiss` in the fake.** Reproduced: `TestSweepIsIdempotentAcrossTicksAndHours`
   stays **PASS** (its four `Sweep` calls are sequential, so `Sweep`'s own fresh-state pre-check blocks the
   second call before the repo predicate is reached — exactly the executor's diagnosis).
   `TestTwoConcurrentSweepsPenaliseOnce` goes **red** with 19 separate per-pet assertions
   (`health = 40, want 70`) on top of the count. **Coverage is real, not relocated** — the concurrent test
   asserts a *stronger* property (per-pet health) than the sequential one would, and
   `TestIntegrationVerdictWritesAreConditional` §2 pins the actual SQL with 8 concurrent writers. Agreed.
2. **`daily_progress` monotonicity (fake drops the `max`).** Reproduced: dropping it leaves the **entire
   `internal/quests` unit package green**, not just the named test. The equivalent SQL mutation
   (`minutes_spent = EXCLUDED.minutes_spent`) **does** fail the integration test
   (`daily_progress = (1, true) ... want (41, true)`). **Coverage here is genuinely relocated, not
   equivalent** — it moved from the always-run unit suite into a `TEST_DATABASE_URL`-gated test. CI runs
   `backend-integration`, so the branch is protected, but the fake's fidelity to the SQL is itself untested
   and will drift. Filed (low).
3. **A met day shielding an unjudged miss (`ApplyTargetMet` also advancing `JudgedThrough`).** Reproduced:
   `TestSweepStillJudgesYesterdayWhenTodaysTargetWasMetFirst` stays **PASS**, because neither
   `saveTargetMetSQL`'s column list nor the fake's `SaveTargetMet` persists `JudgedThrough` — the bug
   cannot reach storage, exactly as reported. `TestApplyTargetMetStampsTheLocalDateItWasAppliedFor` goes
   **red**. **Coverage is real** — a precise engine-level assertion in the same always-run unit suite.
   Agreed.

### The "count is honest" row — genuinely an unreliable detector, not a CI flake

Measured, not estimated. With the `applied` guard around `penalised++` removed:
**detected in 27 of 30 separate runs (90%).** Unmutated: `-count=300` clean, `-count=100 -race` clean —
**no false positives**, so it will not flake CI.

The mechanism is not "goroutine-scheduling variance" in general; it is specific and fixable. Nothing
synchronises the two goroutines, so when one finishes all 20 pets before the other calls
`SweepCandidates`, the second sweep skips every pet via `Sweep`'s own pre-check
(`service.go:160-162`) and never calls `PenaliseMiss` — no spurious increment, test passes.
**Yes, this is worth filing:** it is the *only* unit-level guard for the `PenaliseMiss`-unconditional
mutation too (row 1 above), and a backstop that misses one run in ten is not what the mutation table
claims. A `beforeWrite` hook on the fake plus a barrier makes it deterministic. Filed (low).

### Boundaries, conventions, CODEMAP

* **Boundaries hold.** `grep -rn daily_progress internal/pet/` is empty — pet never touches quests' table.
  Everything crosses through interfaces (`Repo`, `StudyCounter`, `ChallengeStore`, `quests.Pet`).
  Redis-first ordering in the daily loop is unchanged (INCRBY -> EXPIRE -> UPSERT -> hook -> flag ->
  MarkComplete), and `service_test.go`'s call-log test pins it.
* **Conventions.** The new SQL, the `applied bool` return shape, the fake mirroring and the doc-comment
  density all match the surrounding code. `PreviousDate` panics on a malformed date — unreachable
  (`LocalDate` always `Format`s) and the comment justifies it, so I am noting it rather than filing it.
* **CODEMAP** is updated for `store`, `quests` and `pet`, and is accurate apart from the shared
  "caught up at the next one" wording covered by the outage bug. I left it for that fix to correct, so the
  wording can follow the owner's decision.

## Bugs filed

All eight in `harness/ideas/_inbox/`. **None is a blocker** — no failing test, no data loss, no security
hole, no broken developer workflow. The three medium findings are either leniency in the user's favour or a
display inconsistency.

| Priority | Bug |
| --- | --- |
| medium | `ontargetmet-s-read-then-write-erases-a-concurrent-sweep-s-mi.md` — `OnTargetMet`'s absolute-value write clobbers a concurrent `PenaliseMiss`; the plan's residual-race note understates it (100/1, not 90/1). |
| medium | `a-multi-day-sweep-outage-collapses-every-missed-day-into-one.md` — one penalty per catch-up tick; intervening days are marked judged without being judged. Needs an owner decision (catch up vs forgive) either way. |
| medium | `get-quests-daily-still-reads-is-target-met-from-the-volatile.md` — `GET /quests/daily` and `POST /quests/progress` disagree about the same day after a lost counter. |
| low | `pgrepo-save-resets-last-target-met-date-to-null-on-the-reviv.md` — `saveSQL` protects `judged_through` with `GREATEST` but not `last_target_met_date`. |
| low | `testtwoconcurrentsweepspenaliseonce-misses-its-defect-in-1-r.md` — 27/30 detection; needs a barrier. |
| low | `fakeprogressrepo-upsert-s-monotonic-max-is-not-covered-by-an.md` — dropping the fake's `max` leaves the whole quests unit package green. |
| low | `marktargetmet-ignores-rowsaffected-so-a-missing-row-silently.md` — the one new write that cannot tell "done" from "nothing there". |
| low | `gofmt-l-has-been-failing-on-two-internal-quests-files-since-.md` — pre-existing on `main`; CI has no format check. |

## Verdict

**`pass-with-bugs`. Nothing blocks the merge.**

This is a strong piece of work and it does what it set out to do. The seven findings are genuinely closed:
the once-per-day guarantee is now an owned SQL predicate rather than a borrowed Redis edge; the sweep is
civil-date based with no midnight instant anywhere; `daily_progress` is monotonic; the flag-after-hook
ordering is correct in both directions; and the revival rule implements the owner's ruling exactly, pinned
by a test that asserts all three of its halves. The migration is a true inverse, the spec DDL is identical
to the migrations, `cmd/api/main.go` is untouched, CI is green on the branch head, and the executor's three
mutation-table reports and its `gofmt` claim all check out under re-execution.

What it did not close is the *second half* of its own design decisions in two places: the read-then-write
that the plan removed from the sweep is still present in `OnTargetMet` (and mis-described in the Notes), and
the durable-flag read that `RecordProgress` gained did not reach `GET /quests/daily`. Both are filed, both
are narrow, and neither is a reason to hold the branch.
