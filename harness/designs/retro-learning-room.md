# Design: Retro learning room — one item per dialogue box (`/learn/:id`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, with `harness/ideas/2026-09-25-run-01/a-session-a-learner-wants-to-finish-level-true-content-do-to.md` (do-to-complete, feedback per answer) and the inbox bug `harness/ideas/_inbox/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md` (typed content contract). **Depends on that bug's contract**: vocabulary `{words[{term, definition, example}], questions[]}`, reading `{passage, questions[]}`, practice `{questions[]}`, question `{id, prompt, options{A..D}, answer, explanation}`.
**Inherits:** `harness/designs/frontend-shell.md` §2.4 (timer, completion post, error/offline handling), `retro-hub.md` (where it returns to).
**Kit:** `harness/UI-KIT.md` v2.
**Spec wireframe:** frontend spec §7.3. Kept: slim top bar with back + timer, "Câu n / N", one question with lettered options, one bottom button. Departures: the button label changes with the item state ("Trả lời" → "Tiếp tục"); feedback is shown *before* moving on (the wireframe has no feedback step); completion is by the last item, not the timer.

## 0. Research
- **Learner's job:** answer this one thing, find out at once whether it was right and why, and move to the next — until the task ends.
- **The moment that earns the next minute:** the hit. Option line turns `growth`, "Trúng!" with the explanation, `XP +10` ticks in the top bar, the companion hops. A miss is nearly as good: "Trượt…", the correct option marked, the explanation — the learner learns something and wants the next one to be a hit.
- **What today does wrong:** `pages/learn/[id].vue:44-47,112` disables "Hoàn thành" until the countdown hits 0; `ContentViewer.vue` dumps `<pre>` JSON for 59/84 tasks and blank flashcards for the rest; no answer is ever graded. The owner stared at JSON for ten minutes per task.
- **Open questions, answered:** (a) *XP has no backend field.* Decision: XP is a **client-side session score** (hit +10, flashcard "Nhớ" +5, miss 0) shown in the top bar and in the chest, never persisted and never shown on the hub; the real rewards (minutes, HP, streak) come from the `POST /quests/progress` response. If XP should persist, that is a backend idea, filed separately. (b) *Flashcards have no right answer.* Self-rating: "Nhớ rồi" advances, "Học lại" sends the card to the back of the deck; the task ends when every card has been rated "Nhớ rồi" once. (c) *Reading passage length.* The passage is its own item (a scroll panel) before its questions; it counts as item 1 of N.

## 1. The signature
**Every item is a dialogue box the companion holds up.** The `RetroPanel` carries the companion's portrait and name, the prompt in `font-body`, and the options as a JRPG menu with a `▶` cursor. Answering does not navigate: the same box turns `growth` or `ember`, the explanation slides in under the options in one step, and the bottom button flips to "Tiếp tục". The learner never leaves the box; the box changes state. On the last item the box is replaced by a `Chest` that opens on the minutes, HP and XP.

## 2. Flow
Entry: hub tile or bottom button (`quest.taskById(id)`; cold store → `quest.load()` first). Inside: item 1 → … → item N → chest → "Về trại" → `/` with growth-moment before/after values. Back (top-left "‹ Trại"): keeps the timer and answers in the store; re-entry resumes at the same item. Already completed task: opens in **review mode** — items with their answers, no XP, button "Đã xong" disabled, timer hidden.

## 3. Layout (mobile-first, `max-w-md`; desktop same column)
```
┌────────────────────────────────────────┐
│ ‹ Trại        XP 20        ⏱ 04:12     │ 1 top bar: back (secondary, 44 px), XP + timer VT323 20
│ CÂU 2 / 10                             │ 2 eyebrow VT323 16; item type after it: "Câu" / "Thẻ" / "Đoạn"
│ ┌ Mầm Non ─────────────────────────┐   │ 3 RetroPanel speaker + portrait 48
│ │ [p] Choose the correct formal    │   │   prompt font-body 17/26 (English)
│ │     phrasing for requesting a    │   │
│ │     price quotation:             │   │
│ │ ▶ A  Give me the cost details…   │   │ 4 options: 56-px rows, ground-2 when selected, letter VT323 22
│ │   B  Could you please provide…   │   │
│ │   C  Send me how much this…      │   │
│ │ ───────────────────────────────  │   │ 5 feedback strip (after answer): tone line + label + explanation
│ │ ✔ Trúng!  XP +10                 │   │
│ │ "Could you please" is the polite │   │   explanation font-body 16, ink-1
│ │ request form used in business…   │   │
│ └───────────────────────────────────┘   │
├────────────────────────────────────────┤
│ [           TIẾP TỤC →            ]     │ 6 bottom bar: RetroButton primary
└────────────────────────────────────────┘
```
Item kinds and their panel:
- **Question** (`questions[i]`): prompt, options A–D as `role=radio` rows; button "Trả lời" (disabled until a choice) → feedback strip → "Tiếp tục".
- **Flashcard** (`words[i]`): front = `term` in VT323 34 centred + "Chạm để lật" caption; tap/`Lật` flips (2-frame step) to `definition` + `example` in `font-body`; two 48-px buttons side by side in the bottom bar: secondary "Học lại", primary "Nhớ rồi".
- **Passage** (`passage`): panel without portrait, `font-body` 18/30, max height 60 vh scrolling inside a scroll-frame (`line-dim` rails); button "Đọc xong →"; the questions that follow keep the passage reachable through a secondary "Xem lại đoạn" that opens it as a sheet.
- **Chest** (last item answered): `Chest` centred, items `["+10 phút", "XP {session}", …]`; when the progress response crosses the target: `"+20 HP"`, `"🔥 x{streak}"`. Button "Về trại".
Data: `task.content_json` typed per the bug; `duration_minutes`; answers kept in `answers[questionId]` (quest store) and posted with `quest.complete(id, elapsedSeconds, answers)` → `POST /api/v1/quests/progress` (`daily_seconds_spent`, `is_target_met`, `pet_health`, `streak_count`).

## 4. States
- **Loading:** top bar renders; panel shows `StateBlock loading`; no bottom button.
- **Empty (id not today's):** panel "Nhiệm vụ này không có trong hôm nay." + "Về trại".
- **Unclassifiable content (contract still broken):** panel `tone=ember` "Nhiệm vụ này đang được viết lại. Thử lại sau." + secondary "Thử lại" (refetch) + "Về trại" — never `<pre>`.
- **Answer posting error:** `tone=ember` strip above the bottom bar "Chưa ghi được tiến độ. Thử lại." — answers and elapsed seconds kept; retry resubmits.
- **Offline:** items work from the store; on complete, the post is queued and the chest shows "Tớ sẽ ghi lại khi có mạng." with the minutes only (HP/streak unknown until sync).
- **First-time (first item of day 1):** the companion's eyebrow line under "Câu 1 / N": "Cậu chọn, tớ giữ điểm." shown once.
- **Done / review:** all items answered read-only; XP hidden; button "Đã xong" disabled.
- **Reduced motion:** cursor static; flip and feedback appear at once; chest opens on the open frame; no hop/shake.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| Option row | cursor jumps, row `ground-2` | enables "Trả lời" | `answers[id]` (store) |
| "Trả lời" | button drops | row line `growth`+"Trúng!" or `ember`+"Trượt…" with the correct row marked ✔; explanation strip; XP +10 on hit; sprite `hit`/`miss` | session XP (store, not persisted) |
| "Tiếp tục" | drops | next item; eyebrow counter increments | item index (store) |
| Flashcard tap / "Lật" | 2-frame flip | back side | — |
| "Nhớ rồi" / "Học lại" | drops | next card / card requeued; XP +5 on "Nhớ rồi" | rating (store) |
| Last "Tiếp tục" | drops | `quest.complete()`; chest opens on response | `POST /quests/progress` |
| "Về trại" | drops | `/` with before/after values for the growth moment | — |
| "‹ Trại" | drops | `/`, timer keeps running per the timer-persistence plan | — |
Timer: `CountdownTimer` counts up from 00:00 (measured time), `aria-live=off`; it never disables anything. Passing `duration_minutes` shows "⏱ 10:00+" in `ink-1` — no message.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel`, `RetroButton`, `Chest`, `CompanionSprite`, `StateBlock`, `CountdownTimer`, `RetroToast` | kit | — | — |
| `ItemQuestion` | new (kit addition) | `question`, `answer?`, `revealed`; emits `select`, `submit` | options as `role=radiogroup`; feedback strip inside |
| `ItemFlashcard` | new (kit addition) | `word`, `flipped`; emits `flip`, `rate('known'|'again')` | front/back frames |
| `ItemPassage` | new (kit addition) | `text`; emits `done` | scroll-frame with rails |
| `ContentViewer` | rewritten | `content` (typed), `index`; emits per item | picks the item component; `raw` kind → the ember state, never text |
**Kit additions:** the three item components above (reason: the kit has boxes and buttons but no answerable item); `scroll-frame` style (passage rails) as a `RetroPanel` variant `frame=scroll`. Session XP lives in the quest store (`sessionXp`), reset on task entry.

## 7. Copy
Top bar: "‹ Trại" · "XP {n}" · "⏱ {mm:ss}". Eyebrows: "Câu {n} / {N}" · "Thẻ {n} / {N}" · "Đoạn văn". Buttons: "Trả lời" · "Tiếp tục →" · "Lật" · "Nhớ rồi" · "Học lại" · "Đọc xong →" · "Xem lại đoạn" · "Về trại" · "Đã xong". Feedback: "Trúng!" · "Trượt… đáp án đúng là {A}." · "XP +10" · "XP +5". Flashcard caption: "Chạm để lật". Chest: "Kho báu" (speaker) · "+10 phút" · "XP {n}" · "+20 HP" · "🔥 x{n}" · offline "Tớ sẽ ghi lại khi có mạng." First-time: "Cậu chọn, tớ giữ điểm." Errors: "Nhiệm vụ này đang được viết lại. Thử lại sau." · "Chưa ghi được tiến độ. Thử lại." · "Nhiệm vụ này không có trong hôm nay." Companion on miss (strip, once per task): "Không sao, tớ cũng hay nhầm chỗ này."

## 8. Self-critique
- Traded away: the spec's single "Gửi đáp án / Tiếp tục" button became two labels on one button — one more state for the executor, but the learner always knows what the tap does.
- XP is honest but weightless: nothing keeps it. If learners notice, the answer is a backend field, not a fake counter on the hub.
- Flashcard self-rating can be gamed ("Nhớ rồi" ×6 in ten seconds); the timer still measures real time, so the day target is not gamed. Acceptable for now; spaced repetition is its own idea.
- Executor traps: the answer key must be compared on the client only after the tap (never render `answer` into the DOM before reveal); `raw` content must reach the ember state, not an empty panel; review mode must not post; the growth-moment before/after values must be captured *before* the response mutates the pet store.
- Review checks: a 6-card vocabulary task can be finished in under a minute and posts the real elapsed seconds; the passage item counts in "n / N"; keyboard: arrow keys move the cursor, Enter answers.

## Acceptance
1. Every typed content kind (vocabulary, reading, practice) renders as items in a dialogue box; no `<pre>` and no empty flashcard front anywhere.
2. "Trả lời" is disabled until an option is chosen and enables within 100 ms of the choice.
3. Answering shows hit/miss with a label, an icon, the correct option and the explanation before the learner can continue.
4. The bottom button enables on the last answered item regardless of the timer, and the timer keeps counting without gating.
5. Completion posts `duration_seconds` = measured seconds and the answers; the chest shows minutes and, when the target is crossed, HP and streak from the response.
6. Content that fails classification shows the "đang được viết lại" state with a retry.
7. A completed task opens read-only with "Đã xong" disabled and never posts again.
8. XP appears only in the room's top bar and chest, never on the hub.
9. Under reduced motion every feedback state is still visible as a static frame.
