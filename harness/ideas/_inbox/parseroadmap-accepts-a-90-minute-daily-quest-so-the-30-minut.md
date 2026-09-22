---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# ParseRoadmap accepts a 90-minute daily quest so the 30-minute day is unenforced

## Why
The 30-minute day is the product's retention promise: §6.1 constraint 4 says each Daily Quest "MUST
be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins
each)", the quests slice hard-codes `total_minutes_required: 30`, and the pet's health depends on
the learner hitting 1800 seconds. `ParseRoadmap` is the only gate between the model and
`exercises.content_json`, and it validates each task's duration independently against `1..30`
(`backend/internal/airouter/roadmap.go:125-127`). Three 30-minute tasks therefore pass:

```
PROBE 3x30min day accepted: true (err=<nil>)
```

A roadmap where every day is 90 minutes is written to the database, and `GET /quests/daily` then
shows a learner three 30-minute tasks under a header that says 30 minutes required — the day is
unachievable, the target is never met, the pet decays, and nothing in the stack flags it. The
generating model does not have to be malicious for this: "approximately 30 minutes" is a soft
instruction, and a model asked for a C1 roadmap will happily propose longer tasks.

The same routine has two smaller holes on the same theme — the validator checks only what it happens
to check:

- A missing `duration_minutes` is silently rewritten to 10 (`roadmap.go:122-124`) rather than
  rejected. That is a defensible default, but combined with the 1..30 range it means the validator
  never actually enforces "10 mins each" in either direction.
- `Roadmap.Title`, `Module.Title`, `Module.Focus` and `Day.Title` are never checked; only task
  titles are (`roadmap.go:119-121`). An empty roadmap title and empty day titles pass, and
  `harness/CODEMAP.md` overstates this as "empty titles" — it is empty *task* titles.

```
PROBE empty roadmap/module/day titles accepted: true
```

## Expected output
`ParseRoadmap` enforces the §6.1 day budget, not just a per-task sanity range:

- Each task's `duration_minutes` must be within a tight band around 10 (e.g. `5..15`), and each
  day's three tasks must sum to within a stated tolerance of 30 (e.g. `20..40`). The exact band is a
  judgement call and belongs in the code as a named constant with the §6.1 quote above it.
- A missing `duration_minutes` keeps defaulting to `DefaultTaskMinutes` — that behaviour is relied
  on by quests' `toTask` — but the default is applied before the day-sum check, not instead of it.
- `title` is required and non-empty at the roadmap, module and day levels, as it already is for tasks;
  `cefr_level` validation stays as it is.
- `TestParseRoadmapRejects` gains cases: a 3×30 day, a 3×3 day, an empty roadmap title, an empty day title.
- `harness/CODEMAP.md` says "empty **task** titles" rather than "empty titles", and describes
  whatever day-budget band this fix lands on. (The reviewer could not make this one-word correction
  on the branch — the commit was refused by the permission system — so it rides along with this fix.)

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 6).
- `project-base/1st-thinking-architecture-doc.md:348` — §6.1 constraint 4.
- `backend/internal/airouter/roadmap.go:16` (`maxTaskMinutes = 30`), `:119-127` — the per-task checks; no day-level sum, no non-task title check.
- `backend/internal/airouter/roadmap_test.go:66-95` — 15 rejection cases, none about the day budget; `"absurd duration"` uses 120, which the `1..30` range catches, so the band is never probed at its edge.
- `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md:2002` — `total_minutes_required: 30` and `is_target_met = total >= 1800`.
- Reviewer probes (scratch test, removed): a day of three 30-minute tasks and a roadmap with empty roadmap/module/day titles both parse without error.
