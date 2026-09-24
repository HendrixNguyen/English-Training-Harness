---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md
---
# A stale api-state cache survives into the next account when /login is the first route

## Why
`harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md` closed the three *sign-out* paths: `useAuthStore.signOut()` now owns `clearApiCache()`, so the menu, the 401 hook and the route guard's expired-session branch all drop `api-state`. Verified by review, including mutation testing.

What is still open is the *sign-in* side. The guard returns early for `/login` before it ever reaches the `!isAuthenticated` branch, so an expired session that is present when `/login` is the **first** route of a browsing session is never signed out, and `signIn()` does not clear the cache either. The next account then lands on `/` with the previous learner's `GET /quests/daily` and `GET /pet/status` still in `api-state`, and `NetworkFirst` (`networkTimeoutSeconds: 5`) serves them whenever the device is offline or the network is slower than 5 s. That is the same per-user data leak the original idea describes, reached by a different door.

Reachability is narrower than the fixed bug, which is why this is not a merge blocker on that branch: the PWA manifest sets `start_url: '/'` (`frontend/nuxt.config.ts:50`), so the installed app always enters through the guard's sign-out branch, and the OAuth return to `/login?code=` is always preceded by a redirect from `/` that already signed out. The hole is a bookmark, a typed URL or a shared link straight to `/login` on a shared device — exactly the family tablet / classroom laptop deployment the original idea calls out. It is also pre-existing: `main` leaks in the same scenario.

A second, smaller path reaches the same state: `hydrate()` discards a corrupt persisted session with `removeItem(AUTH_STORAGE_KEY)` and no cache clear. On every route but `/login` the guard signs out one statement later and covers it; on `/login` it does not.

## Expected output
Signing a user *in* leaves no other user's data behind. Either `useAuthStore.signIn()` clears `api-state` before it stores the new session (cheap, idempotent, symmetrical with `signOut()` owning the clear), or the guard stops exempting `/login` from the expired-session branch — `if (!auth.isAuthenticated) void auth.signOut()` should run before the `to.path === '/login'` early return, since signing out an already-invalid session on `/login` is a no-op for navigation.

A regression test in the shape the reviewed plan established: seed the Map-backed fake from `frontend/tests/unit/fakeCaches.ts` with an `api-state` entry, persist an expired session, drive the guard with `to.path === '/login'` (and separately drive `signIn()`), and assert `await caches.has(API_STATE_CACHE) === false` while `await caches.keys()` still contains `'assets'`.

## Evidence
- Reviewed plan: `harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md`; review `harness/reviews/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md`.
- `frontend/middleware/auth.global.ts:7-11` — `if (to.path === '/login') { if (auth.isAuthenticated && !to.query.code) return navigateTo('/', ...); return }`. The `return` on line 10 is taken for an expired session, so the `!auth.isAuthenticated` branch on lines 12-15 (which now calls `void auth.signOut()`) is never reached on `/login`.
- `frontend/stores/auth.ts:59-67` — `signIn()` writes the new session and never touches `clearApiCache()`; only `signOut()` (line 74) does.
- `frontend/stores/auth.ts:55-57` — `hydrate()`'s `catch { storageOrNull()?.removeItem(AUTH_STORAGE_KEY) }` drops a session without a cache clear.
- `frontend/pages/login.vue:35` — `auth.signIn(res)` then `navigateTo('/', { replace: true })`, with no cache clear in between.
- `frontend/service-worker/sw.ts:15-18` — `NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 })` for `/quests/daily` + `/pet/status`, keyed on URL only, no plugins.
- `frontend/nuxt.config.ts:50` — `start_url: '/'`, which is why the installed-PWA path is safe and this is medium, not high.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Planned today in `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` (Also planned here).**

*Confirmed (read on this branch).* `frontend/middleware/auth.global.ts:7-11` — the `/login` branch returns before the `!auth.isAuthenticated` sign-out branch, so an expired session present when `/login` is the first route is never dropped; `frontend/stores/auth.ts:59-67` — `signIn()` writes the new session and never calls `clearApiCache()` (only `signOut()` does, line 74); `hydrate()`'s corrupt-storage `catch` (55-57) removes the session with no cache clear. `pages/login.vue:35` calls `auth.signIn(res)` then `navigateTo('/')`, so the next account can be served the previous learner's `/quests/daily` and `/pet/status` from `api-state` whenever the network is slow or offline.

*Fix.* `signIn()` owns a clear — state and storage are written synchronously as today, then `clearApiCache()` runs and `signIn` returns that promise; `login.vue` awaits it before navigating so `/` cannot read the stale entry first. The guard's `/login` branch also signs out a session that is present but expired, so `localStorage` and the cache are clean before the consent redirect. Tests in the shape the 2026-09-23 plan established (`installSeededCaches`): `signIn()` leaves `api-state` gone and `assets` intact; the guard on `to.path === '/login'` with an expired persisted session clears the cache and does not redirect.

*Priority.* Medium: the installed PWA's `start_url: '/'` routes through the existing sign-out branch, so the leak needs a bookmark or typed `/login` on a shared device — a real deployment (family tablet, classroom laptop) but not the common path.
