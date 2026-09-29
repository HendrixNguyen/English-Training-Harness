---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# CompanionSprite motion is off the pixel grid at every size but 128 px, levelup is a CSS filter not a palette swap, and down lacks its translate

## Why
`--px` is fixed at `4` on `:root`, so `retro-breath`/`retro-hop`/`retro-shake` always move 4/8 CSS px. At 32 px (×1) that is 4 sprite pixels. At the 48 px face crop (×3) and 64 px (×2) it is off the grid. Design §3 says transforms stay on the pixel grid, and the kit says "never scale a sprite by 1.06".

Other gaps against design §4:
- `retro-flash` uses `filter: brightness()/saturate()/grayscale()`, where the design asks for a palette swap (all non-`k` chars to torch, then ink-0, then base).
- `down` rotates the wrapper `div` 90° without the `translate(0 7)`.
- Logged deviations 1 and 3: wilted keeps the open-eyed face, and reactions move the pot too.

## Expected output
`CompanionSprite` sets `--px` to its scale (`size/32`, or `size/16` for the face crop). Levelup swaps `--px-*` variables in three steps. `down` matches the design transform. The evaluator decides whether deviations 1 and 3 stand. The design's own test would need a wilted exception for rows 26–29.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`) (Execution summary deviations 1, 3). `frontend/assets/css/retro.css:9-11,33-38`. `frontend/components/retro/CompanionSprite.vue:56`.

## Evaluation
**Verdict: select, low.** The *Why* is real but it is fidelity, not function. The off-grid 4 px step at 32, 48 and 64 px and the `filter` flash are visible only to a careful eye. **Root cause:** `retro.css:9-11` has a global `--px: 4`, `retro-flash` is a `filter` keyframe, and the `down` wrapper only rotates (`CompanionSprite.vue:56`). **Fix scope:** set `--px` per sprite (`size/32`, or `size/16` for the face crop); make levelup a stepped `--px-*` palette swap (reuse the torch override the amend adds for the reduced hold); add `translate(0 7)` to `down`. **Deviations 1 and 3 stand.** For the wilted face (1), health tint, stem and rotation already carry the signal, and changing it needs a row-26–29 exception in the pinned test. For the pot moving with the plant (3), a two-layer sprite is disproportionate to a low bug. Record both in `harness/designs/retro-kit.md` when planned. **Plan deferred, not folded (evaluator, 2026-09-27):** today's bug list already holds five plans (the 5/day cap), and only the blocker amend jumps it. Every file this bug touches exists only on the unmerged retro branch, and today's amend (`harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`) rewrites them. So write the plan against `main` in the first bugfix run after the retro branch merges, before plans 2–6 compose the kit. Selected-but-unplanned bugs rank first in that run.
