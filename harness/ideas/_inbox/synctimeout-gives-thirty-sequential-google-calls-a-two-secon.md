---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# SyncTimeout gives thirty sequential Google calls a two-second mean budget and no resumption

## Why
One sync issues, sequentially: 1 OAuth refresh + 1 Calendar call + 1 `tasklists.insert` (when
rebuilding) + **one `tasks.insert` per roadmap day**, i.e. 28 for a full roadmap — about 31 round
trips to Google, all inside the request (`service.go:122-132`).

The budget for all of them is a single `SyncTimeout = 60 * time.Second` (`handler.go:17,28`). That
works out to a **~1.9 s mean** per call, with no slack. Per-call protection is a 15 s
`http.Client.Timeout` (`client.go:37`), so a *single* slow call consumes a quarter of the whole
budget. Nothing retries, nothing batches, nothing parallelises.

The failure is not merely a 502. It does not converge:
- The deadline fires part-way through the task loop. `state.RoadmapID` is only written *after* the
  loop completes (`service.go:133`), so the stored state has the new `tasklist_id` and an empty
  `roadmap_id`.
- The retry therefore takes the delete-and-rebuild branch (`service.go:100-106`), destroys the
  partially built list, creates a fresh one, and starts the 28 inserts from zero again.
- If Google is merely slow rather than down, every attempt burns ~30 calls against the user's
  quota, finishes no list, and leaves them where they started. Quota exhaustion then produces a
  403, which this package currently reports as `reauth_required` (sibling bug).

On the request side this is also the longest-running route in the backend by a wide margin — a
deliberate, plan-directed synchronous design, and to the slice's credit the **only** route that
bounds itself at all. It does not worsen the standing gap in
`route-has-no-overall-deadline-so-one-call-can-take-90-second.md` (there is still no server
`ReadTimeout`/`WriteTimeout` in `cmd/api`), but it adds a route that can legitimately hold a
goroutine and a Postgres connection for a full 60 s.

## Expected output
Task inserts resume instead of restarting: record progress as it is made (e.g. persist
`tasks_created_count` as the loop advances, and skip the first N days on a retry when
`state.RoadmapID` already matches and `state.TasksCreatedCount < len(days)`), so a timed-out sync
makes forward progress on the next attempt rather than deleting its own work. Alternatively, bound
the loop explicitly against the remaining `ctx` deadline and return a partial `tasks_created_count`
with a distinct status, which §6.4 would need to sanction.

Either way, `SyncTimeout`'s doc comment states the real arithmetic (~31 sequential calls, ~1.9 s
each) rather than "up to 30 Tasks calls", and a test asserts that a sync interrupted at task 12
resumes at 13 rather than at 1.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23), *Notes → Synchronous despite §6.4 "asynchronously"*.
- `backend/internal/google/handler.go:14-17,28` — `SyncTimeout = 60s` for the whole sync.
- `backend/internal/google/client.go:37` — 15 s per-call client timeout; one slow call is 25% of the budget.
- `backend/internal/google/service.go:122-132` — the sequential insert loop; `:133` — `RoadmapID` written only after it completes.
- `backend/internal/google/service.go:100-106` — the retry path that deletes the partially built list.
- Existing, referenced, not duplicated: `harness/ideas/_inbox/route-has-no-overall-deadline-so-one-call-can-take-90-second.md`.
