---
idea: harness/ideas/2026-09-25-run-01/growth-moment-after-every-task-health-gain-streak-and-target.md
status: approved
priority: medium
merged: false
design: harness/designs/growth-moment.md
---
# Growth moment after every task: health gain, streak and target-met celebration on the hub — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/2026-09-25-run-01/growth-moment-after-every-task-health-gain-streak-and-target.md`
**Design:** `harness/designs/growth-moment.md` — read it first; §2 is the timeline, §3 the firing table, §4 the bubble copy, §6 the motion values. This plan does not repeat them.

**Goal:** After `POST /quests/progress` succeeds and the learner lands on `/`, the hub shows what the task did to the plant — the health bar slides from the old value, a chip says "+20 máu" / "🔥 7 ngày", the plant cross-fades to its new stage or takes one grow breath, the bubble speaks a context-aware line — in ≤ 2 s, once, with every keyframe behind `prefers-reduced-motion: no-preference`; nothing fires on a plain load or when nothing changed.

**Why now (`priority: medium`):** 1st-thinking §5.2 step 5 "Render Growth Animation & Status" is the one step of the daily loop with no implementation; the reward that closes the habit loop is silent. Confirmed today on `origin/main`: `pages/learn/[id].vue` `complete()` → `pet.applyProgress(res)` → `navigateTo('/')`; `stores/pet.ts` `applyProgress` overwrites `health_points`/`current_streak` and keeps no before-value; `pages/index.vue` re-runs `pet.load()` on mount so the store holds the *after* numbers before the hub paints; `speechLine` keys on `{stage, health, targetMet}` only; `PlantSvg` swaps stage groups with no transition. Frontend only; wire shapes untouched.

**Arithmetic this reflects (CODEMAP `pet`):** +20 (capped at 100), streak+1 and the stage recompute happen **once per local day, on the task that crosses 30 minutes** (`Service.OnTargetMet` → `Repo.SaveTargetMet`, `backend/internal/pet/repo.go:94-96`). Tasks before that return unchanged `pet_health`/`streak_count`. Stage by streak (`engine.go:51-62`): wilted at 0 health, else 0–2 `sprout`, 3–6 `sapling`, 7–13 `flowering`, 14+ `fruitful`.

**Design decisions (taken; do not re-litigate):**
1. **The store carries the before-values.** `usePetStore.applyProgress` records `lastDelta: GrowthDelta { healthFrom, healthTo, streakFrom, streakTo, stageFrom, stageTo, targetMetNow }` only when something changed (health, streak, stage) or the target was newly met; `consumeDelta()` returns it once and clears it. Pinia state → survives `navigateTo('/')`, not a reload.
2. **`applyProgress` already contains today's bug-plan guard** (`harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md`, lands first): a field that is not a number is neither assigned nor counted in the delta. Write it exactly as in Task 3 so the merge is a superset.
3. **Stage after is derived client-side** by a new pure `stageForStreak(streak, health)` from the table above, written into `status.stage` when both numbers are present; `pet.load()` on the hub confirms it from `GET /pet/status` moments later. No wire change.
4. **`useQuestStore.complete()` returns `CompleteResult = ProgressResponse & { targetMetChanged: boolean }`** (response `is_target_met` vs the previous `daily.is_target_met`). `learn/[id].vue` stays `pet.applyProgress(res)` — the extra field rides along; `applyProgress` reads it as an optional `targetMetChanged`.
5. **`speechLine` keeps returning one string** and gains `accumulatedSeconds`, `streak`, `lastPracticedAt`, `now`; `STREAK_MILESTONES = [3, 7, 14, 21, 28]`. Priority order is design §4. Wilted stays `…`.
6. **The timeline lives in one composable** (`composables/useGrowthMoment.ts`): before-values for the first paint, flip on `requestAnimationFrame`, chips at 150 ms, grow breath at 400 ms only when the stage did not change, chips out at 1800 ms, done at 2000 ms. The page only wires it. Durations are a `MOMENT_MS` constant the tests can shrink.
7. **Motion is CSS-only and gated**: every `animation`/`transition` declaration sits inside `@media (prefers-reduced-motion: no-preference)`; the DOM (chips, lines, final values) is identical under reduced motion, which is also why the unit tests can assert the DOM without emulating the media query.
8. Vietnamese copy from design §4/§5, sentence case; no confetti, no full-screen element, no sound.

**Tech stack:** Nuxt 3 / Vue 3 / Pinia / Tailwind 3.4 / Vitest + `@vue/test-utils` + `happy-dom`. No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` and `timeout` are not installed — use `grep -n`. Frontend checks: `cd frontend && npm run lint && npm run typecheck && npm run test:unit`. `@vue/test-utils` stubs `<Transition>`/`<TransitionGroup>` by default, so transitions never delay a test.

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/utils/plant.ts` | `stageForStreak`, `STREAK_MILESTONES`, `SpeechInput`, context-aware `speechLine` |
| `frontend/stores/quest.ts` | `CompleteResult`; `complete()` returns `targetMetChanged` |
| `frontend/stores/pet.ts` | `GrowthDelta`, `lastDelta`, guarded `applyProgress` that records it, `consumeDelta()` |
| `frontend/components/plant/GrowthChip.vue` | **new** — the "+20 máu" / "🔥 7 ngày" pill |
| `frontend/components/plant/PlantSvg.vue` | `<Transition name="stage" mode="out-in">` around the stage groups; `grow` prop; gated CSS |
| `frontend/components/ui/HealthBar.vue` | `duration-300` → `duration-[400ms]` (shell design: 400 ms) |
| `frontend/components/AppHeader.vue` | `pulse` prop → one-shot `streak-pulse` on the streak chip; gated CSS |
| `frontend/composables/useGrowthMoment.ts` | **new** — `chipsFor`, `MOMENT_MS`, `useGrowthMoment()` |
| `frontend/pages/index.vue` | consume the delta, wire the composable, chips overlay, extra bubble inputs |
| `frontend/tests/unit/plant.test.ts` | `stageForStreak` + `speechLine` tables |
| `frontend/tests/unit/questStore.test.ts` | `targetMetChanged` cases |
| `frontend/tests/unit/petStore.test.ts` | delta recorded / consumed once / nothing when unchanged / absent fields / wilted → sprout |
| `frontend/tests/unit/PlantSvg.test.ts` | `grow` class; stage prop change re-renders the new group |
| `frontend/tests/unit/useGrowthMoment.test.ts` | **new** — `chipsFor` table, phase flip, grow gating, dispose |
| `frontend/tests/unit/indexPage.test.ts` | **new** — chips when a delta exists, nothing otherwise, celebration only when the target newly met, final values after the flip, consumed once |
| `harness/CODEMAP.md` | `shell` paragraph |

`pages/learn/[id].vue` is **not** edited.

---

## Tasks

### Task 1: `utils/plant.ts` — stage table and a context-aware bubble

**Files:**
- Modify: `frontend/utils/plant.ts`, `frontend/tests/unit/plant.test.ts`

- [ ] **Step 1: Write the failing tests** in `plant.test.ts` (import `STREAK_MILESTONES`, `stageForStreak` alongside the existing names):
  - `stageForStreak follows the backend streak table and wilts at 0 health`: `(0, 80) → 'sprout'`, `(2, 80) → 'sprout'`, `(3, 80) → 'sapling'`, `(6, 80) → 'sapling'`, `(7, 80) → 'flowering'`, `(13, 80) → 'flowering'`, `(14, 80) → 'fruitful'`, `(30, 100) → 'fruitful'`, `(5, 0) → 'wilted'`.
  - `speechLine: partway lines count the minutes left, rounded up` (design §4 row 4): `{ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 600 }` → `'Còn 20 phút nữa thôi!'`; `1200` → `'Còn 10 phút nữa thôi!'`; `1770` → `'Còn 1 phút nữa thôi!'`; and `accumulatedSeconds: 600` with `health: 10` still gives the partway line (row 4 outranks health).
  - `speechLine: milestone streaks prefix the met line` (row 2/3): `{ stage: 'flowering', health: 100, targetMet: true, streak: 7 }` → `'7 ngày liên tiếp! Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'`; `streak: 8` → the plain met line; `it.each([3, 7, 14, 21, 28])` all prefix; `expect(STREAK_MILESTONES).toEqual([3, 7, 14, 21, 28])`.
  - `speechLine: a missed day is noticed only when nothing was studied today` (row 5): `now = new Date('2026-09-25T10:00:00Z')`; `{ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 0, lastPracticedAt: '2026-09-23T20:00:00Z', now }` → `'Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?'`; with `lastPracticedAt: '2026-09-24T20:00:00Z'` (yesterday, 1 whole day) → `'Tưới cho tớ 10 phút học đi!'`; with `accumulatedSeconds: 600` and the 2-day-old timestamp → the partway line; `lastPracticedAt: null` → the health line.
  - `speechLine: wilted stays silent whatever else is true`: `{ stage: 'wilted', health: 0, targetMet: true, streak: 7, accumulatedSeconds: 1800 }` → `'…'`.
  - Keep the existing `speaks the wireframe 7.2 line…` case unchanged — the three-argument call must still work.
- [ ] **Step 2: Run red:** `cd frontend && npm run test:unit -- plant` → the new cases fail (`stageForStreak` is not exported; the partway/milestone/missed lines are not produced).
- [ ] **Step 3: Make them pass.** In `utils/plant.ts` add `import { DAILY_TARGET_SECONDS } from '~/utils/progress'` (it imports nothing back — no cycle) and:

```ts
/** CODEMAP `pet`: wilted at 0 health, otherwise by streak — the table `SaveTargetMet` writes (backend engine.go). */
export function stageForStreak(streak: number, health: number): PlantStage {
  if (health <= 0) return 'wilted'
  if (streak >= 14) return 'fruitful'
  if (streak >= 7) return 'flowering'
  if (streak >= 3) return 'sapling'
  return 'sprout'
}

/** Streak days that earn a line of their own in the bubble (design growth-moment §4). */
export const STREAK_MILESTONES = [3, 7, 14, 21, 28] as const

export interface SpeechInput {
  stage: string
  health: number
  targetMet: boolean
  /** GET /quests/daily accumulated_seconds; > 0 means the learner is mid-day. */
  accumulatedSeconds?: number
  streak?: number
  lastPracticedAt?: string | null
  now?: Date
}

const MET_LINE = 'Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'

/** One line, first match wins — design growth-moment §4. */
export function speechLine(o: SpeechInput): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) {
    const milestone = o.streak !== undefined && (STREAK_MILESTONES as readonly number[]).includes(o.streak)
    return milestone ? `${o.streak} ngày liên tiếp! ${MET_LINE}` : MET_LINE
  }
  const accumulated = o.accumulatedSeconds ?? 0
  if (accumulated > 0) return `Còn ${Math.max(1, Math.ceil((DAILY_TARGET_SECONDS - accumulated) / 60))} phút nữa thôi!`
  const missed = daysSince(o.lastPracticedAt, o.now)
  if (missed !== null && missed >= 2) return 'Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?'
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất! Học một chút nhé?'
}
```

  `daysSince` is already in the file (move it above `speechLine` or leave it — function declarations hoist).
- [ ] **Step 4: Run green:** `npm run test:unit -- plant` → all cases pass; `npm run typecheck` clean (`pages/index.vue` still compiles with the three-field call).
- [ ] **Step 5: Commit:** `git commit -am "frontend: stageForStreak and a context-aware speechLine with streak milestones"`.

### Task 2: `stores/quest.ts` — `complete()` says whether the target was newly met

**Files:**
- Modify: `frontend/stores/quest.ts`, `frontend/tests/unit/questStore.test.ts`

- [ ] **Step 1: Write the failing test** in `questStore.test.ts`, `complete reports targetMetChanged only when the response newly meets the target`: load `daily` (`is_target_met: false`); `api.post` resolves `{ daily_seconds_spent: 1800, daily_minutes_spent: 30, is_target_met: true, pet_health: 100, streak_count: 6 }` → `(await q.complete('ex-2', 600)).targetMetChanged === true` and `q.targetMet === true`; a second `complete('ex-3', 600)` with the same response → `targetMetChanged === false` (the day was already met); reset the store, load `daily`, respond `is_target_met: false` → `false`. Also assert the existing spread fields are still on the result (`res.pet_health === 100`).
- [ ] **Step 2: Run red:** `npm run test:unit -- questStore` → `targetMetChanged` is `undefined`.
- [ ] **Step 3: Make it pass.** Add after `ProgressResponse`:

```ts
/** What `complete()` resolves: the §6.2 body plus whether this call newly met today's target (hub growth moment). */
export type CompleteResult = ProgressResponse & { targetMetChanged: boolean }
```

  and in `complete()`: change the signature to `Promise<CompleteResult>`, read `const wasMet = this.daily?.is_target_met ?? false` **before** the `if (this.daily)` block that overwrites the flag, and end with `return { ...res, targetMetChanged: res.is_target_met && !wasMet }`. Nothing else in the store changes — the timer plan (`task-timer-keeps-counting-through-reloads-and-background-tab`, executed before this one in the 14:00 run) owns `Timer`, `startTimer`, `tick`, `elapsedSeconds` and adds a persisted-entry removal inside `complete()`; if the worktree already has it, keep its line and add these two around it.
- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit -- questStore` → clean. `pages/learn/[id].vue` compiles unchanged (`pet.applyProgress(res)` receives a superset).
- [ ] **Step 5: Commit:** `git commit -am "frontend: quest.complete reports targetMetChanged"`.

### Task 3: `stores/pet.ts` — record the delta, hand it out once

**Files:**
- Modify: `frontend/stores/pet.ts`, `frontend/tests/unit/petStore.test.ts`

- [ ] **Step 1: Write the failing tests** in `petStore.test.ts` next to the existing `applyProgress` case (`status` fixture: health 80, streak 5, `sprout`):
  - `applyProgress records a delta the hub consumes exactly once`: `applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })` → `pet.lastDelta` equals `{ healthFrom: 80, healthTo: 100, streakFrom: 5, streakTo: 6, stageFrom: 'sprout', stageTo: 'sapling', targetMetNow: true }` and `pet.status.stage === 'sapling'`; `pet.consumeDelta()` returns that object; a second `consumeDelta()` returns `null`; `pet.lastDelta` is `null`.
  - `applyProgress records nothing when nothing changed` (task 1 or 2 of the day): `applyProgress({ pet_health: 80, streak_count: 5, targetMetChanged: false })` → `consumeDelta()` is `null`, status unchanged.
  - `applyProgress leaves the plant alone and records no delta when the response omits the pet fields`: `applyProgress({})` → status still `{ health_points: 80, current_streak: 5, stage: 'sprout' }`, `consumeDelta()` null. (Today's bug plan adds a case with a similar name; if it is already in the file when you get here, extend it with the `consumeDelta()` assertion instead of duplicating it. Its `still applies a real 0` case must keep passing.)
  - `applyProgress takes a wilted plant to sprout when the day is met`: load `{ ...status, health_points: 0, stage: 'wilted', current_streak: 0 }`; `applyProgress({ pet_health: 20, streak_count: 1, targetMetChanged: true })` → status `{ health_points: 20, current_streak: 1, stage: 'sprout' }`, `pet.isWilted === false`, delta `stageFrom: 'wilted', stageTo: 'sprout'`.
  - `applyProgress clears a stale unconsumed delta`: two `applyProgress` calls in a row, the first changing values, the second changing nothing → `consumeDelta()` is `null`.
- [ ] **Step 2: Run red:** `npm run test:unit -- petStore` → `lastDelta`/`consumeDelta` undefined.
- [ ] **Step 3: Make them pass.** In `stores/pet.ts` import `stageForStreak` from `~/utils/plant`, add:

```ts
/** What the last progress response changed — the hub's growth moment reads it once (design growth-moment §3). */
export interface GrowthDelta {
  healthFrom: number
  healthTo: number
  streakFrom: number
  streakTo: number
  stageFrom: string
  stageTo: string
  targetMetNow: boolean
}
```

  state gains `lastDelta: null as GrowthDelta | null`, and `applyProgress` becomes:

```ts
    /**
     * §6.2 progress response. Either pet field is absent when the backend's pet
     * read failed (CODEMAP quests) — then it is neither applied nor counted.
     * `targetMetChanged` comes from `useQuestStore.complete()`.
     */
    applyProgress(res: { pet_health?: number, streak_count?: number, targetMetChanged?: boolean }) {
      this.lastDelta = null
      if (!this.status) return
      const from = { health: this.status.health_points, streak: this.status.current_streak, stage: this.status.stage }
      if (typeof res.pet_health === 'number') this.status.health_points = res.pet_health
      if (typeof res.streak_count === 'number') this.status.current_streak = res.streak_count
      // Stage after: the same table the backend's SaveTargetMet writes; GET /pet/status on the hub confirms it.
      if (typeof res.pet_health === 'number' && typeof res.streak_count === 'number') this.status.stage = stageForStreak(res.streak_count, res.pet_health)
      const to = { health: this.status.health_points, streak: this.status.current_streak, stage: this.status.stage }
      const targetMetNow = res.targetMetChanged === true
      if (to.health === from.health && to.streak === from.streak && to.stage === from.stage && !targetMetNow) return
      this.lastDelta = { healthFrom: from.health, healthTo: to.health, streakFrom: from.streak, streakTo: to.streak, stageFrom: from.stage, stageTo: to.stage, targetMetNow }
    },
    /** The hub takes the delta once; a reload or a second visit sees the steady state. */
    consumeDelta(): GrowthDelta | null {
      const d = this.lastDelta
      this.lastDelta = null
      return d
    },
```

- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit -- petStore` → clean; the pre-existing `applyProgress updates health and streak` case still passes (its stage now reads `sapling`, which `toMatchObject` does not check).
- [ ] **Step 5: Commit:** `git commit -am "frontend: pet store records a one-shot growth delta from applyProgress"`.

### Task 4: Components — `GrowthChip`, stage cross-fade + grow breath, header pulse, 400 ms bar

**Files:**
- Create: `frontend/components/plant/GrowthChip.vue`
- Modify: `frontend/components/plant/PlantSvg.vue`, `frontend/components/AppHeader.vue`, `frontend/components/ui/HealthBar.vue`, `frontend/tests/unit/PlantSvg.test.ts`

- [ ] **Step 1: Write the failing tests.** In `PlantSvg.test.ts`:
  - `grow adds the one-shot breath class to the plant, not the pot`: `mount(PlantSvg, { props: { stage: 'sapling', health: 80, grow: true } })` → `w.find('[data-plant]').classes()` contains `plant-grow`; without `grow` it does not; the pot `path` is outside `[data-plant]`.
  - `a stage change renders the new stage group`: mount at `sprout`, `await w.setProps({ stage: 'sapling' })` → `[data-stage="sapling"]` exists, `[data-stage="sprout"]` does not (the transition is stubbed; this pins that the `v-if` chain still works inside `<Transition>`).
  - Add `tests/unit/GrowthChip.test.ts`: `mount(GrowthChip, { props: { text: '+20 máu', tone: 'growth' } })` → text `+20 máu`, `attributes('data-growth-chip') === 'growth'`, classes contain `text-growth`; tone `streak` → `text-streak`.
- [ ] **Step 2: Run red:** `npm run test:unit -- PlantSvg GrowthChip` → missing component / class.
- [ ] **Step 3: Make them pass.**
  - `components/plant/GrowthChip.vue` (design §5/§6):

```vue
<script setup lang="ts">
defineProps<{ text: string, tone: 'growth' | 'streak' }>()
</script>

<template>
  <span
    :data-growth-chip="tone"
    class="inline-flex rounded-full px-3 py-1 text-sm font-semibold tabular-nums"
    :class="tone === 'growth' ? 'bg-growth/15 text-growth' : 'bg-streak/15 text-streak'"
  >{{ text }}</span>
</template>
```

  - `PlantSvg.vue`: add `grow?: boolean` to the props (default `false`). Wrap the six stage `<g>` groups (not the pot/soil) in `<g data-plant :class="{ 'plant-grow': grow }"><Transition name="stage" mode="out-in"> … </Transition></g>`. Inside the existing `@media (prefers-reduced-motion: no-preference)` block add `.stage-enter-active, .stage-leave-active { transition: opacity 300ms ease; }`, `.stage-enter-from, .stage-leave-to { opacity: 0; }`, and `.plant-grow { transform-origin: 80px 110px; animation: grow 800ms cubic-bezier(.2, .8, .2, 1) 1; }`; add `@keyframes grow { 0% { transform: scale(1); } 45% { transform: scale(1.06); } 100% { transform: scale(1); } }` next to the existing keyframes (outside the media block, like `sway`). Do **not** add a second `prefers-reduced-motion: no-preference` block.
  - `AppHeader.vue`: props become `defineProps<{ streak?: number | null, pulse?: boolean }>()`; the streak `<span>` gets `:class="{ 'streak-pulse': pulse }"` and `data-streak-chip`; add a `<style scoped>` with `@media (prefers-reduced-motion: no-preference) { .streak-pulse { animation: streak-pulse 500ms ease-out 1; } }` and `@keyframes streak-pulse { 0%, 100% { transform: scale(1); } 50% { transform: scale(1.12); } }`.
  - `HealthBar.vue`: `duration-300` → `duration-[400ms]` (Tailwind 3.4 has no `duration-400`); keep `motion-reduce:transition-none`.
- [ ] **Step 4: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `grep -c 'prefers-reduced-motion: no-preference' frontend/components/plant/PlantSvg.vue frontend/components/AppHeader.vue` → `1` each.
- [ ] **Step 5: Commit:** `git commit -am "frontend: GrowthChip, PlantSvg stage cross-fade and grow breath, header streak pulse, 400 ms health bar"`.

### Task 5: `useGrowthMoment` + the hub

**Files:**
- Create: `frontend/composables/useGrowthMoment.ts`, `frontend/tests/unit/useGrowthMoment.test.ts`, `frontend/tests/unit/indexPage.test.ts`
- Modify: `frontend/pages/index.vue`

- [ ] **Step 1: Write the failing composable tests** (`useGrowthMoment.test.ts`; `vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })` in `beforeEach`; `vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })` so `flushPromises` (which uses `setImmediate`) keeps working; run the composable inside `effectScope()` so `onScopeDispose` has a scope):
  - `chipsFor` table: `{80→100, 5→6, targetMetNow: true}` → `[{ tone: 'growth', text: '+20 máu' }, { tone: 'streak', text: '🔥 6 ngày' }]`; `{100→100, 6→7, true}` → `[{ growth, 'Máu đầy' }, { streak, '🔥 7 ngày' }]`; `{80→80, 5→5, false}` → `[]`; `{0→20, 0→1, true}` → `+20 máu`, `🔥 1 ngày`.
  - `start(null) is inert`: `active.value === false`, `chips.value` empty, `displayHealth(80) === 80`.
  - `before-values for the first paint, then live`: with the stubbed rAF calling synchronously, assert `displayHealth(100)` after `start(d)` — the stub flips the phase inside `start`, so use a *deferred* stub for this case (`vi.stubGlobal('requestAnimationFrame', (cb) => { pending = cb; return 1 })`): before `pending()` → `displayHealth(100) === 80` and `displayStage('sapling') === 'sprout'`; after `pending(0)` → `100` / `'sapling'`.
  - `timeline`: `start(d)` with `targetMetNow: true`, same stage → at `vi.advanceTimersByTime(149)` chips empty; `150` → two chips, `pulse.value === true`; `400` → `grow.value === true`; `1800` → chips empty; `2000` → `active.value === false`, `grow.value === false`, `pulse.value === false`.
  - `grow only when the target was newly met and the stage did not change`: `{ targetMetNow: false }` → `grow` stays `false` at 400 ms; `{ targetMetNow: true, stageFrom: 'sprout', stageTo: 'sapling' }` → `grow` stays `false` (the cross-fade is the growth).
  - `dispose clears the timers`: `scope.stop()` at 100 ms, advance to 2000 ms → `chips.value` still empty and no error.
- [ ] **Step 2: Write the failing page tests** (`indexPage.test.ts`, modelled on `revivePage.test.ts`: `api` mock via `vi.mock('~/composables/useApi')`, `setActivePinia(createPinia())`, `global.components` = `AppHeader, AppCard, StateBlock, PlantSvg, HealthBar, SpeechBubble, SegmentedProgress, QuestRow, GrowthChip`, `global.stubs = { NuxtLink: { template: '<a><slot /></a>' } }`, `global.mocks = { navigateTo: vi.fn() }`; same rAF stub and fake `setTimeout` as above; `api.get` routes `/pet/status` → `STATUS_AFTER` (`health 100, streak 6, sapling`) and `/quests/daily` → `DAILY_MET` (`accumulated_seconds: 1800, is_target_met: true`, three tasks completed)). Before mounting, seed the pet store as the learn page would have left it: `pet.status = { …STATUS_BEFORE (80, 5, sprout) }` then `pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })`, and `quest.daily = DAILY_MET`.
  - `shows the growth moment after a task that met the target`: mount, `await flushPromises()`, `vi.advanceTimersByTime(200)`, `await nextTick()` → `[data-growth-chip="growth"]` text `+20 máu`, `[data-growth-chip="streak"]` text `🔥 6 ngày`, `[data-streak-chip]` has class `streak-pulse`, bubble text contains `đủ nước`; `vi.advanceTimersByTime(300)` (t = 500) → `[data-plant]` **does not** have `plant-grow` because the stage changed (`sprout → sapling`, cross-fade instead); `vi.advanceTimersByTime(1500)` → no `[data-growth-chip]` left.
  - `takes the grow breath when the stage did not change`: seed `STATUS_BEFORE` as `{ 80, 6, sapling }` and apply `{ 100, 7, true }` (stage stays `sapling`) → at t = 500 `[data-plant]` has `plant-grow`; at t = 2000 it does not.
  - `renders the final values after the flip — what a reduced-motion user sees`: after `flushPromises` → `[role="meter"]` `aria-valuenow === '100'`, `[data-stage="sapling"]` exists, header text contains `Streak: 6 ngày`, bubble contains `đủ nước`.
  - `shows nothing on a plain load`: no `applyProgress` before mount → after 200 ms no `[data-growth-chip]`, no `plant-grow`, no `streak-pulse`; bubble is the context line for `DAILY` with `accumulated_seconds: 600, is_target_met: false` → `Còn 20 phút nữa thôi!`.
  - `a task that did not meet the target celebrates nothing`: seed `{ 80, 5, sprout }`, `applyProgress({ pet_health: 80, streak_count: 5, targetMetChanged: false })`, daily `600 / false` → no chips, no `plant-grow`, no `streak-pulse`, bubble `Còn 20 phút nữa thôi!`.
  - `the moment is consumed once`: after the first test's mount, `w.unmount()`, mount again (same pinia) → no `[data-growth-chip]` at 200 ms.
- [ ] **Step 3: Run red:** `npm run test:unit -- useGrowthMoment indexPage` → module not found / no chips.
- [ ] **Step 4: Make them pass.**
  - `composables/useGrowthMoment.ts`:

```ts
import { computed, onScopeDispose, ref } from 'vue'
import type { GrowthDelta } from '~/stores/pet'

export interface GrowthChipSpec { tone: 'growth' | 'streak', text: string }

/** Design growth-moment §2 timeline, in ms from the hub's first paint. */
export const MOMENT_MS = { chipsIn: 150, grow: 400, chipsOut: 1800, end: 2000 } as const

/** The chips a delta earns (design §3). Health first, streak second; nothing for an unchanged value. */
export function chipsFor(d: GrowthDelta): GrowthChipSpec[] {
  const out: GrowthChipSpec[] = []
  if (d.healthTo > d.healthFrom) out.push({ tone: 'growth', text: `+${d.healthTo - d.healthFrom} máu` })
  else if (d.targetMetNow && d.healthTo >= 100) out.push({ tone: 'growth', text: 'Máu đầy' })
  if (d.streakTo > d.streakFrom) out.push({ tone: 'streak', text: `🔥 ${d.streakTo} ngày` })
  return out
}

/**
 * The hub's growth moment: before-values for one paint, then the store's
 * values, chips in at 150 ms, one grow breath at 400 ms (only when the stage
 * did not change — a stage change cross-fades instead), chips out at 1800 ms.
 * All motion is CSS behind prefers-reduced-motion; this only schedules classes.
 */
export function useGrowthMoment(ms: typeof MOMENT_MS = MOMENT_MS) {
  const delta = ref<GrowthDelta | null>(null)
  const painted = ref(false)
  const chipsVisible = ref(false)
  const grow = ref(false)
  const pulse = ref(false)
  const timers: ReturnType<typeof setTimeout>[] = []

  function start(d: GrowthDelta | null) {
    if (!d) return
    delta.value = d
    painted.value = false
    const flip = () => { painted.value = true }
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(flip)
    else flip()
    timers.push(
      setTimeout(() => { chipsVisible.value = true; pulse.value = d.streakTo > d.streakFrom }, ms.chipsIn),
      setTimeout(() => { grow.value = d.targetMetNow && d.stageTo === d.stageFrom }, ms.grow),
      setTimeout(() => { chipsVisible.value = false }, ms.chipsOut),
      setTimeout(() => { delta.value = null; grow.value = false; pulse.value = false }, ms.end),
    )
  }

  onScopeDispose(() => timers.forEach(clearTimeout))

  return {
    start,
    active: computed(() => delta.value !== null),
    chips: computed(() => (delta.value && chipsVisible.value ? chipsFor(delta.value) : [])),
    grow,
    pulse,
    displayHealth: (live: number) => (delta.value && !painted.value ? delta.value.healthFrom : live),
    displayStage: (live: string) => (delta.value && !painted.value ? delta.value.stageFrom : live),
  }
}
```

  - `pages/index.vue`: `const { start, chips, grow, pulse, displayHealth, displayStage } = useGrowthMoment()` (destructure — nested refs on a plain object are not unwrapped in templates); `onMounted` calls `start(pet.consumeDelta())` **before** the `Promise.all([...load()])`; `bubble` passes `accumulatedSeconds: quest.accumulatedSeconds, streak: pet.status.current_streak, lastPracticedAt: pet.status.last_practiced_at`. Template: `<AppHeader :streak="…" :pulse="pulse" />`; the plant card becomes `<AppCard class="relative mb-4">`, and inside its `pet.status` branch, before `<PlantSvg>`:

```vue
        <TransitionGroup name="chip" tag="div" class="absolute right-4 top-4 flex flex-col items-end gap-1.5" aria-live="polite">
          <GrowthChip v-for="c in chips" :key="c.tone" :text="c.text" :tone="c.tone" />
        </TransitionGroup>
        <PlantSvg :stage="displayStage(pet.status.stage)" :health="displayHealth(pet.status.health_points)" :grow="grow" />
        <HealthBar class="mt-3" :health="displayHealth(pet.status.health_points)" />
```

  and a `<style scoped>` with `@media (prefers-reduced-motion: no-preference) { .chip-enter-active { transition: opacity 200ms ease-out, transform 200ms ease-out; } .chip-leave-active { transition: opacity 200ms ease-out; } .chip-enter-from { opacity: 0; transform: translateY(8px); } .chip-leave-to { opacity: 0; } }`.
- [ ] **Step 5: Run green:** `npm run lint && npm run typecheck && npm run test:unit` → clean; `npm run build` succeeds (CI's frontend job runs it).
- [ ] **Step 6: Look at it once** (optional but cheap): `npm run dev` with the backend from `backend/` up (`COMPOSE_PROJECT_NAME=<slug>`), complete the third task of a day, confirm on the hub: bar slides, two chips for ~1.7 s, plant cross-fades or breathes, header chip pulses; then set the OS "reduce motion" and repeat: same chips and lines, no movement. Record the result in the plan's Notes.
- [ ] **Step 7: Commit:** `git commit -am "frontend: growth moment on the hub — useGrowthMoment, chips, before-value flip"`.

### Task 6: CODEMAP

**Files:**
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1:** In the `shell` paragraph, extend the `stores/pet.ts` clause: after "`POST /pet/revive` with a client-side 15-minute challenge anchor at `localStorage['aelp.revive']`" add "; `applyProgress` assigns only the pet fields the response carries and records a one-shot `lastDelta` (health/streak/stage before→after — stage derived by `utils/plant.ts` `stageForStreak` from the backend's streak table — plus `targetMetNow`) that `pages/index.vue` takes once via `consumeDelta()` for the **growth moment** (`harness/designs/growth-moment.md`: `composables/useGrowthMoment.ts` schedules ≤ 2 s of CSS-only motion — health bar slide from the before-value, `GrowthChip`s, `PlantSvg` stage cross-fade or one grow breath, `AppHeader` streak pulse — every keyframe behind `prefers-reduced-motion: no-preference`)". Extend the `stores/quest.ts` clause with "`complete()` resolves `CompleteResult` = the §6.2 body + `targetMetChanged`". Extend the `/` page entry with "speech bubble via `speechLine` (partway minutes left, missed-day, streak milestones 3/7/14/21/28, met, wilted `…`)". If the bug plan's "assigns only the pet fields" clause is already there, merge rather than duplicate.
- [ ] **Step 2:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "harness: CODEMAP — hub growth moment"`.

---

## Verification

```bash
cd frontend
npm run lint && npm run typecheck && npm run test:unit
# expect: all clean; new files useGrowthMoment.test.ts, indexPage.test.ts, GrowthChip.test.ts pass
find tests/unit -type f | wc -l
# expect: 20 (was 17; `ls` output can be mangled by the rtk proxy)
npm run build
# expect: success
grep -c 'consumeDelta' stores/pet.ts
# expect: ≥ 2 (the action and its doc comment)
grep -c 'targetMetChanged' stores/quest.ts
# expect: ≥ 2 (the CompleteResult type and the return)
grep -c 'pet.applyProgress(res)' 'pages/learn/[id].vue'
# expect: 1 (the learn page is untouched)
grep -c 'STREAK_MILESTONES' utils/plant.ts
# expect: ≥ 2 (definition and use)
grep -c 'duration-300' components/ui/HealthBar.vue
# expect: 0
grep -c '<Transition name="stage" mode="out-in">' components/plant/PlantSvg.vue
# expect: 1
grep -c 'prefers-reduced-motion: no-preference' components/plant/PlantSvg.vue components/AppHeader.vue pages/index.vue
# expect: 1 for each file
grep -c '@media' composables/useGrowthMoment.ts
# expect: 0 — the composable schedules classes only; every media query lives in component CSS
grep -rn 'GrowthChip' pages components --include='*.vue' | wc -l
# expect: ≥ 1 (the hub renders it)
grep -c 'requestAnimationFrame' composables/useGrowthMoment.ts
# expect: ≥ 1
cd ..
grep -c 'growth moment' harness/CODEMAP.md
# expect: 1
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output (a plan branch edits no other harness file)
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: backend-unit, backend-integration, harness-tooling, frontend green
```

Mutation checks (record the results in Notes, then revert each):
1. In `stores/pet.ts` `consumeDelta`, drop `this.lastDelta = null` → `applyProgress records a delta the hub consumes exactly once` and `the moment is consumed once` go red.
2. In `utils/plant.ts` `speechLine`, move the three health rows above the partway row → `partway lines count the minutes left` (`health: 10` case) goes red.
3. In `useGrowthMoment.ts`, change `d.targetMetNow && d.stageTo === d.stageFrom` to `d.targetMetNow` → `grow only when …` and `shows the growth moment after a task that met the target` (no `plant-grow` at t = 500) go red.
4. In `chipsFor`, delete the `Máu đầy` branch → the `100→100` table row goes red.

## Notes and open questions

- **Same-day overlap 1 (lands first):** `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` (bug, 10:00 run) rewrites `applyProgress` to the guarded form and marks `ProgressResponse.pet_health?`/`streak_count?` optional. Task 3's `applyProgress` is that guard plus the delta; if the bug branch is already merged into the base you start from, Task 3 replaces its body and keeps its two tests. Task 2 does not need the optional marks but is compatible with them.
- **Same-day overlap 2 (executed before this in the 14:00 run):** the timer feature plan changes `stores/quest.ts` timers and `complete()`'s timer cleanup. This plan touches only `complete()`'s `wasMet` read and return value; expect at most a trivial textual conflict inside `complete()`.
- **Streak shield** (`harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md`, design in progress): its "Khiên mới!" announcement fits as a third chip in the same `TransitionGroup`; nothing here depends on it.
- **Why the stage is written client-side:** the hub would otherwise show the old stage until `pet.load()` returns, and the cross-fade would fire at network speed, not at the flip. `stageForStreak` is a copy of the backend table (`backend/internal/pet/engine.go:51-62`); if the backend ever changes the thresholds, `GET /pet/status` still corrects the hub within one request — the client value is a preview, not a source of truth. A future backend idea could add `stage` to the §6.2 response and delete `stageForStreak`.
- **`requestAnimationFrame` in happy-dom:** the composable falls back to a synchronous flip when it is missing; tests stub it explicitly, so nothing depends on happy-dom's implementation.
- **`SegmentedProgress` keeps `duration-300`** — its fill already eases on value change; aligning it to 400 ms is cosmetic and out of scope.
- **Inferred, not measured:** the ≤ 2 s feel and the chip corner placement come from the design's reasoning, not a device test. Task 5 Step 6 is where the executor checks it and may shorten `MOMENT_MS.chipsOut` (never above 1800).
