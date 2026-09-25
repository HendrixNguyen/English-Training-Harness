---
type: feature
status: proposed
source: ideator
run: 2026-09-25-run-01
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
