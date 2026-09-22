---
type: mvp-slice
status: proposed
source: ideator
run: 2026-09-22-run-02
order: 1
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
