---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Two done branches both add migration 0004 pet shields and rls

## Why
`harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` adds `backend/internal/store/migrations/0004_pet_shields.{up,down}.sql`; `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase` (status done, not merged) adds `0004_rls.{up,down}.sql`. Each was cut from a `main` whose highest migration is `0003`, so each is correct alone, and the executor of the shield plan flagged the risk in its execution summary. Merged together (e.g. into one daily integration branch) the migrator (`store/migrations.go` `Migrate`: sorted full file names as versions) would apply both, but: `git merge-tree` reports conflicts in `store/integration_test.go` (`versions` count, `want` list), `store/migrations_test.go` (hardcoded applied-version list) and the backend spec's DDL block; the numbering convention (one migration per number, spec DDL in order) breaks; and a database that already has one `0004` applied gets the other out of order. Whoever resolves the conflict by hand under time pressure is the risk.

## Expected output
The branch that merges second renumbers its migration to `0005_*` (file names, the `want`/`versions` lists in both store tests, the spec's "Added by migration 0005" block, CODEMAP), on its own branch before integration, so `main` never carries two `0004`s. The daily integration run checks for duplicate migration numbers before merging plan branches.

## Evidence
- `git ls-tree origin/harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase backend/internal/store/migrations/` → `0004_rls.down.sql`, `0004_rls.up.sql`.
- `git ls-tree origin/harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days …` → `0004_pet_shields.*`.
- `git merge-tree --write-tree --name-only origin/<rls branch> origin/<shield branch>` → CONFLICT in `backend/internal/store/integration_test.go`, `backend/internal/store/migrations_test.go`, `harness/CODEMAP.md`, `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`.
- Plans: `harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md` (execution summary, deviation 2), `harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md`.
