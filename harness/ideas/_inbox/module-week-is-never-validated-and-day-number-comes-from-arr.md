---
type: bug
status: selected
source: reviewer
run: _inbox
priority: medium
---
# Module.week is never validated and day_number comes from array position

## Why
`Module.Week` is decoded (`backend/internal/airouter/roadmap.go:34`), stored in
`roadmaps.roadmap_json`, and then never looked at again. `Roadmap.Exercises()` derives the
`exercises.day_number` purely from array position — `mi*DaysPerModule + di + 1`
(`roadmap.go:152`) — and `ParseRoadmap` never asserts that `week` agrees with that position. So a
model that emits its four modules out of order, or numbers them all `1`, produces a roadmap the
validator accepts and the persistence layer writes:

```
PROBE reversed weeks accepted; module[0].week=4 gets day_number=1,
      content={"type":"vocabulary","title":"WEEK4-DAY1-TASK1","duration_minutes":10}
PROBE all weeks=1 accepted: true
```

The result is two sources of truth that disagree inside one transaction. `exercises` say day 1..7 is
week 1; the `roadmap_json` the frontend renders for the roadmap view says those same days are week 4.
Nothing errors — the learner is served week 4's material on day 1 under a week-1 heading, and §6.1
constraint 5 ("difficulty must scale progressively") is quietly inverted. Because the roadmap is
generated once at onboarding and drives the next 28 days, a bad ordering is not self-correcting.

This is the mirror image of the same laxness: a field the schema declares and the prompt asks for,
which the validator neither enforces nor uses. Either it is load-bearing and must be checked, or it
is decoration and should not be in `RoadmapSchema`.

## Expected output
`ParseRoadmap` validates `week` against position: module `i` must declare `week == i+1`, rejected
with the existing `invalid(...)` wrapper (`module %d declares week %d`). That makes
`Exercises()`'s positional `day_number` provably consistent with the stored `roadmap_json`, and it
gives the model a checkable instruction rather than an ignored one.

If the owner would rather tolerate an out-of-order model, the alternative is to *sort* `r.Modules`
by `week` after validating that the set is exactly `{1,2,3,4}` — but silently accepting whatever
arrives is not an option, because it is the one case that produces wrong content with no error.

`TestParseRoadmapRejects` gains a `weeks out of order` case and a `duplicate week` case, and
`TestExercisesFlattensTo84Rows…` asserts that the task at `day_number` 8 came from the module
declaring `week: 2`. `harness/CODEMAP.md`'s `ParseRoadmap` sentence lists the new rejection.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 6).
- `backend/internal/airouter/roadmap.go:34` — `Week int \`json:"week"\``; `:93-130` — the validation loop never reads it; `:145-159` — `Exercises()` uses `mi`, not `m.Week`.
- `backend/internal/airouter/prompt.go:29` — `RoadmapSchema` shows `"week": 1`, so the model is told the field matters.
- `backend/internal/airouter/roadmap_test.go:12-34` — the fixture always sets `Week: m`, so the happy path can never notice.
- Reviewer probe (scratch test, removed): weeks `4,3,2,1` and weeks `1,1,1,1` both parse without error; the first module still receives `day_number` 1.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed: `roadmap.go` decodes `week` and never checks it; `Exercises()` uses array position. A model emitting modules out of order produces a roadmap whose stored JSON and `exercises.day_number` disagree, silently, for 28 days. Plan together with `parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md` — one `ParseRoadmap` validation plan, one `roadmap_test.go` table.
