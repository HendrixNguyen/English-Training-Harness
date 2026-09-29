---
plan: harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/rls-migration-leaves-supabase-default-privileges-so-future-t.md, harness/ideas/_inbox/two-unmerged-branches-both-add-migration-0004-0004-rls-and-0.md]
---
# Review — Migration 0004: row-level security on every table (Supabase closes the anon REST hole; plain Postgres unaffected)

**Plan:** `harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md`
**Branch/worktree:** `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase` / `.worktrees/every-public-table-is-readable-and-writable-through-supabase`
**Diff:** `git diff main...harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase --stat`

## Plan vs idea
Delivered for the current schema. The idea asked for:
- RLS on every existing table.
- A convention test for future `CREATE TABLE`s.
- An optional guarded revoke of `anon`/`authenticated`.
- The spec DDL kept identical.
- A runbook owner step.
- Green CI.

All of these exist. The idea's `0002_rls` naming was corrected to `0004_rls` by the evaluator. The idea's durability goal ("the next `CREATE TABLE` reopens the hole") is only partly met. Supabase's default privileges still grant every future table, view, sequence and function to `anon`. A future view bypasses RLS entirely, and the convention test only looks at `CREATE TABLE`. I filed this as medium, not a blocker, because nothing in today's schema is exposed and the owner's Data API step closes it too.

## Code vs plan
Reviewed at origin head `94d8870` in a detached reviewer worktree, with merge base `67ad0c0`. The diff is 7 files, +219/−5. It merges cleanly with `origin/main` (`27f57e1`).
- Task 1, the migration pair and unit tests: followed. The `DO $$` block runs because `Apply` executes a whole file in one `tx.Exec`, which is justified in the summary. The comment-stripping deviation in the convention regex is justified.
- Task 2, integration: followed. Two extra hard-coded lists (`TestMigrateAppliesPendingVersions`, `const versions = 4`) were updated, which was required and justified.
- Task 3, spec DDL, runbook and CODEMAP: followed. The spec's `0004` statements are identical to the migration (comment-stripped `diff`, exit 0). The runbook has the paragraph and one owner checkbox. CODEMAP `store` has the sentence.

Verification, re-run by the reviewer from `backend/`:
```
go build ./... && gofmt -l . && go vet ./...     -> build-ok, gofmt silent
go test ./internal/store -run 'Migration|EveryTable' -v
--- PASS: TestMigration0001MatchesSpec32 / 0001DownDropsEverything / 0002CreatesGoogleSync / 0003AddsPetVerdictDates
--- PASS: TestMigration0004EnablesRLSOnEveryTable
--- PASS: TestEveryTableCreatedByAMigrationHasRLS
env -u … make check                              -> all 13 packages ok under -race, fmt/vet silent
# isolated stack rv-auth (pg 5442 / redis 6392)
go test -timeout 600s ./... -run Integration -p 1 -count=1 -v
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
--- PASS: TestIntegrationMigrateLeavesNoTableWithoutRLS
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace
ok for every package (auth, google, notify, onboarding, pet, quests, store, airouter): the API's writes work under RLS
diff <(spec 0004 block minus comments) <(0004_rls.up.sql minus comments)  -> IDENTICAL
gh run list --branch <branch> --limit 1 -> completed success (run 36217414467)
```
Extra reviewer proof, a Supabase simulation that CI does not run. I created `anon`/`authenticated` roles with Supabase-style grants and `ALTER DEFAULT PRIVILEGES`, then applied 0001–0003, then applied 0004 twice:
- Grants went from `anon|56 authenticated|56` to none.
- `set role anon; select count(*) from users` → `permission denied`.
- The `DO` block is idempotent.
- Then `CREATE TABLE future_x` and `CREATE VIEW users_v AS SELECT id, email FROM users` gave `anon` 7 privileges on each, and `set role anon; select email from users_v` returned `leak@example.com`.

## Quality
- Boundaries: `store` only, with no Go path changes.
- Tests: the unit tests are honest. The convention test was shown to bite with a throwaway `0099_x`, per the summary. Gaps:
  - The Supabase `DO $$` branch is never exercised by CI, because no `anon` role exists there.
  - The regex misses views, materialized views, `UNLOGGED` tables and quoted names.
  - Both are in the medium bug.
- Security: the current exposure is closed on Supabase once the migration runs. The remaining hole is future objects, via default privileges and views. Medium.
- Integration hazard: `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` also adds a `0004_*` migration, `0004_pet_shields`, which is ALTER-only and adds no new table. Both would apply, since versions are full stems, but both edit the same hard-coded version lists and the same spec DDL spot. Filed low, and flagged for the orchestrator: renumber the second one to land to `0005`.
- Owner follow-ups listed in the summary: run the migration on the live Supabase, turn the Data API off, and check the Security Advisor. These are correct and are the owner's.

## Bugs filed
- `harness/ideas/_inbox/rls-migration-leaves-supabase-default-privileges-so-future-t.md` (medium): default privileges are not revoked, views bypass RLS, the convention test only matches `CREATE TABLE`, and the `DO` block is untested.
- `harness/ideas/_inbox/two-unmerged-branches-both-add-migration-0004-0004-rls-and-0.md` (low): duplicate 0004 with the pet-shields branch.

## Verdict
**pass-with-bugs.** Delivered, verified, CI green, no blockers. If the pet-shields branch goes into the same daily branch, resolve the version-list and spec conflicts and prefer renumbering one migration to `0005`.
