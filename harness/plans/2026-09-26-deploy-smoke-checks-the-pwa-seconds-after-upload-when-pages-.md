---
idea: harness/ideas/_inbox/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md
status: done
priority: high
merged: false
branch: harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-
worktree: .worktrees/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-
---
# Pages ship is honest: wait for the edge before the smoke check, no `404.html` on Pages so deep links answer 200, and the `/login` return-trip check — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B4** of 2026-09-26. **Estimate:** 2.5 h. **Branch:** `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`.

**Idea (head):** `harness/ideas/_inbox/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md`
**Also planned here (their frontmatter points at this plan):**
- `harness/ideas/_inbox/pages-ignores-the-redirects-spa-rewrite-while-404-html-exist.md` → Task 2
- `harness/ideas/_inbox/task-4-follow-up-smoke-web-has-no-login-return-trip-check-an.md` → Task 3
- `harness/ideas/_inbox/frontend-dist-symlink-left-by-nuxi-generate-is-not-gitignore.md` → Task 2 Step 3b

**Goal:** A green ship on `production` reports green: the Deploy workflow waits (≤ 90 s) until Pages' edge serves the new build's hashed asset as immutable **and** a deep link as 200, then runs the unchanged strict smoke check; the Pages upload contains no `404.html`, so Pages' implicit SPA mode makes `/learn/abc` a 200 and the `SMOKE_WEB_SPA_WARN` downgrade is deleted; `deploy/smoke-web.sh` also proves the `/login?code&state` return trip is a 200 with no redirect; the runbook says what Pages really needs.

**Architecture:** `deploy/` + `.github/workflows/deploy.yml` + one `frontend/package.json` script. No app code, no screen — **no design doc** (nothing a learner sees changes; `frontend/nuxt.config.ts` is untouched: the `404.html` is dropped from the Pages upload by the generate step, so the Caddy image keeps its file and its `try_files`).

**Not verifiable live by the executor** (no Pages/Railway secrets in a worktree): the live proof is the next nightly Deploy run; the plan's Verification records exactly what to read there. Everything else is proved locally (the script against the Caddy image in Docker, the workflow through `actionlint` if installed, else a YAML parse).

**⚠ Same-day conflict note:** ticket B5 (`harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md`) appends two checks at the **end** of `deploy/smoke-web.sh` and edits the runbook's *Environment* table and Target B. This plan edits the middle of `smoke-web.sh` (the SPA block, the login check after the `index` check) and the Target A "PWA on Cloudflare Pages" paragraph + *Smoke check* bullet. Keep to those regions; if `origin/main` already has B5 when you start, merge it first and keep both sides.

## Global Constraints
- `rg`/`timeout` not installed: `grep -n`, `curl --max-time`. Never skip, loosen or delete a check to make the workflow pass (AGENTS.md CI rule) — the wait **falls through** to the same strict check.
- The wait step never exits non-zero on its own; it prints elapsed seconds and the last observed values.
- `deploy/smoke-web.sh` stays POSIX `sh` with `set -eu` and the `check`/`status`/`cache_control` helpers; every check prints `ok`/`FAIL` (it must keep working for the `docker-images` CI job, which runs it against the Caddy image at `http://127.0.0.1:18081`).

## Review Focus
1. The wait polls the two signals the smoke check actually asserts — first `/_nuxt/*.js` from `index.html` has `Cache-Control: public, max-age=31536000, immutable` **and** `GET /learn/abc` is `200` — every 5 s for at most 90 s, then falls through.
2. With `404.html` absent from the upload, `frontend/public/_redirects` is still shipped (harmless, documented as belt-and-braces); the runbook states the real rule: **no `404.html` on Pages**.
3. `SMOKE_WEB_SPA_WARN` is gone from both the workflow and the script — the deep-link check is strict everywhere.
4. `check "login return trip (/login?code&state)" "200"` uses `status` (no `-L`), so any 3xx fails.
5. The generate step removes `404.html` only for the Pages upload path; `npx nuxi generate` on its own (Dockerfile, CI `frontend` job) is unchanged.

## File structure

| Path | Change |
| --- | --- |
| `.github/workflows/deploy.yml` | "Generate the static site" removes `.output/public/404.html`; new step "Wait for the Pages edge" before "Smoke-check the PWA"; `SMOKE_WEB_SPA_WARN` env + comment deleted |
| `deploy/smoke-web.sh` | SPA block back to a plain strict `check`; new login return-trip check; comments updated |
| `deploy/README.md` | Target A Pages paragraph rewritten (no `404.html`; `_redirects`/`_headers` shipped; `autoSubfolderIndex: false` and why; login check); *Smoke check* bullet |
| `harness/CODEMAP.md` | Deploy bullet: the wait and the 404.html rule |

## Tasks

### Task 1: The edge wait in `deploy.yml`

**Files:** `.github/workflows/deploy.yml` (steps around lines 115–165).

- [ ] **Step 1:** Insert, between "Smoke-check the API" and "Smoke-check the PWA":

```yaml
      - name: Wait for the Pages edge
        if: env.DRY_RUN != 'true'
        # A fresh Pages deployment serves its hashed assets as no-store for a
        # short window (ship of 2026-09-25 failed on exactly that, then passed
        # two minutes later). Wait, bounded, for the two signals the smoke
        # check asserts; on timeout fall through and let it fail honestly.
        run: |
          start=$(date +%s)
          for _ in $(seq 1 18); do
            asset=$(curl -sS --max-time 10 "$PAGES_URL/" | grep -o '/_nuxt/[^"]*\.js' | head -1 || true)
            cc=$(curl -sS --max-time 10 -I "$PAGES_URL$asset" | tr -d '\r' | awk -F': ' 'tolower($1)=="cache-control"{print $2}' || true)
            spa=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' "$PAGES_URL/learn/abc" || true)
            if [ -n "$asset" ] && [ "$cc" = "public, max-age=31536000, immutable" ] && [ "$spa" = "200" ]; then
              echo "edge ready after $(( $(date +%s) - start ))s"; exit 0
            fi
            sleep 5
          done
          echo "::warning::edge not ready after $(( $(date +%s) - start ))s (asset='$asset' cache-control='$cc' /learn/abc=$spa) — running the smoke check anyway"
```
- [ ] **Step 2:** Delete the `env: SMOKE_WEB_SPA_WARN: '1'` block and its comment from "Smoke-check the PWA".
- [ ] **Step 3:** `actionlint .github/workflows/deploy.yml` if installed, else `python3 -c 'import yaml,sys; yaml.safe_load(open(".github/workflows/deploy.yml"))'`. Commit: `deploy: wait (≤90 s) for the Pages edge before the PWA smoke check`.

### Task 2: No `404.html` on Pages

**Files:** `.github/workflows/deploy.yml` ("Generate the static site" step), `frontend/package.json`, `deploy/smoke-web.sh`, `frontend/public/_redirects` (comment only).

- [ ] **Step 1:** `frontend/package.json` scripts: `"generate:pages": "nuxi generate && rm -f .output/public/404.html"` — one command the workflow calls, so the rule lives next to the build. The "Generate the static site" step runs `npm run generate:pages` instead of `npx nuxi generate` (keep its env/build-args as they are) and adds `test ! -f .output/public/404.html` right after.
- [ ] **Step 2:** `deploy/smoke-web.sh`: replace the `SMOKE_WEB_SPA_WARN` block with the plain `check "spa fallback (/learn/abc)" "200" "$(status "$web/learn/abc")"` and a two-line comment: on Pages this holds because the upload carries no `404.html` (implicit SPA mode); in the Caddy image because of `try_files`.
- [ ] **Step 3:** `frontend/public/_redirects` comment: "Pages ignores this rule while a `404.html` exists; the Deploy workflow removes `404.html` (`npm run generate:pages`), and this file stays as belt-and-braces."
- [ ] **Step 3b (folded: `frontend-dist-symlink-left-by-nuxi-generate-is-not-gitignore`):** `nuxi generate` leaves `frontend/dist` as a **symlink** to `.output/public`; the root `.gitignore` has `dist/`, which matches directories only, so `git status` shows `?? frontend/dist`. Change the rule to `dist` (no slash). Proof: after generate, `git status --short` is empty and `git check-ignore -v frontend/dist` prints the rule.
- [ ] **Step 4:** Local proof: `cd frontend && npm ci && npm run generate:pages && test ! -f .output/public/404.html && ls .output/public/200.html index.html`; then `npx nuxi generate && test -f .output/public/404.html` (plain generate unchanged). Commit: `deploy: Pages upload carries no 404.html so implicit SPA mode answers deep links with 200`.

### Task 3: Login return-trip check and the runbook

**Files:** `deploy/smoke-web.sh`, `deploy/README.md`, `harness/CODEMAP.md`.

- [ ] **Step 1:** `smoke-web.sh`, right after the `index` check: `check "login return trip (/login?code&state)" "200" "$(status "$web/login?code=x&state=y")"` with the comment "no -L: the 2026-09-25 outage was a 308 to /login/ that dropped the OAuth query".
- [ ] **Step 2:** Prove it locally against the Caddy image: `docker build -t aelp-web:pages --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend && docker run -d --rm --name web-pages -p 18082:80 aelp-web:pages && deploy/smoke-web.sh http://127.0.0.1:18082; docker stop web-pages` → 8/8 `ok`.
- [ ] **Step 3:** `deploy/README.md` Target A "PWA on Cloudflare Pages" paragraph: replace the "Known gap…" sentence with: Pages does not apply the Caddyfile; `frontend/public/_redirects` and `_headers` ship with the build; the workflow's `npm run generate:pages` removes `404.html` because Pages serves a `404.html` before any `_redirects` splat and only switches to implicit SPA mode without one; `nitro.prerender.autoSubfolderIndex: false` keeps `/login` a flat file so the OAuth return trip is not a 308 to `/login/`. *Smoke check* bullet: add the login check, delete the `SMOKE_WEB_SPA_WARN` sentence, mention the ≤ 90 s edge wait.
- [ ] **Step 4:** CODEMAP Deploy bullet: one sentence each for the wait and the no-`404.html` rule. Commit: `deploy: smoke-web checks the /login return trip; runbook states the Pages rules`.

## Verification
```
grep -n 'SMOKE_WEB_SPA_WARN' -r .github deploy frontend/public || echo "gone"            # nothing
grep -n 'generate:pages' frontend/package.json .github/workflows/deploy.yml               # both
grep -n 'Wait for the Pages edge' -A 3 .github/workflows/deploy.yml
actionlint .github/workflows/deploy.yml 2>/dev/null || python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/deploy.yml")); print("yaml ok")'
cd frontend && npm run generate:pages && test ! -f .output/public/404.html && cd ..
docker build -t aelp-web:pages --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend && docker run -d --rm --name web-pages -p 18082:80 aelp-web:pages && sleep 2 && deploy/smoke-web.sh http://127.0.0.1:18082; docker stop web-pages   # exit 0, 8 ok lines
git push -u origin harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-   # CI green incl. docker-images (which runs smoke-web.sh)
```
**Live evidence, next nightly ship (reviewer reads the Deploy run):** step "Wait for the Pages edge" prints `edge ready after Ns`; "Smoke-check the PWA" prints 8 `ok` lines including `spa fallback (/learn/abc): 200` and `login return trip (/login?code&state): 200`; `curl -o /dev/null -w '%{http_code}' https://english-learning-e6a.pages.dev/learn/abc` → `200`.

## Execution summary

Built in worktree `.worktrees/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-` on branch `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`, base `origin/main`. Three commits, matching the three tasks:

- `65a6860` — Task 1: the "Wait for the Pages edge" step (≤ 90 s, polling every 5 s, non-failing) inserted between "Smoke-check the API" and "Smoke-check the PWA"; `SMOKE_WEB_SPA_WARN` env block deleted from the latter.
- `e305840` — Task 2: `frontend/package.json` gained `generate:pages` (`nuxi generate && rm -f .output/public/404.html`); the workflow's "Generate the static site" step now runs it and asserts `404.html` is gone; `deploy/smoke-web.sh`'s SPA check went back to a plain strict `check`; `frontend/public/_redirects` comment updated; `.gitignore`'s `dist/` → `dist` so the symlink `nuxi generate` leaves is actually ignored.
- `8931708` — Task 3: `deploy/smoke-web.sh` gained the `/login?code&state` return-trip check (no `-L`); `deploy/README.md` Target A paragraph and *Smoke check* bullet rewritten; `harness/CODEMAP.md` Deploy bullet updated.

**Deviations from the plan, both within its stated intent:**
1. Task 2 Step 2 (plain SPA check) and Task 3 Step 1 (login check) were written in one edit to `deploy/smoke-web.sh` and landed in the Task 2 commit rather than split across the Task 2 and Task 3 commits the plan implies. No functional difference — both checks are present and correct; only the git-history granularity differs from a strict per-task split.
2. `deploy/README.md` had two more `SMOKE_WEB_SPA_WARN` references beyond the Target A paragraph and *Smoke check* bullet the plan named explicitly: the "Ship from `production`" narrative paragraph (`npx nuxi generate` → `npm run generate:pages`, `smoke-web.sh (with SMOKE_WEB_SPA_WARN=1)` → the edge-wait + strict smoke-web description) and the owner-checklist "First ship" bullet (`one WARN spa fallback line is expected` → `"Wait for the Pages edge" prints edge ready after Ns`). Left unfixed, the plan's own Verification grep (`grep -n 'SMOKE_WEB_SPA_WARN' -r .github deploy frontend/public`, which recurses into `deploy/README.md`) would not have said "gone" — fixed both to keep the runbook internally consistent and the verification honest.

**Verification (this plan's block, run in the worktree):**
```
$ grep -n 'SMOKE_WEB_SPA_WARN' -r .github deploy frontend/public || echo "gone"
gone

$ grep -n 'generate:pages' frontend/package.json .github/workflows/deploy.yml
frontend/package.json:15:    "generate:pages": "nuxi generate && rm -f .output/public/404.html",
.github/workflows/deploy.yml:118:          npm run generate:pages

$ grep -n 'Wait for the Pages edge' -A 3 .github/workflows/deploy.yml
159:      - name: Wait for the Pages edge
160-        if: env.DRY_RUN != 'true'
161-        # A fresh Pages deployment serves its hashed assets as no-store for a
162-        # short window (ship of 2026-09-25 failed on exactly that, then passed

$ actionlint .github/workflows/deploy.yml
(no output — ok)

$ cd frontend && NUXT_PUBLIC_API_BASE=https://api.example.test npm run generate:pages && test ! -f .output/public/404.html && cd ..
(build succeeds; test exits 0 — no 404.html)

$ docker build -t aelp-web:pages --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend && docker run -d --rm --name web-pages -p 18082:80 aelp-web:pages && sleep 2 && deploy/smoke-web.sh http://127.0.0.1:18082; docker stop web-pages
ok   index: 200
ok   login return trip (/login?code&state): 200
ok   spa fallback (/learn/abc): 200
ok   sw.js present: 200
ok   sw.js cache-control: no-cache
ok   manifest cache-control: no-cache
ok   hashed asset found: 1
ok   asset cache-control: public, max-age=31536000, immutable
exit=0
web-pages   (container stopped, no leftover — confirmed with `docker ps`)

$ git push -u origin harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-
new branch pushed; CI triggered
```

**Reproduction — red before the fix (Task 2), captured against the pre-fix worktree:**
```
$ NUXT_PUBLIC_API_BASE=https://api.example.test npx nuxi generate   # plain generate, unpatched package.json
...
[nitro]   ├─ /404.html (22ms)
$ test -f .output/public/404.html && echo "RED: 404.html present"
RED: 404.html present (Pages would use 404 mode, not implicit SPA)
$ git status --short
?? frontend/dist
$ git check-ignore -v frontend/dist; echo "exit=$?"
exit=1   # dist/ (with slash) does not match the symlink frontend/dist
```
After the fix: `npm run generate:pages` leaves no `404.html`; `git status --short` is empty after generate; `git check-ignore -v frontend/dist` → `.gitignore:2:dist	frontend/dist`. A plain `npx nuxi generate` afterwards still produces `404.html` (confirmed unchanged, Task 2 Step 4's second assertion).

**Runtime proof (executor role, Definition of done):**
1. **Builds** — `npm run build` (Node 20-equivalent, local Node 22.20.0) completes clean, no new warnings.
2. **Whole suite** — `npm run test:unit`: 16 files, 82 tests, all passed.
3. **Boots and answers** — the Caddy image (`docker build … frontend`, `docker run -p 18082:80`) served `/`, `/login?code=x&state=y`, `/learn/abc`, `/sw.js`, `/manifest.webmanifest` and a hashed `/_nuxt/*.js` asset, all correct per `deploy/smoke-web.sh`'s 8/8 `ok`; container stopped and removed afterward (`--rm`), confirmed absent via `docker ps -a`.
4. **Every documented command** — `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build`, `npm run generate:pages` (both with and without the `404.html` regression check), the plan's docker build/run/smoke sequence, and `actionlint` — all run exactly as documented, all green.
5. **CI green on the branch** — GitHub Actions run [36216016619](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36216016619) on `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-` at commit `89317081ca9d324d74daac4e8e62cdaf27065c38` (`8931708` short): `backend-unit`, `backend-integration`, `docker-images`, `harness-tooling`, `frontend` all passed (`docker-images`'s "Web image serves the PWA with the right cache headers" step exercises `deploy/smoke-web.sh` against the Caddy image, so the login check and the removed `SMOKE_WEB_SPA_WARN` knob are proven in CI too, not just locally). No backend changes were made by this plan; those jobs were unaffected and stayed green.
6. **No orphan processes/containers** — `docker ps` / `docker ps -a` show nothing of this run's; `frontend/.output` and `frontend/dist` removed from the worktree after each local verification pass; `git status --short` is empty before push.

Pushed branch `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-` at commit `8931708` (local short hash) / `89317081ca9d324d74daac4e8e62cdaf27065c38` (full, post-fetch by GitHub). CI run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36216016619 — **success**.

**Left for the owner:** the plan's "Live evidence, next nightly ship" line is not verifiable by the executor (no Pages/Railway secrets in a worktree) — the owner/reviewer should read the next `Deploy` workflow run for the "Wait for the Pages edge" timing and the live `/learn/abc` 200, as the plan's Verification section says.
