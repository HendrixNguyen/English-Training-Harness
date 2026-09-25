# Deploying

Both deployables build into provider-neutral images — `backend/Dockerfile` and `frontend/Dockerfile` — behind one env contract (`deploy/.env.example`). Two targets deploy from the same contract: the free managed split today (Railway / Supabase / Upstash / Cloudflare Pages), or the owner's Dokploy server later, running the whole stack from `deploy/compose.yml`. Continuous delivery (registry push + deploy webhook) is parked in `harness/BACKLOG.md` until the Dokploy server exists. Agents own everything in this repo; accounts, secrets, the Google OAuth console and DNS are the owner's — see *Owner checklist* below.

## Environment

| Variable | Required | Where it comes from | Railway split | Dokploy compose |
| --- | --- | --- | --- | --- |
| `DATABASE_URL` | yes | Postgres connection string | Supabase session-mode pooler URL | built by compose from `POSTGRES_*` |
| `REDIS_URL` | yes | Redis connection string (`store.NewRedis` uses `redis.ParseURL`, TLS accepted) | Upstash `rediss://…` | compose `redis://redis:6379/0` |
| `PORT` | no | listen port | injected by Railway | compose `8080` |
| `GIN_MODE` | no | Gin mode | `release` | `release` |
| `GOMEMLIMIT` | no | Go heap ceiling | `64MiB` on Railway Free | `128MiB` default in compose |
| `JWT_SECRET` | yes | HS256 session key, ≥ 32 bytes — `openssl rand -base64 32` | set on the service | set in `deploy/.env` |
| `ENCRYPTION_SECRET_KEY` | yes | AES-256-GCM key, 64 hex chars — `openssl rand -hex 32`; losing it makes every stored Google refresh token unreadable and every user re-consents | set on the service | set in `deploy/.env` |
| `GOOGLE_CLIENT_ID` | yes | Google Cloud Console OAuth client | set on the service | set in `deploy/.env` |
| `GOOGLE_CLIENT_SECRET` | yes | Google Cloud Console OAuth client | set on the service | set in `deploy/.env` |
| `GEMINI_API_KEY` | optional | Google AI Studio | set if used; missing → its routes answer 503 | set if used |
| `OPENAI_API_KEY` | optional | OpenAI dashboard | set if used; missing → its routes answer 503 | set if used |
| `DEEPSEEK_API_KEY` | optional | DeepSeek dashboard | set if used; missing → its routes answer 503 | set if used |
| `VAPID_PUBLIC_KEY` | optional | `npx web-push generate-vapid-keys` | unset → reminder worker does not start | unset → reminder worker does not start |
| `VAPID_PRIVATE_KEY` | optional | `npx web-push generate-vapid-keys` | unset → reminder worker does not start | unset → reminder worker does not start |
| `VAPID_SUBJECT` | optional | a `mailto:`/`https:` contact | set if VAPID keys are set | set if VAPID keys are set |
| `FRONTEND_ORIGIN` | yes | exact PWA origin(s), comma-separated, never `*` | the Cloudflare Pages domain | the web domain |
| `NUXT_PUBLIC_API_BASE` | yes (build-time) | the API's public URL | the Railway domain | the api domain |
| `NUXT_PUBLIC_GOOGLE_CLIENT_ID` | optional (build-time) | Google Cloud Console OAuth client | set if used | set if used |
| `NUXT_PUBLIC_VAPID_PUBLIC_KEY` | optional (build-time) | must be the same string as `VAPID_PUBLIC_KEY` | set if VAPID is used | set if VAPID is used |

The three `NUXT_PUBLIC_*` values are baked into the static site at build time (Nuxt `runtimeConfig.public`); changing one means a rebuild — a Cloudflare Pages env var change or a new `frontend/Dockerfile` build-arg, never a runtime change. State once: the public VAPID key must be the same string on both sides.

## Target A — Railway Free today (no card)

**API on Railway** — new service from the GitHub repo, **Root Directory `backend`**, builder **Dockerfile**, healthcheck path `/healthz`, **App Sleeping off** (the cron worker and reminder queue run in-process — a sleeping service never decays plants or fires reminders), `GOMEMLIMIT=64MiB`, plus the API rows of the table above. Budget: Railway Free is $1/month of usage with no card; when it is spent Railway shows "workloads stopped" and the API is down until the month resets or the owner upgrades to Hobby ($5); logs are kept 3 days.

**Postgres on Supabase** — free project; use the **session-mode pooler** connection string (`postgresql://postgres.<ref>:<pw>@aws-0-<region>.pooler.supabase.com:5432/postgres`, port **5432** not 6543) with `?sslmode=require` — Railway egress is IPv4-only and the direct `db.<ref>.supabase.co` host is IPv6-only on the free tier. The API runs migrations at boot. Supabase pauses a free project after 7 idle days; the daily cron keeps it awake.

**Redis on Upstash** — free database, copy the **`rediss://`** URL (TLS) into `REDIS_URL`. Free tier is 500K commands/month; the reminder worker's poll is ≈86K/month.

**PWA on Cloudflare Pages** — project from the repo, **Root directory `frontend`**, build command `npx nuxi generate`, output directory `.output/public`, Node 20, env vars `NUXT_PUBLIC_API_BASE=https://<railway-domain>`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY`. Pages serves the SPA fallback and hashed-asset caching by itself and serves `sw.js` with `max-age=0, must-revalidate` by default, so no `_headers` file is needed (the `Caddyfile` rules exist for the image, not for Pages).

**Wiring** — set `FRONTEND_ORIGIN=https://<pages-domain>` on the Railway service; the Google OAuth client's Authorized JavaScript origin is the Pages origin and its Authorized redirect URI is `https://<pages-domain>/login` (`frontend/pages/login.vue` uses `window.location.origin + '/login'`).

## Target B — Dokploy later (deploy only; CD is parked)

Install Dokploy on the server (`curl -sSL https://dokploy.com/install.sh | sh`, the official one-liner — Docker + Traefik). Create a **Compose** application from this repo with compose path `deploy/compose.yml` and paste the contents of `deploy/.env` as its environment. Public HTTPS via Traefik + Let's Encrypt on ports 80/443, or a Cloudflare Tunnel when the box is behind NAT (Google OAuth and Web Push refuse plain HTTP). A domain each for `web` (port 80) and `api` (port 8080). Enable Dokploy's scheduled Postgres backup to an S3-compatible bucket — the `postgres_data` volume is the only copy otherwise. `FRONTEND_ORIGIN` is the web domain; `NUXT_PUBLIC_API_BASE` is the api domain.

Continuous delivery (registry push + deploy webhook on merge to `main`) is parked in `harness/BACKLOG.md` until this target is live — until then, deploys here are manual.

## Run it locally

All from the worktree root. Docker Desktop must be running. Ports `8080`/`6379` may be held by something else on the machine, so use a scratch env and a unique project name:

```sh
cat > deploy/.env <<'ENV'
POSTGRES_PASSWORD=verify-only-<random>
JWT_SECRET=<openssl rand -base64 32>
ENCRYPTION_SECRET_KEY=<openssl rand -hex 32>
GOOGLE_CLIENT_ID=verify-only.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=verify-only
FRONTEND_ORIGIN=http://localhost:18081
NUXT_PUBLIC_API_BASE=http://localhost:18080
API_PORT=18080
WEB_PORT=18081
ENV

export COMPOSE_PROJECT_NAME=aelp-deploy-verify
docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 240
docker compose -f deploy/compose.yml ps
deploy/smoke-api.sh http://127.0.0.1:18080 http://localhost:18081
deploy/smoke-web.sh http://127.0.0.1:18081
docker compose -f deploy/compose.yml down -v
unset COMPOSE_PROJECT_NAME
rm -f deploy/.env
```

`deploy/.env` is gitignored — never commit it. If this repo is checked out as a git worktree, run these commands from the worktree root, not the main checkout (AGENTS.md).

## Smoke check

Run after every deploy:

- `deploy/smoke-api.sh <api-base-url> <frontend-origin>` — `/healthz` is `200` with `{"status":"ok"}` (Postgres and Redis both reachable); `POST /api/v1/auth/google` with an empty body is `400 invalid_request` — the API has **no** OAuth redirect endpoint of its own, the consent URL is built by the PWA, so this only proves the route exists; a CORS preflight from `<frontend-origin>` is `204` and echoes that origin in `Access-Control-Allow-Origin`; a preflight from a foreign origin is `403`.
- `deploy/smoke-web.sh <web-base-url>` — `/` is `200`; a deep path (e.g. `/learn/abc`) is `200` via the SPA fallback; `sw.js` and `manifest.webmanifest` are `200` with `Cache-Control: no-cache`; a hashed `/_nuxt/*.js` asset is `200` with `Cache-Control: public, max-age=31536000, immutable`.

## Owner checklist

- [ ] Create the Supabase project; copy the session-mode pooler connection string.
- [ ] Create the Upstash Redis database; copy the `rediss://` URL.
- [ ] Generate secrets — `openssl rand -base64 32` for `JWT_SECRET`, `openssl rand -hex 32` for `ENCRYPTION_SECRET_KEY`, `npx web-push generate-vapid-keys` for the VAPID pair — store them in a password manager, never in the repo.
- [ ] Create the Railway service (root `backend`, builder Dockerfile, App Sleeping off, healthcheck `/healthz`), set every API variable from the table above, note the generated domain.
- [ ] Create the Cloudflare Pages project (root `frontend`, build `npx nuxi generate`, output `.output/public`), set the three `NUXT_PUBLIC_*` values, note the domain.
- [ ] Set `FRONTEND_ORIGIN` on Railway to the Pages origin and redeploy.
- [ ] In Google Cloud Console, add the Pages origin as an authorized JavaScript origin and `https://<pages-domain>/login` as the redirect URI on the OAuth client; add Gemini/OpenAI/DeepSeek keys if any provider is used.
- [ ] Run both smoke scripts against the public URLs, then sign in from a phone, install the PWA, and allow notifications.
- [ ] When the Dokploy server exists: install Dokploy, complete *Target B* above, move DNS.
