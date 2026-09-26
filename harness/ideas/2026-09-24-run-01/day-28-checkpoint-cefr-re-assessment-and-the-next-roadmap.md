---
type: feature
status: planned
source: ideator
run: 2026-09-24-run-01
priority: medium
plan: harness/plans/2026-09-26-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md
---
# Day-28 checkpoint: CEFR re-assessment and the next roadmap

## Why
The product promises CEFR progression (1st-thinking §1), but every learner hits a dead end after four weeks. `quests.RoadmapDays = 28`, and `day_number` is clamped, so "a learner past day 28 keeps seeing day 28" (`backend/internal/quests/day.go:53`). The same three tasks repeat forever. `POST /onboarding/assessment` returns the existing active roadmap with `200` and makes no AI call, so a learner cannot get a second roadmap even by re-onboarding. The Google Calendar event is `COUNT=28`, so it disappears on the same day. And `users.cefr_current` never changes after placement, so the one progress number the product shows is frozen.

Day 28 is exactly the point where a streak-holding learner is most invested and most at risk of leaving. Research on learning apps agrees that learners who can see concrete progress stay motivated, and that invisible progress "kills motivation". Run-01 dropped this idea only because onboarding and the AI router were not built yet. Both are merged now.

## Expected output
User-visible:
- From day 28, once that day's target is met (or any day after 28), the home hub shows a **checkpoint card** in place of the quests: "You finished your 4-week roadmap — take the level check."
- The level check reuses the placement quiz UI: a fresh question set centred on the learner's current level. The result screen shows old level → new level (or "consolidating at B1"), then generates and displays the next 28-day roadmap.
- `/roadmap` shows completed roadmaps as history (title, CEFR level, dates).
- The pet streak and health carry over; nothing resets.
- If Google sync is enabled, the calendar event and task list are re-pushed for the new roadmap. The google package already rebuilds the task list when `roadmap_id` changes.

Technical:
- A reassessment endpoint (e.g. `POST /api/v1/roadmaps/next` with quiz answers). Onboarding's "already has an active roadmap → 200" guard stays as it is.
- The endpoint reuses `TaskPlacementTest`/`TaskRoadmapGen`, `ParsePlacement`/`ParseRoadmap`, the `ratelimit:ai` slot, and the single-transaction save (deactivate the old roadmap, insert the new one plus 84 exercises), and updates `cefr_current`. The prompt receives the previous level and the goal. If the typed-content idea lands, it can also receive the learner's per-type scores.
- `GET /quests/daily` returns an explicit `roadmap_complete: true` flag (additive) past day 28, so the client does not infer it from the clamp.
- Endpoint added to the backend spec §6/§7 and the CODEMAP.
- Tests cover: the day-28 edge in the learner's timezone, the concurrent double-submit (same class as inbox bug *two-concurrent-assessment-submits…*), and AI failure leaving the old roadmap active.

## Evidence
- `backend/internal/quests/day.go:8,53` (28-day clamp, "keeps seeing day 28"); CODEMAP **onboarding** ("if an active roadmap exists return it with 200, no AI call"), **google** (`COUNT=28`, rebuilds tasks on new `roadmap_id`).
- 1st-thinking §1 (CEFR progression goal), §5.1 (placement → roadmap), §6.1 (4 × 7 × 3 shape).
- Prior run `2026-09-22-run-01/_run.md` Notes: "CEFR re-assessment at the end of module 4 (strong, but depends on onboarding + airouter landing first; re-propose next run)".
- Progress visibility and motivation: https://migaku.com/blog/language-fun/language-learning-progress-tracking , https://journals.sagepub.com/doi/10.1177/00472395241238693
- Inbox noted: `two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` (the same race applies to any roadmap-replacing route).

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Not planned today.**

*Is the Why real?* Yes and verifiable in code: `backend/internal/quests/day.go` clamps `day_number` at 28, `onboarding` returns the existing active roadmap with `200` and no AI call, and `users.cefr_current` is written once at placement. Every learner dead-ends after four weeks with the same three tasks. The first real user reaches day 28 about four weeks after launch — this must be planned before then, not today.

*Achievable in one plan?* Backend and frontend together are a long day; plan it as one backend plan (`POST /api/v1/roadmaps/next` reusing `TaskPlacementTest`/`TaskRoadmapGen`, the parsers, the rate-limit slot and the single-transaction save; `GET /quests/daily` gains an additive `roadmap_complete`; spec §6/§7 and CODEMAP) and one frontend plan (checkpoint card, level-check reuse of the quiz UI, roadmap history; design note first). *Dependencies:* the concurrent double-submit race (`two-concurrent-assessment-submits-…`, selected medium) applies to any roadmap-replacing route — fix it in onboarding first or fold the same guard into the new endpoint.

_Evaluator, 2026-09-26 — planned today (feature queue; backend plan, draft)._ Written on top of today's bug plan B2 (`harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md`), whose `Repo.ReplaceRoadmap` is the single-transaction replace this route needs; the plan says so and must not execute before B2 is on `main`. The frontend (checkpoint card, level-check reuse of the quiz UI, roadmap history) is a separate plan after the retro hub/roadmap screens.
