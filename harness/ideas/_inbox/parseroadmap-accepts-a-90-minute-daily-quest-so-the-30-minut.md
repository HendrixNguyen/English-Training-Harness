---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
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

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium (top 10).** Confirmed: per-task `1..30` only, no day sum, no non-task title checks. A model that drifts to longer tasks yields a day the learner cannot finish under a header that says 30 minutes, and the pet decays for it — the product's core promise broken with no error. Plan together with `module-week-is-never-validated…` as one `ParseRoadmap` validation plan (named constants with the §6.1 quote, day-sum band, week == position, title checks, CODEMAP wording).

## Evaluation — 2026-09-24 daily planning (evaluator)
_Owner instruction 2026-09-24: pick ≤ 5 one-day tickets from the `selected` backlog, split Bug team / Feature team, write and approve the plans, one planning PR._

**Planned today — Bug team ticket B2 (head idea). Estimate 3 h.**
*Root cause (re-read on `main`).* `backend/internal/airouter/roadmap.go:116-126` validates each task's `duration_minutes` against `1..maxTaskMinutes(30)` only; there is no per-day sum, so `3×30` passes and `GET /quests/daily` then advertises a 30-minute day the learner cannot finish. Only task titles are checked (`roadmap.go:113`); roadmap/module/day titles are not. `Module.Week` is decoded (`roadmap.go:34`) and never compared with position (`Exercises()` uses `mi*DaysPerModule+di+1`).
*Why today.* The 30-minute day is the product's promise and the pet's health depends on it; a model that drifts to longer tasks breaks it silently for 28 days. The whole fix is in one function plus its test table.
*Folded into this ticket* (same function, same laxness — a schema field the validator neither enforces nor uses): `module-week-is-never-validated-and-day-number-comes-from-arr.md`. Decision recorded: **reject** out-of-order or duplicate weeks (do not sort) — silently accepting is the one option that yields wrong content with no error, and rejecting gives the model a checkable instruction through the existing retry-once path.
*Bands chosen* (named constants, §6.1 quoted above them): task `5..15` minutes, day sum `20..40`. The missing-duration default of 10 stays and is applied **before** the sum.
*Conflict note.* Feature ticket F2 (typed task content) also edits `ParseRoadmap`; F2 is written to put its logic in a new `content.go` with a one-line hook so the two branches overlap only on `CODEMAP.md` and the test-table tail.
*One-day check.* One file + one test file + CODEMAP; no migration, no frontend.
