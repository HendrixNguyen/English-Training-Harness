---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Typed reading tasks show questions without their passage and vocabulary tasks hide their questions until the frontend slice lands

## Why
The plan's Goal says today's frontend "renders all of it without a change". It does not. `frontend/utils/content.ts` `classifyContent` returns the **first** array it finds, `words` before `questions`, and has no `passage` branch. So once typed roadmaps flow (onboarding, or the regenerate route):
- a **reading** task renders only its 3-5 questions, and the passage they ask about is never shown. Today's raw-JSON fallback at least contains the text;
- a **vocabulary** task renders only the word list, and its 3-5 self-check questions are invisible;
- `example` sentences are dropped (the `Word` type has only `term`/`definition`).

For reading tasks this is a user-visible regression if the backend branch ships before the frontend slice (retro learning room / part 2).

## Expected output
Until part 2 lands, `classifyContent` gains a minimal `passage` branch (passage text above the questions) and renders `questions` after `words` when both exist. Alternatively, the daily PR holds the typed-content branch until the frontend slice that renders `passage`/`questions`/`example` is merged alongside it. Covered by a `content.spec` test per task type using `SampleContent`-shaped fixtures.

## Evidence
- Plan: `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`, Goal line and Architecture ("the frontend keeps its raw fallback").
- `frontend/utils/content.ts` (`Word{term, definition}`, `classifyContent`: `words` wins, then `questions`, else `raw`); no `passage` handling anywhere in `frontend/` (`grep -rn passage frontend/utils frontend/pages` finds nothing).
- Idea `harness/ideas/2026-09-24-run-01/typed-task-content-with-answer-keys-so-every-quest-renders-a.md` Expected output: "Reading/listening tasks show a passage followed by multiple-choice questions".
