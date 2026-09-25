---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Week-2 provenance assertion in TestExercisesFlattens repeats the day-number check

## Why
The idea `module-week-is-never-validated-and-day-number-comes-from-arr.md` asked that `TestExercisesFlattensTo84Rows…` assert that *the task at `day_number` 8 came from the module declaring `week: 2`*. The added check is `r.Modules[1].Week != 2 || ex[21].DayNumber != (r.Modules[1].Week-1)*DaysPerModule+1`. That only restates two facts the test already has: the fixture sets `Week: m`, and line 150 already asserts `ex[21].DayNumber == 8`. It never links the exercise row to module 2's content. It also cannot, because every fixture task title is `"<type> task"`, identical across modules. If `Exercises()` read from the wrong module, this assertion would still pass.

## Expected output
Make the fixture's task titles (or content) unique per module/day, e.g. `fmt.Sprintf("w%d-d%d-%s", m, d, tt)`. Then assert `ex[21].Title` (or its content) equals `r.Modules[1].Days[0].Tasks[0].Title`. That proves the row at day 8 was produced from the module declaring week 2.

## Evidence
- Plan under review: `harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` (Task 2).
- `backend/internal/airouter/roadmap_test.go:20` (fixture task titles), `:150` (existing day-number check), `:155-157` (new assertion).
