---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Two unmerged branches both add migration 0004 (0004_rls and 0004_pet_shields)

## Why
Two unmerged, `done` branches each add a migration numbered 0004: `0004_rls` (RLS plan) and `0004_pet_shields` (pet streak shield plan). `Migrate` keys versions by the full stem, so both would apply in filename order (`0004_pet_shields` before `0004_rls`) and nothing breaks at runtime. But:
- Both branches rewrite the same hard-coded applied-version lists: `TestMigrateAppliesPendingVersions`, `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` and `TestIntegrationConcurrentMigrateDoesNotRace`'s `const versions`. Both also append to the same place in the spec §3.2 DDL block. Merging both into one daily branch gives textual conflicts, and a naive resolution leaves the lists and `versions` wrong.
- Two files sharing one number breaks the "numbered DDL files" convention that `store/migrations.go` documents. A database that has applied only one of them then shows a gap-free but misleading history.

## Expected output
- Whichever branch lands second renumbers its migration to `0005_*`, and updates its tests, the spec DDL comment ("Added by migration 0005") and CODEMAP.
- Optional: a unit test fails when two `*.up.sql` share a numeric prefix.

## Evidence
- Plan under review: `harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md` — `backend/internal/store/migrations/0004_rls.up.sql`.
- `harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md` (status done, merged false), branch `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` — `backend/internal/store/migrations/0004_pet_shields.up.sql`.
- `backend/internal/store/migrations.go:74` — `version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".up.sql")`.
- Found with `git diff --name-only $(git merge-base origin/main $r) $r -- backend/internal/store/` over every `origin/harness/2026-09-2*` branch.
