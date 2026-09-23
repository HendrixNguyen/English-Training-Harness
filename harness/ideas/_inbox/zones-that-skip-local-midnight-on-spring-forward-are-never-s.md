---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
---
# Zones that skip local midnight on spring-forward are never swept that day

## Why
`Service.Sweep` decides who "hit midnight" by sampling the local **hour** at each `:00` UTC:

```go
// backend/internal/pet/service.go:133-137
for _, tz := range zones {
        if now.In(quests.Location(tz)).Hour() == 0 {
                atMidnight = append(atMidnight, tz)
        }
}
```

In a timezone whose DST transition happens **at midnight**, local hour 0 does not exist on the
spring-forward day — the clock goes 23:59:59 → 01:00:00 — so no `:00` UTC sample ever satisfies
`Hour() == 0` and **every user in that zone is skipped for the whole day**. The previous day's
miss penalty is silently never applied.

Measured over all 24 UTC `:00` samples per day (Python `zoneinfo`, same tzdata Go reads):

```
America/Santiago  2026-09-06: 0 hit(s) []          <-- spring forward at 00:00, no local hour 0
America/Santiago  2026-09-23: 1 hit(s) ['03Z->09-23 00:00']
America/Havana    2026-11-01: 2 hit(s) ['04Z->11-01 00:00', '05Z->11-01 00:00']
Asia/Kolkata      any:        1 hit(s) ['19Z->00:30']    (half-hour offsets are fine)
Asia/Kathmandu    any:        1 hit(s) ['19Z->00:45']
Pacific/Chatham   any:        1 hit(s) ['…->00:45']
```

So the two half-hour/quarter-hour cases the plan worried about are **correct**, and the case the
plan declared safe is the broken one. The plan's *Notes* state: "On spring-forward there is still
an hour 0. Not tested." That is false for Chile, Cuba, Lebanon, Paraguay, Iran and the other zones
that transition at midnight — and the note is what a later maintainer will trust.

The fall-back double-hit (Havana, 2 hits) *is* handled: `time.Date(y,m,d,0,0,0,0,loc)` resolves the
ambiguous midnight to its first occurrence, so the `updated_at` guard makes the 05Z pass a no-op.

Impact is leniency, not harm — affected users keep 30 health points they should have lost once a
year — but it is an undetectable, untested hole in the one piece of arithmetic §8 specifies, and
the same hour-sampling shape will be copied by the `notify` slice's cron.

## Expected output
The sweep selects on the local **date** having changed, not on the local hour being 0, so it fires
exactly once per local day in every zone including ones with no midnight:

- `pet_states` gains a `last_swept_date DATE` (or the sweep compares `updated_at`'s local date to
  today's local date) and a zone is swept when the user's local date is greater than the last one
  swept — which is true at 01:00 local on a spring-forward day just as it is at 00:00 otherwise;
- alternatively keep the hourly tick but accept any local hour `< 2` combined with the existing
  `updated_at < local midnight` guard, which already suppresses the second pass;
- a table test in `service_test.go` runs `Sweep` at every `:00` UTC across
  `America/Santiago` 2026-09-06 (spring forward at midnight), `America/Havana` 2026-11-01 (fall
  back), `Asia/Kolkata` and `Pacific/Chatham`, and asserts each user is penalised exactly once per
  local day;
- the plan's and CODEMAP's DST sentences are corrected.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md`
  (*Notes and open questions* — "**DST.** … On spring-forward there is still an hour 0. Not tested.").
- `backend/internal/pet/service.go:133-137` — the `Hour() == 0` selector.
- `backend/internal/pet/service.go:152` — `time.Date(…, 0,0,0,0, loc)`, which is what makes the
  fall-back case safe and the spring-forward case unreachable.
- Reviewer probe, 2026-09-23, enumerating the local hour for all 24 UTC `:00` samples per zone/day
  (output quoted above).
- Backend spec §8 — "Runs at :00 UTC every hour to detect users hitting midnight in their local
  timezone".

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Real and well-evidenced, but the effect is leniency (30 health kept) once a year in a handful of zones. Selecting on local date rather than `Hour() == 0` fits the pet day-judgement plan; include it there with the zone table test.

**Planned (2026-09-23):** `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — decision 3, Task 6 (civil-date selection replaces `Hour() == 0`; zone table test incl. America/Santiago 2026-09-06).
