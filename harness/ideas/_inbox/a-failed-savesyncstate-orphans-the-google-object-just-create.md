---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# A failed SaveSyncState orphans the Google object just created, and CODEMAP claims it cannot

## Why
`Service.Sync` creates a Google object and *then* records its id in Postgres, with no idempotency
key on the remote call. Every failure in that window leaks an object we can never find again:

- `service.go:79-84` inserts the Calendar event, `service.go:85-87` saves the id. If
  `SaveSyncState` fails — a pool hiccup, an FK violation, or, most plausibly, **the client
  disconnecting** (Gin cancels `c.Request.Context()`, which `handler.go:28` wraps, so the `Exec`
  returns `context.Canceled`) — the user now owns a 28-day recurring "English practice" event whose
  id we never stored. The next sync sees `state.CalendarEventID == ""` and inserts a **second**
  one. Repeat per failure. The user must delete each stray event by hand; nothing in the app can.
- `service.go:107-114` does the same for the Tasks list: a failed save leaves a stray empty
  "English daily quests" list in the user's Tasks, and the retry deletes only the *old* stored id.

The plan and CODEMAP both assert this cannot happen. Plan *Architecture*: "State is persisted after
the event step and after the list is created, before the 28 task inserts, so a crash mid-sync never
orphans a Google object the next sync cannot find." CODEMAP `google`: "state is saved after the
event and after the list is created, before the task inserts, so a failure never orphans a Google
object." That ordering only protects against a failure in the *later* Google calls; it does nothing
about a failure of the save itself, which is the step that makes the id findable. The documentation
is wrong, and being wrong here is worse than the gap, because the next maintainer will not look.

## Expected output
`InsertEvent` sends a caller-supplied event id so the insert is idempotent. Calendar v3
`events.insert` accepts `id` (base32hex, 5–1024 chars); deriving it deterministically from the user
id (e.g. a base32hex encoding of the user UUID) makes a repeated insert return `409 duplicate`
instead of creating a second event, and the service can treat that 409 as "already ours" and patch
it. The Tasks API has no client-supplied id, so the list keeps the current ordering — but the plan's
and CODEMAP's claim is corrected to the truth: *a failure of `SaveSyncState` itself can orphan the
object just created*, with the Calendar half now covered by the deterministic id.

A test injects a `SaveSyncState` error immediately after `InsertEvent` and asserts the following
sync does not produce a second event.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), *Architecture* paragraph.
- `backend/internal/google/service.go:79-87` — insert, then save; no idempotency key on the insert.
- `backend/internal/google/service.go:107-114` — same shape for the Tasks list.
- `backend/internal/google/calendar.go:33-44` / `schedule.go:77-85` — `Event.payload()` sends no `id` field.
- `backend/internal/google/handler.go:28` — `context.WithTimeout(c.Request.Context(), SyncTimeout)`, so a client disconnect cancels the in-flight `SaveSyncState`.
- `harness/CODEMAP.md` → `google` — the false "a failure never orphans a Google object" sentence.
- `backend/internal/google/fakes_test.go:130-135` — `fakeRepo.SaveSyncState` always returns nil, so no test can reach this window today (see the sibling bug on missing fake error fields).
