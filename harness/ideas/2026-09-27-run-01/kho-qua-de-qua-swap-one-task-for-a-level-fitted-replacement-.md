---
type: feature
status: proposed
source: ideator
run: 2026-09-27-run-01
order: 1
---
# Khó quá / dễ quá: swap one task for a level-fitted replacement through the unused exercise_generation route

## Why
`docs/PRODUCT.md` names the second reason self-study learners quit: "Generic courses are too easy or too hard, so learners get bored or lost." The spec's answer is "AI-personalized roadmaps" (1st-thinking §1), but today personalisation ends at minute two of onboarding: ten quiz items pick a level, and all 84 tasks are fixed from that one guess. A learner who opens day 5, task 2 and cannot do it has two options — grind through a task pitched wrong, or close the app. Either way the 30 minutes and the plant are at risk, and nothing in the product learns from it.

The only correction in flight is `POST /api/v1/roadmaps/regenerate` (plan `2026-09-26-59-of-84-…`, done/unmerged): it replaces the *whole* roadmap one CEFR step up or down, wipes the 28-day map the learner has been walking, and is not something anyone does in the middle of a task. Meanwhile `airouter` has routed `exercise_generation` → DeepSeek since the MVP (`internal/airouter/types.go:17`, strategies map) and **no package calls it** — a paid-for, cheap (~400 output tokens) capability that the "adaptive" in the product name is waiting on.

Flow-theory results on adaptive learning are consistent with the product's own thesis: learners persist when tasks are neither too easy nor too hard, and sessions with low solving probability show higher drop-out. A one-tap "this one is too hard" that replaces the task in place, keeps the timer running and costs one small AI call is the smallest honest version of "adaptive" a learner can actually feel.

## Expected output
User-visible:
- In the learning room, under the task content, a quiet row "Bài này…" with two chips **Khó quá** / **Dễ quá** (designer owns the wording and placement inside the retro learning-room design). Tapping one asks once ("Đổi bài này lấy bài vừa sức hơn?"), then within ~10–30 s the task is replaced *in place* by a new one of the same `task_type`, same `duration_minutes`, same day topic, pitched one CEFR step down (or up). The countdown keeps running — the learner was studying the whole time and those minutes count as today.
- Limits: one swap per task, at most 3 swaps per local day, never on a completed task. The learner's hint is remembered on the exercise so later adaptation (day-28 checkpoint, regenerate, idea 3 of this run) can read it.
- Failure never costs anything: on `rate_limited`, `ai_unavailable`, `ai_timeout` or `ai_bad_output` the original task stays and a toast says "Chưa tạo được bài mới, cậu cứ tiếp tục bài này nhé."; on the daily cap, the chips are disabled with "Hôm nay đã đổi 3 bài".

Technical (backend `quests` + `airouter`; frontend learning room):
- `POST /api/v1/quests/{exercise_id}/swap` `{hint: "too_hard" | "too_easy"}` behind `auth.Require()`. Order: validate (400 `invalid_request`) → `QuestRepo.CheckExercise` (same rules as progress: caller's active roadmap, today's `day_number`; else 404 `exercise_not_found`) → 409 `exercise_completed` if `is_completed` → per-day cap via `INCR quests:swap:{user_id}:{local-date}` + `EXPIRE` 48 h (key builder in `store/keys.go`, an addition to §4 like `pet:revive`; over 3 → 429 `swap_limit`, and the INCR is undone) → the existing `ratelimit:ai` slot (429 `rate_limited`) → `Route(TaskExerciseGen)` with a new `airouter.ExerciseSystemPrompt` / `ExerciseUserPrompt(level±1 clamped A1..C2, goal, task_type, day title, module focus, original title, hint)` under the default 30 s `TaskTimeout`, validated by the typed-content validator (`validateContent`, plan `2026-09-24-typed-task-content-…`, done/unmerged — same shape rules as a roadmap task), retry once on malformed → 502 `ai_bad_output` → one `UPDATE exercises SET content_json, difficulty_hint, swapped_at`. Response: the task in the `/quests/daily` task shape, so the store replaces it in `daily.tasks` by id.
- Migration `000N`: `exercises.difficulty_hint TEXT NULL CHECK (difficulty_hint IN ('too_hard','too_easy'))`, `exercises.swapped_at TIMESTAMPTZ NULL`; backend spec §3.2 DDL block appended (AGENTS.md rule), §6.2 gains the endpoint block, 1st-thinking §7 gains the row; CODEMAP `quests` and `airouter` updated (the strategies entry finally has a caller).
- Frontend: `stores/quest.ts` gains `swap(exerciseId, hint)`; `pages/learn/[id].vue` renders the chips and the in-place reload (content re-classified through `utils/content.ts`); the task timer is untouched (task-timer plan keys on task id).
- Tests: service tests with a scripted provider through the real router (prompt carries level−1 for `too_hard`, A1 floor / C2 ceiling, the daily cap, completed task, malformed-then-valid retry, provider failure leaves the row untouched); handler tests for every status; a `TEST_DATABASE_URL`-gated integration test that a swap persists `content_json` + `difficulty_hint`; store unit test for the in-place replacement.
- Estimate: one working day (backend ~5 h, frontend ~3 h). Needs the designer role for the learning-room control. Depends on the typed-content backend plan for the validator (information only).

## Evidence
- `docs/PRODUCT.md` "The problem" (too easy / too hard) and "Wrong level → placement quiz" — the only level decision in the product today.
- `backend/internal/airouter/types.go:17` `TaskExerciseGen`; `router.go` strategies `exercise_generation → deepseek`; `grep -rn exercise_generation backend/internal` finds no caller outside `airouter`.
- `harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-…` — whole-roadmap regenerate, one CEFR step; not a per-task correction.
- `harness/plans/2026-09-24-typed-task-content-with-answer-keys-…` — the per-task `content` shape and `validateContent` this idea reuses.
- 1st-thinking §1 (adaptive, AI-personalised), §6.2 (router), §4 (`ratelimit:ai`); backend spec §6.2.
- Adaptive difficulty and motivation / drop-out: https://www.sciencedirect.com/science/article/abs/pii/S0360131513001711 ; student-facing adaptive interventions and drop-out when solving probability is low: https://arxiv.org/pdf/2306.07853 ; LLM exercise generation for language learning: https://arxiv.org/pdf/2306.02457
