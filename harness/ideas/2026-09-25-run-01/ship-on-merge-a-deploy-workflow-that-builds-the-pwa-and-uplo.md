---
type: feature
status: proposed
source: human
run: 2026-09-25-run-01
priority: high
---
# Ship on merge: a deploy workflow that builds the PWA and uploads it to Cloudflare Pages on every push to main, then smoke-checks both public URLs

## Why
The live PWA on Cloudflare Pages (`english-learning`, `https://english-learning-e6a.pages.dev`) has **no Git connection**: every deploy so far (2026-09-25) was a manual `npx nuxi generate` + `wrangler pages deploy` from the orchestrator's machine. Railway rebuilds the API from `main` on every push, so after the daily PR merges the backend ships and the frontend silently does not — the owner noticed the site never changes after merges. With the retro restyle about to land screen by screen, a frontend that does not ship on merge makes the daily PR meaningless for users. "The owner's daily merge is the deploy" (AGENTS.md) must hold for both deployables.

## Expected output
- `.github/workflows/deploy.yml`: on `push` to `main` (and `workflow_dispatch`), after the existing CI jobs succeed (`workflow_run` on `ci.yml` completed+success, or a `needs` on a reusable job), one job that: checks out, `npm ci` in `frontend/`, `npx nuxi generate` with `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY` from **repository variables** (they are public values), then `npx wrangler pages deploy frontend/.output/public --project-name english-learning --branch main` using **repository secrets** `CLOUDFLARE_API_TOKEN` (scope: Pages:Edit) and `CLOUDFLARE_ACCOUNT_ID`; then waits up to 2 min for Railway (`curl --max-time` loop on `/healthz`) and runs `deploy/smoke-web.sh https://english-learning-e6a.pages.dev` and `deploy/smoke-api.sh <api> <origin>` with the URLs from repository variables `PAGES_URL` / `API_URL`. The job fails visibly on a failed upload or smoke check; it never runs on `harness/**` branches or PRs. Concurrency group `deploy` with cancel-in-progress off.
- The web smoke check must pass on Pages: the deep-link 404 fix (`pages-ignores-the-redirects-spa-rewrite…` inbox bug) either lands first or that single check is downgraded to a warning until it does — say which in the plan.
- `deploy/README.md` gains "Ship on merge": the four variables and two secrets, where to create the token (Cloudflare → My Profile → API Tokens → Edit Cloudflare Workers template + Pages Write, or a custom token with Account.Cloudflare Pages:Edit), and the rule that Pages is never uploaded by hand again. Owner checklist item: create the token and set the GitHub variables/secrets (the plan stops there — agents never hold the token).
- AGENTS.md CI paragraph lists `deploy.yml` and its trigger; `harness/CODEMAP.md` CI section too.
- Verification: `actionlint` on the workflow, a `workflow_dispatch` dry-run mode (`inputs.dry_run` skips the upload) proven in CI on the branch, and — after the owner sets the secrets — one real run on `main` whose smoke step is green. Railway side: nothing to add (already deploys `main`); the plan notes that the API currently runs a CLI-uploaded build and that the first push to `main` after the daily PR merge replaces it.

## Evidence
- Pages project has `Git Provider: No` (`wrangler pages project list`, 2026-09-25); deploys 14110572 and 7d4bbbc0 were manual uploads. Railway service `api` source = GitHub `main` (deployment meta `branch: main`).
- `deploy/smoke-web.sh`, `deploy/smoke-api.sh` exist on the deploy branch (in daily PR #30); `.github/workflows/ci.yml` has `backend-unit`, `backend-integration`, `frontend`, `harness-tooling`, `docker-images`.
- Cloudflare: Pages direct upload via wrangler needs `CLOUDFLARE_API_TOKEN` + `CLOUDFLARE_ACCOUNT_ID` — https://developers.cloudflare.com/pages/how-to/use-direct-upload-with-continuous-integration/
- Related: backlog entry "Continuous delivery to Dokploy" (GHCR + webhook) is the later, self-hosted version of this workflow; this idea is the free-tier version and should be written so the Dokploy job can be added beside it.
