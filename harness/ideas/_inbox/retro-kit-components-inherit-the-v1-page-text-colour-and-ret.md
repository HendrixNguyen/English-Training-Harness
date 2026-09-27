---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md
---
# Retro kit components inherit the v1 page text colour and RetroToast has no fill, so panel, toast and map-node text is dark-on-dark

## Why
Some kit components never set a text colour on their own content, so the content inherits the v1 `html` colour. That is `ink` #1E293B in the light scheme, which is dark text on the kit's dark grounds. Plans 2–6 will compose these components on every screen.
- `RetroPanel` default slot: rendered content has `color: rgb(30, 41, 59)` on `ground-1`. In the kit screenshot "Plain panel text" and "Ember panel" can barely be seen.
- `RetroToast` renders its panel with `band`, and `band` drops the border and the `bg-ground-1` fill. The toast has a transparent background, no line, and dark text floating over the page. The design says the toast is a one-line `RetroPanel tone` with a fill and a ring.
- `MapNode` day number: the `ink` colour on `ground-1` is about 1.3:1. The `partial` growth fill is a later absolutely positioned sibling, so it paints over the lower half of the number.

## Expected output
`RetroPanel` sets `text-ink-0`. The toast has a `ground-1` fill and the 2+2 px ring (do not use `band`). `MapNode` sets `text-ink-0`, or `text-ink-2` when locked, and keeps the number above the partial fill. A kit test asserts each of these colour classes.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/retro/RetroPanel.vue:30`, `RetroToast.vue:18` (`band`), `MapNode.vue:50,58,68`.
- Browser: toast panel computed `background rgba(0,0,0,0)`, `border 0px`, `color rgb(30,41,59)`. MapNode buttons `color rgb(30,41,59)`. Screenshot `harness/reviews/retro-kit-screens/kit-mobile-375.png` (panels, map row).

## Evaluation
**Verdict: select, medium; folded.** The *Why* is real. Plans 2–6 compose `RetroPanel`, `RetroToast` and `MapNode` on every screen, and the text colour is inherited dark-on-dark. It is the same root cause as the blocker: kit components rely on the `html` colour. `RetroPanel.vue:30` sets no `text-*`, and `band` drops the fill and the border. `RetroToast.vue:18` uses `band`. In `MapNode.vue:58,68` the `h-1/2` fill overpaints the number. Design addendum A2 fixes it: every surface sets `bg-ground-1 text-ink-0`, `band` keeps the fill and the ring, the toast drops `band`, and the partial band is `h-2` with the number at `z-10`. Folded into the retro kit amend plan `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (it amends `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, the same branch). Tasks 2–3.
