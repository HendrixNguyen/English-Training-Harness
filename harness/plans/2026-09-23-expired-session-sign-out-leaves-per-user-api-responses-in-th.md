---
idea: harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md
status: done
priority: high
merged: false
branch: harness/2026-09-23-high-expired-session-sign-out-leaves-per-user-api-responses-in-th
worktree: .worktrees/expired-session-sign-out-leaves-per-user-api-responses-in-th
---
# Every sign-out drops the `api-state` cache — `useAuthStore.signOut()` owns the clear — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/expired-session-sign-out-leaves-per-user-api-responses-in-th.md`
**Goal:** an expired session (the path every user takes daily) deletes the service worker's `api-state` cache exactly like the menu sign-out and the 401 hook do, and no future sign-out path can forget to.

**Why now (`priority: high`):** `api-state` holds `GET /quests/daily` and `GET /pet/status`, keyed on URL only, with no expiry. `middleware/auth.global.ts:13` is the one sign-out path that does not clear it, and it is the *routine* one — `expires_in` is 86400 s, so every learner lapses through it once a day. On a shared device (the family tablet this PWA is for), the next account to sign in is served the previous learner's task list and plant whenever `NetworkFirst` falls back to cache (offline, or network slower than 5 s). Per-user data leak on the happy path; the fix is small.

**Root cause (confirmed on `main`):** three call sites drop a session. `components/AppHeader.vue:12-13` does `auth.signOut(); await clearApiCache()`; `composables/useApi.ts:17-18` does `auth.signOut(); void clearApiCache()`; `middleware/auth.global.ts:13` does `auth.signOut()` only and never imports `~/utils/session`. The cache clear itself is correct where it runs — `utils/session.ts` deletes `'api-state'`, which is the literal `cacheName` in `service-worker/sw.ts:17` (workbox returns an explicit name verbatim, `workbox-core/_private/cacheNames.js getRuntimeName`). The bug is purely that the clear is a caller's responsibility and one caller forgot.

**Architecture — make the store own it:** `useAuthStore.signOut()` resets state and storage synchronously (so `isAuthenticated` flips before the first `await`, exactly as today) and then returns `clearApiCache()`. The three callers stop calling `clearApiCache()` themselves: `AppHeader` does `await auth.signOut()`, `useApi`'s 401 hook and the middleware do `void auth.signOut()`. `middleware/auth.global.ts` therefore needs **no new import** — it is fixed by the store change, and the "one-line" alternative (a fourth copy of the `clearApiCache()` call in the middleware) is rejected because it leaves the responsibility distributed, which is how this bug happened. `stores/auth.ts` importing `~/utils/session` introduces no cycle (`session.ts` has no imports). `service-worker/sw.ts` is untouched.

**Test approach — prove the cache is gone, not that a function was called.** No existing test covers the clear (`grep -rn 'clearApiCache\|caches' frontend/tests` → nothing), and Node 22 / happy-dom expose no `caches` global (`clearApiCache` early-returns on `typeof caches === 'undefined'`, which is why the suite has been silent). The new tests install a small **Map-backed `CacheStorage` fake** with real `open/put/match/has/delete/keys` semantics via `vi.stubGlobal('caches', …)`, seed an `api-state` entry for `/api/v1/quests/daily` *and* an `assets` entry, run the code under test, and assert on **the fake's state**: `await caches.has('api-state')` is `false` and `await caches.has('assets')` is still `true`. No `vi.fn` wraps `clearApiCache`, so a test cannot pass by "the function was invoked" alone — it passes only if the named cache was actually deleted and nothing else was. The middleware is exercised through its real default export (auto-imports `defineNuxtRouteMiddleware`/`navigateTo` are free identifiers in a `.ts` file, so `vi.stubGlobal` before a dynamic import is enough — the same trick `revivePage.test.ts` uses with `mocks: { navigateTo }` for a template).

**Tech stack:** as on `main`; Vitest + happy-dom; no new dependencies. Frontend-only — **`backend/` and `backend/cmd/api/main.go` are not touched.**

**Run every command from `frontend/` inside the worktree.** `npm ci` first. `rg` is not installed — use `grep -n`. Strict TypeScript; lint clean in one pass (ESLint stylistic: no semicolons, single quotes, trailing commas).

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/tests/unit/fakeCaches.ts` | **New**: Map-backed `CacheStorage` fake shared by the two tests |
| `frontend/tests/unit/authMiddleware.test.ts` | **New**: the expired-session regression test through the real middleware |
| `frontend/tests/unit/authStore.test.ts` | Add two cases: `signOut()` deletes `api-state` only; `signOut()` resolves when `caches` is undefined |
| `frontend/stores/auth.ts` | `signOut()` returns `clearApiCache()` after the synchronous reset |
| `frontend/components/AppHeader.vue` | `await auth.signOut()`; drop the `clearApiCache` import and call |
| `frontend/composables/useApi.ts` | `void auth.signOut()`; drop the `clearApiCache` import and call |
| `frontend/middleware/auth.global.ts` | `void auth.signOut()` (marks the returned promise as deliberately unawaited) |
| `harness/CODEMAP.md` | `shell` bullet: the cache is cleared by `useAuthStore.signOut()` on every path |

---

## Tasks

### Task 1: The regression test, failing on `main`

**Files:**
- Create: `frontend/tests/unit/fakeCaches.ts`
- Create: `frontend/tests/unit/authMiddleware.test.ts`

- [ ] **Step 1: Write the fake**

```ts
/**
 * Map-backed stand-in for the browser CacheStorage — Node and happy-dom expose
 * none. Real semantics for what utils/session.ts and the tests use, no spies:
 * a test proves a cache is gone by asking `has()`, not by counting calls.
 */
class FakeCache {
  private readonly entries = new Map<string, Response>()
  async put(request: RequestInfo | URL, response: Response): Promise<void> {
    this.entries.set(keyOf(request), response)
  }
  async match(request: RequestInfo | URL): Promise<Response | undefined> {
    return this.entries.get(keyOf(request))
  }
}

function keyOf(request: RequestInfo | URL): string {
  return typeof request === 'string' ? request : request instanceof URL ? request.href : request.url
}

export class FakeCacheStorage {
  private readonly stores = new Map<string, FakeCache>()
  async open(name: string): Promise<FakeCache> {
    let c = this.stores.get(name)
    if (!c) {
      c = new FakeCache()
      this.stores.set(name, c)
    }
    return c
  }
  async has(name: string): Promise<boolean> {
    return this.stores.has(name)
  }
  async delete(name: string): Promise<boolean> {
    return this.stores.delete(name)
  }
  async keys(): Promise<string[]> {
    return [...this.stores.keys()]
  }
}

/** Installs a fresh fake as `caches` with one per-user entry and one asset entry seeded. */
export async function installSeededCaches(apiStateName: string): Promise<FakeCacheStorage> {
  const fake = new FakeCacheStorage()
  vi.stubGlobal('caches', fake)
  await (await fake.open(apiStateName)).put('http://api.test/api/v1/quests/daily', new Response('{"day_number":3}'))
  await (await fake.open('assets')).put('http://app.test/_nuxt/entry.js', new Response('// js'))
  return fake
}
```
(Add `import { vi } from 'vitest'` at the top.) `vitest.config.ts` includes only `tests/unit/**/*.test.ts`, so this helper is not collected as a test file.

- [ ] **Step 2: Write the middleware test**

```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'
import { AUTH_STORAGE_KEY } from '~/stores/auth'
import { API_STATE_CACHE } from '~/utils/session'
import { installSeededCaches } from './fakeCaches'

const navigateTo = vi.fn()
vi.stubGlobal('defineNuxtRouteMiddleware', <T>(fn: T) => fn)
vi.stubGlobal('navigateTo', navigateTo)

const { default: guard } = await import('~/middleware/auth.global')

const user = { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' }
const to = { path: '/', query: {} } as RouteLocationNormalized

function persistSession(expiresAt: number) {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify({ accessToken: 't', expiresAt, user }))
}

describe('middleware/auth.global — expired session (the daily sign-out path)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    navigateTo.mockReset()
  })

  it('drops the per-user api-state cache, keeps the assets cache, and redirects to /login', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() - 1) // expires_in elapsed since the last visit

    guard(to, to)

    await vi.waitFor(async () => expect(await caches.has(API_STATE_CACHE)).toBe(false))
    expect(await caches.has('assets')).toBe(true)
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
    expect(navigateTo).toHaveBeenCalledWith('/login', { replace: true })
  })

  it('leaves the cache alone while the session is valid', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() + 60_000)

    guard(to, to)
    await Promise.resolve()

    expect(await caches.has(API_STATE_CACHE)).toBe(true)
    expect(navigateTo).not.toHaveBeenCalled()
  })
})
```
If `nuxi typecheck` objects to the `guard(to, to)` call signature, cast the import once (`as (to: RouteLocationNormalized, from: RouteLocationNormalized) => unknown`) and record it; do not loosen `tsconfig`.

- [ ] **Step 3: Run it and confirm the first case fails for the right reason**

Run: `npx vitest run tests/unit/authMiddleware.test.ts`
Expected: case 1 fails at `vi.waitFor` — `caches.has('api-state')` stays `true` because the middleware never clears it; case 2 passes. If case 1 passes, stop — you are not on the code this plan was written against.

- [ ] **Step 4: Commit the failing test**

```bash
git add tests/unit/fakeCaches.ts tests/unit/authMiddleware.test.ts
git commit -m "auth: regression test — an expired session must drop the api-state cache"
```

---

### Task 2: `signOut()` owns the cache clear; callers stop duplicating it

**Files:**
- Modify: `frontend/stores/auth.ts`, `frontend/components/AppHeader.vue`, `frontend/composables/useApi.ts`, `frontend/middleware/auth.global.ts`
- Modify: `frontend/tests/unit/authStore.test.ts`

- [ ] **Step 1: Extend `authStore.test.ts` first**

Add `import { API_STATE_CACHE } from '~/utils/session'`, `import { installSeededCaches } from './fakeCaches'`, and `vi` to the vitest import; then two cases inside the existing `describe`:

```ts
  it('signOut deletes the service worker api-state cache and nothing else', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())

    await auth.signOut()

    expect(await caches.has(API_STATE_CACHE)).toBe(false)
    expect(await caches.keys()).toEqual(['assets'])
    expect(auth.isAuthenticated).toBe(false)
  })

  it('signOut still resolves where CacheStorage does not exist', async () => {
    vi.stubGlobal('caches', undefined)
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())
    await expect(auth.signOut()).resolves.toBeUndefined()
    expect(auth.accessToken).toBeNull()
  })
```
Run `npx vitest run tests/unit/authStore.test.ts` → the first new case fails (`api-state` still present); the second passes today (`signOut()` returns `undefined`, which `resolves` accepts).

- [ ] **Step 2: `stores/auth.ts`**

Add `import { clearApiCache } from '~/utils/session'` and change the action:

```ts
    /** Drops the session, then the per-user service worker cache — every sign-out path goes through here. */
    signOut(): Promise<void> {
      this.accessToken = null
      this.expiresAt = null
      this.user = null
      storageOrNull()?.removeItem(AUTH_STORAGE_KEY)
      return clearApiCache()
    },
```

- [ ] **Step 3: The three callers**

- `components/AppHeader.vue`: remove `import { clearApiCache } from '~/utils/session'`; the function becomes
  ```ts
  async function signOut() {
    await auth.signOut()
    await navigateTo('/login', { replace: true })
  }
  ```
- `composables/useApi.ts`: remove the `clearApiCache` import; `onUnauthorized` becomes
  ```ts
    onUnauthorized: () => {
      void auth.signOut()
      void navigateTo('/login')
    },
  ```
- `middleware/auth.global.ts`: `auth.signOut()` → `void auth.signOut()` (the guard must return the redirect synchronously; the cache delete completes on its own).

- [ ] **Step 4: Run the two test files, then the suite, then lint/typecheck/build**

```bash
npx vitest run tests/unit/authMiddleware.test.ts tests/unit/authStore.test.ts   # 2 + 7 passed
npm run test:unit                                                              # all passed
npm run lint && npm run typecheck && npm run build                             # exit 0 each
grep -rn 'clearApiCache' --include='*.ts' --include='*.vue' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output
```
Expected for the grep: exactly two hits — the definition in `utils/session.ts` and the call in `stores/auth.ts`.

- [ ] **Step 5: Commit**

```bash
git add stores/auth.ts components/AppHeader.vue composables/useApi.ts middleware/auth.global.ts tests/unit/authStore.test.ts
git commit -m "auth: signOut() owns the api-state cache clear so the expired-session path drops per-user data too"
```

---

### Task 3: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`shell` bullet only)

- [ ] **Step 1:** Replace `` `api-state` cache — cleared by `utils/session.ts` on sign-out `` with `` `api-state` cache — deleted by `useAuthStore.signOut()` (via `utils/session.ts`) on **every** sign-out path: menu, 401 hook, and the route guard's expired-session branch (fix 2026-09-23); callers never clear it themselves ``.

- [ ] **Step 2: Commit**

```bash
git add ../harness/CODEMAP.md
git commit -m "CODEMAP: signOut() owns the api-state cache clear"
```

---

## Verification

From `frontend/` in the worktree, clean shell (`env -u NUXT_PUBLIC_API_BASE -u PORT -u HOST`):

1. `npm ci && npm run lint && npm run typecheck` → exit 0 each, in one pass.
2. `npm run test:unit` → all pass, including `tests/unit/authMiddleware.test.ts` (2) and the two new `authStore.test.ts` cases.
3. **Mutation checks (paste the output into the execution summary):**
   - **Revert the fix:** in `stores/auth.ts` change `return clearApiCache()` to `return Promise.resolve()`. `npx vitest run tests/unit/authMiddleware.test.ts tests/unit/authStore.test.ts` → **`authMiddleware` case 1 must fail at `vi.waitFor(… caches.has(API_STATE_CACHE) … toBe(false))`** and **`authStore` "deletes the service worker api-state cache" must fail on the same assertion**. Restore.
   - **Wrong cache name:** in `utils/session.ts` change only the argument inside `clearApiCache` to the literal `'api_state'`, leaving the exported `API_STATE_CACHE` constant (which the tests seed) as `'api-state'`. Run again → the same two assertions must fail (the seeded `api-state` survives), proving the tests check the name the service worker actually uses. Restore; `git diff --quiet stores/auth.ts utils/session.ts` clean; re-run → all green.
4. `grep -rn 'clearApiCache' --include='*.ts' --include='*.vue' . --exclude-dir=node_modules --exclude-dir=.nuxt --exclude-dir=.output` → two hits (`utils/session.ts`, `stores/auth.ts`).
5. `npm run build` → exit 0; `grep -c 'api-state' .output/public/sw.js` → 1 (the worker's cache name is unchanged).
6. `git diff --stat main..HEAD -- ../backend` → empty.
7. Push; `gh run list --branch <branch>` — `frontend`, `backend-unit`, `backend-integration`, `harness-tooling` all green.

## Notes and open questions

- **`ExpirationPlugin` deliberately not included.** The idea suggests bounding `api-state` with `workbox-expiration` as a second line of defence. That is a new dependency plus a change to `service-worker/sw.ts`, which is excluded from `tsconfig.json`, not unit-tested, and blocked in Playwright (`serviceWorkers: 'block'`) — i.e. it would ship unverified. With `signOut()` owning the clear there is no sign-out path left that keeps the cache, so the expiry would only shrink a window that no longer exists. If the owner still wants a `maxAgeSeconds` bound (e.g. for a device that is never signed out), it should be its own small idea with a browser-level verification step.
- **Why `void auth.signOut()` in the guard rather than `await`:** `defineNuxtRouteMiddleware` may return a promise, but the redirect must not wait on a `caches.delete` that a broken service worker could stall; the state reset that gates `isAuthenticated` is synchronous, and the cache delete is idempotent, so fire-and-forget is correct here.
- **Behaviour on `/login` after a lapse** is unchanged: `/login` fetches nothing from `api-state` URLs, and by the time the next account's dashboard requests `/quests/daily` the delete has long completed.

## Execution summary

Executed 2026-09-23 in `.worktrees/expired-session-sign-out-leaves-per-user-api-responses-in-th` on branch `harness/2026-09-23-high-expired-session-sign-out-leaves-per-user-api-responses-in-th`. All three tasks implemented exactly as specified — no deviation from the plan's design (store-owned `signOut()`, no fourth `clearApiCache()` copy in the middleware).

**Agree with the store-ownership design.** Distributing the clear across three callers is exactly how the middleware forgot it in the first place; putting it in `signOut()` means any future sign-out path (there is only one action to call) can't repeat the bug. The rejected one-liner would have fixed today's instance without fixing the pattern.

**Commits:**
- `ee92978` — regression test (fakeCaches.ts, authMiddleware.test.ts), committed failing
- `4d89a81` — the fix: `stores/auth.ts` returns `clearApiCache()`; `AppHeader.vue`, `useApi.ts`, `middleware/auth.global.ts` updated; `authStore.test.ts` extended
- `1b1503c` — CODEMAP shell bullet updated

**Verification (all from `frontend/`, clean shell):**
1. `npm ci && npm run lint && npm run typecheck` — exit 0 each.
2. `npm run test:unit` — 16 files, 65 tests, all passed (includes `authMiddleware.test.ts` 2/2 and `authStore.test.ts` 7/7, i.e. the 2 new cases).
3. **Mutation checks:**
   - Revert (`return clearApiCache()` → `return Promise.resolve()`): both `authMiddleware` case 1 and `authStore` "deletes the service worker api-state cache" failed identically — `AssertionError: expected true to be false` at `expect(await caches.has(API_STATE_CACHE)).toBe(false)` / the equivalent `vi.waitFor` line. Restored; `git diff --quiet` clean; re-ran green.
   - Wrong cache name (`utils/session.ts`: `caches.delete(API_STATE_CACHE)` → `caches.delete('api_state')`, `API_STATE_CACHE` constant left at `'api-state'`): the same two assertions failed the same way (seeded `api-state` entry survives). Restored; `git diff --quiet stores/auth.ts utils/session.ts` clean; re-ran green.
   - **The `assets` entry surviving is genuinely asserted**, not incidental: `authMiddleware.test.ts` asserts `expect(await caches.has('assets')).toBe(true)` and `authStore.test.ts` asserts `expect(await caches.keys()).toEqual(['assets'])` — a clear that wiped every cache would fail `toEqual(['assets'])`, since `keys()` would be `[]`. Neither mutation above touched this assertion; it passed unchanged both times, confirming a global-wipe bug is a distinct failure this suite would catch.
4. `grep -rn 'clearApiCache' ...` → 3 line-matches across the 2 expected files (`stores/auth.ts`: import + call; `utils/session.ts`: definition). No other caller remains.
5. `npm run build` — exit 0; `grep -c 'api-state' .output/public/sw.js` → 1.
6. `git diff --stat main..HEAD -- ../backend` → empty. Backend untouched.
7. Pushed; CI run [35847512334](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35847512334) — `backend-integration`, `harness-tooling`, `frontend` (lint/typecheck/unit/build all ran), `backend-unit` all green.

**Runtime proof:** Built app served via `node .output/server/index.mjs` on port 3106 (`NUXT_PUBLIC_STUB_ONBOARDING=true`). `curl http://localhost:3106/` → HTTP 200. Playwright: (a) fresh browser, no session, navigate to `/` → redirected to `/login`. (b) `localStorage.setItem('aelp.auth', {accessToken:'t', expiresAt: Date.now()-1000, user})` (an expired session), navigate to `/` → redirected to `/login`, confirming the guard's expired-session branch runs end to end against the built app, not just under Vitest. Server stopped afterward; `pgrep -fl "node .output/server/index.mjs"` empty; no other node dev-server processes were touched (verified against the full process list — only my own preview process existed under that command line).

**Deviations:**
- Plan step "Task 2 Step 1" predicted the "signOut still resolves where CacheStorage does not exist" case would pass before the fix (since `signOut()` returned `undefined` and `.resolves` "accepts" that). On this project's Vitest 3.2.7, `expect(nonPromise).resolves` throws a `TypeError` instead of passing. Cosmetic only — the test still failed pre-fix as intended and passed post-fix identically; no plan or test logic changed.
- One observation, not a deviation: my final `browser_close` call in Playwright reported closing a page on `http://127.0.0.1:3105/login`, a port I never navigated to (I only used `localhost:3106`) and one the task instructions flagged as belonging to another agent. I did not inspect or manage anything on 3105 beyond that automatic tool report, and confirmed via full `pgrep` process listing that no server on that port was killed by my `pkill -f "node .output/server/index.mjs"` (scoped by exact command line, and no such other process existed at the time). Flagging for awareness in case the shared browser tooling is cross-agent.
