# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
@AGENTS.md

## Repository state

Two things live here today:

- **The spec** — [1st-thinking-architecture-doc.md](1st-thinking-architecture-doc.md), the architecture & technical spec for an adaptive English-learning PWA. It was pasted from a rich-text editor, so it contains escaped markdown (`\+`, `\!=`, `\[`) and de-indented Go; do not treat the §6.2 snippets as compilable source.
- **The agent harness** — the pipeline that builds the app. `AGENTS.md` (imported above) explains it; the design is in `docs/superpowers/specs/2026-09-22-agent-harness-design.md`. No app code exists yet; `frontend/` and `backend/` appear when the MVP slices are executed.

### Commands

```bash
python3 -m unittest discover -s tools/harness/tests -v   # harness tooling tests (stdlib only, no deps)
python3 -m unittest tools.harness.tests.test_cli -v     # one module
python3 tools/harness/cli.py validate                    # exit 1 on malformed harness artifacts
python3 tools/harness/cli.py state                       # regenerate harness/STATE.md
```

Slash commands (`/ideate`, `/idea`, `/evaluate`, `/approve`, `/execute`, `/review`, `/harness`) are defined in `.claude/commands/` and are registered at session start.

## Planned architecture

Two deployables on Railway, no shared code:

- **Frontend**: Nuxt 3 PWA (Vue 3, Pinia, Tailwind, `@vite-pwa/nuxt`, service worker for offline + Web Push).
- **Backend**: single Go binary (Gin or Fiber) that also runs the cron worker in-process — the scheduler is not a separate service.

Backing services: PostgreSQL (source of truth), Redis (session, counters, queues), Google Calendar + Tasks APIs, and a multi-provider LLM router.

### Data model boundaries

PostgreSQL owns durable state (§3.2 has the full DDL): `users`, `pet_states` (1:1 with user), `push_subscriptions`, `daily_progress` (unique on `user_id, date`), `roadmaps`, `exercises`. Two JSONB columns carry AI output: `roadmaps.roadmap_json` (the whole 28-day plan) and `exercises.content_json` (per-task payload). Enums `cefr_level`, `pet_stage`, `task_category` are Postgres types, so adding a value is a migration.

Redis holds only ephemeral state (§4), each key with an explicit TTL. The one exception is `queue:webpush:delay`, a persistent ZSET scored by UNIX timestamp that the cron worker polls to fire scheduled reminders.

### The two flows that define the system

**Daily practice loop** (§5.2) — the critical path. Task completion does `INCRBY` on `daily:accumulated:{user_id}:{date}` in Redis *first*, and the returned total is what decides whether the 30-minute target is met. Postgres `daily_progress` and the pet state change follow. Redis is the live counter; Postgres is the durable record. Keep that order.

**Onboarding** (§5.1) — OAuth callback → store Google refresh token → placement quiz graded by AI → roadmap generated → one-way push of a recurring 30-min calendar event and daily checklist tasks to Google. The Google sync is one-way (app → Google); nothing reads changes back.

The pet/plant engine is the retention mechanism: health 0–100, streak, and a stage enum that includes `wilted`. Health decays when the daily target is missed; `POST /api/v1/pet/revive` starts a 15-minute recovery challenge at 0%.

### AI agent router

`airouter` (§6.2) maps a `TaskType` to a `ProviderType` through a `strategies` map, then to an `LLMProvider` implementation. Current routing: roadmap generation and placement test → Gemini, exercise generation → DeepSeek, essay grading → OpenAI. Only two concrete drivers exist — `GeminiProvider` and `OpenAICompatibleProvider` (used for both OpenAI and DeepSeek, differing only in base URL and model).

Providers are registered only if their API key env var is set, and the router falls back to any available provider when the preferred one is missing. Adding a task type means adding a `strategies` entry; adding a provider behind an OpenAI-compatible API needs no new driver.

All roadmap/exercise generation must return strict JSON — Gemini via `response_mime_type: application/json`, OpenAI-compatible via `response_format: json_object`, both at `temperature: 0.2`. The system prompt in §6.1 hard-codes the shape callers depend on: 4 modules × 7 days = 28 daily quests, each quest 3 tasks of ~10 minutes (vocabulary/grammar, reading/listening, practice/interactive). Changing that shape breaks `GET /api/v1/quests/daily`.

AI calls are rate-limited per user at 5 req/min via `ratelimit:ai:{user_id}`.

### API surface

REST under `/api/v1`, enumerated in §7. Keep that list in sync with the code.

## Required environment variables

`DATABASE_URL`, `REDIS_URL`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`, `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`. The router also reads `GEMINI_BASE_URL`, `OPENAI_BASE_URL`, `DEEPSEEK_BASE_URL`.
