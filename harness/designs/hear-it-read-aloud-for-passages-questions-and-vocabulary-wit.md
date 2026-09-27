# Design: Hear it — read-aloud for passages, questions and flashcards (`/learn/:id`)

**Idea:** `harness/ideas/2026-09-26-run-01/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.4, `harness/designs/retro-learning-room.md` (the eyebrow row, the "Đoạn văn" panel, "Thẻ {n} / {N}", the question box) — this doc adds one control to that room and changes nothing else in it.
**Kit:** `harness/UI-KIT.md` v2 + `harness/designs/retro-kit.md` (`PixelArt`, `GLYPHS`, `retro.css`, `RetroToast`).
**Spec wireframe:** frontend spec §7.3. Kept as the learning-room doc keeps it. One departure from the kit, not the wireframe: UI-KIT "Motion budget" says *"Sound-free by default; no audio API in scope."* This feature is the owner-selected exception — the only audio in the app is speech synthesis behind an explicit tap; there are still no sound effects. Proposed kit wording under "Kit additions".

**Designed against:** the kit code on `origin/harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`frontend/components/retro/RetroButton.vue`, `PixelArt.vue`, `frontend/utils/pixelArt.ts` `GLYPHS`, `frontend/assets/css/retro.css` with `retro-blink` and the reduced-motion block, `composables/useRetroToast.ts`). The room's item components (`ItemPassage`, `ItemFlashcard`, `ItemQuestion`) exist only in `retro-learning-room.md` — no branch has them yet — so the placement below is written against that doc's layout. `ContentViewer.vue` on `origin/harness/2026-09-26-high-59-of-84-…` still has only `words | questions | raw` (no `passage` kind; `utils/content.ts` on `main` is the same); `pages/learn/[id].vue` on `origin/harness/2026-09-25-high-task-timer-…` already has `onBeforeUnmount` (line 43) and `complete()` (line 61), which are the two hooks that must also call `stop()`.

## 0. Research
- **Learner's job on this screen:** read (or answer) one English item and, when they want to, hear it said properly — then keep going.
- **The moment that earns the next minute:** the first time a Vietnamese learner taps the horn on a word they were unsure how to say and hears it. A silent passage is homework; a passage that reads itself is practice. Hearing the prompt also unblocks the learner who stalls because they cannot parse a sentence by eye.
- **What today does wrong:** no file under `frontend/` references `speechSynthesis` or `<audio>`; the "reading/listening" category (1st-thinking §6.1) ships with no listening at all. `ContentViewer` renders text only.
- **Open questions, answered:**
  1. *Where does the control live — inside each panel, or one fixed slot?* One slot: the right end of the eyebrow row ("Câu 2 / 10", "Thẻ 2 / 6", "Đoạn văn"). Reasons: the flashcard front is itself the tap-to-flip target (a `<button>`), so a speaker inside it would be a nested button; the passage panel scrolls and the control must not scroll away; and a learner should find "Nghe" in the same place on every item. Nothing changes inside `RetroPanel`.
  2. *Icon only or a word?* A word. "Nghe" / "Dừng" is visible text on every placement (VT323 20 beside a 32-px glyph); the learner never has to guess what a horn does, and the speaking state is a word change plus a glyph change, never colour alone.
  3. *iOS reports zero voices.* Hide only when the API is missing, or when the voice list is *known* (`getVoices()` returned voices, or `voiceschanged` fired) and holds no `en` voice. An empty list is treated as "unknown": the button shows and the utterance carries `lang='en-US'` so the OS picks a voice. Assumption recorded.
  4. *What about the option text and the flashcard back?* Never spoken. Reading the options aloud would give away answer rhythm and triple the length; the back's definition/example is out of scope (idea: term only). The button stays visible on the flipped card and still speaks `term`, so the learner can hear the word while reading its meaning.
  5. *Backgrounded tab?* Speech stops when the tab hides (`visibilitychange`). The room is a reading room, not a podcast; predictable silence beats a plant that keeps talking under a locked screen. Assumption recorded.
  6. *Long passages.* `maxPassageRunes` allows ~4 000 characters. Chrome desktop historically cuts a single utterance at ~15 s. The passage is therefore spoken as a queue of sentence utterances created in the same tap (allowed under the gesture rule), and `stop()` cancels the whole queue.

## 1. The signature
**The eyebrow row can talk.** Every item's counter line ends in a small VT323 chip: a pixel horn and the word "Nghe". Tap it and the horn grows two sound-arcs that blink in `growth` while the item is read at a slightly slow pace; the word becomes "Dừng". Tap again, or move on, and the room is silent. It is the same chip on a passage, a question and a flashcard, so after the first tap the learner knows the whole room can be heard.

## 2. Flow
Entry and exits are unchanged from `retro-learning-room.md` §2. Speech is a side-channel of the current item: it starts only from a tap on the chip, and it ends on any of — the second tap, another chip's tap, the item advancing ("Tiếp tục", "Nhớ rồi", "Học lại", "Đọc xong"), the flashcard flip (no: flipping keeps the term speaking; the text is still the term), "‹ Trại", task completion (before `POST /quests/progress`), the tab hiding, or the page unmounting. Nothing on the hub, roadmap or chest ever speaks. Review mode (completed task) keeps the chip: listening is allowed after finishing.

## 3. Layout (mobile-first, `max-w-md`; desktop same column)
```
│ ‹ Trại        XP 20        ⏱ 04:12     │ top bar unchanged
│ ĐOẠN VĂN                   [🔊 NGHE ]   │ eyebrow row → flex, h-11 (44 px), items-center, justify-between
│ ┌─────────────────────────────────┐    │ passage panel unchanged (scroll-frame)
│ │ The learner reads a short …     │    │
```
- **Eyebrow row** (all three item kinds): `flex items-center justify-between h-11`. Left: the existing eyebrow text (VT323 16 uppercase). Right: `SpeakButton`. When `supported` is false the right slot renders nothing and the row keeps `h-11`, so the panel below never shifts between devices.
- **`SpeakButton` chip:** height 44, min-width 88, `px-2`, `gap-2`, `rounded-sm`, `bg-ground-2`, `border-2 border-line-lit`, hard shadow `0 4px 0 0 line-dim` (the `RetroButton` secondary recipe at 44 px), glyph `PixelArt GLYPHS.speaker` at 32 px (16-grid ×2) left, label VT323 20 `text-ink-0` right. Focus: 2-px `torch` outline, 2-px offset.
  - *Idle:* horn only, label "Nghe", `aria-pressed="false"`.
  - *Speaking:* border `growth`, the `speakerWaves` glyph overlaid on the horn (absolute, same 32-px box) with `retro-blink` (600 ms, `steps(2)`), label "Dừng", `aria-pressed="true"`. Pressed drop (translate 2 px, shadow 2 px) within 100 ms on both taps.
  - *Stopped* = idle again; there is no third visual state.
- **Passage sheet** ("Xem lại đoạn", learning-room §3): its own eyebrow "Đoạn văn" gets the same chip; opening the sheet does not auto-speak.
- **What each placement speaks** (data): passage → `content.passage` (whole string); question → `questions[i].prompt` only — the options object is never passed to `speak`; flashcard → `words[i].term` only, on both faces.

## 4. States
- **Unsupported** (`!('speechSynthesis' in window)`, or voices known and none `en`): chip absent (`v-if`), never disabled, no copy, no toast. Row height kept.
- **Supported, idle:** "Nghe".
- **Speaking:** "Dừng" + blinking arcs + `growth` border. Only one chip in the room is ever in this state.
- **Synthesis error** (`utterance.onerror` with any `error` other than `interrupted` / `canceled`): chip returns to idle; one `RetroToast tone=ember` "Tớ chưa đọc được. Thử lại nhé." — once per tap, not per queued sentence.
- **Offline:** unchanged — synthesis is local. Voice pick prefers `localService === true` so Chrome's network voices are not chosen when a local one exists; if only a remote voice exists and it fails, the error state above handles it.
- **First-time:** the first time a room with a supported chip opens on this device, a `RetroToast` (2 s, plain tone) "Chạm loa, tớ đọc cho cậu nghe." — key `aelp.speakHint` in `localStorage`, set after showing. Never repeated.
- **Loading / empty / error / chest:** no chip (no item on screen).
- **Review (completed task):** chip present and working; no XP, no posting, as before.
- **Reduced motion:** the arcs appear as one static frame (`retro.css` already zeroes every `retro-*` animation); the `growth` border and the word "Dừng" still mark the state.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| "Nghe" | chip drops; arcs + "Dừng" | `stop()` any current speech, `speak(text)` — passage split into sentences and queued; first sound when the engine starts (device-dependent, typically < 300 ms) | nothing |
| "Dừng" (same chip) | chip drops; arcs off, "Nghe" | `speechSynthesis.cancel()`; no toast | nothing |
| Another chip's "Nghe" | that chip enters speaking; the previous returns to idle | its text replaces the queue | nothing |
| Item advance / flip / "‹ Trại" / complete / tab hidden / unmount | — | `stop()` before the next thing happens; the timer is untouched | nothing |
| Keyboard | Space/Enter on the focused chip toggles like a tap | the `torch` focus ring stays visible | — |
Timer: `CountdownTimer` keeps measuring; listening time is task time. `quest.complete()` receives the same `elapsedSeconds` it would without speech. No store field is added; `useSpeech` state lives in the composable module only.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `SpeakButton` (`components/learn/SpeakButton.vue`) | new | props `text: string`, `id?: string` (defaults to a per-instance uid), `lang = 'en-US'`, `rate = 0.9`; emits `start`, `stop` | renders `null` while `!supported`; `active = activeId === id`; click → `active ? stop() : speak(text, {lang, rate, id})`; `watch(text)` and `onBeforeUnmount` → `if (active) stop()`; `@click.stop` so a parent tap target (flashcard) never flips |
| `useSpeech` (`composables/useSpeech.ts`) | new | `supported: Readonly<Ref<boolean>>`, `speaking: Readonly<Ref<boolean>>`, `activeId: Readonly<Ref<string \| null>>`, `speak(text, {lang?, rate?, id?})`, `stop()` | **module-level singleton state** (one `speechSynthesis` per window), so every chip in the room shares `speaking`/`activeId` and a second `speak` cancels the first. `supported` starts `false` and is decided on the client after mount (SSR-safe: nothing renders on the server). `voiceschanged` listener registered once. Reads `globalThis.speechSynthesis` / `SpeechSynthesisUtterance` lazily at call time so unit tests can stub them before mount. `speak`: `cancel()`, split on sentence ends (`/(?<=[.!?…])\s+/`), one utterance per sentence with `lang`, `rate`, `voice` (pick below), chained; `speaking=true` on the first `onstart` (or immediately, if `onstart` has not fired within 250 ms — Safari sometimes skips it), `false` on the last `onend`/`onerror`; `error ∉ {interrupted, canceled}` → toast via `useRetroToast`. Never called on mount, watch or route change — only from a click handler (iOS gesture rule) |
| Voice pick (inside `useSpeech`) | new | — | normalise `lang` (`en_US` → `en-US`, case-insensitive); order: `en-US` local → `en-US` → `en-GB` local → `en-GB` → any `en` local → any `en`; none → `voice` unset, `lang` still `'en-US'` |
| `ContentViewer` / `ItemPassage` / `ItemQuestion` / `ItemFlashcard` | existing (learning-room plan) | — | each item's eyebrow row becomes the flex row in §3 and mounts `<SpeakButton :text="…" />`; the passage sheet too |
| `pages/learn/[id].vue` | existing | — | `onBeforeUnmount` and `onBeforeRouteLeave` → `stop()`; `complete()` → `stop()` first; `visibilitychange` → hidden → `stop()` |
| `RetroToast` / `useRetroToast` | kit | — | error and first-time lines |

**Kit additions (land only through the plan):**
- Two 16-row glyphs in `GLYPHS`: `speaker` — a horn (cone opening to the right, `l` fill, `k` outline, one `i` glint) — and `speakerWaves` — two 1-px arcs to the right of the horn in `g`, everything else `.`, same grid so they overlay pixel-for-pixel. Reason: the kit has no audio glyph; a Unicode 🔊 would be an emoji in `font-body`, against the glyph rule.
- `SpeakButton` as a 44-px labelled icon chip (the `RetroButton` secondary recipe at 44 px with a glyph slot). Reason: `RetroButton` is 48 px and word-only; the eyebrow row cannot carry a 48-px block button without dwarfing the counter.
- Kit wording, "Motion budget" last bullet → "Sound-free by default: no sound effects. The only audio is speech synthesis behind an explicit tap (`SpeakButton`, `useSpeech`); nothing speaks without a user gesture."

## 7. Copy
Chip: "Nghe" · "Dừng" (also the `aria-label`; `aria-pressed` carries the state). First-time toast: "Chạm loa, tớ đọc cho cậu nghe." Error toast: "Tớ chưa đọc được. Thử lại nhé." No other strings; the spoken text is the English content itself.

## 8. Self-critique
- **Traded away:** a speaker *inside* each panel (closer to the words) for one fixed slot in the eyebrow row. Cheaper, no nested buttons, no scroll-away, one habit — but on a long passage the chip is above the text, not beside it.
- **Motion budget:** the blinking arcs are a second animated thing in a room whose motion belongs to the answer feedback. Kept because a silent-volume device otherwise cannot tell that speech is running; it is a 2-frame blink on a 32-px glyph, and reduced motion removes it. If review finds it noisy, static arcs are the fallback (border + word still carry the state).
- **Sentence splitting** changes prosody slightly at each split versus one utterance; accepted to survive Chrome's long-utterance cut-off and to let `stop()` work mid-passage.
- **Executor traps:** `supported` must be decided on the client after mount or Nuxt hydration mismatches; `getVoices()` empty ≠ unsupported (iOS) — only *known-and-no-English* hides the chip; never call `speak` outside a click handler (iOS drops it silently); the singleton must not be created per component (two `speaking` refs → two chips lit); `cancel()` fires `onerror`/`onend` with `interrupted`/`canceled` on some engines — do not toast on those; clear the 250-ms `onstart` fallback timer on stop/unmount; `@click.stop` on the chip or the flashcard flips on every tap; the timer plan's `onBeforeUnmount` already exists — add to it, do not replace it; happy-dom has no `speechSynthesis`, so tests stub `window.speechSynthesis` + `SpeechSynthesisUtterance` and assert calls, not sound.
- **Review checks:** tap "Nghe" on a question, then "Tiếp tục" → no sound on the next item and the new chip reads "Nghe"; complete a task while a passage is speaking → hub is silent; DevTools "Emulate prefers-reduced-motion" → static arcs; delete `window.speechSynthesis` before load → no chip and the eyebrow row keeps its height; posted `duration_seconds` equals the timer with and without listening.

## Acceptance
1. In a browser with `speechSynthesis` and at least one `en` voice, every passage, question and flashcard item shows a "Nghe" chip at the right end of its eyebrow row, 44 px tall and ≥ 88 px wide; loading, empty, error and chest states show none.
2. Tapping "Nghe" shows the pressed drop, the arcs and the word "Dừng" within 100 ms; tapping "Dừng" stops speech and returns the chip to "Nghe" within 100 ms.
3. The passage chip speaks the whole `passage`; a question chip speaks only `prompt` (no option text is ever passed to `speak`); a flashcard chip speaks only `term`, on either face.
4. At most one chip in the room is in the speaking state; tapping a second chip stops the first and starts the second.
5. Advancing the item, tapping "‹ Trại", completing the task, hiding the tab, or unmounting the page cancels speech; nothing is audible on the hub afterwards.
6. Nothing speaks without a tap: opening a room, advancing an item, revealing feedback or opening the passage sheet never calls `speak`.
7. With `window.speechSynthesis` absent, or voices known and none matching `en`, no chip renders (not disabled, not hidden by CSS) and the eyebrow row height is unchanged; with an empty voice list the chip renders and the utterance carries `lang='en-US'`.
8. The chosen voice follows `en-US` (local first) > `en-GB` > any `en`; every utterance has `rate` 0.9.
9. The timer value and the `duration_seconds` posted on completion are identical whether or not speech was used.
10. Under `prefers-reduced-motion: reduce` the speaking state is a static arcs frame with the `growth` border and "Dừng"; a synthesis error shows the ember toast once and the chip returns to "Nghe".
