# CODEMAP

One paragraph per package/module. Read this before exploring code. Executors update the paragraph for any package they change; reviewers correct it.

## Backend packages (`backend/internal/`)

- **store** — Go module `github.com/HendrixNguyen/English-Training-Harness/backend` (Go 1.25, Gin, pgx/v5, go-redis/v9). Postgres + Redis clients from `DATABASE_URL` / `REDIS_URL`; `store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` applies `internal/store/migrations/*.up.sql` in filename order and records each in `schema_migrations` (idempotent, run on every boot from `cmd/api/main.go`). `0001_init` is the spec §3.2 DDL verbatim; `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). Redis key builders and §4 TTLs live in `keys.go` — use them, never literal key strings. Tests: `cd backend && make test` (never touches a live service — the integration tests are gated on `TEST_DATABASE_URL`/`TEST_REDIS_URL`, never on the production `DATABASE_URL`/`REDIS_URL`, because they DROP every table); `make up` (dev stack; copy `backend/.env.example` to `backend/.env` and set `POSTGRES_PORT`/`REDIS_PORT` if 5432 or 6379 is taken — `make down` keeps the `postgres_data` volume) + exported `TEST_DATABASE_URL`/`TEST_REDIS_URL` then `make test-integration` for the live-database assertions, which skip otherwise. Everything else depends on this package.
- **auth** — Google sign-in and sessions. `POST /api/v1/auth/google` ({code, redirect_uri}) exchanges the code, fetches the OpenID profile, upserts `users` on `google_id` (refreshes email/full_name/refresh token; never resets `cefr_current` or `target_goal`; a new user gets `target_goal = ''` because §3.2 makes it NOT NULL with no default), issues an HS256 JWT and writes `sess:{user_id}:token` for 24h. `auth.Scopes` is the single consent list (`openid email profile` + `calendar.events` + `tasks`, with `access_type=offline&prompt=consent`) — the frontend must build its login URL from `auth.AuthCodeURL` so no user has to re-consent when the google slice lands. `auth.Require(tokens, sessions)` guards every per-user route: it accepts a token only if it verifies, is unexpired, and matches the stored session byte for byte, so `DEL sess:{user_id}:token` revokes and a new sign-in supersedes the old token; `auth.UserID(c)` reads the authenticated id. Needs `JWT_SECRET`, which is **absent from the spec §8 env list**. Tests are pure (httptest fakes for Google, in-memory `UserRepo`/`SessionStore`); the upsert also has a `TEST_DATABASE_URL`-gated integration test that skips (matching `internal/store`'s convention — never gated on the production `DATABASE_URL`).
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

## CI (`.github/workflows/ci.yml`)

Three parallel GitHub Actions jobs, each capped at `timeout-minutes: 10`, on every push to `main` or a `harness/**` branch and on every pull request. A pushed `harness/*` branch is therefore checked before `/harness merge`; superseded runs are cancelled per branch but never on `main`, where each commit gets its own concurrency group. `actionlint` validates the file but not action tags or repo names — check those by hand.

- **`backend-unit`** — from `backend/`: `go build ./...`, `go vet ./...`, `go test ./... -count=1`, after a guard that fails if `DATABASE_URL`, `REDIS_URL`, `TEST_DATABASE_URL` or `TEST_REDIS_URL` is exported, so the default suite is proved to need no services. (Whether the destructive suite stays gated is `internal/store/integration_gate_test.go`'s job, not this one's.)
- **`backend-integration`** — `postgres:16-alpine` + `redis:7-alpine` service containers, `TEST_DATABASE_URL` / `TEST_REDIS_URL` exported, `go test ./... -count=1 -v -run Integration`; counts `func TestIntegration*` in `*_test.go` and fails unless that many `--- PASS` lines appear and no `--- SKIP` does, so new integration tests are picked up without editing the workflow and a silent skip is a failure. Reproduce from `backend/` with `POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose up -d --wait` (`make up` has no `--wait`), then the `TEST_*` URLs on those ports.
- **`harness-tooling`** — `python3 -m unittest discover -s tools/harness/tests` and `python3 tools/harness/cli.py validate`, so a malformed frontmatter commit fails CI.

No linter yet: `golangci-lint` defaults flag one `errcheck` in `internal/store/redis_test.go`; add it only with that fixed and a committed `.golangci.yml`.
