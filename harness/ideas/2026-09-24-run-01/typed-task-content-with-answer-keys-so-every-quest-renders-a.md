---
type: feature
status: proposed
source: ideator
run: 2026-09-24-run-01
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
