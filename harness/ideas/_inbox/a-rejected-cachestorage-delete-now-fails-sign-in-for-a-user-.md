---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A rejected CacheStorage delete now fails sign-in for a user who is already signed in

## Why
`useAuthStore.signIn()` now returns `clearApiCache()` (`frontend/stores/auth.ts:73`), and `pages/login.vue:35` awaits it inside the `try` that also wraps the `POST /auth/google`. `utils/session.ts:5-8` only guards `typeof caches === 'undefined'`; `caches.delete()` itself is not caught. If it rejects, the session has already been written to state and `localStorage['aelp.auth']`, but the `catch` renders "Không đăng nhập được. Thử lại." and the page stays on `/login`. Retrying goes through Google again and fails the same way, although the user is in fact signed in (navigating to `/` by hand works).

When `caches.delete` rejects is browser-dependent: the Cache API specifies `SecurityError` rejections where storage is not permitted (e.g. site data blocked, some private-browsing modes). I have not reproduced it in a specific browser — that would settle whether this is real today. Before this branch, a rejection could only surface as an unhandled rejection from the `void`-ed `signOut()` calls; now it breaks the sign-in path.

## Expected output
Clearing the `api-state` cache is best-effort everywhere: `clearApiCache()` swallows (or logs) a rejected `caches.delete`, so `signIn()` / `signOut()` always resolve and `/login` navigates to `/` once the session is stored. A unit test stubs `caches` with a `delete` that rejects and asserts `await auth.signIn(spec61)` resolves with `isAuthenticated === true` (and `signOut()` likewise resolves).

## Evidence
- Plan under review: `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` (Task 2, Design decision 3).
- `frontend/stores/auth.ts:65-74`, `frontend/pages/login.vue:33-45`, `frontend/utils/session.ts:5-8`.
