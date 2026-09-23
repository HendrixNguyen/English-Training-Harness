---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Test-only with no demonstrated defect: time.Date normalises month/year rollover correctly and the reviewer confirmed the behaviour is right; pinning it is tidiness."
---
# schedule.go boundaries are untested at notification_time equals now and month year rollover

## Why
`schedule.go` is the one piece of pure arithmetic in the slice and the thing that decides when 28
calendar reminders fire. `schedule_test.go` covers three cases well — today-vs-tomorrow, the
America/New_York fall-back DST day, and a within-September day-28 due date — and leaves the
boundaries that actually bite untested:

- **`notification_time` exactly equal to `now`.** `NextOccurrence` decides with
  `if !candidate.After(now)` (`schedule.go:62`), so equality rolls to *tomorrow*. That is a
  defensible choice and an undefended one: `schedule_test.go:26-40` only uses strictly-ahead
  (17:00 vs 20:00) and strictly-past (17:00 vs 09:30). Flipping `!candidate.After(now)` to
  `candidate.Before(now)` — a different product decision — breaks no test.
- **Month and year rollover.** `schedule.go:63` builds tomorrow as `time.Date(..., l.Day()+1, ...)`
  and `DayDue` builds day N as `time.Date(..., l.Day()+n-1, ...)` (`schedule.go:110`). Go's
  `time.Date` normalises out-of-range days, so these are correct — but nothing proves it. There is
  no case with `now` on 31 December, and no roadmap created on 20 January whose day 28 lands in
  February, or created in December landing in the next year. `DayDue`'s only test
  (`schedule_test.go:95-107`) stays inside September.
- **DST spring-forward.** Only the 25-hour fall-back day is tested
  (`TestNextOccurrenceKeepsWallClockAcrossDST`). A `notification_time` inside the skipped hour
  (02:00–03:00 on the spring-forward date) is normalised by `time.Date` to an hour that exists; no
  test says which, so the behaviour is unpinned. The repo already has a standing bug about exactly
  this class in the sibling package — `daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md`.

None of these is a demonstrated defect; all of them are behaviour the next maintainer can change
without any test objecting.

## Expected output
`schedule_test.go` gains table cases for: `now` exactly at `notification_time` (pinning the
roll-to-tomorrow rule and naming it in `NextOccurrence`'s doc comment); `now` on 31 December with a
passed notification time (asserting 1 January of the next year); a roadmap created 20 January with
day 28 due in February and one created in December with a due date in January; and a
`notification_time` in the spring-forward gap, asserting whichever normalised instant is intended.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), Task 2.
- `backend/internal/google/schedule.go:55-66` — `NextOccurrence`, the `!candidate.After(now)` boundary and the `l.Day()+1` rollover.
- `backend/internal/google/schedule.go:104-111` — `DayDue`, the `l.Day()+n-1` rollover.
- `backend/internal/google/schedule_test.go:26-40, 50-65, 95-107` — the three existing cases and their limits.
- Same class, different package, already filed: `harness/ideas/_inbox/daynumber-loses-a-calendar-day-at-every-spring-forward-dst-t.md`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject.** By its own account "none of these is a demonstrated defect". Add the table cases opportunistically when `schedule.go` next changes.
