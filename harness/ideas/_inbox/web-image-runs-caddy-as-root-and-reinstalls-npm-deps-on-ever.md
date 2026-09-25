---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Web image runs Caddy as root and reinstalls npm deps on every source change

## Why
Two small quality gaps in `frontend/Dockerfile`:
1. The runtime stage inherits `caddy:2-alpine`'s root user (`id` inside the container gives `uid=0(root)`). `backend/Dockerfile` deliberately runs as non-root `api`. A static file server has no reason to run as root on the Dokploy box.
2. `COPY . .` comes before `RUN npm ci`, so any source edit invalidates the dependency layer and every image build (local, CI `docker-images`, Dokploy) reinstalls all node modules. The comment explains why: `postinstall` runs `nuxi prepare`. It can be avoided with `COPY package.json package-lock.json ./` → `RUN npm ci --ignore-scripts` → `COPY . .` → `RUN npx nuxi prepare && npx nuxi generate`.

## Expected output
- The web image runs as a non-root user: listen on an unprivileged port such as `:8080` with `USER caddy`, or grant `cap_net_bind_service`. Compose, the healthcheck, the runbook and the CI port mapping are updated to match.
- Dependency install is cached across source-only changes.

## Evidence
- Plan: `harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md` (branch origin/harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- @ c46b1df); `frontend/Dockerfile` lines 7-11 and 23-28.
- `docker exec aelp-rev-deploy-web id` gave `uid=0(root)`.
