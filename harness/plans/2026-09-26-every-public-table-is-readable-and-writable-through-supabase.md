---
idea: harness/ideas/_inbox/every-public-table-is-readable-and-writable-through-supabase.md
status: approved
priority: high
merged: false
---
# Migration 0004: row-level security on every table (Supabase closes the anon REST hole; plain Postgres unaffected) — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B3** of 2026-09-26. **Estimate:** 2.5 h. **Branch:** `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase`.

**Idea:** `harness/ideas/_inbox/every-public-table-is-readable-and-writable-through-supabase.md`

**Goal:** After `store.Migrate` runs, every table in `public` has row-level security enabled (no policies), Supabase's `anon`/`authenticated` roles hold no table grants, and a test fails whenever a future migration creates a table without enabling RLS — so the owner's hand-applied stopgap becomes durable and the API keeps working unchanged on Supabase (`postgres` bypasses RLS) and on plain Postgres (the owner bypasses RLS unless `FORCE`).

**Architecture:** `store` only — one new migration pair, two tests, the spec DDL block, the runbook. No Go code path changes; `Migrate` picks up `0004` by filename order.

**Spec:** backend spec §3.2 DDL block ("must stay identical to the migrations" — AGENTS.md); 1st-thinking §8 deployment. Where the specs are silent on RLS, this plan adds a paragraph rather than changing behaviour.

**Why no policies:** the API's role bypasses RLS on both targets (verified `rolbypassrls = t` on Supabase; table owner on compose/Dokploy). Adding policies would be dead configuration the app never exercises. `FORCE ROW LEVEL SECURITY` is **not** used — it would lock the owner role out on plain Postgres.

## Global Constraints
- Work in `.worktrees/<slug>`; run Go from `backend/`; `rg`/`timeout` not installed (`grep -n`, `go test -timeout 120s`); integration tests via `COMPOSE_PROJECT_NAME=<slug> make up` with free ports, `make down` after.
- The `DO $$ … $$` block must be a no-op on plain Postgres 16 (roles absent) and idempotent on Supabase (running twice is fine — `REVOKE` on nothing is fine). Never `DROP ROLE`, never touch `service_role` or `postgres`.
- `schema_migrations` is created by the migrator before any file runs, so `0004` may `ALTER` it too.
- The down migration disables RLS on the same eight tables and does **not** re-grant (grants were Supabase's defaults, not ours).
- `gofmt -l internal/store` prints nothing.

## Review Focus
1. The convention test compares **sets**: `{tables created by any *.up.sql}` ∪ `{schema_migrations}` == `{tables with ENABLE ROW LEVEL SECURITY in any *.up.sql}`. Adding `0005` with a new table and no RLS line must fail it (executor proves with a throwaway file, then deletes it).
2. Integration: after `Migrate` on an empty database, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND NOT rowsecurity` is empty; the existing `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` expects `0004_rls` in the applied list.
3. Integration: the API's own writes still work under RLS — the existing package integration tests (`auth`, `onboarding`, `quests`, `pet`, `google`, `notify`) run against the migrated schema as the connecting role and pass unchanged.
4. The `DO` block is guarded per role: `IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon')`.
5. Spec DDL block gains the exact `0004` text after the `0003` block, with the same "Added by migration 0004" comment style.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/migrations/0004_rls.up.sql` | `ALTER TABLE … ENABLE ROW LEVEL SECURITY` × 8; guarded `REVOKE ALL ON ALL TABLES/SEQUENCES IN SCHEMA public FROM anon, authenticated` |
| `backend/internal/store/migrations/0004_rls.down.sql` | `ALTER TABLE … DISABLE ROW LEVEL SECURITY` × 8 |
| `backend/internal/store/migrations_test.go` | `TestMigration0004EnablesRLSOnEveryTable`, `TestEveryTableCreatedByAMigrationHasRLS` (convention) |
| `backend/internal/store/integration_test.go` | expected list gains `0004_rls`; `TestIntegrationMigrateLeavesNoTableWithoutRLS` |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §3.2 DDL: the `0004` block + one paragraph on RLS-without-policies |
| `deploy/README.md` | Supabase paragraph: RLS is on by migration; owner step — Data API off / `public` unexposed; the hand-applied stopgap is superseded |
| `harness/CODEMAP.md` | `store` bullet |

## Tasks

### Task 1: The migration, with its unit tests first

**Files:** `backend/internal/store/migrations/0004_rls.up.sql`, `0004_rls.down.sql`, `backend/internal/store/migrations_test.go`.

- [ ] **Step 1 (tests first):** In `migrations_test.go` (pattern of `TestMigration0003AddsPetVerdictDates`, using `readMigration`):
  - `TestMigration0004EnablesRLSOnEveryTable`: `up` contains `ALTER TABLE <t> ENABLE ROW LEVEL SECURITY` for each of `users, push_subscriptions, pet_states, daily_progress, roadmaps, exercises, google_sync, schema_migrations`, contains `pg_roles` and `'anon'` and `'authenticated'` and `REVOKE ALL`; `down` contains `DISABLE ROW LEVEL SECURITY` for each of the eight and no `GRANT`.
  - `TestEveryTableCreatedByAMigrationHasRLS` (the convention): walk `MigrationsFS` for `*.up.sql`; regex `(?i)CREATE TABLE(?: IF NOT EXISTS)?\s+(\w+)` → set A (+ `schema_migrations`); regex `(?i)ALTER TABLE\s+(\w+)\s+ENABLE ROW LEVEL SECURITY` → set B; assert A == B with a message naming the offending table and the rule ("every CREATE TABLE in a migration needs ENABLE ROW LEVEL SECURITY in the same or a later up-migration — Supabase exposes public through PostgREST").
- [ ] **Step 2:** Write `0004_rls.up.sql`:

```sql
-- 0004: row-level security on every table. Supabase exposes the public schema
-- through PostgREST with full grants to anon/authenticated; RLS with no
-- policies denies them everything. The API's role bypasses RLS (postgres on
-- Supabase, table owner elsewhere), so the app is unaffected. Convention:
-- every future CREATE TABLE is followed by ENABLE ROW LEVEL SECURITY
-- (TestEveryTableCreatedByAMigrationHasRLS).
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE pet_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE roadmaps ENABLE ROW LEVEL SECURITY;
ALTER TABLE exercises ENABLE ROW LEVEL SECURITY;
ALTER TABLE google_sync ENABLE ROW LEVEL SECURITY;
ALTER TABLE schema_migrations ENABLE ROW LEVEL SECURITY;

-- Supabase only: drop the default REST grants. Skipped where the roles do not exist.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM authenticated;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM authenticated;
    END IF;
END $$;
```
  and `0004_rls.down.sql` with the eight `DISABLE ROW LEVEL SECURITY` lines. Check how `Migrate` splits/executes a file (`grep -n 'Exec\|split' internal/store/migrate*.go`): if it executes the whole file as one `Exec`, the `DO $$` block is fine; if it splits on `;`, the block must be adapted (e.g. a single-statement `DO` with no inner semicolons is impossible — then switch to one `Exec` per file, with a test) — say which in the commit.
- [ ] **Step 3:** `go test -timeout 60s ./internal/store -run 'Migration0004|EveryTableCreated|Migration0001' -v` green. Prove the convention test bites: add a throwaway `0099_x.up.sql` with `CREATE TABLE x (id int);` → test FAILS naming `x`; delete the file. Commit: `store: 0004_rls — row-level security on every table; anon/authenticated grants revoked where those roles exist`.

### Task 2: Integration proof

**Files:** `backend/internal/store/integration_test.go`.

- [ ] **Step 1:** Update the expected applied list to `[]string{"0001_init", "0002_google_sync", "0003_pet_verdict_dates", "0004_rls"}`.
- [ ] **Step 2:** Add `TestIntegrationMigrateLeavesNoTableWithoutRLS`: `reset`, `Migrate`, then `SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND NOT rowsecurity ORDER BY 1` → no rows (list them in the failure message). Then run the down file for `0004` through the pool (`readMigration` + `Exec`) and assert the eight rows come back with `rowsecurity = false` — proving `down` is exact — then `reset`.
- [ ] **Step 3:** With the test stack up: `go test -timeout 300s ./... -run Integration -p 1 -count=1` — **every package**, so the API's own writes are shown to work under RLS. Commit: `store: integration test — no public table without RLS after Migrate`.

### Task 3: Spec DDL, runbook, CODEMAP

**Files:** the backend spec, `deploy/README.md`, `harness/CODEMAP.md`.

- [ ] **Step 1:** Backend spec §3.2: after the `0003` `ALTER TABLE pet_states` block (line ≈158), add `-- Added by migration 0004 (RLS): …` followed by the eight `ENABLE` lines and the `DO` block verbatim (the DDL-identity rule); one sentence in prose below the block: RLS is enabled with no policies because the API's role bypasses it; Supabase's Data API should be disabled or `public` unexposed.
- [ ] **Step 2:** `deploy/README.md` Target A, Postgres paragraph: add "Migration `0004_rls` enables row-level security on every table and revokes `anon`/`authenticated` grants at boot; the SQL-editor stopgap of 2026-09-25 is superseded. Owner step: Project Settings → Data API → disable (or remove `public` from *Exposed schemas*) — the API never uses PostgREST." Owner checklist: one new box for that step.
- [ ] **Step 3:** CODEMAP `store` bullet: "`0004_rls` enables RLS on every table (no policies; the API role bypasses it) and revokes Supabase's `anon`/`authenticated` grants where those roles exist; `TestEveryTableCreatedByAMigrationHasRLS` makes every future `CREATE TABLE` carry `ENABLE ROW LEVEL SECURITY`."
- [ ] **Step 4:** Commit: `docs: RLS migration in the spec DDL, runbook and CODEMAP`.

## Verification
From `backend/` in the worktree:
```
go build ./... && gofmt -l . && go vet ./...
go test -timeout 120s ./internal/store -count=1 -v -run 'Migration|EveryTable' | grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)'
COMPOSE_PROJECT_NAME=<slug> make up && export TEST_DATABASE_URL=… TEST_REDIS_URL=…
go test -timeout 300s ./... -run Integration -p 1 -count=1 -v | grep -E 'RLS|Migrate|^(ok|FAIL)'   # all packages ok
psql "$TEST_DATABASE_URL" -c "select tablename, rowsecurity from pg_tables where schemaname='public' order by 1"   # all t (after a Migrate run)
make down
diff <(sed -n '/-- Added by migration 0004/,/END \$\$;/p' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md") <(sed -n '/-- Supabase only/,$p' internal/store/migrations/0004_rls.up.sql) || true   # eyeball: identical statements
git push -u origin harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase   # backend-unit + backend-integration green
```
Live proof after the ship (owner, recorded in the Execution summary as a follow-up, not run by the executor): Supabase Security Advisor shows 0 `rls_disabled_in_public`; `SET ROLE anon; SELECT count(*) FROM users;` → permission denied.
