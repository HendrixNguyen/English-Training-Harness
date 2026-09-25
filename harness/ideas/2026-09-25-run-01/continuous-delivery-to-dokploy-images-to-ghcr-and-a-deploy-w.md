---
type: feature
status: proposed
source: human
run: 2026-09-25-run-01
---
# Continuous delivery to Dokploy: images to GHCR and a deploy webhook on merge to main

## Why
Once the owner moves hosting to the bare-metal Dokploy server (sibling idea "Containerised deploy…" builds the images and runbook), the deploy must stay what it is today: the owner's daily PR merge. Railway and Cloudflare Pages redeploy on push by themselves, so no CD exists in the repo. A home server behind a tunnel or NAT is better **pulled** than pushed to: CI builds the images once, publishes them to GitHub Container Registry, and pokes Dokploy's deploy webhook; Dokploy pulls the tagged images and restarts. That also makes the harness's runtime-proof honest — the artifact CI tested is the artifact that runs.

Depends on the sibling idea; not before it. Priority is the evaluator's call — it matters only once the Dokploy target is live.

## Expected output
- **`.github/workflows/deploy.yml`** — on push to `main` (and `workflow_dispatch`): build both images from the Dockerfiles with `docker/build-push-action`, tag `ghcr.io/<owner>/english-api:<sha>` and `:main` (same for `english-web`), push with `GITHUB_TOKEN` (`packages: write`), then `curl --fail --max-time 30 "$DOKPLOY_DEPLOY_WEBHOOK"` (repository secret). Runs only after the existing CI jobs succeed (`needs` on a reusable workflow, or `workflow_run` on `ci.yml`), never on `harness/**` branches.
- **`deploy/compose.yml`** switches `api`/`web` from `build:` to `image: ghcr.io/…:main` with `pull_policy: always`; the Dokploy app is created as a Compose app with "deploy on webhook", and the registry is added to Dokploy as a read-only GHCR credential (owner step; a fine-grained PAT with `read:packages`, documented in the runbook).
- **Rollback**: the runbook shows `docker compose pull` of a previous `<sha>` tag and the Dokploy "redeploy previous" button; CI tags are immutable.
- **Smoke check** from the sibling idea runs as the last step of the workflow against the public URL and fails the run (not the deploy — Dokploy has already switched) so the failure is visible in Actions and in the 20:00 review routine's report.
- Docs: runbook "Dokploy" section gains the CD subsection; AGENTS.md's CI paragraph lists `deploy.yml` and states it never runs on `harness/**`.

Out of scope: preview environments per branch, blue/green, database migration gating (migrations still run at API boot), Railway (it keeps building from `main` on its own).

## Evidence
- Repo: `.github/workflows/ci.yml` runs on `main` and `harness/**` pushes; no `deploy.yml`; no registry pushes anywhere.
- Owner decision (conversation 2026-09-25): future hosting is a local bare-metal server with Dokploy; today's Railway/Supabase/Upstash/Pages split is temporary.
- Dokploy: per-application "Deploy webhook" URL and "Docker image" / Compose providers pulling from private registries — https://docs.dokploy.com ; GHCR with `GITHUB_TOKEN` — https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
- A public HTTPS origin is required for Google OAuth redirect URIs and Web Push; Dokploy's Traefik handles Let's Encrypt when 80/443 are reachable, otherwise Cloudflare Tunnel (sibling idea runbook).
