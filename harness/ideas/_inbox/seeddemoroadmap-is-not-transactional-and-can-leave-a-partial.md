---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Overtaken: store.SeedDemoRoadmap was deleted by the onboarding slice (quests' integration test now seeds through onboarding.PgRepo.SaveAssessment, which is transactional)."
---

# SeedDemoRoadmap is not transactional and can leave a partial active roadmap

## Why
`store.SeedDemoRoadmap` issues 85 statements on the pool with no transaction
(`backend/internal/store/seed.go:25-47`): one `INSERT … roadmaps … is_active TRUE RETURNING id`,
then 28 x 3 separate `pool.Exec` inserts in a nested loop. The roadmap is marked active by the first
statement, so any failure part-way (a dropped connection, a cancelled context, a `task_category`
mismatch) leaves an **active** roadmap whose later days have no exercises. `GET /quests/daily` then
answers 200 with `"tasks": []` for those days — a correct-looking empty screen with no error
anywhere. It also never deactivates a pre-existing roadmap, so calling it twice for one user leaves
two rows with `is_active = TRUE`; `ActiveRoadmap`'s `ORDER BY created_at DESC LIMIT 1` tolerates that
but it is luck, not design.

Being a stopgap does not make it harmless: it is the only thing that creates roadmaps until
onboarding lands, so every developer and every integration run goes through it, and a half-seeded
database is a confusing failure to debug.

The unit test that accompanies it, `TestDemoRoadmapShapeMatchesSpec61`
(`backend/internal/store/seed_test.go`), asserts only that two package constants equal the literals
written beside them — it never calls `SeedDemoRoadmap`. The real coverage is
`TestIntegrationDailyAndProgressAgainstRealServices` in `quests`, which is gated, so `go test ./...`
covers this function not at all.

Otherwise the seed is well-formed: it is a single file, the `TEMPORARY … DELETE this file` comment is
unambiguous, nothing in production code calls it (only the gated integration test), and it does write
the `title` and `duration_minutes` keys the `Task` DTO reads — confirmed at runtime
(`"title":"Day 1 vocabulary","duration_minutes":10`).

## Expected output
`SeedDemoRoadmap` runs inside one `pgx` transaction (`pool.Begin` / `defer tx.Rollback` /
`tx.Commit`) so it either produces a complete 28x3 roadmap or nothing, and the 84 exercise inserts
become one multi-row `INSERT` (or `pgx.CopyFrom`) instead of 84 round trips. It also deactivates the
user's existing roadmaps (`UPDATE roadmaps SET is_active = FALSE WHERE user_id = $1`) inside the same
transaction, so "the active roadmap" stays singular. `seed_test.go` either tests something real or
is folded into the gated integration test.

## Evidence
- Plan: `harness/plans/2026-09-22-quests-daily-quest-suite-and-progress-recording.md` (Task 7).
- `backend/internal/store/seed.go:25-47`.
- `backend/internal/store/seed_test.go:5-18` — asserts constants against their own literals.
- `backend/internal/quests/integration_test.go:52-54` — the only caller.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — overtaken.** `find backend -name 'seed*'` → nothing; CODEMAP `onboarding` records the deletion.
