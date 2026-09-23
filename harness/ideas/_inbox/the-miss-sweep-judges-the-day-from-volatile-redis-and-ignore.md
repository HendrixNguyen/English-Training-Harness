---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# The miss sweep judges the day from volatile Redis and ignores the durable daily_progress row

## Why
`Service.Sweep` decides whether a user met yesterday's target by reading the Redis counter and
nothing else:

```go
// backend/internal/pet/service.go:156-164
yesterday := midnight.AddDate(0, 0, -1).Format("2006-01-02")
total, err := s.study.Total(ctx, c.UserID, yesterday)
...
if total >= quests.TargetSeconds {
        continue
}
```

`StudyCounter.Total` is `quests.RedisCounter.Total`, which **maps a missing key to zero, not to an
error** (`backend/internal/quests/counter.go:54-58`: `if err == redis.Nil { return 0, nil }`). So
any way the key goes missing before the sweep reads it is indistinguishable from "this user did not
study", and the sweep takes 30 health points off a user who met the target:

- eviction under `maxmemory` (Redis's default `allkeys-lru` policies evict *any* key, and this one
  is a plain counter with no protection);
- a Redis restart without AOF/RDB, or a failover to an empty replica — spec §9 provisions Redis as
  a Railway plugin and says nothing about persistence;
- an operator `FLUSHDB` during an incident;
- a counter written just before a 48 h TTL boundary in an unusual clock/timezone combination
  (`store.DailyAccumulatedTTL = 48 * time.Hour` is comfortable, but it is the *only* margin).

Postgres already holds the durable answer. `POST /quests/progress` upserts
`daily_progress (user_id, date, minutes_spent, is_target_met)` on every call
(`backend/internal/quests/service.go:113`), and `is_target_met` is exactly the predicate the sweep
needs. The slice's own idea listed it: *"Tables: `pet_states` (read/write), `daily_progress`
(read)"* and *"for users whose previous day has no `daily_progress.is_target_met = true`"*. The
plan substituted the Redis counter — a reasonable call for boundaries, since `daily_progress` is
quests' table and the `StudyCounter` interface keeps pet out of it — but it traded a durable source
for a volatile one without noting the failure mode anywhere.

The harm is asymmetric and silent: an over-penalised user sees their plant drop 30 points (or wilt)
with no event to correlate it to, and nothing in the system can reconstruct whether the penalty was
justified.

## Expected output
The sweep judges a day from durable state, still without pet reading quests' table:

- the `StudyCounter` interface (or a sibling `DayResult` interface) gains a method backed by
  `daily_progress` — e.g. `MetTarget(ctx, userID, localDate string) (met bool, recorded bool, err
  error)` — implemented in `quests` over `SELECT is_target_met FROM daily_progress WHERE user_id =
  $1 AND date = $2`, so the boundary rule is preserved and pet still owns only `pet_states`;
- `Sweep` skips the pet when `met` is true. When `recorded` is false **and** the Redis counter is
  also absent, the sweep treats the day as genuinely unstudied (the current behaviour) — that
  distinction is what makes the change safe rather than merely different;
- the interaction with the known gap
  `ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md` is stated: if the upsert failed
  on the crossing call, `is_target_met` is stale-false, so the Redis total stays the tiebreaker and
  either source meeting 1800 s spares the user;
- `service_test.go` covers: counter present and ≥ 1800 (spared), counter missing but
  `daily_progress.is_target_met = true` (spared — this is the regression test), both absent
  (penalised);
- CODEMAP's pet bullet names both sources instead of only `daily:accumulated`.

## Evidence
- Idea: `harness/ideas/2026-09-22-run-02/pet-health-streak-and-stage-engine-with-revive.md`
  (*Expected output* — "for users whose previous day has no `daily_progress.is_target_met = true`";
  "Tables: `pet_states` (read/write), `daily_progress` (read)").
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 5 —
  the substitution to `StudyCounter`; not listed in *Notes and open questions*).
- `backend/internal/pet/service.go:156-164` — the counter is the sole predicate.
- `backend/internal/quests/counter.go:54-58` — `redis.Nil` → `0, nil`, so absence reads as zero.
- `backend/internal/quests/service.go:113` — the durable `daily_progress` upsert that exists today.
- `backend/internal/store/keys.go:14` — `DailyAccumulatedTTL = 48 * time.Hour`.
- Backend spec §8 Inactivity Logic; §3.2 `daily_progress`; §9 Railway Redis plugin.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium (survivor of the pet day-judgement group).** Confirmed: `Sweep` reads only `study.Total` and `redis.Nil` reads as 0, so any counter loss penalises a user who studied — silently, unrecoverably. Postgres already holds `daily_progress.is_target_met`. Plan: widen `StudyCounter` with `MetTarget(ctx, user, localDate)` implemented in `quests` over `daily_progress`; spare when either source says met; **same plan** also takes `service-ontargetmet-ignores-localdate…`, `ontargetmet-is-lost-forever…`, `a-passed-revival-is-knocked…`, the conditional miss `UPDATE` (`sweep-reads-updated-at…`), the zone-location map (`the-hourly-sweep…`) and the sweep tests (`pet-sweep-tests…`). One pet branch, one merge.
