---
type: bug
status: planned
source: human
run: _inbox
priority: high
plan: harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md
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

## Evaluation
_Evaluator, 2026-09-26 — daily decide (bug queue, ranked #4; head of a three-item Pages plan)._

**Select — high.** *Is the Why real?* Yes: the first real Deploy (run 36156175735) reported failure on a ship that had succeeded, and every nightly ship will look the same until the wait exists; a genuine header regression becomes indistinguishable. *Root cause:* `.github/workflows/deploy.yml` runs `deploy/smoke-web.sh` immediately after the Pages upload; Pages' edge serves the fresh deployment's hashed assets as `no-store` for a short window. *Fix:* a bounded wait (≤ 90 s) on the exact two signals the smoke check needs (immutable asset header and the deep-link 200), falling through to the strict check. Planned together with `pages-ignores-the-redirects-spa-rewrite-while-404-html-exist` (the deep-link 404, same workflow/script) and `task-4-follow-up-smoke-web-has-no-login-return-trip-check-an` (same script/runbook paragraph), so the three edits to `deploy/smoke-web.sh` and the runbook's Pages paragraph land once.
