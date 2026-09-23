---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A roadmap with zero exercise rows creates an empty Google Tasks list recorded as fully synced

## Why
`Sync` distinguishes "no active roadmap" from "a roadmap exists", but not "a roadmap exists with no
exercise rows". With an active `roadmaps` row whose `exercises` are missing — onboarding crashing
between the roadmap insert and the 84 exercise inserts is the obvious way to get there, and
`seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md` is already in the inbox for the
seeding path — `DayTitles` returns an empty slice (`repo.go:117-140` returns `nil, nil` when no rows
match) and the service:

1. creates a real Google Tasks list, "English daily quests" (`service.go:107-111`);
2. saves it (`service.go:112-114`);
3. runs the insert loop zero times (`service.go:122-132`), `created` stays 0;
4. records `state.RoadmapID = rm.ID, state.TasksCreatedCount = 0` (`service.go:133`) — i.e. **marks
   this roadmap fully synced**;
5. answers `{"status":"synced", "calendar_event_id":"…", "tasks_created_count":0}`.

Two consequences. The user gets a permanently empty list in Google Tasks that we created and will
never fill: once `state.RoadmapID == rm.ID`, every later sync takes the no-op branch
(`service.go:97-99`), so even after onboarding backfills the exercises the tasks never appear.
And the 200 body is byte-identical to the legitimate "no roadmap at all" answer
(`service.go:92`), which the plan's *Notes* designates as the way "no roadmap" is conveyed — so
neither the client nor an operator can tell the two apart.

## Expected output
An active roadmap with zero exercise rows does not get marked as synced: either the service skips
the list creation entirely when `len(days) == 0` (leaving `state.RoadmapID` unset so a later sync
retries once the exercises exist), or it creates the list but leaves `RoadmapID` empty for the same
reason. A service test builds a harness whose `DayTitles` returns an empty slice for a real roadmap
and asserts no tasklist is created *and* that a second sync, after the titles appear, creates all
28 tasks.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23).
- `backend/internal/google/service.go:107-133` — list creation and `RoadmapID` assignment are unconditional on `len(days)`.
- `backend/internal/google/service.go:97-99` — the same-roadmap no-op that then locks the state in.
- `backend/internal/google/service.go:92` — the "no roadmap" response, identical to this one.
- `backend/internal/google/repo.go:117-140` — `DayTitles` returns an empty result, not an error, for a roadmap with no exercises.
- No test covers it: `service_test.go` has `TestSyncWithoutARoadmapPushesOnlyTheEvent` (the `ErrNoActiveRoadmap` branch) and nothing for a roadmap with zero days.
- Related, already filed: `harness/ideas/_inbox/seeddemoroadmap-is-not-transactional-and-can-leave-a-partial.md`.
