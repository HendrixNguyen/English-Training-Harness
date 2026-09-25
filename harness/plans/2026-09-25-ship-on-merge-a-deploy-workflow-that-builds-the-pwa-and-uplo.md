---
idea: harness/ideas/2026-09-25-run-01/ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md
status: approved
priority: high
merged: false
---
# Ship on merge: a deploy workflow that builds the PWA and uploads it to Cloudflare Pages on every push to main, then smoke-checks both public URLs — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md`

**Goal:** After the owner merges the daily PR, the PWA ships by itself: a `Deploy` workflow runs when the `CI` workflow finishes green for a push to `main`, builds `frontend/` with `nuxi generate`, uploads `.output/public` to the Cloudflare Pages project `english-learning`, waits for the Railway API, and runs both smoke scripts against the public URLs — failing visibly on a bad upload or smoke check, and never running on `harness/**` branches or pull requests.

**Why now (`priority: high`):** The Pages project has no Git connection (`wrangler pages project list` → `Git Provider: No`); both live deploys were manual `wrangler pages deploy` uploads from one machine. Railway rebuilds the API from `main` on every push, so a merged daily PR moves the API and leaves the PWA behind — the owner's only gate ships half the product. No app code changes; CI/infra only, so no design doc.

**Base branch facts (checked 2026-09-25 against `origin/main` = `68529ad`):** PR #30 is merged, so `deploy/smoke-web.sh`, `deploy/smoke-api.sh`, `deploy/README.md`, both Dockerfiles and the `docker-images` CI job are on the base. The workflow calls the real scripts; there is no inline-`curl` fallback. If — unexpectedly — `test -f deploy/smoke-web.sh` is false on your freshly fetched base, **stop and report**; do not merge or cherry-pick anything. `frontend/public/_redirects` + `_headers` are on the base too (login-308 fix), but the inbox bug `pages-ignores-the-redirects-spa-rewrite-while-404-html-exist` is still open: `GET /learn/abc` on Pages is 404 with the shell, so `deploy/smoke-web.sh` fails that one line against the live URL. This plan downgrades that line to a warning (Task 2) rather than fixing it.

**Design decisions:**
1. **Trigger = `workflow_run` on `CI` + `workflow_dispatch`.** The `CI` workflow is what the repo trusts; a deploy that ran on `push` in parallel would race it. The job's `if:` keeps only `conclusion == 'success' && event == 'push' && head_branch == 'main'` — `workflow_run` fires for every completed CI run, including PRs and `harness/**` pushes, and the guard is what excludes them. The job checks out `workflow_run.head_sha`, the exact commit CI proved (on dispatch: `github.sha`).
2. **`workflow_dispatch` has a boolean `dry_run` input** (default `false`): build the site, skip the upload, the API wait and both smoke checks. It is the owner's first, safe run after merge and the way to check the variables.
3. **`concurrency: { group: deploy, cancel-in-progress: false }`** — one deploy at a time, never cancelled: a cancelled upload can leave Pages on an older build than the API.
4. **Public values are repository *variables*, credentials are *secrets*.** `vars.NUXT_PUBLIC_API_BASE`, `vars.NUXT_PUBLIC_GOOGLE_CLIENT_ID`, `vars.NUXT_PUBLIC_VAPID_PUBLIC_KEY` feed `nuxi generate` (Nuxt reads `NUXT_PUBLIC_*` at build for a static site); `vars.PAGES_URL` / `vars.API_URL` feed the smoke checks; `secrets.CLOUDFLARE_API_TOKEN` (custom token, permission *Account → Cloudflare Pages → Edit*) and `secrets.CLOUDFLARE_ACCOUNT_ID` feed wrangler. A first step fails a real run with one line naming every unset item; a dry run only warns. The repo has none of these today (`gh variable list` / `gh secret list` are empty) — creating them is the owner's checklist, agents never hold the token.
5. **Deep-link 404 is a warning, not a failure.** `deploy/smoke-web.sh` gains an opt-in `SMOKE_WEB_SPA_WARN=1` that turns only the `spa fallback (/learn/abc)` line into `WARN` without touching the exit code; the deploy workflow sets it, the `docker-images` job does not, so the Caddy image stays strictly checked. When the inbox fix lands, its plan removes the `env:` line from the workflow.
6. **Railway needs nothing.** It already rebuilds `main` on push; the job polls `$API_URL/healthz` for up to 2 minutes (24 × 5 s) before `deploy/smoke-api.sh`, because Railway's build started at the push while CI ran. Note for the owner: the API currently runs a CLI-uploaded build; the first push to `main` after this merges replaces it with the Railway build of the same commit.
7. **`npx --yes wrangler@4`**, not a pinned action: it is the same command the runbook documents for the manual upload, so the workflow and the runbook cannot drift, and `--yes` is what keeps `npx` from prompting on a non-TTY runner.
8. **Node 20**, the version the runbook names for Pages; CI's `frontend` job stays on `lts/*`.

**Tech stack:** GitHub Actions (`actions/checkout@v7`, `actions/setup-node@v7`, as in `ci.yml`), `wrangler@4`, POSIX `sh` (the smoke scripts are `#!/bin/sh`), `actionlint` (installed at `/opt/homebrew/bin/actionlint`). No new repository dependencies.

**Run every command from the worktree root** unless a step says `frontend/`. `rg` is not installed — use `grep -n`. This session's shell proxy can mangle `ls` output; use globs or `find`.

---

## File structure

| Path | Change |
|---|---|
| `.github/workflows/deploy.yml` | **New.** The `Deploy` workflow (Task 1). |
| `deploy/smoke-web.sh` | `SMOKE_WEB_SPA_WARN` knob around the one deep-link check (Task 2). |
| `deploy/README.md` | Correct the "PWA on Cloudflare Pages" paragraph, add **Ship on merge**, a Smoke-check note and owner-checklist items (Task 3). |
| `AGENTS.md` | CI paragraph gains the delivery loop sentence (Task 3). |
| `harness/CODEMAP.md` | CI section gains the `deploy.yml` entry (Task 3). |

---

## Tasks

### Task 1: `.github/workflows/deploy.yml`

**Files:** create `.github/workflows/deploy.yml`.

- [ ] Write the file exactly as below. Keep the comments — they are the only place the trigger guard is explained.

```yaml
name: Deploy

# Delivery loop (AGENTS.md; deploy/README.md "Ship on merge"): build the PWA
# and upload it to Cloudflare Pages once CI is green for a push to main, then
# smoke-check both public URLs. Never runs for harness/** or pull requests —
# workflow_run fires for every completed CI run, and the job's `if` is what
# keeps only a green *push* run on main. workflow_run only fires for workflow
# files on the default branch, so this file does nothing until it is merged.
on:
  workflow_run:
    workflows: [CI]
    types: [completed]
  workflow_dispatch:
    inputs:
      dry_run:
        description: Build the PWA but skip the Pages upload and the smoke checks
        type: boolean
        default: false

permissions:
  contents: read

# One deploy at a time and never cancelled: a cancelled upload can leave Pages
# on an older build than the API Railway just rebuilt.
concurrency:
  group: deploy
  cancel-in-progress: false

jobs:
  pages:
    if: >-
      github.event_name == 'workflow_dispatch' ||
      (github.event.workflow_run.conclusion == 'success' &&
       github.event.workflow_run.event == 'push' &&
       github.event.workflow_run.head_branch == 'main')
    runs-on: ubuntu-latest
    timeout-minutes: 15
    env:
      DRY_RUN: ${{ github.event_name == 'workflow_dispatch' && inputs.dry_run == true }}
      PAGES_URL: ${{ vars.PAGES_URL }}
      API_URL: ${{ vars.API_URL }}
      # Public build-time values (Nuxt reads NUXT_PUBLIC_* while generating).
      NUXT_PUBLIC_API_BASE: ${{ vars.NUXT_PUBLIC_API_BASE }}
      NUXT_PUBLIC_GOOGLE_CLIENT_ID: ${{ vars.NUXT_PUBLIC_GOOGLE_CLIENT_ID }}
      NUXT_PUBLIC_VAPID_PUBLIC_KEY: ${{ vars.NUXT_PUBLIC_VAPID_PUBLIC_KEY }}
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v7
        with:
          # The exact commit CI proved; on a manual run, the dispatched ref.
          ref: ${{ github.event.workflow_run.head_sha || github.sha }}

      - name: Check configuration
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
        run: |
          missing=""
          for v in NUXT_PUBLIC_API_BASE NUXT_PUBLIC_GOOGLE_CLIENT_ID NUXT_PUBLIC_VAPID_PUBLIC_KEY \
                   PAGES_URL API_URL CLOUDFLARE_API_TOKEN CLOUDFLARE_ACCOUNT_ID; do
            [ -n "${!v:-}" ] || missing="$missing $v"
          done
          if [ -n "$missing" ]; then
            if [ "$DRY_RUN" = "true" ]; then
              echo "::warning::dry run with unset:$missing — see deploy/README.md 'Ship on merge'"
            else
              echo "::error::not configured:$missing — see deploy/README.md 'Ship on merge'"
              exit 1
            fi
          fi
          echo "dry_run=$DRY_RUN"

      - uses: actions/setup-node@v7
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: frontend/package-lock.json

      - name: Install
        run: npm ci

      - name: Generate the static site
        run: |
          npx nuxi generate
          test -f .output/public/index.html
          test -f .output/public/sw.js
          test -f .output/public/manifest.webmanifest

      - name: Upload to Cloudflare Pages
        if: env.DRY_RUN != 'true'
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
        run: >-
          npx --yes wrangler@4 pages deploy .output/public
          --project-name english-learning --branch main

      - name: Wait for the API (Railway rebuilds main on the same push)
        if: env.DRY_RUN != 'true'
        working-directory: .
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
        working-directory: .
        run: deploy/smoke-api.sh "$API_URL" "$PAGES_URL"

      - name: Smoke-check the PWA
        if: env.DRY_RUN != 'true'
        working-directory: .
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
python3 - <<'PY' 2>/dev/null || ruby -ryaml -e 'y = YAML.load_file(".github/workflows/deploy.yml"); p y[true] ? y[true].keys.sort : y["on"].keys.sort; p y["concurrency"]'
import yaml, sys
y = yaml.safe_load(open(".github/workflows/deploy.yml"))
on = y.get("on", y.get(True))
print(sorted(on.keys())); print(y["concurrency"])
PY
grep -c "harness/\*\*\|pull_request" .github/workflows/deploy.yml
```

Expected: `actionlint ok`; `['workflow_dispatch', 'workflow_run']` and `{'group': 'deploy', 'cancel-in-progress': False}`; the last count is `1` — the only match is the comment line explaining the exclusion (there is no `push:` or `pull_request:` trigger). If `actionlint` reports a shellcheck finding, fix the shell (as the containerised-deploy plan did for `SC2034`), never by disabling the rule.

- [ ] Prove the build step works with the variables the runner will have (from `frontend/`; values are placeholders, nothing is uploaded):

```sh
cd frontend && npm ci && \
NUXT_PUBLIC_API_BASE=https://api.example.test NUXT_PUBLIC_GOOGLE_CLIENT_ID=ci-only NUXT_PUBLIC_VAPID_PUBLIC_KEY=ci-only \
npx nuxi generate && test -f .output/public/index.html && test -f .output/public/sw.js && test -f .output/public/manifest.webmanifest && \
grep -rql 'api.example.test' .output/public && echo "generate ok"
```

Expected: `generate ok` — the same assertion `docker-images` makes for the image (`NUXT_PUBLIC_API_BASE` reaches the generated site).

- [ ] Commit: `ci: deploy workflow builds the PWA and uploads it to Cloudflare Pages after a green CI run on main`

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

- [ ] Prove both behaviours against the live Pages URL (read-only `curl`; the deep-link bug is what makes this a real test today):

```sh
sh -n deploy/smoke-web.sh && echo parses
deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "strict exit=$?"
SMOKE_WEB_SPA_WARN=1 deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "warn exit=$?"
```

Expected: `parses`; the strict run prints `FAIL spa fallback (/learn/abc): expected '200', got '404'` and `strict exit=1`; the warn run prints `WARN spa fallback (/learn/abc): expected '200', got '404' (SMOKE_WEB_SPA_WARN=1)`, every other line `ok`, and `warn exit=0`. If any *other* line fails, the live site has a different problem — record it in the execution summary, do not widen the knob. (If the deep-link fix has already shipped when you run this, both runs exit 0 and no `WARN` line appears; say so.)

- [ ] Commit: `deploy: smoke-web SMOKE_WEB_SPA_WARN downgrades the deep-link check to a warning for Pages`

### Task 3: docs — runbook, AGENTS.md, CODEMAP

**Files:** modify `deploy/README.md`, `AGENTS.md`, `harness/CODEMAP.md`.

- [ ] `deploy/README.md`, Target A, replace the **PWA on Cloudflare Pages** paragraph (it describes a Git-connected project that does not exist) with:

> **PWA on Cloudflare Pages** — project `english-learning`, **direct upload, no Git connection**: `.github/workflows/deploy.yml` uploads every green push to `main` (see *Ship on merge* below); nobody runs `wrangler pages deploy` by hand any more. The three `NUXT_PUBLIC_*` values are baked in at build time from the repository variables, not set in the Pages dashboard. `frontend/public/_redirects` and `_headers` ship with the build (SPA rewrite; `no-cache` on `sw.js` and the manifest). Known gap: Pages serves the generated `404.html` before the `_redirects` splat, so a deep link still answers 404 with the app shell until the inbox fix lands — the deploy workflow warns on that check instead of failing.

- [ ] `deploy/README.md`, insert a new section between *Target A* and *Target B*:

```markdown
## Ship on merge

`.github/workflows/deploy.yml` runs when the `CI` workflow completes **green for a push to `main`** (`workflow_run`), and by hand from the Actions tab (`workflow_dispatch`; tick **dry_run** to build without uploading or smoke-checking). It checks out the commit CI proved, `npm ci` + `npx nuxi generate` in `frontend/` on Node 20, uploads `.output/public` with `npx wrangler@4 pages deploy --project-name english-learning --branch main`, waits up to 2 minutes for `$API_URL/healthz` (Railway rebuilds `main` on the same push), then runs `deploy/smoke-api.sh` and `deploy/smoke-web.sh` against the public URLs. A failed upload or smoke check fails the run. It never runs for `harness/**` branches or pull requests, and one deploy runs at a time (`concurrency: deploy`, never cancelled).

Configuration lives in the GitHub repository, not in the repo files:

| Kind | Name | Value |
|---|---|---|
| variable | `NUXT_PUBLIC_API_BASE` | `https://<railway-domain>` |
| variable | `NUXT_PUBLIC_GOOGLE_CLIENT_ID` | the OAuth client id |
| variable | `NUXT_PUBLIC_VAPID_PUBLIC_KEY` | the VAPID public key |
| variable | `PAGES_URL` | `https://english-learning-e6a.pages.dev` |
| variable | `API_URL` | `https://<railway-domain>` |
| secret | `CLOUDFLARE_API_TOKEN` | Cloudflare → My Profile → API Tokens → Create Token → *Create Custom Token*, permission **Account → Cloudflare Pages → Edit**, scoped to this account only |
| secret | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare dashboard → Workers & Pages → *Account details* |

Variables are public values (they end up in the built site anyway); the two secrets are the only credentials, and only the workflow holds them — agents never do. A run with any of the seven unset fails in its first step naming what is missing (a dry run only warns).
```

- [ ] `deploy/README.md`, *Smoke check* section, append to the `smoke-web.sh` bullet: `Set \`SMOKE_WEB_SPA_WARN=1\` to make only the deep-path check a warning — the deploy workflow does, because of the Pages \`404.html\` gap above; the Docker image is always checked strictly.`

- [ ] `deploy/README.md`, *Owner checklist*, replace the item `Create the Cloudflare Pages project (root \`frontend\`, build \`npx nuxi generate\`, output \`.output/public\`), set the three \`NUXT_PUBLIC_*\` values, note the domain.` with two items:
  - `- [ ] Create the Cloudflare Pages project \`english-learning\` as a **direct-upload** project (no Git connection); note the domain.`
  - `- [ ] Create the Cloudflare API token (custom, **Account → Cloudflare Pages → Edit**) and set the five repository variables and two secrets from *Ship on merge* (\`gh variable set\`, \`gh secret set\`, or Settings → Secrets and variables → Actions). Then run **Deploy** by hand once with \`dry_run\` ticked, then once for real, and check both smoke steps are green.`

  Also change the later item `Run both smoke scripts against the public URLs, then sign in …` to `The Deploy run's smoke steps are green; sign in from a phone, install the PWA, and allow notifications.`

- [ ] `AGENTS.md`, the CI bullet (the line starting `- CI (\`.github/workflows/ci.yml\`)`): append one sentence at its end: `The delivery loop is \`.github/workflows/deploy.yml\`: it runs only when a \`CI\` run for a push to \`main\` completes green (\`workflow_run\`) or on a manual \`workflow_dispatch\`, builds the PWA and uploads it to Cloudflare Pages, then smoke-checks both public URLs; it never runs for \`harness/**\` or pull requests, and its variables and secrets are the owner's (\`deploy/README.md\` "Ship on merge").`

- [ ] `harness/CODEMAP.md`, CI section: add, after the `docker-images` bullet and before the `No linter beyond gofmt` paragraph:

```markdown
- **Deploy** (`.github/workflows/deploy.yml`, separate workflow) — the delivery loop. Trigger `workflow_run` on `CI` `completed` plus `workflow_dispatch` (boolean `dry_run`: build only); the job's `if` keeps only `conclusion == success && event == push && head_branch == main`, so PR and `harness/**` runs never deploy, and it checks out `workflow_run.head_sha`. Steps: configuration check (fails a real run naming any unset variable/secret; a dry run warns), Node 20 `npm ci` + `npx nuxi generate` with `NUXT_PUBLIC_*` from repository variables, `npx --yes wrangler@4 pages deploy .output/public --project-name english-learning --branch main` with `CLOUDFLARE_API_TOKEN`/`CLOUDFLARE_ACCOUNT_ID` secrets, a 2-minute `/healthz` poll on `vars.API_URL`, `deploy/smoke-api.sh`, `deploy/smoke-web.sh` with `SMOKE_WEB_SPA_WARN=1` (deep-link 404 on Pages is a known open bug; the knob goes when it is fixed). `concurrency: deploy`, never cancelled. `workflow_run` only fires for the default branch's copy of the file, so nothing runs on a `harness/*` branch — proof there is `actionlint` and the `frontend` job. Owner setup: `deploy/README.md` "Ship on merge".
```

  Do **not** touch the section's "Three parallel GitHub Actions jobs" opening — that stale count is a separate inbox bug (`codemap-ci-section-still-says-three-parallel-jobs-after-the-`).

- [ ] Check the edits landed where intended:

```sh
grep -n 'Ship on merge' deploy/README.md AGENTS.md
grep -n 'SMOKE_WEB_SPA_WARN' deploy/README.md harness/CODEMAP.md .github/workflows/deploy.yml deploy/smoke-web.sh
grep -c 'direct upload\|direct-upload' deploy/README.md
grep -n 'deploy.yml' AGENTS.md harness/CODEMAP.md
```

Expected: `Ship on merge` appears in the README heading, the README Target A paragraph, the AGENTS.md CI bullet and the CODEMAP entry; `SMOKE_WEB_SPA_WARN` in all four files; the README count is `2`; `deploy.yml` once in AGENTS.md and once in CODEMAP.

- [ ] Commit: `docs: ship-on-merge runbook, AGENTS.md and CODEMAP entries for the deploy workflow`

### Task 4: push and record

- [ ] `python3 tools/harness/cli.py validate` (the `harness-tooling` job runs it too).
- [ ] Push the branch; read its CI run: `gh run list --branch "$(git branch --show-current)" --limit 3` and, once finished, `gh run view <id>` — all five jobs (`backend-unit`, `backend-integration`, `frontend`, `docker-images`, `harness-tooling`) green. `docker-images` proves the strict smoke script still passes against the Caddy image; `frontend` proves `npm ci` + the toolchain on the runner. **No `Deploy` run appears for the branch and none should** (see Design decision 1).
- [ ] Write the execution summary into this plan (`## Execution summary`): the exit codes from Task 2's two live runs, the `actionlint` output, the CI run URL, and the sentence "First real run: owner step after merge — see Verification."

## Verification

Run from the worktree root on the finished branch. Everything here is provable without the Cloudflare token; the end-to-end run is the owner's step at the bottom.

```sh
# 1. Workflow is well-formed and has only the two intended triggers.
actionlint .github/workflows/deploy.yml && echo "actionlint ok"
ruby -ryaml -e 'y = YAML.load_file(".github/workflows/deploy.yml"); on = y["on"] || y[true]; p on.keys.sort; p y["concurrency"]; p y["jobs"]["pages"]["if"].include?("head_branch == '"'"'main'"'"'")'
# expect: ['workflow_dispatch', 'workflow_run'] / {"group"=>"deploy", "cancel-in-progress"=>false} / true

# 2. The build the runner will do, with placeholder public values.
( cd frontend && npm ci && NUXT_PUBLIC_API_BASE=https://api.example.test NUXT_PUBLIC_GOOGLE_CLIENT_ID=ci-only NUXT_PUBLIC_VAPID_PUBLIC_KEY=ci-only npx nuxi generate \
  && test -f .output/public/index.html && test -f .output/public/sw.js && grep -rql 'api.example.test' .output/public && echo "generate ok" )

# 3. The smoke knob: strict fails on the live deep link today, warn passes.
sh -n deploy/smoke-web.sh && echo parses
deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "strict exit=$?"        # expect FAIL spa line, exit=1 (exit=0 once the 404 fix ships)
SMOKE_WEB_SPA_WARN=1 deploy/smoke-web.sh https://english-learning-e6a.pages.dev; echo "warn exit=$?"   # expect WARN spa line, exit=0

# 4. Docs.
grep -n 'Ship on merge' deploy/README.md AGENTS.md | wc -l      # expect 4
grep -n 'deploy.yml' AGENTS.md harness/CODEMAP.md | wc -l        # expect 2

# 5. Harness artifacts and CI on the pushed branch.
python3 tools/harness/cli.py validate
gh run list --branch "$(git branch --show-current)" --limit 1    # CI: completed success; no Deploy run for this branch
git diff --stat origin/main...HEAD                                # only deploy.yml, deploy/smoke-web.sh, deploy/README.md, AGENTS.md, harness/CODEMAP.md, this plan
```

**Owner step (after the daily PR merges — the only true end-to-end proof, and it cannot happen on a branch):**
1. Create the Cloudflare token and set the five variables + two secrets from `deploy/README.md` "Ship on merge".
2. Actions → **Deploy** → *Run workflow* on `main` with **dry_run** ticked: the run is green and its *Check configuration* step prints no `::warning::` (if it does, that variable is still unset).
3. Run it again with `dry_run` unticked: *Upload to Cloudflare Pages* shows a new deployment URL, *Smoke-check the API* and *Smoke-check the PWA* are green; the PWA step shows one `WARN spa fallback` line until the deep-link bug is fixed.
4. From then on, every merged daily PR shows a `Deploy` run next to its `CI` run on `main` (`gh run list --workflow Deploy --branch main`). If a `Deploy` run is red, the site is stale; the run's failing step says why.

## Notes

- **Do not** add a `push:` trigger "to test it on the branch": it would upload every `harness/**` build to production. The branch proof is `actionlint` + the `frontend` job; the dispatch dry run is the owner's first real proof.
- The Dokploy CD backlog entry (`harness/BACKLOG.md`) becomes a second job in this same file (GHCR push + webhook) when Target B exists; keep `pages` self-contained so that job can sit beside it.
- The API currently on Railway is a CLI-uploaded build; the first push to `main` after this merges replaces it with Railway's own build of the same commit. Nothing to do, but expect the `/healthz` wait to be exercised on that run.
