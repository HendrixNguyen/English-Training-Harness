---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# learnPage test finds the completion button by its mt-4 spacing class

## Why
`frontend/tests/unit/learnPage.test.ts:63` and `:99` find the page's primary button with `w.find('button.mt-4')` — a spacing utility class merged onto `AppButton` via attribute fall-through. Any layout tweak (the retro kit migration of `/learn/:id` in `harness/designs/retro-learning-room.md` replaces `AppButton` and its spacing) silently retargets or breaks the test, and a second `mt-4` button would make it assert on the wrong element. The executor chose it because `ContentViewer`'s fallback also renders a "Hoàn thành" button.

## Expected output
The page's completion button carries a stable hook (`data-testid="complete"` or an accessible name distinct from `ContentViewer`'s inner button), and both `learnPage` cases select it by that hook.

## Evidence
- Plan: `harness/plans/2026-09-25-task-timer-keeps-counting-through-reloads-and-background-tab.md` (execution summary, Task 2 test deviation).
- `git show 17205de:frontend/tests/unit/learnPage.test.ts | grep -n "mt-4"` → lines 63, 99.
