---
type: mvp-slice
status: planned
source: ideator
run: 2026-09-22-run-02
order: 1
priority: high
plan: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
---
# Store: Go module, Postgres and Redis clients, migration 0001

## Why
Every spec goal — the ≥30 min/day target (`daily_progress.is_target_met`), CEFR progression (`users.cefr_current`, `roadmaps`), and the pet loop (`pet_states`) — is a row in §3.2 or a key in §4. Nothing user-visible ships until those exist in a real database with a Go process in front of them, and every later slice codes against this schema. Doing the schema once, verbatim from the spec, is cheaper than letting seven slices each invent a subset. A `go test ./...` baseline from day one means each later slice can be reviewed by re-running tests instead of by reading code, which is how the harness reviewer works.

## Expected output
Delivers (Go package `backend/internal/store`):
- `backend/go.mod` (module path chosen by the executor and recorded in CODEMAP), `backend/cmd/api/main.go` booting Gin (§2.1 says Gin/Fiber; pick Gin) with one route `GET /healthz` that pings Postgres and Redis and returns 200/503. No other endpoints.
- `store.Postgres` (pgx pool from `DATABASE_URL`) and `store.Redis` (go-redis from `REDIS_URL`) constructors, plus a config loader for the §8 env vars this slice needs (`DATABASE_URL`, `REDIS_URL`).
- Migration runner `store.Migrate(ctx)` and `backend/internal/store/migrations/0001_init.up.sql` / `.down.sql` containing the §3.2 DDL: types `cefr_level`, `pet_stage` (`seed, sprout, sapling, flowering, fruitful, wilted`), `task_category`; tables `users`, `push_subscriptions`, `pet_states` (`user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE` — the 1:1 constraint merged in plan `add-unique-user-id-to-pet-states-ddl`), `daily_progress` (`UNIQUE(user_id, date)`), `roadmaps`, `exercises`.
- Redis key builders with the §4 TTLs as constants: `sess:{user_id}:token` 24h, `quiz:placement:{user_id}` 2h, `daily:accumulated:{user_id}:{YYYY-MM-DD}` 48h, `queue:webpush:delay` (no TTL), `ratelimit:ai:{user_id}` 1m.
- `backend/Makefile` with `make test` = `go test ./...`; tests cover: migration 0001 applies to an empty database and is idempotent on re-run, `pet_states` rejects a second row for the same `user_id`, key builders produce the exact §4 strings. Integration tests skip (not fail) when `DATABASE_URL`/`REDIS_URL` are unset; a `docker-compose.yml` (dev only) provides local Postgres + Redis.
- CODEMAP `store` paragraph updated with the module path and how to run migrations.
- Tables: all six from §3.2. Redis keys: all five from §4 (builders only). Screens: none.

Depends on: nothing. Every other slice depends on this one.

## Evidence
- Spec §2.1 (lines 9–16) stack: Go + Gin/Fiber, PostgreSQL, Redis.
- Spec §3.2 (lines 144–256) DDL; line 196 `user_id UUID UNIQUE NOT NULL` in `pet_states`.
- Spec §4 (lines 258–266) Redis key table with TTLs.
- Spec §8 (lines 686–705) env vars `DATABASE_URL`, `REDIS_URL`; "Run DDL migration script".
- `harness/CODEMAP.md` "Planned backend packages" → `store`.
- `harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md` (merged) — the UNIQUE NOT NULL change this migration must carry.
- Open spec bugs in this run that touch this DDL: `reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md`, `spec-never-states-when-the-pet-states-row-is-created.md`.

## Evaluation

**Verdict: select — priority `high`.**

**Is the *Why* real?** Yes, and it is the only slice whose absence blocks every other one. `backend/` is empty; there is no module, no client, no schema. Seven of the eight slices in this run write to tables defined in §3.2 or keys defined in §4, so the alternative to this slice is seven partial, mutually inconsistent schemas. The second argument in the *Why* — that a `go test ./...` baseline is what lets the reviewer re-run verification instead of reading code — matches how `.agents/roles/reviewer.md` works and is worth the module/Makefile scaffolding cost on its own.

**Is the *Expected output* achievable in one plan?** Yes, though it is the largest slice in the run. The work is mechanical: `go mod init`, two client constructors, one SQL file transcribed verbatim from §3.2, five key builders, one `/healthz` handler. The only genuine design decision is the migration runner, and the smallest correct version (a `schema_migrations` table plus an embedded `migrations/` FS, applied in one transaction) is ~60 lines. Scope is held down by the idea itself: "No other endpoints."

**Dependencies:** none. Confirmed against the run: `_run.md` lists it at `order: 1` and every other slice names it in *Depends on*.

**Priority rationale:** `high` — it blocks MVP order outright. Nothing in the product can be demonstrated until it lands.

**Test strategy (decided here, since the idea leaves it open).** The plan makes the default `go test ./...` run entirely without live services:
- Key builders, config loading, the DDL text, the migration runner's version bookkeeping, and the `/healthz` handler are all covered by pure unit tests against interfaces and fakes.
- The two assertions that genuinely need Postgres — migration 0001 applies to an empty database and is idempotent, and `pet_states` rejects a second row for the same `user_id` — live in one `*_test.go` that calls `t.Skip` when `DATABASE_URL` is unset. Skipping (not failing) is what the idea asks for, and it keeps CI green on a machine with no database while still being runnable via `docker-compose up -d` locally.
- To keep the DDL honest without a database, a pure test asserts the exact §3.2 constraint lines are present in the embedded SQL (notably `user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE` and `UNIQUE(user_id, date)`). This is the regression guard for the merged `add-unique-user-id-to-pet-states-ddl` change.

**Decisions taken in the plan that the idea left to the executor:**
- Module path `github.com/HendrixNguyen/English-Training-Harness/backend` (the idea says "chosen by the executor and recorded in CODEMAP").
- Gin, per §2.1's "Gin/Fiber" and the idea's explicit "pick Gin".
- `pgxpool` (`jackc/pgx/v5`) and `redis/go-redis/v9` — the mainstream choices; no ORM, since §6.2 and the later slices are written in raw SQL.
- A hand-rolled migration runner rather than `golang-migrate`: one dependency less, and the runner is small enough to unit-test against a fake.

**Open questions recorded for the human (not blocking):**
1. `users.target_goal` is `NOT NULL` with no default (§3.2) but is unknown until onboarding. This migration transcribes §3.2 verbatim, so the auth slice (2) must insert `''`. If the human prefers `DEFAULT ''`, that is a spec change and belongs in a separate bug, not in this migration.
2. The two open spec bugs in this run (`reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul`, `spec-never-states-when-the-pet-states-row-is-created`) both touch `pet_states`. Migration 0001 takes §3.2 as written today (`stage pet_stage DEFAULT 'sprout'`); if those bugs are later fixed in the spec, the correction lands as migration 0002, not by editing 0001.
3. `JWT_SECRET` is absent from the §8 env list. Not this slice's problem — the config loader here reads only `DATABASE_URL`, `REDIS_URL` and `PORT` — but slice 2 will have to add it.
