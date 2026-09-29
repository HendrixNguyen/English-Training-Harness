---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Hub bubble tells a learner who practised yesterday 'Hôm qua tớ nhớ bạn' — missed-day check uses elapsed hours, not calendar days

## Why
Design `harness/designs/growth-moment.md` §4 row 5: "1 whole day means they practised yesterday (normal), 2+ means yesterday was missed." The executor changed the threshold to `daysSince(...) >= 1` (`frontend/utils/plant.ts:67`, deviation 1) because `daysSince` (`plant.ts:47`) floors **elapsed milliseconds** / 86 400 000. With `>= 1`, any learner who met yesterday's target at an earlier clock time than they open the app today (practised yesterday 08:00, opens today 09:00 → 25 h → 1) and has not studied yet today (`accumulated_seconds` 0) is greeted with "Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?" — the plant accuses a learner with an unbroken streak of missing yesterday. That is the first line a consistent daily learner sees each morning, i.e. the happy path. With the design's `>= 2` on the same function the opposite error appears (a real miss read as 1 day). Elapsed hours cannot answer a calendar question.

## Expected output
- Row 5 compares **local calendar dates**: "missed" when the learner's local date of `last_practiced_at` is ≤ today − 2 (yesterday had no met target). Compute it in the learner's timezone (the hub has `Intl` / `quest.daily.date` as today's date), without changing `daysSince` for `/revive`.
- Tests: last practised yesterday 08:00, now 09:00 → normal health line; last practised yesterday 23:30, now 00:30 → normal; last practised two calendar days ago 23:00, now 08:00 (33 h) → missed line.

## Evidence
- Plan: `harness/plans/2026-09-25-growth-moment-after-every-task-health-gain-streak-and-target.md` (execution summary deviation 1; branch head `a2d42fc`).
- `git show a2d42fc:frontend/utils/plant.ts | grep -n "missed >= 1"` → line 67; `daysSince` at line 47 (`Math.floor((now - t) / 86_400_000)`).
- The plan's tests only pin 38 h → missed and 14 h → normal; neither covers 24–36 h after a practised day.
