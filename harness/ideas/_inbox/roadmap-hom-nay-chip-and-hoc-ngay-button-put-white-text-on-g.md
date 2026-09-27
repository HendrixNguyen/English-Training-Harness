---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Roadmap 'HÔM NAY' chip and 'Học ngay' button put white text on growth (2.5:1 on the branch, 1.7:1 on main's kit)

## Why
The roadmap tree adds two pieces of white text on a `growth` fill: the "HÔM NAY" chip (`frontend/components/roadmap/RoadmapNode.vue:33`, `bg-growth … text-xs text-white`) and the primary "Học ngay →" link (`:45`, `bg-growth … text-white`). Measured in a browser on the branch build: `rgb(255,255,255)` on `rgb(16,185,129)` (`#10B981`) ≈ 2.5:1. On `main` the retro kit (PR #50) moved `growth` to `#3DE1B0`, where white is ≈ 1.7:1, and deliberately replaced every `text-white` on growth/alert fills with `text-ground-0` (`grep -rn text-white frontend/components frontend/pages` on main → nothing). Merged as-is, the day's one call to action and the "today" label fail WCAG AA (4.5:1) — the UI kit's accessibility floor ("Body text ≥ 4.5:1"; "ground-0 on growth 11.7:1").

## Expected output
Both elements use `text-ground-0` on `bg-growth` (the kit's primary-button recipe), matching what main did to the old pill in `a0ec6ed`. Resolve this while merging the branch into the daily integration branch, or in an amend.

## Evidence
- Plan: `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md`, design `harness/designs/roadmap-tree.md` §5 ("HÔM NAY" chip `bg-growth text-white`) — the design predates kit v2.
- `git show f37d62b:frontend/components/roadmap/RoadmapNode.vue | grep -n text-white` → lines 33, 45; `git diff 67ad0c0 origin/main -- frontend/components/roadmap/RoadmapNode.vue` shows main's `text-white` → `text-ground-0` change on the same element this branch rewrote (the merge conflicts there).
- `frontend/tailwind.config.ts` on main: `growth: '#3DE1B0'`. Browser check (reviewer rv-frontend): computed chip/CTA colour 255,255,255 on 16,185,129.
