---
type: feature
status: proposed
source: ideator
run: 2026-09-26-run-01
order: 3
---

## Why
The Google integration is the spec's third retention lever (§1, §5.1 steps 6–7) and today it is a one-shot export that goes stale the day after sync. `google.Sync` inserts 28 tasks "Day N: title · title · title" and a recurring 30-minute event, then never touches them again: a learner who meets the target on day 3 still sees "Day 3" unchecked in Google Tasks, and by day 10 the list is ten overdue items — noise they either tick by hand or delete (the inbox already has a bug about a user-deleted list). The calendar reminder fires with no way into the app. Ticking the task when `daily_progress.is_target_met` flips is still app → Google, so it keeps the spec's one-way rule (§5.1: nothing reads changes back) while making the Google surface tell the truth every day — the daily loop becomes visible in the tool the learner already opens, which is exactly why the integration exists. The calendar event carrying the app link makes the Google notification the entry point on the phone, which is the cheapest reminder path we have for iPhone users who never installed the PWA (Web Push needs a Home Screen install there).

## Expected output
User-visible:
- Within a minute of crossing 30 minutes, the day's task in the "English daily quests" list is checked (`status: completed`) with the real completion time. Days met before this ships are back-filled on the next sync.
- The "English practice" calendar event description says "Mở phòng học: https://<app>/ — 3 nhiệm vụ · 30 phút" and carries `source.url`, so the event and its notification open the hub in one tap. Existing users get it at their next sync (the event is patched anyway).
- Nothing new to configure; a learner who never synced sees no change. A revoked token or a deleted list never blocks the daily loop: ticking is best-effort and silent for the learner, logged once for the operator.

Technical (backend `google`, hook from `quests`; frontend copy only):
- Store the task ids: new table `google_sync_tasks (user_id, roadmap_id, day_number, task_id, PRIMARY KEY (user_id, roadmap_id, day_number))` written in the same insert loop that already persists sync state (migration `0004`/`0005`; backend spec §3.2 DDL block appended, per the AGENTS.md rule). `TasksClient` gains `PatchTask(ctx, token, listID, taskID, TaskPatch{Status, Completed})` (`PATCH /lists/{l}/tasks/{t}`); `Event` gains `Description` and `Source{Title, URL}` from `FRONTEND_ORIGIN`'s first entry (or a new `PUBLIC_APP_URL`).
- Delivery: `quests` already fires `Pet.OnTargetMet` exactly on the crossing call; add a second hook `google.QuestHook` that `ZADD`s `queue:google:complete` (score now, member `user_id:date`, key builder in `store/keys.go`) — one line, no Google call inside the request. A poller with the `notify.RunWorker` shape (30 s, one per deployment) pops due members, refreshes a token through the existing `RefreshTokenSource`, looks up the task id for that user/roadmap/day, PATCHes, and drops the member on 2xx/404/410 or `reauth_required`; other failures re-slot once (+5 min) then drop with a log line. Users with no `google_sync` row are skipped without a Google call.
- `Sync` back-fills: after (re)building the list it PATCHes every day whose `daily_progress.is_target_met` is true — the roadmap-tree plan's `daily_progress` range read is the query to share.
- Tests: `httptest` for `PatchTask` and the event description; service tests with the fakes' call log for hook → queue → poller; the 404/410/reauth prune paths; an integration test that the task-id table is one row per day. Backend spec §6.4 unchanged on the wire (`POST /integrations/google/sync` body identical); CODEMAP `google`, `quests`, `store` updated.
- Estimate: one working day. No frontend code beyond the settings copy "Đã đồng bộ · nhiệm vụ tự đánh dấu khi cậu học xong" (designer owns the wording).

## Evidence
- Spec: 1st-thinking §1 (Google Calendar/Tasks as a retention driver), §5.1 steps 6–7 (one-way push), §4 (`queue:webpush:delay` as the persistent-ZSET precedent); backend spec §6.4.
- Code: `backend/internal/google/tasks.go:30-35` (`TasksClient` = InsertTaskList / DeleteTaskList / InsertTask — no patch, no ids kept); `service.go:129-150` (insert loop, `DayDue`); `backend/internal/store/migrations/0002_google_sync.up.sql:8-13` (only `tasklist_id` + a count are stored); CODEMAP `quests` (`Pet.OnTargetMet` fires exactly on the crossing call — the hook point) and `notify` (`RunWorker` shape, re-slot-first rule).
- Inbox context (not re-filed): `harness/ideas/_inbox/a-user-deleted-tasks-list-is-never-rebuilt-and-sync-keeps-an.md` (a stale list is what users delete); `harness/ideas/_inbox/due-plus-re-slot-is-not-atomic-so-two-api-instances-double-s.md` (same ZSET pattern; whichever fix lands there applies here — a double PATCH to `completed` is idempotent anyway).
- Research: Google Tasks `tasks.patch` sets `status: "completed"` with patch semantics — https://developers.google.com/tasks/reference/rest/v1/tasks/patch and https://developers.google.com/tasks/reference/rest/v1/tasks ; Web Push on iOS reaches only Home Screen web apps, so the calendar notification is the fallback entry point — https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/
