---
type: bug
status: planned
source: human
run: _inbox
priority: medium
plan: harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md
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

## Evaluation
**Verdict: select, `priority: medium`, folded into the plan for `nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`.**

*Is the Why real?* Yes — the smoke output is first-hand (4 of 7 checks fail on the live Pages URL). The 404 status on deep links and the `max-age=0, must-revalidate` on hashed chunks are Pages defaults; the runbook's claim that Pages needs no `_headers` is wrong. Frontend spec §3 requires `sw.js`/manifest to be served fresh and assets long-cached; the Caddy image does it (`frontend/Caddyfile`), Pages does not.

*Root cause (read-only):* `frontend/public/` holds only icons — there is no `_redirects` and no `_headers`, so Pages serves its defaults; `nuxi generate` emits `404.html`, which switches Pages out of its implicit SPA mode (Pages docs: without a top-level `404.html` it would fall back to `/`) and makes it answer unknown routes with that file and a 404 status.

*Fix:* `frontend/public/_redirects` (`/*  /index.html  200`) and `frontend/public/_headers` (`/_nuxt/*` immutable; `/sw.js` and `/manifest.webmanifest` `no-cache`). Nuxt copies `public/` verbatim into `.output/public`; the Caddy image serves the two files as inert text and keeps its own rules, so Target B and the CI `docker-images` job are unchanged. The `_headers` values are copied from `frontend/Caddyfile` so the two hosts stay identical.

*Why the same plan:* it is the same defect — the Pages target was declared equivalent to the Caddy image without being exercised — and the sign-in bug's `autoSubfolderIndex` change touches the same generate output. The runbook correction (`deploy/README.md` Target A) and the 7/7 smoke run are conditional on `deploy/` being present on the executor's base (see the head idea's Evaluation).

*Priority rationale:* `medium` as filed — the app still renders behind the 404 and slow revalidation; it does not block a user the way the 308 does.
