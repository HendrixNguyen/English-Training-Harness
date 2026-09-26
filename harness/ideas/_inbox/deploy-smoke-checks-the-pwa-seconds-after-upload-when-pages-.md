---
type: bug
status: proposed
source: human
run: _inbox
priority: high
---
# Deploy smoke checks the PWA seconds after upload, when Pages' edge still serves hashed assets as no-store, so a good ship reports failure

## Why
First real Deploy on `production` (run 36156175735, 2026-09-25 15:44Z): every step green — config check, generate, **upload to Pages**, `/healthz`, API smoke — then "Smoke-check the PWA" failed on one line: `FAIL asset cache-control: expected 'public, max-age=31536000, immutable', got 'no-store'`. Re-running `deploy/smoke-web.sh` against the same URL two minutes later passed 7/7 (the `_headers` rule on `main` is correct; the fresh deployment's edge copies had not picked it up yet). The nightly ship routine reports the Deploy conclusion to the owner, so every ship will look broken while it actually succeeded — and a real header regression would be indistinguishable from this race.

## Expected output
- In `.github/workflows/deploy.yml`, before "Smoke-check the PWA": a bounded wait (≤ 90 s, `curl --max-time`, 5 s steps) until the first `/_nuxt/*.js` referenced by `index.html` returns the immutable `Cache-Control`; if it never does, fall through and let the smoke check fail as today (never skip it, never downgrade it).
- The wait prints how long it took, so a slowly propagating edge is visible in the log.
- actionlint clean; the plan's verification replays the check against the live URL.

## Evidence
```
ship | Upload the PWA to Cloudflare Pages | success
ship | Smoke-check the API | success
ship | Smoke-check the PWA | failure
FAIL asset cache-control: expected 'public, max-age=31536000, immutable', got 'no-store'
# 2 minutes later, same script, same URL:
ok   asset cache-control: public, max-age=31536000, immutable      exit=0
```
