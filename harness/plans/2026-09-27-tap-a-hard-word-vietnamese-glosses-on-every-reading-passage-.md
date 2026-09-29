---
idea: harness/ideas/2026-09-27-run-01/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md
status: approved
priority: high
merged: false
order: 4
design: harness/designs/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md
---
# The reading passage finally renders — with tap-a-word Vietnamese glosses — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Design:** `harness/designs/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md` — every frontend task cites its sections; `## Verification` repeats its Acceptance list verbatim.
**Idea:** `harness/ideas/2026-09-27-run-01/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md`
**Goal:** A reading task opens on its passage (today the passage is rendered nowhere, so learners get questions about text they cannot see), the learner can look back at it from every question, and 5–8 harder words in the passage open a Vietnamese gloss card on tap — generated with the roadmap, no runtime AI, offline-capable.

**Scope:** backend `airouter` (optional `glossary` in the reading content schema + prompt) and frontend learning room (`utils/content.ts`, `utils/gloss.ts`, three `components/learn/*` components, `ContentViewer.vue`). No migration, no endpoint, no wire change outside `content_json`. **Estimate:** one working day (backend ~2–3 h, frontend ~5 h). **Branch:** `harness/2026-09-27-high-tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-`.

**Why high:** after the typed-content branch lands, one of the three daily tasks (reading) is unusable without the passage render — a happy-path break every learner hits daily (evaluation in the idea).

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically
`harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (`airouter/content.go`, `prompt.go` schema text),
`harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to` (`airouter/prompt.go`),
`harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` (onboarding persists typed content),
`harness/2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab` (`pages/learn/[id].vue`, `tests/unit/learnPage.test.ts`),
`harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`components/retro/RetroPanel.vue`, `RetroButton.vue`, `PixelArt.vue`, `GLYPHS.cross`, tokens).
```
for b in 2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a 2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to 2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a 2026-09-25-high-task-timer-keeps-counting-through-reloads-and-background-tab 2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
test -f backend/internal/airouter/content.go && test -f frontend/components/retro/RetroPanel.vue && echo files-ok
```
Any `MISSING` → stop and report `blocked`.

**Same-day overlap:** the read-aloud plan (`2026-09-27-hear-it-read-aloud-…`, draft) also edits `ContentViewer.vue`'s eyebrow row and gates its passage chip on `components/learn/ItemPassage.vue` existing. If it is already on `main`, mount `<SpeakButton :text="passage" />` in both `eyebrow-action` slots (design §6 "Read-aloud compatibility") and keep its question chip intact; if not, leave the slots empty. Either way, do not remove or move anything it added.

## Design decisions (fixed)

1. **Backend is lenient on the glossary, the client is strict.** A 28-day roadmap takes 53–78 s to generate; rejecting it (retry → `ai_bad_output`) because one gloss term is missing from its passage would turn a cosmetic slip into an onboarding failure. So `validateContent` rejects a reading task only when `glossary` is present **and not decodable** as an array of `{term, meaning_vi, definition}` strings; counts, blanks, duplicates and term-in-passage are asked for in the prompt and enforced by the client's `sanitizeGlossary` (design §6). This deliberately departs from the idea's "reject the roadmap" rule; record it in CODEMAP `airouter`. `content_json` is still stored verbatim.
2. **Prompt:** the reading shape becomes `{"passage": "string", "glossary": [{"term": "string", "meaning_vi": "string", "definition": "string"}], "questions": [QUESTION]}` and the rules sentence gains: "reading also has a glossary of 5-8 words from the passage that a {level} learner is unlikely to know, each with a short Vietnamese meaning (meaning_vi) and a one-sentence English definition; each term must appear in the passage exactly as written." Numbers come from constants `MinGlossary, MaxGlossary = 5, 8` like the other bounds.
3. **`SampleContent("reading")`** gains a valid 5-entry glossary whose terms occur in the sample passage, so every fixture exercises the new field.
4. **Token budget:** ~200 output tokens × 28 reading tasks ≈ 5–6 k extra; `GeminiMaxOutputTokens` is 32768. The executor records one real generation's token usage in the runtime proof (the drivers already log usage).

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/content.go` | `Gloss` type, `ReadingContent.Glossary`, `MinGlossary/MaxGlossary`, decode check, sample glossary |
| `backend/internal/airouter/glossary_test.go` (new) | schema table (Task 1) |
| `backend/internal/airouter/prompt.go` + new `prompt_glossary_test.go` | schema text and rule sentence (Task 2) |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` §6.2 | reading `content_json` example gains `glossary` |
| `frontend/utils/content.ts` + `tests/unit/content.test.ts` (or new `contentReading.test.ts`) | `reading` kind, `Gloss` (Task 3) |
| `frontend/utils/gloss.ts` (new) + `tests/unit/gloss.test.ts` | `sanitizeGlossary`, `segmentPassage` (Task 3) |
| `frontend/components/learn/GlossTerm.vue`, `GlossCard.vue`, `ItemPassage.vue` (new) + tests | Task 4 |
| `frontend/components/learn/ContentViewer.vue` + new `contentViewerReading.test.ts` | passage item, toggle (Task 5) |
| `harness/UI-KIT.md`, `harness/CODEMAP.md` | kit additions + 2.5.8 exception; `airouter`, `shell` paragraphs (Task 6) |

## Tasks

### Task 1: glossary in the reading schema (backend, tests first)
- [ ] `glossary_test.go`: reading content without `glossary` validates; with a valid 5-entry glossary validates; `glossary: "x"` (string), `glossary: [1,2]`, an entry with `term: 3` → rejected with a message naming the task ("module 1 day 2 task 2 glossary: …"); an entry with a blank field, 9 entries, a duplicate term, a term absent from the passage → **accepted** (decision 1 — assert explicitly so nobody "fixes" it later). `SampleContent("reading")` decodes with 5 glossary entries whose terms all occur in the passage.
- [ ] Implement in `content.go`: `type Gloss struct { Term string \`json:"term"\`; MeaningVI string \`json:"meaning_vi"\`; Definition string \`json:"definition"\` }`, `Glossary []Gloss \`json:"glossary,omitempty"\``; the existing `json.Unmarshal` into `ReadingContent` already fails on a wrong type — confirm the error path names the task; add the constants and the sample.
- [ ] `cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"`. Commit `feat(airouter): optional glossary on reading content`.

### Task 2: the prompt asks for it
- [ ] `prompt_glossary_test.go`: `RoadmapSchema` (or the system prompt, whichever carries the shape) contains `"glossary"`, `"meaning_vi"`, `"definition"` and `fmt.Sprintf("%d-%d words from the passage", MinGlossary, MaxGlossary)`; the existing `TestRoadmapSchemaStatesTheContentBounds` still passes.
- [ ] Edit `prompt.go` per decision 2 (keep the `var RoadmapSchema = fmt.Sprintf(...)` form the typed-content plan chose; append the two numbers to its argument list in order). Backend spec §6.2 reading example gains the field.
- [ ] Commit `feat(airouter): roadmap prompt requests 5-8 Vietnamese glosses per passage`.

### Task 3: `reading` kind and gloss helpers (design §6 `utils/content.ts`, `utils/gloss.ts`, §8 traps)
- [ ] Tests: `classifyContent` — reading content (top level and under `content`) → `kind: 'reading'` with passage, questions, sanitised glossary; blank/missing `passage` with questions → `questions`; vocabulary still `words`; garbage still `raw`; `questions: []` allowed. `sanitizeGlossary` — drops non-objects, blank fields, duplicates (case-insensitive, first wins), unplaceable terms, caps at 8, trims. `segmentPassage` — joining segment texts equals the passage for every case; "Negotiated," at sentence start marks for term "negotiate" (mark text "Negotiated"); "start" not marked for "art"; multi-word term with a line break between words matches; longer terms placed first; only the first occurrence marked; regex metacharacters in a term are escaped.
- [ ] Implement both (use the `u` flag for `\p{L}`).
- [ ] `cd frontend && npm run lint && npm run typecheck && npm run test:unit`. Commit `feat(frontend): reading content kind and gloss segmentation`.

### Task 4: `GlossTerm`, `GlossCard`, `ItemPassage` (design §1, §3 A–D, §4, §5, §7)
- [ ] Tests: `ItemPassage` item mode shows eyebrow "Đoạn văn", caption only when ≥ 1 mark, scroll frame `role=region` `aria-label="Đoạn văn"` `tabindex=0`, `whitespace-pre-line`; sheet mode has no caption and `max-h-[40vh]`; both eyebrow rows `flex h-11 items-center justify-between` with an `eyebrow-action` slot; tap → card with term/meaning_vi/definition; same term → closes; other term → switches; "✖" and `Esc` close and return focus to the term; `closeGloss()` exposed; no `v-html` anywhere (assert by rendering a passage containing `<b>` and finding it as text). `GlossTerm` name "Giải nghĩa {word}", `aria-expanded`, `aria-controls`. `GlossCard` `aria-live="polite"`, close named "Đóng giải nghĩa".
- [ ] Implement the three components with the classes in design §3 (no new token, radius or animation).
- [ ] Commit `feat(frontend): ItemPassage with tap-a-word gloss card`.

### Task 5: wire into `ContentViewer` (design §2, §3 "Question item", §5, §6 `ContentViewer`)
- [ ] `contentViewerReading.test.ts`: a reading fixture opens on the passage item with button "Đọc xong →" and no "Câu" line; next item shows "Câu 1 / 3"; the passage item never emits `answer`; "Xem lại đoạn" toggles `ItemPassage mode="sheet"` above the prompt with `aria-expanded`, label "Ẩn đoạn văn" when open, and the open state carries to the next question; hiding the sheet, "Đọc xong →" and "Tiếp tục" close any open card; a zero-question reading task emits `finished` on "Đọc xong →". Existing `words`/`questions`/`raw` tests and `learnPage.test.ts` pass unchanged; a learn-page test asserts the timer still advances while a card is open and the posted duration is unchanged.
- [ ] Implement (passage item as index `-1` or a separate `onPassage` flag — keep the existing templates for questions). If `components/learn/SpeakButton.vue` exists, mount it in both `eyebrow-action` slots (see *Same-day overlap*).
- [ ] Commit `feat(frontend): reading tasks open on their passage`.

### Task 6: docs
- [ ] `harness/UI-KIT.md`: `ItemPassage`, `GlossTerm`, `GlossCard` entries and the WCAG 2.5.8 inline-target exception (design §6 "Kit additions"). CODEMAP `airouter` (glossary field, decision 1) and `shell` (reading kind, new components). Commit `docs: UI kit and CODEMAP for passage and glosses`.

## Verification
```
cd backend && make test && go vet ./... && test -z "$(gofmt -l .)"
cd ../frontend && npm run lint && npm run typecheck && npm run test:unit && npm run build
grep -rn 'v-html' frontend/components/learn frontend/pages/learn   # empty
grep -n 'glossary' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../backend/internal/airouter/prompt.go
```
Runtime proof (local stack with one real provider key, scratch `COMPOSE_PROJECT_NAME=<slug>`): complete onboarding once; record the roadmap call's logged output-token count; `psql -c "select content_json->'glossary' from exercises where task_type='reading' limit 1"` shows 5–8 entries; open that reading task in the browser at 390×844 — passage first, tap a marked word, card shows the Vietnamese meaning; screenshot attached to the `done` summary.

Design Acceptance (verbatim):
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
