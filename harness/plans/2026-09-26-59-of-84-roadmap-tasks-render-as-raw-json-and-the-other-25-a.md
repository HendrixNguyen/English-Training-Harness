---
idea: harness/ideas/_inbox/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md
status: done
priority: high
merged: false
branch: harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a
worktree: .worktrees/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a
---
# Roadmap regeneration: `POST /api/v1/roadmaps/regenerate` replaces the active roadmap (optionally one CEFR step up or down) so a roadmap stored before the typed-content contract can be re-made — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B2** of 2026-09-26. **Estimate:** 3.5 h. **Branch:** `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a`.

**Idea:** `harness/ideas/_inbox/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md` — this plan is the **"regenerate stored roadmaps" part** of its Expected output. The backend contract is already on the `typed-task-content` branch (see the idea's `## Evaluation`); the frontend half is `harness/designs/retro-learning-room.md` (retro plan 3, written tomorrow).

**Goal:** An authenticated learner (or the owner with the learner's token) can call `POST /api/v1/roadmaps/regenerate` and get a fresh 28-day roadmap that replaces the active one in a single transaction — at the current level, or one CEFR step up/down — using the same generation path, parser and limiter as onboarding, so the owner's pre-contract roadmap is replaced by a typed one the moment the contract is on `main`.

**Architecture:** `onboarding` package only (CODEMAP: it owns `roadmaps`/`exercises` writes and `users.cefr_current`). `Service.Regenerate` reuses `routeJSON`/`route` (`TaskRoadmapGen`, `RoadmapSystemPrompt`, `RoadmapUserPrompt`) and the `ratelimit:ai` limiter; `Repo` gains `ReplaceRoadmap` (users row lock via `UPDATE users SET cefr_current`, deactivate, insert roadmap + 84 exercises — the same statements `SaveAssessment` runs minus goal/timezone/notification). Handler `RegenerateHandler` mounts under `guarded` in `cmd/api/main.go`. No new package, no migration.

**Wire contract (not in either spec — added here; the backend spec §6.1 gets it as "6.1.3", and the §7 / 1st-thinking §7 endpoint lists gain the route):**

```
POST /api/v1/roadmaps/regenerate        Authorization: Bearer …
{ "cefr_level": "B1" }                  # optional; must be users.cefr_current ± one step (A1..C2)
201 { "status": "success", "assessed_level": "B1", "roadmap_id": "<uuid>" }
400 invalid_request   (level not in the enum, or more than one step from the current level)
404 no_active_roadmap (never onboarded — use POST /onboarding/assessment)
429 rate_limited · 503 ai_unavailable · 502 ai_bad_output · 504 ai_timeout · 502 ai_upstream_failed · 500 internal_error
```
The old roadmap's `exercises` rows stay (history); `daily_progress` and the pet are untouched; `day_number` restarts at 1 because `quests` counts from `roadmaps.created_at` (CODEMAP `quests`). Google sync rebuilds the task list on the next `POST /integrations/google/sync` because `roadmap_id` changed (CODEMAP `google`) — nothing here calls Google.

**⚠ Merge note:** `origin/harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` also edits `backend/internal/onboarding/fakes_test.go` (`fixtureRoadmap` gets typed content). If `origin/main` has it when you start, `git fetch origin main && git merge origin/main --no-edit` first. Do not touch `fixtureRoadmap`'s content shape here; only add the new fakes described below.

## Global Constraints
- Work in `.worktrees/<slug>`; never edit the main checkout. Run Go from `backend/`. `rg`/`timeout` not installed: `grep -n`, `go test -timeout 120s`. Integration tests need `TEST_DATABASE_URL`/`TEST_REDIS_URL` (`make up` with `COMPOSE_PROJECT_NAME=<slug>` and free ports in the scratch `backend/.env`; `make down` after).
- Same error sentinels and the same handler `switch` order as `AssessmentHandler`; `ErrInvalidRequest`, `ErrBadAIOutput`, `ErrAITimeout` already exist. Add `ErrNoActiveRoadmap`.
- The limiter is consumed **once per request**, before any AI call, exactly like `Assess`.
- No write happens before the roadmap has parsed (`ParseRoadmap` inside `routeJSON`); `ReplaceRoadmap` is the only write and it is one transaction.
- `gofmt -l internal/onboarding cmd/api` prints nothing.

## Review Focus
1. `ReplaceRoadmap` runs `UPDATE users SET cefr_current = $2::cefr_level WHERE id = $1` **first** (row lock → two concurrent regenerates serialise; the second sees the first's roadmap deactivated and still ends with exactly one active roadmap) — integration test proves one active roadmap after two sequential calls, and a test that the users row is locked is not required (same guarantee `SaveAssessment` relies on).
2. Level step rule: from `B1`, `A2`/`B1`/`B2` are accepted, `C1` is `invalid_request`; from `A1`, `A1`/`A2`; from `C2`, `C1`/`C2`. Omitted → current level.
3. A user with no active roadmap gets `404` **before** the limiter is consumed and before any AI call.
4. The request body may be absent or `{}` (Gin binds an empty body as invalid — use `ShouldBindJSON` only when `c.Request.ContentLength > 0`, or a `binding:"omitempty"` tag; test both).
5. Nothing else in `Assess` changes; `TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious` still passes.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/onboarding/types.go` | `RegenerateRequest{CEFRLevel string \`json:"cefr_level"\`}`, `RegenerateResult` |
| `backend/internal/onboarding/repo.go` | `Profile` gains `TargetGoal`; `Repo` gains `ReplaceRoadmap`; SQL `updateLevelSQL` |
| `backend/internal/onboarding/service.go` | `ErrNoActiveRoadmap`; `Service.Regenerate`; `stepAllowed(current, requested)` |
| `backend/internal/onboarding/handler.go` | `RegenerateHandler(svc)` |
| `backend/internal/onboarding/{service,handler,integration}_test.go`, `fakes_test.go` | tests; fake repo gains `ReplaceRoadmap` + call log |
| `backend/cmd/api/main.go` | `guarded.POST("/roadmaps/regenerate", onboarding.RegenerateHandler(onboardingSvc))` |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §6.1 addition + §7 endpoint row |
| `project-base/1st-thinking-architecture-doc.md` | §7 endpoint list row |
| `harness/CODEMAP.md` | `onboarding` bullet |

## Tasks

### Task 1: `Repo.ReplaceRoadmap` and the richer `Profile`

**Files:** `backend/internal/onboarding/repo.go`, `backend/internal/onboarding/integration_test.go`, `backend/internal/onboarding/fakes_test.go`.

- [ ] **Step 1 (test first):** In `integration_test.go` add `TestIntegrationReplaceRoadmapDeactivatesPreviousAndKeepsHistory`: seed a user + `SaveAssessment` (existing helper pattern), call `ReplaceRoadmap(ctx, userID, "B2", fixtureRoadmap())`; assert `roadmaps` has 2 rows with exactly one `is_active`, the new one active; `exercises` count = 168 (84 old + 84 new); `users.cefr_current = 'B2'`; `target_goal`/`timezone`/`notification_time` unchanged. A second `ReplaceRoadmap` → still exactly one active. `ReplaceRoadmap` for an unknown user → `ErrUnknownUser`.
- [ ] **Step 2:** `Profile` gains `TargetGoal string`; `profileSQL` becomes `SELECT COALESCE(cefr_current::text, 'A1'), COALESCE(target_goal, '') FROM users WHERE id = $1`; `Profile()` scans both.
- [ ] **Step 3:** Add to the `Repo` interface: `// ReplaceRoadmap sets cefr_current, deactivates active roadmaps and inserts the new active roadmap with its 84 exercises, atomically. Returns the roadmap id.` `ReplaceRoadmap(ctx, userID, cefrLevel string, roadmap airouter.Roadmap) (string, error)`. Implement it by extracting the deactivate + insert + batch section of `SaveAssessment` into a private `insertActiveRoadmap(ctx, tx, userID, roadmap) (string, error)` used by both; `ReplaceRoadmap` runs `updateLevelSQL = UPDATE users SET cefr_current = $2::cefr_level WHERE id = $1` (0 rows → `ErrUnknownUser`) then `insertActiveRoadmap`, in one tx.
- [ ] **Step 4:** `fakes_test.go`: the fake repo records `ReplaceRoadmap` calls (`replaceCalls []struct{level string; roadmap airouter.Roadmap}`), returns a fixed id or a scripted error; `Profile` returns a configurable `TargetGoal`.
- [ ] **Step 5:** `go test -timeout 120s ./internal/onboarding -run 'Integration' -count=1 -p 1` green (with the test stack up). Commit: `onboarding: Repo.ReplaceRoadmap — swap the active roadmap and set cefr_current in one transaction`.

### Task 2: `Service.Regenerate`

**Files:** `backend/internal/onboarding/service.go`, `types.go`, `service_test.go`.

- [ ] **Step 1 (tests first), in `service_test.go` with the existing fakes:**
  - `TestRegenerateReplacesTheActiveRoadmapAtTheCurrentLevel`: active roadmap present, profile `B1`/goal `"Business English"`, generator scripted with `fixtureRoadmap()` JSON → one `Route` call with `TaskRoadmapGen`, user prompt contains `Current CEFR level: B1` and `Business English`; repo `replaceCalls` has one entry with level `B1`; result `RoadmapID` = fake id, `AssessedLevel` = `B1`; limiter called once.
  - `TestRegenerateAcceptsOneStepAndRejectsTwo`: table over (current, requested, want) — `B1,B2,ok` · `B1,A2,ok` · `B1,C1,ErrInvalidRequest` · `A1,A1,ok` · `C2,B2,ErrInvalidRequest` · `B1,"b2",ErrInvalidRequest` · `B1,"",ok→B1`; on error the limiter and generator are not called.
  - `TestRegenerateWithoutAnActiveRoadmapIs404`: `ActiveRoadmapID` ok=false → `ErrNoActiveRoadmap`; limiter not called.
  - `TestRegenerateAIFailureWritesNothing`: generator returns `airouter.ErrAllProvidersFailed` → error returned, `replaceCalls` empty; a malformed body twice → `ErrBadAIOutput`, `replaceCalls` empty.
- [ ] **Step 2:** `types.go`: `RegenerateRequest{CEFRLevel string \`json:"cefr_level"\`}` and `RegenerateResult{Status, AssessedLevel, RoadmapID}` (json tags as in the contract).
- [ ] **Step 3:** `service.go`: `var ErrNoActiveRoadmap = errors.New("onboarding: no active roadmap")`; `var cefrOrder = []string{"A1","A2","B1","B2","C1","C2"}`; `stepAllowed(current, requested string) bool` (index distance ≤ 1); `func (s *Service) Regenerate(ctx, userID string, req RegenerateRequest) (RegenerateResult, error)`: `ActiveRoadmapID` (ok=false → `ErrNoActiveRoadmap`) → `Profile` → level = `req.CEFRLevel` or `profile.CEFRCurrent`; `!cefrLevels[level] || !stepAllowed(...)` → `fmt.Errorf("%w: cefr_level must be within one step of %s", ErrInvalidRequest, current)` → `s.limiter.Allow` → `s.routeJSON(ctx, airouter.TaskRoadmapGen, airouter.RoadmapSystemPrompt, airouter.RoadmapUserPrompt(level, profile.TargetGoal, DailyMinutes), parse)` → `s.repo.ReplaceRoadmap(ctx, userID, level, roadmap)` → result. Log one line `onboarding: regenerated roadmap %s for %s at %s`.
- [ ] **Step 4:** `go test -timeout 60s ./internal/onboarding -run Regenerate -v` green. Commit: `onboarding: Service.Regenerate — a fresh roadmap at the current level or one step away`.

### Task 3: Handler and route

**Files:** `backend/internal/onboarding/handler.go`, `handler_test.go`, `backend/cmd/api/main.go`.

- [ ] **Step 1 (tests first), `handler_test.go` (same harness as the assessment handler tests):** rows — no body → `201` (level omitted); `{"cefr_level":"B2"}` → `201` with body `{"status":"success","assessed_level":"B2","roadmap_id":…}`; `{"cefr_level":"C1"}` from `B1` → `400 invalid_request`; malformed JSON → `400 invalid_request`; no active roadmap → `404 no_active_roadmap`; `ErrRateLimited` → `429`; `ErrNoProviders` → `503 ai_unavailable`; `ErrBadAIOutput` → `502 ai_bad_output`; `ErrAITimeout` → `504 ai_timeout`; `ErrAllProvidersFailed` → `502 ai_upstream_failed`; no `user_id` in context → `401`.
- [ ] **Step 2:** `RegenerateHandler(svc *Service) gin.HandlerFunc`: user id from `auth.UserID`; body optional (`if c.Request.ContentLength != 0 { ShouldBindJSON }`); call `svc.Regenerate`; the same `switch` as `AssessmentHandler` plus `case errors.Is(err, ErrNoActiveRoadmap): 404 {"error":"no_active_roadmap"}`; success → `201`.
- [ ] **Step 3:** `main.go`: after the onboarding routes, `guarded.POST("/roadmaps/regenerate", onboarding.RegenerateHandler(onboardingSvc))`.
- [ ] **Step 4:** `go build ./... && go vet ./... && go test -timeout 120s ./internal/onboarding ./cmd/... -count=1`. Commit: `api: POST /api/v1/roadmaps/regenerate`.

### Task 4: Spec and CODEMAP

**Files:** the two spec documents, `harness/CODEMAP.md`.

- [ ] **Step 1:** Backend spec: under §6.1 (search `grep -n 'onboarding/assessment' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"`), add "6.1.3 `POST /api/v1/roadmaps/regenerate`" with the request/response block above (escape like the surrounding text). §7 endpoint table (if any) and the 1st-thinking §7 list (`grep -n 'Core REST'`) get one row each, marked "(added 2026-09-26)".
- [ ] **Step 2:** CODEMAP `onboarding` bullet: one sentence on `Regenerate`/`ReplaceRoadmap` (route, step rule, single tx, old exercises kept, `day_number` restarts, Google rebuilds on next sync).
- [ ] **Step 3:** Commit: `docs: roadmaps/regenerate in the specs and CODEMAP`.

## Verification
From `backend/` in the worktree:
```
go build ./... && gofmt -l . && go vet ./...
go test -timeout 120s ./... -count=1 -race
COMPOSE_PROJECT_NAME=<slug> make up   # then export TEST_DATABASE_URL/TEST_REDIS_URL per backend/.env.example
go test -timeout 300s ./... -run Integration -p 1 -count=1 -v | grep -E 'ReplaceRoadmap|SaveAssessment|^(ok|FAIL|---)'
make down
grep -n 'roadmaps/regenerate' cmd/api/main.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../project-base/1st-thinking-architecture-doc.md ../harness/CODEMAP.md
git push -u origin harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a   # CI: backend-unit + backend-integration green
```
**Owner's one-off, after tonight's daily PR ships (record in the Execution summary, not run by the executor):** with the PWA's token from `localStorage['aelp.auth']`, `curl -X POST "$API_URL/api/v1/roadmaps/regenerate" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"cefr_level":"B1"}'` → `201`; `GET /api/v1/quests/daily` then returns day 1 of a typed roadmap (`words[].term`, `questions[].options`).

## Execution summary

**Built exactly per plan, all 4 tasks, no deviations from the file structure or the Review Focus points.**

- Task 1: `Repo.ReplaceRoadmap` + richer `Profile` (`TargetGoal`). `SaveAssessment` and `ReplaceRoadmap` share the extracted `insertActiveRoadmap(ctx, tx, userID, roadmap)` (deactivate → insert roadmap → batch-insert 84 exercises). `updateLevelSQL` runs first inside `ReplaceRoadmap`'s own transaction, giving the row-lock serialisation the plan's Review Focus #1 calls for.
- Task 2: `Service.Regenerate` — `ActiveRoadmapID` (`ErrNoActiveRoadmap` → checked before the limiter or any AI call), `stepAllowed` bounds `cefr_level` to one step via `cefrOrder`, one `ratelimit:ai` slot, reuses `routeJSON`/`TaskRoadmapGen`/`RoadmapUserPrompt`/`ParseRoadmap`, then `ReplaceRoadmap`. Table-driven test covers every case in Review Focus #2 (`B1,C1→invalid`, `C2,B2→invalid`, `B1,b2→invalid` (case-sensitive), `B1,""→B1`).
- Task 3: `RegenerateHandler` — body optional via `c.Request.ContentLength != 0` gate (Review Focus #4: both no-body and malformed-body cases pass), same `switch` as `AssessmentHandler` plus `ErrNoActiveRoadmap → 404`. Mounted at `guarded.POST("/roadmaps/regenerate", …)` in `cmd/api/main.go`.
- Task 4: backend spec §6.1.3, 1st-thinking §7 endpoint list, and the CODEMAP `onboarding` bullet all updated; `grep -n 'roadmaps/regenerate'` across all four locations confirmed (see Verification output below).
- No merge was needed: `origin/main` did not yet have `harness/2026-09-25-medium-typed-task-content-with-answer-keys…` when the worktree was cut from `origin/main` at `67ad0c0`, so the merge note in the plan did not apply.

**Verification** (backend/, worktree `.worktrees/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a`):
```
go build ./... && gofmt -l . && go vet ./...         # clean
go test -timeout 120s ./... -count=1 -race           # ok, all 13 packages
COMPOSE_PROJECT_NAME=regenerate, POSTGRES_PORT=55442, REDIS_PORT=56389 (scratch backend/.env)
docker compose up -d --wait --wait-timeout 120       # postgres+redis healthy
TEST_DATABASE_URL/TEST_REDIS_URL exported; go test -timeout 300s ./... -run Integration -p 1 -count=1 -v
  → TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious PASS
  → TestIntegrationReplaceRoadmapDeactivatesPreviousAndKeepsHistory PASS (new)
  → every other package's Integration tests PASS, no skips
docker compose down                                  # containers + network removed
grep -n 'roadmaps/regenerate' cmd/api/main.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../project-base/1st-thinking-architecture-doc.md ../harness/CODEMAP.md
  → one match in each of the four files
```

**Runtime proof:** booted the real API (`go run ./cmd/api`, `PORT=18090`, real Postgres/Redis on the ports above, no AI keys) and exercised the live endpoint with `curl` against real bearer tokens minted through `auth.TokenIssuer` and sessions written to Redis (`sess:{user_id}:token`), for users seeded through `onboarding.PgRepo.SaveAssessment` and plain SQL — no shortcuts around auth or the service:
- No bearer token → `401 {"error":"unauthorized"}`.
- User with no active roadmap → `404 {"error":"no_active_roadmap"}`, before any AI call (per Review Focus #3).
- User with an active B1 roadmap, `{"cefr_level":"C2"}` (two steps away) → `400 {"error":"invalid_request"}`.
- User with an active B1 roadmap, `{"cefr_level":"B2"}` → `503 {"error":"ai_unavailable"}` — this environment has no `GEMINI_API_KEY`/`OPENAI_API_KEY`/`DEEPSEEK_API_KEY`, so `airouter.Route` correctly reports no providers; this is the documented, expected behaviour (CODEMAP `airouter`), not a bug. A genuine `201` needs a real provider key, which the executor does not hold — that's the plan's own "Owner's one-off" step above, to run after tonight's daily PR carries this branch (and the typed-content branch) to `main`.
- Cleaned up: killed the `go run`/compiled API process, `docker compose down` for the `regenerate` project, deleted the scratch `backend/.env` and the two throwaway `cmd/devseed*` seeding programs (never committed — `git status --short` was clean before every commit). `pgrep -fl exe/api` and `docker ps` both confirmed clear afterward.

**Deviations:** none from the plan's file structure, SQL, or test list. The two temporary `cmd/devseed`/`cmd/devseed2` programs used only for the runtime-proof step are not part of the plan's File structure table and were deleted before the final build/test pass and before every commit; they never touched git.

**Push + CI:** pushed `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` at `c74ac85`. CI run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36216835431 — **success** (backend-unit, backend-integration, docker-images, frontend, harness-tooling all green).

**Left for the owner:** the "Owner's one-off" curl against the real deployed API with a real AI provider key, once tonight's daily PR (carrying this branch and the typed-content branch) merges to `main` and ships.
