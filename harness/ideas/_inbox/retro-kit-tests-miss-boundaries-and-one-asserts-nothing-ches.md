---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Retro kit tests miss boundaries and one asserts nothing: Chest unmount, SpeechBox mid-type unmount, empty inputs, out-of-range values

## Why
The read-only test-gap pass (the-validator) found tests that give false confidence for the kit plans 2–6 will build on:
- `Chest.test.ts:39-44` "clears its timer on unmount" only advances timers and never asserts anything. If the `clearTimeout` were removed, the test would still pass.
- `SpeechBox.test.ts` never unmounts while typing, so the `setInterval` cleanup is untested. It also never tests `line: ''`.
- Missing boundaries: `HpBar` `value` of -1 or over max; `DayBar` negative `valueSeconds` or values past the target; `Chest` `items: []`; `Badge` `count: 0` or omitted for `streak`; `MapNode` `day <= 0`.
- `RetroToast.test.ts` shares the module-level `queue`/`timer` singleton (`composables/useRetroToast.ts:13-14`) and drains it only in `beforeEach`. There is no `vi.resetModules()`, so tests depend on order.

## Expected output
The unmount tests assert that no emit happens and no timer is pending (`vi.getTimerCount() === 0`). Each boundary above has one `it`. The toast suite resets the module between tests, or `useRetroToast` exposes a test reset.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`). Files: `frontend/tests/unit/{Chest,SpeechBox,HpBar,DayBar,Badge,MapNode,RetroToast}.test.ts`.
