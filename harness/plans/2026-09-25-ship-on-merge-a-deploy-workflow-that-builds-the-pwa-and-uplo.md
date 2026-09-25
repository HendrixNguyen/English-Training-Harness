---
idea: harness/ideas/2026-09-25-run-01/ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md
status: approved
priority: high
merged: false
---
# Ship on merge: a deploy workflow that builds the PWA and uploads it to Cloudflare Pages on every push to main, then smoke-checks both public URLs — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md`

> **Revised 2026-09-25 (owner decision, before execution).** The first draft deployed the PWA on every green CI run on `main`. The owner replaced that with a **long-lived release branch `production`**: `deploy.yml` runs only on pushes to `production` (plus a manual `workflow_dispatch` with `dry_run`), ships **both** deployables from that commit — the API through the Railway CLI and the PWA through wrangler — and a **nightly ship routine** fast-forwards `production` to `origin/main` after the owner's daily merge. Railway's own GitHub auto-deploy from `main` is disconnected by the owner. The idea's `## Evaluation` records the decision.

**Goal:** A push to `production` ships the whole product: the `Deploy` workflow deploys `backend/` to the Railway service `api` with the Railway CLI, builds `frontend/` with `nuxi generate` and uploads `.output/public` to the Cloudflare Pages project `english-learning`, waits for the API, and runs both smoke scripts against the public URLs — failing visibly on a bad deploy or smoke check, never running on `main`, `harness/**` or pull requests. A nightly routine (`.agents/routines/daily-ship.md`, 22:00 local) fast-forwards `production` to `origin/main` when `main`'s CI is green and reports the Deploy run; the owner can push `production` by hand for the same effect.

**Why now (`priority: high`):** The Pages project has no Git connection (`wrangler pages project list` → `Git Provider: No`); both live deploys were manual `wrangler pages deploy` uploads from one machine, and the API deploys from `main` on every push, so a merged daily PR moves the API and leaves the PWA behind — the two deployables drift on every merge. No app code changes; CI/infra + one routine + docs, so no design doc.

**Base branch facts (checked 2026-09-25 against `origin/main` = `68529ad`):** PR #30 is merged, so `deploy/smoke-web.sh`, `deploy/smoke-api.sh`, `deploy/README.md`, both Dockerfiles and the `docker-images` CI job are on the base; the workflow calls the real scripts — there is no inline-`curl` fallback. If `test -f deploy/smoke-web.sh` is unexpectedly false on your freshly fetched base, **stop and report**; do not merge or cherry-pick. `frontend/public/_redirects` + `_headers` are on the base (login-308 fix), but the inbox bug `pages-ignores-the-redirects-spa-rewrite-while-404-html-exist` is still open: `GET /learn/abc` on Pages is 404 with the shell, so `deploy/smoke-web.sh` fails that one line against the live URL; this plan downgrades it to a warning (Task 2) rather than fixing it. The remote has **no** `production` branch yet (`git ls-remote --heads origin production` is empty) and **no** repository variables or secrets (`gh variable list` / `gh secret list` empty). Railway service `api` is configured with **Root Directory `backend`**, builder Dockerfile, healthcheck `/healthz` (`deploy/README.md` Target A).

**Design decisions:**
1. **Trigger = `push` to `production` + `workflow_dispatch`.** `on.push.branches: [production]` and nothing else — no `main`, no `harness/**`, no `pull_request`. `production` only ever receives fast-forwards of `origin/main` (Design decision 6), so every commit on it has already passed CI on `main`; the workflow does not re-run CI. `workflow_dispatch` has a boolean `dry_run` (default `false`): build the site, skip both deploys, the API wait and both smoke checks — the owner's first, safe run and the way to check the variables. A dispatch runs on whatever ref is chosen in the Actions UI; the routine and the runbook say `production`.
2. **`concurrency: { group: deploy, cancel-in-progress: false }`** — one ship at a time, never cancelled: a cancelled run can leave Pages and Railway on different commits.
3. **Both deployables, API first.** Step order: configuration check → `npx -y @railway/cli@latest up . --service api --ci` from the repo root (the service's Root Directory `backend` is applied by Railway to the uploaded tree, which is the form that matches the service as configured; `--ci` streams the build log and exits non-zero on a failed build) → Node 20 `npm ci` + `npx nuxi generate` → `npx --yes wrangler@4 pages deploy .output/public --project-name english-learning --branch main` → 2-minute `/healthz` poll → `deploy/smoke-api.sh` → `deploy/smoke-web.sh`. Railway goes first so its build and health-checked swap run while the PWA builds. `railway up` runs before `npm ci` so `frontend/node_modules` is not in the upload (the CLI also honours `.gitignore`). *Inference, to be confirmed on the owner's first real run:* Railway's monorepo docs say the Root Directory setting applies to CLI uploads; if that run's build log cannot find `backend/Dockerfile`, the owner clears Root Directory on the service and the executor of the follow-up changes the command to `up ./backend` — not both.
4. **Public values are repository *variables*, credentials are *secrets*.** Variables `NUXT_PUBLIC_API_BASE`, `NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `NUXT_PUBLIC_VAPID_PUBLIC_KEY` (baked in by `nuxi generate`), `PAGES_URL`, `API_URL` (smoke targets); secrets `CLOUDFLARE_API_TOKEN` (custom token, *Account → Cloudflare Pages → Edit*), `CLOUDFLARE_ACCOUNT_ID`, `RAILWAY_TOKEN` (a Railway **project token**: project → Settings → Tokens, environment `production`). The first step fails a real run with one line naming every unset item; a dry run only warns. Creating them is the owner's checklist — agents never hold a token.
5. **Deep-link 404 is a warning, not a failure.** `deploy/smoke-web.sh` gains an opt-in `SMOKE_WEB_SPA_WARN=1` that turns only the `spa fallback (/learn/abc)` line into `WARN` without touching the exit code; the workflow sets it, the `docker-images` job does not, so the Caddy image stays strictly checked. The inbox fix's plan removes the `env:` line.
6. **`production` moves only by pure fast-forward.** The nightly routine (`.agents/routines/daily-ship.md`) pushes `origin/main:production` only when `git merge-base --is-ancestor origin/production origin/main` holds and `main`'s latest CI run is green; otherwise it stops and reports. Never a merge commit, never `--force`. The owner ships by hand with the same push or `gh workflow run deploy.yml --ref production`. Pushing `production` is the one exception to "push only `harness/*`": the routine, the owner, and once the executor's bootstrap task (Task 5) — never `main`.
7. **Railway's GitHub auto-deploy from `main` must be disconnected** (Service → Settings → Source → disconnect) — otherwise `main` merges keep deploying the API ahead of the PWA. Owner checklist item; the CLI push is then the only deploy path. Until the owner does it, the API can deploy twice per day (once from the merge, once from the ship) — harmless, but the drift the owner wants gone remains.
8. **Bootstrap.** The executor creates `production` from the current `origin/main` (`git push origin origin/main:refs/heads/production`) so the routine has something to fast-forward. That commit does not contain `deploy.yml`, so nothing runs on creation; the first Deploy run happens on the first fast-forward after this plan's daily PR merges. The owner protects the branch afterwards (block force-push and deletion; no PR requirement, the routine pushes directly).
9. **`npx --yes wrangler@4` / `npx -y @railway/cli@latest`**, not pinned actions: the same commands the runbook documents for a manual deploy, so workflow and runbook cannot drift; `--yes` keeps `npx` from prompting on a non-TTY runner. Node 20 (the runbook's Pages version); CI's `frontend` job stays on `lts/*`.

**Tech stack:** GitHub Actions (`actions/checkout@v7`, `actions/setup-node@v7`, as in `ci.yml`), `wrangler@4`, `@railway/cli`, POSIX `sh` (the smoke scripts are `#!/bin/sh`), `actionlint` (`/opt/homebrew/bin/actionlint`). No new repository dependencies.

**Run every command from the worktree root** unless a step says `frontend/`. `rg` is not installed — use `grep -n`. This session's shell proxy can mangle `ls` output; use globs or `find`.

---

## File structure

| Path | Change |
|---|---|
| `.github/workflows/deploy.yml` | **New.** The `Deploy` workflow: API to Railway, PWA to Pages, smoke checks (Task 1). |
| `deploy/smoke-web.sh` | `SMOKE_WEB_SPA_WARN` knob around the one deep-link check (Task 2). |
| `.agents/routines/daily-ship.md` | **New.** The 22:00 nightly ship routine (Task 3). |
| `.agents/routines/README.md` | Its row in *The day*; the `production` push exception in the shared rules (Task 3). |
| `deploy/README.md` | Correct the Target A paragraphs, add **Ship from `production`**, a Smoke-check note and owner-checklist items (Task 4). |
| `AGENTS.md` | CI bullet gains the `deploy.yml` sentence; the push rule gains the `production` exception (Task 4). |
| `harness/CODEMAP.md` | CI section gains the `deploy.yml` entry (Task 4). |
| remote branch `production` | **Created** from `origin/main` by Task 5 — the only non-file change. |

---

## Tasks

### Task 1: `.github/workflows/deploy.yml`

**Files:** create `.github/workflows/deploy.yml`.

- [ ] Write the file exactly as below. Keep the comments — they are the only place the trigger choice is explained.

```yaml
name: Deploy

# Delivery loop (AGENTS.md; deploy/README.md "Ship from production"): on a
# push to the release branch `production` — which only ever receives
# fast-forwards of main, by the nightly ship routine or the owner — deploy the
# API to Railway with the Railway CLI, build the PWA and upload it to
# Cloudflare Pages, then smoke-check both public URLs. Never on main, never on
# harness/** branches, never on pull requests. `production` commits have
# already passed CI on main; this workflow does not re-run it.
on:
  push:
    branches: [production]
  workflow_dispatch:
    inputs:
      dry_run:
        description: Build the PWA but skip both deploys and the smoke checks
        type: boolean
        default: false

permissions:
  contents: read

# One ship at a time and never cancelled: a cancelled run can leave Pages and
# Railway on different commits.
concurrency:
  group: deploy
  cancel-in-progress: false

jobs:
  ship:
    runs-on: ubuntu-latest
    timeout-minutes: 25
    env:
      DRY_RUN: ${{ github.event_name == 'workflow_dispatch' && inputs.dry_run == true }}
      PAGES_URL: ${{ vars.PAGES_URL }}
      API_URL: ${{ vars.API_URL }}
      # Public build-time values (Nuxt reads NUXT_PUBLIC_* while generating).
      NUXT_PUBLIC_API_BASE: ${{ vars.NUXT_PUBLIC_API_BASE }}
      NUXT_PUBLIC_GOOGLE_CLIENT_ID: ${{ vars.NUXT_PUBLIC_GOOGLE_CLIENT_ID }}
      NUXT_PUBLIC_VAPID_PUBLIC_KEY: ${{ vars.NUXT_PUBLIC_VAPID_PUBLIC_KEY }}
    steps:
      - uses: actions/checkout@v7

      - name: Check configuration
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
        run: |
          missing=""
          for v in NUXT_PUBLIC_API_BASE NUXT_PUBLIC_GOOGLE_CLIENT_ID NUXT_PUBLIC_VAPID_PUBLIC_KEY \
                   PAGES_URL API_URL CLOUDFLARE_API_TOKEN CLOUDFLARE_ACCOUNT_ID RAILWAY_TOKEN; do
            [ -n "${!v:-}" ] || missing="$missing $v"
          done
          if [ -n "$missing" ]; then
            if [ "$DRY_RUN" = "true" ]; then
              echo "::warning::dry run with unset:$missing — see deploy/README.md 'Ship from production'"
            else
              echo "::error::not configured:$missing — see deploy/README.md 'Ship from production'"
              exit 1
            fi
          fi
          echo "dry_run=$DRY_RUN ref=$GITHUB_REF_NAME sha=$GITHUB_SHA"

      # API first: Railway builds and health-checks the swap while the PWA
      # builds. Runs from the repo root before npm ci so frontend/node_modules
      # is never uploaded; the service's Root Directory (backend) applies.
      - name: Deploy the API to Railway
        if: env.DRY_RUN != 'true'
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
        run: npx -y @railway/cli@latest up . --service api --ci

      - uses: actions/setup-node@v7
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json

      - name: Install
        working-directory: frontend
        run: npm ci

      - name: Generate the static site
        working-directory: frontend
        run: |
          npx nuxi generate
          test -f .output/public/index.html
          test -f .output/public/sw.js
          test -f .output/public/manifest.webmanifest

      - name: Upload the PWA to Cloudflare Pages
        if: env.DRY_RUN != 'true'
        working-directory: frontend
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
        run: >-
          npx --yes wrangler@4 pages deploy .output/public
          --project-name english-learning --branch main

      - name: Wait for the API
        if: env.DRY_RUN != 'true'
        run: |
          for _ in $(seq 1 24); do
            code=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' "$API_URL/healthz" || true)
            if [ "$code" = "200" ]; then echo "healthz: 200"; exit 0; fi
            sleep 5
          done
          echo "::error::$API_URL/healthz did not answer 200 within 2 minutes"
          exit 1

      - name: Smoke-check the API
        if: env.DRY_RUN != 'true'
        run: deploy/smoke-api.sh "$API_URL" "$PAGES_URL"

      - name: Smoke-check the PWA
        if: env.DRY_RUN != 'true'
        env:
          # Deep links answer 404 on Pages until the inbox bug
          # "pages-ignores-the-redirects-spa-rewrite-while-404-html-exist" is
          # fixed; that one check warns instead of failing. Remove with the fix.
          SMOKE_WEB_SPA_WARN: '1'
        run: deploy/smoke-web.sh "$PAGES_URL"
```

- [ ] Lint and check the trigger set:

```sh
actionlint .github/workflows/deploy.yml && echo "actionlint ok"
ruby -ryaml -e 'y = YAML.load_file(".github/workflows/deploy.yml"); on = y["on"] || y[true]; p on.keys.sort; p on["push"]; p y["concurrency"]'
grep -c 'main\]\|harness/\*\*\|pull_request:' .github/workflows/deploy.yml
```

Expected: `actionlint ok`; `["push", "workflow_dispatch"]`, `{"branches"=>["production"]}`, `{"group"=>"deploy", "cancel-in-progress"=>false}`; the count is `1` — the single match is the header comment's "never on harness/** branches"; there is no `push` to `main`, no `harness/**` branch filter and no `pull_request:` trigger (wrangler's `--branch main` is the Pages production alias and does not match). If `actionlint` reports a shellcheck finding, fix the shell (as the containerised-deploy plan did for `SC2034`), never by disabling the rule.

- [ ] Prove the build step works with the variables the runner will have (from `frontend/`; placeholders, nothing is uploaded):

```sh
cd frontend && npm ci && \
NUXT_PUBLIC_API_BASE=https://api.example.test NUXT_PUBLIC_GOOGLE_CLIENT_ID=ci-only NUXT_PUBLIC_VAPID_PUBLIC_KEY=ci-only \
npx nuxi generate && test -f .output/public/index.html && test -f .output/public/sw.js && test -f .output/public/manifest.webmanifest && \
grep -rql 'api.example.test' .output/public && echo "generate ok"
```

Expected: `generate ok` — the same assertion `docker-images` makes for the image.

- [ ] Commit: `ci: deploy workflow ships the API to Railway and the PWA to Cloudflare Pages on push to production`

### Task 2: `deploy/smoke-web.sh` — `SMOKE_WEB_SPA_WARN` knob

**Files:** modify `deploy/smoke-web.sh` (on the base from PR #30).

- [ ] Replace the single line `check "spa fallback (/learn/abc)" "200" "$(status "$web/learn/abc")"` with:

```sh
# SMOKE_WEB_SPA_WARN=1 downgrades only this check to a warning: Cloudflare Pages
# serves the generated 404.html before the _redirects splat, so deep links
# answer 404 there until that fix lands (harness inbox: pages-ignores-the-
# redirects-spa-rewrite-while-404-html-exist). The deploy workflow sets it; the
# docker-images CI job does not, so the Caddy image is still held to 200.
spa=$(status "$web/learn/abc")
if [ "${SMOKE_WEB_SPA_WARN:-}" = "1" ] && [ "$spa" != "200" ]; then
  echo "WARN spa fallback (/learn/abc): expected '200', got '$spa' (SMOKE_WEB_SPA_WARN=1)"
else
  check "spa fallback (/learn/abc)" "200" "$spa"
fi
```

Nothing else in the script changes; the exit code still comes only from `fail`.

- [ ] Prove both behaviours against the live Pages URL (read-only `curl`; the open deep-link bug is what makes this a real test today):

```sh
sh -n deploy/smoke-web.sh && echo parses
deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "strict exit=$?"
SMOKE_WEB_SPA_WARN=1 deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "warn exit=$?"
```

Expected: `parses`; the strict run prints `FAIL spa fallback (/learn/abc): expected '200', got '404'` and `strict exit=1`; the warn run prints `WARN spa fallback (/learn/abc): expected '200', got '404' (SMOKE_WEB_SPA_WARN=1)`, every other line `ok`, and `warn exit=0`. If any *other* line fails, the live site has a different problem — record it in the execution summary, do not widen the knob. (If the deep-link fix has already shipped, both runs exit 0 and no `WARN` line appears; say so.)

- [ ] Commit: `deploy: smoke-web SMOKE_WEB_SPA_WARN downgrades the deep-link check to a warning for Pages`

### Task 3: the nightly ship routine

**Files:** create `.agents/routines/daily-ship.md`; modify `.agents/routines/README.md`.

- [ ] Create `.agents/routines/daily-ship.md` with exactly this content:

```markdown
---
name: daily-ship
schedule: "0 22 * * *"
schedule_utc: "0 15 * * *"
role: .agents/roles/executor.md
skills: []
writes: the remote branch `production` (pure fast-forward of `origin/main` only) — no repository files
pr_title: none — this routine commits nothing and opens no PR; it reports
budget: 30 minutes
---

# Daily ship — 22:00 local

Unattended ship run for the English-Training-Harness repo (GitHub, `gh`). It runs after the 20:00 review run and after the owner's daily merge, and does one thing: if `origin/main` has moved past `production` and its CI is green, fast-forward `production` to it and report the `Deploy` run that push starts (`.github/workflows/deploy.yml` ships the API to Railway and the PWA to Cloudflare Pages, then smoke-checks both). It never merges anything, never pushes `main`, never force-pushes, and changes no files. Make no other choices; report at the end.

1. **Sync** per `.agents/routines/README.md`: abort if the working tree is dirty (a regenerated `harness/STATE.md` may be restored) or if on `main`; then `git fetch origin main production`. Do **not** merge harness PRs here — the owner's daily merge is the gate, and this run ships only what is already on `origin/main`.

2. **Nothing new?** `git rev-parse origin/main origin/production`. If they are equal, stop: report "production is already at `<sha>`; nothing to ship" and end the run. (If `origin/production` does not exist, stop and report — the bootstrap task of the ship plan creates it; never create it here.)

3. **Require green CI on `main`.** `gh run list --workflow CI --branch main --limit 1 --json headSha,status,conclusion,url`. The `headSha` must equal `origin/main` and the run must be `completed` / `success`. Anything else (in progress, failure, cancelled, an older sha) → stop and report the run URL; never ship a red or unproven `main`.

4. **Fast-forward only.** `git merge-base --is-ancestor origin/production origin/main` must succeed; if it fails, `production` has a commit `main` does not — stop and report `git log --oneline origin/main..origin/production` for the owner. Otherwise push the fast-forward: `git push origin origin/main:production` (no `--force`, no merge commit, no local branch needed). This push is the one allowed push to `production` (AGENTS.md).

5. **Watch the Deploy run.** Within a minute `gh run list --workflow deploy.yml --branch production --limit 1 --json databaseId,headSha,status,url` shows a run for the new sha; `gh run watch <id> --exit-status` (the job's `timeout-minutes` is 25). Then `gh run view <id> --log | grep -E '^ *(ok|FAIL|WARN) |healthz:|::error::'` for the smoke lines.

6. **Report**: the old and new `production` sha, the CI run URL that qualified `main`, the Deploy run URL and its conclusion, every `ok`/`FAIL`/`WARN` smoke line (a `WARN spa fallback` line is expected until the Pages deep-link bug is fixed), and anything waiting on the owner (a red Deploy run means the live site is stale or half-shipped — say which step failed). A run that pushed but could not find or watch the Deploy run is a failed run; say so.

**Manual ship** (the owner, any time): the same `git fetch origin main production && git push origin origin/main:production`, or `gh workflow run deploy.yml --ref production` to re-ship the current `production` commit (tick `dry_run` to only build).
```

- [ ] `.agents/routines/README.md`, *The day* table: add the row after `daily-review`:

`| 22:00 | 15:00 | [daily-ship](daily-ship.md) | \`production\` fast-forwarded to \`origin/main\` when its CI is green; one \`Deploy\` run (API + PWA + smoke checks) reported. No PR. |`

- [ ] `.agents/routines/README.md`, *Rules shared by every routine*, append to the **One bookkeeping PR per run** bullet (after "the owner merges."): `The only other branch a routine may push is \`production\`, and only \`daily-ship\` does it, only as a pure fast-forward of \`origin/main\` (\`git merge-base --is-ancestor\` first, never a merge commit, never \`--force\`).`

- [ ] Check: `grep -n 'daily-ship\|production' .agents/routines/README.md` shows the table row and the rule; `sed -n '1,12p' .agents/routines/daily-ship.md` shows the frontmatter with `schedule: "0 22 * * *"` and `schedule_utc: "0 15 * * *"`.

- [ ] Commit: `routines: nightly daily-ship fast-forwards production to main and reports the Deploy run`

### Task 4: docs — runbook, AGENTS.md, CODEMAP

**Files:** modify `deploy/README.md`, `AGENTS.md`, `harness/CODEMAP.md`.

- [ ] `deploy/README.md`, Target A, in the **API on Railway** paragraph replace `new service from the GitHub repo,` with `service \`api\`, **source disconnected from GitHub** (the \`Deploy\` workflow pushes it with the Railway CLI — see *Ship from \`production\`*),` and keep the rest of the paragraph.

- [ ] `deploy/README.md`, Target A, replace the **PWA on Cloudflare Pages** paragraph (it describes a Git-connected project that does not exist) with:

> **PWA on Cloudflare Pages** — project `english-learning`, **direct upload, no Git connection**: the `Deploy` workflow uploads every push to `production` (see *Ship from `production`* below); nobody runs `wrangler pages deploy` by hand any more. The three `NUXT_PUBLIC_*` values are baked in at build time from the repository variables, not set in the Pages dashboard. `frontend/public/_redirects` and `_headers` ship with the build (SPA rewrite; `no-cache` on `sw.js` and the manifest). Known gap: Pages serves the generated `404.html` before the `_redirects` splat, so a deep link still answers 404 with the app shell until the inbox fix lands — the workflow warns on that check instead of failing.

- [ ] `deploy/README.md`, insert a new section between *Target A* and *Target B*:

```markdown
## Ship from `production`

`production` is the long-lived release branch. It only ever moves by a **pure fast-forward of `origin/main`**: the nightly routine `.agents/routines/daily-ship.md` (22:00 local) pushes `origin/main:production` when `main`'s latest CI run is green, and the owner can do the same by hand at any time (`git fetch origin main production && git push origin origin/main:production`). Nothing deploys on a merge to `main`; `main` is where the daily PR lands and CI proves it, `production` is what is live.

`.github/workflows/deploy.yml` runs on every push to `production` (and by hand from the Actions tab — `gh workflow run deploy.yml --ref production`; tick **dry_run** to build without deploying or smoke-checking). In order: a configuration check; `npx -y @railway/cli@latest up . --service api --ci` from the repo root (the service's Root Directory `backend` applies; `--ci` fails the step on a failed build); `npm ci` + `npx nuxi generate` in `frontend/` on Node 20; `npx wrangler@4 pages deploy .output/public --project-name english-learning --branch main`; a wait of up to 2 minutes for `$API_URL/healthz`; `deploy/smoke-api.sh`; `deploy/smoke-web.sh` (with `SMOKE_WEB_SPA_WARN=1`). A failed deploy or smoke check fails the run — a red `Deploy` run means the live site is stale or half-shipped; the failing step says which. One run at a time (`concurrency: deploy`), never cancelled. It never runs on `main`, `harness/**` or pull requests.

Configuration lives in the GitHub repository (Settings → Secrets and variables → Actions, or `gh variable set` / `gh secret set`):

| Kind | Name | Value |
|---|---|---|
| variable | `NUXT_PUBLIC_API_BASE` | `https://<railway-domain>` |
| variable | `NUXT_PUBLIC_GOOGLE_CLIENT_ID` | the OAuth client id |
| variable | `NUXT_PUBLIC_VAPID_PUBLIC_KEY` | the VAPID public key |
| variable | `PAGES_URL` | `https://english-learning-e6a.pages.dev` |
| variable | `API_URL` | `https://<railway-domain>` |
| secret | `CLOUDFLARE_API_TOKEN` | Cloudflare → My Profile → API Tokens → Create Token → *Create Custom Token*, permission **Account → Cloudflare Pages → Edit**, scoped to this account only |
| secret | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare dashboard → Workers & Pages → *Account details* |
| secret | `RAILWAY_TOKEN` | Railway → the project → Settings → Tokens → new **project token** for the `production` environment |

Variables are public values (they end up in the built site anyway); the three secrets are the only credentials and only the workflow holds them — agents never do. A run with any of the eight unset fails in its first step naming what is missing (a dry run only warns). Railway's GitHub auto-deploy from `main` must stay **disconnected** (Service → Settings → Source): the CLI push is the only deploy path, so the API and the PWA always come from the same `production` commit.
```

- [ ] `deploy/README.md`, *Target B* paragraph: replace `Continuous delivery (registry push + deploy webhook on merge to \`main\`) is parked` with `Continuous delivery to Dokploy (registry push + deploy webhook on a push to \`production\`, as a second job in \`deploy.yml\`) is parked`.

- [ ] `deploy/README.md`, *Smoke check* section, append to the `smoke-web.sh` bullet: `Set \`SMOKE_WEB_SPA_WARN=1\` to make only the deep-path check a warning — the deploy workflow does, because of the Pages \`404.html\` gap above; the Docker image is always checked strictly.`

- [ ] `deploy/README.md`, *Owner checklist*: replace the item `Create the Cloudflare Pages project (root \`frontend\`, build \`npx nuxi generate\`, output \`.output/public\`), set the three \`NUXT_PUBLIC_*\` values, note the domain.` with these items:
  - `- [ ] Create the Cloudflare Pages project \`english-learning\` as a **direct-upload** project (no Git connection); note the domain.`
  - `- [ ] On the Railway service \`api\`: Settings → Source → **disconnect** the GitHub repo (the workflow deploys it); Settings → Tokens (project) → create a project token for \`production\`.`
  - `- [ ] Create the Cloudflare API token (custom, **Account → Cloudflare Pages → Edit**) and set the five repository variables and three secrets from *Ship from \`production\`*.`
  - `- [ ] Protect the \`production\` branch (Settings → Branches): block force pushes and deletion; **no** pull-request requirement — the nightly routine pushes fast-forwards directly.`
  - `- [ ] First ship: Actions → **Deploy** → *Run workflow* on \`production\` with \`dry_run\` ticked (green, no \`::warning::\` in *Check configuration*), then \`git push origin origin/main:production\` and watch the run: the Railway step's build log ends in a successful deploy, the Pages step prints a deployment URL, both smoke steps are green (one \`WARN spa fallback\` line is expected until the deep-link bug is fixed). If the Railway build log says it cannot find \`backend/Dockerfile\`, clear the service's Root Directory and file an inbox bug to change the command to \`up ./backend\`.`

  Also change the later item `Run both smoke scripts against the public URLs, then sign in …` to `The Deploy run's smoke steps are green; sign in from a phone, install the PWA, and allow notifications.`

- [ ] `AGENTS.md`, the CI bullet (the line starting `- CI (\`.github/workflows/ci.yml\`)`): append at its end: `\`.github/workflows/deploy.yml\` runs only on \`production\`; the nightly ship routine (\`.agents/routines/daily-ship.md\`) fast-forwards \`main\` into it, and the owner can push it by hand. It ships the API (Railway CLI) and the PWA (Cloudflare Pages) from that commit and smoke-checks both public URLs; its variables and secrets are the owner's (\`deploy/README.md\` "Ship from \`production\`").`

- [ ] `AGENTS.md`, the rule `- Pushing \`harness/*\` branches is allowed without asking. Pushing \`main\` is not.`: append ` Pushing \`production\` is allowed only as a pure fast-forward of \`origin/main\` — by the nightly ship routine, by the owner, and once by the executor's bootstrap task that creates the branch — never as a merge commit or a force push, and never \`main\`.`

- [ ] `harness/CODEMAP.md`, CI section: add, after the `docker-images` bullet and before the `No linter beyond gofmt` paragraph:

```markdown
- **Deploy** (`.github/workflows/deploy.yml`, separate workflow) — the delivery loop. Trigger `push` to the release branch `production` only (never `main`, `harness/**` or PRs) plus `workflow_dispatch` (boolean `dry_run`: build only). `production` moves only by fast-forward of `origin/main` — `.agents/routines/daily-ship.md` at 22:00 local after a green `main` CI run, or the owner by hand — so the workflow does not re-run CI. Job `ship`, in order: configuration check (fails a real run naming any unset variable/secret; a dry run warns), `npx -y @railway/cli@latest up . --service api --ci` with `RAILWAY_TOKEN` (repo root; the service's Root Directory `backend` applies; runs before `npm ci` so `node_modules` is never uploaded), Node 20 `npm ci` + `npx nuxi generate` with `NUXT_PUBLIC_*` from repository variables, `npx --yes wrangler@4 pages deploy .output/public --project-name english-learning --branch main` with `CLOUDFLARE_API_TOKEN`/`CLOUDFLARE_ACCOUNT_ID`, a 2-minute `/healthz` poll on `vars.API_URL`, `deploy/smoke-api.sh`, `deploy/smoke-web.sh` with `SMOKE_WEB_SPA_WARN=1` (deep-link 404 on Pages is a known open bug; the knob goes when it is fixed). `concurrency: deploy`, never cancelled, `timeout-minutes: 25`. Nothing runs on a `harness/*` branch — proof there is `actionlint` and the `frontend` job. Owner setup and the manual ship: `deploy/README.md` "Ship from `production`".
```

  Do **not** touch the section's "Three parallel GitHub Actions jobs" opening — that stale count is a separate inbox bug (`codemap-ci-section-still-says-three-parallel-jobs-after-the-`).

- [ ] Check the edits landed where intended:

```sh
grep -n 'Ship from `production`' deploy/README.md AGENTS.md harness/CODEMAP.md
grep -n 'SMOKE_WEB_SPA_WARN' deploy/README.md harness/CODEMAP.md .github/workflows/deploy.yml deploy/smoke-web.sh
grep -n 'daily-ship' AGENTS.md harness/CODEMAP.md deploy/README.md .agents/routines/README.md
grep -n 'RAILWAY_TOKEN' deploy/README.md .github/workflows/deploy.yml
grep -n 'Pushing `production`' AGENTS.md
```

Expected: the section title in the README heading plus references in the README Target A paragraphs, the AGENTS.md CI bullet and the CODEMAP entry; `SMOKE_WEB_SPA_WARN` in all four files; `daily-ship` in all four; `RAILWAY_TOKEN` in the README table and twice in the workflow; the push exception once.

- [ ] Commit: `docs: ship-from-production runbook, AGENTS.md rules and CODEMAP entry for the deploy workflow`

### Task 5: bootstrap `production`, push, record

- [ ] `python3 tools/harness/cli.py validate` (the `harness-tooling` job runs it too).
- [ ] **Bootstrap the release branch** — the one allowed push outside `harness/*` (Design decision 8; AGENTS.md exception). Only if it does not exist yet:

```sh
git fetch origin main
if [ -z "$(git ls-remote --heads origin production)" ]; then
  git push origin origin/main:refs/heads/production
else
  echo "production already exists at $(git ls-remote --heads origin production | cut -c1-12) — not touched"
fi
git ls-remote --heads origin production main
```

Expected: `production` now exists at the same sha as `origin/main`. It carries no `deploy.yml` yet, so no workflow runs; the first `Deploy` run is the first nightly fast-forward after this plan's daily PR merges. Never push `main`, never force, never push anything else to `production`.

- [ ] Push this plan's branch; read its CI run: `gh run list --branch "$(git branch --show-current)" --limit 3` and, once finished, `gh run view <id>` — all five jobs (`backend-unit`, `backend-integration`, `frontend`, `docker-images`, `harness-tooling`) green. `docker-images` proves the strict smoke script still passes against the Caddy image; `frontend` proves `npm ci` + the toolchain. **No `Deploy` run appears for the branch and none should** (`on.push.branches` is `production` only).
- [ ] Write the execution summary into this plan (`## Execution summary`): the exit codes from Task 2's two live runs, the `actionlint` output, the `production` sha from the bootstrap, the CI run URL, and the sentence "First real Deploy run: owner step after merge — see Verification."

## Verification

Run from the worktree root on the finished branch. Everything here is provable without any token; the end-to-end run is the owner's step at the bottom.

```sh
# 1. Workflow is well-formed, triggers only on production + dispatch, one job, never cancelled.
actionlint .github/workflows/deploy.yml && echo "actionlint ok"
ruby -ryaml -e 'y = YAML.load_file(".github/workflows/deploy.yml"); on = y["on"] || y[true]; p on.keys.sort; p on["push"]; p y["concurrency"]; p y["jobs"].keys'
# expect: ["push", "workflow_dispatch"] / {"branches"=>["production"]} / {"group"=>"deploy", "cancel-in-progress"=>false} / ["ship"]
grep -c 'railway/cli.*up \. --service api --ci' .github/workflows/deploy.yml   # expect 1
grep -c 'wrangler@4 pages deploy' .github/workflows/deploy.yml                  # expect 1

# 2. The PWA build the runner will do, with placeholder public values.
( cd frontend && npm ci && NUXT_PUBLIC_API_BASE=https://api.example.test NUXT_PUBLIC_GOOGLE_CLIENT_ID=ci-only NUXT_PUBLIC_VAPID_PUBLIC_KEY=ci-only npx nuxi generate \
  && test -f .output/public/index.html && test -f .output/public/sw.js && grep -rql 'api.example.test' .output/public && echo "generate ok" )

# 3. The smoke knob: strict fails on the live deep link today, warn passes.
sh -n deploy/smoke-web.sh && echo parses
deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "strict exit=$?"        # expect FAIL spa line, exit=1 (exit=0 once the 404 fix ships)
SMOKE_WEB_SPA_WARN=1 deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "warn exit=$?"   # expect WARN spa line, exit=0

# 4. Routine and docs.
sed -n '1,10p' .agents/routines/daily-ship.md                     # schedule "0 22 * * *", schedule_utc "0 15 * * *"
grep -c 'merge-base --is-ancestor' .agents/routines/daily-ship.md # expect >= 1
grep -n 'daily-ship' .agents/routines/README.md AGENTS.md | wc -l # expect >= 2
grep -n 'Ship from `production`' deploy/README.md AGENTS.md harness/CODEMAP.md | wc -l   # expect >= 4
grep -n 'Pushing `production`' AGENTS.md | wc -l                  # expect 1

# 5. Release branch, harness artifacts and CI on the pushed branch.
git ls-remote --heads origin production                           # exists; sha == origin/main at bootstrap time
python3 tools/harness/cli.py validate
gh run list --branch "$(git branch --show-current)" --limit 1    # CI: completed success; no Deploy run for this branch
git diff --stat origin/main...HEAD                                # only deploy.yml, deploy/smoke-web.sh, deploy/README.md, AGENTS.md, harness/CODEMAP.md, .agents/routines/{daily-ship.md,README.md}, this plan
```

**Owner step (after the daily PR merges — the only true end-to-end proof, and it cannot happen on a branch):**
1. Disconnect the Railway service's GitHub source; create the Railway project token, the Cloudflare token, and set the five variables + three secrets from `deploy/README.md` "Ship from `production`"; protect `production`.
2. Actions → **Deploy** → *Run workflow* on `production` with **dry_run** ticked: green, and its *Check configuration* step prints no `::warning::` (if it does, that item is still unset).
3. `git fetch origin main production && git push origin origin/main:production` (or wait for the 22:00 routine): the `Deploy` run's Railway step ends in a successful build, the Pages step prints a deployment URL, both smoke steps are green (one `WARN spa fallback` line until the deep-link bug is fixed).
4. From then on `gh run list --workflow deploy.yml --branch production` shows one run per ship; a red one means the live site is stale or half-shipped and the failing step says why.

## Notes

- **Do not** add `main` or `harness/**` to `on.push.branches` "to test it on the branch": it would deploy every branch build to production. The branch proof is `actionlint` + the `frontend` job; the dispatch dry run is the owner's first real proof.
- The Dokploy CD backlog entry (`harness/BACKLOG.md`) becomes a second job in this same file (GHCR push + webhook, also on `production`) when Target B exists; keep `ship` self-contained so that job can sit beside it.
- Until the owner disconnects Railway's GitHub source, a `main` merge still rebuilds the API once and the nightly ship rebuilds it again from the same commit — harmless duplication, not a failure.
- The Railway step waits for the build (`--ci`), not for the health-checked swap; the `/healthz` poll and `smoke-api.sh` prove the public URL is healthy, and Railway's own healthcheck (`/healthz`, Target A) gates the swap. A `200` from the previous deployment during the swap is therefore possible; a bad new build never replaces a good one.
