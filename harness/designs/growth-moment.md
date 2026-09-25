# Design: Growth moment — the hub after a recorded task

**Idea:** `harness/ideas/2026-09-25-run-01/growth-moment-after-every-task-health-gain-streak-and-target.md`
**Inherits:** `harness/designs/frontend-shell.md` (palette, type, card shapes, Vietnamese register, "motion is spent in one place: the plant", reduced-motion floor). This document adds only what the shell did not draw: what the hub does in the two seconds after `POST /quests/progress` succeeds. Wire contracts are untouched — backend spec §6.2 (`pet_health`, `streak_count`, `is_target_met`) and §6.3 (`GET /pet/status`).
**Arithmetic it reflects:** CODEMAP `pet` — +20 health (capped at 100), streak+1 and the stage recompute happen **once per local day, on the task that crosses 30 minutes**. Earlier tasks change only `accumulated_seconds`. Stage by streak: wilted at 0 health, else 0–2 `sprout`, 3–6 `sapling`, 7–13 `flowering`, 14+ `fruitful`.

**Subject and job.** The learner has just spent ten minutes on a task and pressed "Hoàn thành". They land on the hub. The hub's job in this moment is to show, without a word of explanation, what those ten minutes did to the plant — and, on the task that completes the day, to make the day feel finished. It is a reward, not a report: short, once, then the hub is the hub again.

**Register.** Vietnamese, sentence case, the plant speaks in the first person ("tớ") as it already does. No exclamation inflation — one "!" per line at most. Nothing says "congratulations"; the plant growing *is* the congratulation.

---

## 1. The signature: the plant grows while you watch

The hub's first frame after a task is **yesterday's plant** — the health bar at the old value, the old stage, the old streak. Then, in one orchestrated sequence of about 1.2 seconds, the bar slides to the new value, a small chip rises beside the plant saying what changed, the plant itself grows (a stage cross-fade when the stage changed, a single "breath" upward when it did not), and the speech bubble says the new line. Two seconds after landing, the chips are gone and the page is the steady hub.

This is the one place motion is spent (shell §1 "Shape, spacing, motion"), and the structure encodes a true fact: the plant is the same plant, one frame older. Showing the *before* values for a frame is the deliberate risk — a frame of stale numbers, on purpose — because it is the only way the bar can visibly move; the store already holds the *after* values by the time the hub renders (`pet.load()` runs on mount).

What deliberately does **not** happen: no confetti, no full-screen sheet, no sound, no number ticker on the streak (6→7 is a one-digit change; a pulse says it), no toast. Under `prefers-reduced-motion: reduce` every keyframe and transition is off; the chips and the lines still render, the bar and plant snap to their final state.

---

## 2. Layout (mobile, `max-w-md`, unchanged card order)

```
┌──────────────────────────────────────┐
│ [A] Nguyen Hendrix    (🔥 Streak: 7 ngày) │  ← existing chip; one-shot pulse when streak rose
├──────────────────────────────────────┤
│ Plant hub card (relative)            │
│                        ┌──────────┐  │
│         (Cây ảo SVG)   │ +20 máu  │  │  ← GrowthChip, growth tone
│           🌿  ↑ grow   │ 🔥 7 ngày│  │  ← GrowthChip, streak tone
│                        └──────────┘  │     absolute, top-right of the card; no layout shift
│ Máu cây: [██████████░░] 80% → 100%   │  ← HealthBar slides from the before-value
│ 💬 "7 ngày liên tiếp! Cảm ơn bạn,    │  ← SpeechBubble, new line (text change, no motion)
│     hôm nay tớ đủ nước rồi 🌿"       │
├──────────────────────────────────────┤
│ TIẾN ĐỘ HÔM NAY: 30 / 30 PHÚT        │  ← existing; segments fill, "Mục tiêu hôm nay đã đạt ✓"
│ [███][███][███]                      │
├──────────────────────────────────────┤
│ Nhiệm vụ hôm nay (Quests) …          │  ← unchanged
└──────────────────────────────────────┘
```

The chips overlay the empty corner of the plant card (the SVG is 160 px centred in a 28 rem card) so their arrival and departure move nothing else. Two chips at most, stacked, health above streak. Everything else keeps its place and size; the moment adds no new row to the page.

### Timeline (from the hub's first paint after `navigateTo('/')`)

| t | What | Duration / easing |
| --- | --- | --- |
| 0 ms (first paint) | The hub renders the **before** values: bar at `healthFrom`, plant at `stageFrom`. Bubble already shows the new line. | one paint |
| next frame | The flip: bar slides `healthFrom` → `healthTo`; if the stage changed, the old stage group fades out and the new one fades in. | bar 400 ms ease-out; cross-fade 300 + 300 ms out-in |
| 150 ms | Chips enter: fade + 8 px rise. Header streak chip pulses if the streak rose. | chips 200 ms ease-out; pulse 500 ms |
| 400 ms | Same stage and target newly met → one-shot `grow` breath (scale 1 → 1.06 → 1 from the pot). Skipped when the stage changed — the cross-fade already is the growth. | grow 800 ms `cubic-bezier(.2,.8,.2,1)` |
| 1800 ms | Chips leave: fade. | 200 ms |
| 2000 ms | Steady hub. Nothing is left in the DOM that was not there before. | — |

Total ≤ 2 s. The plant does one thing (cross-fade *or* breath), and the chips arrive after the bar has started moving — one thing at a time is what makes a short sequence readable.

---

## 3. What fires, and when

| Situation (after `POST /quests/progress`) | Bar | Chips | Plant | Bubble | Header |
| --- | --- | --- | --- | --- | --- |
| Task 1 or 2 done, target not yet met (`pet_health`/`streak_count` unchanged) | no change | none | none (sway only) | partway line ("Còn 20 phút nữa thôi!") | — |
| Task crosses the target (`is_target_met` newly `true`; health +20, streak +1) | slides `healthFrom` → `healthTo` | "+20 máu", "🔥 7 ngày" | cross-fade if the stage changed, else grow breath | met line, with the milestone prefix on 3 / 7 / 14 / 21 / 28 | pulse |
| Target crossed with health already 100 | no change | "Máu đầy", "🔥 7 ngày" | grow breath | met line | pulse |
| Wilted plant completes the day (health 0 → 20, `wilted` → `sprout`) | slides 0 → 20 | "+20 máu", "🔥 1 ngày" | cross-fade droop → sprout | met line (no longer `…`); the revive banner leaves because `isWilted` is false | pulse |
| Response omits `pet_health`/`streak_count` (backend pet read failed — today's bug plan) | no change | none for the missing field | none | met/partway line by `is_target_met` | — |
| Plain load, reload, back-navigation, retry | steady | none | none | context line | — |

A moment is recorded only when something changed (health, streak, stage) or the target was newly met; the hub consumes it once, so a second visit to `/` in the same session shows the steady hub. Refresh = steady hub (the record lives in Pinia, not storage).

---

## 4. The bubble, made context-aware

`speechLine` keeps its shape (one string) and gains the inputs the hub already has. Priority top to bottom; the first match wins.

| # | Condition | Line |
| --- | --- | --- |
| 1 | `wilted` or health ≤ 0 | `…` (the revive banner speaks; unchanged) |
| 2 | target met, streak in {3, 7, 14, 21, 28} | "**7 ngày liên tiếp!** Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿" |
| 3 | target met | "Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿" (unchanged) |
| 4 | not met, `accumulated_seconds` > 0 | "Còn *n* phút nữa thôi!" — *n* = minutes left to 30, rounded up (20 after one task, 10 after two, 1 at 29:30) |
| 5 | not met, nothing today, `last_practiced_at` ≥ 2 whole days ago | "Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?" |
| 6 | not met, health ≥ 60 | "Tưới cho tớ 10 phút học đi!" (wireframe 7.2, unchanged) |
| 7 | not met, health 30–59 | "Tớ hơi khát rồi… 10 phút thôi?" (unchanged) |
| 8 | not met, health < 30 | "Tớ sắp héo mất! Học một chút nhé?" (unchanged) |

Row 4 outranks health because a learner who is mid-day is already watering; the tone of the bar says the rest. Row 5 uses `daysSince` (already in `utils/plant.ts`): 1 whole day means they practised yesterday (normal), 2+ means yesterday was missed. `last_practiced_at` is set only when a target is met, so a learner with partial days still gets row 4 when they are mid-day today. The milestone line persists for the rest of the day (it is the day's steady state, not a one-shot); the chips are the one-shot.

---

## 5. Components

| Component | Status | Change |
| --- | --- | --- |
| `components/plant/GrowthChip.vue` | **new** | `text`, `tone: 'growth' \| 'streak'`; pill `rounded-full px-3 py-1 text-sm font-semibold`, `bg-growth/15 text-growth` or `bg-streak/15 text-streak` (the header's streak chip recipe); `data-growth-chip="<tone>"` for tests |
| `components/plant/PlantSvg.vue` | modify | the stage groups sit inside a `<Transition name="stage" mode="out-in">` (`stage-enter/leave` = opacity, 300 ms, gated); new `grow?: boolean` prop adds a one-shot `plant-grow` class on the active group (800 ms keyframe, gated); `data-stage` and the `aria-label` unchanged |
| `components/ui/HealthBar.vue` | modify | `duration-300` → `duration-400` to match the shell's 400 ms; nothing else (it already animates width and honours `motion-reduce`) |
| `components/AppHeader.vue` | modify | `pulse?: boolean` prop adds a one-shot `streak-pulse` class (scale 1 → 1.12 → 1, 500 ms, gated) to the existing streak chip |
| `components/plant/SpeechBubble.vue` | unchanged | already `aria-live="polite"`; a new line is announced once |
| `composables/useGrowthMoment.ts` | **new** | owns the timeline: takes the consumed delta, exposes `displayHealth` / `displayStage` (before-values for the first paint, then the store's), `chips`, `grow`, `pulse`, `active`; flips phase on `requestAnimationFrame`, clears chips at 1800 ms, cleans up on unmount |
| `pages/index.vue` | modify | consumes the delta on mount, feeds the composable, renders the chips (`<TransitionGroup>`), passes `grow` / `pulse` / `displayHealth`, passes the extra bubble inputs |
| `stores/pet.ts` / `stores/quest.ts` / `utils/plant.ts` | modify | the record (`lastDelta`, `consumeDelta`), `targetMetChanged`, `speechLine` + `stageForStreak` — plan |

No new icon, no new font size, no new colour.

---

## 6. Tokens and motion (all existing tokens; nothing new registered)

- Colour: chips `growth` / `streak` at 15 % on the card surface with full-strength text — the same recipe as the header streak chip, so a chip reads as "a fact about the plant" not a notification. Bar tones unchanged (`healthTone`).
- Type: chips `text-sm font-semibold tabular-nums` (body face — they are data, not the plant's voice); bubble stays Fraunces.
- Shape: chips fully rounded, like the header chip. Chip stack `gap-1.5`, card corner `right-4 top-4`.
- Motion (every rule inside `@media (prefers-reduced-motion: no-preference)`; outside it the class is inert and the value is already final):
  - `health-bar` width: 400 ms ease-out (Tailwind `transition-[width] duration-400`; `motion-reduce:transition-none` stays).
  - `stage-enter-active` / `stage-leave-active`: opacity 300 ms ease; `mode="out-in"`.
  - `plant-grow`: 800 ms `cubic-bezier(.2,.8,.2,1)`, `transform-origin: 80px 110px` (the pot — same origin as `plant-sway`), keyframes 0 % scale(1) → 45 % scale(1.06) → 100 % scale(1). Runs once (`animation-iteration-count: 1`, class removed at the end so a later moment can re-fire).
  - chips `chip-enter`: opacity 0 → 1 and translateY(8px) → 0, 200 ms ease-out; `chip-leave`: opacity → 0, 200 ms.
  - `streak-pulse`: 500 ms, scale 1 → 1.12 → 1.
- Accessibility: chips container `aria-live="polite"` (announced once, after the bubble); the before-values exist for one paint only and the health meter is not a live region, so nothing stale is ever announced; after the flip the plant's `aria-label` and the meter read the final stage and health; focus order unchanged; nothing is focusable inside the moment.

---

## 7. Self-critique

Removed before writing the plan: a count-up on the streak number (a digit ticker for a one-digit change is noise), a "Ngày hoàn thành" full-width banner (competes with the wilted banner's slot and says what the bubble and the solid segments already say), a persistent "+20" badge on the bar (would outlive the moment and become decoration), and animating the segments again on the hub (they already ease on value change; a second choreography there would fight the plant). The risk kept is the one stale frame of before-values: it is invisible to assistive tech, lasts one paint, and is the whole mechanism by which "the plant grows while you watch" can be true. If the streak-shield idea lands later, its "Khiên mới!" is a third chip in the same stack, not a new device.
