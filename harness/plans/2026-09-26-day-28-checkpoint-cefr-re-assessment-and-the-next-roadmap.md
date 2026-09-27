---
idea: harness/ideas/2026-09-24-run-01/day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md
status: draft
priority: medium
merged: false
---
# Day-28 checkpoint (backend): `POST /api/v1/roadmaps/next` re-grades the level and replaces the roadmap; `GET /quests/daily` says `roadmap_complete` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F5** of 2026-09-26. **Estimate:** 4 h. **Branch:** `harness/2026-09-26-medium-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap`.

**Idea:** `harness/ideas/2026-09-24-run-01/day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md` — **backend half**; the frontend (checkpoint card on the hub, level check reusing the quiz UI, roadmap history on `/roadmap`) is a separate plan with its own design after the retro hub/roadmap screens. Backend only; **no design doc**.

**Depends on (must be on `origin/main` before execution):** plan B2 of 2026-09-26 (`harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md`) — `onboarding.Repo.ReplaceRoadmap`, `RegenerateResult`, the handler error `switch`; and F3 (`GradeFloor`, `maxLevel`). If either is missing when you start, stop and report.

**Goal:** A learner past day 28 sees `roadmap_complete: true` on `GET /api/v1/quests/daily`, and `POST /api/v1/roadmaps/next` with fresh placement answers re-grades them (AI + floor), writes the new `cefr_current`, and replaces the roadmap with a new 28-day plan at that level in one transaction — pet, streak and history untouched.

**Architecture:** `quests` — additive `roadmap_complete` computed from the existing day arithmetic (`DayNumber`) and the day's target flag; `onboarding` — `Service.Next(ctx, userID, NextRequest{Answers})` = placement grading (`PlacementSystemPrompt`, `ParsePlacement`, `GradeFloor`, `maxLevel`) then the F3 prompt and `Repo.ReplaceRoadmap`; one limiter slot per request; new `Repo.ActiveRoadmapStartedAt` so the route can refuse a roadmap younger than 28 days (`409 roadmap_not_complete`) without reading `quests`' tables (the `quests.DayNumber(createdAt, now, loc)` pure function is imported — packages share functions, never tables).

**Wire contract (not in either spec — added; backend spec §6.1 "6.1.4", §7 rows):**
```
POST /api/v1/roadmaps/next     { "answers": [{question_id, selected_option} × 10] }
201 { "status": "success", "previous_level": "A2", "assessed_level": "B1", "roadmap_id": "<uuid>" }
400 invalid_request · 404 no_active_roadmap · 409 roadmap_not_complete · 429/503/502/504 as the assessment route
GET /api/v1/quests/daily        … + "roadmap_complete": true|false   (additive)
```
`roadmap_complete` = calendar days since `roadmaps.created_at` in the user's timezone `> 28`, or `== 28 && is_target_met`. Google: the task list is rebuilt on the next sync because `roadmap_id` changes (CODEMAP `google`); re-pushing the **calendar event** (its `COUNT=28` has run out) is a `google` follow-up filed by the executor as an inbox idea, not done here.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` not installed; integration tests via `COMPOSE_PROJECT_NAME=<slug> make up` … `make down`.
- Concurrency: `ReplaceRoadmap`'s `UPDATE users` row lock serialises two `next` calls; the second still re-grades and replaces (idempotency is not promised — the 5/min limiter bounds the cost). Say so in CODEMAP.
- No write before the roadmap has parsed; errors map exactly like `AssessmentHandler` plus `409 roadmap_not_complete`.
- `gofmt -l internal/quests internal/onboarding cmd/api` empty.

## Review Focus
1. `roadmap_complete` on day 27 = false; day 28 target not met = false; day 28 met = true; day 29+ = true, in the learner's timezone (table test around midnight `Asia/Saigon`).
2. `Next` order: validate answers → active roadmap (404) → started-at + `DayNumber` gate (409) → limiter → placement grade (AI + floor) → roadmap gen at the new level → `ReplaceRoadmap`. Nothing written on any failure.
3. `previous_level` comes from `Profile` before the write; `assessed_level` may equal it ("consolidating").
4. The onboarding `Assess` path is untouched (its "active roadmap → 200" guard stays).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/quests/{service,handler,types}.go` + tests | `roadmap_complete` |
| `backend/internal/onboarding/repo.go` | `ActiveRoadmapStartedAt(ctx, userID) (time.Time, bool, error)` |
| `backend/internal/onboarding/{types,service,handler}.go` + tests | `NextRequest`, `NextResult`, `ErrRoadmapNotComplete`, `Service.Next`, `NextHandler` |
| `backend/cmd/api/main.go` | `guarded.POST("/roadmaps/next", …)` |
| Backend spec §6.1/§7, 1st-thinking §7, `harness/CODEMAP.md` | contract |

## Tasks

### Task 1: `roadmap_complete` on the daily quest

**Files:** `backend/internal/quests/*.go`, `quests/*_test.go`.

- [ ] **Step 1 (tests first):** in the daily-quest service tests (find them: `grep -ln 'day_number' internal/quests/*_test.go`), a table over (created_at, now, target met) → `roadmap_complete` per Review Focus 1; handler test asserts the JSON key is present and boolean.
- [ ] **Step 2:** Add `RoadmapComplete bool \`json:"roadmap_complete"\`` to the daily response type; compute where `day_number` is computed: `days := calendar days (unclamped)`; `complete = days > RoadmapDays || (days == RoadmapDays && isTargetMet)`. (If the unclamped day count is not exposed, add `DaysSince(createdAt, now, loc) int` beside `DayNumber` in `day.go` and make `DayNumber` clamp it.)
- [ ] **Step 3:** `go test ./internal/quests -v -run 'Complete|Daily'`. Commit: `quests: GET /quests/daily reports roadmap_complete past day 28`.

### Task 2: `Service.Next`

**Files:** `backend/internal/onboarding/repo.go`, `types.go`, `service.go`, `service_test.go`, `integration_test.go`, `fakes_test.go`.

- [ ] **Step 1 (tests first):** `service_test.go` — `TestNextRegradesAndReplacesTheRoadmap` (fake grader `B1`, floor from answers `A2`, profile `A2`, started 29 days ago → one placement `Route`, one roadmap `Route` whose user prompt contains `B1`, `replaceCalls` = 1 at `B1`, result `{previous_level: A2, assessed_level: B1}`); `TestNextRefusesAYoungRoadmap` (started 20 days ago → `ErrRoadmapNotComplete`, limiter not called); `TestNextWithoutARoadmapIs404`; `TestNextAIFailureWritesNothing`; `TestNextValidatesAnswers` (empty / unknown id → `ErrInvalidRequest`, reuse `validate`'s answer rules). Integration: `ActiveRoadmapStartedAt` returns the active roadmap's `created_at`.
- [ ] **Step 2:** `repo.go`: `ActiveRoadmapStartedAt` (`SELECT created_at FROM roadmaps WHERE user_id=$1 AND is_active ORDER BY created_at DESC LIMIT 1`); fake in `fakes_test.go`.
- [ ] **Step 3:** `types.go`: `NextRequest{Answers []Answer}`, `NextResult{Status, PreviousLevel, AssessedLevel, RoadmapID}`. `service.go`: `ErrRoadmapNotComplete`; `Next` per Review Focus 2, using `quests.DayNumber`/`DaysSince` with `time.LoadLocation(profile.Timezone)` (add `Timezone` to `Profile`/`profileSQL` — `COALESCE(timezone,'UTC')`).
- [ ] **Step 4:** `go test ./internal/onboarding -run Next -v`. Commit: `onboarding: Service.Next — re-grade the level and replace the roadmap after day 28`.

### Task 3: Handler, route, docs

**Files:** `handler.go`, `handler_test.go`, `cmd/api/main.go`, the specs, CODEMAP.

- [ ] **Step 1 (tests first):** handler rows for every status in the contract, including `409 {"error":"roadmap_not_complete"}` and `401` without a user.
- [ ] **Step 2:** `NextHandler(svc)` (bind `NextRequest`, same `switch` as `RegenerateHandler` + the 409 case); `main.go`: `guarded.POST("/roadmaps/next", onboarding.NextHandler(onboardingSvc))`.
- [ ] **Step 3:** Backend spec §6.1 "6.1.4" block + §7 row; 1st-thinking §7 row; quests §6.2 response gains `roadmap_complete`; CODEMAP `quests`/`onboarding` bullets (gate, order, concurrency note, calendar-event follow-up).
- [ ] **Step 4:** File the inbox idea for the calendar-event re-push (`python3 tools/harness/cli.py new-idea --run harness/ideas/_inbox --title "Google calendar event is not re-pushed for the next roadmap" --type feature --source reviewer` is the reviewer's tool — the executor instead lists it in the Execution summary's *Follow-ups* for the reviewer to file). Commit: `api: POST /api/v1/roadmaps/next; specs and CODEMAP`.

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
COMPOSE_PROJECT_NAME=<slug> make up && go test -timeout 300s ./... -run Integration -p 1 -count=1 && make down
grep -n 'roadmaps/next' cmd/api/main.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../project-base/1st-thinking-architecture-doc.md ../harness/CODEMAP.md
grep -n 'roadmap_complete' internal/quests/*.go "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
git push -u origin harness/2026-09-26-medium-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap
```
