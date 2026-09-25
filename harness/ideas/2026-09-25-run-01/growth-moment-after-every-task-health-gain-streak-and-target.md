---
type: feature
status: planned
source: ideator
run: 2026-09-25-run-01
priority: medium
plan: harness/plans/2026-09-25-growth-moment-after-every-task-health-gain-streak-and-target.md
---
# Growth moment after every task: health gain, streak and target-met celebration on the hub

## Why
1st-thinking §5.2 ends the daily loop at step 5: "Render Growth Animation & Status". That step is the retention mechanism — the plant is why the learner comes back, and the moment it visibly grows is the reward that closes the habit loop. Today the loop is silent: `learn/[id].vue` `complete()` posts progress, `pet.applyProgress(res)` overwrites `health_points` and `current_streak` in the store, and the app `navigateTo('/')`. The hub re-renders with new numbers and nothing announces what just happened. A learner who took the plant from 60 to 80 health, extended a streak to 7 days, or crossed 30 minutes and moved the stage from `sprout` to `sapling`, sees the same static page as before. The `SpeechBubble` has three fixed lines keyed on health only, and `PlantSvg` has a sway animation and no transition between stages.

Duolingo attributes a measurable day-7 retention lift to a single redesigned streak animation, and gates dramatic celebration to milestones so each one stays rare. We have all the ingredients (health delta, streak, stage before/after, `is_target_met` before/after) in the `POST /quests/progress` response and the previous store state; we just throw them away. The pending *streak shield* idea (2026-09-22-run-01) assumes a "daily-quest completion animation" exists to announce shields — it does not yet. This idea builds it.

## Expected output
User-visible:
- After a task is recorded, the learner lands on the hub with a **growth moment**: the health bar animates from the old value to the new one, a chip shows "+20 máu" (or "+0" capped at 100), and if the streak changed the header's 🔥 count ticks up with a brief highlight.
- Crossing the daily target (first response with `is_target_met: true` for the day) gets a distinct, one-time celebration: the plant does a short grow animation, the speech bubble says the day is done (e.g. "Đủ 30 phút rồi! Mai gặp lại nhé 🌱"), and the segmented progress bar fills to "met". A stage change (`sprout → sapling → flowering → fruitful`) cross-fades the `PlantSvg` instead of swapping instantly.
- Streak milestones 3, 7, 14, 21, 28 get one extra line in the bubble ("7 ngày liên tiếp!"). No confetti or full-screen takeover — this is a ≤ 2 s moment on the hub, skipped entirely under `prefers-reduced-motion` (values still update).
- The speech bubble becomes context-aware from data the hub already has: not started today / partway / met; health tone; `last_practiced_at` older than a day ("Hôm qua tớ nhớ bạn…"); wilted stays `…` and the revive banner.
- Nothing fires on a plain page load or when the values did not change; refreshing the hub after a celebration shows the steady state.

Technical:
- `stores/pet.ts`: `applyProgress` records a `lastDelta: { healthFrom, healthTo, streakFrom, streakTo, stageFrom, stageTo, targetMetNow }` (stage after is derived client-side from the existing threshold table until a later idea adds it to the response) and the hub consumes and clears it once (`consumeDelta()`), so it survives the `navigateTo('/')` but not a reload.
- `stores/quest.ts`: expose `targetMetChanged` from the `complete()` response vs. the previous `daily.is_target_met`.
- `utils/plant.ts`: `speechLine` gains `accumulatedSeconds`, `streak`, `lastPracticedAt` inputs and a milestone table; pure function, fully unit-tested.
- `components/ui/HealthBar.vue`: CSS transition on width; `components/plant/PlantSvg.vue`: `<Transition>` between stage groups and a one-shot `.plant-grow` keyframe class; a small `GrowthChip.vue` for "+20 máu" / "🔥 7". All motion behind `@media (prefers-reduced-motion: no-preference)`.
- Tests: `speechLine` table; `consumeDelta` returns once; hub renders the chip when a delta is present and nothing when absent; target-met celebration only when `targetMetChanged`. No backend change; `POST /quests/progress` and `GET /pet/status` shapes untouched (backend spec §6.2/§6.3).
- CODEMAP `shell` paragraph updated.

## Evidence
- 1st-thinking §5.2 step 5 "Render Growth Animation & Status" (unimplemented); §1 "gamified … virtual pet/plant state engine" as the retention driver.
- Frontend spec §7.2 wireframe (plant hub with speech bubble "Tưới cho tớ 10 phút học đi!", streak in header, today's progress bar) and §6 design system (motion tokens for the plant).
- Backend spec §6.2 `POST /quests/progress` returns `pet_health`, `streak_count`, `is_target_met` — the deltas are computable client-side with no new endpoint; §8 success logic (+20 capped at 100, streak+1) is what the chip mirrors.
- Code: `frontend/pages/learn/[id].vue` `complete()` (silent `applyProgress` then `navigateTo('/')`), `frontend/stores/pet.ts` `applyProgress`, `frontend/utils/plant.ts` `speechLine` (three fixed lines), `frontend/components/plant/PlantSvg.vue` (stage groups swap with no transition), CODEMAP `pet` (stage thresholds by streak: 0–2 sprout, 3–6 sapling, 7–13 flowering, 14+ fruitful).
- Prior idea depending on this: `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md` ("the daily-quest completion animation announces 'Shield earned'").
- Duolingo: redesigned streak-milestone animation moved D7 retention; celebrations gated to milestones: https://blog.duolingo.com/streak-milestone-design-animation ; streak mechanic and 7-day effect on next-day retention: https://duolingo.deconstructoroffun.com/mechanics/streaks ; retention-team breakdown of streak reveal moments: https://www.recall.it/summary/business/behind-the-product-duolingo-streaks-or-jackson-shuttleworth-group-pm-retention-team

## Evaluation
_Evaluator, 2026-09-25 — daily decide (feature slot 3 of 5; two-cap rule, owner 2026-09-25)._

**Verdict: select, `priority: medium`.** 1st-thinking §5.2 step 5 "Render Growth Animation & Status" is the only step of the daily loop with no implementation, and it is the reward that closes the habit loop the whole pet engine exists for. Nothing is broken and no user is blocked, so it is not `high`; it is the clearest retention value in this run, so it is not `low`. Planned today.

**Why, checked against the code today (`origin/main` as merged into this branch):**
- `frontend/pages/learn/[id].vue` `complete()` → `quest.complete(...)`, `pet.applyProgress(res)`, `navigateTo('/', { replace: true })` — nothing is announced. `frontend/stores/pet.ts` `applyProgress` overwrites `health_points` / `current_streak` and keeps no before-value. `frontend/stores/quest.ts` `complete()` overwrites `daily.is_target_met` and discards the previous flag.
- `frontend/pages/index.vue` calls `pet.load()` and `quest.load()` in `onMounted`, so by the time the hub renders the store already holds the *after* numbers; `HealthBar`'s existing `transition-[width]` never sees the *before* value. The growth moment therefore has to carry the before-value itself (the `lastDelta` below) and render it for one frame.
- `frontend/utils/plant.ts` `speechLine` keys on `{stage, health, targetMet}` only — three lines plus the met line; `accumulated_seconds`, `current_streak` and `last_practiced_at` are already on the hub and unused. `PlantSvg.vue` swaps stage `<g>` groups with `v-if`/`v-else-if` — no transition.
- **One correction to the idea's arithmetic:** the backend applies +20 / streak+1 **exactly once per local day, when the 30-minute target is met** (CODEMAP `pet`: `Service.OnTargetMet` → `Repo.SaveTargetMet`, conditional on `last_target_met_date < D`). So `pet_health` / `streak_count` in a `POST /quests/progress` response change only on the task that crosses the target; the two earlier tasks return the unchanged values. The health chip and the plant's grow keyframe are therefore inherently target-met events, and "growth after every task" for tasks 1–2 is the segment fill (already animated) plus a context-aware bubble line ("Còn 20 phút nữa thôi!"). That is the right gating anyway — the chip stays rare.

**Achievable in one plan:** yes — frontend only, no wire change (backend spec §6.2/§6.3 untouched). Stage-after is derived client-side from the same threshold table the backend uses (`backend/internal/pet/engine.go:51-62` / `repo.go:94-96`: wilted at 0, else by streak 0–2 sprout, 3–6 sapling, 7–13 flowering, 14+ fruitful), and `pet.load()` on the hub confirms it from `GET /pet/status` moments later.

**Decisions carried into the plan (`harness/plans/…growth-moment…`, design `harness/designs/growth-moment.md`):**
1. `stores/pet.ts` `applyProgress` records `lastDelta { healthFrom, healthTo, streakFrom, streakTo, stageFrom, stageTo, targetMetNow }`; the hub takes it once via `consumeDelta()`. It survives `navigateTo('/')` (Pinia state) and not a reload — so a refresh shows the steady state.
2. `stores/quest.ts` `complete()` returns `ProgressResponse & { targetMetChanged }` (response `is_target_met` vs the previous `daily.is_target_met`); `learn/[id].vue` passes it through unchanged.
3. `utils/plant.ts`: `speechLine` gains `accumulatedSeconds`, `streak`, `lastPracticedAt` (+ `now` for tests) and a milestone table `[3, 7, 14, 21, 28]`; pure, table-tested; wilted stays `…`. New pure `stageForStreak(streak, health)`.
4. Motion ≤ 2 s, nothing full-screen: health bar width transition from the before-value, `PlantSvg` `<Transition>` cross-fade between stage groups plus a one-shot grow keyframe on target-met, a `GrowthChip.vue` ("+20 máu", "🔥 7 ngày"), a one-shot pulse on the header streak chip. Every keyframe/transition sits behind `@media (prefers-reduced-motion: no-preference)`; the chip text and the final values render either way. Nothing fires on a plain load or when nothing changed.
5. Vietnamese copy, sentence case, the shell design's register.

**Same-day overlaps (both branches land before this one):**
- `harness/plans/2026-09-25-a-pet-state-failure-reports-pet-health-0-which-means-a-dead-.md` (bug, 10:00 run) changes `applyProgress` to accept `{ pet_health?, streak_count? }` and assign only numbers. This plan's `applyProgress` is written with that guard already in it (no assignment and no delta for an absent field), so the merge is a superset, not a conflict of intent.
- The timer feature plan (`task-timer-keeps-counting-through-reloads-and-background-tab`, `priority: high`, executed first in the 14:00 run) rewrites `stores/quest.ts` timers and removes persisted entries in `complete()`. This plan's `quest.ts` edit is confined to `complete()`'s return value and the read of the previous `daily.is_target_met`; the executor rebases onto the timer branch's `complete()` if both touch the same lines.
- The pending streak-shield idea (`harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md`, design being written in parallel) announces a newly earned shield with a speech line of its own; the growth moment can absorb that later ("Khiên mới!" as a second chip) but does not depend on it.

**Dependencies on unbuilt work:** none.
