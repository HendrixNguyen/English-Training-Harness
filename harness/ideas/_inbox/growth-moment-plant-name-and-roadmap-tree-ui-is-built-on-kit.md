---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Growth moment, plant-name and roadmap-tree UI is built on kit v1; kit v2 on main forbids its radii and eased scale motion

## Why
`harness/UI-KIT.md` on `main` is kit v2 (retro, merged with PR #50 on 2026-09-27) and "supersedes kit v1 … for visual style". Three of today's feature branches were designed and built on kit v1 (their designs predate v2) and add new v1-style UI that v2 forbids:
- **growth moment** (`a2d42fc`): `GrowthChip` `rounded-full` pills (`components/plant/GrowthChip.vue:8`); the grow breath is a `scale(1.06)` tween with `cubic-bezier(.2,.8,.2,1)` easing (`components/plant/PlantSvg.vue:106,124`) — v2: "Radius 0–2 px … nothing rounder, ever", "All UI motion is stepped … No easing curves, no scale tweens", "Never scale a sprite by 1.06"; header `streak-pulse` is also a smooth scale.
- **plant name** (`55650d5`): the caption is `font-display` (v1 Fraunces; v2 reserves display for VT323 with its own size table) and the field is `rounded-btn` 12 px.
- **roadmap tree** (`f37d62b`): `rounded-full` markers and chip, `font-display text-[28px]`, `AppCard` panels, v1 tokens (`streak`, `mute`, `ink`/`paper`); v2 specifies `MapNode` for this screen.
None of this breaks on merge (v1 tokens are aliased), and the retro screen designs (`harness/designs/retro-hub.md`, `retro-roadmap.md`, `retro-onboarding.md`) already plan to redo these screens — but each new v1 surface is more to migrate, and the growth moment's motion contradicts v2's motion budget outright.

## Expected output
When the retro hub / roadmap / onboarding plans are written, they explicitly cover: GrowthChip → a 2-px-radius torch/growth chip or `RetroToast`; the grow breath → `CompanionSprite` `levelup` (stepped, 3 frames); streak pulse → stepped; the plant-name caption → `SpeechBox` `speaker=plant_name` / VT323 name tab; the roadmap spine → `MapNode` states (cleared/today/partial/missed/locked map 1:1 to this branch's `dayState`). The evaluator should fold this into those plans rather than plan it separately.

## Evidence
- Plans: `harness/plans/2026-09-25-growth-moment-after-every-task-health-gain-streak-and-target.md`, `harness/plans/2026-09-25-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md`, `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md` (the plant-name executor noted the v1/v2 gap in its summary).
- `harness/UI-KIT.md` "Grid, shape, depth", "Motion budget", "Components" (`MapNode`, `CompanionSprite`, `SpeechBox`).
