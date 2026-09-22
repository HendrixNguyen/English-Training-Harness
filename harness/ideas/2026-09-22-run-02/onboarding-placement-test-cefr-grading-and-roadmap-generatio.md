---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 6
priority: high
plan: harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md
---
# Onboarding placement test CEFR grading and roadmap generation

## Why
<!-- Business rationale. Tie to spec goals: retention (30 min/day), CEFR progression, gamified pet engine. -->

## Expected output
User-visible:
- A signed-in user with no active roadmap can fetch a placement quiz, answer it, and submit; within one request they get back their graded CEFR level and a 28-day roadmap.
- Re-submitting while a roadmap is already active does not create a second one (the existing active roadmap is returned).

Technical (`backend/internal/onboarding`):
- `GET /api/v1/onboarding/quiz` — issues the placement item set and seeds `quiz:placement:{user_id}` (Hash, TTL 2h, §4). Not in §7's list; add it to the §7 endpoint enumeration and to CODEMAP.
- `POST /api/v1/onboarding/assessment` (§7, line 672) — reads the staged answers, routes `TaskPlacementTest` through `airouter` to grade, writes `users.cefr_current` (+ `target_goal` from the request), then routes `TaskRoadmapGen` with the §6.1 system prompt.
- Roadmap persistence: one `roadmaps` row (`roadmap_json` JSONB, `is_active = true`, previous rows deactivated in the same transaction) plus `exercises` rows — 28 days × 3 tasks, `task_type` ∈ (`vocabulary`,`reading`,`practice`), `day_number` 1..28.
- Validation before persistence: reject AI output that is not 4 modules × 7 days × 3 tasks (reuse the roadmap JSON validator from the `airouter` slice); on failure retry once, then return 502 without writing.
- Redis key builder for `quiz:placement:*` lives in `store`; the 2h TTL is asserted in a test.
- Unit tests with a stubbed `LLMProvider`: grading path, shape validation (accept + reject), idempotent re-submit, exercise row count = 84.

Depends on: `store` (1) for tables/Redis, `auth` (2) for the authenticated user, `airouter` (5) for grading and generation. Blocks: `quests` (3) for real data, `google` (7), `frontend-shell` (9).

## Evidence
- Spec §5.1 sequence, lines 288–302: Submit Test → Route Task (Grade & Gen JSON) → Return Roadmap → push to Google.
- Spec §7, line 672: `POST /api/v1/onboarding/assessment` — "Submits placement quiz answers, grades level, triggers AI roadmap generation."
- Spec §4, line 263: `quiz:placement:{user_id}` Hash, 2-hour TTL.
- Spec §6.1: the roadmap system prompt — 4 modules × 7 daily quests = 28 days, 3 tasks of ~10 minutes each.
- Spec §6.2, lines 371/425: `TaskPlacementTest` and its `ProviderGemini` strategy entry.
- Spec §3.2: `roadmaps`, `exercises`, `users.cefr_current`, `users.target_goal`.
- `harness/CODEMAP.md`: the `onboarding` package entry, which run-02 did not cover — flagged in that run's `_run.md` Notes.

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** The section above is empty, so here is the argument the product makes for it: onboarding is the only path that writes `users.cefr_current`, `users.target_goal` and a `roadmaps` row, and §5.1 puts it before everything else a learner sees. Without it `GET /quests/daily` is a permanent 404 `no_active_roadmap` (the quests plan's own words), the pet never has a reason to grow, and the Google sync (7) has nothing to push. `harness/ideas/2026-09-22-run-02/_run.md` recorded onboarding as the one §7/§4 gap left by run-02; this slice closes it and retires the `store.SeedDemoRoadmap` stopgap the quests plan marked TEMPORARY.

**Is the *Expected output* achievable in one plan?** Yes, given airouter (5). Two routes, one question bank, two prompts, one transaction, one Redis hash. The LLM is stubbed everywhere, so the whole grading/generation path is a pure test.

**Dependencies:** `store` (1) and `auth` (2) merged; `quests` (3) executing — this plan edits its `integration_test.go` (to seed through onboarding's repo instead of `SeedDemoRoadmap`), so it lands after quests merges; `pet` (4) for the §6.1 `pet_state` in the response — via an onboarding-owned `Pet` interface (`Ensure(ctx, userID) (PetState, error)`) that `cmd/api/main.go` adapts `*pet.Service` to, so `onboarding` never imports `pet` and quests' internal test can import `onboarding` without an import cycle; `airouter` (5) for `Router`, `RateLimiter`, `RoadmapSystemPrompt`, `ParseRoadmap`, `Roadmap.Exercises()`.

**Spec reconciliation:**
- **`POST /api/v1/onboarding/assessment`** is the backend spec §6.1 contract field for field: request `{target_goal, notification_time, timezone, answers[{question_id, selected_option}]}`, response `201 {status: "success", assessed_level, roadmap_id, pet_state{plant_name, health_points, stage}}`. All four request fields are persisted (`users.target_goal`, `notification_time`, `timezone`, `cefr_current`), since §3.2 has the columns and nothing else writes them.
- **`GET /api/v1/onboarding/quiz` is not in either spec** (backend spec §6.1 lists only `/auth/google` and `/onboarding/assessment`; 1st-thinking §7 the same). It stays, because a client cannot answer questions it was never given, and the plan says so: response `{questions[{id, prompt, options{A,B,C,D}}]}` (correct answers and CEFR tags never leave the server). CODEMAP records it as an addition; the spec change is the human's.
- **`quiz:placement:{user_id}`** (§4: Hash, 2 h, "active placement test answers before grading"): the assessment handler `HSET`s the submitted answers and `EXPIRE`s 2 h *before* calling the grader, and `DEL`s on success, so a failed grading leaves the answers inspectable for two hours. `store.PlacementQuizKey` / `store.PlacementQuizTTL` already exist on `main`; a unit test asserts the TTL passed is `store.PlacementQuizTTL`.
- **Grading** routes `TaskPlacementTest` with the bank (each item's CEFR tag and correct option) plus the learner's answers and expects `{"cefr_level": "B1"}` — parsed against the `cefr_level` enum. **Generation** routes `TaskRoadmapGen` with `airouter.RoadmapSystemPrompt` and validates with `airouter.ParseRoadmap`. Each call retries **once** on a malformed body, then the request is a 502 `ai_bad_output` and **nothing has been written** — the users update, roadmap and 84 exercises all happen in one transaction after both AI calls succeed. `ErrRateLimited` → 429, `ErrNoProviders` → 503.
- **Persistence:** in one tx `UPDATE users …`, `UPDATE roadmaps SET is_active = FALSE WHERE user_id = $1 AND is_active`, `INSERT roadmaps (roadmap_json = the validated JSON, is_active = TRUE)`, 84 `INSERT exercises` with `content_json` = the task object (`title`, `duration_minutes` present — what quests' `toTask` reads). Two concurrent first submissions therefore end with exactly one active roadmap.
- **Idempotency:** a submit while an active roadmap exists returns `200` with the same body shape (`roadmap_id` of the active one, `assessed_level` = `users.cefr_current`) and makes no AI call and no write. Retaking the placement is a product decision recorded as an open question.
- **One rate-limit slot per assessment** (not per LLM call), so a legitimate retry-once path cannot trip the 5/min limit on its own.

**Priority rationale:** `high` — MVP `order` 6; the last backend slice before the integrations, and the one that makes quests (3) and pet (4) real for a new user.

**Test strategy:** stubbed `LLMProvider` (scripted responses per task type) behind a real `airouter.Router`; fakes for the repo, quiz store, limiter and pet; tests for happy path (201, 84 exercises with `title`/`duration_minutes`, users columns), idempotent re-submit (200, no AI call), bad-shape retry-then-502 with zero writes, 429, timezone/notification_time validation, quiz DTO never leaking answers; one `TestIntegrationSaveRoadmapPersists84ExercisesAndDeactivatesPrevious` on `TEST_DATABASE_URL`; the quests integration test is re-pointed at `onboarding.NewPgRepo(...).SaveRoadmap` and `store/seed.go` + `seed_test.go` are deleted.
