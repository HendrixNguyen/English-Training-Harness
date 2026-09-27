---
type: bug
status: proposed
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
