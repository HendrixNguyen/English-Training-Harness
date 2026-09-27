---
idea: harness/ideas/2026-09-26-run-01/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md
status: approved
priority: medium
merged: false
order: 2
design: harness/designs/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md
---
# Hear it — read-aloud for passages, questions and flashcards (`useSpeech`, `SpeakButton`) on `/learn/:id` — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F5** of 2026-09-27. **Estimate:** 5 h. **Branch:** `harness/2026-09-27-medium-hear-it-read-aloud-for-passages-questions-and-vocabulary-wit`.

**Design:** `harness/designs/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md` — every frontend task cites its sections; `## Verification` repeats its Acceptance list verbatim.
**Idea:** `harness/ideas/2026-09-26-run-01/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md`. Frontend only — no backend, no wire change, no new dependency.

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`components/retro/PixelArt.vue`, `utils/pixelArt.ts` `GLYPHS`/`PALETTE`, `assets/css/retro.css` `retro-blink` + the reduced-motion block, `composables/useRetroToast.ts`, `components/retro/RetroToast.vue`, the `tokens` export of `tailwind.config.ts`), `harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` (`pages/learn/[id].vue` with `onBeforeUnmount`, the `visibilitychange` listener and `complete()`; `tests/unit/learnPage.test.ts`), `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` and `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (the typed content contract: `words[].term`, `questions[].prompt`, `passage`) — if any is missing when you start, stop and report. Verify with
```
for b in 2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n 2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab 2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a 2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
test -f frontend/components/retro/PixelArt.vue && test -f frontend/composables/useRetroToast.ts && grep -n 'onBeforeUnmount' 'frontend/pages/learn/[id].vue'
```
(four `ok`, then a hit). If a branch was deleted after merging, `git log origin/main --oneline | grep -c -E 'retro-adventure-ui|task-timer-keeps-counting|59-of-84|typed-task-content'` ≥ 4 is the fallback check.

**Goal:** Every English item on `/learn/:id` — each question prompt and each flashcard term today, and the passage once the retro learning room renders it — carries a "Nghe" chip in its eyebrow row that reads the item aloud through the device's own `speechSynthesis` on a tap, and nothing in the room ever speaks without one.

**Architecture:** One module-level singleton, `composables/useSpeech.ts`, owns the only `speechSynthesis` conversation in the window: `supported`, `speaking`, `activeId`, `speak`, `stop`, the voice pick and sentence splitting (design §6). One presentational chip, `components/learn/SpeakButton.vue`, reads that singleton and renders nothing while `!supported` (design §3, §4). The kit gains two 16×16 glyphs, `speaker` and `speakerWaves` (design §6 "Kit additions"). The room mounts the chip in each item's eyebrow row and the page stops speech on every exit (design §2, §5). No store field is added; the timer and `quest.complete()` are untouched (design §5 "Timer").

**Tech stack:** Nuxt 3 SPA (`ssr: false`), Vue 3 `<script setup>`, TypeScript strict, Tailwind with the retro tokens, Vitest + `@vue/test-utils` on `happy-dom` (which has **no** `speechSynthesis` — every test installs a fake).

## Placement on the post-merge tree (read this before Task 5)

The design was written against `harness/designs/retro-learning-room.md`, whose item components (`ItemPassage`, `ItemFlashcard`, `ItemQuestion`, the "Xem lại đoạn" sheet) are **not on any branch and not yet planned**. After the 2026-09-26 PR, `frontend/components/learn/ContentViewer.vue` still renders only `words | questions | raw` from `utils/content.ts` `classifyContent` (the 59-of-84 and typed-content branches carry no `frontend/` change — `git diff --stat origin/main...<branch> -- frontend` is empty for both), with one eyebrow `<p>` "Câu {n} / {N}" for both kinds and the word's `term` and `definition` on one face (no flip). So:

- **Task 5 (always):** the chip goes into `ContentViewer`'s eyebrow row — question → `prompt`, word → `term`. No `passage` exists on screen yet; do not add a `passage` kind to `classifyContent` (that is the learning-room plan's job).
- **Task 6 (gated):** only if `frontend/components/learn/ItemPassage.vue` exists on `origin/main` when you start, mount the chip in the three item components and the passage sheet as design §3 describes. Otherwise skip it, say so in the `done` summary, and the reviewer marks Acceptance 3's passage clause, the flashcard "either face" clause and the passage-sheet part of 6 as not-yet-applicable.

## Global Constraints

- Work in `.worktrees/<slug>` on the branch above; every command from `frontend/`. `rg` and `timeout` are not installed — `grep -n`, and Vitest's own timeouts.
- **Never call `speak` except from a click handler** (design §6, §8 — iOS drops utterances that do not follow a user gesture). No `speak` in `onMounted`, `watch`, a route hook or a timer callback other than the chaining inside one `speak` call.
- **One singleton:** `useSpeech`'s refs are declared at module scope, not inside the function; every `SpeakButton` shares them. `useSpeech()` returns `readonly()` views.
- `supported` starts `false` and is decided on the client in `onMounted` (via `ensureSpeechInit()`), never at import time. Read `globalThis.speechSynthesis` and `globalThis.SpeechSynthesisUtterance` lazily at call time — never `window.speechSynthesis` captured in a top-level `const` — so tests can stub before mount.
- An **empty** `getVoices()` is "unknown", not "unsupported" (iOS): the chip shows and the utterance carries `lang = 'en-US'` with no `voice` (design §0.3, Acceptance 7).
- `cancel()` makes some engines fire `onend`/`onerror` with `interrupted`/`canceled` on the old utterances: every handler checks a generation counter and ignores stale events; never toast on `interrupted`/`canceled` (design §4, §8).
- The chip is `v-if`, never `disabled`, never hidden by CSS; the eyebrow row keeps `h-11` either way (Acceptance 7).
- `@click.stop` on the chip (design §6 — a parent tap target must never flip).
- Do not restyle `ContentViewer` or the page beyond the eyebrow row; do not change `stores/quest.ts`, the timer, `complete()`'s arguments or `utils/content.ts`.
- The existing `onBeforeUnmount` and `visibilitychange` `sync` listener in `pages/learn/[id].vue` stay; add to them, never replace them (design §8).
- Copy is exactly design §7: "Nghe", "Dừng", "Chạm loa, tớ đọc cho cậu nghe.", "Tớ chưa đọc được. Thử lại nhé." — no other strings.
- Each task: `npm run lint && npm run typecheck && npm run test:unit` green before its commit; one commit per task.

## Review Focus

1. `speak` is reachable only from `SpeakButton`'s click handler: `grep -rn '[^.A-Za-z]speak(' frontend --include=*.vue --include=*.ts | grep -v tests` shows the definition in `useSpeech.ts` and one call in `SpeakButton.vue`, nothing else.
2. The singleton: two mounted chips share one `activeId`; tapping the second stops the first (Acceptance 4, `speakButton.test.ts`).
3. Unsupported vs unknown: missing API → no chip; voices known with no `en` → no chip; empty voice list → chip, `lang 'en-US'`, no `voice` (Acceptance 7, `useSpeech.test.ts`).
4. Voice order `en-US` local → `en-US` → `en-GB` local → `en-GB` → any `en` local → any `en`, `en_US` normalised; `rate` 0.9 on every utterance (Acceptance 8).
5. Stale events after `cancel()` never flip `speaking` back on, never toast; the 250 ms `onstart` fallback timer is cleared by `stop()` and by a finished queue.
6. Every exit stops speech: item advance, `‹ Quay lại` / route leave, `complete()` (before the POST), tab hidden, unmount (Acceptance 5); the posted `elapsedSeconds` is unchanged (Acceptance 9).
7. Question chips pass `prompt` only — no option text ever reaches `speak` (Acceptance 3).

## File structure

| Path | Change |
| --- | --- |
| `frontend/utils/pixelArt.ts` | `speaker` and `speakerWaves` glyphs in `GLYPHS` (Task 1) |
| `frontend/tests/unit/pixelArt.test.ts` | the two glyphs: square 16, palette chars only, waves never overlap the horn (Task 1) |
| `frontend/tests/unit/fakeSpeech.ts` (new) | fake `speechSynthesis` + `SpeechSynthesisUtterance` and `installFakeSpeech`/`uninstallFakeSpeech` (Task 2) |
| `frontend/composables/useSpeech.ts` (new) | singleton state, `ensureSpeechInit`, `pickVoice`, `splitSentences`, `speak`, `stop`, first-time hint, `__resetSpeechForTests` (Tasks 2–3) |
| `frontend/tests/unit/useSpeech.test.ts` (new) | support detection, voice pick, splitting, queue, stale events, errors, hint (Tasks 2–3) |
| `frontend/components/learn/SpeakButton.vue` (new) | the 44-px chip (Task 4) |
| `frontend/tests/unit/speakButton.test.ts` (new) | render/absent, states, toggling, singleton, unmount, `watch(text)`, `@click.stop` (Task 4) |
| `frontend/components/learn/ContentViewer.vue` | eyebrow row → flex `h-11` + `SpeakButton` for `words`/`questions`; `stop()` on `next()` (Task 5) |
| `frontend/tests/unit/contentViewerSpeech.test.ts` (new) | what each kind speaks, advance stops, raw has no chip (Task 5) |
| `frontend/pages/learn/[id].vue` | `stop()` in `onBeforeUnmount`, `onBeforeRouteLeave`, `complete()`, tab hidden; mount `<RetroToast />` if no ancestor does (Task 5) |
| `frontend/tests/unit/learnPage.test.ts` | exits stop speech; posted elapsed unchanged with speech (Task 5) |
| `frontend/components/learn/ItemPassage.vue`, `ItemQuestion.vue`, `ItemFlashcard.vue`, the passage sheet + their tests | **gated** Task 6 only |
| `harness/UI-KIT.md`, `harness/CODEMAP.md` | "Motion budget" bullet (design §6 "Kit additions"), `shell` bullet (Task 7) |

## Tasks

### Task 1: `speaker` and `speakerWaves` glyphs (design §6 "Kit additions")

**Files:** `frontend/utils/pixelArt.ts`, `frontend/tests/unit/pixelArt.test.ts`.

- [ ] **Step 1 (tests first):** in `pixelArt.test.ts` add `describe('speaker glyphs')`: both `GLYPHS.speaker` and `GLYPHS.speakerWaves` are 16 rows of 16 chars; `speaker` uses only `k`, `l`, `i`, `.` and has at least one `i` (the glint); `speakerWaves` uses only `g` and `.`, has ≥ 6 `g` pixels, and every `g` sits at a position where `speaker` is `.` (the overlay never covers the horn) and in columns ≥ 9 (to the right of the horn). The existing "every glyph is square and uses only palette chars" test picks both up automatically.
- [ ] **Step 2:** `npx vitest run tests/unit/pixelArt.test.ts` — FAIL (`GLYPHS.speaker` undefined).
- [ ] **Step 3:** add `speakerGlyph()` and `speakerWavesGlyph()` in the style of the neighbouring helpers (`canvas(16)`, `fillRect`, `setPx`, `toRows`): a horn whose small rectangular back (`l` fill, `k` outline) sits around columns 2–4, rows 6–9, and whose cone widens to the right up to column 8, rows 3–12, with one `i` glint pixel; `speakerWaves` = two 1-px arcs in `g` — a short one around column 10 (rows 5–10) and a long one around column 12–13 (rows 3–12) — everything else `.`. Register both in `GLYPHS` after `cursor`.
- [ ] **Step 4:** `npx vitest run tests/unit/pixelArt.test.ts` — PASS. `npm run lint && npm run typecheck`.
- [ ] **Step 5:** Commit: `retro kit: speaker and speakerWaves glyphs`.

### Task 2: the fake engine and `useSpeech` support detection + voice pick (design §0.3, §4 "Unsupported", §6 `useSpeech` / "Voice pick")

**Files:** `frontend/tests/unit/fakeSpeech.ts` (new), `frontend/composables/useSpeech.ts` (new), `frontend/tests/unit/useSpeech.test.ts` (new).

- [ ] **Step 1 (fake):** `fakeSpeech.ts` (a helper module, not a `*.test.ts`, like `fakeCaches.ts`):
  ```ts
  export class FakeUtterance {
    text: string; lang = ''; rate = 1; voice: FakeVoice | null = null
    onstart: (() => void) | null = null
    onend: (() => void) | null = null
    onerror: ((e: { error: string }) => void) | null = null
    constructor(text: string) { this.text = text }
  }
  export interface FakeVoice { name: string, lang: string, localService: boolean, default?: boolean }
  export function installFakeSpeech(voices: FakeVoice[] = [{ name: 'Samantha', lang: 'en-US', localService: true }]) { … }
  export function uninstallFakeSpeech() { … }   // vi.unstubAllGlobals() is enough if you stub with vi.stubGlobal
  ```
  `installFakeSpeech` stubs `speechSynthesis` and `SpeechSynthesisUtterance` with `vi.stubGlobal` and returns a handle: `synth.speak` / `synth.cancel` as `vi.fn`, `queue: FakeUtterance[]` (spoken, not yet ended), `spoken: FakeUtterance[]` (every `speak` argument ever), `start()` (fires `onstart` on `queue[0]`), `end()` (fires `onend` on `queue[0]` and shifts), `fail(error)` (fires `onerror({error})` on `queue[0]` and shifts), `setVoices(v)` + `fireVoicesChanged()` (calls every `voiceschanged` listener added through `addEventListener`), and a `cancel` that fires `onerror({error:'canceled'})` on every queued utterance (the worst-case engine) and empties the queue. Unsupported = `vi.stubGlobal('speechSynthesis', undefined)`.
- [ ] **Step 2 (tests first):** `useSpeech.test.ts` — `beforeEach(__resetSpeechForTests)`, `afterEach(() => vi.unstubAllGlobals())`:
  - `supported` is `false` before `ensureSpeechInit()` even with the fake installed (nothing decided at import).
  - API missing → `ensureSpeechInit()` leaves `supported` `false`.
  - voices `[en-US]` → `true`; voices `[vi-VN, fr-FR]` → `false`; voices `[]` → `true` (unknown); `[]` then `setVoices([vi-VN]) + fireVoicesChanged()` → `false`; `[vi-VN]` then `setVoices([vi-VN, en-GB])` + fire → `true`.
  - `ensureSpeechInit()` called three times registers one `voiceschanged` listener.
  - `pickVoice` table (exported, pure): `[en-GB local, en-US remote, en-US local]` → the `en-US` local; `[en-GB local, en-US remote]` → `en-US` remote; `[en-AU local, en-GB remote, en-GB local]` → `en-GB` local; `[en-IN remote, en-AU local]` → `en-AU` local; `[en_US remote]` → it (normalised, case-insensitive `EN-us` too); `[vi-VN]` → `null`; `[]` → `null`.
- [ ] **Step 3:** `npx vitest run tests/unit/useSpeech.test.ts` — FAIL (module missing).
- [ ] **Step 4:** `useSpeech.ts` — module scope: `const supported = ref(false)`, `const speaking = ref(false)`, `const activeId = ref<string | null>(null)`, `let initialised = false`, `let voices: SpeechSynthesisVoice[] = []`. `function synth(): SpeechSynthesis | undefined { const s = (globalThis as { speechSynthesis?: SpeechSynthesis }).speechSynthesis; return s && typeof s.speak === 'function' ? s : undefined }`. `export function pickVoice(list, lang = 'en-US')` normalises `lang.replace('_', '-').toLowerCase()` and walks the six-step order of design §6 "Voice pick" (the requested `lang` first, then `en-gb`, then any `en-*`, local before remote at each step). `function refreshVoices()` reads `getVoices()`; `supported = !!synth() && (voices.length === 0 || voices.some(isEnglish))`. `export function ensureSpeechInit()` — idempotent; `refreshVoices()`; `addEventListener('voiceschanged', refreshVoices)` once (guard `typeof addEventListener === 'function'`, fall back to `onvoiceschanged`). `export function useSpeech()` returns `{ supported: readonly(supported), speaking: readonly(speaking), activeId: readonly(activeId), speak, stop }` (`speak`/`stop` stubs for now). `export function __resetSpeechForTests()` resets every module variable and removes the listener.
- [ ] **Step 5:** `npx vitest run tests/unit/useSpeech.test.ts` — PASS; `npm run lint && npm run typecheck`.
- [ ] **Step 6:** Commit: `learn: useSpeech singleton — support detection and voice pick`.

### Task 3: `speak` / `stop`, sentence queue, errors, first-time hint (design §3 "What each placement speaks", §4 "Synthesis error" / "First-time", §5, §6 `useSpeech`)

**Files:** `frontend/composables/useSpeech.ts`, `frontend/tests/unit/useSpeech.test.ts`.

- [ ] **Step 1 (tests first):** extend `useSpeech.test.ts` (fake installed, `ensureSpeechInit()` called; `vi.useFakeTimers()` where timing matters; `vi.mock('~/composables/useRetroToast')` with a shared `show` spy):
  - `splitSentences` (exported, pure): `'One. Two! Three? Four… five'` → `['One.', 'Two!', 'Three?', 'Four…', 'five']`; a single word → `[word]`; surrounding/blank whitespace dropped; `''` → `[]`.
  - `speak('Hello there. How are you?', { id: 'a' })` → `synth.cancel` called once **before** the first `synth.speak`; two utterances enqueued synchronously in that call, texts `['Hello there.', 'How are you?']`, each `lang 'en-US'`, `rate 0.9`, `voice` = the picked voice; `activeId` is `'a'` immediately (the chip turns "Dừng" within the tap, design §5).
  - `speaking` becomes `true` on the first `onstart`; with no `onstart`, it becomes `true` after 250 ms (`vi.advanceTimersByTime(250)`); after the last utterance's `onend`, `speaking` is `false` and `activeId` `null`; after only the first `onend`, still `true`.
  - `speak(text, { rate: 1.2, lang: 'en-GB' })` passes them through.
  - voices `[]` → the utterance has `lang 'en-US'` and `voice` `null`/unset.
  - `stop()` → `cancel` called, `speaking` `false`, `activeId` `null`; the fake's `canceled` errors fired by that cancel do **not** toast and do **not** set `speaking` back; a pending 250 ms fallback does not fire afterwards (`vi.getTimerCount()` 0).
  - `speak('A.', {id:'a'})` then `speak('B.', {id:'b'})` → `activeId 'b'`; stale `onend`/`onerror('interrupted')` of `'A.'` leave `'b'` active.
  - `fail('synthesis-failed')` on the first of three queued sentences → `speaking false`, `activeId null`, `show` called once with `('Tớ chưa đọc được. Thử lại nhé.', 'ember')`, and a second `fail` on the next queued sentence does not toast again (once per tap).
  - `speak` with the API missing is a no-op (no throw).
  - First-time hint: `maybeShowSpeakHint()` with `localStorage['aelp.speakHint']` absent → `show('Chạm loa, tớ đọc cho cậu nghe.')` (plain tone) and the key set; a second call → no toast; with `supported` false → no toast and no key; `localStorage` throwing → no throw, no toast.
- [ ] **Step 2:** `npx vitest run tests/unit/useSpeech.test.ts` — FAIL.
- [ ] **Step 3:** implement in `useSpeech.ts`: `let generation = 0`, `let fallback: ReturnType<typeof setTimeout> | null = null`. `speak(text, { lang = 'en-US', rate = 0.9, id = 'speech' } = {})`: `const s = synth(); if (!s) return`; `stop()`-equivalent reset without clearing the new id (increment `generation`, `s.cancel()`, clear `fallback`); `const gen = ++generation`; `activeId.value = id`; `const parts = splitSentences(text)`; if none, reset and return; build one `new (globalThis.SpeechSynthesisUtterance)(part)` per part with `lang`, `rate`, `voice = pickVoice(voices, lang)` when non-null; `let errored = false`; `onstart` → `if (gen === generation) { speaking = true; clear fallback }`; last utterance `onend` → `if (gen === generation) reset()`; every `onerror(e)` → ignore when `gen !== generation` or `e.error` is `interrupted`/`canceled`; else `reset()` and, if `!errored`, `errored = true; useRetroToast().show('Tớ chưa đọc được. Thử lại nhé.', 'ember')`; `s.speak(u)` for every utterance in the same call; `fallback = setTimeout(() => { if (gen === generation) speaking.value = true }, 250)`. `stop()`: `generation++`, clear `fallback`, `synth()?.cancel()`, `speaking = false`, `activeId = null`. `splitSentences(text)`: `text.split(/(?<=[.!?…])\s+/).map(t => t.trim()).filter(Boolean)`. `maybeShowSpeakHint()`: module flag + `try { … localStorage … } catch {}` around the key read/write; only when `supported`.
- [ ] **Step 4:** `npx vitest run tests/unit/useSpeech.test.ts` — PASS; `npm run lint && npm run typecheck && npm run test:unit`.
- [ ] **Step 5:** Commit: `learn: useSpeech speak/stop — sentence queue, stale-event guard, error toast, first-time hint`.

### Task 4: `SpeakButton.vue` (design §1, §3 "`SpeakButton` chip", §4, §5, §6 `SpeakButton`, §7)

**Files:** `frontend/components/learn/SpeakButton.vue` (new), `frontend/tests/unit/speakButton.test.ts` (new).

- [ ] **Step 1 (tests first):** `speakButton.test.ts` (`__resetSpeechForTests` + `installFakeSpeech()` in `beforeEach`; `await nextTick()` after mount so `onMounted` decided `supported`):
  - API missing → `wrapper.find('button').exists()` is `false` and `wrapper.html()` is an empty comment (not `disabled`, no `hidden`/`display:none`); voices known with no `en` → same.
  - supported → one `<button type="button">` with visible text "Nghe", `aria-label="Nghe"`, `aria-pressed="false"`, classes include `h-11` and `min-w-[88px]`; a `PixelArt` with `GLYPHS.speaker` rows at `size 32`; no `speakerWaves` rendered.
  - click → `synth.speak` called with the prop `text` (joined across utterances it equals the input sentences), `rate 0.9`; the button now reads "Dừng", `aria-label="Dừng"`, `aria-pressed="true"`, border class `border-growth`, and a second `PixelArt` with `GLYPHS.speakerWaves` and class `retro-blink` — all synchronously after `await trigger('click')` (the ≤ 100 ms rule is "same tick").
  - second click → `synth.cancel` called, text "Nghe", `aria-pressed="false"`; `start`/`stop` events emitted in order.
  - two chips A and B mounted together: click A then click B → A reads "Nghe", B reads "Dừng", only one `aria-pressed="true"` (Acceptance 4).
  - unmount while active → `cancel` called; unmount while another chip is active → that chip keeps speaking.
  - `setProps({ text: 'other' })` while active → `cancel` called and the chip reads "Nghe"; while idle → no `cancel`.
  - click bubbles: mount inside a parent `<div @click="spy">` → `spy` not called (`@click.stop`).
  - Space and Enter are native button activation — assert the element is a real `<button>` (no `div role=button`) and has the `focus-visible:outline-torch` class.
  - first mount with `supported` → the hint toast shown once (`show` spy), a second chip mounted after it → not again.
- [ ] **Step 2:** `npx vitest run tests/unit/speakButton.test.ts` — FAIL.
- [ ] **Step 3:** `SpeakButton.vue`:
  - props `text: string`, `id?: string` (default: a per-instance id — `useId()` is available in Vue 3.5; if the unit-test build lacks it, a module counter `speak-${++n}`), `lang = 'en-US'`, `rate = 0.9`; emits `start`, `stop`.
  - `const { supported, activeId, speak, stop } = useSpeech()`; `const active = computed(() => activeId.value === myId)`; `onMounted(() => { ensureSpeechInit(); maybeShowSpeakHint() })`; `watch(() => props.text, () => { if (active.value) stop() })`; `onBeforeUnmount(() => { if (active.value) stop() })`; `function toggle() { if (active.value) { stop(); emit('stop') } else { speak(props.text, { lang: props.lang, rate: props.rate, id: myId }); emit('start') } }`.
  - template: `<button v-if="supported" type="button" class="sb relative inline-flex h-11 min-w-[88px] items-center gap-2 rounded-sm border-2 bg-ground-2 px-2 font-display text-[20px] leading-6 text-ink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch" :class="active ? 'border-growth' : 'border-line-lit'" :aria-pressed="active ? 'true' : 'false'" :aria-label="active ? 'Dừng' : 'Nghe'" @click.stop="toggle">` → `<span class="relative inline-block h-8 w-8">` with `<PixelArt :rows="GLYPHS.speaker" :palette="palette" :size="32" />` and, `v-if="active"`, `<PixelArt :rows="GLYPHS.speakerWaves" :palette="palette" :size="32" class="retro-blink absolute inset-0" />` → `<span>{{ active ? 'Dừng' : 'Nghe' }}</span>`. `palette` maps `PALETTE` through `tokens` exactly as `Badge.vue` does (earned branch). Scoped style: the `RetroButton` pressed recipe at 44 px — `box-shadow: 0 4px 0 0 <line-dim>`, `:active` → `translateY(2px)` + `0 2px 0 0`, no transition (design §3 "Pressed drop … within 100 ms"). Reduced motion needs no code: `retro.css` already zeroes `retro-*` animations, leaving one static arcs frame (design §4 "Reduced motion").
- [ ] **Step 4:** `npx vitest run tests/unit/speakButton.test.ts` — PASS; `npm run lint && npm run typecheck && npm run test:unit`.
- [ ] **Step 5:** Commit: `learn: SpeakButton — the 44-px "Nghe" chip`.

### Task 5: wire the room — `ContentViewer` and `pages/learn/[id].vue` (design §2, §3 "Eyebrow row" / "What each placement speaks", §5, §6 `ContentViewer` / `pages/learn/[id].vue`)

**Files:** `frontend/components/learn/ContentViewer.vue`, `frontend/tests/unit/contentViewerSpeech.test.ts` (new), `frontend/pages/learn/[id].vue`, `frontend/tests/unit/learnPage.test.ts`.

- [ ] **Step 1 (tests first) — `contentViewerSpeech.test.ts`:** mount `ContentViewer` (register `AppButton`, `SpeakButton` globally as `learnPage.test.ts` does) with the fake installed:
  - `questions` content `[{id:'q1', prompt:'Choose the past tense of go.', options:{A:'goed',B:'went'}}, {id:'q2', prompt:'Second?', options:{A:'x'}}]` → the eyebrow row element (the parent of "Câu 1 / 2") has classes `flex`, `h-11`, `items-center`, `justify-between` and contains the chip; click the chip → every `spoken` utterance text joined equals `'Choose the past tense of go.'` and none contains `goed`/`went` (Acceptance 3).
  - `words` content `[{term:'Inquire', definition:'To ask for information'}]` → chip click speaks exactly `'Inquire'` (never the definition).
  - chip speaking, click "Tiếp tục" → `synth.cancel` called, the new item's chip reads "Nghe" (Acceptance 5), and `synth.speak` call count did not increase (Acceptance 6: advancing never speaks).
  - `raw` content → no chip.
  - API missing → no chip, and the eyebrow row still has `h-11` (Acceptance 7).
  - mounting never calls `synth.speak` (Acceptance 6).
- [ ] **Step 2 — `learnPage.test.ts` (extend; stub `onBeforeRouteLeave` with `vi.stubGlobal` next to the existing `useRoute`/`navigateTo` stubs, capturing its callback):** with a `questions` task and the fake installed —
  - start speaking via the chip, then `wrapper.unmount()` → `cancel` called.
  - start speaking, invoke the captured route-leave callback → `cancel` called.
  - start speaking, set `document.visibilityState` to `'hidden'` (`Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })`) and dispatch `visibilitychange` → `cancel` called; the existing wall-clock test still passes.
  - start speaking, reach the completable state, click "Hoàn thành" → `cancel` was called **before** `api.post` (compare `mock.invocationCallOrder`), and the posted `duration_seconds` equals the value the existing "completing posts the wall-clock elapsed" test asserts for the same clock (Acceptance 9).
- [ ] **Step 3:** `npx vitest run tests/unit/contentViewerSpeech.test.ts tests/unit/learnPage.test.ts` — FAIL.
- [ ] **Step 4 — `ContentViewer.vue`:** replace the eyebrow `<p v-if="content.kind !== 'raw'" class="text-sm text-mute">Câu … / …</p>` with `<div v-if="content.kind !== 'raw'" class="flex h-11 items-center justify-between">` holding that same `<p>` (text and classes unchanged) and `<SpeakButton v-if="speakText" :text="speakText" />`, where `const speakText = computed(() => props.content.kind === 'questions' ? props.content.questions[index.value]?.prompt ?? '' : props.content.kind === 'words' ? props.content.words[index.value]?.term ?? '' : '')`. In `next()`, call `stop()` (from `useSpeech()`) first. Nothing else in the file changes.
- [ ] **Step 5 — `pages/learn/[id].vue`:** `const { stop } = useSpeech()`; add `stop()` to the existing `onBeforeUnmount`; `onBeforeRouteLeave(() => { stop() })`; a new `function stopWhenHidden() { if (document.visibilityState === 'hidden') stop() }` registered on `visibilitychange` in `onMounted` and removed in `onBeforeUnmount` (beside `sync`, which stays); `complete()` calls `stop()` as its first statement after the guard. If no `<RetroToast />` is mounted in `app.vue` or a layout on your tree (`grep -rn '<RetroToast' frontend --include=*.vue`), mount one at the end of the page's `<main>` so the error and hint toasts are visible; if one already is, do not add a second.
- [ ] **Step 6:** `npx vitest run tests/unit/contentViewerSpeech.test.ts tests/unit/learnPage.test.ts` — PASS; `npm run lint && npm run typecheck && npm run test:unit`.
- [ ] **Step 7:** Commit: `learn: the "Nghe" chip on every question and word; speech stops on every exit`.

### Task 6 (gated): the retro learning-room items (design §3 "Eyebrow row" / "Passage sheet" / "What each placement speaks", §2 flip rule)

**Gate:** `test -f frontend/components/learn/ItemPassage.vue && test -f frontend/components/learn/ItemFlashcard.vue && test -f frontend/components/learn/ItemQuestion.vue` on the tree you started from. If it fails, skip this task entirely, state "Task 6 skipped — learning-room items not on main" in the `done` summary, and do not create those components.

**Files:** the three item components, the passage sheet (wherever the learning-room plan put it), their existing tests.

- [ ] **Step 1 (tests first):** in each item's test: the eyebrow row ("Đoạn văn" / "Câu {n} / {N}" / "Thẻ {n} / {N}") is `flex h-11 items-center justify-between` with a chip; `ItemPassage` chip speaks the whole `passage` (joined sentences equal the input after whitespace normalisation); `ItemQuestion` speaks `prompt` only; `ItemFlashcard` speaks `term` on the front **and** after flipping (the flip does not cancel, design §2), and a chip click does not flip (`@click.stop`); opening the passage sheet does not call `speak`, and its "Đoạn văn" eyebrow carries its own chip; every advance handler ("Tiếp tục", "Nhớ rồi", "Học lại", "Đọc xong") calls `cancel`.
- [ ] **Step 2:** run them — FAIL. **Step 3:** mount `<SpeakButton :text="…" />` in each eyebrow row as in Task 5 Step 4, and `stop()` first in each advance handler. If `ContentViewer` now delegates to these items, remove the Task 5 chip from `ContentViewer` so each item shows exactly one.
- [ ] **Step 4:** PASS; `npm run lint && npm run typecheck && npm run test:unit`. **Step 5:** Commit: `learn: the "Nghe" chip on the passage, the flashcard and the passage sheet`.

### Task 7: kit wording and CODEMAP (design §6 "Kit additions")

**Files:** `harness/UI-KIT.md`, `harness/CODEMAP.md`.

- [ ] **Step 1:** `harness/UI-KIT.md` "Motion budget" — replace the bullet `Sound-free by default; no audio API in scope.` with `Sound-free by default: no sound effects. The only audio is speech synthesis behind an explicit tap (\`SpeakButton\`, \`useSpeech\`); nothing speaks without a user gesture.`; add `speaker` / `speakerWaves` wherever the kit lists the glyphs, and `SpeakButton` (44-px labelled icon chip, `RetroButton` secondary recipe) to its component list.
- [ ] **Step 2:** `harness/CODEMAP.md` `shell` bullet — one sentence: `/learn/:id` read-aloud: `composables/useSpeech.ts` (module singleton over `speechSynthesis`; `speak` only from a click; voice `en-US` > `en-GB` > any `en`, local first; `rate 0.9`; sentence queue; empty voice list = unknown, not unsupported) + `components/learn/SpeakButton.vue` in each item's eyebrow row; the page stops speech on unmount, route leave, `complete()` and tab hidden; tests use `tests/unit/fakeSpeech.ts` (happy-dom has no `speechSynthesis`).
- [ ] **Step 3:** `python3 tools/harness/cli.py validate` (from the repo root). Commit: `docs: UI-KIT audio exception and CODEMAP for read-aloud`.

## Notes

- **Why a module singleton, not a Pinia store:** the design (§5 "Timer", §6) says no store field is added; there is exactly one `speechSynthesis` per window, and a store would invite persisting a state that is meaningless across reloads.
- **Why enqueue every sentence in the tap:** the gesture rule is satisfied by the synchronous `speak` calls inside the click handler; chaining the next sentence from `onend` would be a `speak` outside a gesture on iOS. `speechSynthesis` queues natively; `cancel()` clears the whole queue, which is what `stop()` needs.
- **Unconfirmed on real hardware:** iOS Safari's empty `getVoices()` and skipped `onstart`, Chrome's ~15 s single-utterance cut-off — design §0 records them as assumptions from the idea's research links; unit tests prove the handling, not the devices. The reviewer's manual check (design §8 "Review checks") on a phone is the only real evidence; say in the `done` summary whether you ran one.
- **`useId`:** Vue 3.5.43 ships `useId()`; it needs an app instance, which `mount()` provides. If it misbehaves under `@vue/test-utils`, use a module counter — the only requirement is uniqueness per instance.
- **Out of scope:** speaking option text or the flashcard back (design §0.4), a rate/voice setting, highlighting the spoken word, sound effects, the review mode's chip on a completed task (today's page does not render `ContentViewer` for a completed task; the learning-room plan brings the review mode, and Task 6 covers it if that plan landed first).
- **Follow-up owed:** the retro learning-room plan (`harness/designs/retro-learning-room.md`, not yet planned) must mount `SpeakButton` in `ItemPassage`, `ItemQuestion`, `ItemFlashcard` and the passage sheet if Task 6 was skipped here — the evaluator writing that plan should cite this design's §3.

## Verification

```
cd frontend
npm run lint && npm run typecheck && npm run test:unit && npm run build
npx vitest run tests/unit/useSpeech.test.ts tests/unit/speakButton.test.ts tests/unit/contentViewerSpeech.test.ts tests/unit/learnPage.test.ts tests/unit/pixelArt.test.ts
grep -rn '[^.A-Za-z]speak(' components composables pages utils --include=*.vue --include=*.ts | grep -v 'function speak'   # one hit: SpeakButton.vue's click handler
grep -rn 'speechSynthesis' components pages utils --include=*.vue --include=*.ts                                     # empty: only composables/useSpeech.ts touches the engine
grep -n 'no audio API in scope' ../harness/UI-KIT.md                                                                  # empty: kit wording updated
git diff origin/main --stat -- stores/quest.ts utils/content.ts                                                     # empty: timer store and classifier untouched
cd .. && python3 tools/harness/cli.py validate
git push -u origin harness/2026-09-27-medium-hear-it-read-aloud-for-passages-questions-and-vocabulary-wit
```

Design Acceptance (from `harness/designs/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md`, verbatim; the reviewer checks each — items or clauses that need the learning-room components are not-yet-applicable when Task 6 was skipped):

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

Manual (design §8 "Review checks", on `npm run dev` or the built preview in a Chromium with an English voice): tap "Nghe" on a question, then "Tiếp tục" → silence and a "Nghe" chip on the next item; complete a task while speaking → the hub is silent; DevTools "Emulate prefers-reduced-motion" → static arcs; `delete window.speechSynthesis` in a startup script (or a browser without it) → no chip and the row keeps its height; posted `duration_seconds` equals the timer with and without listening. Note that the page's back link reads "‹ Quay lại" on today's tree — "‹ Trại" arrives with the learning-room restyle; the behaviour is the same.
