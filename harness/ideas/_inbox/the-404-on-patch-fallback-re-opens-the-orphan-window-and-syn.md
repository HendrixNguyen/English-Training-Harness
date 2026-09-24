---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# The 404-on-patch fallback re-opens the orphan window and Sync's new doc says the Calendar half is covered

## Why
The plan that introduced this branch exists because a `SaveSyncState` failure right after `InsertEvent`
orphaned a 28-day recurring event on the user's own calendar, and because the documentation asserted
that could not happen. The deterministic-id path closes that for the normal case. The 404-on-patch
fallback does not, and the new doc comment says it does.

`backend/internal/google/service.go:94-100` inserts **without** a client id after `PatchEvent` returns
`ErrNotFound`, and `service.go:105-108` only then sets `state.CalendarEventID` and calls
`SaveSyncState`. If that save fails — the exact window this plan was filed to close — the
Google-assigned event is orphaned with an id nothing stored. It then repeats without bound: the next
sync inserts the deterministic id, Google 409s (the id is still reserved), the patch 404s again, and
the fallback inserts *another* Google-assigned event. Each failed save in this branch leaves one more
stray "English practice" event the app can never find or delete — the original bug, one level deeper.

The branch is narrow (it needs a manual delete plus Google releasing the id), and the comment at the
fallback site is honest about it ("today's non-idempotent path"). But `Sync`'s function doc
(`service.go:39-42`) ends "The Calendar half is covered by the deterministic id", which is false in
this branch, and `harness/CODEMAP.md`'s `google` bullet describes the fallback factually without
naming the residual gap. Replacing one over-broad invariant with a narrower over-broad invariant is
the failure mode the parent idea called "the more dangerous half".

## Expected output
`Sync`'s doc comment and the CODEMAP `google` bullet state that the 404-on-patch fallback is *not*
covered by the deterministic id: a `SaveSyncState` failure after it orphans the Google-assigned event,
and repeated failures orphan repeatedly. Either the docs say so, or the fallback is made idempotent
(e.g. a second deterministic id derived from the user id plus a generation counter persisted
write-ahead, so the retry 409s on that id too). A test pins whichever is chosen: inject
`failSaveAt` on the sync that goes through the fallback and assert the documented outcome.

## Evidence
- Plan under review: `harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md`
- `backend/internal/google/service.go:94-100` — fallback insert with `ev.ID = ""`.
- `backend/internal/google/service.go:105-108` — `state.CalendarEventID = id` then `SaveSyncState`; the save can fail.
- `backend/internal/google/service.go:40-43` — "The Calendar half is covered by the deterministic id."
- `harness/CODEMAP.md` `google` bullet — "a 404 on that patch falls back to one Google-assigned insert", with no gap noted.
- `backend/internal/google/service_test.go:209-223` (`TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert`) exercises the fallback on a *successful* save only; no test injects a save failure on that path.
- Parent idea: `harness/ideas/_inbox/a-failed-savesyncstate-orphans-the-google-object-just-create.md`.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.**

*Confirmed (read on this branch).* `backend/internal/google/service.go:39-42` ends "The Calendar half is covered by the deterministic id", while the 404-on-patch fallback (`:94-100`) inserts with `ev.ID = ""` and `SaveSyncState` runs after (`:105-108`); a failed save there orphans a Google-assigned event, and the retry repeats the fallback. The CODEMAP google bullet describes the fallback without the gap.

*Fix, when planned.* Correct `Sync`'s doc comment and the CODEMAP bullet to name the residual window, and pin it with `failSaveAt` on a sync that goes through the fallback (asserting the documented outcome). Making the fallback idempotent (a second deterministic id with a persisted generation counter) is more than the branch is worth today.

*Priority.* Low, below the reviewer's medium: the branch needs a manual delete on the user's calendar, Google releasing the reserved id, *and* a `SaveSyncState` failure on the same sync; the user-facing defect is a documentation overclaim first. Plan with the other google test/doc follow-ups (`fakecalendar-returns-the-same-nextid-…`, `practiceeventid-is-a-lossy-filter-…`).
