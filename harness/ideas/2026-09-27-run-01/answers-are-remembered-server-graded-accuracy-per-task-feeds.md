---
type: feature
status: proposed
source: ideator
run: 2026-09-27-run-01
order: 3
---
# Answers are remembered: server-graded accuracy per task feeds the world map, task swaps and the day-28 checkpoint

## Why
The product measures minutes and never measures learning. Backend spec §6.2 sends `user_answers` with every `POST /api/v1/quests/progress`; the handler accepts the field and drops it (`internal/quests/handler.go:40–44`, CODEMAP: "accepted, not persisted"). So the roadmap tree shows "12/30 phút", the growth moment shows health, and the day-28 checkpoint has to re-quiz the learner because there is no performance record to read. For a product whose second stated goal is CEFR progression (1st-thinking §1), "did you get it right?" is the one signal it throws away.

The typed-content plan put `answer` and `explanation` into every question's `content_json`, which means the server can grade an answer set with a map lookup — no AI, no new dependency. Its part 2 (frontend instant feedback and a local "4/5") is explicitly deferred and "decide then whether a score is persisted"; this idea is that decision: **persist and use it.** A learner who sees "Ngày 3 · 12/15 câu đúng" on the map sees themselves learning, not just spending time (progress visibility is the most reproduced retention lever in the category); the task swap (idea 1) and the checkpoint get a real input instead of a guess; and the owner gets the first learning metric the specs never defined.

## Expected output
User-visible:
- After a task, the quest row on the hub shows "Đúng 4/5" next to "Xong" (only for tasks with questions; a task from a pre-typed-content roadmap shows nothing). The world map's expanded day shows "12/15 câu đúng" and each region tally shows accuracy alongside days met. The learning room's own instant feedback (typed-content part 2) is unchanged by this idea and can land before or after it.
- The checkpoint / regenerate path can read "accuracy over the last 7 days"; whether the AI prompt uses it is the checkpoint plan's decision — this idea only makes the number exist and exposes it.
- Nothing about the plant changes: time still waters it, not scores (writing-practice idea's rule, kept).

Technical (backend `quests`; frontend hub + roadmap):
- `POST /quests/progress`: `user_answers` is parsed as `{question_id: "A".."D"}` (anything else → 400 `invalid_request`; unknown ids are ignored; the body is already bounded at 64 KiB). Grading happens **server-side** against `exercises.content_json` questions' `answer` (typed content only; a task without questions grades to null). The result is written in the same `MarkComplete` write: `exercises.answers_json JSONB NULL`, `exercises.score_correct SMALLINT NULL`, `exercises.score_total SMALLINT NULL` (migration + backend spec §3.2 DDL appended). The INCRBY → upsert → hook → mark order the service tests pin is untouched; grading is pure and happens before any write.
- Additive fields: progress response gains `score: {correct, total} | null`; `GET /quests/daily` tasks gain `score`; `GET /api/v1/roadmap` (roadmap-tree plan, done/unmerged) days gain `score_correct`/`score_total` summed over the day's tasks and modules gain the same. Backend spec §6.2 updated; CODEMAP `quests`.
- Frontend: `QuestRow` and the retro world map (`retro-roadmap.md` day expansion and region tally) render the numbers; designer adds the two strings. Store types gain the optional fields.
- Tests: grader table (all right, some wrong, unknown id, letter case, no questions → null, legacy raw content → null); service test that a rejected request still writes nothing; integration test that `MarkComplete` persists answers and score; roadmap read-back; frontend unit tests for the row and the map.
- Estimate: one working day (backend ~5 h, frontend ~3 h).

## Evidence
- `backend/internal/quests/handler.go:40–44` — `UserAnswers json.RawMessage` accepted, never read; CODEMAP `quests` ("`user_answers` accepted, not persisted").
- `harness/plans/2026-09-24-typed-task-content-with-answer-keys-…` — `answer`/`explanation` in `content_json`; "Part 2 … decide then whether a score is persisted"; part 2 not yet planned.
- `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-…` — `GET /api/v1/roadmap` day/module shape this idea extends; `harness/designs/retro-roadmap.md` — the day expansion and region tally.
- `harness/plans/2026-09-26-day-28-checkpoint-…` — re-grades from a quiz because no performance data exists.
- Backend spec §6.2 (`user_answers` in the request); 1st-thinking §1 (CEFR progression), §3.2 (`exercises`).
- Knowledge tracing and answer-level analytics in adaptive language systems: https://www.sciencedirect.com/science/article/pii/S266630742300030X ; option tracing (what was answered, not only whether correct): https://arxiv.org/pdf/2104.09043 ; progress visibility and retention: https://trophy.so/blog/how-to-create-duolingo-style-progress-reports-for-your-app
