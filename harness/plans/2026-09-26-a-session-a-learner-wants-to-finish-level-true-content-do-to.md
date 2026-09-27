---
idea: harness/ideas/2026-09-25-run-01/a-session-a-learner-wants-to-finish-level-true-content-do-to.md
status: done
priority: high
merged: false
branch: harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to
worktree: .worktrees/a-session-a-learner-wants-to-finish-level-true-content-do-to
---
# Level-true content: the roadmap prompt carries a CEFR descriptor and the goal's register, and a deterministic placement floor stops a 10/10 learner being graded A2 — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F3** of 2026-09-26. **Estimate:** 3 h. **Branch:** `harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to`.

**Idea:** `harness/ideas/2026-09-25-run-01/a-session-a-learner-wants-to-finish-level-true-content-do-to.md` — this plan is cause (3) of its Why; causes (1), (2), (4) are owned by the regenerate route (today, bug queue), retro plan 3 and the `growth-moment` draft (see the idea's `## Evaluation`). Backend only; **no design doc**.

**Goal:** A B1 Business-English learner gets a roadmap written for B1 in workplace language (email, meetings, quotations), not "Good morning / one, two, three"; and a learner who answers every placement item correctly cannot be graded below the level their answers prove.

**Architecture:** `airouter` gains `level.go` (`LevelGuidance`) used by `RoadmapUserPrompt`; `onboarding` gains `GradeFloor` in `grade.go` and one `max` in `Assess`. `RoadmapSystemPrompt` stays verbatim (§6.1; `prompt_test.go` pins it) — the guidance goes in the **user** turn. `RoadmapSchema` is not touched (the typed-content branch owns it).

**⚠ Merge note:** `origin/harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` edits `prompt.go` (`RoadmapSchema`) and `roadmap_test.go`'s fixture; today's bug plan B2 edits `onboarding/service.go` (`Regenerate`) and `repo.go`. Keep this plan's `prompt.go` edit to the body of `RoadmapUserPrompt` only, `service.go` to the three lines after `ParsePlacement`, and all new code in `level.go`/`grade.go`. If `origin/main` has either when you start, merge it first, keep both sides.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` not installed (`grep -n`, `go test -timeout 60s`); `gofmt -l internal/airouter internal/onboarding` empty.
- The floor only **raises** a level, never lowers one, and never exceeds the bank's highest level (C1 today — `C2` stays reachable only through the model, which the existing enum test covers).
- Prompt text is English, plain, no markdown headings inside the user turn (Gemini/OpenAI JSON mode both tolerate prose before the schema — the existing prompt already does this).

## Review Focus
1. `LevelGuidance` has an entry for all six levels; `RoadmapUserPrompt("B1", "Business English", 30)` contains the B1 descriptor, the sentence `Every task must use the language of the learner's goal ("Business English")`, and the rule about never falling back to beginner content unless the level is A1.
2. `GradeFloor`: 10/10 → `C1`; A1+A2 items right, B1 one right, rest wrong → `A2`; only one A1 item right → `A1` (floor is at least A1); all wrong → `A1`; answers with unknown ids are skipped.
3. In `Assess`, `level = maxLevel(aiLevel, GradeFloor(req.Answers))`, applied before `StageLevel` so a retry reuses the floored level; a log line shows both values when they differ.
4. The existing fake-provider onboarding tests still pass unchanged (they answer a fixed level; with the fixture answers the floor must not raise it — check `fakes_test.go`'s answers and, if they are 10/10, assert the raised level instead and say so in the commit).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/level.go` + `level_test.go` | `LevelGuidance(level string) string`; descriptor table |
| `backend/internal/airouter/prompt.go` + `prompt_test.go` | `RoadmapUserPrompt` body |
| `backend/internal/onboarding/grade.go` + `grade_test.go` | `GradeFloor(answers []Answer) string`, `maxLevel(a, b string) string` |
| `backend/internal/onboarding/service.go` + `service_test.go` | apply the floor |
| `harness/CODEMAP.md` | `airouter`, `onboarding` bullets |

## Tasks

### Task 1: `LevelGuidance` and the user prompt

**Files:** `backend/internal/airouter/level.go`, `level_test.go`, `prompt.go`, `prompt_test.go`.

- [ ] **Step 1 (tests first):** `level_test.go` — `TestLevelGuidanceCoversEveryLevel` (each of `A1..C2` returns a non-empty string containing the level code and the words `vocabulary`, `grammar`, `passage`); `TestLevelGuidanceUnknownLevelFallsBackToB1` (documented default). `prompt_test.go` — `TestRoadmapUserPromptCarriesLevelAndGoalGuidance`: output contains `LevelGuidance("B1")`, `Business English` inside the goal-register sentence, the phrase `never generic greetings, numbers or classroom basics unless the level is A1`, and still contains `RoadmapSchema` and `Daily study time: 30 minutes` (existing pin).
- [ ] **Step 2:** `level.go`: `var levelGuidance = map[string]string{ "A1": "…", … "C2": "…" }` — each 2–3 sentences: vocabulary range (e.g. B1 ≈ 2,000–2,500 word families; B2 ≈ 4,000), grammar to use (B1: past perfect, first/second conditional, passive, reported speech; B2: mixed conditionals, inversion, cleft sentences…), reading passage length (A1 60–90 words · A2 90–130 · B1 130–180 · B2 180–250 · C1 250–320 · C2 300–380) and expected question difficulty. `func LevelGuidance(level string) string` (unknown → B1).
- [ ] **Step 3:** `RoadmapUserPrompt`: after the profile block add
```
Write every task for a %s learner. %s
Every task must use the language of the learner's goal ("%s"): its situations, vocabulary and register. Day 1 starts inside that goal — never generic greetings, numbers or classroom basics unless the level is A1.
```
  (`cefrLevel`, `LevelGuidance(cefrLevel)`, `targetGoal`), keeping the schema instruction last.
- [ ] **Step 4:** `go test -timeout 60s ./internal/airouter -run 'Level|RoadmapUserPrompt|Prompt' -v`. Commit: `airouter: roadmap user prompt carries a CEFR descriptor and the goal's register`.

### Task 2: The placement floor

**Files:** `backend/internal/onboarding/grade.go`, `grade_test.go`, `service.go`, `service_test.go`.

- [ ] **Step 1 (tests first):** `grade_test.go` — `TestGradeFloor` table: `all correct → C1`; `q1..q4 correct, q5 correct, q6 wrong, rest wrong → A2` (B1 has one of two right → not both → floor A2); `q1..q6 correct, q7 correct, q8 wrong → B1`; `q1 correct only → A1`; `none → A1`; `unknown ids only → A1`. `TestMaxLevel` (`maxLevel("A2","B1") == "B1"`, `("C1","A1") == "C1"`, unknown strings → the known one). `service_test.go` — `TestAssessRaisesTheAILevelToTheFloor`: fake grader answers `{"cefr_level":"A2"}`, request answers all correct → `AssessedLevel == "C1"`, the roadmap user prompt sent to the fake generator contains `C1`, `StageLevel` stored `C1`; `TestAssessKeepsAHigherAILevel`: grader `B2`, answers floor `A2` → `B2`.
- [ ] **Step 2:** `grade.go`: `GradeFloor(answers []Answer) string` — per level count correct/total from `Bank` via `Lookup`; walk `A1..C1` in order; the floor advances to level L while both items at L are correct; stop at the first level that fails; return the last passed level or `A1`. `maxLevel` by index in `cefrOrder` (define it here if B2's `cefrOrder` is not on the branch; if it is, reuse — one definition).
- [ ] **Step 3:** `service.go`, in `Assess` right after `ParsePlacement` sets `level` (inside the `level == ""` branch, before `StageLevel`): `if floor := GradeFloor(req.Answers); floor != level { if raised := maxLevel(level, floor); raised != level { log.Printf("onboarding: placement %s raised to %s by the answer floor for %s", level, raised, userID); level = raised } }`.
- [ ] **Step 4:** `go test -timeout 60s ./internal/onboarding -count=1 -v -run 'Grade|Assess|MaxLevel'`; whole package green. Commit: `onboarding: a deterministic placement floor — the answers can raise the AI's level, never lower it`.

### Task 3: CODEMAP

- [ ] **Step 1:** `airouter` bullet: "`RoadmapUserPrompt` adds `LevelGuidance(level)` (vocabulary/grammar/passage-length descriptor per CEFR level) and the goal-register rule"; `onboarding` bullet: "`GradeFloor` … `level = max(ai, floor)`, logged when it raises".
- [ ] **Step 2:** Commit: `codemap: level guidance and placement floor`. Push.

## Verification
```
cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
go test ./internal/airouter -run TestRoadmapUserPromptCarriesLevelAndGoalGuidance -v
go test ./internal/onboarding -run 'TestGradeFloor|TestAssessRaisesTheAILevelToTheFloor' -v
git push -u origin harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to
```
Live proof (owner, after the ship, via the regenerate route from plan B2): `POST /api/v1/roadmaps/regenerate {"cefr_level":"B1"}` → day-1 tasks are workplace English (record the three day-1 titles in the Execution summary follow-up).

## Execution summary

Built exactly to plan: Task 1 (`airouter/level.go` — `LevelGuidance`, all six CEFR descriptors; `prompt.go`'s `RoadmapUserPrompt` gains the level/goal-register sentences, schema kept last, `RoadmapSystemPrompt` untouched), Task 2 (`onboarding/grade.go` — `GradeFloor` walks `A1..C1` requiring both bank items of a level correct to advance past it, so a wrong A1 item caps the floor at A1 regardless of later answers; `maxLevel`/`cefrIndex`; `service.go`'s `Assess` applies `level = max(ai, floor)` right after grading, logged when it raises), Task 3 (CODEMAP `airouter`/`onboarding` bullets).

**Deviation (anticipated by the plan's own Review Focus #4):** the fixed fake-provider fixture `validRequest()` in `fakes_test.go` answers every bank item correctly (10/10), so `GradeFloor` is `C1` for it — higher than the harness's scripted `B1` grade. Updated the tests that exercise the full grading path and assert on the resulting level to expect the raised `C1` instead of `B1`: `TestAssessHappyPathGradesGeneratesAndPersistsOnce` and `TestAssessKeepsTheGradeWhenTheRoadmapFailsAndSkipsGradingOnTheRetry`. Added `TestAssessRaisesTheAILevelToTheFloor` (AI grades A2, 10/10 answers → C1) and `TestAssessKeepsAHigherAILevel` (AI grades B2, only the A1 pair right → floor stays A1, B2 kept) per the plan's Step 1. All other existing tests are unchanged and still pass — they either don't reach the grading branch (idempotent/validation/rate-limit/no-provider paths) or don't assert on the resulting level.

### Verification output

```
$ cd backend && go build ./... && gofmt -l . && go vet ./... && go test -timeout 120s ./... -count=1 -race
ok  	.../backend/cmd/api	2.776s
ok  	.../backend/internal/airouter	2.351s
ok  	.../backend/internal/auth	4.079s
ok  	.../backend/internal/config	1.956s
ok  	.../backend/internal/google	3.047s
ok  	.../backend/internal/health	3.542s
ok  	.../backend/internal/middleware	4.403s
ok  	.../backend/internal/notify	5.088s
ok  	.../backend/internal/onboarding	5.491s
ok  	.../backend/internal/pet	6.027s
ok  	.../backend/internal/quests	5.113s
ok  	.../backend/internal/secrets	5.090s
ok  	.../backend/internal/store	5.155s
(gofmt -l and go vet: no output — clean)

$ go test ./internal/airouter -run TestRoadmapUserPromptCarriesLevelAndGoalGuidance -v
--- PASS: TestRoadmapUserPromptCarriesLevelAndGoalGuidance

$ go test ./internal/onboarding -run 'TestGradeFloor|TestAssessRaisesTheAILevelToTheFloor' -v
--- PASS: TestGradeFloor (+ 6 subtests)
--- PASS: TestAssessRaisesTheAILevelToTheFloor
```

### Runtime proof

Booted the real compiled `cmd/api` binary against a fresh, isolated dev stack (`COMPOSE_PROJECT_NAME=level-true`, Postgres `55611`, Redis `56611`, API `8611`) plus a small local stub standing in for the Gemini endpoint (`GEMINI_BASE_URL` pointed at `127.0.0.1:45123`, no real provider keys available in this unattended run) — this exercises the actual HTTP path (Gin routing, Postgres, Redis session store, JWT auth, `airouter.Router`, `onboarding.Service`), not a unit-test fake.

- `GET /healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`.
- Minted a real session (inserted a `users` row, signed a JWT with `JWT_SECRET`, wrote `sess:{user_id}:token` to Redis) and called `GET /api/v1/onboarding/quiz` with it → `200`, ten questions, real DB/Redis/JWT round trip.
- `POST /api/v1/onboarding/assessment` with `target_goal: "Business English"` and every bank item answered correctly → `201 {"status":"success","assessed_level":"C1", ...}`. The stub always grades placement `B1`; the response's `C1` is the placement floor raising it, observed through the real API, not a test double.
- The stub's request log confirms the roadmap-generation prompt actually sent by the running `airouter.Router` carried `Current CEFR level: C1`, `Write every task for a C1 learner.`, and `Every task must use the language of the learner's goal ("Business English"): its situations, vocabulary and register.` — `LevelGuidance("C1")` and the goal-register rule, live.
- Cleaned up: killed the API process and the stub, `docker compose -p level-true down` (containers, network and volume all removed — verified with `docker ps`), removed the scratch `.env` and the scratch `cmd/tmpmint` helper (never committed).

Actual AI-generated day-1 titles are not available from this proof (no live Gemini/OpenAI/DeepSeek key in this unattended run) — the plan's own "Live proof" line already defers that check to the owner post-ship via the regenerate route.

**CI:** branch `harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to`, run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36226444437 — all five jobs (`frontend`, `backend-unit`, `backend-integration`, `docker-images`, `harness-tooling`) green.
