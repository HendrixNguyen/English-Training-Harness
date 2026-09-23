---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# A user-deleted Tasks list is never rebuilt and sync keeps answering synced with a stale count

## Why
The realistic production failure for a one-way push is that the user tidies up on Google's side.
If they delete the "English daily quests" list, `Sync` never notices and never rebuilds it.

`service.go:97-99`: when `state.TasklistID != "" && state.RoadmapID == rm.ID` the service returns
immediately with `{"status":"synced", "tasks_created_count": <stored count>}` and makes **zero**
Tasks calls. The stored count is whatever the last successful sync wrote. So the endpoint reports
`synced, tasks_created_count: 28` to a user who has no list and no tasks at all, forever — the only
escape is a brand-new roadmap, which is a 28-day event. The user presses "Sync to Google" again,
gets a 200 that says everything is fine, and nothing appears.

The Calendar half of the same scenario *is* handled: a deleted event 404s on PATCH and
`service.go:70-73` clears the id and re-inserts (`TestResyncReinsertsTheEventWhenGoogleLostIt`).
Tasks has no equivalent because nothing ever touches Google on this path.

The plan's *Notes* anticipates a narrower case — "titles edited in Google are left alone; a
partially-deleted list is not repaired" — but understates it: a *fully* deleted list is also not
repaired, and unlike an edited title, the endpoint's 200 is then a lie.

## Expected output
Either:
(a) the same-roadmap path verifies the list still exists before claiming success — a single
    `tasklists.get`, and on 404 fall through to the existing delete-and-rebuild branch (which
    already tolerates a 404 delete, `tasks.go:62-68`), at the cost of one extra call per sync; or
(b) the endpoint stops asserting a state it has not checked — e.g. a `force` flag or a
    documented "tasks are pushed once per roadmap; deleting the list in Google is not detected"
    line in CODEMAP and the §6.4 notes, so the frontend can tell the user.

Option (a) is one read of an object we created, which the 1st-thinking §5.1 "one-way" constraint is
about *subscribing to user changes*, not about existence checks — worth stating explicitly in
whichever direction is chosen. A test drives a `tasklists.get` 404 on the same-roadmap path and
asserts the list is rebuilt.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), *Notes → Idempotency granularity for Tasks is per roadmap, not per task*.
- `backend/internal/google/service.go:97-99` — the unconditional same-roadmap short-circuit returning the stored count.
- `backend/internal/google/service.go:68-77` — the Calendar path that *does* recover from a user deletion, for contrast.
- `backend/internal/google/service_test.go:96-118` (`TestResyncSameRoadmapPatchesEventAndCreatesNothing`) — pins the no-op as correct with no coverage of a missing list.
- Backend spec §6.4: the 200 body is `{"status": "synced", …}`; a `synced` that is not synced is a contract violation.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Real: a user who deletes the Tasks list in Google gets `synced, 28` forever. But it needs a deliberate user action on Google's side, the Calendar half already recovers, and the fix is one `tasklists.get` on the same-roadmap path. Ride along with the next `google` plan after the orphan fix lands (same `fakeTasks`/`TasksClient` surface).
