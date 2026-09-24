---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# authStore.test.ts leaves caches stubbed undefined for every test added after it

## Why
The plan `harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md` exists because `clearApiCache()` silently early-returned across the whole suite — no `caches` global under Node 22 / happy-dom meant every call was a no-op and no test ever verified a cache clear. The new tests fix that by installing a Map-backed fake.

The last case in `frontend/tests/unit/authStore.test.ts` deliberately stubs the global back to `undefined` to exercise the `typeof caches === 'undefined'` guard, and nothing ever unstubs it. `frontend/vitest.config.ts` sets `restoreMocks: true`, which restores spies and mocks but has no effect on `vi.stubGlobal`; `unstubGlobals` is not set, and there is no `vi.unstubAllGlobals()` in an `afterEach`. Any case appended to that `describe` therefore runs with no `caches` global and recreates exactly the silent-no-op condition this plan was written to eliminate — and it would pass, because the code under test would early-return.

Verified during review by appending a throwaway probe case to the file: it printed `typeof caches = undefined`. The same probe as a separate test file printed `undefined` too, confirming the stub does **not** leak across files (Vitest's default per-file isolation holds), so the hazard is contained to this one file. The probe was removed; `git diff --quiet` is clean.

The direction of the leak is the safe one today — a later test sees the same absent `caches` as the pre-change baseline rather than a half-populated fake — which is why this is low, not a blocker.

## Expected output
A test added to the bottom of `authStore.test.ts` sees the same global state as one added to the top. Either `afterEach(() => vi.unstubAllGlobals())` in that file, or `unstubGlobals: true` in `frontend/vitest.config.ts` so every `vi.stubGlobal` is reverted between cases suite-wide (this is the stronger option and costs nothing, since `installSeededCaches` re-stubs per case anyway).

## Evidence
- `frontend/tests/unit/authStore.test.ts:89-94` — `it('signOut still resolves where CacheStorage does not exist', ...)` calls `vi.stubGlobal('caches', undefined)` and never unstubs.
- `frontend/vitest.config.ts:10-14` — `test: { environment: 'happy-dom', include: [...], restoreMocks: true }`; no `unstubGlobals`.
- `frontend/tests/unit/fakeCaches.ts:45-50` — `installSeededCaches()` calls `vi.stubGlobal('caches', fake)` and leaves the revert to the caller.
- `frontend/utils/session.ts:5-8` — the `if (typeof caches === 'undefined') return` early return that makes a missing global silent rather than loud.
- Reviewed plan: `harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md`, *Test approach*.
