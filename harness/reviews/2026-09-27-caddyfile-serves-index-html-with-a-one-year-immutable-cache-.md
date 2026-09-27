---
plan: harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/the-nuxt-immutable-cache-rule-also-covers-the-unhashed-build.md]
---
# Review — Dokploy target hardening: Caddy answers 404 for a missing chunk and `no-cache` for the shell, compose passes the AI base-URL/model variables, smoke-api reports every check

**Plan:** `harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md`
**Branch/worktree:** `harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-` / `.worktrees/caddyfile-serves-index-html-with-a-one-year-immutable-cache-`
**Diff:** `git diff main...harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache- --stat`

## Plan vs idea
Delivered for all three ideas. (1) Head idea: on Caddy a missing `/_nuxt/` chunk is now a plain 404 with no immutable header, and the shell is `no-cache`. (2) `compose.yml` passes the six `*_BASE_URL`/`*_MODEL` variables, and `VAPID_SUBJECT` no longer defaults to a placeholder. (3) `smoke-api.sh` reports every check when the host is unreachable.

## Code vs plan
Branch head: CI `36216572852` completed/success (`gh run list --branch <branch> --limit 1`). `git merge-tree origin/main origin/<branch>` merges cleanly. Against plan 4 (`deploy-smoke-checks…`), the only conflict is `harness/CODEMAP.md`.

- Task 1 (Caddyfile `handle /_nuxt/*` + `@exists` header, `handle` catch-all `no-cache`, `_headers` `/` + `/index.html`, the two smoke-web checks, `SMOKE_WEB_ASSET_404=1` in `docker-images`): followed. The `@exists` matcher replaced the plan's fallback option, and it works.
- Task 2 (six vars + `VAPID_SUBJECT` in compose, `.env.example`, README): followed.
- Task 3 (`|| true` on smoke-api curls): followed.

```
$ deploy/smoke-api.sh http://127.0.0.1:1 http://x; echo exit=$?
FAIL healthz status: expected '200', got '000'
FAIL healthz body: expected '1', got '0'
FAIL auth/google empty body: expected '400', got '000'
FAIL preflight status: expected '204', got ''
FAIL preflight allow-origin: expected 'http://x', got ''
FAIL preflight foreign origin: expected '403', got '000'
exit=1          (6 FAIL lines: the plan said 5, but it has 6 checks. The plan miscounted; the script is fine.)
$ docker build -t aelp-web:rv-deploy --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend && docker run -d --rm -p 18097:80 ...
$ SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18097
ok index / spa fallback / sw.js present / sw.js cache-control / manifest cache-control / hashed asset found /
ok asset cache-control: public, max-age=31536000, immutable
ok index cache-control: no-cache
ok missing asset is 404 (/_nuxt/does-not-exist.js): 404        exit=0
$ curl -sI .../_nuxt/old-deleted-chunk.js   → HTTP/1.1 404 Not Found (no Cache-Control)
$ curl -sI .../  and  .../learn/abc          → 200, Cache-Control: no-cache, text/html
$ docker compose -p rv-deploy -f deploy/compose.yml --env-file <scratch, OPENAI_BASE_URL/MODEL set> config
DEEPSEEK_BASE_URL: "" / DEEPSEEK_MODEL: "" / GEMINI_BASE_URL: "" / GEMINI_MODEL: "" /
OPENAI_BASE_URL: https://openrouter.example.test/api/v1 / OPENAI_MODEL: x / VAPID_SUBJECT: ""
$ grep -n 'BASE_URL\|_MODEL' deploy/.env.example deploy/README.md | wc -l   → 15
```
Blank values reach the container as `""`. `airouter/config.go:38-47` wraps each one in `or(lookup(..), Default…)`, so empty means the default, as intended. The full-stack runtime (compose up plus smoke-api against a live API) was the executor's. I did not repeat it because the containers and API code are unchanged by this plan.

## Quality
- Pages parity was checked read-only against the live site. Pages returns the `_headers` value verbatim (`/sw.js` → `cache-control: no-cache`), so the new `/` rule will pass smoke-web's exact-match `index cache-control` check once this ships. `/index.html` gets a 308 from Pages, so that rule is inert but harmless.
- The catch-all `handle` now marks every non-`/_nuxt/` static file `no-cache`: favicon, PWA icons, `logo.svg`. That costs a revalidation per load (304s) for files that rarely change. It is acceptable and arguably safer than heuristic caching, so no bug.
- An unhashed file sits under the immutable rule: `/_nuxt/builds/latest.json`. The rule predates this plan, but the new comment calls everything there "content-hashed" → low bug.
- Cross-plan: plan 4 removes `404.html` from Pages. After that, Pages' implicit SPA mode may answer a missing `/_nuxt/` chunk with index.html under the `_headers` immutable rule. That is the Pages twin of this plan's Caddy bug. It is filed against plan 4.
- `.env.example` calls `VAPID_SUBJECT` "required in production", but the binary still silently falls back to `config.DefaultVAPIDSubject`. The plan scoped that out explicitly and the README row says so. Noted, no bug.
- CODEMAP Deploy bullet: accurate.

## Bugs filed
- `harness/ideas/_inbox/the-nuxt-immutable-cache-rule-also-covers-the-unhashed-build.md` (low). `/_nuxt/builds/latest.json` is not hashed but is cached immutable for a year on both targets. Real-world impact is near nil: Nuxt's check fetches it with a `?${Date.now()}` cache-buster (`nuxt/dist/app/plugins/check-outdated-build.client.js:26`). The fix is defensive, plus correcting the comment.

## Verdict
`pass-with-bugs`. It may go into today's daily PR. It merges cleanly onto main, and its CODEMAP conflicts with plan 4's.
