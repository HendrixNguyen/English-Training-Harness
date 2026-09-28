# Design: Tap a hard word — the reading passage, with Vietnamese glosses (`/learn/:id`)

**Idea:** `harness/ideas/2026-09-27-run-01/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md` (and its `## Evaluation`: the passage is rendered nowhere today — that is job one; the glossary rides on it).
**Inherits:** `harness/designs/frontend-shell.md` §2.4 (timer, completion post), `harness/designs/retro-learning-room.md` (the "Đoạn văn" item, "Đọc xong →", "Xem lại đoạn", `ItemPassage`) — this doc builds that passage item for real and adds glosses; nothing else in the room changes. Compatible with `harness/designs/hear-it-read-aloud-for-passages-questions-and-vocabulary-wit.md` (§6 "Read-aloud compatibility").
**Kit:** `harness/UI-KIT.md` v2 + `harness/designs/retro-kit.md` (`RetroPanel`, `RetroButton`, `PixelArt` + `GLYPHS.cross`, `useReducedMotion`, the `tokens` export).
**Spec wireframe:** frontend spec §7.3. Kept: slim top bar, "Câu n / N" eyebrow, one question with lettered options, one bottom button. Departure: a reading task opens on a passage item before its questions (the wireframe shows only a question), as `retro-learning-room.md` already decided.

**Designed against the post-merge tree** (the four gating branches on top of `origin/main`): `utils/content.ts` = `words | questions | raw` (`pick()` finds arrays only, top level or under `content`); `components/learn/ContentViewer.vue` = v1 (index, `selected`, "Câu {n} / {N}", `AppButton` "Tiếp tục"/"Gửi đáp án"/"Hoàn thành", emits `answer`/`finished`); `pages/learn/[id].vue` from the task-timer branch (v1 `AppCard` host, timer never touched by `ContentViewer`); retro kit components exist under `components/retro/` but the room is not restyled yet and `ItemQuestion`/`ItemFlashcard` exist nowhere. Reading content per the typed-content branch: `{passage: 200..2000 runes, questions[3..5]}`, plus this idea's optional `glossary[5..8]{term, meaning_vi, definition}`.

## 0. Research
- **Learner's job:** read a short English passage at their level, understand it well enough to answer 3–5 questions about it — without leaving the app to translate.
- **The moment that earns the next minute:** the tap on a word they half-know. A dotted word opens a card with the Vietnamese meaning in one step; they read on, and the next question lands. The passage stops being a wall and becomes something they can finish.
- **What today does wrong:** (1) `classifyContent` sees `questions` in reading content and renders them — the `passage` is never shown (`grep -rn passage frontend` hits tests only), so one task in three asks about a text the learner cannot see. (2) Even with a passage, unknown words send the learner to Google Translate mid-timer on a phone — the exit the distraction-free room (spec §7.3) exists to prevent.
- **Open questions, answered:**
  1. *Does the passage need the full retro room?* No. `ItemPassage` is built in retro tokens now and dropped into the v1 `ContentViewer`; when the learning-room plan restyles the room it reuses `ItemPassage` unchanged. A dark `RetroPanel` inside the v1 `AppCard` is an accepted interim look (§8).
  2. *Look back while answering — modal sheet or inline?* Inline. A "Xem lại đoạn" toggle above each question expands the same passage panel in place (`aria-expanded`). No focus trap, no `<dialog>` quirks in happy-dom, and the question stays on the same page. It keeps the name "passage sheet" for the read-aloud plan's references.
  3. *Where does the gloss card go?* Inside the passage panel, pinned **below the scroll frame** — never inside the scrolling text. A card inside the frame would scroll away from the word or push the text; below the frame it is always on screen next to what was tapped.
  4. *Inflected forms* (glossary "negotiate", passage "negotiated")? The match must start at a word boundary; the mark extends to the end of that word, so "negotiated" is marked with the "negotiate" gloss. A term found only mid-word ("art" in "start") is not marked.
  5. *Bad glossary data* (old roadmap, model drift)? The client never trusts it: entries with a blank field, duplicate terms (case-insensitive) or terms not found in the passage are dropped; at most 8 are kept. Zero left → the passage renders plain, no caption. The backend validator is the real gate; this is the backstop.
  6. *Counting:* the passage is not a question, so its eyebrow is "Đoạn văn" and the questions count "Câu 1 / 3 … 3 / 3" (not "Câu 2 / 4"). Small departure from `retro-learning-room.md` review note "the passage item counts in n / N" — a passage labelled "Câu" would misname it.

## 1. The signature
**Words you might not know are quietly underlined, and one tap tells you in Vietnamese.** The passage reads like a book page in Nunito 18/30 on the night panel; five to eight words carry a dotted `line-lit` underline. Tap "negotiated" and a small inset card appears under the text: **negotiate** · *đàm phán, thương lượng* · "to discuss something to reach an agreement." Tap it again and it is gone; tap another word and the card switches. The timer never pauses, the text never shifts under the finger, and the same marks are there when the learner flips back to the passage from a question.

## 2. Flow
Entry unchanged (hub quest → `/learn/:id`). For a `reading` task: **item 0 = passage** (eyebrow "Đoạn văn", button "Đọc xong →") → items 1..Q = questions (eyebrow "Câu k / Q", options, button as today) → last answer → `finished` → the page's "Hoàn thành" as today. On every question a secondary toggle "Xem lại đoạn" expands the passage above the question; its open/closed state carries over to the next question within the task. "Đọc xong →", "Tiếp tục" and toggling the passage closed all close any open gloss card. Back ("‹ Quay lại") and re-entry behave as today (the viewer restarts at item 0; answers and timer are the page/store's). Nothing on the hub, roadmap or chest changes.

## 3. Layout (mobile-first, `max-w-md`; desktop same column)
```
│ ‹ Quay lại                 Thời gian: 04:12 │ page top bar (unchanged)
│ ĐOẠN VĂN                        [ slot ]     │ A eyebrow row: flex h-11 items-center justify-between
│ Chạm vào từ có gạch chấm để xem nghĩa.       │ B caption (only when ≥1 mark), body 14/20 ink-1
│ ┌───────────────────────────────────────┐   │ C RetroPanel (no speaker, no portrait)
│ │═══════════════════════════════════════│   │   scroll frame: border-y-2 line-dim, max-h-[50vh]
│ │ Last spring the two companies began   │   │   passage: font-body 18/30 ink-0, whitespace-pre-line
│ │ to negotiate a new contract. The ...  │   │   GlossTerm: dotted underline line-lit
│ │    ˙˙˙˙˙˙˙˙˙                          │   │   (dots = the marked word)
│ │═══════════════════════════════════════│   │
│ │ ┌───────────────────────────────────┐ │   │ D GlossCard (when open): inset, ground-2
│ │ │ negotiate                 [✖]     │ │   │   term body 17 700 ink-0 · close 44×44
│ │ │ đàm phán, thương lượng            │ │   │   meaning_vi body 17 ink-0
│ │ │ to discuss something to reach an  │ │   │   definition body 14/20 ink-1
│ │ │ agreement.                        │ │   │
│ │ └───────────────────────────────────┘ │   │
│ └───────────────────────────────────────┘   │
│ [            ĐỌC XONG →              ]      │ E ContentViewer's bottom button (v1 AppButton today)
```
Question item of a reading task:
```
│ CÂU 2 / 3                       [ slot ]     │ ContentViewer's eyebrow row (read-aloud chip lands here)
│ [          XEM LẠI ĐOẠN            ]        │ F RetroButton secondary block, aria-expanded
│ (open → the passage panel C+D, frame max-h-[40vh], no caption, no button)
│ "What did the companies agree on?"          │ question + options exactly as today
```
- **A** — `ItemPassage` eyebrow: `font-display text-base leading-5 uppercase tracking-[0.05em] text-ink-1` "Đoạn văn"; right side `<slot name="eyebrow-action" />`, empty here. The row is `h-11` whether or not the slot is filled.
- **C** — `RetroPanel` (plain tone). Scroll frame: `max-h-[50vh] overflow-y-auto overscroll-contain border-y-2 border-line-dim py-2`, `tabindex="0"`, `role="region"`, `aria-label="Đoạn văn"` (keyboard-scrollable). Text `font-body text-lg leading-[30px] text-ink-0 whitespace-pre-line` — the passage's own line breaks survive; no markdown.
- **GlossTerm** — inline `<button type="button">` inheriting font/size; `text-ink-0 underline decoration-dotted decoration-2 underline-offset-4 decoration-line-lit`; active: `bg-ground-2` + `decoration-solid`; focus: 2-px `torch` outline, 2-px offset. `aria-label="Giải nghĩa {marked text}"` (contains the visible word — label-in-name), `aria-expanded`, `aria-controls` = the card's id.
- **D** — `GlossCard`: `mt-3 border-2 border-line-lit bg-ground-2 p-3`, `rounded-none`, `role="region"`, `aria-live="polite"`, `aria-label="Giải nghĩa"`. Header row `flex items-start justify-between gap-2`: `term` (`font-body text-[17px] font-bold`) left; right a 44×44 close button (`bg-ground-1 border-2 border-line-lit rounded-sm`, `PixelArt GLYPHS.cross` 16 px in `ink-0` palette override, `aria-label="Đóng giải nghĩa"`). Then `meaning_vi` (`text-[17px] text-ink-0`), then `definition` (`text-sm text-ink-1`). All three fields are text, never HTML.
- **F** — `RetroButton variant="secondary" block` "Xem lại đoạn" / "Ẩn đoạn văn", `aria-controls` the panel id. Only rendered for `reading` questions.
- **Data:** `content.passage`, `content.glossary` (sanitised), `content.questions` from `classifyContent(task.content_json)`.

## 4. States
- **Loading / empty / not today / posting error / offline banner:** the page's, unchanged. Offline: `content_json` is in the store's cached daily quest, so passage and glosses work with no network.
- **Passage with glossary:** caption B + marks; card closed on entry.
- **Passage without glossary** (pre-feature roadmaps, or every entry dropped by sanitising): plain text, no caption, no card, no buttons inside the text.
- **Reading content with a blank or missing `passage`:** falls through to today's `questions` kind (no passage item, no toggle) — never `raw`, never an empty panel.
- **Card open:** one card at a time; the active term shows `bg-ground-2` + solid underline.
- **First-time:** the caption B is the whole onboarding — shown on every passage that has marks (no storage key, never a toast).
- **Done / review:** unchanged (completed tasks show the page's summary, not the viewer).
- **Reduced motion:** nothing here animates; the card appears and disappears in one step in both modes. No `retro-*` class is used.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| Tap a closed GlossTerm | term active; card shows its gloss | focus stays on the term; the card is announced once (`aria-live`) | nothing |
| Tap the active GlossTerm | term inactive; card removed | — | nothing |
| Tap another GlossTerm | previous term inactive; card switches to the new gloss | — | nothing |
| Card "✖" | card removed | focus returns to the term that opened it | nothing |
| `Esc` with focus inside the passage panel | card removed | focus to its term | nothing |
| "Đọc xong →" | button pressed | card closed; item 1 (first question) | viewer index |
| "Xem lại đoạn" / "Ẩn đoạn văn" | button drops; panel shown/hidden in one step | closing also closes the card; label and `aria-expanded` flip | viewer-local `passageOpen` (kept across questions of this task) |
| Question options / "Gửi đáp án" / "Tiếp tục" | as today | "Tiếp tục" also closes the card | `answers[id]` (page) |
Timer: untouched — `ContentViewer` never reads or writes it; card time is task time. Keyboard: Tab reaches each GlossTerm in reading order; Enter/Space toggles like a tap.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `utils/content.ts` | existing | adds `Gloss {term, meaning_vi, definition}` and `{ kind: 'reading', passage: string, questions: Question[], glossary: Gloss[] }` | order: `words` → `reading` (non-blank string `passage`, top level or under `content`, same rule as `pick`) → `questions` → `raw`. `questions` may be `[]`. `glossary` = `sanitizeGlossary(raw, passage)` |
| `utils/gloss.ts` | new | `sanitizeGlossary(raw: unknown, passage: string): Gloss[]`; `segmentPassage(passage: string, glossary: Gloss[]): Segment[]` with `Segment = { text: string } \| { text: string, gloss: number }` | pure, no Vue. Sanitise: keep objects with three non-blank strings, trim, drop duplicate terms (case-insensitive, first wins), drop terms `segmentPassage` cannot place, cap 8. Segment: per term, first case-insensitive occurrence whose start is a word boundary (prev char not `\p{L}\p{N}`), spaces in the term match `\s+`, regex-escaped; the mark extends to the word's end (`[\p{L}\p{N}'’-]*`); longer terms placed first; an occurrence overlapping an already-placed mark is skipped for that term's next occurrence; unplaceable → no mark. Concatenating every segment's `text` returns the passage exactly |
| `components/learn/ItemPassage.vue` | new (named per `retro-learning-room.md`) | props `passage: string`, `glossary: Gloss[]`, `mode: 'item' \| 'sheet' = 'item'`; slot `eyebrow-action`; exposes `closeGloss()` | owns `openIndex: number \| null`. `item`: caption (when marks), frame `max-h-[50vh]`. `sheet`: no caption, frame `max-h-[40vh]`. Never renders a bottom button — `ContentViewer` owns it |
| `components/learn/GlossTerm.vue` | new | props `text`, `active`, `controls` (card id); emits `toggle` | inline `<button>`; see §3 |
| `components/learn/GlossCard.vue` | new | props `gloss: Gloss`, `id`; emits `close` | see §3; the one "word card" look (term · meaning · definition). When the retro room builds `ItemFlashcard`, its back face should reuse this block, not restyle it |
| `components/learn/ContentViewer.vue` | existing | unchanged props/emits | `reading` kind: the passage is item 0 (implementation may use `index = -1`) (eyebrow "Đoạn văn" comes from `ItemPassage`; ContentViewer hides its own "Câu" line there), button "Đọc xong →"; `0..Q-1` are the questions via the existing question template with "Câu {k} / {Q}" and, above the prompt, the "Xem lại đoạn" toggle + `ItemPassage mode="sheet"`. `words`/`questions`/`raw` render exactly as today. A `reading` task with `questions: []` finishes on "Đọc xong →" |
| `RetroPanel`, `RetroButton`, `PixelArt` | kit | — | no change |
**Kit additions (land only through the plan):** `ItemPassage` (proposed by `retro-learning-room.md`; built here without its `frame=scroll` `RetroPanel` variant — the frame is two classes local to `ItemPassage`, so no kit component changes); `GlossTerm` (the inline mark — the kit has no in-text control; dotted underline in `line-lit` because the kit forbids colour-only meaning and `line-dim` may not carry meaning); `GlossCard` (the inset word card: `ground-2` fill + `line-lit` border — the kit's "raised/inset" role). Kit-rule exception, recorded: GlossTerm is an inline target inside running text, below the kit's 44-px floor — WCAG 2.2 SC 2.5.8 exempts inline targets constrained by line height; marks are sparse (≤ 8 per passage) so neighbours never collide. No token, no font, no radius added.

**Read-aloud compatibility** (plan `2026-09-27-hear-it-read-aloud-…`, design §3): both eyebrow rows this plan touches are `flex h-11 items-center justify-between` with the right end free — `ItemPassage`'s `eyebrow-action` slot (item and sheet) and ContentViewer's "Câu k / Q" row, which the read-aloud plan's Task 5 already fills for every question. Whichever lands second adapts: *this plan second* → if `components/learn/SpeakButton.vue` exists, mount `<SpeakButton :text="passage" />` in both `eyebrow-action` slots (one line each) and keep ContentViewer's question chip intact; *read-aloud second* → its gated Task 6 finds `ItemPassage.vue` and mounts the chip in the slot (item + sheet); `ItemQuestion`/`ItemFlashcard` still do not exist, so questions and words keep the Task 5 chip in ContentViewer. The gloss card never speaks and has no chip; a later card-level "Nghe" goes left of "✖" in the card header (information only). Speech is not stopped by opening or closing a card.

## 7. Copy
Eyebrows: "Đoạn văn" · "Câu {k} / {Q}". Caption: "Chạm vào từ có gạch chấm để xem nghĩa." Buttons: "Đọc xong →" · "Xem lại đoạn" · "Ẩn đoạn văn". Accessible names: "Giải nghĩa {word}" (term) · "Đóng giải nghĩa" (✖) · "Giải nghĩa" (card region) · "Đoạn văn" (scroll frame). The term, `meaning_vi`, `definition` and the passage are content, shown as given. No companion line — the room's companion voice belongs to answer feedback.

## 8. Self-critique
- **Traded away:** a modal sheet for an inline toggle — cheaper and accessible, but on a long passage the learner scrolls the page between passage and question. The 40-vh frame limits that. A card beside the word (popover) would be closer but covers text and fights the scroll frame on a 360-px phone.
- **Interim look:** until the retro room plan lands, a dark `RetroPanel` passage sits inside the v1 `AppCard` with v1 question buttons. Accepted: building the passage in v1 would be thrown away within days; the panel carries its own ground and ink so it is legible on either scheme.
- **No gloss persistence:** tapped words are not saved or fed to review. That is the "answers are remembered" idea's territory.
- **Executor traps:** the `reading` check must run before `questions` (reading content has both); `passage` is a string, so `pick()` (arrays only) cannot find it — add a string variant; never `v-html` the passage or glosses (model text); the segment texts must rejoin to the exact passage (test it) or words vanish; `\p{L}` needs the `u` flag; the `index = -1` passage item must not emit `answer`; "Đọc xong →" on a zero-question reading task must emit `finished`; closing the sheet must also clear `openIndex` or a hidden card stays "open" for screen readers; GlossTerm inherits line-height — no padding that changes the 30-px rhythm; `learnPage.test.ts`, `content.test.ts` and existing ContentViewer behaviour for `words`/`questions`/`raw` must keep passing unchanged.
- **Review checks:** a real reading task (fixture) shows the passage before its questions; a passage with "Negotiated," at a sentence start and a comma after still marks it; "start" is not marked for the term "art"; with the card open, the timer still advances each second; delete `glossary` from the fixture → plain passage, no caption.

## Acceptance
1. A `reading` task (`passage` + `questions`) opens on a "Đoạn văn" item that shows the whole passage, with its line breaks, in `font-body` 18/30 inside a `RetroPanel` scroll frame; its questions follow, counted "Câu 1 / Q … Q / Q", and no reading question is ever shown before the passage item.
2. On every question of a reading task, "Xem lại đoạn" expands the same passage (with its marks) above the question and "Ẩn đoạn văn" hides it, within 100 ms, with `aria-expanded` reflecting the state; the open/closed choice carries to the next question of the task.
3. With a glossary, only the first valid occurrence of each term (case-insensitive, starting at a word boundary, punctuation-adjacent allowed, extended to the end of the word) is marked with a dotted underline, at most 8 marks; a term found only inside another word, a blank or duplicate entry, or a term absent from the passage is not marked; the rendered text equals the passage exactly.
4. Tapping a marked word shows, within 100 ms and below the scroll frame, one card with the term, its `meaning_vi` and its `definition`; tapping the same word closes it, tapping another switches it, "✖" or `Esc` closes it and returns focus to the word.
5. Each marked word is a keyboard-reachable button named "Giải nghĩa {word}" with `aria-expanded` and `aria-controls`; the card is a polite live region; the scroll frame is focusable and keyboard-scrollable.
6. A passage without a glossary (or with none left after sanitising) renders as plain text with no caption, no buttons in the text and no card.
7. "Đọc xong →", "Tiếp tục" and hiding the passage all close an open card; the timer keeps counting while a card is open and `ContentViewer` never touches it; the posted `duration_seconds` and answers are unchanged by glossary use.
8. Reading content with a blank or missing `passage` renders as today's question list (never `raw`, never an empty panel); `words`, `questions` and `raw` content render exactly as before.
9. Passage and glossary text are rendered as text (no `v-html`); no new token, font, radius or animation is added, and nothing on the screen animates under either motion setting.
10. Both passage eyebrow rows (item and sheet) are `flex h-11 items-center justify-between` with an `eyebrow-action` slot on the right, so the read-aloud "Nghe" chip mounts there without layout change.
