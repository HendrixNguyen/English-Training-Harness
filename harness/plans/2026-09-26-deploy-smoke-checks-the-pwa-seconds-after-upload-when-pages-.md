---
idea: harness/ideas/_inbox/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md
status: approved
priority: high
merged: false
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
