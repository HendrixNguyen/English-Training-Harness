---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
plan: harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md
---

# DayNumber loses a calendar day at every spring-forward DST transition

## Why
`DayNumber` converts both timestamps to local midnight and then divides the **wall-clock difference**
by 24 hours (`backend/internal/quests/day.go:36-48`):

```go
days := int(today.Sub(start).Hours()/24) + 1
```

`time.Sub` returns absolute elapsed time. Between two local midnights that straddle a spring-forward
transition only 23 hours elapse, so the quotient truncates and the learner silently loses a day —
permanently, for the rest of the roadmap, since the deficit never comes back until an equal and
opposite fall-back. Every DST zone is affected; `Asia/Ho_Chi_Minh` (the spec §6.2 example) and `UTC`
are not, which is why the existing timezone tests pass.

Reproduced with a verbatim copy of `DayNumber`/`startOfDay`:

```
Europe/London     created 2026-03-25  ->  local 2026-03-30  DayNumber= 5  correct= 6   (and every day after)
America/New_York  created 2026-03-05  ->  local 2026-03-09  DayNumber= 4  correct= 5
Australia/Sydney  created 2026-09-29  ->  local 2026-10-05  DayNumber= 6  correct= 7
```

The consequence is that a learner in a DST zone is served yesterday's three exercises for the rest of
the 28-day roadmap, and reaches day 28 a day late (two days late after a second transition). It is
the function that selects all quest content, so the error is user-visible on the main screen.

Note `LocalDate` (`day.go:29-31`) is correct — it formats in the location and is unaffected.

## Expected output
`DayNumber` counts **calendar** days, not elapsed hours. The usual Go idiom is to normalise both
local midnights onto a fixed-offset day and subtract, or to walk with `AddDate`:

```go
start := startOfDay(createdAt, loc)
today := startOfDay(now, loc)
su := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
tu := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
days := int(tu.Sub(su)/(24*time.Hour)) + 1
```

`day_test.go` gains a case per direction — a roadmap spanning a spring-forward (e.g. `Europe/London`,
created 2026-03-25, asserted day 6 on 2026-03-30) and a fall-back — so a future rewrite cannot
reintroduce it.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 1; the plan's own timezone cases are `UTC` and `Asia/Ho_Chi_Minh`, neither of which observes DST).
- `backend/internal/quests/day.go:36-48`.
- `backend/internal/quests/day_test.go:31-69` — no DST case.
- Reproduction above (standalone copy of the function, three zones).

## Evaluation

**Verdict: select, `high`, folded into the blockers' amending plan.** Not a blocker, but the reviewer called it the
highest-value non-blocking fix, it is in the same package on the same open branch, and the owner wants the MVP right the
first time rather than shipping a `DayNumber` that serves DST learners yesterday's quests for the rest of the roadmap.
Reproduced by the evaluator with absolute-time arithmetic (Go's `Sub`): Europe/London created 2026-03-25 → 03-30 gives 5,
correct 6, and reaches day 28 as 27 on 2026-04-21; America/New_York 03-05 → 03-09 gives 4, correct 5; Australia/Sydney
09-29 → 10-05 gives 6, correct 7. Fall-back dates (London 10-25, New York 11-01, Sydney 04-05) are unaffected by either
formula, so they go in as regression cases. 2026 transitions confirmed from the system tzdata.

**Root cause:** `day.go:40` — `int(today.Sub(start).Hours()/24) + 1` divides *elapsed* time between two local midnights
by 24h; a spring-forward day is 23h long, so the quotient truncates one day short, permanently.

**Fix:** count calendar days — normalise both local midnights onto UTC dates and divide, so every day is exactly 24h —
with table tests for the three zones across both transitions. `LocalDate` is already correct and untouched.

**Plan:** `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md` (Task 1).
