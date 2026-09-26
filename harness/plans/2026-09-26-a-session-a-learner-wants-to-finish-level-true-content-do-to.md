---
idea: harness/ideas/2026-09-25-run-01/a-session-a-learner-wants-to-finish-level-true-content-do-to.md
status: approved
priority: high
merged: false
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
