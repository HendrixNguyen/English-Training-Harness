---
idea: harness/ideas/2026-09-25-run-01/task-timer-keeps-counting-through-reloads-and-background-tab.md
status: approved
priority: high
merged: false
---
# Task timer keeps counting through reloads and background tabs so studied minutes are never lost — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/task-timer-keeps-counting-through-reloads-and-background-tab.md`

**Goal:** The `/learn/:id` countdown is anchored to the wall-clock moment the learner first opened the task and survives reloads, tab closes and PWA eviction within the same day, so the minutes a learner really spent are what `POST /quests/progress` reports — never fewer.

**Why now (`priority: high`):** Confirmed on this branch: `frontend/stores/quest.ts` keeps `Timer { totalSeconds, remainingSeconds }` in memory and `tick()` subtracts one second per `setInterval` firing from `pages/learn/[id].vue:28`; `elapsedSeconds` is `totalSeconds - remainingSeconds`. A reload recreates the Pinia store with `timers: {}` (the "re-entry resumes" comment at `[id].vue:35` holds only within one page lifetime), and a hidden tab's interval is throttled (Chrome: once a minute after 5 min hidden; Firefox: clamped), so ten real minutes are recorded as four or five. `duration_seconds` is the only thing that credits the 30-minute day (backend spec §6.2), and a short day costs the plant −30 at midnight (backend spec §8). This is user-facing data loss on the happy path, on a phone in particular (iOS suspends an installed PWA on app switch). Half a day, frontend only. Feature slot 2 of 5 (two-cap rule, owner 2026-09-25); high feature → auto-approved.

**Root cause:** the countdown is a tick counter, not a clock, and it has no persistence — the design (`harness/designs/frontend-shell.md` §2.4) promised "a re-entry resumes" but the store never wrote the anchor down. `stores/pet.ts` already solved the same problem for the revive challenge (`localStorage['aelp.revive']`, `storageOrNull()`, `hydrateChallenge()` with `try/catch`); the timer copies that pattern.

**Design decisions (taken by the evaluator — do not re-litigate):**
1. **A timer is an anchor, not a counter.** `Timer` becomes `{ startedAt: number /* epoch ms */, totalSeconds: number, date: string /* daily.date it belongs to */ }`. `elapsedSeconds(taskId, now?)` and `remainingSeconds(taskId, now?)` are computed from `now`; nothing is decremented.
2. **`tick` only forces re-render.** The store keeps a reactive `nowMs`; `tick(now = Date.now())` sets it, and the two computations default `now` to `this.nowMs`, so a template `computed` that calls `quest.remainingSeconds(id)` re-runs once a second — and immediately on `tick()` after `visibilitychange`/`focus`. Tests inject `now` explicitly or drive `Date.now()` with `vi.useFakeTimers()`; both paths exercise the same arithmetic. When the page posts, it passes `Date.now()` explicitly so the reported duration is exact, not up to one second stale.
3. **Persist at `localStorage['aelp.timers']`** — a JSON map `{ [taskId]: Timer }`, written on `startTimer` and rewritten with the entry removed on a successful `complete()`. The store hydrates the raw map in its `state()` initializer (store creation) and **prunes** once it knows the day: after `load()` sets `daily`, every entry whose `date !== daily.date`, whose task is not in `daily.tasks`, or whose task is `is_completed` is dropped and the map rewritten. Every storage access goes through `storageOrNull()` (`typeof localStorage === 'undefined'` guard) inside `try/catch`, exactly like `stores/pet.ts`; a missing or throwing `localStorage` leaves the store working in memory.
4. **The wire is untouched.** The page still posts `quest.elapsedSeconds(id)` and `complete()` still applies `clampDuration` (1..3600); the button gate uses `remainingSeconds === 0` (elapsed ≥ total) while the posted `duration_seconds` is the real elapsed clamped to 3600. Backend spec §6.2 and `quests.MaxDurationSeconds` are unchanged. **No backend change.**
5. **`startTimer` needs a day.** It reads `this.daily.date` for the anchor's `date`; the page only calls it once `task` resolves, which implies `daily` is loaded. Without `daily` it is a no-op (documented in the code).
6. **`learn/[id].vue` re-syncs on `visibilitychange` and `focus`** by calling `quest.tick()`; the interval stays at one second for the visible case. `CountdownTimer.vue` is unchanged (it still takes `remaining-seconds`).

**UI note (no design doc — the only UI change is when an existing label appears):** the primary button on `/learn/:id` has two timed states, unchanged in copy from `harness/designs/frontend-shell.md` §2.4: **"Hoàn thành"** — disabled while wall-clock elapsed < `duration_minutes` and the content is not `finished`, enabled once `finished`; **"Hết giờ — Hoàn thành"** — enabled, shown as soon as wall-clock elapsed ≥ `duration_minutes`, including the instant a tab or PWA is resumed after being away that long (the countdown reads `00:00` in `text-alert`). "Đã hoàn thành" for a completed task is unchanged.

**Tech stack:** Nuxt 3 + Pinia 3 + Vitest 3 (`happy-dom`, no Nuxt runtime in unit tests — stores are imported directly and `~/composables/useApi` is `vi.mock`ed). No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n`. Frontend checks are `cd frontend && npm run lint && npm run typecheck && npm run test:unit` (run `npm ci` once in the worktree first).

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/stores/quest.ts` | `Timer` → wall-clock anchor with `date`; `TIMER_STORAGE_KEY`; `storageOrNull`/`readTimers`/`writeTimers` helpers; state `timers` hydrated + `nowMs`; `startTimer`, `tick`, `elapsedSeconds`, new `remainingSeconds`, new `pruneTimers` (called from `load()`); `complete()` rewrites storage after removing the entry |
| `frontend/pages/learn/[id].vue` | `remaining` computed from the store; `tick()` on interval, `visibilitychange`, `focus`; posts `elapsedSeconds(id, Date.now())`; button label/gate read `remaining` |
| `frontend/tests/unit/questStore.test.ts` | the tick-based timer case replaced by six wall-clock/persistence cases; `complete` case asserts the storage entry is gone |
| `frontend/tests/unit/learnPage.test.ts` | new: the button flips to "Hết giờ — Hoàn thành" on `visibilitychange` with no interval tick |
| `harness/CODEMAP.md` | `shell` paragraph: the `stores/quest.ts` clause names the wall-clock anchor and `localStorage['aelp.timers']` |

Not touched: `frontend/components/learn/CountdownTimer.vue`, `frontend/utils/progress.ts`, `frontend/stores/pet.ts`, anything under `backend/`.

---

## Tasks

### Task 1: Store — wall-clock timers persisted at `aelp.timers`

**Files:**
- Modify: `frontend/stores/quest.ts`, `frontend/tests/unit/questStore.test.ts`

- [ ] **Step 1: Write the failing tests** in `questStore.test.ts`. Add `afterEach` to the vitest import, `localStorage.clear()` to `beforeEach`, and `afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })`. Add a helper `const T0 = Date.parse('2026-09-23T13:00:00Z')` and `async function loaded() { api.get.mockResolvedValue(daily); const q = useQuestStore(); await q.load(); return q }`. Replace the case `timers count down from duration_minutes and report elapsed seconds` with these six (import `TIMER_STORAGE_KEY` from `~/stores/quest`):
  - `timers are anchored to the wall clock: elapsed is the real delta however often tick runs` — `vi.useFakeTimers({ toFake: ['Date'] }); vi.setSystemTime(T0)` (fake only `Date` so awaited promises and `flushPromises` are never queued on a faked `setTimeout`/`setImmediate`); `q = await loaded()`; `q.startTimer('ex-2', 10)`; expect `q.timers['ex-2']` `toEqual({ startedAt: T0, totalSeconds: 600, date: '2026-09-23' })` and `q.remainingSeconds('ex-2')` `600`; `vi.setSystemTime(T0 + 18_000)`; call `q.tick()` once → `elapsedSeconds` `18`, `remainingSeconds` `582`; call `q.tick()` twenty more times without moving the clock → still `18`; `vi.setSystemTime(T0 + 700_000)` and **no** `tick()`, then `q.elapsedSeconds('ex-2', T0 + 700_000)` → `700` and `q.remainingSeconds('ex-2', T0 + 700_000)` → `0`; `q.startTimer('ex-2', 10)` again → `startedAt` still `T0` (re-entry resumes, never resets).
  - `a timer survives a reload: a fresh store hydrates aelp.timers and load keeps today's entry` — `q = await loaded()`; `q.startTimer('ex-2', 10, T0)`; expect `JSON.parse(localStorage.getItem(TIMER_STORAGE_KEY) ?? 'null')` `toEqual({ 'ex-2': { startedAt: T0, totalSeconds: 600, date: '2026-09-23' } })`; `setActivePinia(createPinia())` (the simulated reload); `q2 = await loaded()`; `q2.elapsedSeconds('ex-2', T0 + 300_000)` → `300`; `q2.startTimer('ex-2', 10, T0 + 300_000)` → `q2.timers['ex-2'].startedAt` still `T0`.
  - `load drops a timer from another day` — `localStorage.setItem(TIMER_STORAGE_KEY, JSON.stringify({ 'ex-2': { startedAt: T0 - 86_400_000, totalSeconds: 600, date: '2026-09-22' } }))`; `q = await loaded()` → `q.timers['ex-2']` `undefined`, storage item `'{}'`; `q.startTimer('ex-2', 10, T0)` → a fresh anchor at `T0`.
  - `load drops a timer whose task is already completed` — same seed for `'ex-1'` (completed in `daily`) with `date: '2026-09-23'`, plus a valid `'ex-2'` entry → after `loaded()` only `'ex-2'` remains in `q.timers` and in storage.
  - `the countdown floors at 0 for the button gate while the posted duration is the real elapsed clamped to 3600` — `q = await loaded()`; `api.post.mockResolvedValue({...})` as in the existing `complete` case; `q.startTimer('ex-2', 10, T0)`; `now = T0 + 5_000_000` (≈ 83 min); `q.remainingSeconds('ex-2', now)` → `0`; `q.elapsedSeconds('ex-2', now)` → `5000`; `await q.complete('ex-2', q.elapsedSeconds('ex-2', now))` → `api.post` called with `duration_seconds: 3600`.
  - `the store works without localStorage and when it throws` — `vi.stubGlobal('localStorage', undefined)`; `setActivePinia(createPinia())`; `q = await loaded()`; `q.startTimer('ex-2', 10, T0)`; `q.elapsedSeconds('ex-2', T0 + 60_000)` → `60`. Then `vi.stubGlobal('localStorage', { getItem() { throw new Error('denied') }, setItem() { throw new Error('denied') }, removeItem() { throw new Error('denied') } })`; `setActivePinia(createPinia())`; same sequence → `60` again, no throw.
  - In the existing `complete posts the §6.2 body…` case, seed via `q.startTimer('ex-2', 10, T0)` and add `expect(localStorage.getItem(TIMER_STORAGE_KEY)).toBe('{}')` after `expect(q.timers['ex-2']).toBeUndefined()`.
- [ ] **Step 2: Run red:** `cd frontend && npm run test:unit -- questStore` → the six new cases fail (`startTimer` has no third argument, `remainingSeconds` is not a function, `TIMER_STORAGE_KEY` undefined).
- [ ] **Step 3: Make them pass** in `stores/quest.ts`. Replace the `Timer` interface and add the helpers above `useQuestStore`:

```ts
/** A per-task countdown anchored to the wall clock so reloads, background tabs and PWA suspension never lose minutes. */
export interface Timer {
  startedAt: number // epoch ms of the learner's first open of the task
  totalSeconds: number
  date: string // GET /quests/daily `date` the anchor belongs to; other days are discarded on load
}

export const TIMER_STORAGE_KEY = 'aelp.timers'

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

function readTimers(): Record<string, Timer> {
  try {
    const raw = storageOrNull()?.getItem(TIMER_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Record<string, Partial<Timer>>
    const out: Record<string, Timer> = {}
    for (const [id, t] of Object.entries(parsed)) {
      if (typeof t.startedAt === 'number' && typeof t.totalSeconds === 'number' && typeof t.date === 'string') out[id] = { startedAt: t.startedAt, totalSeconds: t.totalSeconds, date: t.date }
    }
    return out
  } catch {
    return {}
  }
}

function writeTimers(timers: Record<string, Timer>) {
  try {
    storageOrNull()?.setItem(TIMER_STORAGE_KEY, JSON.stringify(timers))
  } catch {
    // storage denied or full: the in-memory anchor still works for this page lifetime
  }
}
```

  State: `timers: readTimers()` (replacing `{} as Record<string, Timer>`) and `nowMs: Date.now()`. Actions:

```ts
    /** Idempotent: re-entering a task resumes its wall-clock anchor (design §2.4), across reloads via aelp.timers. Needs `daily` for the anchor's date. */
    startTimer(taskId: string, durationMinutes: number, now = Date.now()) {
      this.nowMs = now
      const date = this.daily?.date
      if (!date || this.timers[taskId]) return
      this.timers[taskId] = { startedAt: now, totalSeconds: Math.max(60, Math.floor(durationMinutes * 60)), date }
      writeTimers(this.timers)
    },
    /** Re-renders every countdown from the clock; the page calls it each second and on visibilitychange/focus. */
    tick(now = Date.now()) {
      this.nowMs = now
    },
    elapsedSeconds(taskId: string, now = this.nowMs): number {
      const t = this.timers[taskId]
      return t ? Math.max(0, Math.floor((now - t.startedAt) / 1000)) : 0
    },
    remainingSeconds(taskId: string, now = this.nowMs): number {
      const t = this.timers[taskId]
      return t ? Math.max(0, t.totalSeconds - this.elapsedSeconds(taskId, now)) : 0
    },
    /** Drops anchors from another day, for tasks not in today's list, or for tasks already completed. */
    pruneTimers() {
      if (!this.daily) return
      const today = this.daily
      for (const [id, t] of Object.entries(this.timers)) {
        const task = today.tasks.find(x => x.id === id)
        if (t.date !== today.date || !task || task.is_completed) Reflect.deleteProperty(this.timers, id)
      }
      writeTimers(this.timers)
    },
```

  In `load()`, right after `this.daily = await useApi().get<DailyQuests>(...)`, call `this.pruneTimers()`. In `complete()`, after the existing `Reflect.deleteProperty(this.timers, exerciseId)` add `writeTimers(this.timers)`. Do not touch `ProgressResponse`, the getters, or any other line of `complete()` (two other plans land on this file today — see Notes).
- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit -- questStore` → clean; `typecheck` now fails in `pages/learn/[id].vue` (`quest.tick(task.value!.id)` no longer matches, `timer.value?.remainingSeconds` is gone) — that is Task 2; if `typecheck` blocks the commit, do Task 2's Step 3 in the same commit and say so in the message.
- [ ] **Step 5: Commit:** `git commit -am "frontend: task timers are wall-clock anchors persisted at aelp.timers"`.

### Task 2: Page — resync on visibility and focus, post the real elapsed

**Files:**
- Modify: `frontend/pages/learn/[id].vue`
- Create: `frontend/tests/unit/learnPage.test.ts`

- [ ] **Step 1: Write the failing test** `learnPage.test.ts`, following `revivePage.test.ts` (mock `~/composables/useApi`; `vi.stubGlobal('useRoute', () => ({ params: { id: 'ex-2' } }))` and `vi.stubGlobal('navigateTo', vi.fn())` **before** `await import('~/pages/learn/[id].vue')`; register `AppButton`, `AppCard`, `StateBlock`, `CountdownTimer`, `ContentViewer` plus whatever `ContentViewer` renders for the fallback shape — `components: { pathPrefix: false }` means bare names; `afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })`; `localStorage.clear()` in `beforeEach`). One `daily` with `ex-2` (`duration_minutes: 10`, `content_json: {}` so the fallback content branch renders, no `finished`). Cases:
  - `the countdown and button follow the wall clock on visibilitychange, not the interval` — `vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)` (`setTimeout`/`setImmediate` stay real so `flushPromises` resolves; the page's interval is faked and never advanced); `api.get.mockResolvedValue(daily)`; mount; `await flushPromises()`; the timer reads `10:00` and the button text is `Hoàn thành` and `disabled`; `vi.setSystemTime(T0 + 600_000)` **without** `vi.advanceTimersByTime` (no interval tick fires); `document.dispatchEvent(new Event('visibilitychange'))`; `await nextTick()` → the timer reads `00:00`, the button text is `Hết giờ — Hoàn thành` and it is enabled. (The `focus` listener is the same one-line handler; assert it once with `window.dispatchEvent(new Event('focus'))` after resetting the clock to `T0 + 599_000` → still `Hoàn thành`, then to `T0 + 600_000` → `Hết giờ — Hoàn thành`.)
  - `completing posts the wall-clock elapsed` — same mount; `vi.setSystemTime(T0 + 615_000)`; `visibilitychange`; `api.post.mockResolvedValue({ daily_seconds_spent: 1215, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })`; click the button; `await flushPromises()` → `api.post` called with `duration_seconds: 615`; `localStorage.getItem('aelp.timers')` is `'{}'`.
- [ ] **Step 2: Run red:** `npm run test:unit -- learnPage` → fails (`quest.tick(task.value!.id)` stores nothing; the label never flips without an interval tick).
- [ ] **Step 3: Make it pass** in `pages/learn/[id].vue`:
  - `const remaining = computed(() => quest.remainingSeconds(id.value))` next to the existing `timer` computed (keep `timer` for the `v-if`).
  - `onMounted`: after `quest.startTimer(...)`, `quest.tick()` once, then `interval = setInterval(() => quest.tick(), 1000)`; add `document.addEventListener('visibilitychange', sync)` and `window.addEventListener('focus', sync)` with `function sync() { quest.tick() }` — comment: "a resumed tab or PWA snaps to the wall clock immediately instead of waiting for the next (throttled) tick". Register both listeners regardless of the timer branch (they are harmless for a completed task) and remove both in `onBeforeUnmount`; change the `clearInterval` comment to "the store keeps the wall-clock anchor (and aelp.timers keeps it across reloads), so re-entry resumes".
  - `buttonLabel`: `if (timer.value && remaining.value === 0) return 'Hết giờ — Hoàn thành'`.
  - `complete()`: `quest.complete(task.value.id, quest.elapsedSeconds(task.value.id, Date.now()), answers.value)` — the exact elapsed at post time.
  - Template: `<CountdownTimer v-if="timer" :remaining-seconds="remaining" />`; the button's `:disabled` becomes `task.is_completed || !online || (!finished && (!timer || remaining !== 0))`.
- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; the whole suite passes (`revivePage`, `onboardingPage` untouched).
- [ ] **Step 5: Commit:** `git commit -am "frontend: learn room resyncs the countdown on visibilitychange/focus and posts the wall-clock elapsed"`.

### Task 3: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** In the `shell` paragraph, change the `stores/quest.ts` clause `(`GET /quests/daily`, per-task countdown timers, `POST /quests/progress` with `clampDuration` 1..3600)` to `(`GET /quests/daily`, per-task countdown timers anchored to the wall clock — `{startedAt, totalSeconds, date}` persisted at `localStorage['aelp.timers']`, pruned on `load()` to today's uncompleted tasks, so a reload, background tab or suspended PWA never loses minutes; `/learn/:id` re-syncs on `visibilitychange`/`focus` — `POST /quests/progress` with `clampDuration` 1..3600)`. No other `harness/` file changes on this branch.
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → exit 0. Commit: `git commit -am "harness: CODEMAP — wall-clock task timers at aelp.timers"`.

---

## Verification

```bash
cd frontend
npm run lint && npm run typecheck && npm run test:unit
# expect: all three clean; questStore.test.ts and learnPage.test.ts pass
grep -c '^  it(' tests/unit/questStore.test.ts
# expect: 10 (was 5: the tick-based case replaced, six added — see Task 1)
grep -c '^  it(' tests/unit/learnPage.test.ts
# expect: 2
grep -n "aelp.timers" stores/quest.ts
# expect: 1 line (TIMER_STORAGE_KEY)
grep -c 'remainingSeconds' stores/quest.ts
# expect: 1 (the action's declaration; was 4 when it was a stored field)
grep -n 'Date.now()' stores/quest.ts
# expect: 3 lines (state nowMs, startTimer default, tick default)
grep -n 'setInterval\|visibilitychange\|focus' "pages/learn/[id].vue"
# expect: 6 lines — 2 setInterval (the `let interval` type and the call, which is `quest.tick()` with no task argument) + 2 visibilitychange + 2 focus (add/remove each); the file has no other `focus` today
grep -n 'elapsedSeconds(task.value.id, Date.now())' "pages/learn/[id].vue"
# expect: 1 line
grep -n "remainingSeconds" components/learn/CountdownTimer.vue | wc -l
# expect: 3 (file unchanged)
cd ..
git diff --stat origin/main...HEAD -- backend/ frontend/stores/pet.ts frontend/utils/progress.ts
# expect: no output (no backend change; pet store and clampDuration untouched)
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
grep -c "aelp.timers" harness/CODEMAP.md
# expect: 1
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: frontend, backend-unit, backend-integration, harness-tooling green
```

Mutation checks (record each result, then revert):
- In `elapsedSeconds`, replace `Math.floor((now - t.startedAt) / 1000)` with `this.nowMs === now ? 0 : Math.floor(...)` → the wall-clock case must go red on the twenty-tick assertion.
- Delete the `this.pruneTimers()` call in `load()` → both "load drops" cases go red.
- Delete the `writeTimers(this.timers)` line in `complete()` → the `complete posts the §6.2 body` storage assertion goes red.
- Remove the `visibilitychange` listener in `[id].vue` → the page test's first case goes red.

## Notes and open questions

- **Same-day overlap on `stores/quest.ts` and the merge order.** Today's approved bug plan `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` changes `ProgressResponse.pet_health?`/`streak_count?` in `stores/quest.ts` and `applyProgress` in `stores/pet.ts`; the growth-moment feature plan (slot 3, same run folder) changes `complete()` to expose `targetMetChanged` and `stores/pet.ts` again. This plan's edits to `stores/quest.ts` are confined to: the `Timer` interface and the new helpers above `useQuestStore`; the `timers`/`nowMs` state lines; `startTimer`/`tick`/`elapsedSeconds`/`remainingSeconds`/`pruneTimers`; one `this.pruneTimers()` call in `load()`; and one `writeTimers(this.timers)` line after `Reflect.deleteProperty(this.timers, exerciseId)` in `complete()`. It does not touch `ProgressResponse`, the getters, `stores/pet.ts` or `complete()`'s response handling. **The 14:00 feature run executes this plan before the growth-moment plan** (the run summary already says ideas 1 and 2 must not run in parallel worktrees), and the daily integration merge should then be trivial.
- **Why prune in `load()` rather than in `state()`.** A store is created before `GET /quests/daily` answers, so "today" is unknown at creation; hydrating raw and pruning once `daily` exists is the only order that works and keeps the guard testable (`pruneTimers` is an action). Between creation and `load()` an old-day entry can exist in memory but is never rendered: `/learn/:id` only starts or reads a timer after `task` resolves, which needs `daily`.
- **Why the `date` field when `load()` already checks task ids.** Task ids are per-exercise and stable across days, so an anchor from yesterday for a task not yet regenerated today would otherwise be resumed; the date check is the cheap, explicit guard the idea asked for.
- **Clock skew and a learner who moves the device clock.** `startedAt` is device time; a backwards jump reads as elapsed `0` (the `Math.max(0, …)`), a forward jump as more elapsed — bounded on the wire by `clampDuration` to 3600 and on the server by `MaxDurationSeconds`. Same exposure as any client-reported duration in backend spec §6.2; not worse than today.
- **Multiple tabs.** Two tabs on the same task share one anchor via `aelp.timers`; the second tab's `startTimer` is a no-op on an existing key. Each tab's `complete()` clears the entry; a second post is rejected or double-counts exactly as it does today (not in scope).
- **No `storage` event listener.** Cross-tab live sync is unnecessary for a countdown that is recomputed from the anchor on every tick and on focus.
- **Backend spec §6.2, frontend spec §7.3 and `harness/designs/frontend-shell.md` §2.4** remain accurate; §2.4's "a re-entry resumes" is now true across reloads, which CODEMAP records.
