---
type: feature
status: selected
source: ideator
run: 2026-09-24-run-01
priority: medium
---
# Typed task content with answer keys so every quest renders and gives instant feedback

## Why
The 30 minutes a learner spends each day is only worth anything if the content in those minutes teaches. Today it often cannot. The roadmap prompt says `"content" is free-form JSON for the task material` (`backend/internal/airouter/prompt.go:45`), and `ParseRoadmap` validates everything *except* `content`. The PWA's `classifyContent` (`frontend/utils/content.ts`) guesses: if it finds a `words` or `questions` array it renders them, otherwise it falls back to `{kind: 'raw'}`, which prints the JSON with `JSON.stringify` — a learner sees curly braces instead of a lesson. Even when questions render, there is no correct answer anywhere in the payload, so `/learn/:id` cannot say right or wrong, and `user_answers` is "accepted, not persisted" (CODEMAP **quests**). So the product has a timer and a plant but no learning signal.

Research on corrective feedback in second-language learning finds a medium overall effect (d ≈ 0.64) that holds over time, with immediate feedback beating delayed feedback. A fixed content contract is also what every later adaptive feature depends on: re-assessment, spaced repetition and exercise generation all need to know what a task *is* and whether the learner got it right.

## Expected output
User-visible:
- Every quest task renders as a real exercise; no user ever sees raw JSON.
- Vocabulary tasks show term, definition and example sentence, then a short self-check.
- Reading/listening tasks show a passage followed by multiple-choice questions.
- Practice tasks show fill-in or multiple-choice items.
- Answering a question shows correct/incorrect immediately, with a one-line explanation. A task ends with a score ("4/5").

Technical:
- `airouter.RoadmapSchema` defines `content` per task type, for example:
  - `vocabulary` → `{words[{term, definition, example}], questions[]}`
  - `reading` → `{passage, questions[{id, prompt, options{A..D}, answer, explanation}]}`
  - `practice` → `{questions[]}`
- The system prompt states that schema, and `ParseRoadmap` rejects content that does not match it. It rejects `answer` values that are not in `options`, and question counts outside set bounds. Invalid output uses the existing one-retry path → `ai_bad_output`.
- `GET /quests/daily` keeps returning `content_json`. **Decision for the evaluator/planner:** send `answer` to the client (simple, offline-capable, the answer key is low-stakes) or keep it server-side and grade in `POST /quests/progress`. The idea does not presume either.
- `POST /quests/progress` persists the score, either as a new nullable column on `exercises` or as a `daily_progress` field (migration + backend spec DDL update). The score does not change the 30-minute or pet logic.
- `classifyContent` gains typed branches and stops producing `raw` for valid roadmaps. `raw` stays only as a legacy fallback for roadmaps generated before this change.
- Tests: parser cases for each type, frontend rendering and feedback for each type.

## Evidence
- `backend/internal/airouter/prompt.go:36–45` (content `{}` placeholders, "free-form JSON"); CODEMAP **airouter** (`ParseRoadmap` checks everything but content), **quests** (`user_answers` accepted, not persisted).
- `frontend/utils/content.ts` (`raw` fallback = `JSON.stringify`), `frontend/pages/learn/[id].vue` (answers collected, never evaluated).
- 1st-thinking §6.1 (3 tasks × ~10 min: vocabulary/grammar, reading/listening, practice/interactive) — "interactive" is not delivered.
- Li (2010), *The Effectiveness of Corrective Feedback in SLA: A Meta-Analysis*: https://onlinelibrary.wiley.com/doi/abs/10.1111/j.1467-9922.2010.00561.x ; feedback timing review: https://pmc.ncbi.nlm.nih.gov/articles/PMC9995700/ ; CALL feedback meta-analysis: https://tesl-ej.org/wordpress/issues/volume24/ej94/ej94a4/

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Not planned today; plan right after the settings screen.**

*Is the Why real?* Yes, and it is a happy-path defect dressed as a feature: `backend/internal/airouter/prompt.go` declares `content` free-form, `ParseRoadmap` validates everything but `content`, and `frontend/utils/content.ts` falls back to `JSON.stringify` — a learner can be shown braces instead of a lesson, and no task can say right or wrong. Every later adaptive feature (re-assessment, spaced repetition, writing) needs this contract.

*Achievable in one plan?* No — two, and they must land in order: (1) backend — `RoadmapSchema` gains per-type `content` shapes (`vocabulary → {words[{term, definition, example}], questions[]}`, `reading → {passage, questions[{id, prompt, options{A..D}, answer, explanation}]}`, `practice → {questions[]}`), the §6.1 system prompt states them, `ParseRoadmap` rejects mismatches (answer ∉ options, counts out of bounds) through the existing retry-once → `ai_bad_output` path, and `POST /quests/progress` persists a score (migration + backend spec DDL); (2) frontend — typed renderers on `/learn/:id` with instant feedback and a score, `raw` kept only for pre-change roadmaps (design note first). *Decision for the planner:* send `answer` to the client — the key is low-stakes, it keeps the exercise offline-capable, and server-side grading would add a POST round trip for no security gain.

*Priority.* Medium, not high: the app functions and the fallback renders *something*; but it is the highest-value feature after the settings screen and gates two other selected ideas.
