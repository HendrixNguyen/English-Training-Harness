---
type: feature
status: planned
source: human
run: 2026-09-25-run-01
priority: high
plan: harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md
---
# A session a learner wants to finish: level-true content, do-to-complete tasks, feedback and growth every task

## Why
Owner feedback after the first real session on the live app (2026-09-25): "I can't learn with the current UI, it's not interesting, I can't focus at least 30 minutes, that UI is not attractive to me." The 30-minute daily target is the product's whole retention thesis (spec §1, §5.2), so a session that cannot hold attention for 30 minutes is a product failure, not a polish item. Four causes were observed, in order of weight:

1. **Broken task rendering** — filed separately as the inbox bug "59 of 84 roadmap tasks render as raw JSON…"; this idea assumes that bug's typed-content contract lands first.
2. **Completion by waiting, not doing** — the learning room disables "Hoàn thành" until a 10-minute countdown expires, whatever the content (six flashcards → nine minutes of staring). Nothing to answer, nothing to get right or wrong.
3. **Content below the learner** — the placement graded the owner **A2** and the roadmap opens with "Good morning", "one, two, three", "manager: person in charge of a team". The owner answered "Had it not been for the delay, we ___ the deadline" correctly (B2 grammar) and is a working software engineer. Either the 10-question quiz cannot discriminate B1/B2, the free grader is miscalibrated, or the roadmap prompt ignores `target_goal` depth; the evaluator should check the placement fixtures and the grading prompt.
4. **No reward loop inside the session** — the plant only reacts at the 30-minute day target; there is no per-task growth, streak flame, XP or correct-answer moment, though `growth-moment-after-every-task…` (planned, medium) already proposes the plant reaction.

## Expected output
User-visible, in one 30-minute session:
- Every task is an activity with an end: flashcards flipped and self-rated, passage read then 3–5 questions answered, practice questions answered — and "Hoàn thành" lights up when the last item is done. Real time still counts toward the 30 minutes; a fast learner finishes early and gets the next task, a slow one is never blocked.
- Every answer gives instant feedback: correct/incorrect, the explanation, and a small visible gain (health tick, XP, streak flame; the growth-moment idea), so there is a reason to answer the next one.
- Content matches the learner: the roadmap prompt receives the graded level **and** the target goal and is told to write at that level (B1 → workplace email/meeting language, not greetings); the day-1 tasks are shown in onboarding with a one-tap "too easy / about right / too hard" that regenerates the roadmap one level up or down before the plan is committed.
- The learning room follows spec §7.3 (progress "Câu 2 / 10", one item at a time, big touch targets) and §6.1 (emerald/amber/coral tokens, micro-animation on correct answers and task completion); the dashboard shows the three tasks as a path with the plant reacting between them (§7.2).

Technical:
- Frontend: rework `pages/learn/[id].vue` + `ContentViewer` around item progress instead of the timer; a `useSessionScore` (or extend `useQuestStore`) for per-task answers → `POST /api/v1/quests/{id}/complete` already accepts `answers`; design doc from the frontend-design skill for the learning room and the dashboard path.
- Backend: roadmap prompt gains level-specific guidance and the owner's goal text; placement fixtures reviewed (does a 10/10 land on B2?); optional `POST /api/v1/roadmaps/regenerate?level=` for the onboarding calibration tap (reuses the staged-level path from plan `2026-09-25-providertimeout-…`).
- Out of scope here: speaking/recording, spaced repetition (already an idea), essay grading (already an idea).

## Evidence
- Live DB 2026-09-25: `users.cefr_current = A2`, `target_goal = Business English`; roadmap title "28-Day Business English Roadmap for A2 Learners"; day-1 vocabulary = greetings and numbers (see the inbox bug for the raw shapes).
- `pages/learn/[id].vue:26-27,44-47,112`: timer started on open, button disabled until `remainingSeconds === 0`; `components/learn/CountdownTimer.vue`; no answer feedback anywhere in `ContentViewer.vue`.
- Spec: frontend §6.1 (gamified minimalist, micro-animated), §7.2 (dashboard path + plant speech bubble), §7.3 (learning room wireframe); 1st-thinking §1 (30 min/day retention), §5.2 (daily loop).
- Related ideas: `growth-moment-after-every-task-health-gain-streak-and-target` (planned, medium), `spaced-repetition-vocabulary-review-in-the-daily-quest`, `ai-graded-writing-practice-through-the-essay-grading-route`.

## Evaluation
_Evaluator, 2026-09-26 — daily decide (feature queue; owner idea, high)._

**Select — high. Planned today: the backend part (level-true content + placement floor). The other three causes are owned elsewhere.**

*Is the Why real?* Yes — the owner's first session, with database evidence. Of the four causes: (1) broken rendering is the inbox bug `59-of-84…` (backend contract done on the typed-content branch; regenerate route planned today; frontend = retro learning room, plan 3); (2) do-to-complete and (4) per-answer reward are exactly `harness/designs/retro-learning-room.md` (XP in the top bar, "Trúng!/Trượt…", chest) plus the draft `growth-moment` plan; (3) content below the learner is not owned by anything yet — that is this plan.

*Root cause of (3), read on `main`:* `airouter.RoadmapUserPrompt` (`prompt.go:48`) passes only `Current CEFR level: A2` / `Target goal: Business English` / minutes — no descriptor of what the level means, no instruction to write in the goal's register, so a free model defaults to greetings and numbers. And the level itself: `onboarding.Bank` has two items per level A1–C1; the grader is an LLM asked to "estimate"; `ParsePlacement` accepts whatever level it returns. A learner who answers 10/10 (including the C1 items) can still be graded A2 by a weak free model, and nothing in Go contradicts it.

*Achievable in one plan?* Yes, ≈3 h, backend only: (a) `airouter.LevelGuidance(level)` — a CEFR descriptor per level (vocabulary range, grammar, passage length, register) and an explicit goal-register instruction appended to `RoadmapUserPrompt`; (b) a deterministic placement floor `onboarding.GradeFloor(answers)` — the highest level at which both bank items are correct with every lower level at least half right — applied as `level = max(ai, floor)` in `Assess`, logged. The calibration tap ("too easy / about right / too hard") is frontend (retro onboarding, plan 4) on top of today's regenerate route (`cefr_level` ± 1) — not here.

*Priority.* High — the roadmap the regenerate route produces must be worth doing; auto-approved.
