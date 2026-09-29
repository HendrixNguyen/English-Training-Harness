---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# The /_nuxt/* immutable cache rule also covers the unhashed builds/latest.json app manifest

## Why
Nuxt writes `/_nuxt/builds/latest.json` (`{"id": <buildId>, "timestamp": …}`), and its name is the same on every deploy. The Caddy image (`frontend/Caddyfile` `handle /_nuxt/*`) and Pages (`frontend/public/_headers` `/_nuxt/*`) both serve it `public, max-age=31536000, immutable`. **Impact is small, and I checked it:** Nuxt's outdated-build check fetches it as `buildAssetsURL("builds/latest.json") + \`?${Date.now()}\`` (`frontend/node_modules/nuxt/dist/app/plugins/check-outdated-build.client.js:26`), so browsers never reuse the cached copy. What remains is a defensive and documentation fix. Anything else that reads the file without a cache-buster (a proxy that ignores query strings, a future SW precache rule, a hand-written update check) would pin the first build id for a year. The Caddyfile's new comment also calls everything under `/_nuxt/` "content-hashed", which is not true for this file. The rule predates the caddyfile plan.

## Expected output
`/_nuxt/builds/latest.json` is served `Cache-Control: no-cache` on both targets: a more specific rule placed before the immutable one in the Caddyfile, and a `/_nuxt/builds/latest.json` block in `_headers`. `/_nuxt/builds/meta/<id>.json` and hashed chunks stay immutable. `deploy/smoke-web.sh` checks `latest.json`'s Cache-Control, and the Caddyfile comment says what is actually hashed.

## Evidence
- Plan under review: `harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md` (the rule itself predates it: `origin/main:frontend/Caddyfile` `header /_nuxt/*`, `origin/main:frontend/public/_headers:2`).
- `docker build -t aelp-web:rv-deploy --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend` on the plan branch, then `docker run --rm --entrypoint sh aelp-web:rv-deploy -c 'find /srv/_nuxt -name "*.json"; cat /srv/_nuxt/builds/latest.json'` → `/srv/_nuxt/builds/latest.json`, `/srv/_nuxt/builds/meta/9c656239-….json`, `{"id":"9c656239-…","timestamp":1790395051017}`.
- `frontend/Caddyfile` (branch) `handle /_nuxt/* { @exists file; header @exists Cache-Control "public, max-age=31536000, immutable"; file_server }`.
