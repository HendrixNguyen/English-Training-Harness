---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# v1Contrast source guard only scans class attributes, so script-level class maps can put text-white on a growth fill unnoticed

## Why
The test named for Acceptance (amend 2) item 3 (`frontend/tests/unit/v1Contrast.test.ts:71-90`) is meant to stop any component or page from drawing `text-white` on a `bg-growth`/`bg-alert` fill again. That pairing is 1.67:1 / 3.08:1 with the v2 `growth`. It extracts candidates with `content.match(/class="[^"]*"/g)` (`:82`), which sees static `class="…"` and `:class="…"` attributes but not class maps built in `<script setup>`. That is exactly how `AppButton.vue:16-17` and `RoadmapNode.vue:9-13` build theirs. With `text-white` restored in `AppButton`'s `primary` string, the guard still reports zero offenders (reviewer and validator both confirmed this). Today the per-component tests above it catch those two files, so nothing ships broken. A new component or a pending branch (roadmap-tree's `RoadmapNode` pill, growth-moment's `GrowthChip`) that uses the same computed-class idiom is unguarded, while the test title claims it covers everything.

## Expected output
- The guard scans each class-bearing string literal in `.vue` files (template attributes and single-quoted strings in `<script>`), or scans line by line like the plan's `grep -rnE 'bg-(growth|alert)[^/].*text-white'`. It fails on a same-string `bg-growth`/`bg-alert` + `text-white` pairing wherever the string lives.
- A negative self-check inside the test proves that the matcher flags a script-literal sample such as `"primary: 'bg-growth text-white'"`.
- While in the file, stub `NuxtLink` for the `AppHeader`/`QuestRow` mounts, as the `RoadmapNode` case already does, so the three `Failed to resolve component: NuxtLink` warnings go away.

## Evidence
- Plan `harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md` (Task 2 Step 1, the source guard); parent `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`.
- `frontend/tests/unit/v1Contrast.test.ts:82` at `2d9cab9`; `frontend/components/ui/AppButton.vue:16-17`; `frontend/components/roadmap/RoadmapNode.vue:11`.
- Review: `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n-3.md`.
