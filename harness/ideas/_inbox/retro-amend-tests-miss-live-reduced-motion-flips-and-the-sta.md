---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md
---
# Retro amend tests miss live reduced-motion flips and the StateBlock ember tone, and SpeechBox ignores a mid-line OS flip

## Why
The amend made the OS reduced-motion setting the default for six kit components (design A4). The tests prove only two static snapshots, not the live behaviour:
- **No consumer test flips the OS setting after mount.** `useReducedMotion.test.ts` flips the composable's own ref on a `change` event. No Chest/DayBar/HpBar/QuestNode/SpeechBox/CompanionSprite test mounts with `reduced` unset, fires `change`, and checks that the render updates.
- **`CompanionSprite` has no OS-default test at all.** `tests/unit/CompanionSprite.test.ts` never mocks `useReducedMotion`; its reduced tests all pass `reduced: true`. Yet it has the most OS-dependent logic: the hold timer, the `reacted` emit, `data-frame`, the torch palette and the `hit` translate.
- **SpeechBox is inconsistent.** `components/retro/SpeechBox.vue:51` runs `watch(() => props.line, startTyping, { immediate: true })`, which does not watch `isReduced`. If the learner turns on reduced motion mid-line, the typing interval keeps running until the next line. `CompanionSprite.vue` watches `[react, isReduced]`, so the two siblings behave differently.
- **Dishonest assertion.** `tests/unit/StateBlock.test.ts:26-38` is titled "wraps a RetroPanel toned ember" but asserts only `style.length > 0`. `RetroPanel` always emits a `box-shadow`, so the test still passes if StateBlock's `tone` binding is reverted to `'plain'`.
- Minor: `CountdownTimer.test.ts:30-33` re-asserts pre-existing `aria-live="off"`; no test passes a falsy palette override to `PixelArt`, although the code guards for it.

## Expected output
- Per consumer, one test with `reduced` unset: mount with OS `false`, fire the mocked `change` to `true`, `await nextTick()`, and assert the reduced render (no transition, whole line shown, chest open, cursor static, sprite hold then `reacted`).
- A `CompanionSprite` OS-default block (OS true → hold then emit; explicit `reduced=false` → animation class).
- `SpeechBox` reacts to an `isReduced` change mid-line by revealing the whole line and emitting `settled` once; add `isReduced` to the watch sources, and pin it with a test.
- The StateBlock error test asserts the panel style contains `tokens.ember`, and the empty state does not.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, amend `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`, branch @ `2e0125c`.
- Validator pass (read-only) during the re-review. The 13 touched test files pass 114/114, so these are gaps, not failures.
- Files: `frontend/components/retro/SpeechBox.vue:51`, `frontend/tests/unit/StateBlock.test.ts:34`, `frontend/tests/unit/CompanionSprite.test.ts` (describes at :7 and :62, no `useReducedMotion` mock).

## Evaluation
**Verdict: select, priority medium; folded into the amend-2 plan for the blocker `retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md`.** It lands on the same branch and touches the same test suite, and it is small (under 1 h).

**Why it matters.** A learner who turns on reduced motion mid-line keeps seeing typing. The tests also let a reverted tone binding pass unnoticed.

**Root cause.** `SpeechBox.vue:51` watches only `props.line`, so an `isReduced` flip is ignored until the next line arrives. `StateBlock.test.ts:34` asserts `style.length > 0`, which `RetroPanel`'s ring always satisfies.

**Fix (design A11).**
- Add `watch(isReduced, r => { if (r) revealAll() })`. `revealAll` already guards `settled`, so the event fires only once.
- Do not add `isReduced` to the `line` watch, because that would retype the line.
- Add a live-flip test for each of the six consumers, an OS-default block for `CompanionSprite`, and a `tokens.ember` assertion for the `StateBlock` error state.

The `CountdownTimer` re-assert and the falsy `PixelArt` palette are minor, and this plan does not cover them.
