---
type: bug
status: proposed
source: human
run: _inbox
priority: medium
---
# Pages ignores the _redirects SPA rewrite while 404.html exists, so deep links still answer 404

## Why
Follow-up the plan `2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect` predicted: after deploying its build (flat `login.html`, `_redirects` with `/*  /index.html  200`, `_headers`) to `https://english-learning-e6a.pages.dev`, sign-in and the cache headers are fixed but an unknown route still returns HTTP 404 with the app shell — Cloudflare Pages serves the generated `404.html` before applying the splat rewrite. A push-notification click target or a bookmarked deep link therefore still loads with a 404 status (the app renders, but in-app browsers and crawlers treat it as broken, and `deploy/smoke-web.sh` keeps failing that check).

## Expected output
- `GET /learn/abc` on Pages → 200 with the app shell. Take the plan's own documented fallback: stop emitting `404.html` for the Pages target (e.g. `nitro.prerender.ignore` / a post-generate step that removes `.output/public/404.html`, or `nuxt.config` `app` option that suits the Caddy image too), so Pages' implicit SPA mode applies; keep `_redirects` only if it is then still needed. The Caddy image keeps `try_files`.
- `deploy/smoke-web.sh` 7/7 (8/8 with the login check) against the live Pages URL; runbook Target A paragraph states the real rule (no `404.html` on Pages).

## Evidence
```
$ curl -sS -o /dev/null -w '%{http_code}\n' 'https://english-learning-e6a.pages.dev/learn/abc'      # after deploy 7d4bbbc0
404
$ curl -sS -o /dev/null -w '%{http_code} %{redirect_url}\n' 'https://english-learning-e6a.pages.dev/login?code=x&state=y'
200                                                                                                # the sign-in half is fixed
$ curl -sI https://english-learning-e6a.pages.dev/sw.js | grep -i cache-control
cache-control: no-cache                                                                            # _headers applied
```
Cloudflare docs on 404.html vs SPA mode: https://developers.cloudflare.com/pages/configuration/serving-pages/#single-page-application-spa-rendering
