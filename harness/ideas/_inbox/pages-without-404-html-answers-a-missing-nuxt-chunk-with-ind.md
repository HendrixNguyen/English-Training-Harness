---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Pages without 404.html answers a missing /_nuxt/ chunk with index.html as 200 under the one-year immutable header

## Why
The deploy-smoke plan removes `404.html` from the Pages upload (`npm run generate:pages`) so that deep links answer 200. That switches Pages to implicit SPA mode, which also answers **any** missing file with the app shell. For a URL under `/_nuxt/`, `frontend/public/_headers` then attaches `Cache-Control: public, max-age=31536000, immutable` to that HTML. So after every deploy, an open tab that lazy-loads a chunk from the previous build, or fetches `builds/meta/<old id>.json`, gets HTML with a 200, and the browser caches it for a year under the chunk URL. This is the exact defect the same-day caddyfile plan fixes on the Caddy image ("never index.html under a chunk URL"), now reintroduced on the live target. Today the live site answers these requests `404` + `no-store` (because `404.html` exists), so this is a regression the branch introduces. Impact is bounded: Nuxt's chunk-error handler reloads the tab onto the new index.html, which never asks for the poisoned URL again. So this is medium, not a blocker. Nuxt's own error text warns against it: "Ensure that `builds/meta/*.json` is served as JSON by your hosting/proxy and not rewritten to an HTML fallback" (`nuxt/dist/app/composables/manifest.js:23`).

## Expected output
On Pages, a request for a `/_nuxt/` path that is not in the upload answers `404` (never `text/html` with `immutable`), while deep links like `/learn/abc` and `/login?code&state` stay `200`. Try first a `_redirects` rule placed before the splat that sends `/_nuxt/*` misses to a 404; check whether Pages evaluates it only when no asset matches. If that doesn't work, use a Pages `_worker.js`/Function or drop the SPA splat for `/_nuxt/`. `deploy/smoke-web.sh`'s `missing asset is 404` check (added by the caddyfile plan behind `SMOKE_WEB_ASSET_404=1`) should then run on Pages too, so drop the guard. Prove it locally with `npx wrangler@4 pages dev .output/public`, which reproduces both behaviours below.

## Evidence
- Plan: `harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md` (branch `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`, `frontend/package.json:15` `generate:pages`, `.github/workflows/deploy.yml:118`); rule at `frontend/public/_headers:2-3`.
- Local Pages emulation on the branch: `cd frontend && npm ci && NUXT_PUBLIC_API_BASE=https://api.example.test npm run generate:pages && CI=1 npx wrangler@4 pages dev .output/public --port 18099`:
  - `/learn/abc` → `200 text/html`, `public, max-age=0, must-revalidate` (the plan's intent works)
  - `/_nuxt/does-not-exist.js` → `200 text/html; Cache-Control: public, max-age=31536000, immutable`
  - `/_nuxt/builds/meta/nope.json` → `200 text/html; Cache-Control: public, max-age=31536000, immutable`
- Same emulation with a `404.html` put back: `/_nuxt/does-not-exist.js` → `404 text/html; Cache-Control: no-store`. Live today: `curl -sI https://english-learning-e6a.pages.dev/_nuxt/does-not-exist.js` → `404`, `cache-control: no-store`.
- Related: `harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md` (the Caddy half of the same defect; its plan text notes that "Pages' implicit SPA mode answers 200 for any miss").
