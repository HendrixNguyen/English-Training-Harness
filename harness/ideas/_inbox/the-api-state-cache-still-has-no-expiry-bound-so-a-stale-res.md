---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# The api-state cache still has no expiry bound so a stale response can be served indefinitely

## Why
The idea `harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md` asked for two things in its *Expected output*: every session-dropping path clears `api-state`, **and** the `api-state` route carries an `ExpirationPlugin` with a `maxAgeSeconds` bound. The executed plan delivered the first and deliberately deferred the second, with reasoning recorded in its *Notes and open questions*: `workbox-expiration` is a new dependency plus a change to `service-worker/sw.ts`, which is excluded from `tsconfig.json`, is not unit-tested, and is blocked in Playwright (`serviceWorkers: 'block'`) — so it would ship unverified.

The reviewer agrees with the deferral and files this so the second half of the idea is not silently lost. The residual risk is a device that is never signed out: a learner who stays signed in for weeks can be served a `GET /pet/status` or `GET /quests/daily` body of arbitrary age whenever the network is offline or slower than 5 s, because `NetworkFirst` falls back to a cache entry that never expires. That is a staleness bug (a plant health or streak from last week shown as today's), not a privacy leak.

## Expected output
`api-state` entries stop being served past a defined age — an `ExpirationPlugin({ maxAgeSeconds, maxEntries })` on the `NetworkFirst` route in `frontend/service-worker/sw.ts`, with `maxAgeSeconds` chosen against the daily loop (a day's worth at most, since both endpoints are per-day state).

Because the worker is outside `tsconfig.json` and outside both test harnesses, this idea's plan must carry its own verification step: a browser-level check (Playwright with `serviceWorkers: 'allow'` for that one spec, or a manual runtime proof) that a seeded `api-state` entry older than the bound is not served. Do not land it on unit-test evidence alone.

## Evidence
- Original idea: `harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md`, *Expected output*, second sentence.
- Deferral and its reasoning: `harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md`, *Notes and open questions*, first bullet.
- `frontend/service-worker/sw.ts:15-18` — `new NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 })`, no `plugins` array.
- `frontend/playwright.config.ts` — `serviceWorkers: 'block'`, which is why no existing test can observe the worker.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.**

*Confirmed (read on this branch).* `frontend/service-worker/sw.ts` `NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 })` has no `plugins`; `playwright.config.ts` blocks service workers, and the worker is outside `tsconfig.json`, so no existing harness can observe it. The 2026-09-23 plan's deferral reasoning still holds.

*Fix, when planned.* `ExpirationPlugin({ maxAgeSeconds: 86400, maxEntries: 4 })` on that route (both endpoints are per-day state), `workbox-expiration` added, and a plan whose *Verification* is browser-level — one Playwright spec with `serviceWorkers: 'allow'` that seeds an aged `api-state` entry and proves it is not served. Low: staleness of a per-day screen on a never-signed-out device, not a privacy leak; it must not land on unit-test evidence alone, which is why it does not ride with today's frontend follow-ups.
