---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Overtaken by the merged amend (a-rejected-post-quests-progress…): RecordProgress now returns a typed ErrInvalidDuration mapped to 400, and CheckExercise scopes the exercise to the caller's active roadmap and today's day_number (404 otherwise); both are covered in service_test.go."
---

# Progress request validation is incomplete outside the Gin binding tags

## Why
Two small gaps in what `POST /quests/progress` accepts:

**1. The service's own duration guard produces a 500, not a 400.** `RecordProgress` rejects
`seconds <= 0` with a bare `fmt.Errorf` (`service.go:59-61`), which `ProgressHandler`'s `switch`
(`handler.go:64-74`) cannot distinguish from a database failure, so it falls through to
`500 {"error":"internal_error"}`. Today that is unreachable over HTTP because Gin's
`binding:"required,gt=0"` catches 0 and negatives first and returns 400 — the tests confirm both
layers independently. But the service is an exported API of the package, the two validations are
duplicated with different contracts, and the next caller (a batch import, a future edge, another
slice) gets a 500 for a caller error. The `gt=0` tag is the only thing keeping the status code right.

**2. `MarkComplete` is not scoped to the current day.** `markCompleteSQL` matches on
`id` + `roadmap_id` only (`repo.go:76-79`), so a client can post progress against any of its own
roadmap's 84 exercises regardless of `day_number` — e.g. tick day 28's three tasks while on day 1.
Ownership is enforced (the important part), but "today's quests" is not. `GET /quests/daily` then
shows a future day pre-completed when the learner reaches it, on top of the already-recorded fact
that nothing ever resets `is_completed`.

Neither is exploitable across users and neither corrupts another user's data; both are the kind of
thing that is much cheaper to tighten now than after a client depends on the looseness.

## Expected output
- A typed `ErrInvalidDuration` (beside `ErrNoActiveRoadmap` / `ErrExerciseNotFound`) returned by the
  service and mapped to `400 {"error":"invalid_request"}` in the handler's `switch`, so both entry
  points give the same answer. A service test asserts `errors.Is(err, ErrInvalidDuration)` rather
  than just "an error".
- `MarkComplete` (or the ownership lookup that the write-before-validate fix introduces) also takes
  the resolved `day_number` and matches on it, returning `ErrExerciseNotFound` for an exercise
  outside today — with a test that posting yesterday's or tomorrow's exercise id is a 404.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Tasks 3, 5, 6).
- `backend/internal/quests/service.go:59-61`; `backend/internal/quests/handler.go:39,64-74`.
- `backend/internal/quests/repo.go:76-79`.
- `backend/internal/quests/service_test.go:213-225` — asserts only `err != nil`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — overtaken.** Both asks shipped: `quests/service.go:14,72,100` (`ErrInvalidDuration` → 400) and `service.go:89` (`CheckExercise(ctx, roadmap.ID, exerciseID, day)`). CODEMAP `quests` records it.
