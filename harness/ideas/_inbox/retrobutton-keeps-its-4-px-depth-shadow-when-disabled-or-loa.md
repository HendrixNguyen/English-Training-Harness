---
type: bug
status: proposed
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
