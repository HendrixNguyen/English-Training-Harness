---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md
---
# deploy/compose.yml drops the AI provider base-URL and model variables the live deployment depends on

## Why
The runbook and AGENTS.md call `deploy/.env.example` + `deploy/compose.yml` "one env contract" for both targets, but `internal/config` also reads `GEMINI_BASE_URL`, `OPENAI_BASE_URL`, `DEEPSEEK_BASE_URL`, `GEMINI_MODEL`, `OPENAI_MODEL` and `DEEPSEEK_MODEL`. Compose passes only the keys listed under `api.environment`, so those six never reach the container. The live deployment runs its only AI provider (OpenRouter) through `OPENAI_BASE_URL` + `OPENAI_MODEL`. Moving to Dokploy with this file would silently fall back to api.openai.com with the default model. With an OpenRouter key that means every AI route fails (roadmap generation, placement, exercises), and nothing at boot says why. The runbook's env table leaves the six out too, so the Railway rows are incomplete as well.

## Expected output
- `deploy/compose.yml` `api.environment` passes all six through as `${VAR:-}`.
- `deploy/.env.example` lists them, blank, with a one-line comment that `OPENAI_*` works for any OpenAI-compatible endpoint, e.g. OpenRouter.
- The `deploy/README.md` env table gains the six rows (optional; default provider URL/model when blank).
- Optional, same fix: `VAPID_SUBJECT` should not silently default to `mailto:admin@example.com` in production. Leave it blank, or require it whenever the VAPID keys are set.

## Evidence
- Plan: `harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md` (branch origin/harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- @ c46b1df).
- `grep -o '"[A-Z_]*"' backend/internal/config/*.go backend/internal/airouter/*.go` lists all six variables. `deploy/compose.yml` lines 36-56 (`api.environment`) have none of them.
- Reproduced: I appended `OPENAI_BASE_URL=…` and `OPENAI_MODEL=…` to a scratch `deploy/.env` and booted the stack (project `aelp-rev-deploy`). Both `docker compose -f deploy/compose.yml config | grep -c 'OPENAI_BASE_URL\|OPENAI_MODEL'` and `docker compose exec api env | grep -c …` returned `0`.
- The live AI setup (OpenRouter in the OPENAI slot) is the owner's 2026-09-25 choice, recorded with the live deployment.

## Evaluation
_Evaluator, 2026-09-26._ **Select — medium, folded into** `harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md` (the Dokploy hardening plan: Caddyfile miss handling + shell `no-cache`, the six AI base-URL/model variables through compose and the runbook table, `smoke-api.sh` reporting every check). The deploy branch it was filed against is on `origin/main` now.
