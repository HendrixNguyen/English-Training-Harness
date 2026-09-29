---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Retro kit small contract gaps: HpBar shows unclamped values, bar steps use total not delta, band panels keep their ring, reduced QuestNode drops its cursor, MapNode missed ring is not ember

## Why
Small divergences from design §5 that plans 2–6 will inherit:
- `HpBar` label and `aria-valuenow` print the raw `value`. A stale or negative health shows "HP -5/100" or "HP 130/100" while the fill is clamped.
- `HpBar`/`DayBar` use `steps(lit)` (the total) where the design says `steps(|Δlit|)`. A 1-cell change animates in 20 steps.
- `DayBar` `role="progressbar"` has no accessible name.
- `RetroPanel band` still applies the outer ring, although the design says "no outer line".
- `QuestNode` with `reduced` removes the `▶` cursor from `current`. The reduced variant should keep a static cursor ("designed, not disabled"), and the cursor is the state glyph.
- The `MapNode missed` ring is drawn in `line-dim` where the kit says "hollow ember ring".
- `SpeechBox` nests an `aria-live` span inside the `aria-live` `RetroPanel`, which risks a double announcement. Its tap-to-skip has no keyboard path.

## Expected output
Each item above matches design §5 and has a unit test pinning it.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`). `frontend/components/retro/HpBar.vue:33,39,44`, `DayBar.vue:32,42-47`, `RetroPanel.vue:30-31`, `QuestNode.vue:48`, `MapNode.vue:65`, `SpeechBox.vue:62-69`.

## Evaluation
**Verdict: select, low.** It is real. The unclamped `HP -5/100` label is the one item a learner could see (on stale pet data). The others are design-§5 fidelity (`steps(|Δ|)`, the missed ring in ember, the static cursor) and a11y polish: the `DayBar` accessible name, and SpeechBox's nested `aria-live` and missing keyboard skip. **Overlap:** the amend's A4 already makes the reduced QuestNode cursor static, and its A2 already keeps the `band` ring, so drop those two bullets when planning. Each remaining item gets one pinned test. **Plan deferred, not folded (evaluator, 2026-09-27):** today's bug list already holds five plans (the 5/day cap), and only the blocker amend jumps it. Every file this bug touches exists only on the unmerged retro branch, and today's amend (`harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`) rewrites them. So write the plan against `main` in the first bugfix run after the retro branch merges, before plans 2–6 compose the kit. Selected-but-unplanned bugs rank first in that run.
