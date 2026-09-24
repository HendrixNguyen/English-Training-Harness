---
idea: harness/ideas/_inbox/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md
status: done
priority: medium
merged: false
branch: harness/2026-09-24-medium-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa
worktree: .worktrees/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa
---
# frontend: close the 2026-09-23 review follow-ups on `/onboarding`, sign-in and `/revive` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/onboarding-s-ai-error-copy-is-untested-deleting-the-whole-br.md` → Task 1
- `harness/ideas/_inbox/a-stale-api-state-cache-survives-into-the-next-account-when-.md` → Task 2
- `harness/ideas/_inbox/authstore-test-ts-leaves-caches-stubbed-undefined-for-every-.md` → Task 3
- `harness/ideas/_inbox/the-revive-missed-days-line-renders-bo-hoc2-ngay-with-no-spa.md` → Task 4
- `harness/ideas/_inbox/no-test-pins-the-stale-data-wins-branch-order-on-revive-so-a.md` → Task 5

**Goal:** A learner can no longer post a malformed reminder time from `/onboarding` and loop on an opaque error after ten answered questions; every error code the assessment can return renders copy that is pinned by a test; signing *in* leaves no other account's `api-state` cache behind (the last open door after the sign-out fix); the wilted screen's headline reads "bỏ học 2 ngày" with its space; and the two `/revive` behaviours the reviewer found untested — stale-data-wins ordering and the file-level `caches` stub — are held in place.

**Why now (`priority: medium`):** onboarding is the last step before the first quest and its failure mode has no exit but a reload that loses the answers; the sign-in cache leak is a per-user data leak on shared devices; the rest are small, user-visible or test-integrity fixes in the same three screens that the 2026-09-23 reviews already proved with mutations. Six ideas, one branch: each is a handful of lines plus a regression test, all in `frontend/`.

**Root causes (from each idea's `## Evaluation`, re-read on this branch):**
- `frontend/pages/onboarding.vue`: `time` (line 24) is bound to an unguarded `<input type="time">` (95); the start button is `:disabled="!goal"` only (97); `next()` posts `` `${time.value}:00` `` (64) → `":00"` → backend `400 invalid_request`; `assessErrorMessage` (13-17) has no branch for it. The time input lives only on the goal step, so a gate on the start button is sufficient.
- `frontend/tests/unit/onboardingPage.test.ts`: one error case (429); deleting `onboarding.vue:15` (the `ai_*` branch) leaves the suite green.
- `frontend/middleware/auth.global.ts:7-11` returns before the sign-out branch on `/login`; `frontend/stores/auth.ts` `signIn()` (59-67) never clears the cache; `pages/login.vue:35` navigates right after `signIn`.
- `frontend/tests/unit/authStore.test.ts:90` stubs `caches` to `undefined` with no `afterEach`; `vitest.config.ts` has `restoreMocks` only.
- `frontend/pages/revive.vue:83`: the space before `{{ missedDays }}` is the first text node inside `<template v-if>`, which Vue's `whitespace: 'condense'` strips.
- `frontend/pages/revive.vue:69` (`pet.status`) above `:109` (`pet.error`) is the behaviour; `revivePage.test.ts` never combines the two.

**Design decisions (read before the tasks):**
1. **Gate, do not fall back.** The start button is disabled while `time` is not `HH:MM`, with a one-line hint under the field — the same pattern the goal already uses. Silently substituting `20:00` would post a time the learner did not choose.
2. **`invalid_request` gets its own copy** naming the two fields the learner controls (goal, reminder time). It is defence in depth once the gate exists (the backend validates more than the time).
3. **`signIn()` owns a clear, symmetrical with `signOut()`.** State and storage are written synchronously as today (existing callers and tests keep working); the method then returns `clearApiCache()`'s promise, and `login.vue` awaits it before `navigateTo('/')` so the hub cannot read the stale entry first. The guard's `/login` branch additionally drops a session that is *present but expired*, so `localStorage` and the cache are clean before the consent redirect — the door the idea describes.
4. **Unstub per file, not suite-wide.** `afterEach(() => vi.unstubAllGlobals())` in `authStore.test.ts` only. `unstubGlobals: true` in `vitest.config.ts` would also revert the top-level `vi.stubGlobal('navigateTo', …)` / `defineNuxtRouteMiddleware` stubs in `onboardingPage.test.ts` and `authMiddleware.test.ts` between cases and break them.
5. **The revive sentence is computed in script** (`missedLine`), so the template interpolates one string and whitespace condensing has nothing to eat.
6. **No design doc.** These are fixes inside existing screens (`harness/designs/frontend-shell.md` still describes them); the only new UI is a one-line hint under an existing field, styled with the existing `text-alert` token.

**Tech stack:** Nuxt 3 / Vue 3 / Pinia (resolved versions in CODEMAP), Vitest 3 + happy-dom + `@vue/test-utils`. No new dependencies.

**Run every command from `frontend/` inside the worktree.** First `npm ci` (and `npx playwright install chromium` only if you run the optional e2e line). `rg` is not installed — use `grep -n`.

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/pages/onboarding.vue` | `timeValid`/`canStart`; hint under the time field; gated start; `invalid_request` copy |
| `frontend/tests/unit/onboardingPage.test.ts` | Cleared-time gate case; `400 invalid_request` case; `it.each` over the three `ai_*` codes |
| `frontend/stores/auth.ts` | `signIn()` returns `clearApiCache()` after writing the session |
| `frontend/pages/login.vue` | `await auth.signIn(res)` |
| `frontend/middleware/auth.global.ts` | `/login` branch signs out a present-but-expired session |
| `frontend/tests/unit/authStore.test.ts` | `afterEach(vi.unstubAllGlobals)`; `signIn` clears case; trailing no-leak case |
| `frontend/tests/unit/authMiddleware.test.ts` | `/login` + expired session case; `/login` + no session case |
| `frontend/pages/revive.vue` | `missedLine` computed; template interpolates it |
| `frontend/tests/unit/revivePage.test.ts` | Rendered-text case (3 days / null); stale-status-wins case; failed-revive case |
| `harness/CODEMAP.md` | `shell` paragraph: onboarding gate + copy; `signIn()` clears; guard on `/login` |

---

## Tasks

### Task 1: `/onboarding` — gate the start on a valid reminder time; name `invalid_request`; pin the `ai_*` copy

**Files:**
- Modify: `frontend/tests/unit/onboardingPage.test.ts`
- Modify: `frontend/pages/onboarding.vue`

- [ ] **Step 1: Write the failing tests** (append inside the existing `describe`)

```ts
  it('does not start the quiz while the reminder time is cleared, and says why', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await w.find('input[type="time"]').setValue('')

    const start = w.findAll('button').find(b => b.text().includes('Bắt đầu bài kiểm tra'))
    if (!start) throw new Error('no start button')
    expect(start.attributes('disabled')).toBeDefined()
    expect(w.find('[role="note"]').text()).toContain('giờ nhắc học')
    await start.trigger('click')
    await flushPromises()
    expect(api.get).not.toHaveBeenCalledWith('/api/v1/onboarding/quiz')

    await w.find('input[type="time"]').setValue('07:30')
    expect(start.attributes('disabled')).toBeUndefined()
    expect(w.find('[role="note"]').exists()).toBe(false)
  })

  it('names a 400 invalid_request so the learner knows what to fix, keeping their answers', async () => {
    api.post.mockRejectedValue(new ApiError(400, 'invalid_request'))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    const alert = w.find('[role="alert"]').text()
    expect(alert).toContain('giờ nhắc học')
    expect(alert).not.toContain('Không tạo được lộ trình')
    expect(w.text()).toContain('Câu 10 / 10')
  })

  it.each([
    [503, 'ai_unavailable'],
    [502, 'ai_bad_output'],
    [502, 'ai_upstream_failed'],
  ])('renders the AI-specific copy for %i %s and keeps the learner on the quiz', async (status, code) => {
    api.post.mockRejectedValue(new ApiError(status, code))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(w.find('[role="alert"]').text()).toContain('Máy chủ AI đang bận')
    expect(w.text()).toContain('Câu 10 / 10')
    expect(w.text()).not.toContain('Trình độ của bạn')
  })
```

- [ ] **Step 2: Run to see them fail**

Run: `npx vitest run tests/unit/onboardingPage.test.ts`
Expected: the gate case fails (`disabled` undefined, no `[role="note"]`), the `invalid_request` case fails (generic copy); the three `ai_*` cases already pass — they are the mutation guard, proven in Step 5.

- [ ] **Step 3: Implement in `onboarding.vue`**

Script — after `const time = ref('20:00')`:

```ts
/** <input type="time"> yields HH:MM, or '' once cleared; §6.1 wants HH:MM:SS, built in next(). */
const TIME_RE = /^\d{2}:\d{2}$/
const timeValid = computed(() => TIME_RE.test(time.value))
const canStart = computed(() => goal.value !== null && timeValid.value)
```

`startQuiz()`: change `if (!goal.value) return` to `if (!canStart.value) return`.

`assessErrorMessage` — insert before the generic fallback:

```ts
  if (e instanceof ApiError && e.code === 'invalid_request') return 'Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu và giờ nhắc học rồi thử lại.'
```

Template — the time field and the start button:

```vue
      <label class="mt-6 block">
        <span class="text-sm text-mute">Chọn giờ nhắc học hằng ngày</span>
        <input v-model="time" type="time" required :aria-invalid="!timeValid || undefined" class="mt-1 block w-full rounded-btn border border-ink/15 bg-transparent px-3 py-2 dark:border-paper/15">
        <span v-if="!timeValid" class="mt-1 block text-sm text-alert" role="note">Chọn một giờ nhắc học để tiếp tục.</span>
      </label>
      <AppButton class="mt-6" block :disabled="!canStart" :loading="loading" @click="startQuiz">
```

- [ ] **Step 4: Run the file**

Run: `npx vitest run tests/unit/onboardingPage.test.ts`
Expected: 8 passed (3 existing + gate + invalid_request + 3 `ai_*`).

- [ ] **Step 5: Prove the `ai_*` guard has teeth**

Delete the `e.code.startsWith('ai_')` line from `assessErrorMessage`; rerun → the three `it.each` cases FAIL (`Máy chủ AI đang bận` absent). Restore; rerun → 8 passed.

- [ ] **Step 6: Lint, typecheck, commit**

```bash
npm run lint && npm run typecheck
git add pages/onboarding.vue tests/unit/onboardingPage.test.ts
git commit -m "onboarding: gate the quiz on a valid reminder time; name invalid_request; pin the ai_* copy"
```

### Task 2: Signing in clears the previous account's cache; `/login` drops an expired session

**Files:**
- Modify: `frontend/tests/unit/authStore.test.ts`, `frontend/tests/unit/authMiddleware.test.ts`
- Modify: `frontend/stores/auth.ts`, `frontend/pages/login.vue`, `frontend/middleware/auth.global.ts`

- [ ] **Step 1: Write the failing tests**

`authStore.test.ts` — append inside the `describe`:

```ts
  it('signIn drops the previous account\'s api-state cache and keeps assets', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const auth = useAuthStore()

    await auth.signIn(spec61, Date.now())

    expect(await caches.has(API_STATE_CACHE)).toBe(false)
    expect(await caches.keys()).toEqual(['assets'])
    expect(auth.isAuthenticated).toBe(true)
  })
```

`authMiddleware.test.ts` — append inside the `describe`:

```ts
  it('drops an expired session and its cache when /login is the first route, without redirecting', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() - 1) // a bookmark straight to /login on a shared device
    const login = { path: '/login', query: {} } as RouteLocationNormalized

    guard(login, login)

    await vi.waitFor(async () => expect(await caches.has(API_STATE_CACHE)).toBe(false))
    expect(await caches.has('assets')).toBe(true)
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('leaves /login alone when there is no session at all', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const login = { path: '/login', query: {} } as RouteLocationNormalized

    guard(login, login)
    await Promise.resolve()

    expect(await caches.has(API_STATE_CACHE)).toBe(true) // nothing to drop; signIn() will
    expect(navigateTo).not.toHaveBeenCalled()
  })
```

- [ ] **Step 2: Run to see them fail**

Run: `npx vitest run tests/unit/authStore.test.ts tests/unit/authMiddleware.test.ts`
Expected: `signIn drops …` FAIL (`has(API_STATE_CACHE)` true); `/login is the first route` FAIL (cache still there / storage not cleared). The no-session case passes.

- [ ] **Step 3: Implement**

`stores/auth.ts` `signIn`:

```ts
    /**
     * Stores the new session, then drops any previous account's cached API
     * responses — the mirror of signOut(). State and storage are written
     * synchronously; the returned promise is the cache clear, and /login
     * awaits it before navigating so the hub never reads a stale entry.
     */
    signIn(res: SignInResponse, now: number = Date.now()): Promise<void> {
      if (typeof res.access_token !== 'string' || typeof res.expires_in !== 'number') return Promise.resolve()
      this.accessToken = res.access_token
      this.expiresAt = now + res.expires_in * 1000
      this.user = res.user
      this.hydrated = true
      const p: Persisted = { accessToken: this.accessToken, expiresAt: this.expiresAt, user: res.user }
      storageOrNull()?.setItem(AUTH_STORAGE_KEY, JSON.stringify(p))
      return clearApiCache()
    },
```

`pages/login.vue` line 35: `auth.signIn(res)` → `await auth.signIn(res)`.

`middleware/auth.global.ts`:

```ts
  if (to.path === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    // A session that is present but expired is dropped here too — with its
    // api-state cache — so the next account never inherits it (fix 2026-09-24).
    if (auth.accessToken !== null && !auth.isAuthenticated) void auth.signOut()
    return
  }
```

- [ ] **Step 4: Run the two files**

Run: `npx vitest run tests/unit/authStore.test.ts tests/unit/authMiddleware.test.ts`
Expected: all pass (the pre-existing `signIn` tests still call it synchronously and read state immediately — unchanged).

- [ ] **Step 5: Lint, typecheck, commit**

```bash
npm run lint && npm run typecheck
git add stores/auth.ts pages/login.vue middleware/auth.global.ts tests/unit/authStore.test.ts tests/unit/authMiddleware.test.ts
git commit -m "auth: signIn clears api-state; /login drops an expired session before consent"
```

### Task 3: `authStore.test.ts` unstubs globals between cases

**Files:**
- Modify: `frontend/tests/unit/authStore.test.ts`

- [ ] **Step 1: Add the trailing test first** (last `it` in the `describe`; import `afterEach` from vitest)

```ts
  it('leaves no caches stub behind for the next case (the file-level afterEach unstubs)', () => {
    // vi.stubGlobal('caches', undefined) defines the property; only an unstub
    // removes it. Without the afterEach, every case appended after the
    // "CacheStorage does not exist" case would silently run with no caches.
    expect('caches' in globalThis).toBe(false)
  })
```

- [ ] **Step 2: Run to see it fail**

Run: `npx vitest run tests/unit/authStore.test.ts`
Expected: the trailing case FAILS (`true`), because the earlier `vi.stubGlobal('caches', undefined)` is never reverted. (If it passes before your change, happy-dom on your `node_modules` defines `caches` natively — then assert `Object.getOwnPropertyDescriptor(globalThis, 'caches')?.value` is not `undefined`-by-stub via `vi.isMockFunction`-free means: `expect(globalThis.caches).not.toBeUndefined()`; note the substitution in the execution summary.)

- [ ] **Step 3: Add the unstub**

Inside the `describe`, after `beforeEach`:

```ts
  afterEach(() => vi.unstubAllGlobals())
```

- [ ] **Step 4: Run the file**

Run: `npx vitest run tests/unit/authStore.test.ts`
Expected: 9 passed (7 existing + Task 2's `signIn` case + this one). Remove the `afterEach` line, rerun → the trailing case fails; restore.

- [ ] **Step 5: Commit**

```bash
git add tests/unit/authStore.test.ts
git commit -m "tests: authStore unstubs globals between cases"
```

### Task 4: `/revive` renders "bỏ học N ngày" with its space

**Files:**
- Modify: `frontend/tests/unit/revivePage.test.ts`
- Modify: `frontend/pages/revive.vue`

- [ ] **Step 1: Write the failing test** (append; `WILTED.last_practiced_at` is `2026-09-20T13:00:00Z`)

```ts
  it('renders the missed-days sentence with its space, and cleanly with no last practice', async () => {
    vi.setSystemTime(new Date('2026-09-23T13:00:00Z')) // 3 whole days after last_practiced_at; Date only, timers untouched
    try {
      routeGet(() => Promise.resolve(WILTED))
      const w = mountPage()
      await flushPromises()
      expect(w.text()).toContain('Bạn đã bỏ học 3 ngày liên tiếp. Hãy hoàn thành')
      expect(w.text()).not.toContain('bỏ học3')
    } finally {
      vi.useRealTimers()
    }

    setActivePinia(createPinia())
    routeGet(() => Promise.resolve({ ...WILTED, last_practiced_at: null }))
    const w2 = mountPage()
    await flushPromises()
    expect(w2.text()).toContain('Bạn đã bỏ học. Hãy hoàn thành')
    expect(w2.text()).not.toContain('bỏ học  ')
  })
```

- [ ] **Step 2: Run to see it fail**

Run: `npx vitest run tests/unit/revivePage.test.ts -t 'missed-days'`
Expected: FAIL — `bỏ học3 ngày`.

- [ ] **Step 3: Implement in `revive.vue`**

Script — after `missedDays`:

```ts
// One string, computed here: a leading space inside <template v-if> is a
// text node Vue's whitespace condensing strips, which rendered "bỏ học2 ngày".
const missedLine = computed(() =>
  missedDays.value === null ? 'Bạn đã bỏ học.' : `Bạn đã bỏ học ${missedDays.value} ngày liên tiếp.`)
```

Template — replace line 83's paragraph body with:

```vue
        <p class="font-display text-lg">
          "{{ missedLine }} Hãy hoàn thành Bài kiểm tra Cứu Cây 15 phút để hồi sinh!"
        </p>
```

- [ ] **Step 4: Run the file**

Run: `npx vitest run tests/unit/revivePage.test.ts`
Expected: 4 passed.

- [ ] **Step 5: Commit**

```bash
git add pages/revive.vue tests/unit/revivePage.test.ts
git commit -m "revive: the missed-days sentence keeps its space"
```

### Task 5: `/revive` — stale status wins over a later error, pinned

**Files:**
- Modify: `frontend/tests/unit/revivePage.test.ts` (import `usePetStore` from `~/stores/pet`)

- [ ] **Step 1: Write the tests** (append)

```ts
  it('keeps the last-known wilted state when a later reload fails — stale data wins over the error card', async () => {
    let fails = false
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve(WILTED)))
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)

    fails = true
    await usePetStore().load()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
    expect(w.text()).toContain('Cứu cây ngay')
  })

  it('a failed POST /pet/revive keeps the wilted screen and shows only the inline message', async () => {
    routeGet(() => Promise.resolve(WILTED))
    api.post.mockRejectedValue(new Error('offline'))
    const w = mountPage()
    await flushPromises()

    const start = w.findAll('button').find(b => b.text().includes('Cứu cây ngay'))
    if (!start) throw new Error('no revive button')
    await start.trigger('click')
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
    const alerts = w.findAll('[role="alert"]').map(a => a.text())
    expect(alerts.some(t => t.includes('héo rũ'))).toBe(true)
    expect(alerts.some(t => t.includes('Không bắt đầu được thử thách'))).toBe(true)
  })
```

- [ ] **Step 2: Run, then prove the ordering guard has teeth**

Run: `npx vitest run tests/unit/revivePage.test.ts` → 6 passed.
Mutation: in `revive.vue`, move the `<template v-else-if="pet.error">` block above the `<template v-else-if="pet.status">` block; rerun → the stale-data case FAILS (`[role="status"]` present, no wilted stage). Restore; rerun → 6 passed.

- [ ] **Step 3: Commit**

```bash
git add tests/unit/revivePage.test.ts
git commit -m "tests: /revive keeps stale status over a later error — branch order pinned"
```

### Task 6: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`**shell**` bullet)

- [ ] **Step 1: Edit** — in the `/onboarding` clause, after "both 200 and 201 are success": "; the start button is gated on a goal **and** a valid `HH:MM` reminder time (a cleared field posted `":00"` → 400 before 2026-09-24), and `rate_limited` / `ai_*` / `invalid_request` each get their own copy, all pinned by `tests/unit/onboardingPage.test.ts`". In the service-worker clause, after "callers never clear it themselves": " — and `useAuthStore.signIn()` clears it too before the new session is used (`/login` awaits it), while the route guard drops a present-but-expired session on `/login` itself (fix 2026-09-24), so no account inherits another's cache through a bookmark to `/login`".

- [ ] **Step 2: Commit**

```bash
git add ../harness/CODEMAP.md
git commit -m "harness: CODEMAP records the onboarding gate and the sign-in cache clear"
```

---

## Verification

```bash
cd frontend
npm run lint && npm run typecheck
# expect: clean
npm run test:unit
# expect: all files pass; onboardingPage 8, authStore 9, authMiddleware 4, revivePage 6
npx vitest run tests/unit/onboardingPage.test.ts tests/unit/authStore.test.ts tests/unit/authMiddleware.test.ts tests/unit/revivePage.test.ts
# expect: 4 files, 27 tests passed
npm run build
# expect: succeeds (CI's frontend job runs exactly lint → typecheck → test:unit → build)
grep -n 'canStart\|invalid_request' pages/onboarding.vue
# expect: the gate on the start button and the invalid_request copy
grep -n 'return clearApiCache()' stores/auth.ts
# expect: 2 hits — signIn and signOut
grep -n 'await auth.signIn' pages/login.vue
# expect: 1
grep -n 'unstubAllGlobals' tests/unit/authStore.test.ts
# expect: 1
grep -n 'bỏ học<template' pages/revive.vue
# expect: no output
git log --oneline origin/main..HEAD | wc -l
# expect: 6 commits, one per task, each with the Co-Authored-By trailer
python3 ../tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
```

Mutation checks (each must turn the named test red, then restore):

| Mutation | Test that fails |
| --- | --- |
| `onboarding.vue`: `:disabled="!goal"` (drop the time gate) | `does not start the quiz while the reminder time is cleared` |
| `onboarding.vue`: delete the `ai_` branch | the three `it.each` `ai_*` cases |
| `onboarding.vue`: delete the `invalid_request` branch | `names a 400 invalid_request` |
| `stores/auth.ts`: `return Promise.resolve()` instead of `clearApiCache()` at the end of `signIn` | `signIn drops the previous account's api-state cache` |
| `auth.global.ts`: remove the new `signOut()` line | `/login is the first route` |
| `authStore.test.ts`: remove `afterEach` | `leaves no caches stub behind` |
| `revive.vue`: restore `bỏ học<template v-if…> {{ missedDays }}` | `renders the missed-days sentence with its space` |
| `revive.vue`: swap the `pet.error` and `pet.status` branches | `keeps the last-known wilted state` |

Optional browser proof (local only; needs `npx playwright install chromium` once): `npm run build && npm run test:e2e` still green — no e2e spec changes here.

## Notes and open questions

- **Why not also validate `time` in `next()`?** The field exists only on the goal step and cannot change once the quiz starts; a second check would guard a state the UI cannot reach. If a later design moves the time picker into the quiz, add the check there.
- **`invalid_request` copy** names the goal and the reminder time because those are the only two request fields the learner controls; `timezone` comes from `Intl` and `answers` from the quiz UI.
- **`signIn` returning a promise** is a signature change from `void`; every caller was checked (`pages/login.vue` only; tests call it without awaiting and still read state synchronously, which the ordering preserves).
- **Playwright is untouched.** Service workers are blocked in e2e, so the cache behaviour is proven at the store level with the Map-backed `FakeCacheStorage`, exactly as the 2026-09-23 sign-out plan did.
- **Out of scope:** the `api-state` expiry bound (`the-api-state-cache-still-has-no-expiry-bound-…`, selected low) needs a browser-level proof and its own plan.

## Execution summary

Executed in `.worktrees/a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa` on branch `harness/2026-09-24-medium-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa`, based on freshly fetched `origin/main`. All 6 tasks (onboarding gate + `invalid_request` + pinned `ai_*` copy; sign-in cache clear + `/login` expired-session drop; authStore global unstubbing; revive missed-days spacing; revive branch-order pin; CODEMAP) implemented exactly as written — no deviations from the plan's design decisions, file list, or copy text.

For each task: wrote the failing tests first, ran them to see the expected failure (matched the plan's stated expectations, e.g. "`Không tạo được lộ trình. Thử lại.` to contain `giờ nhắc học`"), implemented, reran to green, ran the specified mutation to confirm the guard has teeth, restored, reran green, then `npm run lint && npm run typecheck`, then committed with the plan's exact message.

One minor deviation: in Task 6's CODEMAP edit, rather than literally inserting the new onboarding sentence before the pre-existing "`rate_limited` / `ai_*` get their own copy" clause (which would have left it redundant with the new, more complete sentence covering the same three error codes), I replaced that old clause with the new one so the paragraph reads once, not twice, over the same fact. All information the plan wanted recorded (gate, `invalid_request`, pinning) is present.

### Plan Verification (all commands run from `frontend/` in the worktree)

```
npm run lint && npm run typecheck
# clean, no errors

npm run test:unit
# Test Files  16 passed (16)
# Tests  78 passed (78)
# onboardingPage.test.ts: 8 passed
# authStore.test.ts: 9 passed
# authMiddleware.test.ts: 4 passed
# revivePage.test.ts: 6 passed

npx vitest run tests/unit/onboardingPage.test.ts tests/unit/authStore.test.ts tests/unit/authMiddleware.test.ts tests/unit/revivePage.test.ts
# 4 files, 27 tests passed

npm run build
# ✨ Build complete! (client + server + service worker all built)

grep -n 'canStart\|invalid_request' pages/onboarding.vue        # gate + copy present, 4 hits
grep -n 'return clearApiCache()' stores/auth.ts                  # 2 hits (signIn, signOut)
grep -n 'await auth.signIn' pages/login.vue                      # 1 hit
grep -n 'unstubAllGlobals' tests/unit/authStore.test.ts          # 1 hit
grep -n 'bỏ học<template' pages/revive.vue                       # no output (fixed)
git log --oneline origin/main..HEAD | wc -l                      # 6
python3 tools/harness/cli.py validate; echo "exit=$?"            # exit=0 (run from worktree root, not frontend/)
```

All mutation checks in the plan's table were run and confirmed (each named test turned red, then was restored to green): the `!goal` gate regression, the `ai_*` branch deletion, the `invalid_request` branch deletion, `signIn` returning `Promise.resolve()` instead of clearing the cache, removing the guard's new `signOut()` line, removing `authStore.test.ts`'s `afterEach`, restoring the old `bỏ học<template>` markup, and swapping the `pet.error`/`pet.status` template branches (this last mutation actually failed *two* tests, not just the one the plan named — a stronger confirmation than specified).

### Runtime proof

- `npm run build` succeeded (client, Nitro server, and the `injectManifest` service worker all built; 50-entry precache manifest generated).
- Booted the built app: `PORT=13001 NUXT_PUBLIC_API_BASE=http://localhost:18085 NUXT_PUBLIC_GOOGLE_CLIENT_ID=test-client NUXT_PUBLIC_VAPID_PUBLIC_KEY=test-vapid node .output/server/index.mjs` (no backend needed for this proof — the app is `ssr:false`, so real content is client-rendered and store-driven).
- `curl` confirmed `/`, `/onboarding`, `/login`, `/revive` all return `200` and serve the real `<title>Học 30 phút</title>` SPA shell.
- Drove it with the in-app browser (Claude_Browser): `/login` rendered "Đăng nhập bằng Google"; with a fake session seeded into `localStorage['aelp.auth']`, `/onboarding` rendered the real goal step (goal cards, time input, start button). Clicked "IELTS 7.0", cleared the `<input type="time">` via `form_input`, and confirmed live in the DOM (`document.querySelectorAll('button')`) that the start button's `.disabled` property became `true` with the hint "Chọn một giờ nhắc học để tiếp tục." showing — the exact defect from the idea, now gated, proven against the actual built artifact rather than only the test harness. `/revive` correctly rendered the error state ("Không tải được trạng thái cây…") since no backend was running — the expected, non-crashing behavior.
- Cleaned up: killed the node server (verified via `pgrep`, nothing left), closed the browser tab. No Docker containers were started for this proof (not needed since the runtime check only exercised the frontend's own error/loading states, not a real backend response) — `docker ps` confirmed nothing running.

### CI

Pushed `harness/2026-09-24-medium-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa`. CI run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35959491527 — all 4 jobs green (`frontend`, `backend-unit`, `harness-tooling`, `backend-integration`).
