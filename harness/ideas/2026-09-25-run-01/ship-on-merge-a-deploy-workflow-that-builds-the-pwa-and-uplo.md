---
type: feature
status: planned
source: human
run: 2026-09-25-run-01
priority: high
plan: harness/plans/2026-09-25-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md
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

## Evaluation

**Verdict: select, `priority: high` (auto-approve).** The *Why* is real and verified today: `wrangler pages project list` shows the Pages project with no Git provider, and both live deploys were CLI uploads from one machine. Railway rebuilds `main` on every push, so after each daily PR the API moves and the PWA does not — the owner's daily merge is the deploy for only one of the two deployables, which breaks the harness's one gate. With the retro restyle landing screen by screen this is the difference between the daily PR reaching users or not; that is a happy-path user impact, not tidiness.

**Achievable in one plan (~2 h):** one new workflow file, a one-knob edit to `deploy/smoke-web.sh`, three doc edits. No app code. Facts checked against `origin/main` (`68529ad`, fetched during this evaluation):
- PR #30 (`harness/daily-2026-09-25`) is **merged** (2026-09-25 14:18Z): `deploy/smoke-web.sh`, `deploy/smoke-api.sh`, `deploy/README.md`, both Dockerfiles and the `docker-images` CI job are on `origin/main`. The executor's base has the scripts, so the workflow calls them directly — no inline-`curl` fallback is planned.
- `.github/workflows/ci.yml` is named `CI`, triggers on `push` to `main`/`harness/**` and `pull_request`, jobs `backend-unit`, `backend-integration`, `frontend`, `docker-images`, `harness-tooling`. `frontend` builds with `npm run build` (`nuxi build`); the Pages artefact needs `npx nuxi generate` (`ssr: false`, `nitro.prerender.autoSubfolderIndex: false` — output `.output/public`).
- `frontend/public/_redirects` and `_headers` are on `main` (login-308 fix merged), but the inbox bug `pages-ignores-the-redirects-spa-rewrite-while-404-html-exist` is still `proposed`: a deep link on Pages answers 404 with the shell because `404.html` wins over the splat rewrite. `deploy/smoke-web.sh` therefore fails its "spa fallback" line against the live URL today.
- `actionlint` is installed on this machine (`/opt/homebrew/bin/actionlint`) and was the verification tool of the containerised-deploy plan; the repo's CI does not run it.
- The GitHub repo (`HendrixNguyen/English-Training-Harness`, default branch `main`) has **no** repository variables or secrets yet (`gh variable list`, `gh secret list` both empty).

**Design answers to the idea's open questions:**
1. *Trigger.* `workflow_run` on workflow `CI`, `types: [completed]`, guarded in the job's `if:` by `conclusion == 'success' && head_branch == 'main' && event == 'push'` — so PR runs and `harness/**` runs never deploy — plus `workflow_dispatch` with a boolean `dry_run` (build only, no upload, no smoke). The job checks out `workflow_run.head_sha`, the exact commit CI proved. `concurrency: { group: deploy, cancel-in-progress: false }`. A `workflow_run` event only fires for workflow files on the default branch, so this workflow cannot run on its own `harness/*` branch at all — CI proof on the branch is `actionlint` + the existing `frontend` job; the first real run is an owner step after merge.
2. *Deep-link 404.* Downgraded to a warning, not fixed here: `deploy/smoke-web.sh` gains an opt-in `SMOKE_WEB_SPA_WARN=1` that turns only the "spa fallback" line into `WARN` (exit unaffected); the workflow sets it, the `docker-images` job does not, so the Caddy image stays strictly checked. When the inbox bug's fix lands, the knob is removed from the workflow.
3. *Configuration.* Public values from repository **variables** `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY`, `PAGES_URL`, `API_URL`; secrets `CLOUDFLARE_API_TOKEN` (custom token, Account → Cloudflare Pages → Edit), `CLOUDFLARE_ACCOUNT_ID`. A first step fails a real run with one message naming every missing item; a dry run only warns. Agents never hold the token — the plan ends at the owner checklist.
4. *Railway.* Nothing to configure; the job polls `$API_URL/healthz` for up to 2 min before `deploy/smoke-api.sh`, because Railway's rebuild started at the push while CI ran.

**Dependencies:** none unbuilt. The Dokploy CD backlog entry is a later sibling job in the same file; nothing here blocks it.

**Priority rationale:** `high` — the owner's only gate (the daily merge) does not ship the PWA today, so every merged frontend change is invisible to users until someone uploads by hand.

**Revision (owner, 2026-09-25, before execution):** the "deploy on every green `main` run" design is replaced. CD runs only on pushes to a long-lived release branch **`production`** (plus `workflow_dispatch` with `dry_run`) — never `main`, `harness/**` or PRs — and ships **both** deployables from that commit: the API through `npx -y @railway/cli@latest up . --service api --ci` with a project-token secret `RAILWAY_TOKEN` (Railway's GitHub auto-deploy from `main` is disconnected by the owner), the PWA through wrangler as before, then the `/healthz` poll and both smoke scripts. A tool-neutral nightly routine `.agents/routines/daily-ship.md` (22:00 local / 15:00 UTC, after the review run and the owner's merge) fast-forwards `production` to `origin/main` only when `main`'s CI is green and the push is a pure fast-forward, then reports the Deploy run; the owner can push `production` by hand. The executor bootstraps `production` from `origin/main` (allowed — it is not `main`); the owner protects it. Everything else (actionlint proof, `SMOKE_WEB_SPA_WARN`, docs) stays. The plan is revised accordingly and stays `approved`.
