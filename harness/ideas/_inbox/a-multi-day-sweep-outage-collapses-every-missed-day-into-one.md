---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# A multi-day sweep outage collapses every missed day into one penalty

## Why
`Service.Sweep` judges exactly one day per pet per tick - `judged = PreviousDate(LocalDate(now,
tz))` - and then advances `judged_through` straight to that day. Every day between the old
`judged_through` and `judged` is therefore marked judged **without ever being judged**. If the cron
does not run for three days (a failed deploy, a crashed instance, a Railway restart loop), the user
comes back to a single -30 instead of four, and a plant that should be `wilted` is still `sprout`
at 70 health. The section 8 inactivity logic silently under-applies in exactly the situation where
the product most wants it to bite.

The code and CODEMAP both claim more than the code does: `backend/internal/pet/service.go:130-132`
says "a tick the process slept through is caught up at the next one", and the CODEMAP pet bullet
repeats it. That is true for a missed **hour** and false for a missed **day** - the wording should
not be left as is whichever way the behaviour is decided.

This is strictly more lenient than `main` (whose `Hour() == 0` trigger lost the day outright), so it
is not a regression - it is an unclosed half of the plan's own design decision 3.

## Expected output
The owner decides one of the two, and the code and CODEMAP say which:

- **catch up** - `Sweep` loops from `max(judged_through + 1, first day the pet existed)` up to
  `judged`, applying `PenaliseMiss` per day, bounded by a cap (e.g. 7 days) so a long-dormant pet
  cannot be hit with an unbounded loop of round trips; or
- **forgive by design** - one penalty per catch-up is the intended product behaviour (an outage is
  the operator's fault, not the learner's), in which case the doc comment and the CODEMAP sentence
  say "the most recent ended day is judged; days skipped by an outage are forgiven".

Either way a test pins it: seed `judged_through = D-5`, run one `Sweep` at D, and assert the
resulting health and `judged_through` against the decided rule.

## Evidence
- Plan: `harness/plans/2026-09-23-the-miss-sweep-judges-the-day-from-volatile-redis-and-ignore.md` (status `done`), design decision 3.
- `backend/internal/pet/service.go:156-196` - one `judged` per candidate per tick;
  `PenaliseMiss` / `MarkJudged` set `judged_through = judged` outright.
- `backend/internal/pet/service.go:130-132` and `harness/CODEMAP.md` (pet bullet) - "a tick the
  process slept through is caught up at the next one".
- Reproduced in review with a scratch test against the package fakes: `judged_through` seeded to
  `2026-09-18`, one `Sweep` at `2026-09-23T00:00Z` -> `penalised=1, health=70,
  judged_through=2026-09-22`. Five further ticks the same local day add nothing. Four days
  (19, 20, 21, 22) were missed; one -30 was applied, and the plant did not wilt.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Not planned today; the product decision the idea asks for is recorded here.**

*Confirmed (read on this branch).* `backend/internal/pet/service.go` `Sweep`: `judged = PreviousDate(quests.LocalDate(now, loc))` once per pet, and both `PenaliseMiss` and `MarkJudged` set `judged_through = judged` outright, so every day between the old marker and `judged` is marked judged without being judged. The doc comment ("a tick the process slept through is caught up at the next one") and the CODEMAP pet bullet are true for a missed hour and false for a missed day.

*Decision (evaluator; the owner may overrule before this is planned): catch up, bounded.* Backend spec §8's inactivity logic is per day, and the penalty is for the learner's inactivity, not the operator's outage — a plant at 0 after four unstudied days is exactly the wilted → revive path §5.2 designs for. `Sweep` walks from `max(judged_through + 1, first day the pet existed)` to `judged`, applying `PenaliseMiss` per day, and stops early at health 0 or after 7 days (the floor makes further iterations pointless; the cap bounds round trips for a long-dormant pet). The leniency read of `daily:accumulated` for each caught-up day still applies where the key exists (48 h TTL), so a learner who studied during the outage is spared those days. The service.go comment and the CODEMAP sentence change to say exactly this. Test: seed `judged_through = D-5`, one `Sweep` at `D` → four penalties, `wilted`, `judged_through = D-1`.

*Why not today.* An outage of the in-process cron is rare (it needs the API itself to be down for a day), and the five slots go to a production blocker, a spec-mandated security control, and happy-path fixes. Medium: it under-applies the retention signal exactly when it should bite.
