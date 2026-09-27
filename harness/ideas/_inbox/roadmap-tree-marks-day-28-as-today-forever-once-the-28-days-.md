---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Roadmap tree marks day 28 as today forever once the 28 days are over (DayNumber clamps)

## Why
`quests.DayNumber` clamps to 1..28 (`backend/internal/quests/day.go:52-72`, "a learner past day 28 keeps seeing day 28"). The roadmap tree feeds that number straight into `dayState` (`frontend/utils/roadmap.ts`), whose first rule is `day_number === todayNumber → 'today'`. So from day 29 on, the tree shows day 28 as "HÔM NAY · Đang học · n/30 phút" (or "Đã đủ 30 phút") indefinitely, scrolls to it and expands it — day 28 can never read "Đã hoàn thành" or "Bỏ lỡ", and the "Học ngay →" button points at a day that is over. The `/roadmap` outline itself knows the real dates (`DayDate`), so the page could tell. The day-28 checkpoint plan (`harness/plans/2026-09-26-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md`) may absorb this; until then it is a wrong state on the one page whose job is to be the honest record.
Test gaps found alongside (reviewer + test-gap pass): `TestRoadmapWithTheWrongShapeFails` covers only `modules: []`; the per-module `len(m.Days) != 7` branch (`roadmap.go` in `Service.Roadmap`) is untested; no page/util test covers `day_number` 28 on a finished roadmap.

## Expected output
- `GET /api/v1/roadmap` (or the client, from `date` vs the learner's local date) distinguishes "day 28 is today" from "the roadmap is over": after day 28, no row is `today`; day 28 is `completed`/`partial`/`missed` by its own progress, and the header can say the plan is finished.
- Tests: an outline whose `created_at` is 30 days ago → day 28 is not `today`; a stored document with a 6-day module → 500 naming `days`.

## Evidence
- Plan: `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md` (branch head `f37d62b`); design `harness/designs/roadmap-tree.md` §4.1.
- `backend/internal/quests/day.go:52-72` (clamp), `frontend/utils/roadmap.ts` `dayState` (today first).
