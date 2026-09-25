---
type: bug
status: proposed
source: human
run: _inbox
priority: high
---
# Every public table is readable and writable through Supabase's anon REST API because migrations never enable RLS

## Why
Supabase's Security Advisor flags all eight `public` tables as **critical: RLS disabled**. Supabase exposes the `public` schema through its auto-generated PostgREST API, and its `anon` and `authenticated` roles hold full grants on every table (8 tables × 7 privileges each, verified 2026-09-25). With RLS off, anyone who obtains the project's anon key — which Supabase treats as a publishable, client-side key — can `SELECT`/`INSERT`/`UPDATE`/`DELETE` `users` (Google refresh tokens, encrypted), `push_subscriptions` (endpoints + keys), `roadmaps`, `daily_progress`, `pet_states`, `exercises`, `google_sync`, `schema_migrations`. The Go API never uses that REST API (it connects as `postgres`, which has `rolbypassrls = true`, over the pooler), so the fix costs the app nothing. Standing priority: security outranks tidiness.

**Not yet applied on the live database (2026-09-25):** the orchestrator prepared `ALTER TABLE … ENABLE ROW LEVEL SECURITY` for all eight tables (no policies → deny for `anon`/`authenticated`; `postgres` bypasses, so the API is unaffected) but the production DDL needs the owner's hand — run it in the Supabase SQL editor, or approve the orchestrator running it. Either way it is a stopgap: it is *not* in the migrations, so the next `CREATE TABLE` reopens the hole and the advisor flags it again.

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
