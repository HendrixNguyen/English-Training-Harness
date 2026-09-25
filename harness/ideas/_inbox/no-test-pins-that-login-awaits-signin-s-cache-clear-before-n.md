---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# No test pins that login awaits signIn's cache clear before navigating to the hub

## Why
The plan's Design decision 3 rests on `pages/login.vue:35` awaiting `auth.signIn(res)` so the `api-state` cache is gone before `navigateTo('/')` and the hub cannot read the previous account's `/quests/daily` or `/pet/status` from the NetworkFirst fallback. No unit test mounts `pages/login.vue` (`frontend/tests/unit/` has none), and the e2e `login.spec.ts` blocks service workers, so reverting line 35 to `auth.signIn(res)` leaves the whole suite green. The store-level test (`authStore.test.ts` "signIn drops the previous account's api-state cache") proves the clear happens, not that `/login` waits for it. The executor's mutation table has no row for this line.

Low: the window is short and the stale entry is read only when the network fails, but it is exactly the per-user leak the sibling idea fixed.

## Expected output
A component test for `pages/login.vue` in the shape of `onboardingPage.test.ts`: seed the Map-backed fake from `tests/unit/fakeCaches.ts` with `api-state`, set `sessionStorage['aelp.oauth_state']` and a route query with matching `code`/`state`, mock `useApi().post` to resolve the §6.1 body, stub `navigateTo` to record `await caches.has(API_STATE_CACHE)` at call time, mount, flush, and assert it was `false` when `navigateTo('/')` ran. Dropping the `await` on line 35 must turn it red.

## Evidence
- Plan under review: `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` (Task 2 Step 3, Design decision 3, Verification mutation table).
- `frontend/pages/login.vue:35`; `ls frontend/tests/unit/` — no login page test; `frontend/tests/e2e/login.spec.ts` (service workers blocked per CODEMAP `shell`).

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.** A real gap (no unit test mounts `pages/login.vue`; the e2e suite blocks service workers), no behaviour change. Plan it with `a-rejected-cachestorage-delete-now-fails-sign-in-for-a-user-.md` as one `/login` branch: the component test the reviewer describes is also the natural place to prove the best-effort cache clear.
