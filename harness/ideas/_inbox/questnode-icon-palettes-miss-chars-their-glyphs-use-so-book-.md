---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# QuestNode icon palettes miss chars their glyphs use, so book, scroll and sword pixels render solid black

## Why
`QuestNode` passes `hexPalette('k','l')` to the task icon. The glyphs use more chars than that: `book` uses `i`, `scroll` uses `d` and `i`, and `sword` uses `T`. Their fills are `var(--px-<char>)`, and a missing variable resolves to black. In the browser 48 rects rendered `rgb(0,0,0)`, all of them in the quest tiles. The hub path (plan 2) is the screen a learner sees every day. `Chest`, `Badge`, `StateBlock`, `MapNode` and the star/padlock overlays hand-pick their palettes the same way and are one glyph edit away from the same bug. `QuestNode.test.ts` never checks that a palette covers its glyph.

## Expected output
Every `PixelArt` caller passes a palette that covers every char its rows use. Two options: build the palette from `PALETTE` with per-caller overrides, as `CompanionSprite`/`Badge` do, or have `PixelArt` fall back to the `PALETTE` token for any char not passed. A unit test mounts each component in each state and fails if a `<rect>` fill references an undefined variable.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/retro/QuestNode.vue:58`. Glyph chars come from `frontend/utils/pixelArt.ts` (`GLYPHS.book/scroll/sword`).
- Browser: `svg.retro-pixel rect` with computed `fill: rgb(0, 0, 0)`: 48, all inside `#quests button > svg`. Screenshot `harness/reviews/retro-kit-screens/kit-mobile-375.png` (quest tiles).
