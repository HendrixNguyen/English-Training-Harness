---
idea: harness/ideas/2026-09-25-run-01/containerised-deploy-dockerfiles-production-compose-runbook-.md
status: done
priority: high
merged: false
branch: harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-
worktree: /Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.worktrees/containerised-deploy-dockerfiles-production-compose-runbook-
---
# Containerised deploy: Dockerfiles, production compose, runbook and CI image build — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/containerised-deploy-dockerfiles-production-compose-runbook-.md`
**Goal:** Make both deployables build into provider-neutral images (`backend/Dockerfile`, `frontend/Dockerfile`), add a production `deploy/compose.yml` with two smoke scripts and a runbook (`deploy/README.md`: env table, Railway-Free-today, Dokploy-later, owner checklist), gate the images in CI with a `docker-images` job, and point the docs at it — so the owner can deploy by completing a checklist, with nothing left for agents that needs a cloud account.

**Architecture:** No app code changes. The Go binary is already self-contained (`internal/store/migrations.go` embeds the migrations, `cmd/api/main.go` asserts embedded tzdata, `config.Load` reads `PORT`, `GET /healthz` exists), so the API image is a two-stage `golang:1.25-alpine` → `alpine` build with a busybox-`wget` `HEALTHCHECK`. The PWA is `ssr: false`, so `nuxi generate` produces a static `.output/public` that a Caddy image serves with SPA fallback and the two cache rules the service worker needs. `deploy/compose.yml` wires `api`, `web`, `postgres:16`, `redis:7` (AOF on, named volumes) from `deploy/.env`; it is what Dokploy deploys and what the local verification boots. The three post-deploy curls live in `deploy/smoke-api.sh` / `deploy/smoke-web.sh` so CI, the local verification and the owner run the same checks.

**Tech Stack:** Docker 28 / Compose v2 (`docker compose`), `golang:1.25-alpine`, `alpine:3.21`, `node:20-alpine`, `caddy:2-alpine`, GitHub Actions (`actions/checkout@v7`), `actionlint` (installed at `/opt/homebrew/bin/actionlint`).

**Branch:** new `harness/*` branch in `.worktrees/containerised-deploy-dockerfiles-production-compose-runbook-`, per the execute skill. Run commands from the **worktree root** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n` and each tool's own timeout flag. Never capture `$(ls …)`.

**Host ports on this machine:** `8080` is held by an unrelated `api` process and `6379` by an unrelated Docker container; both must keep running. Every local step therefore uses `API_PORT=18080 WEB_PORT=18081` and a unique `COMPOSE_PROJECT_NAME=aelp-deploy-verify`. The production compose file never publishes Postgres or Redis ports.

**What this plan deliberately does not do** (see the idea's `## Evaluation`): no `distroless` final image (no shell → no `HEALTHCHECK` without new Go code), no nginx, no registry push, no deploy webhook (parked in `harness/BACKLOG.md`), no booting the full compose stack in CI (local verification does that), no changes to any routine. Two idea details were wrong and are corrected here: the API has **no** `GET /api/v1/auth/google` redirect — the consent URL is built by the PWA (`frontend/utils/googleAuth.ts`) and the API only serves `POST /api/v1/auth/google` (`backend/internal/auth/handler.go`), so the smoke check posts an empty body and expects `400 {"error":"invalid_request"}`; and the PWA's OAuth redirect URI is `<origin>/login` (`frontend/pages/login.vue:16`), not `/auth/callback`.

## File structure

| Path | Change |
| --- | --- |
| `backend/.dockerignore` | **new** — keeps `.env`, git metadata and test artefacts out of the build context |
| `backend/Dockerfile` | **new** — two-stage API image, non-root, `EXPOSE 8080`, `HEALTHCHECK` on `/healthz` |
| `frontend/.dockerignore` | **new** — excludes `node_modules`, `.nuxt`, `.output`, `.env*`, test output |
| `frontend/Dockerfile` | **new** — `node:20-alpine` `npm ci && npx nuxi generate` → `caddy:2-alpine` serving `/srv` |
| `frontend/Caddyfile` | **new** — SPA fallback, `sw.js`/manifest `no-cache`, `/_nuxt/*` immutable, `/healthz` |
| `deploy/compose.yml` | **new** — production stack: `postgres`, `redis`, `api`, `web` |
| `deploy/.env.example` | **new** — every variable `compose.yml` interpolates, secrets blank |
| `deploy/smoke-api.sh` | **new** — `/healthz`, `POST /auth/google` 400, CORS preflight allow + deny |
| `deploy/smoke-web.sh` | **new** — `/` 200, deep path 200 (SPA fallback), `sw.js` `no-cache`, `/_nuxt/` immutable |
| `deploy/README.md` | **new** — runbook: env table, Railway Free today, Dokploy later, owner checklist, smoke check |
| `.github/workflows/ci.yml` | `docker-images` job appended |
| `README.md:22` | "Hosting" row names both targets and the runbook |
| `CLAUDE.md:25-27` | "Planned architecture" intro names both targets |
| `AGENTS.md` | "Deployment" paragraph after *Reading the spec*; CI bullet lists `docker-images` |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §9 addendum (three lines) |
| `harness/CODEMAP.md` | new `## Deploy (\`deploy/\`)` section; CI section gains the job |

Nothing under `backend/internal/`, `backend/cmd/` or `frontend/` **source** changes. If a step seems to require an app-code change, stop and record it in `## Failure` — the evaluation established none is needed.

---

## Tasks

### Task 1: Backend image

**Files:**
- Create: `backend/.dockerignore`
- Create: `backend/Dockerfile`

- [ ] **Step 1: Write the check first — it must fail because nothing exists yet**

Run from the worktree root:

```sh
docker build --pull -t aelp-api:local -f backend/Dockerfile backend 2>&1 | tail -3
```
Expected: FAIL — `failed to read dockerfile: open Dockerfile: no such file or directory` (or similar). Do not proceed until you have seen this fail.

- [ ] **Step 2: Write `backend/.dockerignore`**

```
.git
.env
.env.*
!.env.example
*.log
integration.log
docker-compose.yml
```

- [ ] **Step 3: Write `backend/Dockerfile`**

```dockerfile
# syntax=docker/dockerfile:1

# Build stage. The binary is static (CGO_ENABLED=0): migrations are embedded
# (internal/store/migrations.go) and tzdata is embedded (cmd/api/main.go asserts
# it at boot), so nothing else from this stage is needed at runtime.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime stage. alpine rather than distroless so the HEALTHCHECK has an HTTP
# client (busybox wget); ca-certificates is for Google, Gemini, OpenAI, DeepSeek
# and the TLS Postgres/Redis URLs the runbook documents.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
 && adduser -D -H -u 10001 api
COPY --from=build /out/api /usr/local/bin/api
USER api
# config.Load reads PORT; Railway overrides it, compose and Dokploy keep 8080.
ENV PORT=8080
EXPOSE 8080
# GET /healthz is 200 only when Postgres and Redis answer (internal/health), so
# an unhealthy container means a dependency is down, not just the process.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${PORT}/healthz" || exit 1
ENTRYPOINT ["/usr/local/bin/api"]
```

- [ ] **Step 4: Build it**

```sh
docker build --pull -t aelp-api:local -f backend/Dockerfile backend 2>&1 | tail -3
docker image inspect aelp-api:local --format '{{.Config.User}} {{.Config.ExposedPorts}} {{index .Config.Env 0}} {{.Config.Healthcheck.Test}}'
```
Expected: the build ends `naming to docker.io/library/aelp-api:local`; the inspect line prints `api map[8080/tcp:{}] PATH=… [CMD-SHELL wget -qO- "http://127.0.0.1:${PORT}/healthz" || exit 1]` (the first `Env` entry may be `PATH`; what matters is `User` = `api` and `8080/tcp`).

- [ ] **Step 5: Prove the binary runs in the image — it must refuse to boot without its env**

```sh
set +e
out=$(docker run --rm aelp-api:local 2>&1); code=$?
set -e
echo "exit=$code"; echo "$out" | tail -1
```
Expected: `exit=1` and the last line contains `config: DATABASE_URL is required`. That is `config.Load` running inside the image — the proof that the static binary, its user and its entrypoint are right. (If it prints a `tzdata:` or `exec format error` line instead, the build stage is wrong; fix the Dockerfile, do not touch Go code.)

- [ ] **Step 6: Commit**

```sh
git add backend/.dockerignore backend/Dockerfile
git commit -m "deploy: backend Dockerfile (static Go binary on alpine, non-root, healthcheck on /healthz)"
```

### Task 2: Frontend image

**Files:**
- Create: `frontend/.dockerignore`
- Create: `frontend/Caddyfile`
- Create: `frontend/Dockerfile`

- [ ] **Step 1: Write the check first**

```sh
docker build --pull -t aelp-web:local -f frontend/Dockerfile \
  --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend 2>&1 | tail -3
```
Expected: FAIL — no Dockerfile.

- [ ] **Step 2: Write `frontend/.dockerignore`**

```
.git
node_modules
.nuxt
.output
dist
.env
.env.*
!.env.example
test-results
playwright-report
tests
```

- [ ] **Step 3: Write `frontend/Caddyfile`**

```
# Serves the static Nuxt build (`nuxi generate`, ssr: false) from /srv.
:80 {
	root * /srv
	encode gzip zstd

	# Liveness for compose/Dokploy; no file behind it.
	respond /healthz 200

	# The service worker and the manifest must never be served stale, or an
	# installed PWA keeps an old precache list (frontend spec §3).
	@fresh path /sw.js /manifest.webmanifest
	header @fresh Cache-Control "no-cache"

	# Nuxt writes content-hashed assets under /_nuxt/; they are safe forever.
	header /_nuxt/* Cache-Control "public, max-age=31536000, immutable"

	# SPA fallback: every route the router owns (/login, /learn/:id, …) is index.html.
	try_files {path} /index.html
	file_server
}
```

- [ ] **Step 4: Write `frontend/Dockerfile`**

```dockerfile
# syntax=docker/dockerfile:1

# Build stage. The PWA is client-rendered (nuxt.config.ts: ssr: false), so
# `nuxi generate` emits a static site. The three NUXT_PUBLIC_* values are baked
# into that site at build time (Nuxt runtimeConfig.public); pass them as
# --build-arg (compose.yml does) — there is no runtime injection.
FROM node:20-alpine AS build
WORKDIR /src
# The whole tree first: package.json's postinstall runs `nuxi prepare`, which
# needs nuxt.config.ts and the app present.
COPY . .
RUN npm ci
ARG NUXT_PUBLIC_API_BASE=http://localhost:8080
ARG NUXT_PUBLIC_GOOGLE_CLIENT_ID=
ARG NUXT_PUBLIC_VAPID_PUBLIC_KEY=
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE \
    NUXT_PUBLIC_GOOGLE_CLIENT_ID=$NUXT_PUBLIC_GOOGLE_CLIENT_ID \
    NUXT_PUBLIC_VAPID_PUBLIC_KEY=$NUXT_PUBLIC_VAPID_PUBLIC_KEY
RUN npx nuxi generate

# Runtime stage: Caddy serving the static output (Caddyfile has the SPA
# fallback and the cache rules the service worker depends on).
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/.output/public /srv
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:80/healthz || exit 1
```

- [ ] **Step 5: Build it**

```sh
docker build --pull -t aelp-web:local -f frontend/Dockerfile \
  --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend 2>&1 | tail -3
docker run --rm --entrypoint sh aelp-web:local -c 'ls /srv | head -20; echo ---; grep -rl "api.example.test" /srv | head -3'
```
Expected: the build ends `naming to docker.io/library/aelp-web:local`; `/srv` lists `index.html`, `200.html`, `404.html`, `sw.js`, `manifest.webmanifest`, `_nuxt/`, the `pwa-*.png` icons; the grep prints at least one path — the build arg reached the generated site. If the grep prints nothing, `nuxi generate` did not see the env; fix the Dockerfile (the `ENV` lines must precede `RUN npx nuxi generate`).

- [ ] **Step 6: Commit**

```sh
git add frontend/.dockerignore frontend/Caddyfile frontend/Dockerfile
git commit -m "deploy: frontend Dockerfile (nuxi generate into Caddy with SPA fallback and SW cache rules)"
```

### Task 3: Smoke scripts

The same checks CI, the local verification and the owner run. Written before `compose.yml` so the stack has something to pass.

**Files:**
- Create: `deploy/smoke-api.sh`
- Create: `deploy/smoke-web.sh`

- [ ] **Step 1: Write `deploy/smoke-api.sh`**

```sh
#!/bin/sh
# Post-deploy smoke check for the API (deploy/README.md "Smoke check").
#   deploy/smoke-api.sh <api-base-url> <frontend-origin>
# e.g. deploy/smoke-api.sh https://api.example.com https://app.example.com
# Exit 0 only if every check passes. Needs curl.
set -eu
api=${1:?usage: smoke-api.sh <api-base-url> <frontend-origin>}
origin=${2:?usage: smoke-api.sh <api-base-url> <frontend-origin>}
fail=0
check() { # check <label> <expected> <actual>
  if [ "$2" = "$3" ]; then echo "ok   $1: $3"; else echo "FAIL $1: expected '$2', got '$3'"; fail=1; fi
}

# 1. /healthz answers 200 with every dependency ok (internal/health).
body=$(curl -sS --max-time 10 -w '\n%{http_code}' "$api/healthz")
check "healthz status" "200" "$(printf '%s' "$body" | tail -1)"
check "healthz body" "1" "$(printf '%s' "$body" | head -1 | grep -c '"status":"ok"')"

# 2. POST /api/v1/auth/google is routed: an empty body is a 400 invalid_request
#    (the real code exchange can only be driven by a browser through the PWA).
code=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' -X POST \
  -H 'Content-Type: application/json' -d '{}' "$api/api/v1/auth/google")
check "auth/google empty body" "400" "$code"

# 3. CORS: a preflight from FRONTEND_ORIGIN is 204 and echoes the origin …
hdr=$(curl -sS --max-time 10 -i -X OPTIONS "$api/api/v1/auth/google" \
  -H "Origin: $origin" -H 'Access-Control-Request-Method: POST' -o - )
check "preflight status" "204" "$(printf '%s' "$hdr" | head -1 | awk '{print $2}')"
check "preflight allow-origin" "$origin" "$(printf '%s' "$hdr" | tr -d '\r' | awk -F': ' 'tolower($1)=="access-control-allow-origin"{print $2}')"

# 4. … and a preflight from anywhere else is refused (403), so the allow-list is exact.
code=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' -X OPTIONS "$api/api/v1/auth/google" \
  -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: POST')
check "preflight foreign origin" "403" "$code"

exit $fail
```

- [ ] **Step 2: Write `deploy/smoke-web.sh`**

```sh
#!/bin/sh
# Post-deploy smoke check for the PWA (deploy/README.md "Smoke check").
#   deploy/smoke-web.sh <web-base-url>
# Exit 0 only if every check passes. Needs curl.
set -eu
web=${1:?usage: smoke-web.sh <web-base-url>}
fail=0
check() { if [ "$2" = "$3" ]; then echo "ok   $1: $3"; else echo "FAIL $1: expected '$2', got '$3'"; fail=1; fi; }
status() { curl -sS --max-time 10 -o /dev/null -w '%{http_code}' "$1"; }
cache_control() { curl -sS --max-time 10 -I "$1" | tr -d '\r' | awk -F': ' 'tolower($1)=="cache-control"{print $2}'; }

check "index" "200" "$(status "$web/")"
check "spa fallback (/learn/abc)" "200" "$(status "$web/learn/abc")"
check "sw.js present" "200" "$(status "$web/sw.js")"
check "sw.js cache-control" "no-cache" "$(cache_control "$web/sw.js")"
check "manifest cache-control" "no-cache" "$(cache_control "$web/manifest.webmanifest")"
# One hashed asset: the first /_nuxt/ script index.html references.
asset=$(curl -sS --max-time 10 "$web/" | grep -o '/_nuxt/[^"]*\.js' | head -1)
check "hashed asset found" "1" "$([ -n "$asset" ] && echo 1 || echo 0)"
[ -n "$asset" ] && check "asset cache-control" "public, max-age=31536000, immutable" "$(cache_control "$web$asset")"

exit $fail
```

- [ ] **Step 3: Make them executable and prove they fail against nothing**

```sh
chmod +x deploy/smoke-api.sh deploy/smoke-web.sh
sh -n deploy/smoke-api.sh && sh -n deploy/smoke-web.sh && echo "syntax ok"
deploy/smoke-web.sh http://127.0.0.1:18081 ; echo "exit=$?"
```
Expected: `syntax ok`; the web script prints `curl: (7) Failed to connect …` lines and `FAIL …` lines and `exit=1` (nothing is listening on 18081 yet). If it exits 0 here, the script is not checking anything — fix it.

- [ ] **Step 4: Run the web script against the Task 2 image**

```sh
docker run -d --rm --name aelp-web-smoke -p 18081:80 aelp-web:local
sleep 2; deploy/smoke-web.sh http://127.0.0.1:18081 ; echo "exit=$?"
docker stop aelp-web-smoke
```
Expected: every line `ok …`, `exit=0`. The API script is exercised in Task 4 (it needs the stack).

- [ ] **Step 5: Commit**

```sh
git add deploy/smoke-api.sh deploy/smoke-web.sh
git commit -m "deploy: smoke scripts for the API (healthz, auth route, CORS allow/deny) and the PWA (SPA fallback, SW cache headers)"
```

### Task 4: Production compose

**Files:**
- Create: `deploy/.env.example`
- Create: `deploy/compose.yml`

- [ ] **Step 1: Write `deploy/.env.example`**

```sh
# Copy to deploy/.env (gitignored) and fill every blank. `docker compose -f
# deploy/compose.yml` reads it from this directory; `${VAR:?}` entries in
# compose.yml refuse to start with a blank. deploy/README.md has the table of
# where each value comes from.

# Postgres (the compose-managed one; on Railway/Supabase this block is unused
# and DATABASE_URL is set on the service instead).
POSTGRES_USER=english
POSTGRES_PASSWORD=
POSTGRES_DB=english

# API secrets — backend/.env.example documents each.
JWT_SECRET=                 # openssl rand -base64 32  (>= 32 bytes)
ENCRYPTION_SECRET_KEY=      # openssl rand -hex 32     (exactly 64 hex chars)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=

# AI providers: leave blank to skip a provider (its routes answer 503).
GEMINI_API_KEY=
OPENAI_API_KEY=
DEEPSEEK_API_KEY=

# Web Push: `npx web-push generate-vapid-keys`. The public key is ALSO baked
# into the PWA (NUXT_PUBLIC_VAPID_PUBLIC_KEY below) — keep them identical.
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_SUBJECT=mailto:admin@example.com

# Exact browser origin(s) of the PWA, comma-separated, never "*".
FRONTEND_ORIGIN=https://app.example.com

# Baked into the PWA at image build time (frontend/Dockerfile build args).
NUXT_PUBLIC_API_BASE=https://api.example.com
NUXT_PUBLIC_GOOGLE_CLIENT_ID=
NUXT_PUBLIC_VAPID_PUBLIC_KEY=

# Host ports published by compose (Dokploy's Traefik targets the container
# ports directly; these matter for a plain `docker compose up`).
API_PORT=8080
WEB_PORT=80
# Go heap ceiling (runtime GOMEMLIMIT); the runbook sets 64MiB on Railway Free.
GOMEMLIMIT=128MiB
```

Note for the executor: Compose v2 strips an unquoted ` # comment` after whitespace in `.env` values, but to leave no doubt write the two generator hints on their **own** comment lines above `JWT_SECRET=` / `ENCRYPTION_SECRET_KEY=`, not trailing on the same line as shown above.

- [ ] **Step 2: Write `deploy/compose.yml`**

```yaml
# Production stack (deploy/README.md). Dokploy deploys this file as a Compose
# app; on the Railway/Supabase/Upstash/Cloudflare split it is documentation of
# the same env contract. Values come from deploy/.env (copy deploy/.env.example).
# Postgres and Redis are never published to the host.
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-english}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD in deploy/.env}
      POSTGRES_DB: ${POSTGRES_DB:-english}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 12
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    # AOF: queue:webpush:delay (spec §4) is a persistent ZSET and must survive
    # a restart; everything else in Redis has a TTL and may be lost.
    command: ["redis-server", "--appendonly", "yes"]
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 12
    restart: unless-stopped

  api:
    build:
      context: ../backend
    image: aelp-api:local
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER:-english}:${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD in deploy/.env}@postgres:5432/${POSTGRES_DB:-english}?sslmode=disable
      REDIS_URL: redis://redis:6379/0
      PORT: "8080"
      GIN_MODE: release
      GOMEMLIMIT: ${GOMEMLIMIT:-128MiB}
      JWT_SECRET: ${JWT_SECRET:?set JWT_SECRET in deploy/.env (openssl rand -base64 32)}
      ENCRYPTION_SECRET_KEY: ${ENCRYPTION_SECRET_KEY:?set ENCRYPTION_SECRET_KEY in deploy/.env (openssl rand -hex 32)}
      GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID:?set GOOGLE_CLIENT_ID in deploy/.env}
      GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET:?set GOOGLE_CLIENT_SECRET in deploy/.env}
      GEMINI_API_KEY: ${GEMINI_API_KEY:-}
      OPENAI_API_KEY: ${OPENAI_API_KEY:-}
      DEEPSEEK_API_KEY: ${DEEPSEEK_API_KEY:-}
      VAPID_PUBLIC_KEY: ${VAPID_PUBLIC_KEY:-}
      VAPID_PRIVATE_KEY: ${VAPID_PRIVATE_KEY:-}
      VAPID_SUBJECT: ${VAPID_SUBJECT:-mailto:admin@example.com}
      FRONTEND_ORIGIN: ${FRONTEND_ORIGIN:?set FRONTEND_ORIGIN in deploy/.env}
    ports:
      - "${API_PORT:-8080}:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  web:
    build:
      context: ../frontend
      args:
        NUXT_PUBLIC_API_BASE: ${NUXT_PUBLIC_API_BASE:?set NUXT_PUBLIC_API_BASE in deploy/.env}
        NUXT_PUBLIC_GOOGLE_CLIENT_ID: ${NUXT_PUBLIC_GOOGLE_CLIENT_ID:-}
        NUXT_PUBLIC_VAPID_PUBLIC_KEY: ${NUXT_PUBLIC_VAPID_PUBLIC_KEY:-}
    image: aelp-web:local
    ports:
      - "${WEB_PORT:-80}:80"
    depends_on:
      - api
    restart: unless-stopped

# Named so `docker compose down` keeps the data; `down -v` discards it.
volumes:
  postgres_data:
  redis_data:
```

- [ ] **Step 3: `config` must refuse without the required values, then accept with them**

```sh
set +e
docker compose -f deploy/compose.yml config -q 2>&1 | head -2; echo "exit=${PIPESTATUS[0]}"
set -e
```
Expected: an error naming `POSTGRES_PASSWORD` (`required variable POSTGRES_PASSWORD is missing a value: set POSTGRES_PASSWORD in deploy/.env`) and a non-zero exit — the `:?` guards work. (In zsh use `$pipestatus[1]`.) Then write a scratch env and re-run:

```sh
cat > deploy/.env <<EOF
POSTGRES_PASSWORD=verify-only-$(openssl rand -hex 4)
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_SECRET_KEY=$(openssl rand -hex 32)
GOOGLE_CLIENT_ID=verify-only.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=verify-only
FRONTEND_ORIGIN=http://localhost:18081
NUXT_PUBLIC_API_BASE=http://localhost:18080
API_PORT=18080
WEB_PORT=18081
EOF
docker compose -f deploy/compose.yml config -q && echo "config ok"
docker compose -f deploy/compose.yml config | grep -n 'appendonly\|redis_data\|18080:8080\|18081:80\|FRONTEND_ORIGIN'
```
Expected: `config ok`, and the grep shows `--appendonly`, `redis_data`, the two published ports and `FRONTEND_ORIGIN: http://localhost:18081`. `deploy/.env` is gitignored (`.env*`); `git status --short deploy/` must not list it.

- [ ] **Step 4: Boot the stack and run both smoke scripts**

```sh
export COMPOSE_PROJECT_NAME=aelp-deploy-verify
docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 240
docker compose -f deploy/compose.yml ps
docker compose -f deploy/compose.yml logs api | grep -n 'migrations applied\|cors: allowing\|listening on'
deploy/smoke-api.sh http://127.0.0.1:18080 http://localhost:18081 ; echo "api exit=$?"
deploy/smoke-web.sh http://127.0.0.1:18081 ; echo "web exit=$?"
```
Expected: `ps` shows all four services `running` and `api`/`postgres`/`redis` `(healthy)` (`web` becomes healthy after its 5 s start period); the api log has `migrations applied: […]`, `cors: allowing [http://localhost:18081]`, `listening on [::]:8080`; both scripts print only `ok` lines and `exit=0`. `--wait` failing means a healthcheck never went green — read `docker compose -f deploy/compose.yml logs <service>`; do not lengthen the timeout to hide it.

- [ ] **Step 5: Tear down and prove nothing is left**

```sh
docker compose -f deploy/compose.yml down -v
unset COMPOSE_PROJECT_NAME
docker ps --format '{{.Names}}' | grep -c aelp-deploy-verify || true
```
Expected: `0`. Keep `deploy/.env` only until `## Verification` is done, then delete it (step g of the execute skill).

- [ ] **Step 6: Commit**

```sh
git add deploy/.env.example deploy/compose.yml
git status --short deploy/   # must NOT list deploy/.env
git commit -m "deploy: production compose (api, web, postgres 16, redis 7 with AOF) and its .env.example"
```

### Task 5: CI `docker-images` job

**Files:**
- Modify: `.github/workflows/ci.yml` (append a job after `frontend`)

- [ ] **Step 1: Append the job**

Add at the end of `jobs:` in `.github/workflows/ci.yml`:

```yaml
  docker-images:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v7
      - name: Build API image
        run: docker build --pull -t aelp-api:ci -f backend/Dockerfile backend
      - name: API image runs (config.Load refuses to boot without env)
        run: |
          set +e
          out=$(docker run --rm aelp-api:ci 2>&1); code=$?
          set -e
          echo "$out" | tail -3
          if [ "$code" -eq 0 ] || ! echo "$out" | grep -q 'config: DATABASE_URL is required'; then
            echo "::error::expected the API to exit non-zero with 'config: DATABASE_URL is required' (exit $code)"
            exit 1
          fi
      - name: Build web image
        run: |
          docker build --pull -t aelp-web:ci -f frontend/Dockerfile \
            --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend
      - name: Web image serves the PWA with the right cache headers
        run: |
          docker run -d --rm --name web -p 18081:80 aelp-web:ci
          for i in $(seq 1 20); do curl -sf --max-time 2 http://127.0.0.1:18081/healthz >/dev/null && break; sleep 1; done
          docker run --rm --entrypoint sh aelp-web:ci -c 'grep -rql "api.example.test" /srv' \
            || { echo "::error::NUXT_PUBLIC_API_BASE build arg did not reach the generated site"; exit 1; }
          deploy/smoke-web.sh http://127.0.0.1:18081
          docker stop web
      - name: Production compose file is valid
        run: |
          cat > "$RUNNER_TEMP/deploy.env" <<EOF
          POSTGRES_PASSWORD=ci-only
          JWT_SECRET=$(openssl rand -base64 32)
          ENCRYPTION_SECRET_KEY=$(openssl rand -hex 32)
          GOOGLE_CLIENT_ID=ci-only
          GOOGLE_CLIENT_SECRET=ci-only
          FRONTEND_ORIGIN=https://app.example.test
          NUXT_PUBLIC_API_BASE=https://api.example.test
          EOF
          docker compose --env-file "$RUNNER_TEMP/deploy.env" -f deploy/compose.yml config -q
          # And the guards hold: without the env file, config must fail.
          if docker compose -f deploy/compose.yml config -q 2>/dev/null; then
            echo "::error::compose config succeeded with no env — the :? guards are missing"
            exit 1
          fi
```

- [ ] **Step 2: Lint the workflow**

```sh
actionlint .github/workflows/ci.yml && echo "actionlint ok"
python3 -c 'import yaml,sys; d=yaml.safe_load(open(".github/workflows/ci.yml")); print(sorted(d["jobs"]))'
```
Expected: `actionlint ok` and `['backend-integration', 'backend-unit', 'docker-images', 'frontend', 'harness-tooling']`. (If `yaml` is missing, `ruby -ryaml -e 'puts YAML.load_file(".github/workflows/ci.yml")["jobs"].keys.sort'`.)

- [ ] **Step 3: Run the job's `run:` blocks locally, exactly as written**

You cannot watch a CI run before pushing, so run each block from the worktree root in order (the two `docker build` lines, the API-run check, the web check with `deploy/smoke-web.sh`, and the compose block with `RUNNER_TEMP=$(mktemp -d)` exported first). Expected: every block exits 0; the final `if` prints nothing (config without env fails, as required). `docker stop web` must leave `docker ps` free of a `web` container.

- [ ] **Step 4: Commit**

```sh
git add .github/workflows/ci.yml
git commit -m "ci: docker-images job builds both images, boots each once, validates deploy/compose.yml"
```

### Task 6: Runbook `deploy/README.md`

**Files:**
- Create: `deploy/README.md`

- [ ] **Step 1: Write the runbook**

Write `deploy/README.md` with exactly these sections, in this order. Prose is the executor's, but every fact listed under a section must appear in it; these facts were checked against the code on 2026-09-25.

1. **`# Deploying`** — one paragraph: two images (`backend/Dockerfile`, `frontend/Dockerfile`), one env contract, two targets (Railway Free split today, Dokploy later); CD is parked (`harness/BACKLOG.md`). Agents own the repo; accounts, secrets, OAuth console and DNS are the owner's (see *Owner checklist*).

2. **`## Environment`** — one table, columns `Variable | Required | Where it comes from | Railway split | Dokploy compose`. Rows: `DATABASE_URL` (Supabase session-mode pooler URL on Railway / compose builds it from `POSTGRES_*`), `REDIS_URL` (Upstash `rediss://…` — `store.NewRedis` uses `redis.ParseURL`, TLS accepted / compose `redis://redis:6379/0`), `PORT` (Railway injects it; compose 8080), `GIN_MODE=release`, `GOMEMLIMIT` (64MiB on Railway Free, 128MiB default in compose), `JWT_SECRET` (≥32 bytes, `openssl rand -base64 32`), `ENCRYPTION_SECRET_KEY` (64 hex, `openssl rand -hex 32`; losing it → every stored Google refresh token unreadable, users re-consent), `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` (both required at boot), `GEMINI_API_KEY` / `OPENAI_API_KEY` / `DEEPSEEK_API_KEY` (optional; a missing provider's routes answer 503), `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` (`npx web-push generate-vapid-keys`; both keys unset → reminder worker does not start), `FRONTEND_ORIGIN` (exact PWA origin(s), comma-separated, never `*`), and the three PWA build-time values `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY` (baked at build; changing one is a rebuild — Cloudflare Pages env vars / compose build args). State once: the public VAPID key must be the same string on both sides.

3. **`## Target A — Railway Free today (no card)`** — subsections:
   - *API on Railway*: new service from the GitHub repo, **Root Directory `backend`**, builder **Dockerfile**, healthcheck path `/healthz`, **App Sleeping off** (the cron worker and reminder queue run in-process — a sleeping service never decays plants or fires reminders), `GOMEMLIMIT=64MiB`, the env table's API rows. Budget: Railway Free is $1/month of usage with no card; when it is spent Railway shows "workloads stopped" and the API is down until the month resets or the owner upgrades to Hobby ($5); logs are kept 3 days.
   - *Postgres on Supabase*: free project; use the **session-mode pooler** connection string (`postgresql://postgres.<ref>:<pw>@aws-0-<region>.pooler.supabase.com:5432/postgres`, port **5432** not 6543) with `?sslmode=require` — Railway egress is IPv4-only and the direct `db.<ref>.supabase.co` host is IPv6-only on the free tier. The API runs migrations at boot. Supabase pauses a free project after 7 idle days; the daily cron keeps it awake.
   - *Redis on Upstash*: free database, copy the **`rediss://`** URL (TLS) into `REDIS_URL`. Free tier 500K commands/month; the reminder worker's poll is ≈86K/month.
   - *PWA on Cloudflare Pages*: project from the repo, **Root directory `frontend`**, build command `npx nuxi generate`, output directory `.output/public`, Node 20, env vars `NUXT_PUBLIC_API_BASE=https://<railway-domain>`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY`. Pages serves the SPA fallback and hashed-asset caching by itself and serves `sw.js` with `max-age=0, must-revalidate` by default, so no `_headers` file is needed (the `Caddyfile` rules exist for the image, not for Pages).
   - *Wiring*: set `FRONTEND_ORIGIN=https://<pages-domain>` on the Railway service; the Google OAuth client's Authorized JavaScript origin is the Pages origin and its Authorized redirect URI is `https://<pages-domain>/login` (`frontend/pages/login.vue` uses `window.location.origin + '/login'`).

4. **`## Target B — Dokploy later (deploy only; CD is parked)`** — install Dokploy on the server (`curl -sSL https://dokploy.com/install.sh | sh`, official one-liner, Docker + Traefik); create a **Compose** application from this repo with compose path `deploy/compose.yml` and paste `deploy/.env` contents as its environment; public HTTPS via Traefik + Let's Encrypt on ports 80/443 **or** a Cloudflare Tunnel when the box is behind NAT (Google OAuth and Web Push refuse plain HTTP); a domain each for `web` (port 80) and `api` (port 8080); enable Dokploy's scheduled Postgres backup to an S3-compatible bucket (the `postgres_data` volume is the only copy otherwise). `FRONTEND_ORIGIN` = the web domain; `NUXT_PUBLIC_API_BASE` = the api domain.

5. **`## Run it locally`** — the Task 4 step 3–5 commands (scratch `deploy/.env`, `COMPOSE_PROJECT_NAME`, `API_PORT=18080 WEB_PORT=18081`, `up -d --build --wait`, both smoke scripts, `down -v`), plus the worktree caveat from AGENTS.md.

6. **`## Smoke check`** — `deploy/smoke-api.sh <api-url> <pwa-origin>` and `deploy/smoke-web.sh <pwa-url>`; what each line checks; run after every deploy. Note explicitly that the API has no OAuth redirect endpoint — the consent URL is built by the PWA — so the API check posts an empty body to `POST /api/v1/auth/google` and expects `400 invalid_request`.

7. **`## Owner checklist`** — `- [ ]` items in dependency order. Agents do none of these: (1) create the Supabase project, copy the pooler URL; (2) create the Upstash Redis, copy the `rediss://` URL; (3) `openssl rand -base64 32` → `JWT_SECRET`, `openssl rand -hex 32` → `ENCRYPTION_SECRET_KEY`, `npx web-push generate-vapid-keys` → VAPID pair — store them in a password manager, never in the repo; (4) create the Railway service (root `backend`, Dockerfile, sleeping off, healthcheck `/healthz`), set every API variable, note the generated domain; (5) create the Cloudflare Pages project (root `frontend`, `npx nuxi generate`, `.output/public`), set the three `NUXT_PUBLIC_*` values, note the domain; (6) set `FRONTEND_ORIGIN` on Railway to the Pages origin and redeploy; (7) in Google Cloud Console, add the Pages origin as an authorized JavaScript origin and `https://<pages-domain>/login` as the redirect URI on the OAuth client, and add Gemini/OpenAI/DeepSeek keys if any; (8) run both smoke scripts against the public URLs, then sign in from a phone, install the PWA, allow notifications; (9) when the Dokploy server exists: install Dokploy, complete *Target B*, move DNS.

- [ ] **Step 2: Check every code reference in the runbook exists**

```sh
for f in backend/Dockerfile frontend/Dockerfile deploy/compose.yml deploy/.env.example deploy/smoke-api.sh deploy/smoke-web.sh frontend/pages/login.vue backend/internal/store/redis.go backend/.env.example harness/BACKLOG.md; do [ -e "$f" ] && echo "ok $f" || echo "MISSING $f"; done
grep -c '\- \[ \]' deploy/README.md
grep -n 'auth/callback' deploy/README.md || echo "no stale /auth/callback: ok"
```
Expected: every `ok`; the checkbox count is ≥ 9; the last line prints `no stale /auth/callback: ok`.

- [ ] **Step 3: Commit**

```sh
git add deploy/README.md
git commit -m "deploy: runbook — env contract, Railway Free split, Dokploy, owner checklist, smoke check"
```

### Task 7: Docs

**Files:**
- Modify: `README.md:22`
- Modify: `CLAUDE.md:25-27`
- Modify: `AGENTS.md` (after the *Reading the spec* section; and the CI bullet under *Rules every role follows*)
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` (§9, after item 3)
- Modify: `harness/CODEMAP.md` (`## CI` section; new `## Deploy` section)

- [ ] **Step 1: README "Hosting" row** — replace line 22 with:

```
| Hosting | Two images (`backend/Dockerfile`, `frontend/Dockerfile`), one env contract. Today: API on Railway Free, Postgres on Supabase, Redis on Upstash, PWA on Cloudflare Pages; later: the owner's Dokploy server running `deploy/compose.yml`. Runbook: `deploy/README.md` |
```

- [ ] **Step 2: CLAUDE.md** — replace the line `Two deployables on Railway, no shared code:` with:

```
Two deployables, no shared code, each built into its own image (`backend/Dockerfile`, `frontend/Dockerfile`; runbook `deploy/README.md`). Hosting today is the free split — API on Railway, Postgres on Supabase, Redis on Upstash, PWA on Cloudflare Pages; later the owner's Dokploy server runs the whole stack from `deploy/compose.yml`. Either way the API needs a host that never sleeps: the cron worker runs in-process.
```

- [ ] **Step 3: AGENTS.md** — insert after the *Reading the spec* section (before `## Rules every role follows`):

```
## Deployment

`deploy/README.md` is the runbook. Two images (`backend/Dockerfile`, `frontend/Dockerfile`) and one env contract (`deploy/.env.example`) deploy to either target: the free managed split today (Railway / Supabase / Upstash / Cloudflare Pages) or the owner's Dokploy server later (`deploy/compose.yml`). Agents own everything in the repo; the owner completes the runbook's checklist — accounts, secrets, the Google OAuth console, DNS — and agents never hold production secrets. `deploy/smoke-api.sh` / `deploy/smoke-web.sh` are the post-deploy checks. CD (registry push + deploy webhook) is parked in `harness/BACKLOG.md`.
```

And in the CI bullet under *Rules every role follows*, change `backend-unit`, `backend-integration` and `harness-tooling` run on every push` to `backend-unit`, `backend-integration`, `frontend`, `docker-images` and `harness-tooling` run on every push`.

- [ ] **Step 4: Backend spec §9 addendum** — after item 3 (`Deploy compiled Go binary…`) add:

```
4.  Addendum (2026-09-25, see `deploy/README.md`): `ENCRYPTION_SECRET_KEY` (64 hex) and `JWT_SECRET` (≥ 32 bytes) are boot requirements; `FRONTEND_ORIGIN` must list the PWA's exact origin. Hosting today is the free split (API on Railway Free, Postgres on Supabase via the session-mode pooler on port 5432 with `sslmode=require` — Railway egress is IPv4-only — Redis on Upstash over `rediss://`, PWA on Cloudflare Pages); later the owner's Dokploy server runs `deploy/compose.yml`. The images are `backend/Dockerfile` and `frontend/Dockerfile`.
```

- [ ] **Step 5: CODEMAP** — in `## CI`, add a bullet after `harness-tooling`:

```
- **`docker-images`** — `docker build` of `backend/Dockerfile` and `frontend/Dockerfile`; boots the API image with no env and requires the `config: DATABASE_URL is required` exit (the static binary runs), boots the web image and runs `deploy/smoke-web.sh` against it (SPA fallback, `sw.js` `no-cache`, `/_nuxt/*` immutable, build arg baked in), then `docker compose -f deploy/compose.yml config` with a scratch env file — and asserts it fails without one, so the `:?` guards stay. No registry push; the full compose stack is booted only in the plan's local verification.
```

And add a new section before `## Harness tooling`:

```
## Deploy (`deploy/`)

`backend/Dockerfile` (golang:1.25-alpine → alpine, `CGO_ENABLED=0`, non-root `api`, `HEALTHCHECK` = busybox `wget` on `/healthz`) and `frontend/Dockerfile` (node:20-alpine `npm ci && npx nuxi generate` → caddy:2-alpine with `frontend/Caddyfile`: `try_files … /index.html`, `sw.js` + manifest `no-cache`, `/_nuxt/*` immutable, `/healthz`). The three `NUXT_PUBLIC_*` values are **build args** — a static site has no runtime config. `deploy/compose.yml` = `postgres:16-alpine` + `redis:7-alpine` (`--appendonly yes`, named volumes: `queue:webpush:delay` must survive restarts) + `api` + `web`, interpolated from `deploy/.env` (`deploy/.env.example`; `${VAR:?}` for every secret so `config` refuses a blank). Host ports `API_PORT`/`WEB_PORT` are overridable; Postgres/Redis are never published. `deploy/smoke-api.sh <api> <origin>` and `deploy/smoke-web.sh <web>` are the post-deploy checks CI and the runbook share. `deploy/README.md` is the runbook (env table, Railway Free split, Dokploy, owner checklist). Local boot: unique `COMPOSE_PROJECT_NAME`, `API_PORT=18080 WEB_PORT=18081` (8080 and 6379 are taken on the owner's machine).
```

- [ ] **Step 6: Check and commit**

```sh
grep -n 'deploy/README.md' README.md CLAUDE.md AGENTS.md harness/CODEMAP.md "project-base/Adaptive English Learning Platform - Backend Technical Specification.md" | wc -l
python3 tools/harness/cli.py validate
git add README.md CLAUDE.md AGENTS.md harness/CODEMAP.md "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"
git commit -m "docs: hosting targets, deployment paragraph, spec §9 addendum, CODEMAP deploy section"
```
Expected: the count is ≥ 5; validate prints nothing and exits 0.

---

## Verification

All from the worktree root, no cloud account needed. Docker Desktop must be running.

1. **Both images build**
   ```sh
   docker build --pull -t aelp-api:local -f backend/Dockerfile backend 2>&1 | tail -1
   docker build --pull -t aelp-web:local -f frontend/Dockerfile --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend 2>&1 | tail -1
   ```
   Expected: both end with `naming to docker.io/library/aelp-…:local`.

2. **The API binary runs in its image**
   ```sh
   docker run --rm aelp-api:local 2>&1 | tail -1; echo "exit=${PIPESTATUS[0]}"
   ```
   Expected: `… config: DATABASE_URL is required`, `exit=1`.

3. **Compose config refuses a blank, accepts a filled env**
   ```sh
   rm -f deploy/.env; docker compose -f deploy/compose.yml config -q; echo "no-env exit=$?"
   ```
   Expected: an error naming `POSTGRES_PASSWORD`, non-zero exit. Then create the scratch `deploy/.env` exactly as in Task 4 step 3 and:
   ```sh
   docker compose -f deploy/compose.yml config -q && echo "config ok"
   ```

4. **The stack boots locally and both smoke checks pass**
   ```sh
   export COMPOSE_PROJECT_NAME=aelp-deploy-verify
   docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 240
   docker compose -f deploy/compose.yml ps
   docker compose -f deploy/compose.yml logs api | grep -n 'migrations applied\|cors: allowing\|listening on'
   deploy/smoke-api.sh http://127.0.0.1:18080 http://localhost:18081 ; echo "api exit=$?"
   deploy/smoke-web.sh http://127.0.0.1:18081 ; echo "web exit=$?"
   ```
   Expected: four services running, `api`/`postgres`/`redis` healthy; the three log lines; both scripts all `ok`, both `exit=0`. The API check proves `/healthz` 200 with Postgres and Redis reachable, the auth route routed (`400 invalid_request`), the CORS preflight from `FRONTEND_ORIGIN` answered `204` with the origin echoed, and a foreign origin refused with `403`.

5. **Redis persistence is on** (the `queue:webpush:delay` requirement)
   ```sh
   docker compose -f deploy/compose.yml exec redis redis-cli config get appendonly
   ```
   Expected: `appendonly` / `yes`.

6. **Tear down, leave nothing**
   ```sh
   docker compose -f deploy/compose.yml down -v; unset COMPOSE_PROJECT_NAME; rm -f deploy/.env
   docker ps --format '{{.Names}}' | grep -c 'aelp-deploy-verify\|^web$' || echo "clean"
   git status --short | grep -c '\.env$' || echo "no .env staged"
   ```
   Expected: `clean` (or `0`) and `no .env staged` (or `0`).

7. **Workflow is valid and every `docker-images` step passes locally**
   ```sh
   actionlint .github/workflows/ci.yml && echo "actionlint ok"
   ```
   Then run the job's five `run:` blocks as in Task 5 step 3. Expected: all exit 0.

8. **Existing suites untouched** — from `backend/`: `make check` (gofmt clean, vet, `go test ./... -count=1 -race` all pass; nothing under `backend/` changed except the two new files). From `frontend/`: `npm run lint && npm run typecheck && npm run test:unit` pass.

9. **CI on the pushed branch** (execute skill step 10): `gh run list --branch <branch>` shows `docker-images` green alongside the other four jobs.

## Notes

- **Owner checklist is not executed by the executor.** The runbook's checklist is the hand-off; the plan is `done` when everything above passes locally and CI is green, not when the app is public. The reviewer verifies the runbook against the code, not against a live deployment.
- If `docker build --pull` of `golang:1.25-alpine` fails to resolve, check `backend/go.mod`'s `go` line and use the matching minor; do not downgrade the module.
- If `npm ci` inside the image fails with `ERESOLVE`, the lockfile and `package.json` disagree — that is a frontend bug to file, not something to fix with `--legacy-peer-deps` in the Dockerfile.
- Compose `.env` parsing: keep comments on their own lines in `deploy/.env.example`.

## Execution summary

Built exactly as planned. All 7 tasks completed, committed one per task, in `backend/Dockerfile`, `frontend/Dockerfile` + `Caddyfile`, `deploy/smoke-api.sh` + `deploy/smoke-web.sh`, `deploy/compose.yml` + `.env.example`, `.github/workflows/ci.yml` (`docker-images` job), `deploy/README.md`, and doc updates (README.md, CLAUDE.md, AGENTS.md, backend spec §9, harness/CODEMAP.md). No app code under `backend/internal/`, `backend/cmd/` or `frontend/` source changed.

**Deviations:**
1. Task 5's CI snippet as written in the plan (`for i in $(seq 1 20); do curl ...; done`) failed `actionlint` with a shellcheck `SC2034` (unused loop variable `i`). Fixed by renaming the loop variable to `_` — a one-token change within the plan's intent, not a loosened check. actionlint is clean after the fix.
2. `docker compose -f deploy/compose.yml config -q` without an env file reported a different missing `:?` variable on different runs (`POSTGRES_PASSWORD` in most runs, once `FRONTEND_ORIGIN`, once `NUXT_PUBLIC_API_BASE`) rather than deterministically naming `POSTGRES_PASSWORD` as the plan's expected output shows. This is Compose's own interpolation-order nondeterminism (confirmed by running the same command repeatedly), not a defect in `compose.yml` — every run still failed non-zero with a real `:?` guard message, which is the property that matters (the guards work; the *specific* first-named variable is not load-bearing).
3. `docker compose -f deploy/compose.yml config` on this machine's Compose v2.39.2 prints ports/healthchecks as expanded YAML objects (`published: "18080"` / `target: 8080`) rather than the plan's `"18080:8080"` shorthand string. Confirmed by grep against the expanded fields instead — the published ports, `--appendonly`, `redis_data` and `FRONTEND_ORIGIN` were all correct.

No other deviations. Every code reference in the runbook was checked to exist; `cli.py validate` and the doc cross-reference count (5, plan required ≥5) both passed.

### Plan `## Verification` output (all 9 items, run from the worktree root)

1. **Both images build** — `aelp-api:local` and `aelp-web:local` both ended `naming to docker.io/library/aelp-…:local`. Sizes: `aelp-api:local` 55MB, `aelp-web:local` 86.3MB.
2. **API binary runs in its image** — `2026/09/25 04:59:47 config: DATABASE_URL is required`, `exit=1`.
3. **Compose config refuses a blank, accepts a filled env** — no-env run: non-zero exit with a `:?` guard error (see deviation 2 above); filled scratch `deploy/.env`: `config ok`.
4. **Stack boots locally, both smoke checks pass**:
   ```
   NAME                            STATUS
   aelp-deploy-verify-api-1        Up (healthy)
   aelp-deploy-verify-postgres-1   Up (healthy)
   aelp-deploy-verify-redis-1      Up (healthy)
   aelp-deploy-verify-web-1        Up (healthy)

   api-1  | migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates]
   api-1  | cors: allowing [http://localhost:18081]
   api-1  | listening on [::]:8080 (GIN_MODE=release)

   smoke-api.sh: ok healthz status 200, ok healthz body 1, ok auth/google empty body 400,
                 ok preflight status 204, ok preflight allow-origin, ok preflight foreign origin 403
                 -> api exit=0
   smoke-web.sh: ok index 200, ok spa fallback 200, ok sw.js present 200, ok sw.js cache-control no-cache,
                 ok manifest cache-control no-cache, ok hashed asset found, ok asset cache-control immutable
                 -> web exit=0
   ```
5. **Redis persistence** — `appendonly` / `yes`.
6. **Tear down, leave nothing** — `docker compose down -v` removed all 4 containers, both volumes, the network; `docker ps` count 0; `git status --short` showed no `.env` staged.
7. **Workflow valid, docker-images steps pass locally** — `actionlint ok` (after deviation 1's fix); all 5 `run:` blocks (API build, API-run check, web build, web check incl. `deploy/smoke-web.sh`, compose config with/without env) exited 0.
8. **Existing suites untouched** — backend `make check` (gofmt clean, vet clean, `go test ./... -count=1 -race`): all 13 packages `ok`. Frontend: `npm run lint` clean, `npm run typecheck` clean, `npm run test:unit` 16 files / 78 tests passed, `npm run build` succeeded.
9. **CI on the pushed branch** — https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36096793213 — conclusion `success`, all 5 jobs green: `backend-unit`, `backend-integration`, `harness-tooling`, `frontend`, `docker-images`.

### Runtime proof (executing-plans/harness-execute skill step 8)

- Built both images from a clean `docker build --pull`.
- Ran the full backend suite (`make check`, race detector on, no service env vars exported) and the full frontend suite (`lint && typecheck && test:unit && build`) — not just plan-added tests.
- Booted the real 4-service stack (`postgres`, `redis`, `api`, `web`) via `docker compose -f deploy/compose.yml up -d --build --wait`, exercised it end to end with both smoke scripts against `http://127.0.0.1:18080` / `18081`, and inspected the API's own boot log lines (migrations applied, CORS allow-list, listening address).
- Ran every command the plan documents a human to run: both direct `docker build`s, the API no-env run, both `compose config` guard checks (blank and filled), the CI job's five `run:` blocks locally, `actionlint`.
- Cleaned up after every stage: `docker compose down -v` + volume/network removal confirmed, `deploy/.env` deleted, `aelp-web-smoke`/`web` throwaway containers stopped and confirmed absent from `docker ps`, `COMPOSE_PROJECT_NAME` unset. Used the unique `aelp-deploy-verify` project name and ports `18080`/`18081` throughout, since `8080`/`6379` are held by unrelated processes on this machine (confirmed via `lsof` before starting).

### Push and CI

Pushed `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-` to `origin`. No PR opened (owner takes one PR per day). CI run https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36096793213 — green.

### Owner checklist (not executed here)

`deploy/README.md`'s *Owner checklist* — creating the Supabase/Upstash/Railway/Cloudflare Pages accounts, generating and storing secrets, wiring the Google OAuth console, and running the smoke scripts against public URLs — is explicitly the owner's hand-off, per the plan's own Notes ("Owner checklist is not executed by the executor"). None of it was attempted.
