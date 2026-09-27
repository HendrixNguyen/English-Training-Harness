---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md
---
# Roadmap page never refetches: today goes stale and a new learner keeps seeing 'no roadmap' until a full reload

## Why
The roadmap tree's whole job is to be the honest record (design `harness/designs/roadmap-tree.md` §1), and the plan's decision 4 says the outline is "refetched on page open". On the branch it is fetched once per page lifetime: `frontend/pages/roadmap.vue:17` loads only `if (!roadmap.outline && !roadmap.noRoadmap)`, and nothing else ever calls `useRoadmapStore().load()` (the hub reloads `quest`/`pet` on every mount, never `roadmap`). Two happy-path regressions against `main` (whose `/roadmap` read `quest.daily`, which the hub refreshes):
- **A new learner is told they have no roadmap after they made one.** `/roadmap` with no roadmap → `noRoadmap = true` → "Tạo lộ trình 28 ngày" → `/onboarding` → SPA back to the app → `/roadmap` still renders "Bạn chưa có lộ trình học." until a hard reload.
- **Today and minutes go stale in an open PWA.** Open `/roadmap`, study, come back (or leave the PWA open across midnight): today's row still reads the old minutes, and after the server's `day_number` moves on, the previous day is still marked `today` (and the new today `locked`).

## Expected output
- Every visit to `/roadmap` calls `roadmap.load()` (keep showing the cached `outline` while it loads, per the page-state convention: `StateBlock loading` only when there is no outline yet), so `noRoadmap`, `day_number`, `minutes_spent` and `is_target_met` are the server's current answer.
- Scroll-to-today and the default-expanded row follow the freshly loaded `day_number`.
- A unit test in `tests/unit/roadmapPage.test.ts`: mount with a store that already holds an outline (day 9) / `noRoadmap: true`, make `api.get('/api/v1/roadmap')` answer day 10 / an outline, and assert the second mount calls the API and renders day 10 as `aria-current="step"` / the tree.

## Evidence
- Plan: `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md` (decision 4: "refetched on page open"); branch head `f37d62b`.
- Code: `frontend/pages/roadmap.vue:17` `if (!roadmap.outline && !roadmap.noRoadmap) await roadmap.load()`; `grep -rn "roadmap.load\|useRoadmapStore" frontend/pages frontend/components` → only `pages/roadmap.vue`.
- Reproduced in a real browser (reviewer rv-frontend, 2026-09-27) against the branch's built `.output` and a stub API: (1) loaded `/roadmap` at server `day_number` 9, switched the stub to day 10, `router.push('/')` then `router.push('/roadmap')` → `aria-current="step"` still on `day-9`, and the stub logged no second `GET /api/v1/roadmap`; (2) stub answered `404 no_active_roadmap`, full load of `/roadmap` → empty state; stub switched to an active roadmap, `router.push('/')` (hub loaded quests fine) then `router.push('/roadmap')` → still "Bạn chưa có lộ trình học.".
- Blocker because it is a regression the branch introduces on the new-learner path and contradicts the plan; the fix is one condition plus a test.
