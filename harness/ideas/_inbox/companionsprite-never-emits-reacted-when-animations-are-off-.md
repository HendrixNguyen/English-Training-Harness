---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# CompanionSprite never emits reacted when animations are off, and reduced motion has no static reaction frames

## Why
The design says `hit`/`miss`/`levelup` emit `reacted` "then return to idle". The emit only fires on `animationend`. Two cases never produce that event:
- With `reduced`, no class is applied at all.
- Under OS `prefers-reduced-motion`, `retro.css` sets `animation: none`.

In the browser with reduced motion emulated, clicking "hit" left `data-react="hit"` and emitted nothing. A learning room (plan 3) that waits for `reacted` would hold the sprite in `hit` forever for every reduced-motion learner.

The design's reduced variants are also missing: hit should raise the plant 2 units for 300 ms, and levelup should show the torch palette for 300 ms. Today the reduced sprite just stays at base, which breaks the acceptance item "every reaction still shows its static frame". The existing test (`CompanionSprite.test.ts:29`) is named "shows a static frame" but asserts only that no animation class is present and that `data-frame` is 0.

## Expected output
Under `reduced`, or when the animation does not run, the transient reactions show their static frame for 300 ms and then emit `reacted`, using a timer that is cleared on unmount. Tests use fake timers to assert both the frame (`data-frame` or a palette variable) and the emit.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/retro/CompanionSprite.vue:50-54, 64-66`. `frontend/assets/css/retro.css:77-84`. `frontend/tests/unit/CompanionSprite.test.ts:29-34`.
- Browser (Playwright `reducedMotion: reduce`): after the hit click and an 800 ms wait, `reacted` was empty and `data-react` was `hit`.
