---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# A passed revival is knocked from 50 back to 20 by the same local day's miss penalty

## Why
The revival challenge costs 15 minutes (`ReviveSeconds = 900`, from the §7 "15-minute revival
challenge"). The daily target is 30 minutes. So a user can pass the challenge and still miss the
day — and `Sweep` will take the §8 penalty out of the health the revival just restored.

Timeline, entirely from the shipped code:

```
20:00 local  POST /pet/revive passes   -> ApplyRevive: health 50, sprout, streak 0
                                          Save stamps updated_at = today 20:00       (service.go:107-110)
00:00 local  Sweep: updated_at < local midnight, so not skipped                      (service.go:153)
             study.Total(user, yesterday) = 900 (the challenge) < TargetSeconds 1800 (service.go:157-164)
             ApplyMiss -> health 20, streak 0                                        (engine.go:74-80)
```

Four hours after being told their plant was restored to 50 %, the user opens the app to 20 % — and
if they revived late in their local evening it can be minutes. One more missed day wilts it again,
so the 15 minutes bought them a single day instead of the "reset to 50 %" §6.3 promises.

Strictly, §8 says `daily time < 1,800 seconds → Health = Max(0, Health - 30)` with no exception, so
the arithmetic is spec-literal and this is not an executor defect. But §6.3 and the 1st-thinking §7
describe revival as the way out of a wilted plant, and the two rules as implemented cancel each
other within one local day. The spec does not reconcile them; the code silently picks the harsher
reading, and neither the plan's *Notes* nor CODEMAP mentions the interaction, so a maintainer
reading either would not expect it.

The mirror case is also unhandled: a user who revives at 00:10 local — *after* that hour's sweep —
keeps the full 50 until the next midnight. The outcome of an identical action differs by up to 24 h
of health depending on the wall clock, which is the tell that the two rules are not composed.

## Expected output
Passing the revival protects the plant for the local day it was passed:

- `ApplyRevive` (or `Service.Revive`) records that the day was resolved — the simplest form is to
  set `LastPracticedAt = now` alongside the health reset, and have `Sweep` skip a pet whose
  `last_practiced_at` falls on the local day being judged, in addition to the existing
  `updated_at` guard;
- the owner decides and the plan records which reading wins: either a passed revival exempts that
  day from `ApplyMiss`, or the revival is explicitly a partial-credit top-up that the night's
  penalty may reduce. Either is defensible; leaving it implicit is not;
- `service_test.go` gains a test that revives a wilted pet and then runs `Sweep` at the following
  local midnight, asserting the decided outcome (50 under the exemption reading);
- CODEMAP's pet bullet states the interaction in one clause, so the next reader of
  `ApplyRevive` knows what happens at midnight.

## Evidence
- Plan: `harness/plans/2026-09-22-pet-health-streak-and-stage-engine-with-revive.md` (Task 2
  `ApplyRevive`; Task 5 `Sweep`; *Notes and open questions*, which cover the streak reset, the
  half-hour zones and the wilted-plant recovery path but not this one).
- `backend/internal/pet/service.go:107-110` — the pass writes health 50 and stamps `updated_at`.
- `backend/internal/pet/service.go:153-164` — the sweep's only exemptions are `updated_at >= local
  midnight` and `yesterday's total >= 1800`; 900 s satisfies neither.
- `backend/internal/pet/engine.go:14-15` — `ReviveHealth = 50`, `ReviveSeconds = 900` vs
  `quests.TargetSeconds = 1800`.
- Backend spec §6.3 ("Resets health to 50% upon passing") vs §8 Inactivity Logic.
- 1st-thinking §7 — "15-minute revival challenge".

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Real and on the revive happy path: a user who revives in the evening sees 50 → 20 at their local midnight because the 15-minute challenge cannot also satisfy the 30-minute day. Both rules are spec-literal; the composition is undecided. Decision for the plan: a passed revival exempts that local day from `ApplyMiss` (`ApplyRevive` stamps `LastPracticedAt`; `Sweep` skips a pet whose `last_practiced_at` falls on the judged day). Belongs in the pet "durable day judgement" plan with `the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`, `service-ontargetmet-ignores-localdate-so-pet-has-no-idempote.md` and `ontargetmet-is-lost-forever-if-a-write-after-the-incrby-fail.md` — all four change how pet decides whether a day was met.
