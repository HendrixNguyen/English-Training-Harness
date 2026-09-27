---
plan: harness/plans/2026-09-25-growth-moment-after-every-task-health-gain-streak-and-target.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/hub-bubble-tells-a-learner-who-practised-yesterday-hom-qua-t.md, harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md]
---
# Review — Growth moment after every task: health gain, streak and target-met celebration on the hub

**Plan:** `harness/plans/2026-09-25-growth-moment-after-every-task-health-gain-streak-and-target.md`
**Branch/worktree:** `harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target` / `.worktrees/growth-moment-after-every-task-health-gain-streak-and-target`
**Diff:** `git diff main...harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target --stat`

## Plan vs idea
Delivered. §5.2 step 5 now exists: after `POST /quests/progress` the hub paints the before-values, flips to the after-values, shows "+20 máu" / "🔥 n ngày" (or "Máu đầy"), cross-fades the stage or takes one grow breath, pulses the header streak, and speaks a context line; nothing fires on a plain load, on an unmet task, or twice. Every keyframe is behind `prefers-reduced-motion: no-preference`. One design row is wrong in the result: the missed-day line (§4 row 5) fires for learners who practised yesterday (bug, medium).

## Code vs plan
Branch head `a2d42fc` (base `67ad0c0`); CI run 36230372009 green on that head.
- Task 1 (`utils/plant.ts`): followed, with deviation 1 (`daysSince >= 1` instead of `>= 2`) — **not justified**: it swaps one calendar error for a worse, more frequent one (bug filed).
- Task 2 (`quest.complete` → `targetMetChanged`): followed.
- Task 3 (`applyProgress` delta): followed; deviation 2 (recompute stage only when health/streak moved) is justified — an unconditional recompute flips a stored stage and records a spurious delta on a no-change call.
- Task 4 (components): followed; `type="transition"` added to the stage `<Transition>` — justified and important (the infinite `plant-sway` animation otherwise makes `out-in` wait for an `animationend` that never fires; the executor saw the plant vanish live).
- Task 5 (composable + hub): followed; `start(pet.consumeDelta())` moved from `onMounted` to setup — justified (the first render must already show the before-values).
- Task 6 (CODEMAP): followed; accurate.

Re-run in a detached worktree at the branch head:
```
npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
LINT=0  TC=0  Test Files 19 passed (19) / Tests 118 passed (118)  BUILD=0
find tests/unit -type f | wc -l -> 20
grep -c consumeDelta stores/pet.ts -> 2 ; grep -c targetMetChanged stores/quest.ts -> 2
grep -c 'pet.applyProgress(res)' pages/learn/[id].vue -> 1 (untouched) ; grep -c STREAK_MILESTONES utils/plant.ts -> 2
grep -c duration-300 components/ui/HealthBar.vue -> 0
grep -c 'prefers-reduced-motion: no-preference' PlantSvg.vue / AppHeader.vue / pages/index.vue -> 1 / 1 / 1
grep -c @media composables/useGrowthMoment.ts -> 0 ; requestAnimationFrame -> 1 ; GrowthChip used in pages/components -> 1
grep -c 'growth moment' harness/CODEMAP.md -> 1 ; harness/ diff outside CODEMAP -> none ; cli.py validate -> 0
```
Merged onto current `origin/main` (27f57e1): only `harness/CODEMAP.md` conflicts; resolved, lint 0, typecheck 0, `test:unit` 38 files / 299 tests passed.

**Screen walk (design `harness/designs/growth-moment.md` §2/§3).** Branch `.output` served on :3105 against a stub API (health 80, streak 5, sprout, 20/30 min). Hub before the task: bubble "Còn 10 phút nữa thôi!" (row 4) ✓. Completed `/learn/ex-3` through the page's own button; the hub was sampled every 40 ms: t≈130 ms `/` renders bar 80 % and stage sprout (the before-values) ✓; t≈310 ms chips "+20 máu", "🔥 6 ngày" and header `streak-pulse` ✓; bubble "Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿" ✓. The browser pane was hidden, so `requestAnimationFrame` never fired and the flip only happened at the 2000 ms fallback (bar → 100 %); the cross-fade/breath and the CSS transitions could not be observed here and rest on the executor's live proof plus `useGrowthMoment.test.ts`/`indexPage.test.ts`. Useful side-finding: with rAF paused (tab hidden while the hub mounts) the before-values persist to 2 s and the moment resumes on return — acceptable, not filed. Reduced motion: every `animation`/`transition` added is inside the no-preference block (greps above); DOM is identical, as the `indexPage` "final values" case pins.

## Quality
- Boundaries: stores stay in their lanes; `stageForStreak` duplicates the backend table knowingly (plan note), corrected by `GET /pet/status` on the hub.
- Correctness: missed-day row (bug). Test-gap pass (read-only): the `applyProgress` no-status guard and a double `start()` on one composable instance are untested; the second is unreachable in the app (one `start` per hub mount, timers cleared on unmount) — not filed.
- Tests: honest (four mutation checks recorded and consistent with the tests read); `@vue/test-utils` stubs `<Transition>`, which is why the `type="transition"` defect needed a live browser — worth remembering for the retro migration.
- UI kit: built on kit v1 per its design; `harness/UI-KIT.md` v2 on main forbids the `rounded-full` chips, the eased `scale(1.06)` grow breath and the smooth streak pulse (one consolidated low bug across today's three design branches; the retro hub plan should absorb it).
- Merge note: `utils/plant.ts` `speechLine` and `pages/index.vue` conflict with the plant-name branch (both widen `speechLine`); the resolution must keep the milestone prefix **and** the plant name in the met line, and the growth `bubble` inputs plus `name`. `stores/quest.ts` conflicts with the timer branch inside `complete()` (keep the timer's `writeTimers` line and this branch's `wasMet`/return).

## Bugs filed
- `harness/ideas/_inbox/hub-bubble-tells-a-learner-who-practised-yesterday-hom-qua-t.md` (medium) — missed-day line on elapsed hours.
- `harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md` (low, shared with plant-name and roadmap-tree) — kit v2 drift.

## Verdict
**pass-with-bugs.** Delivered, verified, no blocker; merge with care for the `speechLine`/`index.vue` conflict with the plant-name branch.
