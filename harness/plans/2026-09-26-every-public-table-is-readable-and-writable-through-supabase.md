---
idea: harness/ideas/_inbox/every-public-table-is-readable-and-writable-through-supabase.md
status: done
priority: high
merged: false
branch: harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase
worktree: .worktrees/every-public-table-is-readable-and-writable-through-supabase
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

## Execution summary

Built exactly per plan: `0004_rls.up.sql`/`.down.sql` (ENABLE/DISABLE RLS on all 8 tables + guarded `DO $$` revoking `anon`/`authenticated` grants), `TestMigration0004EnablesRLSOnEveryTable` and `TestEveryTableCreatedByAMigrationHasRLS` (convention test) in `migrations_test.go`, `TestIntegrationMigrateLeavesNoTableWithoutRLS` in `integration_test.go`, spec §3.2 DDL block + prose, `deploy/README.md` Supabase paragraph + owner checklist box, `harness/CODEMAP.md` `store` bullet. Three commits: `87080bd` (migration + unit tests), `2e03010` (integration test), `94d8870` (docs).

**Deviations:**
- Migrate.Apply runs each `*.up.sql` as one `tx.Exec` (confirmed by `0001_init`, which already relies on multi-statement execution in one call), so the `DO $$ … $$` block needed no adaptation — noted in the plan as an open question, resolved by inspection, no design change.
- Two pre-existing tests not named in the plan also hardcode the applied-migrations list and would have broken on `go build`/`go test ./...` once `0004_rls` existed: `TestMigrateAppliesPendingVersions` (`migrations_test.go`) and `TestIntegrationConcurrentMigrateDoesNotRace`'s `const versions = 3` (`integration_test.go`). Updated both to include `0004_rls` — required for a green `go test ./...`, not a scope change.
- The convention-test regex initially false-positived on its own migration's prose comment ("every future CREATE TABLE is followed by ENABLE ROW LEVEL SECURITY" matched `CREATE TABLE is`); fixed by stripping `-- ` line comments before matching (`stripSQLComments`) rather than reworking the comment wording, since the plan's own convention comment is worth keeping verbatim in the SQL file.

**Plan Verification:** all commands run as specified (worktree slug `every-public-table-is-readable-and-writable-through-supabase`, `COMPOSE_PROJECT_NAME=rls`, ports 55452/56399/18091 per the routine's instructions rather than the plan's placeholder). `go build ./... && gofmt -l . && go vet ./...` clean. `go test ./internal/store -run 'Migration|EveryTable' -v`: all PASS including the two new tests. `go test ./... -run Integration -p 1 -count=1 -v` against the migrated (RLS-enabled) DB: every package's integration tests PASS — `store`, `airouter`, `auth`, `google`, `notify`, `onboarding`, `pet`, `quests` — proving the API's own reads/writes are unaffected by RLS. `psql`-equivalent (`docker exec … psql`) confirmed all 8 tables `rowsecurity = t` after `Migrate`. The DDL-identity diff on the `DO $$ … END $$;` block (spec vs. migration) is byte-identical (exit 0).

**Red→green evidence:**
- `TestMigration0004EnablesRLSOnEveryTable` / `TestEveryTableCreatedByAMigrationHasRLS`: moved `0004_rls.{up,down}.sql` aside → both FAIL (`EveryTableCreated` names all 8 tables as missing RLS, reproducing the underlying bug — every table Supabase exposes has no RLS). Restored the files → both PASS.
- Convention-bites proof (Review Focus #1): added a throwaway `0099_x.up.sql` with `CREATE TABLE x (id int);` → `TestEveryTableCreatedByAMigrationHasRLS` FAILS naming `x`; deleted the file → PASS again.
- `TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent` against the live compose DB, before editing its expected list: FAILED (`applied [... 0004_rls], want [...]` — the old hardcoded list), confirming the new migration runs; updated the expectation → PASS.

**Runtime proof:** `go build ./...`, `gofmt -l .`, `go vet ./...` clean. Full unit suite in a clean shell (`env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -race`): all 13 packages `ok`. Booted the real API binary (`go run`-built, `cmd/api`) against the migrated compose DB (`COMPOSE_PROJECT_NAME=rls`, Postgres 55452, Redis 56399, API 18091) — `/healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`. Exercised a normal authenticated path end to end: inserted a `users` row directly, issued a real JWT via `auth.TokenIssuer.Issue` and a matching Redis session via `auth.RedisSessionStore.Put` (both from a throwaway `cmd/rlsproof` program inside the module, deleted afterward — never committed), then `GET /api/v1/pet/status` with that bearer token → `200`, and the resulting `pet_states` row was confirmed in Postgres (`INSERT … ON CONFLICT DO NOTHING` succeeded under RLS via the API's own `postgres` role, proving the app is unaffected). `docker exec … psql … pg_tables` showed `rowsecurity = t` for all 8 tables. Cleaned up: killed the API process (`pgrep -fl` confirmed gone), `docker compose down` (project `rls`; `docker ps --filter name=rls` empty afterward), removed the scratch `backend/.env` and the throwaway `cmd/rlsproof` (never staged/committed).

**CI:** pushed `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase`. Run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36217414467 — all 5 jobs green (`backend-unit`, `backend-integration`, `frontend`, `docker-images`, `harness-tooling`).

**Left for the owner:** apply this migration to the live Supabase project (the hand-applied 2026-09-25 SQL-editor stopgap already has RLS on, but `0004_rls` must still run there to register in `schema_migrations` and to revoke the `anon`/`authenticated` grants, which the stopgap did not do) and then disable Supabase's Data API / remove `public` from *Exposed schemas* (`deploy/README.md` owner checklist). After that: verify in Supabase Security Advisor (0 `rls_disabled_in_public`) and `SET ROLE anon; SELECT count(*) FROM users;` → permission denied, per this plan's "Live proof after the ship" note.
