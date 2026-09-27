---
idea: harness/ideas/2026-09-26-run-01/missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md
status: approved
priority: medium
merged: false
order: 4
design: harness/designs/missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md
---
# Missed days are not lost — "Học bù" on any earlier day, minutes count toward today — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Design:** `harness/designs/missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md` — every frontend task cites its sections; `## Verification` repeats its Acceptance list verbatim.
**Idea:** `harness/ideas/2026-09-26-run-01/missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md`
**Goal:** A learner can open any earlier roadmap day that was not target-met, do its tasks, and have the minutes count toward today's 30 minutes and today's plant — without ever rewriting that past day's verdict.

**Scope:** backend `quests` (additive query param, relaxed ownership check, one additive roadmap field) + frontend (quest store, hub catch-up mode, roadmap panel action, learning-room `?day`). **Estimate:** one working day (backend ~3 h, frontend ~5 h). **Branch:** `harness/2026-09-27-medium-missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks`.

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically
`harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-` (`GET /api/v1/roadmap`, `quests/roadmap.go`, `stores/roadmap.ts`, `utils/roadmap.ts`, `pages/roadmap.vue`),
`harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (retro kit, `pages/index.vue`, `pages/learn/[id].vue`),
`harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` (`stores/quest.ts`, learning room),
`harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target` (`stores/quest.ts`, hub growth moment),
`harness/2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-` (`quests/service.go`),
`harness/2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile` (`quests/repo.go`, `service.go`).
```
for b in 2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with- 2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n 2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab 2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target 2026-09-25-medium-a-pet-state-failure-reports-pet-health-0-which-means-a-dead- 2026-09-24-medium-get-quests-daily-still-reads-is-target-met-from-the-volatile; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
test -f backend/internal/quests/roadmap.go && test -f frontend/stores/roadmap.ts && echo files-ok
```
Any `MISSING` (and the file check failing) → stop and report `blocked`. The design names both v1 and retro components; build on whichever is on `main` (design header "Which screens it lands on").

## Contract (backend spec §6.2 — additive only)

- `GET /api/v1/quests/daily?day=N`: `N` absent → today (unchanged). `N` a base-10 integer with `1 ≤ N ≤ today's day_number` → that day's tasks with `day_number: N`. Anything else (0, negative, > today, non-integer, empty string) → `400 invalid_request`. The body gains `is_today: bool` on every response (true without `day`). `date`, `accumulated_seconds`, `is_target_met` are **always today's**.
- `POST /api/v1/quests/progress`: an exercise on the caller's active roadmap with `day_number ≤ today` is accepted (was `= today`); future day or other roadmap → `404 exercise_not_found` (unchanged code). Everything after the check is untouched: INCRBY on **today's** date, `daily_progress` upsert for today, `Pet.OnTargetMet`, `MarkComplete`.
- `GET /api/v1/roadmap`: each day gains `completed_tasks` (0–3, count of that day's exercises with `is_completed`) — the design's contract addition (design header, §4.1).

## Global constraints

- No migration. No new endpoint. No change to `is_target_met`/`minutes_spent` of a past date anywhere.
- New Go logic goes into new files where the existing file is busy: `quests/catchup.go` + `catchup_test.go`; edit `repo.go`/`service.go`/`handler.go` only where a signature must change.
- Design §8 "Executor traps" (1)–(7) are review blockers; read them before Task 4.
- Each task: backend `cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"`; frontend `cd frontend && npm run lint && npm run typecheck && npm run test:unit`. One commit per task.

## Tasks

### Task 1: relaxed ownership check (backend, tests first)
- [ ] `catchup_test.go`: fake repo table — exercise on day 3 with today = 5 → accepted; day 5 → accepted; day 6 → `ErrExerciseNotFound`; other roadmap → `ErrExerciseNotFound`. Service test: a day-3 exercise's progress INCRBYs today's key and upserts today's date (fake counter/repo call log), `MarkComplete` is called with the day-3 id, and the call order pinned by the existing service tests is unchanged.
- [ ] Rename the semantics: `CheckExercise(ctx, roadmapID, exerciseID, maxDay int)` — SQL `day_number <= $3` instead of `= $3`; update the interface comment, the fake, and the one call site. Integration test (`TEST_DATABASE_URL`-gated, in `catchup_test.go` or a new `catchup_integration_test.go`): a past-day exercise passes, a future-day one does not.
- [ ] Commit `feat(quests): progress accepts an earlier day's exercise`.

### Task 2: `?day=N` on the daily read
- [ ] Tests: `ParseDayParam("") → (0, nil)`; `"3"` with today 5 → 3; `"5"` → 5; `"6"`, `"0"`, `"-1"`, `"3.5"`, `"abc"`, `" 3"` → `ErrInvalidDay`. Service: `DailyFor(ctx, user, day)` with day 3 returns day-3 tasks, `day_number 3`, `is_today false`, and today's `date`/`accumulated_seconds`/`is_target_met`; day 0 → today with `is_today true`. Handler: `?day=6` → 400 `{"error":"invalid_request"}` in the existing error envelope; no param → body identical to before plus `is_today: true`.
- [ ] Implement `ParseDayParam` in `catchup.go`, `Service.DailyFor` (existing `Daily` becomes `DailyFor(ctx, u, 0)`), `IsToday` field on `DailySuite`, handler reads `c.Query("day")` only when present (`c.GetQuery`).
- [ ] Commit `feat(quests): GET /quests/daily?day=N serves an earlier day`.

### Task 3: `completed_tasks` on the roadmap outline
- [ ] Test (roadmap service/repo fakes): a day with 2 of 3 exercises completed → `completed_tasks: 2`; days with no exercises → 0. Integration read-back.
- [ ] Repo: one grouped query `SELECT day_number, count(*) FILTER (WHERE is_completed) FROM exercises WHERE roadmap_id=$1 GROUP BY day_number`, merged into `RoadmapDay.CompletedTasks int \`json:"completed_tasks"\``.
- [ ] Backend spec §6.2: document `?day`, `is_today`, the relaxed progress rule and `completed_tasks` (the roadmap block if §6 has one, else the §7 endpoint list note); 1st-thinking §7 list note for `?day`. CODEMAP `quests`.
- [ ] Commit `feat(quests): roadmap days report completed_tasks; spec §6.2`.

### Task 4: quest store two-day state (design §5 "Store")
- [ ] `tests/unit/questStore.test.ts` (or a new `questStoreCatchUp.test.ts` if the file is busy): `loadDay(3)` sets `viewingDay 3`, `dayQuests`, never replaces `daily`, copies today's `accumulated_seconds`/`is_target_met` into `daily`; `clearDay()` resets; `taskById` finds a task in `dayQuests`; `complete()` of a day-3 task ticks it in `dayQuests.tasks` and updates `daily`'s counters; 400 sets `dayError='invalid'`, network sets `dayError='network'`.
- [ ] Implement in `stores/quest.ts`: `viewingDay`, `dayQuests`, `dayLoading`, `dayError`, `loadDay`, `clearDay`, `viewSortedTasks`, `viewNextTaskId`; types gain `is_today?: boolean` and roadmap day `completed_tasks?: number`.
- [ ] Commit `feat(frontend): quest store holds a catch-up day beside today`.

### Task 5: roadmap panel action (design §3.1, §4.1, §6 `DayDetail`/`RoadmapNode`)
- [ ] `tests/unit/roadmap.test.ts` (or new `catchUpAction.test.ts`): `catchUpAction(day, todayNumber)` over every row of §4.1 including the fallback when `completed_tasks` is undefined; second caption strings.
- [ ] Implement `catchUpAction` in `utils/roadmap.ts`; add the action slot + second caption to the expanded past-day panel (`DayDetail` retro or `RoadmapNode` v1); `pages/roadmap.vue` refetches on every mount (design §5 last line). Component test: past missed day shows "Học bù →" linking `/?day=N`; met day and future day show none.
- [ ] Commit `feat(frontend): Học bù / Học tiếp on past roadmap days`.

### Task 6: hub catch-up mode (design §1, §3.2, §3.3, §4.2, §7)
- [ ] `tests/unit/indexPage.test.ts` (or new `indexCatchUp.test.ts`): `/?day=3` calls `loadDay(3)`; torch banner "Ngày 3" (+ date only from the roadmap store); path from `dayQuests`; `DayBar` still bound to `daily`; "‹ Về hôm nay" present in loading/error/done; no bottom button while loading; done state copy + "Về hôm nay"; `?day=<today>` → `navigateTo('/', {replace:true})`; 400 → "Ngày này chưa mở."; network → retry refetches only the day; normal hub shows "Học bù ngày N · k/3 nhiệm vụ còn lại" only when today is met and such a day exists.
- [ ] Implement in `pages/index.vue` (plus a small `components/hub/CatchUpBanner.vue` if the page grows past readability). Copy exactly design §7. Offline behaviour per §4.2 (SW NetworkFirst already keys on full URL — verify `?day` is not stripped in `service-worker/sw.ts`; if it is, include the query in the cache key).
- [ ] Commit `feat(frontend): hub catch-up view for an earlier day`.

### Task 7: learning room `?day=N` (design §3.4, §5 "Complete in room")
- [ ] `tests/unit/learnPage.test.ts` (or new `learnCatchUp.test.ts`): with `?day=3` the eyebrow "Học bù · Ngày 3" and back link "‹ Ngày 3" → `/?day=3`; task not in either list → `loadDay(3)` first, still missing → design §7 not-found copy; completion navigates `replace` to `/?day=3`; without `?day` behaviour is unchanged. Timer still keyed by task id.
- [ ] Implement in `pages/learn/[id].vue`; hub tiles/bottom button in catch-up mode link `/learn/:id?day=N`.
- [ ] Commit `feat(frontend): learning room returns to the catch-up day`.

### Task 8: docs
- [ ] CODEMAP `quests` (relaxed check, `?day`, `completed_tasks`) and `shell`/frontend paragraph (catch-up mode). Commit `docs: CODEMAP for catch-up days`.

## Verification
```
cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"
TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./internal/quests/... -count=1   # scratch compose project with COMPOSE_PROJECT_NAME=<slug>
cd ../frontend && npm run lint && npm run typecheck && npm run test:unit && npm run build
grep -n 'is_today\|completed_tasks\|day=N' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
```
Runtime proof (local stack): create a roadmap dated 3 days ago, `curl -H "Authorization: Bearer …" ':8080/api/v1/quests/daily?day=2'` → `day_number 2`, `is_today false`, today's `date`; `?day=5` → 400; post progress for a day-2 exercise → today's `accumulated_seconds` rises and `daily_progress` for day 2's date is unchanged.

Design Acceptance (verbatim):
1. On `/roadmap`, an expanded past day with `is_target_met=false` shows "Học bù →" (untouched) or "Học tiếp →" (started) per §4.1; target-met past days, days with 3/3 tasks done, today and future days show no catch-up button.
2. Tapping the button opens `/?day=N`, which requests `GET /api/v1/quests/daily?day=N` and shows a torch "Học bù" banner reading "Ngày N" (with the date from the roadmap outline when loaded, never the body's `date`).
3. In the catch-up view, the path lists day N's three tasks with their real `is_completed` state, while the day bar still shows today's `accumulated_seconds` and today's met state.
4. A "‹ Về hôm nay" link is always visible in the catch-up view (loading, error and done included) and returns to `/` with the normal today path.
5. Opening a catch-up task goes to `/learn/:id?day=N` with the eyebrow "Học bù · Ngày N"; completing it posts progress, returns to `/?day=N`, ticks that tile, moves today's day bar and plays the growth moment once.
6. When all three tasks of day N are done, the banner reads "Đã học bù xong ngày N." and the bottom button is "Về hôm nay".
7. After any catch-up, day N's glyph and first status line on `/roadmap` are unchanged (no ★ added, no streak change); only the second caption "Đã học bù c/3 nhiệm vụ" updates, fetched fresh on the next visit.
8. `/?day=` with a future day, 0 or a non-integer shows "Ngày này chưa mở." and a "Về hôm nay" button; `/?day=<today>` redirects to `/`; network errors show a retry that refetches only that day.
9. The bottom button is absent while the catch-up day loads; offline, a previously loaded day renders from cache with one toast and an unvisited day shows the offline message.
10. On the normal hub, the row "Học bù ngày N · k/3 nhiệm vụ còn lại" appears only after today's target is met and only when such a past day exists; under reduced motion no animation runs in any of these views.
