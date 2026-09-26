---
type: feature
status: proposed
source: ideator
run: 2026-09-27-run-01
order: 4
---
# Tap a hard word: Vietnamese glosses on every reading passage so the learner never leaves the room to translate

## Why
The reading task is a 200–2000-character English passage "at the learner's level" (typed-content plan). At level means some words are unknown by design; today the only way to look one up is to leave the PWA for Google Translate. The learning room is specified as *distraction-free* (frontend spec §7.3) and the product promises a learner "never has to struggle with English just to use the app" (`docs/PRODUCT.md`) — yet the one moment they are guaranteed to struggle sends them out of the app, on a phone, mid-timer. Each exit is a chance not to come back for the rest of the 30 minutes.

The research is unusually clear for this one: first-language glosses beat second-language glosses for vocabulary learning from reading (meta-analysis, 26 studies), and glossed reading roughly doubles the words retained versus unglossed reading in immediate tests. For a Vietnamese-first product this is the cheapest content improvement available: the glossary is generated with the passage in the same roadmap call (no runtime AI, no dictionary dependency, works offline like the rest of `content_json`) and reuses the word card the vocabulary task already renders.

## Expected output
User-visible:
- In a reading passage, 5–8 harder words are subtly marked (dotted underline in the retro kit's ink). Tapping one opens a small card under the passage: **word · nghĩa tiếng Việt · one-line English definition**; tapping the word again or another word closes/switches it. The card is the same component as the vocabulary task's word card, so it feels like one system.
- Passages generated before this ships have no glossary and simply show no marks. No setting, no new screen. The read-aloud idea (2026-09-26 run) can put its speaker on the same card — information only.
- Reading time is unchanged; the timer keeps counting while the card is open.

Technical (backend `airouter` content schema; frontend learning room):
- The typed-content schema's reading `content` gains an **optional** `glossary[5..8]{term, meaning_vi, definition}`; the roadmap prompt asks for it ("5–8 words a {level} learner is unlikely to know, each with a short Vietnamese meaning and a one-sentence English definition; each term must appear in the passage"). `validateContent` accepts a missing glossary (old roadmaps and any model that omits it still validate), and when present requires 5..8 entries, non-empty fields, unique terms, and each `term` to occur in the passage case-insensitively (else the roadmap is rejected like any other shape error — retry once → `ai_bad_output`). The swap prompt (idea 1) and regenerate reuse the same rule.
- Token cost: about 8 × 25 tokens per reading task × 28 days ≈ 5–6 k output tokens on top of the typed-content roadmap; the executor must re-check Gemini's `maxOutputTokens` and the thinking-budget inbox bug (`gemini-thinking-tokens-share-maxoutputtokens-…`) against a real generation and record the token count in the runtime proof.
- Frontend: `utils/content.ts` reading branch carries `glossary`; `ContentViewer` renders the passage as text nodes with the first occurrence of each term wrapped in a `GlossTerm` button (accessible name "Giải nghĩa {term}"), `GlossCard.vue` reuses the vocabulary word card. Designer places both inside `harness/designs/retro-learning-room.md`'s "Đoạn văn" panel. Reduced motion: none needed.
- Specs: backend spec §6.2 example `content_json` for a reading task gains `glossary`; CODEMAP `airouter` (schema line) and `shell`.
- Tests: schema table (missing ok, 4 or 9 entries rejected, term not in passage rejected, duplicate term rejected); a prompt test that the fixture roadmap carries glossaries; frontend unit tests for marking (first occurrence only, case-insensitive, punctuation-adjacent), open/close/switch, and a passage with no glossary rendering plain.
- Estimate: one working day (airouter ~3 h, frontend ~4 h, designer ~1 h). Depends on the typed-content backend plan (done, unmerged) for the schema file; lands after it.

## Evidence
- `harness/plans/2026-09-24-typed-task-content-with-answer-keys-…` — reading `{passage 200..2000 chars, questions[3..5]}`, `validateContent`, the prompt text this idea extends.
- `frontend/utils/content.ts`, `frontend/components/learn/ContentViewer.vue` — `words` / `questions` / `raw` today; `harness/designs/retro-learning-room.md` — the "Đoạn văn" panel.
- Frontend spec §7.3 (distraction-free room); `docs/PRODUCT.md` "The interface is in Vietnamese … never has to struggle with English just to use the app".
- `harness/ideas/_inbox/gemini-thinking-tokens-share-maxoutputtokens-…` (medium) — the output budget any content growth must respect.
- L1 vs L2 glosses meta-analysis (L1 more effective, 26 studies): https://journals.sagepub.com/doi/abs/10.1177/1362168820981394 ; glosses and incidental vocabulary learning from reading, meta-analysis: https://www.frontiersin.org/journals/language-sciences/articles/10.3389/flang.2026.1815571/full ; glossed vs non-glossed reading (45 % vs 27 % words learned): https://www.sciencedirect.com/science/article/abs/pii/S1041608015300212
