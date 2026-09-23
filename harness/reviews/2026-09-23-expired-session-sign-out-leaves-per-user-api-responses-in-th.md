---
plan: harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-stale-api-state-cache-survives-into-the-next-account-when-.md, harness/ideas/_inbox/the-api-state-cache-still-has-no-expiry-bound-so-a-stale-res.md, harness/ideas/_inbox/authstore-test-ts-leaves-caches-stubbed-undefined-for-every-.md]
---
# Review — Every sign-out drops the `api-state` cache — `useAuthStore.signOut()` owns the clear

**Plan:** `harness/plans/2026-09-23-expired-session-sign-out-leaves-per-user-api-responses-in-th.md`
**Branch/worktree:** `harness/2026-09-23-high-expired-session-sign-out-leaves-per-user-api-responses-in-th` / `.worktrees/expired-session-sign-out-leaves-per-user-api-responses-in-th`
**Diff:** 8 files, +131 −10 — `frontend/` (4 source, 3 test) and `harness/CODEMAP.md`. `git diff --stat main..HEAD -- backend` is empty; `backend/cmd/api/main.go` is untouched.

## Plan vs idea

The idea asked for two things. The first — *"every path that drops a session also drops the per-user cache … or, better, `useAuthStore.signOut()` owns the cache clear so no future caller can forget it"* — is delivered in the stronger form the idea itself preferred, and is proven by tests that read cache contents rather than call counts.

The second — *"the `api-state` route carries an `ExpirationPlugin` with a `maxAgeSeconds` bound"* — is deliberately deferred, with the reasoning recorded in the plan's *Notes and open questions*: `service-worker/sw.ts` is excluded from `tsconfig.json`, is not unit-tested, and is blocked in Playwright (`serviceWorkers: 'block'`), so the bound would ship unverified. **The deferral is correct**, and the reviewer agrees that a change nothing can observe should not ride along on a change everything can. It is filed as its own idea so the second half of the original is not silently lost, with the browser-level verification requirement written into it.

### Sign-out paths — enumerated, and all three go through `signOut()`

`grep -rn 'signOut'` across `*.ts`/`*.vue` (excluding `node_modules`, `.nuxt`, `.output`) returns exactly three production call sites, and all three now delegate:

| # | Path | Call site | Form |
| --- | --- | --- | --- |
| 1 | Menu sign-out | `frontend/components/AppHeader.vue:11` | `await auth.signOut()` then `navigateTo('/login', { replace: true })` |
| 2 | 401 on any API response | `frontend/composables/useApi.ts:16` | `void auth.signOut()` then `void navigateTo('/login')` |
| 3 | Route guard, expired/absent session | `frontend/middleware/auth.global.ts:13` | `void auth.signOut()` then `return navigateTo('/login', { replace: true })` |

Nothing else clears session state: the only other writer of `accessToken = null` or `removeItem(AUTH_STORAGE_KEY)` is `stores/auth.ts:55-57`, `hydrate()`'s `catch` on malformed persisted JSON. That path discards the stored session without a cache clear, but the guard's `!isAuthenticated` branch runs one statement later on the same tick (`middleware/auth.global.ts:5` then `:12`) and covers it — **on every route except `/login`**.

That `/login` exemption is the one residual hole, and it is on the **sign-in** side, not the sign-out side: `middleware/auth.global.ts:7-11` returns before the `!isAuthenticated` branch, and `stores/auth.ts:59-67` (`signIn()`) never clears the cache. If `/login` is the *first* route of a browsing session while a stale `aelp.auth` is still in `localStorage`, the previous learner's `api-state` entries survive into the next account. It is **not a blocker**: it is pre-existing on `main`, it is narrower than the bug that was fixed (`nuxt.config.ts:50` sets `start_url: '/'`, so the installed PWA always enters through the guard's sign-out branch, and the OAuth return to `/login?code=` is always preceded by a redirect from `/` that already signed out), and merging this branch strictly reduces the leak surface. Filed as a medium inbox bug with the fix shape.

## Code vs plan

All three tasks followed exactly; no deviation from the plan's design. The one reported deviation is confirmed below and is cosmetic.

**Task 1 (regression test, committed failing)** — `tests/unit/fakeCaches.ts` and `tests/unit/authMiddleware.test.ts` are verbatim the plan's listings, committed in `ee92978` ahead of the fix in `4d89a81`.
**Task 2 (store owns the clear)** — `stores/auth.ts:69-75` returns `clearApiCache()` after the synchronous reset; the three callers drop their own `clearApiCache` import and call. No fourth copy in the middleware, as designed.
**Task 3 (CODEMAP)** — the `shell` bullet is replaced with the agreed text; accurate as of this branch.

The `useApi.ts` edit is **minimal and correct**, not merely small: it removes one import line and replaces `auth.signOut(); void clearApiCache()` with `void auth.signOut()`. Behaviour is unchanged (`signOut()`'s state reset is still synchronous, so `getToken: () => auth.accessToken` on `utils/apiClient.ts` returns `null` before the next request, exactly as before; the cache delete was already fire-and-forget on this path). Nothing near `/onboarding` is touched.

`AppHeader.vue` preserves the original ordering — the cache delete still completes before the redirect, because `await auth.signOut()` awaits the promise the store now returns.

### Verification re-run in the worktree (all from `frontend/`, `env -u NUXT_PUBLIC_API_BASE -u PORT -u HOST`)

```
npm run lint       → exit 0
npm run typecheck  → exit 0   (nuxi typecheck, no diagnostics)
npm run test:unit  → exit 0   Test Files 16 passed (16) | Tests 65 passed (65)
npm run build      → exit 0
grep -c 'api-state' .output/public/sw.js → 1
grep -rn 'clearApiCache' … → 3 line-matches in 2 files (stores/auth.ts import + call, utils/session.ts definition)
git diff --stat main..HEAD -- backend → empty
```

CI on the pushed branch, run [35847512334](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35847512334): `frontend`, `backend-unit`, `backend-integration`, `harness-tooling` — all four `success`. No PR exists, and none should; the owner takes one PR per day via `/harness daily-pr`.

### Mutation testing — five mutations, all reproduced by the reviewer

The two the executor reported, plus three of the reviewer's own on paths the executor did not test. Every mutation was reverted; `git diff --quiet` and `git status --short` are clean at the end, and the full suite is green again.

| # | Mutation | Result |
| --- | --- | --- |
| a | `stores/auth.ts:74` `return clearApiCache()` → `return Promise.resolve()` | **Caught.** `authStore` "deletes the service worker api-state cache and nothing else" and `authMiddleware` case 1 both fail `AssertionError: expected true to be false` on `caches.has(API_STATE_CACHE)`. Reproduces the executor's report exactly. |
| b | `utils/session.ts:7` `caches.delete(API_STATE_CACHE)` → `caches.delete('api_state')`, constant left at `'api-state'` | **Caught.** Same two assertions, same message — the tests check the name the worker actually uses, not the constant they seeded with. |
| c | *(reviewer)* `clearApiCache` → `for (const name of await caches.keys()) await caches.delete(name)` — an over-broad clear that wipes every cache | **Caught, and distinguishably.** `authStore` fails `expected [] to deeply equal [ 'assets' ]`; `authMiddleware` fails `expected false to be true` on `caches.has('assets')`. Different failure text from (a) and (b), so the suite tells "cleared nothing" apart from "cleared too much". This settles the load-bearing question: the `assets`-survives assertions are not decoration. |
| d | *(reviewer)* `signOut(): Promise<void> { … return clearApiCache() }` → `signOut() { … void clearApiCache() }`, i.e. `main`'s non-promise shape | **Caught**, by `TypeError: You must provide a Promise to expect() when using .resolves, not 'undefined'`. Confirms the executor's reported deviation verbatim. Note the cache-contents case still passes under this mutation — awaiting `undefined` yields a microtask tick, which is enough for the already-scheduled delete. That is fine (the behaviour is still correct) but worth knowing: the suite asserts the cache is gone, not that `signOut()` is awaitable. |
| e | *(reviewer)* remove the `if (typeof caches === 'undefined') return` guard from `utils/session.ts:6` | **Caught.** `AssertionError: promise rejected "TypeError: Cannot read properties of undefined (reading 'delete')" instead of resolving`. The "still resolves where CacheStorage does not exist" case is load-bearing, not a tautology. |

**On the reported deviation.** Confirmed cosmetic. The plan predicted the `.resolves` case would *pass* pre-fix; on Vitest 3.2.7 it throws a `TypeError` instead, so it failed pre-fix for a different reason than predicted. It fails only where `signOut()` returns a non-promise, and post-fix it passes for the right reason — mutation (e) proves the assertion is still exercising the `typeof caches === 'undefined'` early return and not merely accepting anything. No test or plan logic was changed to accommodate it.

## Quality

**The fake is faithful enough to trust, for what these tests depend on.** `FakeCacheStorage.open()` creates-on-miss and returns the same instance on a second call; `has()` is true for any opened cache including an empty one; `delete()` returns a boolean and removes the entry; `keys()` returns names in insertion order, which matches the real `CacheStorage`'s creation order and is what `toEqual(['assets'])` relies on. `FakeCache.put`/`match` key on the full URL via `keyOf()`, which is the real Cache API's URL-only keying — the very property the original idea identified as the cause of the leak, so the fake models the bug's mechanism rather than modelling around it. Nothing it does is wrong in the same direction as the bug: mutation (c) shows that an over-broad clear, the one failure a sloppy fake would hide, is caught and reported distinctly.

What it omits: `CacheStorage.match()`, and `Cache.add/addAll/delete/keys/matchAll`. None is reachable from `clearApiCache()` or the tests, and the failure mode if a future change reaches for one is a loud `TypeError`, not a silent pass — acceptable. It is not declared `implements CacheStorage`, so TypeScript does not enforce fidelity; declaring it would force implementing the omitted members. Minor, and arguably the right trade for a 50-line helper. Not filed.

**The assertions read the fake, not a spy.** Confirmed in both files: `authStore.test.ts:84-85` asserts `await caches.has(API_STATE_CACHE)` is `false` **and** `await caches.keys()` equals `['assets']`; `authMiddleware.test.ts:34-35` asserts the same `has()` is `false` and `await caches.has('assets')` is still `true`. No `vi.fn` wraps `clearApiCache`, so "the function was called" cannot carry a test. The middleware case additionally asserts `localStorage.getItem(AUTH_STORAGE_KEY)` is `null` and the redirect argument, so it covers the whole guard branch, not just the cache.

**The stub does not leak between files.** Verified, not assumed: a throwaway probe test file added to `tests/unit/` and run in the same suite printed `typeof caches = undefined`, so Vitest's default per-file isolation holds and the 14 unrelated test files still run against the same absent-`caches` environment as before. Nothing else in the suite changed behaviour (16 files / 65 tests, all passing, same as the executor reported). The probe was removed.

Within `authStore.test.ts`, however, the stub *does* persist: the last case sets `vi.stubGlobal('caches', undefined)` and nothing unstubs it — `vitest.config.ts` sets `restoreMocks: true`, which does not cover `vi.stubGlobal`, and `unstubGlobals` is not set. A second probe appended to that file printed `typeof caches = undefined`, confirming it. Any case added below it would silently re-enter exactly the no-op condition this plan exists to eliminate. The direction is the safe one (absent, matching the old baseline, rather than a half-populated fake), so it is low, but it is the same class of silent failure and is filed.

**Conventions, boundaries, CODEMAP.** The store importing `~/utils/session` introduces no cycle (`session.ts` has no imports) and keeps the dependency pointing the right way. `signOut()`'s synchronous-reset-then-return-promise shape preserves the invariant the guard depends on: `isAuthenticated` flips before the first `await`, so `return navigateTo('/login')` is still a synchronous redirect and does not wait on a `caches.delete` a wedged worker could stall. The `void` markers on the two fire-and-forget call sites are the project's existing idiom (`useApi.ts` already used `void navigateTo`). Lint and strict typecheck pass in one pass. The CODEMAP `shell` bullet is accurate for this branch.

**Test honesty.** This is the strongest part of the change. The evaluator's finding that `clearApiCache()` had been early-returning across the entire suite is real, and the plan's response — a contents-asserting fake instead of a spy — is the right one. Five mutations, including three the executor did not run, are all caught, two of them with distinguishable messages. The suite now proves a named cache was deleted and nothing else was.

## Bugs filed

- `harness/ideas/_inbox/a-stale-api-state-cache-survives-into-the-next-account-when-.md` — **medium**. The guard exempts `/login` from the expired-session branch and `signIn()` never clears, so a stale `api-state` can survive into the next account when `/login` is the first route. Pre-existing on `main`, narrower than the fixed bug (`start_url: '/'`), on the sign-in side. Not a blocker.
- `harness/ideas/_inbox/the-api-state-cache-still-has-no-expiry-bound-so-a-stale-res.md` — **low**. The `ExpirationPlugin` half of the original idea, deferred with correct reasoning; filed so it is not lost, with its browser-level verification requirement.
- `harness/ideas/_inbox/authstore-test-ts-leaves-caches-stubbed-undefined-for-every-.md` — **low**. `vi.stubGlobal('caches', undefined)` is never unstubbed in `authStore.test.ts`; contained to that file, but it recreates the silent-no-op condition for any case added after it.

## Verdict

**`pass-with-bugs`.** The plan delivers the idea's primary ask in the stronger form the idea preferred, every sign-out path goes through `useAuthStore.signOut()`, and the tests genuinely prove the cache is gone — five mutations, three of them the reviewer's own, are all caught. Lint, typecheck, the 65-test suite, the build and all four CI jobs reproduce green in the worktree.

**Nothing blocks the merge.** All three filed bugs are medium/low inbox items; none carries `blocks`, and `python3 tools/harness/cli.py blockers --plan <plan>` exits 0.
