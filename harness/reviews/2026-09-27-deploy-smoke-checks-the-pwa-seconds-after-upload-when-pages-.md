---
plan: harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/pages-without-404-html-answers-a-missing-nuxt-chunk-with-ind.md]
---
# Review — Pages ship is honest: wait for the edge before the smoke check, no `404.html` on Pages so deep links answer 200, and the `/login` return-trip check

**Plan:** `harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md`
**Branch/worktree:** `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-` / `.worktrees/deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`
**Diff:** `git diff main...harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages- --stat`

## Plan vs idea
Delivered for all four ideas. (1) Head idea: the Deploy workflow now waits, bounded at 90 s, for the Pages edge to serve the new hashed asset as immutable and a deep link as 200, then runs the unchanged strict smoke check. On timeout it falls through; nothing was loosened. (2) `404.html` is gone from the Pages upload, so deep links answer 200 under implicit SPA mode, and `SMOKE_WEB_SPA_WARN` is deleted everywhere. (3) smoke-web gains the `/login?code&state` return-trip check (no `-L`). (4) `.gitignore` `dist` now ignores the `frontend/dist` symlink. The live proof ("edge ready after Ns" in a real Deploy run) can only come from the next production ship, as the plan says.

## Code vs plan
Branch head `8931708`, CI `36216016619` completed/success. `git merge-tree origin/main origin/<branch>` is clean. Against plan 3 (caddyfile) the only conflict is `harness/CODEMAP.md`. The branch keeps main's `workflow_run`/tagging `deploy.yml` intact: it edits only the generate step, adds the wait step and drops the env knob. Nothing is reverted.

- Task 1 (wait step): followed. It polls every 5 s × 18, the `|| true` guards keep it non-failing, and it prints elapsed time.
- Task 2 (`generate:pages`, `test ! -f 404.html`, strict SPA check, `_redirects` comment, `.gitignore`): followed.
- Task 3 (login check, runbook, CODEMAP): followed. The two extra README `SMOKE_WEB_SPA_WARN` mentions the executor fixed were justified; without them the plan's own grep would have failed.

```
$ grep -n 'SMOKE_WEB_SPA_WARN' -r .github deploy frontend/public || echo gone
gone
$ grep -n 'generate:pages' frontend/package.json .github/workflows/deploy.yml
frontend/package.json:15 / .github/workflows/deploy.yml:118
$ actionlint .github/workflows/deploy.yml && echo actionlint ok
actionlint ok
$ cd frontend && npm ci && NUXT_PUBLIC_API_BASE=https://api.example.test npm run generate:pages; test ! -f .output/public/404.html
gen exit=0 / no 404.html   (output keeps 200.html, login.html, _headers, _redirects)
$ docker build ... frontend && docker run -p 18098:80 ... && deploy/smoke-web.sh http://127.0.0.1:18098
ok index / ok login return trip (/login?code&state): 200 / ok spa fallback (/learn/abc): 200 / ok sw.js present /
ok sw.js cache-control / ok manifest cache-control / ok hashed asset found / ok asset cache-control   exit=0
$ docker run --rm --entrypoint sh <image> -c 'ls /srv/404.html'   → /srv/404.html   (Caddy image unchanged, as designed)
$ curl -s -o /dev/null -w '%{http_code}' 'https://english-learning-e6a.pages.dev/login?code=x&state=y'   → 200 (live, read-only)
```
Pages emulation (`CI=1 npx wrangler@4 pages dev .output/public --port 18099`, local, no credentials): `/`, `/learn/abc` and `/login?code=x&state=y` are all `200 text/html`, so the plan's core claim holds under Pages' own routing. I did not re-run the frontend lint/typecheck/unit suite. The only frontend change is a package.json script, and CI's `frontend` job is green on the head.

## Quality
- **Regression (medium bug):** implicit SPA mode also answers a missing `/_nuxt/` file with index.html as `200`, and `_headers`' `/_nuxt/*` rule then stamps it `immutable` for a year. In emulation: `/_nuxt/does-not-exist.js` → `200 text/html; public, max-age=31536000, immutable`. With `404.html` restored, or live today, it's `404; no-store`. This is the Pages twin of the defect plan 3 fixes for Caddy. The branch trades an honest-404 deep link for poisoned-chunk caching. Nuxt's chunk-error reload bounds the damage, so it is not a blocker.
- The wait step's shell is fine under GitHub's default `bash -e`. Every command substitution carries `|| true`, and an empty `asset` just re-curls `/`.
- `generate:pages` lives in `package.json`, so the Dockerfile/CI `nuxi generate` path is untouched and the Caddy image keeps `404.html` + `try_files`. That split is clean.
- CODEMAP Deploy bullet: accurate for the branch.

## Bugs filed
- `harness/ideas/_inbox/pages-without-404-html-answers-a-missing-nuxt-chunk-with-ind.md` (medium). Without `404.html`, Pages serves index.html under the immutable `/_nuxt/*` header for missing chunks. Regression introduced by this branch; reproduced with `wrangler pages dev`.

## Verdict
`pass-with-bugs`. It may go into today's daily PR; it merges cleanly onto main, and its CODEMAP conflicts with plan 3's. The medium bug should be planned soon, because the next production ship makes it live.
