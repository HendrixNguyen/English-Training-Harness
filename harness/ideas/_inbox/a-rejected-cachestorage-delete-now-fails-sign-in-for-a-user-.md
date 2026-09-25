---
type: bug
status: selected
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

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.** Confirmed on `main`: `frontend/utils/session.ts` guards only `typeof caches === 'undefined'`, and `pages/login.vue` awaits `signIn()` inside the `try` that renders the sign-in failure. Unreproduced in a named browser (the reviewer says so), so it stays low, but the fix is cheap and the shape is right: cache clearing is best-effort. Decision recorded for the plan: `clearApiCache()` catches a rejected `caches.delete` and logs it; the unit test stubs a rejecting `delete`. Plan it together with the missing `/login` component test (`no-test-pins-that-login-awaits-signin-s-cache-clear-before-n.md`) — same page, same test file.
