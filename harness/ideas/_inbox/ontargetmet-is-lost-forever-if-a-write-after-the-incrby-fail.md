---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md
---

# OnTargetMet is lost forever if a write after the INCRBY fails

## Why
`newly_met` is derived from the counter alone — `total >= 1800 && total-seconds < 1800`
(`backend/internal/quests/service.go:82-83`). That makes "fires exactly once" cheap and correct on
the happy path, and the once-only tests are genuine. But the derivation is *stateless*: the only
record that the crossing was handled is the Redis total itself, which is raised **before** the two
writes that can fail.

If `progress.Upsert` or `quests.MarkComplete` returns an error on the call that crossed 1800
(`service.go:85-90` both return early), the counter is already >= 1800. Every later call that day
therefore computes `total-seconds >= 1800` and `newlyMet == false`, so `Pet.OnTargetMet` never fires
for that user on that local day — no health bump, no streak increment, silently, with no log line
and no retry. The same holds for the bricked-counter case in the sibling bug: the hook is skipped
for good.

The plan reasons about the crash-between-Redis-and-Postgres case and concludes the system
"self-heals within the 48h TTL", which is true for `daily_progress.minutes_spent` (the next call
re-derives it from the total) but is **not** true for the hook — `newly_met` is a one-shot edge that
the re-derivation cannot reconstruct. CODEMAP repeats the self-healing claim without that caveat.
Today the only registered `Pet` is `NopPet`, so nothing is visibly lost; the pet slice (MVP order 4)
is what turns this into a lost streak.

## Expected output
The once-only trigger survives a failed write on the crossing call. Two workable shapes:
- derive "already handled" from a durable fact rather than an inferred edge — e.g. have
  `ProgressRepo.Upsert` report whether it flipped `is_target_met` from false to true (`RETURNING`
  plus a `WHERE daily_progress.is_target_met = FALSE` guard on the DO UPDATE), and fire the hook on
  that transition; or
- record the fire with its own idempotent marker (a Redis `SETNX targetmet:{user}:{date}` inside the
  same pipeline as the INCRBY) so a retried call re-attempts the hook exactly once.

Either way a service test drives `fakeProgressRepo.err` on the crossing call, clears it, replays, and
asserts `fakePet.fired == 1` — coverage the suite does not have today. Whatever is chosen,
CODEMAP's "self-heals" sentence is narrowed to `minutes_spent`.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 5; *Notes and open questions*, "a crash between the two leaves Postgres behind").
- `backend/internal/quests/service.go:82-98`.
- `harness/CODEMAP.md` — `quests` paragraph, "Known accepted gap: a crash between the INCRBY and the upsert …".
- `backend/internal/quests/fakes_test.go:96,107-109` — `fakeProgressRepo.err` exists and is set by no test.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed in `quests/service.go`: `newly_met` is a one-shot edge on the Redis counter, so a failed upsert/`MarkComplete` on the crossing call silently costs the user that day's +20 and streak. CODEMAP already records the gap. Fix belongs in the pet "durable day judgement" plan with `the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md`, `service-ontargetmet-ignores-localdate…` and `a-passed-revival-is-knocked…`: derive the hook from `daily_progress.is_target_met` flipping false→true (`RETURNING`), which also gives pet its own once-per-day guard.

**Planned (2026-09-23):** `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` — decision 1, Task 7 (monotonic `daily_progress`; the hook fires from the durable flag, before `MarkTargetMet`/`MarkComplete`).
