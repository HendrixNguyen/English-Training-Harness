---
type: feature
status: planned
source: human
run: 2026-09-25-run-01
priority: high
plan: harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md
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

## Evaluation

**Verdict: select, `priority: high`.** The *Why* is real and the strongest in the queue: every merged feature — Google sign-in, the installable PWA, Web Push, the plant decaying while nobody visits — is invisible until a learner can reach a public HTTPS origin, and the codebase has no Dockerfile in either deployable (checked 2026-09-25: `backend/` has only the dev-only `docker-compose.yml`; `frontend/` has none; CI has `backend-unit`, `backend-integration`, `frontend`, `harness-tooling` and no image build). The owner's hosting decision (Railway Free + Supabase + Upstash + Cloudflare Pages now, own Dokploy server later) is settled and is not re-litigated here; the deliverable stays provider-neutral so moving hosts is a config change plus the runbook's second section.

**Achievable in one plan?** Yes, in roughly half a day, because almost nothing the plan needs is missing from the code:
- `backend/cmd/api/main.go` serves `GET /healthz` (`internal/health`: 200 when Postgres and Redis ping, 503 otherwise), listens on `":"+cfg.Port` with `PORT` read by `config.Load`, embeds migrations (`internal/store/migrations.go` `//go:embed`) and tzdata (`main.go` asserts `Asia/Ho_Chi_Minh` loads) — so a static `CGO_ENABLED=0` binary is self-contained.
- `store.NewRedis` uses `redis.ParseURL`, which accepts Upstash's `rediss://` (TLS) URLs; pgx accepts `sslmode=require` for the Supabase pooler.
- `config.Load` already refuses to boot without `JWT_SECRET` ≥32 bytes and a 64-hex `ENCRYPTION_SECRET_KEY`, and `middleware.CORS` already reads `FRONTEND_ORIGIN` (daily PR #17) — the env table in the runbook documents what exists rather than inventing a contract.
- `frontend/nuxt.config.ts` is `ssr: false` with `runtimeConfig.public` for the three `NUXT_PUBLIC_*` values (`frontend/.env.example`), so `nuxi generate` emits a static `.output/public` that any file server can host; the `injectManifest` worker is emitted as `sw.js`, which is what the `Cache-Control: no-cache` rule must target.

**Dependencies that do not exist yet:** none in the repo. Outside it, everything is the owner's — accounts, secrets, the Google OAuth console entries, DNS. The plan therefore ends with an explicit *Owner checklist* the executor does not perform, and its `## Verification` is provable without any cloud account: `docker build` of both images, `docker compose -f deploy/compose.yml config`, booting the compose stack locally under a unique `COMPOSE_PROJECT_NAME` on non-default host ports and curling `/healthz`, the OAuth redirect and a CORS preflight against it, plus the CI job.

**Narrowed to keep it finishable today:**
- The backend's final image is `alpine` (with `ca-certificates` and busybox `wget` for `HEALTHCHECK`) rather than `distroless/static` — distroless has no shell or HTTP client, so a Docker `HEALTHCHECK` there would need a new health-probe mode in the Go binary (app code). The idea offered either; alpine is the one that needs no app change.
- The web image is Caddy (one `Caddyfile`, SPA fallback and cache headers in six lines) rather than nginx.
- The CI `docker-images` job builds both images, boots each once (the API must exit with `config: DATABASE_URL is required`, proving the binary runs; the web image must serve `/` and `/sw.js` with the right `Cache-Control`) and runs `docker compose -f deploy/compose.yml config` — it does **not** boot the full compose stack in CI (that is the local verification step) and pushes nothing to a registry (CD is parked in `harness/BACKLOG.md`).
- Docs are updated in place (README "Hosting" row, CLAUDE.md "Planned architecture", AGENTS.md "Deployment" paragraph, CODEMAP `deploy/` entry, backend spec §9 addendum); no new doc beyond `deploy/README.md`.
- The post-deploy smoke check is written into the runbook as three `curl` commands the owner (or the review routine, once the URL exists) runs; wiring it into a routine is not part of this plan.

**Priority rationale:** `high` — the idea file already carries it (owner-set), and by the role's rule it blocks users: no user can use anything until this lands. Auto-approve applies (`type: feature`, `priority: high`).
