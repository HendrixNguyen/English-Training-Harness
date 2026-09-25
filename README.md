# Học 30 phút — Adaptive English Learning PWA

Learn English for 30 minutes a day with an AI-built study plan, and keep a virtual plant alive while you do it. The app is written for Vietnamese-speaking learners and syncs to Google Calendar and Google Tasks.

**What the product is and why it exists:** [docs/PRODUCT.md](docs/PRODUCT.md)

## How it works

1. **Sign in with Google**, then take an AI-graded **placement quiz** that sets your CEFR level.
2. Get a **28-day roadmap**: every day has 3 tasks of about 10 minutes each.
3. Reach **30 minutes** a day. The plant gains health and your streak grows. Miss a day and it loses health; at 0 it wilts, and a 15-minute challenge revives it.
4. A recurring **Calendar event**, daily **Tasks** and **Web Push** reminders keep you on schedule.

## Architecture

| Part | Stack |
|---|---|
| `frontend/` | Nuxt 3 PWA (Vue 3, Pinia, Tailwind, `@vite-pwa/nuxt`), client-rendered, installable, works offline |
| `backend/` | A single Go binary (Gin, pgx, go-redis). It serves `/api/v1` and runs the reminder and plant-decay cron in the same process |
| Data | PostgreSQL is the source of truth; Redis holds sessions, daily counters, rate limits and the push queue |
| AI | `airouter` sends each task type to the provider that handles it (Gemini, DeepSeek or OpenAI) |
| Hosting | Two images (`backend/Dockerfile`, `frontend/Dockerfile`), one env contract. Today: API on Railway Free, Postgres on Supabase, Redis on Upstash, PWA on Cloudflare Pages; later: the owner's Dokploy server running `deploy/compose.yml`. Runbook: `deploy/README.md` |

The frontend and backend share no code; they talk only over REST.

## Repository layout

```
frontend/        Nuxt 3 PWA
backend/         Go API + cron worker (cmd/api, internal/<package>)
project-base/    The three canonical specs (system, backend, frontend)
harness/         Agent-harness artifacts: ideas, plans, reviews, CODEMAP.md, STATE.md
tools/harness/   Harness CLI (Python, stdlib only)
.agents/         Agent roles, skills and templates (tool-neutral)
docs/            Product overview and harness design docs
```

## Getting started

Prerequisites: Go 1.25+ (see `backend/go.mod`), Node LTS, Docker and Python 3.

### Backend

```bash
cd backend
cp .env.example .env          # change POSTGRES_PORT / REDIS_PORT if 5432 / 6379 are taken
make up                       # Postgres + Redis in Docker
export DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... JWT_SECRET=...
make run                      # API on :8080; migrations run on boot
```

AI keys (`GEMINI_API_KEY`, `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`) and Web Push keys (`VAPID_*`) are optional. Without AI keys the API still starts and the AI routes return 503. Without VAPID keys reminders don't run. See `backend/.env.example`.

### Frontend

```bash
cd frontend
cp .env.example .env          # NUXT_PUBLIC_API_BASE, NUXT_PUBLIC_GOOGLE_CLIENT_ID, …
npm ci
npm run dev                   # http://localhost:3000
```

## Tests

```bash
cd backend && make test                    # unit tests; never touch a live service
cd backend && make test-integration        # needs make up + TEST_DATABASE_URL / TEST_REDIS_URL (drops tables!)
cd frontend && npm run lint && npm run typecheck && npm run test:unit
python3 -m unittest discover -s tools/harness/tests
```

CI (`.github/workflows/ci.yml`) runs all of these on every push to `main` and to `harness/**` branches.

## How this repo is built: the agent harness

Coding agents (Claude Code, Gemini CLI, Codex, …) build and improve the app through a file-driven pipeline with four roles:

**ideator** proposes features → **evaluator** ranks them and writes plans → **executor** implements each plan in a git worktree on a `harness/*` branch → **reviewer** checks the work and files bugs.

A human approves plans and merges one integration PR per day. Nothing is merged to `main` automatically.

```bash
python3 tools/harness/cli.py context       # what's going on right now (agents load this at session start)
python3 tools/harness/cli.py state         # full dashboard -> harness/STATE.md
python3 tools/harness/cli.py validate      # check harness artifacts
```

- Agent rules: [AGENTS.md](AGENTS.md)
- Package-by-package guide: [harness/CODEMAP.md](harness/CODEMAP.md)
- Harness design: [docs/superpowers/specs/2026-09-22-agent-harness-design.md](docs/superpowers/specs/2026-09-22-agent-harness-design.md)

## Specs

The three documents in `project-base/` are canonical. Where they disagree, the backend spec wins for the backend and the frontend spec wins for the frontend.

- `1st-thinking-architecture-doc.md`: goals, stack, data model, flows, AI router, endpoint list
- `Adaptive English Learning Platform - Backend Technical Specification.md`: REST contracts, security, plant maths, deployment
- `Adaptive English Learning Platform - Frontend Technical Specification.md`: service worker, stores, UI mapping, design system, wireframes
