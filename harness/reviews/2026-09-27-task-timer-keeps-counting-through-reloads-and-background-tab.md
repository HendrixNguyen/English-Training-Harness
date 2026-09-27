---
plan: harness/plans/2026-09-25-task-timer-keeps-counting-through-reloads-and-background-tab.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/task-timer-credits-idle-wall-clock-time-one-10-minute-task-l.md, harness/ideas/_inbox/learnpage-test-finds-the-completion-button-by-its-mt-4-spaci.md]
---
# Review — Task timer keeps counting through reloads and background tabs so studied minutes are never lost

**Plan:** `harness/plans/2026-09-25-task-timer-keeps-counting-through-reloads-and-background-tab.md`
**Branch/worktree:** `harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` / `.worktrees/task-timer-keeps-counting-through-reloads-and-background-tab`
**Diff:** `git diff main...harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab --stat`

## Plan vs idea
Delivered. Every Expected-output line exists: `Timer {startedAt, totalSeconds, date}` computed from a `now`, persisted at `localStorage['aelp.timers']`, hydrated in `state()` and pruned in `load()` (other day, not in today's list, completed), `tick` only re-renders, `/learn/:id` re-syncs on `visibilitychange`/`focus`, the wire unchanged (`clampDuration` 1..3600). One consequence the idea asked for but nobody weighed: posting the *real* wall-clock elapsed since first open means idle time is credited — a 10-minute task opened in the morning and completed in the evening posts 3600 s and meets the day alone (bug filed, medium). Before the branch the posted value could not exceed the task's own length.

## Code vs plan
Branch head `17205de` (base `959cb5f`); CI run 36106803353 green on that head (`gh run list --branch … --limit 1` → completed success 17205de).
- Task 1 (store): followed. Deviation (justified): `elapsedSeconds`/`remainingSeconds` take `now?` and resolve `now ?? this.nowMs` in the body (a `this` default parameter does not typecheck in an options-API action).
- Task 2 (page + `learnPage.test.ts`): followed; folded into the Task 1 commit as the plan allowed. Test deviation: the page button is found by `button.mt-4` (bug filed, low).
- Task 3 (CODEMAP): followed, wording matches the code.

Re-run in a detached worktree at the branch head:
```
npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
LINT=0  TC=0  Test Files 17 passed (17) / Tests 85 passed (85)  BUILD=0
grep -c '^  it(' tests/unit/questStore.test.ts   -> 10
grep -c '^  it(' tests/unit/learnPage.test.ts    -> 2
grep -n "aelp.timers" stores/quest.ts            -> 2 lines (44 TIMER_STORAGE_KEY, 117 the plan-prescribed doc comment; the plan's "1" was its own miscount)
grep -c 'remainingSeconds' stores/quest.ts       -> 1
grep -n 'Date.now()' stores/quest.ts             -> 85, 118, 126
grep -n 'setInterval\|visibilitychange\|focus' pages/learn/[id].vue -> 6 lines (23, 34, 37, 38, 45, 46)
git diff --stat 959cb5f..HEAD -- backend/ frontend/stores/pet.ts frontend/utils/progress.ts -> no output
python3 tools/harness/cli.py validate -> exit 0
```
Merged onto current `origin/main` (27f57e1, includes the retro kit) in the scratch worktree: only `harness/CODEMAP.md` conflicts; with that resolved, `npm run typecheck` 0 and `npm run test:unit` → 36 files / 270 tests passed. The executor's runtime proof (real browser, reload keeps the anchor, 15-minutes-ago anchor renders 00:00 + "Hết giờ — Hoàn thành", post carried the real elapsed) is consistent with the code and the tests; not re-driven in a browser here (the unit/page tests exercise the same paths with a faked clock and `visibilitychange`).

## Quality
- Design: the anchor-not-counter change is the right fix and copies the `stores/pet.ts` revive pattern (`storageOrNull` + `try/catch`); storage failure degrades to in-memory. `pruneTimers()` rewrites storage on every `load()` even when nothing changed — harmless (one small `setItem`).
- Correctness: idle wall-clock time is credited up to 3600 s per task (bug, medium). Clock skew/multi-tab are noted in the plan and bounded by `clampDuration`.
- Tests: honest — the twenty-tick assertion, the simulated reload via a fresh Pinia, the storage-throws stub and the `visibilitychange`-without-interval page case would each fail without the fix (executor mutation checks recorded). Only the `button.mt-4` selector is brittle.
- Kit: the plan has no `design:`; for the record, `harness/UI-KIT.md` v2 (now on main) says the timer "measures, never gates the button" — this branch keeps the v1 gate, which `harness/designs/retro-learning-room.md` already plans to remove. Not filed separately.
- CODEMAP `shell` clause is accurate. Merge note for the daily PR: `frontend/stores/quest.ts` conflicts with the growth-moment branch (`complete()` tail) and `harness/CODEMAP.md` conflicts with main and every other branch reviewed today — resolve by keeping both hunks.

## Bugs filed
- `harness/ideas/_inbox/task-timer-credits-idle-wall-clock-time-one-10-minute-task-l.md` (medium) — idle time since first open is posted as study time, up to 3600 s per task.
- `harness/ideas/_inbox/learnpage-test-finds-the-completion-button-by-its-mt-4-spaci.md` (low) — brittle selector in `learnPage.test.ts`.

## Verdict
**pass-with-bugs.** The idea is delivered and verified; no blocker. Mergeable into the daily branch (resolve the CODEMAP and `stores/quest.ts` conflicts).
