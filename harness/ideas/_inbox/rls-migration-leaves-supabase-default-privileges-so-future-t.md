---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# RLS migration leaves Supabase default privileges so future tables views and functions are granted to anon again

## Why
`0004_rls` closes the anon REST hole for the eight tables that exist today, but it does not make the fix durable the way the idea asked ("the next `CREATE TABLE` reopens the hole").
- **Default privileges are untouched.** `REVOKE ALL ON ALL TABLES IN SCHEMA public` covers existing objects only. Supabase's `ALTER DEFAULT PRIVILEGES … GRANT ALL ON TABLES/SEQUENCES/FUNCTIONS TO anon, authenticated` stays in force, so every future table, view, sequence and function is granted to `anon` again.
- **Views bypass RLS.** A view runs with its owner's rights (`postgres`, which has `rolbypassrls`), so RLS on the base table does not apply. Any future `CREATE VIEW` over `users` is world-readable through PostgREST, and `TestEveryTableCreatedByAMigrationHasRLS` only matches `CREATE TABLE`. It also misses `CREATE UNLOGGED TABLE`, `CREATE TABLE "quoted"` and `CREATE MATERIALIZED VIEW`.
- **The Supabase-only `DO $$` block is never executed by any test.** CI's Postgres has no `anon`/`authenticated` roles, so the revoke branch is dead code under test.

The owner's manual step (Data API off) also closes this, but it is a checklist item, not something the migrations enforce.

## Expected output
- `0004_rls` (or a follow-up migration) also runs, inside the same role guard, `ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated`, plus the same for `SEQUENCES` and `FUNCTIONS`. The spec DDL block is kept identical.
- The convention test fails on any up-migration containing `CREATE VIEW`, `CREATE MATERIALIZED VIEW`, `CREATE UNLOGGED TABLE` or a quoted/schema-qualified table name that has no matching RLS line. Alternatively, a live integration check over `pg_class` (relkind `r`/`v`/`m` in `public`) replaces the regex as the source of truth.
- An integration test creates `anon`/`authenticated` roles with Supabase-style grants and default privileges before `Migrate`. It then asserts zero `role_table_grants` for them afterwards, and that a table created after `Migrate` carries no `anon` grant.

## Evidence
- Plan: `harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md` (review 2026-09-27).
- `backend/internal/store/migrations/0004_rls.up.sql` — the `DO $$` block revokes only `ON ALL TABLES` / `ON ALL SEQUENCES`, with no `ALTER DEFAULT PRIVILEGES`.
- `backend/internal/store/migrations_test.go` — `createTableRe = (?i)CREATE TABLE(?:\s+IF NOT EXISTS)?\s+(\w+)`.
- Reviewer simulation on the isolated Postgres 16 (script `rlssim.sh`: roles `anon`/`authenticated`, Supabase-style `GRANT` and `ALTER DEFAULT PRIVILEGES`, then 0001–0004):
  - Before 0004: `anon|56 authenticated|56`.
  - After 0004 applied twice: no grants, and `set role anon; select count(*) from users` → `ERROR: permission denied for table users`. The fix works for existing tables.
  - Then `CREATE TABLE future_x …; CREATE VIEW users_v AS SELECT id, email FROM users;` → `future_x|7`, `users_v|7` anon grants, and `set role anon; select email from users_v;` → `leak@example.com`.
