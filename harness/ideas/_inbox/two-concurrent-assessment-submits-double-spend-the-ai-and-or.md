---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Two concurrent assessment submits double-spend the AI and orphan a roadmap with its 84 exercises

## Why
Onboarding's "only one roadmap per user" guarantee is check-then-act with no
database constraint behind it. `Service.Assess` reads `ActiveRoadmapID` on the
pool (`service.go:48`) and only much later opens the transaction that inserts
(`repo.go:100`). A double-tapped submit button — the single most likely gesture
on the one button of the onboarding flow — puts two requests inside that window.
Both see "no active roadmap", both pay for a placement grading *and* a 28-day
roadmap generation (four Gemini calls instead of two), and both run
`SaveAssessment`.

The `roadmaps` table has no unique constraint on the active row
(`backend/internal/store/migrations/0001_init.up.sql:53-59` — `is_active
BOOLEAN DEFAULT TRUE`, nothing more), so "active" is a convention, not an
invariant. What actually saves the invariant today is accidental: `SaveAssessment`
happens to run `updateUserSQL` first (`repo.go:106`), which takes a row lock on
`users`, so the second transaction blocks there and — under READ COMMITTED —
its later `deactivateSQL` statement takes a fresh snapshot that already contains
the first roadmap and deactivates it. Reorder those two statements, or move the
user update out of the transaction, and the system silently starts producing two
active roadmaps, at which point `quests.PgRepo`'s
`ORDER BY created_at DESC LIMIT 1` (`internal/quests/repo.go:68-73`) picks a
winner non-deterministically between two rows whose `created_at` is
`CURRENT_TIMESTAMP` (transaction start) and can tie.

Even with the accident working, the observable damage is real: one roadmap row
plus its 84 `exercises` rows are written and immediately deactivated as dead
weight, and the request that won the race is answered with a `roadmap_id` that
is already `is_active = FALSE` by the time the client reads it — so the client's
`roadmap_id` does not match what `GET /api/v1/quests/daily` will serve.

## Expected output
- `roadmaps` carries a partial unique index — `CREATE UNIQUE INDEX ... ON
  roadmaps (user_id) WHERE is_active` — in a new migration, so at most one active
  roadmap per user is enforced by Postgres rather than by statement ordering. The
  backend spec's DDL is updated in the same change (AGENTS.md requires the
  migration and the spec to stay identical).
- The idempotency check is made race-safe: either `ActiveRoadmapID` is taken
  inside the transaction with `SELECT ... FOR UPDATE` on the `users` row, or the
  insert relies on the new constraint and the unique-violation is caught and
  answered as the existing-roadmap 200.
- A second concurrent submit costs at most one extra AI call, not a full extra
  roadmap, and never leaves an inactive roadmap with 84 orphan exercises behind.
- `SaveAssessment` documents that the `users` update is deliberately first for
  the lock it takes, or stops depending on it.

## Evidence
- Plan: `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md` (status `done`, branch
  `harness/2026-09-23-high-onboarding-placement-test-cefr-grading-and-roadmap-generatio`).
- Check-then-act window: `backend/internal/onboarding/service.go:48` (pool
  `SELECT`) vs `backend/internal/onboarding/repo.go:100` (`Begin`) —
  nothing between them holds a lock.
- No constraint: `backend/internal/store/migrations/0001_init.up.sql:53-59`.
- Accidental serialisation: `backend/internal/onboarding/repo.go:106`
  (`UPDATE users`) runs before `repo.go:113` (`UPDATE roadmaps SET is_active =
  FALSE`).
- Non-deterministic reader: `backend/internal/quests/repo.go:68-73` and
  `backend/internal/onboarding/repo.go:46`, both `ORDER BY created_at DESC LIMIT 1`.
- Concrete input: two `POST /api/v1/onboarding/assessment` with the same bearer
  token issued within the AI round-trip window (seconds, since two Gemini calls
  run between the check and the write).
- Not covered by any test: `TestAssessIsIdempotentWhileARoadmapIsActive`
  (`service_test.go:86`) is sequential, and
  `TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious`
  (`integration_test.go:44-51`) calls `SaveAssessment` twice in series.
