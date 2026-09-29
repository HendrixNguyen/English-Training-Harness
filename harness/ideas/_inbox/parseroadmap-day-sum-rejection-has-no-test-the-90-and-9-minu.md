---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
plan: harness/plans/2026-09-26-parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md
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

## Evaluation
_Evaluator, 2026-09-26 — daily decide; blocker, evaluated first._

**Select — high (blocker). Amend plan on the same branch.**

*Is the Why real?* Yes, and reproduced by reading the branch (`origin/harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` @ 42f8fd6): `roadmap.go` checks the task band (`5..15`) before summing, so the `[30 30 30]` and `[3 3 3]` rows never reach the day-sum `if`; with tasks in `5..15` the day sum can only fail at 15..19 or 41..45 and no row uses those. The review's mutation probe (day-sum `if` disabled → suite green) is the proof.

*Root cause.* `TestParseRoadmapRejects` asserts only `err != nil` and `errors.Is(err, ErrInvalidRoadmap)` — never the reason — so a row passes for whichever rule fires first. The two "day" rows were chosen outside the task band, so the earlier rule always fires. (The general weakness is the low inbox item `testparseroadmaprejects-checks-only-that-an-error-occurred-s.md`; this plan fixes it for the day-sum rows and gives the table a `want` reason column, which also closes that item.)

*Is the amended plan's code on `main`?* No — `origin/main` still has `maxTaskMinutes = 30` and no day sum (`backend/internal/airouter/roadmap.go:16`). The fix lands on the branch under review, per the blocker rule.

*Smallest correct fix.* Two rows that only the day sum can reject (`5+5+5`, `15+15+15`), the table keyed by expected reason substring, the two misnamed rows renamed to the task band, and a verification step that disables the day-sum `if` and shows the new rows fail. No production change.
