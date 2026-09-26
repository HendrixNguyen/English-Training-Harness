---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A non-UUID exercise_id on POST quests progress answers 500 instead of 400 or 404

## Why
`ProgressRequest.ExerciseID` is bound with `binding:"required"` only, so any string reaches
`QuestRepo.CheckExercise`, which passes it as `$1` to `WHERE id = $1` on a `uuid` column. Postgres
answers `invalid input syntax for type uuid`, `PgRepo.CheckExercise` wraps it as a generic error, and
`ProgressHandler` falls through to `500 {"error":"internal_error"}`.

No data is written (the check now runs before the INCRBY), so this is not a data bug. But a 500 is the
wrong contract: it pages as a server fault, a client retry loop will retry a request that can never
succeed, and it is distinguishable from the `404 exercise_not_found` that other bad ids get, which
leaks a little about id shape. The quests amend plan named this in its Notes as "pre-existing; same low
inbox bug", but that bug (`progress-request-validation-is-incomplete-outside-the-gin-bi.md`) was later
rejected as overtaken, and its body never mentioned the UUID case — so nothing tracks it now.

## Expected output
- `POST /api/v1/quests/progress` with a malformed `exercise_id` answers `400 {"error":"invalid_request"}`
  (add `uuid` to the binding tag) — or `404 exercise_not_found` if the owner prefers ids stay opaque.
  Either way, never 500, and still no write to Redis or `daily_progress`.
- A handler test row for `"exercise_id":"not-a-uuid"` asserting the status and `len(h.log.calls) == 0`.
- Check the other endpoints that take an exercise or roadmap id in a body or path for the same shape.

## Evidence
- Reviewing `harness/plans/2026-09-22-a-rejected-post-quests-progress-still-writes-redis-and-daily.md`
  (Notes: "A non-UUID `exercise_id` still yields a 500").
- `backend/internal/quests/handler.go:38` — `ExerciseID string json:"exercise_id" binding:"required"`.
- `backend/internal/quests/repo.go:99-102,175-185` — `checkExerciseSQL`, error wrapped as
  `quests: checking exercise: …`; `handler.go:75` maps it to 500.
- Reproduced on origin/main `3f4242d` against `postgres:16-alpine`:
  `SELECT 1 FROM e WHERE id='not-a-uuid'` on a `uuid` column →
  `ERROR: invalid input syntax for type uuid: "not-a-uuid"`.

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Wrong status (500 vs 400) with no data write; `binding:"required,uuid"` + one handler row. Low impact — a client never sends a non-UUID. Next free bug slot.
