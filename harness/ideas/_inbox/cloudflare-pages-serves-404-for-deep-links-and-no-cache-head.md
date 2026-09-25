---
type: bug
status: proposed
source: human
run: _inbox
priority: medium
---
# Cloudflare Pages serves 404 for deep links and no-cache headers differ from the Caddy image, so smoke-web fails on the live site

## Why
First real deploy (2026-09-25, owner + orchestrator): the PWA on `https://english-learning-e6a.pages.dev` loads, but `deploy/smoke-web.sh` fails 4 of 7 checks. A learner who opens a deep link (a bookmark to `/learn/...`, or the push-notification click target) gets an HTTP 404 status with the SPA shell — the app still renders, but crawlers, some in-app browsers and the smoke check treat it as broken. Hashed assets are served `max-age=0, must-revalidate` instead of immutable, so every reload revalidates every chunk on a phone connection.

The runbook (`deploy/README.md`, Target A) claims Pages "serves the SPA fallback and hashed-asset caching by itself, so no `_headers` file is needed" — that is wrong: `nuxi generate` emits `404.html`, and Pages serves it with a 404 status instead of falling back to `index.html` with 200; Pages' default `Cache-Control` for all files is `public, max-age=0, must-revalidate`.

## Expected output
- `frontend/public/_redirects` with `/*  /index.html  200` (SPA fallback with a 200), and `frontend/public/_headers` giving `/_nuxt/*` → `Cache-Control: public, max-age=31536000, immutable` and `/sw.js`, `/manifest.webmanifest` → `Cache-Control: no-cache`. Both files are copied verbatim into `.output/public` by Nuxt; the Caddy image ignores them, so Target B is unchanged.
- `deploy/smoke-web.sh` passes 7/7 against the Pages URL; runbook Target A paragraph corrected; the frontend CI job still green.

## Evidence
```
$ sh deploy/smoke-web.sh https://english-learning-e6a.pages.dev
ok   index: 200
FAIL spa fallback (/learn/abc): expected '200', got '404'
ok   sw.js present: 200
FAIL sw.js cache-control: expected 'no-cache', got 'public, max-age=0, must-revalidate'
FAIL manifest cache-control: expected 'no-cache', got 'public, max-age=0, must-revalidate'
ok   hashed asset found: 1
FAIL asset cache-control: expected 'public, max-age=31536000, immutable', got 'public, max-age=0, must-revalidate'
```
Pages docs: `_redirects` and `_headers` files in the build output — https://developers.cloudflare.com/pages/configuration/redirects/ , https://developers.cloudflare.com/pages/configuration/headers/
