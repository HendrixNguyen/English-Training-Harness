---
type: bug
status: planned
source: human
run: _inbox
priority: high
plan: harness/plans/2026-09-26-every-public-table-is-readable-and-writable-through-supabase.md
---
# Every public table is readable and writable through Supabase's anon REST API because migrations never enable RLS

## Why
Supabase's Security Advisor flags all eight `public` tables as **critical: RLS disabled**. Supabase exposes the `public` schema through its auto-generated PostgREST API, and its `anon` and `authenticated` roles hold full grants on every table (8 tables × 7 privileges each, verified 2026-09-25). With RLS off, anyone who obtains the project's anon key — which Supabase treats as a publishable, client-side key — can `SELECT`/`INSERT`/`UPDATE`/`DELETE` `users` (Google refresh tokens, encrypted), `push_subscriptions` (endpoints + keys), `roadmaps`, `daily_progress`, `pet_states`, `exercises`, `google_sync`, `schema_migrations`. The Go API never uses that REST API (it connects as `postgres`, which has `rolbypassrls = true`, over the pooler), so the fix costs the app nothing. Standing priority: security outranks tidiness.

**Stopgap applied on the live database by the owner (2026-09-25, Supabase SQL editor):** `ALTER TABLE … ENABLE ROW LEVEL SECURITY` on all eight tables, no policies. Verified afterwards: `rowsecurity = t` for every table, `SET ROLE anon; SELECT count(*) FROM users` → 0, `SET ROLE anon; INSERT … pet_states` → "new row violates row-level security policy", `/healthz` → postgres ok. The API is unaffected because it connects as `postgres` (`rolbypassrls = t`). It is still only a stopgap: nothing in the migrations enforces it, so the next `CREATE TABLE` reopens the hole — the plan below makes it durable.

## Expected output
- Migration `0002_rls.up.sql` (+ down): `ENABLE ROW LEVEL SECURITY` on every existing table, and a documented convention that every future `CREATE TABLE` in `backend/internal/store/migrations/` is followed by `ENABLE ROW LEVEL SECURITY` (a migration test greps for it). No policies are created — the API's role bypasses RLS on Supabase (`bypassrls`) and is the owner on plain Postgres/Dokploy (owners bypass RLS unless `FORCE`), so behaviour is identical on both targets; the migration must be a no-op-safe idempotent statement on a plain Postgres 16 (integration test runs it).
- Optionally in the same migration: `REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated` guarded by `DO $$ … IF EXISTS role$$` so it is skipped on plain Postgres where those roles do not exist.
- Backend spec DDL (§3.2 / backend spec) gets the RLS lines so the "DDL identical to migrations" rule holds; `deploy/README.md` Target A gains an owner step: Supabase → Project Settings → Data API → disable (or remove `public` from exposed schemas) — the API does not use it. `deploy/README.md` lives on the deploy branch; make that edit conditional as earlier plans did.
- `make check` + `backend-integration` green; the migration test proves RLS is on for every table after `up`.

## Evidence
```
$ psql "$DATABASE_URL" -c "select tablename, rowsecurity from pg_tables where schemaname='public'"
daily_progress|f  exercises|f  google_sync|f  pet_states|f  push_subscriptions|f  roadmaps|f  schema_migrations|f  users|f     # before
$ psql … -c "select current_user, rolbypassrls from pg_roles where rolname=current_user"
postgres|t
$ psql … -c "select grantee, count(*) from information_schema.role_table_grants where table_schema='public' and grantee in ('anon','authenticated','service_role') group by 1"
anon|56  authenticated|56  service_role|56
```
Supabase docs: https://supabase.com/docs/guides/database/postgres/row-level-security ; advisor lint `rls_disabled_in_public`: https://supabase.com/docs/guides/database/database-advisors?lint=0013_rls_disabled_in_public

## Evaluation
_Evaluator, 2026-09-26 — daily decide (bug queue, ranked #3: security)._

**Select — high.**

*Is the Why real?* Yes — Supabase exposes `public` through PostgREST with full grants to `anon`/`authenticated`, the owner verified 8 tables × 7 privileges, and the only thing standing between the anon key and `users` (encrypted refresh tokens) is the owner's hand-applied `ENABLE ROW LEVEL SECURITY`. The next `CREATE TABLE` reopens it. Security outranks tidiness (AGENTS.md standing priority).

*Root cause.* `backend/internal/store/migrations/0001_init.up.sql` … `0003` create tables with no RLS and no grant changes; nothing in the repo knows Supabase's extra roles exist. The Go API connects as `postgres` (`rolbypassrls = t`) on Supabase and as the owner on plain Postgres, so enabling RLS with no policies is invisible to the app on both targets.

*Smallest correct fix.* Migration `0004_rls` (the idea says `0002`; `0002`/`0003` exist) enabling RLS on all eight tables; a `DO $$` block revoking `anon`/`authenticated` grants only where those roles exist; a unit test enforcing the convention (every table any up-migration creates is RLS-enabled by some up-migration); the integration migration test asserting `pg_tables.rowsecurity` for all eight after `Migrate`; the spec DDL and runbook updated. One plan, ≈2.5 h.
