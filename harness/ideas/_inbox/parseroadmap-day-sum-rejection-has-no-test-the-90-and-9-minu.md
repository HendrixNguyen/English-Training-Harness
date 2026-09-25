---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
---
# ParseRoadmap day-sum rejection has no test; the 90- and 9-minute day rows fail on the task band

## Why
The day-sum check is the main thing this plan delivers: each day must add up to 20..40 minutes (§6.1 "approximately 30"). No test covers it. The two rows meant to cover it fail earlier, on the per-task band:

- `"three thirty-minute tasks (90-minute day)"`: 30 > `maxTaskMinutes` (15), so it is rejected as `task 1 duration 30 is outside 5..15`.
- `"three three-minute tasks (9-minute day)"`: 3 < `minTaskMinutes` (5), so it is rejected by the task band too.

With per-task limits of 5..15, a three-task day can only total 15..45. The day-sum branch only fires for 15..19 and 41..45, and no row uses those totals. Mutation probe: changing the check to `if false && (dayMinutes < minDayMinutes || dayMinutes > maxDayMinutes)` leaves `go test ./internal/airouter` fully green. Someone could delete the day-sum rule tomorrow and CI would not notice. The row names also claim to test the day budget when they do not. The reviewer role counts a dishonest test as a blocker.

## Expected output
- `TestParseRoadmapRejects` (or a dedicated test) has rows that only the day-sum rule can reject: `5+5+5 = 15` and `15+15+15 = 45`, each task inside 5..15.
- These rows assert the reason (the error mentions `adds up to`), so a task-band rejection cannot satisfy them.
- The two existing rows are renamed to what they actually test (task band), or kept as they are next to the new day-sum rows.
- Verification: with the day-sum `if` disabled, the new rows FAIL.

## Evidence
- Plan under review: `harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` (Task 1), branch `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` @ 42f8fd6.
- `backend/internal/airouter/roadmap_test.go:109-118`: the two rows. `backend/internal/airouter/roadmap.go:155-162`: task band, then day sum.
- Reviewer probe (scratch test, removed): `[5 5 5]` gives `module 1 day 1 adds up to 15 minutes, want 20..40`; `[15 15 15]` gives `adds up to 45`; `[30 30 30]` and `[3 3 3]` give `task 1 duration … is outside 5..15`.
- Mutation probe above: suite stays green with the day-sum check disabled.
