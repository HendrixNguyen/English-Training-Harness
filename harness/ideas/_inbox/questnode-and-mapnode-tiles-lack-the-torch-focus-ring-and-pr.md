---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md
---
# QuestNode and MapNode tiles lack the torch focus ring and pressed inset, and MapNode has no 44 px hit area

## Why
Design §3 says "every focusable retro element sets `focus-visible:outline … outline-torch`" and asks for a pressed `:active` inset (border to `line-dim`, translateY 2 px) on tiles. Acceptance item 4 asks for a MapNode hit area of at least 44 px (`p-0.5` on the wrapper).
- The `QuestNode` and `MapNode` `<button>`s have none of these. Keyboard users get the v1 `ring-growth` rule instead.
- A MapNode measures exactly 40×40 in the browser, below the kit's 44 px target floor.
- In `locked`, the `QuestNode` task icon stays `line-lit`, although "everything `text-ink-2`" is the spec.

## Expected output
Both tiles carry the torch focus outline and the `:active` inset. MapNode sits in a `p-0.5` (≥ 44 px) hit wrapper. The locked QuestNode icon is dimmed. Tests assert the classes and the wrapper size.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/retro/QuestNode.vue:50-58`, `MapNode.vue:48-51`.
- Browser: MapNode `getBoundingClientRect` 40×40. QuestNode buttons have no torch outline class.

## Evaluation
**Verdict: select, medium; folded.** The *Why* is real. Keyboard focus is off-kit, and the MapNode is 40×40, below the kit's 44 px floor (acceptance item 4 fails). **Root cause:** the tile `<button>`s (`QuestNode.vue:50-57`, `MapNode.vue:48-51`) carry no focus, active or hit-wrapper classes, and the locked icon palette is not dimmed. The fix is design addendum A5: the torch outline plus `ring-0` to cancel the v1 global ring, the `active:`/`group-active:` inset bound only when the tile can act, and the MapNode `<button class="group relative p-0.5">` around a 40×40 `[data-tile]`. The locked icon is handled by A3. Folded into the retro kit amend plan `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (it amends `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, the same branch). Tasks 1 and 3.
