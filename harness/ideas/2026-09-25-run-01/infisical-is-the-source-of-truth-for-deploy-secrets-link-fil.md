---
type: feature
status: planned
source: human
run: 2026-09-25-run-01
priority: medium
plan: harness/plans/2026-09-26-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil.md
---
# Infisical is the source of truth for deploy secrets: link file in the repo, runbook section, Railway sync

## Why
Until 2026-09-25 the production env existed only on the Railway service and in one gitignored file on the owner's Mac. A new device or a second person had to rebuild it from four dashboards. The owner chose Infisical (open source, free plan: 5 identities, unlimited projects, 3 environments, 10 secret syncs; self-hostable later on the Dokploy box). Done by hand today: Secret Manager project `english-learning` (id `8573a7c5-4e89-4d44-8ead-8f4bc8528cc9`), environment `prod` holds the 24 non-empty keys of `deploy/.env`; `deploy/`'s parent has an untracked `.infisical.json` pointing at it. Agents never see values: `infisical export --env prod --format dotenv > deploy/.env` regenerates the file on any machine after `infisical login`.

## Expected output
- `.infisical.json` committed at the repo root (`{"workspaceId":"8573a7c5-…","defaultEnvironment":"prod"}` — a project id, not a secret) so `infisical export` works from any checkout; `deploy/.env` stays gitignored.
- `deploy/README.md` gains an "Env source of truth" section: `infisical login`, `infisical export --env prod --format dotenv > deploy/.env`, `infisical secrets set --env prod --file deploy/.env` (note: the CLI refuses a file with an empty value — filter `grep -E '^[A-Z_]+=.+'` first), inviting a collaborator, and the rule that Railway variables are pushed by Infisical's **Railway secret sync** (owner creates a Railway account token, adds it as an Infisical Railway connection, sync → project `english-learning` / env `production` / service `api`, auto-redeploy on) rather than by hand. `deploy/README.md` lives on the deploy branch — conditional edit as earlier plans did, or land this after the daily PR.
- The owner checklist artifact/`deploy/README.md` step list starts with "infisical export" instead of "fill the file".
- Optional: `deploy/smoke-*.sh` read `RAILWAY_DOMAIN`/`PAGES_DOMAIN` via `infisical run -- sh deploy/smoke-api.sh …` example.

## Evidence
- Infisical free plan and Railway secret sync: https://infisical.com/pricing , https://infisical.com/docs/integrations/secret-syncs/railway
- CLI facts verified 2026-09-25: `infisical secrets set --file` exists; a blank value aborts the whole upload ("Secret key 'DEEPSEEK_API_KEY' has an empty value"); `infisical init` lists every product's projects by name only, so a Secret Manager project must exist first (org sample projects are of types agent-vault, kms, secret-scanning, cert-manager, pam plus one secret-manager).

## Evaluation
_Evaluator, 2026-09-26 — daily decide (feature queue; owner idea)._

**Select — medium. Plan written today (draft — waits for `/approve`).**

*Is the Why real?* Yes — the owner has already moved the 24 production keys into Infisical and works from `infisical export`; today the repo does not know that, so a second machine or person rebuilds the env from four dashboards. *Achievable in one plan?* Yes, ≈1.5 h and no code: commit `.infisical.json` (a project id, not a secret — the idea says so and `deploy/.env` stays gitignored), a runbook section, the owner checklist reordered, a smoke example. *Dependencies:* none (the deploy branch is on `main`). *Priority:* medium — developer-workflow, no learner impact; not auto-approved.
