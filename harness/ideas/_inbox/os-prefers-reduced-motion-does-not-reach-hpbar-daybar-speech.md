---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md
---
# OS prefers-reduced-motion does not reach HpBar, DayBar, SpeechBox typing or Chest; only the reduced prop does

## Why
Acceptance item 7 says: "under `prefers-reduced-motion` in the browser no `retro-*` animation runs … typing text, chest and toast appear at once". The CSS block only matches elements that carry a `retro-` class.
- `HpBar`/`DayBar` put their `transition: width 300ms steps(n)` inline on an element with no `retro-` class. In the browser with reduced motion emulated, the computed `transition-duration` was still `0.3s`.
- `SpeechBox` typing (a 30 ms `setInterval`) and the `Chest` 200 ms reveal are JavaScript. They look only at the `reduced` prop (code reading), and no composable maps the media query to that prop. Pages would each have to wire this themselves, and plans 2–6 have no instruction to do it.

## Expected output
Choose one of:
- Give the bar fills a `retro-anim` class so the existing CSS block snaps them.
- Add a small `useReducedMotion()` composable (`matchMedia`, client-only) and make it the default for every kit component's `reduced` prop.

Tests pin the default.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/retro/HpBar.vue:33,49`, `DayBar.vue:32,55`, `SpeechBox.vue:31-44`, `Chest.vue:28-36`, `frontend/assets/css/retro.css:77-84`.
- Browser: `[data-fill]` computed `transitionDuration: 0.3s` with `matchMedia('(prefers-reduced-motion: reduce)').matches === true`.

## Evaluation
**Verdict: select, medium; folded.** The *Why* is real. It fails acceptance item 7, and pages would otherwise each have to wire the media query themselves. **Root cause:** the `retro.css:77-84` block matches only `retro-` classes. The bars' inline `transition` (`HpBar.vue:33`, `DayBar.vue:32`) and the JS timers (`SpeechBox.vue:31-44`, `Chest.vue:28-36`) read only the `reduced` prop. The fix is design addendum A4: a `useReducedMotion()` composable becomes the default of every `reduced` prop (with the `withDefaults({ reduced: undefined })` trap called out), and the bar fills get `retro-anim`. Folded into the retro kit amend plan `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (it amends `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, the same branch). Task 4.
