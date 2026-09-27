---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Task timer credits idle wall-clock time: one 10-minute task left open posts up to 60 minutes and can meet the day alone

## Why
The timer fix swapped under-counting for over-counting. `elapsedSeconds` is now `now − startedAt` from the learner's **first open** of the task (`frontend/stores/quest.ts:129-132`), persisted all day in `aelp.timers`, and `/learn/:id` posts it (`pages/learn/[id].vue:66`, clamped only to 3600 by `clampDuration`). Before the branch the posted value could not exceed the task's `totalSeconds` (`remainingSeconds` floored at 0). So a learner who opens a 10-minute task at 08:00, leaves, and taps "Hết giờ — Hoàn thành" at 17:00 posts `duration_seconds: 3600` — one task meets the 30-minute day (and earns +20 health, streak+1 via `OnTargetMet`) without 30 minutes of study. The backend does not cap per exercise (`backend/internal/quests/service.go:78` only checks `1..MaxDurationSeconds` = 3600). The metric the product is judged on (≥ 30 min/day, 1st-thinking §1) and the plant's health now reward an abandoned tab. The idea asked for "the real elapsed clamped to 3600", so this is a plan/idea-level decision, but its consequence on the happy path was not weighed.

## Expected output
- Pick one rule and pin it with a test: e.g. post `min(elapsed, totalSeconds + grace)` (a task can credit at most its own planned length plus a small grace), or pause the anchor while the page is hidden (`visibilitychange` accumulates visible time only), or both.
- A learner who opens a task and returns hours later cannot meet the day's target with that single task.
- `questStore.test.ts` case: anchor at T0, complete at T0 + 9 h → posted `duration_seconds` ≤ the chosen cap, and the day is not met by one task.

## Evidence
- Plan: `harness/plans/2026-09-25-task-timer-keeps-counting-through-reloads-and-background-tab.md` (design decision 4; branch head `17205de`).
- Code: `frontend/stores/quest.ts:129-132`, `frontend/pages/learn/[id].vue:66`, `frontend/utils/progress.ts` `clampDuration` (1..3600), `backend/internal/quests/service.go:78`.
- The plan's own test `the countdown floors at 0 for the button gate while the posted duration is the real elapsed clamped to 3600` posts 3600 s for a 10-minute task (≈ 83 min after open) — the behaviour is pinned, not accidental.
- The executor's runtime proof posted `duration_seconds: 917` for a 3-minute task after rewriting `startedAt` 15 min back.
