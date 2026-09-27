---
idea: harness/ideas/2026-09-26-run-01/google-tasks-tick-themselves-when-the-day-s-target-is-met-an.md
status: approved
priority: medium
merged: false
order: 3
---
# Google Tasks tick themselves when the day's target is met, and the calendar event opens the app — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-26-run-01/google-tasks-tick-themselves-when-the-day-s-target-is-met-an.md`
**Goal:** Within a minute of a learner crossing 30 minutes, that day's task in the "English daily quests" Google Tasks list is `completed`; a sync back-fills days already met; and the "English practice" calendar event carries a one-tap link into the app — all still one-way (app → Google), best-effort, and never able to fail a `POST /quests/progress`.

**Scope:** backend only (`google`, `store`, `cmd/api`). **No frontend change** — the idea's optional settings-screen copy is dropped (YAGNI: the settings screen itself is unmerged, and the ticked task is its own feedback). No wire change: `POST /api/v1/integrations/google/sync` request and response (backend spec §6.4) are identical.

**Estimate:** one working day. **Branch:** `harness/2026-09-27-medium-google-tasks-tick-themselves-when-the-day-s-target-is-met-an`.

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR and today's bugfix branches that touch the same files —
`harness/2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn` (`google/service.go`),
`harness/2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re` (`google/client.go`, `google/schedule.go`),
`harness/2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever` (`google/token.go`),
`harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint` (`store/keys.go`),
`harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase` and `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` (the two pending `0004_*` migrations — this plan's migration takes the next free number after both land).
Check before starting:
```
for b in 2026-09-27-low-the-404-on-patch-fallback-re-opens-the-orphan-window-and-syn 2026-09-27-low-google-403-accessnotconfigured-api-disabled-still-maps-to-re 2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever 2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint 2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase 2026-09-26-medium-pet-streak-shield-earned-by-target-days; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
ls backend/internal/store/migrations/
```
Any `MISSING` → stop and report `blocked` (fallback when a branch was deleted after merging: the file-level change is present on `origin/main`, e.g. two distinct `0004_*`/`0005_*` migrations exist). Also read `harness/ideas/_inbox/due-plus-re-slot-is-not-atomic-so-two-api-instances-double-s.md`: if its fix has landed in `notify`, mirror the same atomic pop here (Task 5); if not, keep the notify pattern — a double PATCH to `completed` is idempotent.

## Design decisions (fixed — do not re-open)

1. **Task ids are stored.** New table `google_sync_tasks (user_id UUID REFERENCES users(id) ON DELETE CASCADE, roadmap_id UUID REFERENCES roadmaps(id) ON DELETE CASCADE, day_number SMALLINT CHECK (day_number BETWEEN 1 AND 28), task_id TEXT NOT NULL, PRIMARY KEY (user_id, roadmap_id, day_number))`. `InsertTask` returns Google's id; `Sync` saves it in the same loop. A list rebuild (new roadmap) deletes the old roadmap's rows.
2. **Delivery is a queue, not a request-path call.** A `cmd/api` adapter wraps the pet hook: `OnTargetMet` calls the pet first, then `ZADD queue:google:complete <now> <user_id>:<local_date>` (enqueue error logged, never returned; the pet's error is returned unchanged so `quests` keeps its "flag the day only when the hook succeeded" rule). `quests` is not edited.
3. **One poller per deployment**, `google.RunCompleter(ctx, svc, 30*time.Second)`, same shape as `notify.RunWorker`. Per due member: skip (drop) when the user has no `google_sync` row or no stored task for that day; otherwise refresh the access token through the existing `RefreshTokenSource` + `TokenRefresher`, compute the roadmap day from the date (`DayNumberForDate`, the inverse of `DayDue`), `PatchTask(status=completed, completed=<now RFC3339>)`. Drop on 2xx, `ErrNotFound`, `ErrReauthRequired`, `ErrNoRefreshToken`. Any other error: re-slot once at `now+5min` (member gets a `#r` suffix so the second failure drops it with one log line).
4. **Back-fill:** at the end of `Sync` (both the "list already current" and the "list rebuilt" paths) PATCH every stored task whose day is target-met in `daily_progress` for dates within the roadmap's 28 days. Back-fill errors other than reauth are logged and do not fail the sync (the §6.4 response is still `synced`); `ErrReauthRequired` propagates as today.
5. **Calendar link:** `Event` gains `Source{Title, URL}`; payload adds `"source": {"title": "Học 30 phút", "url": <APP_URL>}` and the description becomes `"Mở phòng học: <APP_URL>/ — 3 nhiệm vụ · 30 phút"`. `<APP_URL>` = first entry of `FRONTEND_ORIGIN` (config already parses it; expose `cfg.AppURL()` or pass `origins[0]` from `main.go`). Existing users get it at their next sync (the event is patched on every sync already).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/000N_google_sync_tasks.{up,down}.sql` (new; N = next free) | the table in decision 1 |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` §3.2 | append the same DDL after the last migration block (AGENTS.md rule) |
| `backend/internal/store/keys.go`, `keys_test.go` | `GoogleCompleteQueueKey = "queue:google:complete"` (persistent ZSET, no TTL — the §4 `queue:webpush:delay` precedent) |
| `backend/internal/google/tasks.go`, `tasks_test.go` | `InsertTask` returns `(string, error)`; `PatchTask(ctx, token, listID, taskID, TaskPatch)` |
| `backend/internal/google/schedule.go` + new `schedule_source_test.go` | `Event.Source`, payload `source`, description copy, `DayNumberForDate` |
| `backend/internal/google/repo.go` | `SaveTaskID`, `DeleteTaskIDs(user, roadmap)`, `TaskID(user, roadmap, day)`, `MetDays(user, roadmap)` |
| `backend/internal/google/complete.go` (new) | `Completer` service: `Enqueue`, `Tick`, `RunCompleter`, `BackfillMet` |
| `backend/internal/google/complete_test.go` (new), `fakes_test.go` | fakes gain the new methods + call log |
| `backend/internal/google/service.go` | save ids in the insert loop, delete rows on rebuild, back-fill at the end |
| `backend/internal/google/integration_test.go` | one row per day; `MetDays` read-back (gated on `TEST_DATABASE_URL`) |
| `backend/cmd/api/hooks.go` (new), `hooks_test.go` (new), `main.go` | the pet+google adapter; start `RunCompleter` |
| `harness/CODEMAP.md` | `google`, `store`, `quests` (hook note) paragraphs |

## Tasks

### Task 1: migration + DDL in the spec
- [ ] `ls backend/internal/store/migrations/` → pick N = highest number + 1. Write `000N_google_sync_tasks.up.sql` (decision 1, with a header comment in the style of `0002_google_sync.up.sql`) and `.down.sql` (`DROP TABLE IF EXISTS google_sync_tasks;`).
- [ ] Extend the store migrations test that asserts the file list/order (`migrations_test.go`) if it enumerates files.
- [ ] Append the DDL to backend spec §3.2 (fenced ```sql block, `-- Migration 000N` comment).
- [ ] `cd backend && make test` green; diff the spec block against the `.up.sql` (`diff <(sed -n '/google_sync_tasks/,/);/p' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md") <(sed -n '/CREATE TABLE/,/);/p' backend/internal/store/migrations/000N_google_sync_tasks.up.sql)` → empty).
- [ ] Commit `feat(store): google_sync_tasks keeps one Google task id per roadmap day`.

### Task 2: `PatchTask`, `InsertTask` returns the id (tests first)
- [ ] In `tasks_test.go` (httptest): `InsertTask` decodes `{"id":"t1"}` and returns `"t1"`; empty id → error. `PatchTask` sends `PATCH /lists/L/tasks/T` with body exactly `{"status":"completed","completed":"2026-09-27T12:00:00Z"}`, bearer header; 404/410 → `ErrNotFound`; 401 → `ErrReauthRequired`.
- [ ] Implement: `type TaskPatch struct{ Status string; Completed time.Time }`; interface gains `PatchTask`; `InsertTask(...) (string, error)`. Update the fake in `fakes_test.go` and the one call site in `service.go` (keep behaviour; ids saved in Task 4).
- [ ] `go test ./internal/google/...` green. Commit `feat(google): PatchTask and task ids from InsertTask`.

### Task 3: calendar source link + day-from-date helper
- [ ] New `schedule_source_test.go`: `PracticeEvent(..., appURL)` sets `Description == "Mở phòng học: https://app.example/ — 3 nhiệm vụ · 30 phút"` and `payload()["source"] == map{"title":"Học 30 phút","url":"https://app.example/"}`; empty appURL → no `source` key and the old description. `DayNumberForDate(createdAt, "2026-09-29", loc)` is the inverse of `DayDue` for n = 1, 3, 28; a date before day 1 or after day 28 → `ok=false`.
- [ ] Implement in `schedule.go`; thread `appURL` through `NewService` (new trailing field `AppURL string` set via an option or constructor arg — update `main.go` and test constructors).
- [ ] Commit `feat(google): practice event links into the app`.

### Task 4: repo methods + Sync saves ids and back-fills
- [ ] Repo interface gains `SaveTaskID(ctx, userID, roadmapID string, day int, taskID string) error` (upsert), `DeleteTaskIDs(ctx, userID, roadmapID string) error`, `TaskID(ctx, userID, roadmapID string, day int) (string, error)` (`ErrNoTask` when absent), `MetDays(ctx, userID, roadmapID string) ([]int, error)` — `SELECT date FROM daily_progress WHERE user_id=$1 AND is_target_met` mapped to day numbers in Go with `DayNumberForDate` and the profile timezone (keep SQL dumb; drop out-of-range dates).
- [ ] Service tests (`service_test.go` gets a new test function only; put larger cases in `complete_test.go` if `service_test.go` conflicts): a fresh sync saves 28 ids; a rebuild deletes the old roadmap's ids before inserting; a sync with days 2 and 5 met PATCHes exactly those two; a back-fill 404 is logged and the result is still `synced`; a back-fill `ErrReauthRequired` returns the error.
- [ ] Implement `PgRepo` methods and wire them into `Sync` (decision 4).
- [ ] Integration test (`TEST_DATABASE_URL`-gated): after a fake-client sync, `SELECT count(*) FROM google_sync_tasks WHERE user_id=$1` = 28 and the PK rejects a duplicate day.
- [ ] Commit `feat(google): sync stores task ids and back-fills met days`.

### Task 5: queue key + Completer (enqueue, tick, run)
- [ ] `store/keys.go`: `GoogleCompleteQueueKey` + a `keys_test.go` assertion.
- [ ] `complete.go`: `type CompleteQueue interface { Add(ctx, member string, at time.Time) error; Due(ctx, now time.Time, limit int) ([]string, error); Remove(ctx, member string) error }` with a Redis implementation (`ZADD`, `ZRANGEBYSCORE -inf now LIMIT 0 100`, `ZREM`) — mirror `notify/queue.go`. `Completer{queue, tokens, oauth, tasks, repo, now}` with `Enqueue(ctx, userID, localDate)`, `Tick(ctx, now) (Stats, error)` and `RunCompleter(ctx, c, every)`.
- [ ] Tests in `complete_test.go` (fakes, call log): enqueue → tick PATCHes the right task with `Completed == now`; no sync row → dropped with no Google call; no stored task → dropped; 404 / reauth / no refresh token → dropped; 500 → re-slotted at +5 min as `member#r`, second 500 → dropped and one log line; a member with a malformed body is dropped.
- [ ] Commit `feat(google): queue-driven completer ticks the day's task`.

### Task 6: wire the hook and the worker
- [ ] `cmd/api/hooks.go`: `type petWithGoogle struct{ quests.Pet; enq interface{ Enqueue(ctx context.Context, userID, date string) error } }` whose `OnTargetMet` calls the pet, then enqueues (log on error), returns the pet's error. `hooks_test.go`: pet ok → enqueued, nil; pet error → still enqueued, pet's error returned; enqueue error → nil returned, logged; `State` delegates.
- [ ] `main.go`: build the `Completer` next to `googleSvc`, pass `petWithGoogle{pet.NewQuestHook(petSvc), completer}` to `quests.NewService`, `go google.RunCompleter(ctx, completer, 30*time.Second)`. Keep every other line as it is on `origin/main`.
- [ ] `cd backend && make test && go vet ./... && gofmt -l .` (empty). Commit `feat(api): target-met enqueues the Google task tick`.

### Task 7: docs
- [ ] CODEMAP `google` (task ids table, Completer, back-fill, source link), `store` (migration N, queue key), `quests` (hook is wrapped in `cmd/api`, quests unchanged). 1st-thinking §4 Redis list gets `queue:google:complete` beside `queue:webpush:delay`.
- [ ] Commit `docs: CODEMAP and spec for the Google task tick`.

## Verification
```
cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"
TEST_DATABASE_URL=… TEST_REDIS_URL=… go test ./internal/google/... ./internal/store/... -run 'Integration' -count=1   # scratch compose project, COMPOSE_PROJECT_NAME=<slug>
grep -n 'google_sync_tasks' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" backend/internal/store/migrations/*.up.sql
grep -n 'queue:google:complete' backend/internal/store/keys.go project-base/1st-thinking-architecture-doc.md
grep -rn 'google' backend/internal/quests/*.go | grep -v _test   # empty: quests does not know about google
```
Acceptance:
1. A fresh sync stores 28 task ids (integration test).
2. Crossing 30 minutes enqueues exactly one member; one tick later the day's task receives `PATCH … {"status":"completed","completed":…}` (unit test through fakes).
3. A user who never synced causes zero Google calls.
4. 404/410, 401/403 and a missing refresh token drop the member without retry; other errors retry once after 5 min, then drop.
5. A re-sync PATCHes every met day's task; a back-fill 404 does not fail the sync.
6. The calendar event payload carries `source.url` = first `FRONTEND_ORIGIN` entry and the Vietnamese description.
7. `POST /quests/progress` behaviour and tests are unchanged; `quests` imports nothing new.
8. Migration and spec §3.2 DDL are identical.
