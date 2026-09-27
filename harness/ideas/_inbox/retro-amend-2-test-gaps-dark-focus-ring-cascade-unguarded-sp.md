---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Retro amend 2 test gaps: dark focus-ring cascade unguarded, SpeechBox flip-back case asserts no emit count

## Why
Two behaviours that amend 2 relies on are verified only by hand:
1. **Dark focus ring.** The dark `ring-growth` focus ring works only because the second `@media (prefers-color-scheme: dark) { :focus-visible { … } }` block comes *after* the base `:focus-visible` rule in `frontend/assets/css/main.css:14-29` (the executor's logged deviation). If a future edit moves it back into the `html` dark block, as design A9 literally says, the dark ring silently reverts to `growth-deep` (4.31:1 on white, but dimmer on the dark ground than designed). No unit test fails. The reviewer confirmed the dark ring in Playwright (`rgb(61, 225, 176)` in dark, `rgb(23, 138, 105)` in light), but nothing in CI pins it.
2. **SpeechBox flip-back.** The third SpeechBox live-flip case (`frontend/tests/unit/SpeechBox.test.ts:113-132`, flip `true` → `false` after the reveal, then a new line) checks only `[data-typed]`. It never checks the `settled` emit count, so a double emit on that path would pass. The second case (`:99-111`, flip after settle) passes even with the new `watch(isReduced…)` removed. It proves the `revealAll` guard, not the watch; only the first case (`:85-97`) proves the watch.

## Expected output
- A test pins the focus-ring cascade. For example, a Vitest case compiles `assets/css/main.css` with the project's Tailwind config through PostCSS and asserts that the last `:focus-visible` rule inside a `prefers-color-scheme: dark` media query sets the `growth` ring colour and comes after the base rule. Alternatively, it asserts the source order of the two blocks in `main.css`.
- The SpeechBox flip-back case asserts that `settled` is emitted exactly once for the first line and once more for the new line after it types out.

## Evidence
- Plan `harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md` (Notes, the "Deviation (CSS cascade)" paragraph; Task 3).
- `frontend/assets/css/main.css:14-29` and `frontend/tests/unit/SpeechBox.test.ts:99-132` at `2d9cab9`.
- Review: `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n-3.md`.
