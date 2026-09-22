---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# migrations run on every boot with no advisory lock

## Why
`cmd/api/main.go:36` calls `store.Migrate` on every process start. For a single instance this is
correct and idempotent: `Migrate` reads `schema_migrations`, skips applied versions, and `PgMigrator.Apply`
runs the DDL and the version row in one transaction, so a failure leaves no half-applied schema.

It is not safe with more than one instance. `Migrate` does
`EnsureVersionTable` → `AppliedVersions` → `Apply` with nothing serialising the three steps. Two
processes booting at the same time both read an empty `schema_migrations`, both start applying
`0001_init`, and the loser hits `duplicate key value violates unique constraint "pg_type_typname_nsp_index"`
on `CREATE TYPE cefr_level` (or the `schema_migrations` primary key). `main.go:38` turns that into
`log.Fatalf`, so the instance dies and the platform restarts it — a crash loop that only resolves if
the winner happens to commit first. `EnsureVersionTable`'s `CREATE TABLE IF NOT EXISTS` races the
same way.

Spec §8 deploys to Railway, where a scale-up to two replicas, or a rolling deploy onto a fresh
database, is enough to hit this. It also makes a migration failure indistinguishable from a
transient one in the logs.

## Expected output
Migration is serialised across processes. The standard, dependency-free fix for Postgres:
take `pg_advisory_lock(<constant>)` on a dedicated connection at the top of `Migrate`, release it at
the end (or use `pg_try_advisory_lock` and skip with a log line when another instance holds it).
Concurrent boots then either apply exactly once or wait; no instance crashes. A test with a fake
`Migrator` that records lock/unlock ordering keeps it honest.

Secondary: `main.go` should not use a deadline-free `context.Background()` for the migration, so an
unreachable database fails the boot in bounded time rather than hanging.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Tasks 4, 5, 8).
- `backend/internal/store/migrations.go:33-47` — `EnsureVersionTable`, `AppliedVersions`, then the
  apply loop, with no lock between them.
- `backend/internal/store/postgres.go:48-51` (`CREATE TABLE IF NOT EXISTS`), `:73-87` (`Apply`).
- `backend/cmd/api/main.go:17` (`context.Background()`), `:36-39` (`Migrate` then `log.Fatalf`).
- Spec §8 "Deploy compiled Go binary as lightweight Docker container on Railway."
