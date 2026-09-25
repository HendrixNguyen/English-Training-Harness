---
type: feature
status: proposed
source: human
run: 2026-09-25-run-01
priority: high
---
# Containerised deploy: Dockerfiles, production compose, runbook and CI image build

## Why
The app has no public deployment, so nothing the spec's retention loop depends on — Google sign-in, the installable PWA, Web Push reminders, the plant decaying while nobody visits — can be used by a real learner. The owner's hosting decision (conversation, 2026-09-24/25): free managed services **today** (API on Railway Free, Postgres on Supabase free, Redis on Upstash free, PWA on Cloudflare Pages — all no-card), and **later** the owner's own bare-metal server running Dokploy, which brings the stack back to the spec's original one-host topology (1st-thinking §8, backend spec §9). The deliverable must therefore be **provider-neutral**: the same two images and the same env contract deploy to either target, and moving hosts is a config change plus a second runbook section, not new work.

This is the highest-impact feature in the queue: every other feature is invisible until a learner can reach the app.

## Expected output
User-visible:
- The PWA is reachable at a public HTTPS URL, installable, with Google sign-in and push reminders working end to end against the public API. The plant decays and reminders fire on schedule without anyone visiting.

Technical (agents own everything in the repo; the owner owns accounts and secrets — see the runbook's owner checklist):
- **`backend/Dockerfile`** — multi-stage: `golang:1.2x` build of `./cmd/api` with `CGO_ENABLED=0`, final `gcr.io/distroless/static` (or `alpine`) image, non-root, `EXPOSE 8080`, `HEALTHCHECK` on `GET /healthz` (already served by `cmd/api/main.go`). Migrations keep running at boot as today. `PORT` is honoured by `config.Load`.
- **`frontend/Dockerfile`** — multi-stage: `node:20` `npm ci && npx nuxi generate` (`ssr: false` in `nuxt.config.ts`), final Caddy (or nginx) image serving `.output/public` with SPA fallback to `/index.html` and correct `Cache-Control` for the service worker (`sw.js` must be `no-cache`; hashed assets immutable). Public runtime config (`NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY`) is a build arg, documented.
- **`deploy/compose.yml`** — production compose: `api`, `web`, `postgres:16`, `redis:7` with named volumes, healthchecks, `restart: unless-stopped`, Redis `appendonly yes` (the `queue:webpush:delay` ZSET must survive restarts, §4). Env comes from `deploy/.env.example`. This is what Dokploy deploys as a Compose app; on Railway it is documentation.
- **`deploy/README.md` runbook** — one env table (every variable, where it comes from, which target needs it: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET` ≥32 bytes, `ENCRYPTION_SECRET_KEY`, `GOOGLE_CLIENT_ID/SECRET`, `GEMINI_/OPENAI_/DEEPSEEK_API_KEY`, `VAPID_PUBLIC/PRIVATE_KEY`, `VAPID_SUBJECT`, `FRONTEND_ORIGIN`), then two target sections:
  - **Railway Free today**: service rooted at `backend/`, built from the Dockerfile, sleeping off, `GOMEMLIMIT=64MiB`; Supabase `DATABASE_URL` via the **session-mode pooler** (`…pooler.supabase.com:5432`, `sslmode=require` — Railway egress is IPv4-only); Upstash `rediss://` URL (`store.NewRedis` uses `redis.ParseURL`, TLS accepted); Cloudflare Pages project rooted at `frontend/` with the same build; Google OAuth origin + `/auth/callback` redirect URI; the $1/month budget, what "workloads stopped" looks like, 3-day logs, Hobby upgrade path.
  - **Dokploy later** (deploy only; CD comes later): install, create the Compose app from `deploy/compose.yml`, Traefik + Let's Encrypt or Cloudflare Tunnel for public HTTPS (OAuth and Web Push refuse plain HTTP), scheduled Postgres backups to an S3-compatible bucket.
  - **Owner checklist** with checkboxes: account creation order, secrets, OAuth console, DNS/tunnel. Agents never hold production secrets; the plan stops at "owner completes the checklist".
  - **Post-deploy smoke check** (three curls: `/healthz` 200, `GET /api/v1/auth/google` 302 to accounts.google.com, `OPTIONS` preflight from `FRONTEND_ORIGIN` returns the CORS headers) that the 20:00 review routine runs after the owner's daily merge and reports.
- **CI**: a `docker-images` job in `.github/workflows/ci.yml` that `docker build`s both Dockerfiles on every push (no push to a registry yet), so a broken Dockerfile fails the PR, not the deploy.
- Docs: README "Hosting" row and CLAUDE.md "Planned architecture" describe both targets; backend spec §9 gets an addendum (env additions `FRONTEND_ORIGIN`, `ENCRYPTION_SECRET_KEY`, pooler note); `harness/CODEMAP.md` gains a `deploy/` entry; AGENTS.md gains a "Deployment" paragraph pointing at the runbook.

Out of scope: registry pushes and a deploy webhook (a later idea, once the Dokploy box exists: CI pushes images to GHCR and calls Dokploy's deploy webhook on merge to `main`), custom domains, multi-environment, Railway-hosted Postgres/Redis, backups beyond the Dokploy note.

## Evidence
- Repo: `backend/cmd/api/main.go` serves `GET /healthz`; CORS middleware with `FRONTEND_ORIGIN` shipped in daily PR #17 (2026-09-24), which also made `ENCRYPTION_SECRET_KEY` and a 32-byte `JWT_SECRET` boot requirements (`backend/.env.example` lines 55–67); `frontend/nuxt.config.ts` `ssr: false`; `backend/docker-compose.yml` is dev-only; no Dockerfile exists in either deployable; `.github/workflows/ci.yml` has `backend-unit`, `backend-integration`, `frontend`, `harness-tooling`.
- Spec: 1st-thinking §2.1 (cron and worker in-process — the host must stay awake), §4 (`queue:webpush:delay` persistent), §8; backend spec §9 (Railway checklist, env list); frontend design §5 (JWT in `Authorization`, no cookies — so no CORS credentials flag).
- Free-tier survey (owner conversation 2026-09-24, full text on local branch `claude/free-vps-deployment-91d777`, commit e5ac6f0, idea "Deploy on Railway Free with Supabase and Upstash"): Railway Free $1/month usage, no card, awake by default (https://railway.com/pricing, https://docs.railway.com/reference/app-sleeping); Supabase free 500 MB, pauses after 7 idle days, IPv4 via pooler; Upstash Redis free 500K commands/month (worker poll ≈ 86K); rejected Render (sleeps, in-memory KV), Cloud Run (no CPU between requests), Oracle/Koyeb/Northflank/Fly (card required).
- Dokploy: self-hosted PaaS on Docker + Traefik; deploys Dockerfile, Nixpacks or Compose apps; managed Postgres/Redis with volumes; scheduled DB backups to S3; GitHub webhook redeploys — https://docs.dokploy.com
