---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-26-parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md
---
# TestParseRoadmapRejects checks only that an error occurred, so a row can pass for the wrong reason

## Why
`TestParseRoadmapRejects` is a `map[string]string` of raw inputs. Each row only asserts `err != nil` and `errors.Is(err, ErrInvalidRoadmap)`. A row therefore passes when *any* rule rejects its input, not the rule it is named after. That is how the day-sum blocker (`parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md`) got through: both "day" rows were caught by the task band. The table now has about 25 rows and every new `ParseRoadmap` rule adds more (F2's content validation is next), so this weakness keeps growing. The pattern predates this plan; this plan extended it.

## Expected output
Each row carries the expected rejection reason, for example a substring of the `invalid(...)` message such as `"adds up to"`, `"declares week"`, `"has no title"` or `"outside 5..15"`. The loop asserts `strings.Contains(err.Error(), want)` in addition to `errors.Is`. Existing rows are updated. No production change.

## Evidence
- Plan under review: `harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md`.
- `backend/internal/airouter/roadmap_test.go`, `TestParseRoadmapRejects` loop (the `err == nil` / `errors.Is` checks only).

## Evaluation
_Evaluator, 2026-09-26._ **Select — low, folded into the blocker's amend plan** `harness/plans/2026-09-26-parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md` (Task 1 gives the table a `want` reason column and asserts it). No separate plan.
