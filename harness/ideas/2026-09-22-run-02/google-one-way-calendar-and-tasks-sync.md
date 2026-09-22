---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 6
---
# Google: one-way Calendar and Tasks sync

## Why
The spec's core objective is a 30-minute daily habit, and habits form where the user's day is already planned: the Calendar block and the Tasks checklist (§5.1 steps 6–7) put the practice slot and the day's three quests in Google's own apps, with Google's own reminders, at zero engineering cost to us. It is the only feature in the spec that reaches the user when the PWA is closed and no push subscription exists, and it is explicitly one-way, so it is cheap: no webhooks, no reconciliation.

## Expected output
Delivers (Go package `backend/internal/google`, route behind `auth.Require()`):
- `POST /api/v1/integrations/google/sync` — refreshes an access token from `users.google_refresh_token`; creates one recurring 30-minute Calendar event ("English practice") starting at `users.notification_time` in `users.timezone`, `RRULE:FREQ=DAILY;COUNT=28`; creates a Google Tasks list ("English daily quests") and one task per day of the active roadmap (title from `exercises.content_json`, due date = roadmap day) — or, when no active roadmap exists, only the Calendar event, and the response says so. One-way only: nothing is read back or subscribed to. Response `{calendar_event_id, tasklist_id, tasks_created}`.
- Idempotent re-sync: a second call updates the existing event/tasklist instead of duplicating. §3.2 has no column for the Google ids; the executor stores them in `roadmaps.roadmap_json` under a `google` key or adds a migration `0002` — decision recorded in CODEMAP (see `_run.md` Notes).
- Google 401/`invalid_grant` → 409 `{error:"reauth_required"}` so the client can send the user through `/login` again.
- Tests: `httptest` fakes for the token endpoint, Calendar `events.insert`/`patch`, Tasks `tasklists.insert`/`tasks.insert`; timezone + `notification_time` produce the right RFC3339 start; re-sync path patches rather than inserts; `invalid_grant` maps to `reauth_required`.
- Tables: `users` (read `google_refresh_token`, `notification_time`, `timezone`), `roadmaps`, `exercises` (read; optional id storage). Redis keys: none. Screens: none (a "Sync to Google" button belongs to the settings placeholder in frontend-shell).

Depends on: store (1), auth (2) — must have requested the `calendar.events` and `tasks` scopes with `access_type=offline`; quests (3) for the roadmap/exercises data shape.

## Evidence
- Spec §5.1 (lines 270–302) steps 6–7: "Push 30-min Recurring Event" and "Push Daily Checklist Tasks" to the Google API Gateway; heading "Initial One-Way Sync".
- Spec §7 (line 684) `POST /api/v1/integrations/google/sync`: "Pushes 30-minute recurring study block to Google Calendar and task list to Google Tasks".
- Spec §3.2 `users` (lines 152–174): `google_refresh_token`, `notification_time TIME DEFAULT '20:00:00'`, `timezone VARCHAR(50) DEFAULT 'UTC'`.
- Spec §2.2 architecture diagram: Google APIs (Calendar & Tasks) as a direct dependency of the Go API server.
- `harness/CODEMAP.md` → `google`: "one-way Calendar + Tasks sync".
- Prior run `harness/ideas/2026-09-22-run-01/_run.md` Notes: two-way sync was considered and dropped as contradicting §5.1.
