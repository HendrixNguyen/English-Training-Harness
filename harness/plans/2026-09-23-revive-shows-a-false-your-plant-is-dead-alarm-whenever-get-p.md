---
idea: harness/ideas/_inbox/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md
status: approved
priority: high
merged: false
---
# /revive: render an error state when GET /pet/status fails, never a false wilted plant — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md`
**Goal:** `/revive` shows a retryable error block when the pet status could not be loaded, and enters the wilted UI (red banner, 0 % wilted plant, "you skipped N days", revive CTA) **only** when real data says the plant is wilted — exactly how `/` and `/roadmap` already treat the same failure.

**Why now (`priority: high`):** the plant is the retention mechanism (spec §1, §5.2), and "your plant has withered" to a user whose plant is healthy is the single most damaging false message the app can show. It appears whenever `GET /pet/status` fails with no cached copy — the offline path the PWA is sold on (Frontend spec §3). The other three data-driven routes handle this correctly; `/revive` is the outlier. One template edit plus a regression test.

**Root cause (confirmed on `main`, `frontend/pages/revive.vue`):** the template has four branches — `pet.loading && !pet.status` → skeleton; `passed` → revived; `pet.notWilted || (pet.status && !pet.isWilted)` → healthy; `v-else` → wilted. The `v-else` is reached both when the plant is really wilted **and** when `pet.load()` failed (`stores/pet.ts` sets `error`, leaves `status` null, clears `loading`). `pet.error` is never read by the page (`grep -n "pet.error" pages/revive.vue` → none; the page's own `error` ref is the `revive()` action's message).

**Architecture:** reorder the branches so every one is gated on a positive fact and the wilted branch requires `pet.status` to be non-null. Final order: `passed` → healthy (`pet.notWilted || (pet.status && !pet.isWilted)`) → wilted (`pet.status`, which at this point means `isWilted`) → error (`pet.error`, status null) → `v-else` skeleton (status null, no error: the tick before `onMounted` fires and the load itself). The skeleton no longer needs `pet.loading` — "no status and no error" *is* loading. No store change: `stores/pet.ts` already exposes `error`/`status` correctly. The error copy tells the user the state is *unknown*, not dead.

**Test approach:** a Vitest component test that mounts the page with the same setup the store tests use (`setActivePinia(createPinia())`, `vi.mock('~/composables/useApi')`, `happy-dom`) and registers the page's auto-imported components explicitly (`AppCard`, `AppButton`, `StateBlock`, `PlantSvg`, `SegmentedProgress` — none renders `NuxtLink`; `navigateTo` is provided as a mock). Three cases: load fails → error block, no alert, no wilted plant; retry → real wilted data → wilted UI; real wilted data first time → wilted UI. Playwright is local-only (CODEMAP), so the browser proof is the reviewer's manual reproduction, run once and pasted into the execution summary.

**Tech stack:** Nuxt 3 / Vue 3 / Pinia / Tailwind as on `main`; Vitest + `@vue/test-utils` + `happy-dom` (already dev dependencies). No new dependencies.

**Run every command from `frontend/` inside the worktree.** `npm ci` first. `rg` is not installed — use `grep -n`.

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/tests/unit/revivePage.test.ts` | **New**: three component tests |
| `frontend/pages/revive.vue` | Template only: reorder branches, add the error branch, gate wilted on `pet.status` |
| `harness/CODEMAP.md` | `shell` bullet: one clause on the page-state convention |

---

## Tasks

### Task 1: Pin the behaviour with a component test that fails today

**Files:**
- Create: `frontend/tests/unit/revivePage.test.ts`

- [ ] **Step 1: Write the test**

```ts
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlantSvg from '~/components/plant/PlantSvg.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'
import StateBlock from '~/components/ui/StateBlock.vue'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { default: RevivePage } = await import('~/pages/revive.vue')

const DAILY = { date: '2026-09-23', day_number: 3, total_minutes_required: 30, accumulated_seconds: 600, is_target_met: false, tasks: [] }
const WILTED = { plant_name: 'My Green Buddy', health_points: 0, stage: 'wilted', current_streak: 0, last_practiced_at: '2026-09-20T13:00:00Z' }

/** Routes GET by path so /quests/daily can succeed while /pet/status fails. */
function routeGet(pet: () => Promise<unknown>) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? pet() : Promise.resolve(DAILY)))
}

function mountPage() {
  return mount(RevivePage, {
    global: {
      components: { AppButton, AppCard, PlantSvg, SegmentedProgress, StateBlock },
      mocks: { navigateTo: vi.fn() },
    },
  })
}

describe('/revive (wireframe 7.5) when GET /pet/status fails', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
  })

  it('renders an error state with retry — never the wilted alarm', async () => {
    routeGet(() => Promise.reject(new Error('offline')))
    const w = mountPage()
    await flushPromises()

    expect(w.text()).not.toContain('héo rũ')
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('[data-stage="wilted"]').exists()).toBe(false)
    expect(w.text()).toContain('Thử lại')
    expect(w.find('[role="status"]').text()).toContain('Không tải được')
  })

  it('retry reloads the status and then shows the real wilted state', async () => {
    let fails = true
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve(WILTED)))
    const w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Thử lại')

    fails = false
    await w.find('[role="status"] button').trigger('click')
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.text()).toContain('Cứu cây ngay')
  })

  it('shows the wilted UI on real wilted data', async () => {
    routeGet(() => Promise.resolve(WILTED))
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.find('[role="status"]').exists()).toBe(false)
  })
})
```

- [ ] **Step 2: Run it and confirm the first two cases fail for the right reason**

Run: `npx vitest run tests/unit/revivePage.test.ts`
Expected: case 1 fails on `expect(w.text()).not.toContain('héo rũ')` (the false alarm), case 2 fails because there is no `[role="status"] button` to click, case 3 passes. If case 1 passes, stop — you are not on the code this plan was written against. If mounting fails on an unresolved component or import, register/mock that one thing in `mountPage()` and record it as a deviation; do not stub the whole page.

- [ ] **Step 3: Commit the failing test**

```bash
git add tests/unit/revivePage.test.ts
git commit -m "revive: component test — a failed pet load must not render the wilted alarm"
```

---

### Task 2: Reorder the template and add the error branch

**Files:**
- Modify: `frontend/pages/revive.vue` (template block only; the script is unchanged)

- [ ] **Step 1: Replace the `<main>` block's branch structure**

Keep every existing card's inner markup byte-for-byte; only the `<template v-if/v-else-if>` wrappers change. The new skeleton of `<main>`:

```vue
<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <template v-if="passed">
      <!-- unchanged: "Cây đã hồi sinh!" card -->
    </template>

    <template v-else-if="pet.notWilted || (pet.status && !pet.isWilted)">
      <!-- unchanged: "Cây của bạn vẫn khỏe 🌱" card -->
    </template>

    <!-- Real data, and it says wilted: the only way into the alarm. -->
    <template v-else-if="pet.status">
      <!-- unchanged: red role="alert" banner, PlantSvg stage="wilted", the two challenge cards, the revive() error line -->
    </template>

    <!-- The load failed and nothing is cached: say the state is unknown, offer a retry
         (same shape as / and /roadmap). Never guess "dead". -->
    <template v-else-if="pet.error">
      <AppCard class="mt-4">
        <StateBlock state="error" message="Không tải được trạng thái cây. Chưa thể biết cây có héo hay không." action="Thử lại" @action="pet.load()" />
      </AppCard>
    </template>

    <!-- No status, no error: the load is in flight (or onMounted has not run yet). -->
    <template v-else>
      <AppCard class="mt-4">
        <StateBlock state="loading" />
      </AppCard>
    </template>
  </main>
</template>
```

Rules while editing: the wilted block's inner markup (banner, `PlantSvg stage="wilted" :health="0"`, `challengeActive` cards, the `error` paragraph) moves verbatim under `v-else-if="pet.status"`; the old first branch (`pet.loading && !pet.status` skeleton) becomes the final `v-else`. Do not touch `<script setup>`.

- [ ] **Step 2: Run the new test, then the whole unit suite**

Run: `npx vitest run tests/unit/revivePage.test.ts` → 3 passed.
Run: `npm run test:unit` → all passed (the `petStore` tests are unaffected: no store change).

- [ ] **Step 3: Lint, typecheck, build**

Run: `npm run lint && npm run typecheck && npm run build`
Expected: exit 0 each. Fix any lint complaint in the new test file to match the repo style (no semicolons, single quotes, as in `tests/unit/petStore.test.ts`).

- [ ] **Step 4: Commit**

```bash
git add pages/revive.vue
git commit -m "revive: error state with retry when GET /pet/status fails; wilted UI only on real data"
```

---

### Task 3: Browser proof and CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md` (`shell` bullet)

- [ ] **Step 1: Reproduce the reviewer's scenario against the fix**

From `frontend/` after `npm run build` (nothing listens on 3198, so every API call fails):
```bash
PORT=3101 HOST=127.0.0.1 NUXT_PUBLIC_API_BASE=http://127.0.0.1:3198 node .output/server/index.mjs &
```
In a browser at `http://127.0.0.1:3101/login`, seed a session in the console:
`localStorage.setItem('aelp.auth', JSON.stringify({accessToken:'t', expiresAt: Date.now()+86400000, user:{id:'u1',email:'a@b.c',full_name:'Review User',cefr_current:'B1'}}))`
then navigate to `/revive`. Expected: the error card ("Không tải được trạng thái cây…", "Thử lại" button); **no** red "héo rũ" banner, no wilted plant, no "Cứu cây ngay" button. Record what you saw (an accessibility snapshot or the visible text) in the execution summary, then stop the node process. If a browser is not available, use `npx playwright test --project=chromium` with an ad-hoc spec that `page.route`s `**/api/v1/pet/status` to `route.abort()` and asserts the same — and note which method you used.

- [ ] **Step 2: CODEMAP**

In `harness/CODEMAP.md` → `**shell**`, after the sentence listing the pages, add one clause:

> Page state convention: every data-driven page (`/`, `/roadmap`, `/revive`) renders `StateBlock state="error"` with a retry when its store has `error` and no `status`/`daily`, and enters a data-dependent branch (wilted, no-roadmap, …) only on non-null data — `/revive`'s wilted alarm is gated on `pet.status` (fix 2026-09-23).

- [ ] **Step 3: Commit**

```bash
git add ../harness/CODEMAP.md
git commit -m "CODEMAP: shell page-state convention"
```

---

## Verification

From `frontend/` in the worktree:

1. `npm ci && npm run lint && npm run typecheck` → exit 0.
2. `npm run test:unit` → all tests pass, including the three in `tests/unit/revivePage.test.ts`.
3. **Mutation check (paste the output):** temporarily revert the wilted branch's condition to a bare `v-else` (i.e. remove the error branch) and run `npx vitest run tests/unit/revivePage.test.ts` → cases 1 and 2 must fail; restore, run again → 3 passed. `git diff --quiet pages/revive.vue` clean against your commit.
4. `npm run build` → exit 0.
5. The browser reproduction in Task 3 step 1, with the observed text recorded.
6. Push; `gh run watch` — the `frontend` job (and the three backend jobs) green on the branch.

## Notes and open questions

- **Offline with a cached status** is already right: the service worker's `NetworkFirst` serves the last `/pet/status` body, `pet.status` is non-null, and the page shows the last-known state. This plan only fixes the *no cache* case. The related privacy finding about that cache surviving an expired-session sign-out is `expired-session-sign-out-leaves-per-user-api-responses-in-th.md` (selected, high) — separate branch.
- **Why not change the store:** `stores/pet.ts` already models the three states (`loading`, `error`, `status`) correctly; the bug is purely in which template branch reads them. Keeping the store untouched keeps `petStore.test.ts` and the `/` page out of the diff.
- **`daysSince(null)`** returns `null`, so the "bỏ học N ngày" clause was already guarded; unchanged.
- No design doc: the error card reuses `StateBlock` exactly as `/` does (design `harness/designs/frontend-shell.md`, states section).
