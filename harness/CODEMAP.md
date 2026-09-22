# CODEMAP

One paragraph per package/module. Read this before exploring code. Executors update the paragraph for any package they change; reviewers correct it.

## Backend packages (`backend/internal/`)

- **store** — Go module `github.com/HendrixNguyen/English-Training-Harness/backend` (Go 1.25, Gin, pgx/v5, go-redis/v9). Postgres + Redis clients from `DATABASE_URL` / `REDIS_URL`; `store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` applies `internal/store/migrations/*.up.sql` in filename order and records each in `schema_migrations` (idempotent, run on every boot from `cmd/api/main.go`). `0001_init` is the spec §3.2 DDL verbatim; `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). Redis key builders and §4 TTLs live in `keys.go` — use them, never literal key strings. Tests: `cd backend && make test` (never touches a live service — the integration tests are gated on `TEST_DATABASE_URL`/`TEST_REDIS_URL`, never on the production `DATABASE_URL`/`REDIS_URL`, because they DROP every table); `make up` + exported `TEST_DATABASE_URL`/`TEST_REDIS_URL` then `make test-integration` for the live-database assertions, which skip otherwise. Everything else depends on this package.
- **auth** — Google OAuth code exchange, refresh-token storage, JWT issue/verify. `POST /api/v1/auth/google`.
- **onboarding** — placement test (Redis `quiz:placement:*`), CEFR grading via airouter, roadmap kickoff. `POST /api/v1/onboarding/assessment`.
- **quests** — daily quests + progress. Redis `INCRBY daily:accumulated:*` first, then Postgres `daily_progress`. `GET /quests/daily`, `POST /quests/progress`.
- **pet** — health/streak/stage engine, decay on missed target, revive challenge. `GET /pet/status`, `POST /pet/revive`.
- **airouter** — LLM providers + task→provider strategies (spec §6.2). Rate limit `ratelimit:ai:*`.
- **google** — one-way Calendar + Tasks sync. `POST /integrations/google/sync`.
- **notify** — Web Push, `queue:webpush:delay` ZSET, in-process cron. `POST /settings/notifications`.

## Planned frontend areas (`frontend/`) — none exist yet

- **auth flow**, **onboarding wizard**, **daily quest screen**, **pet view**, **settings**, **PWA shell** (`@vite-pwa/nuxt`, service worker, push subscription).

## Harness tooling (`tools/harness/`)

- `frontmatter.py` YAML-subset parser · `schema.py` required keys/enums/transitions · `scan.py` walk + validate · `state.py` STATE.md renderer · `cli.py` all mutations. Tests: `python3 -m unittest discover -s tools/harness/tests`.
