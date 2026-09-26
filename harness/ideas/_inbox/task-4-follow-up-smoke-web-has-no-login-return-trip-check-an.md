---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md
---
# Task 4 follow-up: smoke-web has no /login return-trip check and the deploy runbook still says Pages needs no _headers

## Why
The sign-in outage was masked by the deploy runbook: `deploy/README.md` (on `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-`, line 39) says "Pages serves the SPA fallback and hashed-asset caching by itself … so no `_headers` file is needed". The fix plan skipped its Task 4 (the `deploy/` files were not on `origin/main`), and that branch is still not merged into `origin/main` as of this review. Once it merges, the runbook contradicts `frontend/public/_redirects`/`_headers`, and `deploy/smoke-web.sh` has no check that would catch a return of the `/login` 308 — the idea's *Expected output* explicitly asked for that check.

## Expected output
- `deploy/smoke-web.sh`: `check "login return trip (/login?code&state)" "200" "$(status "$web/login?code=x&state=y")"` (no `-L`, so any 3xx fails).
- `deploy/README.md` *PWA on Cloudflare Pages* paragraph: Pages does not apply the Caddyfile rules; `_redirects`/`_headers` are required; `nitro.prerender.autoSubfolderIndex: false` and why. Smoke bullet mentions the login check. Exact text is in the plan's Task 4.
- Do this on a branch based on `origin/main` after the deploy branch has merged; never cherry-pick it.

## Evidence
- Plan `harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`, Task 4 (skipped; execution summary "Task 4 skipped: deploy/ not on base").
- `git merge-base --is-ancestor origin/harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- origin/main` → false (2026-09-25 review).
- Idea `harness/ideas/_inbox/nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`, *Expected output* bullet 3 (smoke-web check).

## Evaluation
_Evaluator, 2026-09-26._ **Select — medium, folded into** `harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md` (one Pages plan: edge-propagation wait, no `404.html` on Pages so deep links are 200, the `/login` return-trip smoke check and the runbook paragraph). The deploy branch it depended on is on `origin/main` now, so the runbook edit is unconditional.
