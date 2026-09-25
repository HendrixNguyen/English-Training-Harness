---
type: feature
status: proposed
source: ideator
run: 2026-09-26-run-01
order: 2
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
