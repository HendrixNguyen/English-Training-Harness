# CODEMAP

One paragraph per package/module. Read this before exploring code. Executors update the paragraph for any package they change; reviewers correct it.

## Planned backend packages (`backend/internal/`) — none exist yet

- **store** — Postgres + Redis clients, migrations (DDL from spec §3.2). Everything else depends on it.
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
