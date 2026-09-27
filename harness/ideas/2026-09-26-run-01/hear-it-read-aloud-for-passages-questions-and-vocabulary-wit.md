---
type: feature
status: planned
source: ideator
run: 2026-09-26-run-01
order: 2
priority: medium
plan: harness/plans/2026-09-27-hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md
---

## Why
The spec promises three task categories and the second one is "reading/listening" (1st-thinking §6.1), yet the app has no sound at all: `ContentViewer.vue` renders `words` and `questions` as text, the typed-content plan adds a `passage`, and no file in `frontend/` references `speechSynthesis` or `<audio>`. For a Vietnamese learner the gap is felt immediately — hearing how a word or a sentence sounds is the first thing they want from an English app, and a silent passage is homework, not practice. The device already has an English voice: the Web Speech `SpeechSynthesis` API is supported by every browser the PWA targets (Chrome, Safari incl. iOS, Firefox, Samsung Internet), runs offline, and costs nothing per call — no provider, no rate limit, no backend. One tap turns every reading task into a listening task and every flashcard into a pronunciation card, which is the cheapest way to lift the "learning signal" of the 30 minutes without touching the AI budget. It builds on the learning room the typed-content bug and the retro learning-room design are already reshaping, and adds one control to it rather than a new surface.

## Expected output
User-visible:
- A speaker button on: the passage (reads the whole passage, highlights nothing — keep it simple), each question prompt and, on flashcards, the term (front) — pressing it again stops. Rate is slightly slow (`rate 0.9`), language `en-US` with a fallback to any `en-*` voice; on iOS `getVoices()` may be empty, so the system default is used silently.
- Playback survives nothing: navigating away or completing the task stops speech (`speechSynthesis.cancel()` on unmount/route leave) so the plant never talks over the hub.
- No voice available (`!("speechSynthesis" in window)` or zero English voices after `voiceschanged`) → the button is hidden, not disabled; nothing else changes. Screen-reader label "Nghe" / "Dừng".
- Listening time counts like reading time: the task timer is unchanged.

Technical (frontend only):
- `composables/useSpeech.ts`: `speak(text, {lang, rate})`, `stop()`, reactive `speaking`, `supported`, voice pick (`en-US` > `en-GB` > any `en`), `voiceschanged` handling, a guard that every `speak` follows a user gesture (iOS requirement). One `SpeakButton.vue` in `components/learn/` used by `ContentViewer` for the three item kinds. Copy and placement from the designer role inside the retro learning-room design (the "Đoạn văn" panel and the flashcard front).
- Unit tests with a fake `speechSynthesis` (`happy-dom` has none): supported/unsupported, voice choice, cancel on unmount, second tap stops. No e2e needed.
- Estimate: half a day. No backend, no wire change, no new dependency.

## Evidence
- Spec: 1st-thinking §6.1 (task categories "reading/listening"); frontend spec §5 (7.3 task view renders the content), §7.3 (learning room), §4 (`useQuestStore` timers untouched).
- Code: `frontend/components/learn/ContentViewer.vue:11-54` (`words` / `questions` kinds only); `grep -rn speechSynthesis frontend` → no match; `harness/designs/retro-learning-room.md:82` ("Đoạn văn", "Thẻ {n} / {N}", "Lật") — the panels the button goes on.
- Depends on (information only): the typed-content contract (`harness/plans/2026-09-24-typed-task-content-…` backend half, done) and the high inbox bug `59-of-84-roadmap-tasks-render-as-raw-json-…` for the passage to exist on screen; the button itself works on today's `words`/`questions` already.
- Research: `SpeechSynthesis` support across Chrome 33+, Edge 14+, Firefox 49+, Safari 7+, Samsung Internet 5+; synthesis is local and works offline — https://www.testmuai.com/learning-hub/speech-synthesis-api-browser-support/ ; Safari limitations (`getVoices()` can be empty, user-gesture requirement) — https://weboutloud.io/bulletin/speech_synthesis_in_safari/ ; API reference — https://developer.mozilla.org/en-US/docs/Web/API/Web_Speech_API

## Evaluation
_Evaluator, 2026-09-27 — daily decide (feature queue, planned today as **F5**)._

**Verdict: select, priority medium.**

- **Is the *Why* real?** Yes. The spec's second task category is "reading/listening" (1st-thinking §6.1) and nothing under `frontend/` produces sound (`git grep -n speechSynthesis origin/main -- frontend` → no match; the same on every unmerged 2026-09-25/26 branch). For a Vietnamese learner, hearing a word or a prompt said properly is the most direct learning value one control can add, and the Web Speech API gives it offline, per-device, with no provider, no rate limit and no AI budget. It adds one chip to an existing screen, not a new surface.
- **Achievable in one plan?** Yes — frontend only, ~5 h: a singleton composable, one chip component, two kit glyphs, their unit tests against a fake `speechSynthesis`, then wiring into the room. No backend, no wire change, no new dependency.
- **Dependencies.** The design (`harness/designs/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md`) was drawn against the retro kit on `harness/2026-09-26-high-retro-adventure-ui-…` (`PixelArt`, `GLYPHS`, `retro.css` `retro-blink`, `useRetroToast`) and the page on `harness/2026-09-25-high-task-timer-…` (`onBeforeUnmount`, `complete()`); both are `done` and unmerged because the 2026-09-26 review run never ran, so the plan is gated on the 2026-09-26 daily code PR. The `59-of-84` and `typed-task-content` branches carry no `frontend/` change (checked with `git diff --stat origin/main...<branch> -- frontend`), so after the merge `ContentViewer.vue` still renders only `words | questions | raw` and `utils/content.ts` has no `passage` kind: a reading task's `passage` is not on screen until the retro learning-room plan (`harness/designs/retro-learning-room.md` — `ItemPassage`, `ItemFlashcard`, `ItemQuestion`, the passage sheet; not yet planned) lands. The plan therefore wires the chip into today's `ContentViewer` (question `prompt`, word `term`) unconditionally and into the learning-room item components in a gated last task that runs only if they are on `origin/main` when execution starts; otherwise the passage placement is handed to the learning-room plan through a note in the design.
- **Priority rationale.** Medium: clear learning and retention value on the happy path of every task, but nothing is broken without it and it does not block a merge. Not high, so the plan stays `draft` for the owner's `/approve`.
- **Kit exception.** UI-KIT "Motion budget" says "no audio API in scope"; the design records this feature as the owner-selected exception (speech behind an explicit tap, still no sound effects) and the plan updates that bullet.
