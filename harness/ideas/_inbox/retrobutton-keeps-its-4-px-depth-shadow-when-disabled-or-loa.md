---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# RetroButton keeps its 4 px depth shadow when disabled or loading

## Why
The kit says disabled means "50 % opacity, no shadow". The scoped `.rb { box-shadow: 0 4px 0 0 var(--rb-shadow) }` wins over Tailwind's `disabled:shadow-none`. In the browser, the disabled and loading buttons both computed `rgb(23,138,105) 0px 4px 0px 0px`, so a disabled button still looks pressable.

## Expected output
Add `.rb:disabled { box-shadow: none }` in the scoped style, or drop the Tailwind variant, and add a test that asserts it.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`). `frontend/components/retro/RetroButton.vue:33,48-54`. Browser computed styles for the "Tắt" (disabled) and loading buttons in the kit page.

## Evaluation
**Verdict: select, low.** The *Why* is real: UI-KIT says disabled means "50 % opacity, no shadow", and a disabled button that still looks pressable is a small learner-visible lie. It does not block anything, because no page uses `RetroButton` until plan 2. **Root cause:** the scoped `.rb { box-shadow }` outranks Tailwind's `disabled:shadow-none` (`RetroButton.vue:33,48-54`). **Fix:** add `.rb:disabled, .rb[aria-busy=true] { box-shadow: none }` plus a test. This is a one-task plan. **Plan deferred, not folded (evaluator, 2026-09-27):** today's bug list already holds five plans (the 5/day cap), and only the blocker amend jumps it. Every file this bug touches exists only on the unmerged retro branch, and today's amend (`harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`) rewrites them. So write the plan against `main` in the first bugfix run after the retro branch merges, before plans 2–6 compose the kit. Selected-but-unplanned bugs rank first in that run.
