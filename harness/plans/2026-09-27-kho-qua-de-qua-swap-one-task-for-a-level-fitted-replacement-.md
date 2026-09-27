---
idea: harness/ideas/2026-09-27-run-01/kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md
status: draft
priority: medium
merged: false
order: 1
design: harness/designs/kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md
---
# Khó quá / Dễ quá — swap one task for a level-fitted replacement — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Design:** `harness/designs/kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md` — every frontend task cites its sections; `## Verification` repeats its Acceptance list verbatim.
**Idea:** `harness/ideas/2026-09-27-run-01/kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md`
**Goal:** From the learning room, a learner can replace one of today's incomplete tasks, in place, with a new task of the same type, duration and day topic pitched one CEFR step easier or harder — through the routed-but-unused `exercise_generation` AI route — without losing a second of the timer, at most once per task and three times per local day.

**Scope:** backend `quests` (new endpoint, cap, persistence), `airouter` (exercise prompt), `store` (migration, key builder); frontend `stores/quest.ts`, `components/learn/SwapRow.vue`, `pages/learn/[id].vue`, kit `RetroButton size`. **Estimate:** one working day (backend ~5 h, frontend ~3 h). **Branch:** `harness/2026-09-27-medium-kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-`.

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically
`harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (`airouter/content.go` `validateContent`, `SampleContent`),
`harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to` (`airouter/level.go` `LevelGuidance`, `prompt.go`),
`harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` (onboarding persists typed content),
`harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` (`stores/quest.ts` timers, `pages/learn/[id].vue`),
`harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`RetroButton`, `RetroPanel fog`, `CompanionSprite`, `useRetroToast`),
`harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase` and `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` (both add a `0004_*` migration — this plan's migration takes the next free number),
`harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint` (`store/keys.go`),
`harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s` (`airouter/router.go`, `timeouts.go`).
```
for b in 2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a 2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to 2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a 2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab 2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n 2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase 2026-09-26-medium-pet-streak-shield-earned-by-target-days 2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint 2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
ls backend/internal/store/migrations/
```
Any `MISSING` → stop and report `blocked`. If the Google-tasks plan (`2026-09-27-google-tasks-tick-…`) landed first it also added a migration — N is simply the next free number.

## Contract (backend spec §6.2 gains the block; 1st-thinking §7 gains the row)

`POST /api/v1/quests/{exercise_id}/swap` behind `auth.Require()`, body `{"hint": "too_hard" | "too_easy"}`.
Order (each step's failure writes nothing):
1. Body/UUID validation → `400 invalid_request`.
2. `QuestRepo.CheckExercise` with today's `day_number` (caller's active roadmap; if the catch-up plan has relaxed it to `≤ today`, pass today and additionally require the exercise's `day_number == today` in the swap read — swaps are for today's tasks only) → `404 exercise_not_found`; no active roadmap → `404 no_active_roadmap`.
3. Swap read (`SwapTarget`: `is_completed`, `difficulty_hint`, `task_type`, `content_json` title, `day_number`, the day's title and module focus from `roadmap_json`, `users.cefr_current`, the user's goal field, timezone) → `409 exercise_completed` if completed; `409 already_swapped` if `difficulty_hint` is already set (one swap per task — the UI never offers it; the server enforces it).
4. Daily cap: `INCR quests:swap:{user_id}:{local-date}` + `EXPIRE 48h` (`store.QuestSwapKey`); result > 3 → `DECR` and `429 swap_limit`.
5. `airouter` rate limiter `Allow` → `429 rate_limited` (and `DECR` the cap).
6. `Route(TaskExerciseGen, ExerciseSystemPrompt, ExerciseUserPrompt(...))` under the default 30 s `TaskTimeout`; parse as one `airouter.Task` and run `validateContent`; malformed/invalid → retry once → `502 ai_bad_output`; provider errors map exactly like `onboarding/handler.go` (`503 ai_unavailable`, `504 ai_timeout`). Any AI failure `DECR`s the cap (a failed swap is not counted — design §5).
7. One `UPDATE exercises SET content_json=$new, difficulty_hint=$hint, swapped_at=now() WHERE id=$id AND is_completed=FALSE AND difficulty_hint IS NULL` — 0 rows → `409 exercise_completed` (race with a completion).
8. `200` with the task in the `GET /quests/daily` task shape (same `id`).
`GET /quests/daily` tasks gain additive nullable `difficulty_hint` (design header "One data need").
Level step: `too_hard` → one CEFR step down, `too_easy` → one up, clamped A1..C2 (a clamped request still generates a fresh same-level task).

## Global constraints

- New Go code in new files: `quests/swap.go`, `swap_handler.go`, `swap_test.go`, `swap_integration_test.go`; `airouter/exercise.go`, `exercise_test.go`. Edit `quests/repo.go`/`service.go`/`handler.go` only to add `difficulty_hint` to the daily read.
- No change to `POST /quests/progress`, the counter, the pet hook or `MarkComplete`.
- Frontend: design §8 "Executor traps" are review blockers (no `window.confirm`; timers untouched; replace at the same index; reset answers only on 200; `fog` makes the box inert; bottom button disabled while swapping; the promise lives in the store).
- Each task: backend `cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"`; frontend `cd frontend && npm run lint && npm run typecheck && npm run test:unit`. One commit per task.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/000N_exercise_swap.{up,down}.sql` (new) | `ALTER TABLE exercises ADD COLUMN difficulty_hint TEXT NULL CHECK (difficulty_hint IN ('too_hard','too_easy')), ADD COLUMN swapped_at TIMESTAMPTZ NULL` |
| backend spec §3.2, §6.2; 1st-thinking §4, §7 | DDL block; endpoint block; `quests:swap:{user_id}:{date}` key (48 h TTL) |
| `backend/internal/store/keys.go`, `keys_test.go` | `QuestSwapKey(userID, localDate)`, TTL constant |
| `backend/internal/airouter/exercise.go` (new) + `exercise_test.go` | `ExerciseSystemPrompt`, `ExerciseUserPrompt`, `ShiftLevel`, `ParseExercise` |
| `backend/internal/quests/swap.go` (new) + `swap_test.go`, `swap_integration_test.go` | `SwapService` / `Swap`, `SwapRepo`, `SwapCounter` |
| `backend/internal/quests/swap_handler.go` (new) + handler tests | `SwapHandler` |
| `backend/internal/quests/repo.go`, `service.go` | `difficulty_hint` in the daily task read/shape |
| `backend/cmd/api/main.go` | wire the service, `guarded.POST("/quests/:id/swap", ...)` |
| `frontend/components/retro/RetroButton.vue` + test | `size: 'md' \| 'sm'` |
| `frontend/stores/quest.ts` + new `questStoreSwap.test.ts` | `swap`, `swapping`, `swapCapReached`, `swapsToday` |
| `frontend/components/learn/SwapRow.vue` (new) + test | the row |
| `frontend/pages/learn/[id].vue` + new `learnSwap.test.ts` | mount, fog, bottom disable, error mapping |
| `harness/UI-KIT.md`, `harness/CODEMAP.md` | `RetroButton size`; `quests`, `airouter`, `store`, `shell` |

## Tasks

### Task 1: migration, key, specs
- [ ] `ls backend/internal/store/migrations/` → N = next number. Write up/down (`down`: drop both columns). Extend the migrations list test if it enumerates files. Append the DDL to backend spec §3.2 (`-- Migration 000N`). `store.QuestSwapKey` + `QuestSwapTTL = 48*time.Hour` with a `keys_test.go` case; 1st-thinking §4 gains the key.
- [ ] `make test`; spec block equals the `.up.sql`. Commit `feat(store): exercises remember a difficulty swap`.

### Task 2: exercise prompt (airouter, tests first)
- [ ] `exercise_test.go`: `ShiftLevel("B1","too_hard")=="A2"`, `("A1","too_hard")=="A1"`, `("C2","too_easy")=="C2"`, `("B2","too_easy")=="C1"`, unknown level → `B1` baseline shifted; `ExerciseUserPrompt` contains the shifted level, `LevelGuidance(level)`, goal, task type, duration, day title, module focus, original title, and the hint phrased in English ("the learner found the previous task too hard"); `ExerciseSystemPrompt` demands strict JSON of one task `{"type","title","duration_minutes","content"}` with the typed-content shape for that type; `ParseExercise(raw, wantType, wantMinutes)` accepts a valid sample (built from `SampleContent`), rejects wrong type, wrong duration, invalid content (reuses `validateContent`), non-JSON.
- [ ] Implement `exercise.go`. Commit `feat(airouter): exercise_generation prompt and parser`.

### Task 3: swap service (tests first)
- [ ] `swap_test.go` with fakes and a scripted provider through the real `airouter.Router`: happy path writes content + hint and returns the daily task shape; prompt carries level−1 for `too_hard` and A1 floor/C2 ceiling; completed → `ErrExerciseCompleted` with no AI call; already swapped → `ErrAlreadySwapped`; cap: 4th swap → `ErrSwapLimit`, counter back to 3; rate limited → `ErrRateLimited`, counter decremented; malformed then valid → success with two provider calls; malformed twice → `ErrAIBadOutput`, row untouched, counter decremented; provider timeout → timeout error mapped, row untouched; UPDATE affecting 0 rows → `ErrExerciseCompleted`.
- [ ] Implement `quests/swap.go` (`SwapRepo` = `SwapTarget` + `ApplySwap`; `SwapCounter` = `Incr/Decr` over Redis; `Swap(ctx, userID, exerciseID, hint)`).
- [ ] Integration test (`TEST_DATABASE_URL` + `TEST_REDIS_URL`-gated): a swap persists `content_json` and `difficulty_hint`, `swapped_at` is set, a second swap on the same exercise is refused.
- [ ] Commit `feat(quests): swap one task through exercise_generation`.

### Task 4: handler, daily field, wiring
- [ ] Handler tests: each status in *Contract* with the exact `{"error": "<code>"}` body (`invalid_request` for a bad hint, a non-UUID id and an empty body; `exercise_not_found`; `no_active_roadmap`; `exercise_completed`; `already_swapped`; `swap_limit`; `rate_limited`; `ai_bad_output`; `ai_unavailable`; `ai_timeout`). Daily read test: a swapped task carries `difficulty_hint`, others `null`.
- [ ] Implement `swap_handler.go`; add `DifficultyHint *string \`json:"difficulty_hint"\`` to the daily task and select the column; wire in `main.go` with the existing router and rate limiter instances.
- [ ] Backend spec §6.2 endpoint block and the daily field; 1st-thinking §7 row; CODEMAP `quests`, `airouter` (the strategies entry now has a caller), `store`.
- [ ] Commit `feat(api): POST /quests/{id}/swap`.

### Task 5: kit `RetroButton size` + store action (design §6)
- [ ] `RetroButton` test: `size="sm"` renders 44-px height classes and VT323 20; default unchanged (existing tests untouched).
- [ ] `questStoreSwap.test.ts`: `swap(id,'too_hard')` sets `swapping[id]` during the request and clears it after; on 200 replaces the task at the same index, clears that task's answers and item index, never touches `timers`; on error leaves the task and answers intact and rethrows `ApiError`; `swap_limit` sets `swapCapReached`; `swapsToday` counts tasks with `difficulty_hint`.
- [ ] Implement. Commit `feat(frontend): quest store swaps a task in place`.

### Task 6: `SwapRow` and the room (design §1, §3, §4, §5, §7)
- [ ] `SwapRow` tests: idle → confirm → "Thôi"/Esc restores with focus on the opening chip; the "làm lại từ câu đầu" line only when `answered`; swapping line and the 15-s change (fake timers); swapped caption; cap and offline disabled captions (`aria-disabled`, not removed); error line `role="alert"`; group `aria-label="Độ khó bài này"`.
- [ ] `learnSwap.test.ts` (design review checks): row between content and bottom button, absent on completed/review/loading/empty/unclassifiable; during a pending swap the box has `inert`, bottom button disabled, timer still advances; 200 → content re-classified, item 1, toast; 502 → original content and answers intact, chips + error copy; 409 → reload + review toast; 404 → existing strip; leaving and re-entering mid-swap shows the waiting state; `window.confirm` spy never called.
- [ ] Implement `SwapRow.vue` and mount in `pages/learn/[id].vue`; stop read-aloud (`useSpeech().stop()` if the composable exists) and close any open gloss card (`ItemPassage.closeGloss()` if present) on "Đổi bài".
- [ ] Commit `feat(frontend): Khó quá / Dễ quá in the learning room`.

### Task 7: docs
- [ ] `harness/UI-KIT.md` `RetroButton size`; CODEMAP `shell`. Commit `docs: UI kit and CODEMAP for task swaps`.

## Verification
```
cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"
TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./internal/quests/... ./internal/store/... -count=1   # scratch COMPOSE_PROJECT_NAME=<slug>
cd ../frontend && npm run lint && npm run typecheck && npm run test:unit && npm run build
grep -rn 'TaskExerciseGen' ../backend/internal --include='*.go' | grep -v _test | grep -v airouter/types.go   # a caller exists
grep -rn 'window.confirm\|confirm(' ../frontend/components/learn ../frontend/pages/learn   # empty
```
Runtime proof (local stack, one real provider key): `curl -X POST -H "Authorization: Bearer …" -d '{"hint":"too_hard"}' :8080/api/v1/quests/<today's id>/swap` → 200 with new content and `difficulty_hint`; the same call again → 409 `already_swapped`; four swaps across tasks → the 4th is 429 `swap_limit`; record latency and token usage.

Design Acceptance (verbatim):
1. On an incomplete task, every item of the room shows a "Bài này" row with "Khó quá" and "Dễ quá" (44 px, secondary) directly below the dialogue box and above the bottom button — not in the eyebrow row and not inside the passage panel; completed tasks, review mode, the chest, and the loading, empty and unclassifiable states show no row.
2. Tapping a chip replaces the row within 100 ms with an in-page torch panel ("Đổi lấy bài dễ hơn?" / "…khó hơn?", "Thôi", "Đổi bài"); `window.confirm` is never called; "Thôi" or Esc restores the row with no request sent; the "làm lại từ câu đầu" line appears only when at least one item is answered.
3. "Đổi bài" sends `POST /api/v1/quests/{id}/swap` with the chosen `hint`; within 100 ms the dialogue box is fogged and inert, the row shows the companion line with stepping dots, and the bottom button is disabled; after 15 s the line changes to "Sắp xong rồi, cậu chờ tớ chút nhé".
4. The countdown keeps running during and after the swap and is never reset; `quest.timers[id]` is unchanged by the swap, and the `duration_seconds` posted on completion includes the waiting time.
5. On 200 the task is replaced in place in `daily.tasks` (same id, same index), the room restarts at item 1 with the new content, that task's answers are cleared, a growth toast appears, and the row becomes "Đã đổi sang bài dễ hơn." / "…khó hơn." with no chips — also after a reload (from `difficulty_hint`).
6. On `ai_bad_output`, `ai_unavailable`, `ai_timeout`, `invalid_request` or a network error, the original content and its answers stay, the fog lifts, the chips return enabled with "Chưa tạo được bài mới, cậu cứ tiếp tục bài này nhé."; on `rate_limited` the line is "Tớ cần nghỉ một phút. Cậu cứ làm bài này, lát thử lại nhé."
7. On `swap_limit`, or when three of today's tasks have a `difficulty_hint`, the chips on every unswapped task are disabled (not removed) with "Hôm nay đã đổi 3 bài"; offline they are disabled with "Cần mạng để đổi bài".
8. On `exercise_completed` the store reloads and the room shows review mode with "Bài này đã xong rồi, không cần đổi nữa."; on `exercise_not_found` the row disappears and the existing "Nhiệm vụ này không còn trong hôm nay…" strip shows.
9. Leaving with "‹ Trại" during a swap does not lose it: re-entering shows the waiting state until the response lands, and the hub tile shows the new title after it does.
10. Under `prefers-reduced-motion: reduce` the dots are static and the fog toggles without transition; every state remains distinguishable by text.
