---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: Two wrong code comments with correct code behind them; fix them in the next quests change (the pet day-judgement plan edits quests/service.go) rather than on their own branch.
---
# Two comments in the new progress validation misstate the code they describe

## Why
Both bounds added by `551770a` are correct; two of the comments explaining them are not, and both
sit on exactly the reasoning a maintainer would trust instead of re-deriving.

**1. `MaxDailySeconds`'s race-headroom figure is wrong by ~36x.**
`day.go:22-29` says *"each can overshoot by at most MaxDurationSeconds, so the INT limit stays ~10^6
concurrent max-size requests away."* `daily_progress.minutes_spent` is INT, so the limit is
2147483647 minutes = 1.288e11 seconds; from the 86400 ceiling that is
`(1.288e11 - 86400) / 3600 ~= 3.6e7` simultaneous max-size requests, not 1e6. The amending plan's own
*Notes* states the correct 3.6e7 — the constant's doc comment disagrees with the plan it implements.
The error is in the safe direction (it understates the headroom), but it is the one number a future
reader would use to decide whether the read-then-INCRBY race needs a compensating write, and being
36x off could argue a real fix into existence that the arithmetic does not justify.

**2. The `ErrInvalidDuration` handler comment is false for one of its two branches.**
`handler.go:65-68` justifies reusing `400 invalid_request` with *"the request is malformed, not the
state."* That is true of `seconds <= 0 || seconds > MaxDurationSeconds`, but `ErrInvalidDuration` is
also returned when `total + seconds > MaxDailySeconds` (`service.go:97-99`) — there the request is
perfectly well formed and is rejected *because of* the state. Verified live: with the counter at
86300, an ordinary `{"duration_seconds":200}` returns `400 {"error":"invalid_request"}`, byte for
byte what a malformed body returns. Collapsing the two into one code is a deliberate, documented
product decision (the amending plan's *Notes*: "If the product later wants a distinguishable code
(`daily_limit_reached`), it is a one-line handler change"), so the behaviour is not the bug — the
comment asserting the opposite of what half the branch does is.

## Expected output
- `day.go`'s `MaxDailySeconds` comment carries the correct order of magnitude (~3.6e7 concurrent
  max-size requests) and matches the amending plan's *Notes*, or drops the figure and points at the
  plan instead of restating it.
- `handler.go`'s `ErrInvalidDuration` comment names both branches: a malformed magnitude *and* a
  well-formed report the day has no room for, with a pointer to the decision that they intentionally
  share `invalid_request`.
- Optionally, if the frontend ever needs to tell a learner "you have hit today's limit" rather than
  "bad request", the one-line `daily_limit_reached` split the plan describes — a separate product
  call, not part of this fix.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md`
  (amended by `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md`, Task 3).
- `backend/internal/quests/day.go:22-29`; `backend/internal/quests/handler.go:65-68`;
  `backend/internal/quests/service.go:97-99`.
- Live, real binary + real Redis/Postgres: `SET daily:accumulated:<uid>:<date> 86300`, then
  `POST /api/v1/quests/progress {"exercise_id":"<own day-1 task>","duration_seconds":200}`
  -> `400 {"error":"invalid_request"}`, counter unchanged at 86300.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject.** Cosmetic. Noted in the pet day-judgement survivor so the next `quests` edit corrects both comments.
