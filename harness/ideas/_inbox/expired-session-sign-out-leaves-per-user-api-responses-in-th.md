---
type: bug
status: selected
source: reviewer
run: _inbox
priority: high
---
# Expired-session sign-out leaves per-user API responses in the service worker cache

## Why
Two of the three sign-out paths clear the service worker's `api-state` cache; the third does not. `components/AppHeader.vue:12-13` (menu sign-out) and `composables/useApi.ts:17-18` (401 hook) both call `auth.signOut()` then `clearApiCache()`. `middleware/auth.global.ts:13` — the path taken when a session simply lapses (`expires_in` 86400 elapses, `isAuthenticated` goes false) — calls `auth.signOut()` alone.

The `api-state` cache holds `GET /api/v1/quests/daily` and `GET /api/v1/pet/status` responses, which are per-user: the daily task list and titles, minutes studied, plant name, health and streak. The Cache API keys on URL only — the `Authorization` header is not part of the key and the strategy sets no `Vary` handling — so after an expired-session sign-out, the *next* account signed in on the same device hits the same two URLs. `NetworkFirst` with `networkTimeoutSeconds: 5` falls back to the cache whenever the network is offline or slower than 5s, which then serves the previous user's data. The entries never expire: no `ExpirationPlugin`, no `maxAgeSeconds`, no `maxEntries`.

Shared devices (a family tablet, a classroom laptop) are exactly the deployment this PWA targets.

## Expected output
Every path that drops a session also drops the per-user cache. `middleware/auth.global.ts` calls `clearApiCache()` alongside `auth.signOut()`, or — better — `useAuthStore.signOut()` owns the cache clear so no future caller can forget it. Additionally, the `api-state` route carries an `ExpirationPlugin` with a `maxAgeSeconds` bound so a stale per-user response cannot be served indefinitely.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` (Tasks 3 and 13).
- `frontend/middleware/auth.global.ts:12-15` — `if (!auth.isAuthenticated) { auth.signOut(); return navigateTo('/login', ...) }`, no `clearApiCache()`; the file does not import `~/utils/session` at all.
- Contrast `frontend/components/AppHeader.vue:11-15` and `frontend/composables/useApi.ts:16-20`, which both clear it.
- `frontend/service-worker/sw.ts:15-18` — `new NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 })` matching `/api/v1/(quests/daily|pet/status)$`, with no plugins array.
- The cache name is not prefixed, so `caches.delete('api-state')` does match the worker's cache: `node_modules/workbox-core/_private/cacheNames.js` → `getRuntimeName: (userCacheName) => userCacheName || _createCacheName(...)`. The clear works where it is called; the problem is only the path that omits it.
- The executor's own Runtime proof records the mechanism from the other side: "the SW's `NetworkFirst` cache for `/quests/daily` had cached an earlier 200 response, which persisted stale data across a backend state change until the SW/cache were cleared".

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — high (was medium; top 10).** Confirmed: `middleware/auth.global.ts:12-15` calls `auth.signOut()` without `clearApiCache()`, unlike the two other sign-out paths. Sessions are 24 h, so *every* user takes this path daily; on a shared device the next account, offline or on a slow network, is served the previous user's quests and plant from the `api-state` cache. Personal-data leak on the happy path, one-line fix plus making `signOut()` own the cache clear so no caller can forget again, plus an `ExpirationPlugin` bound. Cheap, high value — plan next after the current three.
