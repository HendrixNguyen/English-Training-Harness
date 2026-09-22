---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 6
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
