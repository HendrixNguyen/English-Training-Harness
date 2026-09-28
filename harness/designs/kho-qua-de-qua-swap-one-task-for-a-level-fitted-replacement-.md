# Design: Khó quá / Dễ quá — swap one task for a level-fitted one (`/learn/:id`)

**Idea:** `harness/ideas/2026-09-27-run-01/kho-qua-de-qua-swap-one-task-for-a-level-fitted-replacement-.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.4 (timer, errors, offline), `harness/designs/retro-learning-room.md` (top bar, eyebrow row, dialogue box, bottom bar, review mode, chest). This doc adds one row to that room and changes nothing inside the dialogue box.
**Siblings in the same room (same day):** `hear-it-read-aloud-…-wit.md` owns the right end of the **eyebrow row** ("Nghe" chip); `tap-a-hard-word-…-.md` owns the **passage panel** (glosses). The swap control lives in neither — it is its own row *below* the dialogue box.
**Kit:** `harness/UI-KIT.md` v2 + `harness/designs/retro-kit.md` (`RetroPanel` incl. `fog`, `RetroButton`, `CompanionSprite`, `RetroToast`, `retro-dots`).
**Spec wireframe:** frontend spec §7.3. No departure from the wireframe's structure; one row is added between the content card and the bottom button.

**Designed against:** `pages/learn/[id].vue` on `origin/harness/2026-09-25-high-task-timer-…` (timers in `quest.timers[taskId]`, keyed by task id, persisted in `aelp.timers`; `online` ref; `complete()`); `RetroButton.vue`, `RetroPanel.vue`, `useRetroToast.ts` on `origin/harness/2026-09-26-high-retro-adventure-ui-…`. The retro branch has not yet restyled `pages/learn/[id].vue`; placement below is written against `retro-learning-room.md` §3.

**Backend contract (planned by the evaluator):** `POST /api/v1/quests/{exercise_id}/swap` `{hint: "too_hard"|"too_easy"}` → 200 the task in the `GET /quests/daily` task shape (same `id`); 400 `invalid_request`, 404 `exercise_not_found`, 409 `exercise_completed`, 429 `swap_limit` (3 per local day), 429 `rate_limited`, 502 `ai_bad_output`/`ai_unavailable`, 504 `ai_timeout`. Latency 10–30 s.
**One data need this design adds:** the daily task shape gains `difficulty_hint: "too_hard" | "too_easy" | null` (additive, nullable; the column the idea already adds). The room needs it to know after a reload that a task was already swapped (one swap per task) and to count today's swaps (`daily.tasks.filter(t => t.difficulty_hint).length`). If the plan cannot add it, the fallback is a per-date `aelp.swaps` set in `localStorage` — worse (lost across devices); the plan must say which.

## 0. Research
- **Learner's job on this screen:** do this task — and when it is pitched wrong, get one that fits without losing the day.
- **The moment that earns the next minute:** tapping "Khó quá" on a task they were about to abandon, watching the companion say it is writing an easier one, and getting "Câu 1 / 8" of something they can actually answer — with the clock still counting their minutes.
- **What a lesser design gets wrong:** (1) a browser `confirm()` or a modal that takes over the room for a 30-second wait with nothing to look at — the learner closes the tab; (2) the control lives next to "Nghe" or inside the passage, so three same-day features fight for one strip and the learner taps the wrong chip; (3) a failure that eats the swap, resets progress, or leaves a blank panel — "adaptive" becomes "broken".
- **Open questions, answered:**
  1. *Where?* A quiet row under the dialogue box, above the bottom bar. It is about the whole task, not the item, so it belongs outside the item's box; it is present on every item until the chest, so a learner who struggles at question 4 can still use it.
  2. *What happens to answers already given?* The new task replaces the old one; answers and the item index for that task are cleared and the learner starts at item 1. The confirm says so when at least one item is answered. Session XP already earned is kept (it is weightless by design, `retro-learning-room.md` §0a).
  3. *What does the learner see for 10–30 s?* The old content stays on screen under `RetroPanel fog` (inert), the companion's line in the swap row says what is happening, and the bottom button is disabled. No spinner page, no modal.
  4. *Can they leave during the wait?* Yes. The request is owned by the quest store; "‹ Trại" works, and returning to the room mid-request shows the waiting state again; on success the store already holds the new task.
  5. *Cancel?* No cancel button: the server keeps generating and would still count the swap. Assumption recorded (§8).
  6. *Timer?* Untouched. `quest.timers[id]` is keyed by the task id, which the swap keeps; waiting time is study time and counts toward today.
  7. *Unclassifiable content ("đang được viết lại")?* No swap row there; that state is a data bug with its own retry, and a difficulty hint would be a lie.

## 1. The signature
**The companion rewrites the scroll in front of you.** Tap "Khó quá", say yes in the row, and the dialogue box fogs over while the row reads "Tớ đang viết bài dễ hơn…" with stepping dots; the timer keeps ticking in the top bar. The fog lifts on a new task starting at item 1, and the row settles into a small, permanent note: "Đã đổi sang bài dễ hơn." Nothing leaves the room.

## 2. Flow
Entry: any incomplete task in the room (`retro-learning-room.md` §2). Row states: **idle → confirm → swapping → swapped** (or back to idle with an error line). Exits unchanged. On success the hub shows the new title on the same tile (store replaced in place by id, no refetch). A swapped task is completed and reviewed like any other; review mode shows the note, never chips. Chest, review mode, loading, empty and unclassifiable states have no row.

## 3. Layout (mobile-first, `max-w-md`; desktop same column)
```
│ ‹ Trại        XP 20        ⏱ 04:12     │ top bar (unchanged)
│ CÂU 4 / 10                 [🔊 NGHE ]   │ eyebrow row (hear-it owns the right end)
│ ┌ Mầm Non ────────────────────────┐    │ dialogue box (unchanged; glosses live in the passage)
│ │ …                               │    │
│ └─────────────────────────────────┘    │
│ BÀI NÀY       [ Khó quá ] [ Dễ quá ]   │ SWAP ROW  mt-4, h-11, flex items-center justify-between
├────────────────────────────────────────┤
│ [           TRẢ LỜI                ]    │ bottom bar (unchanged)
```
- **Swap row (idle):** `role="group"`, `aria-label="Độ khó bài này"`. Left: caption "Bài này" VT323 16 uppercase `ink-2`. Right: two `RetroButton variant=secondary size=sm` (44 px, VT323 20, `px-3`, gap 8) — "Khó quá", "Dễ quá". Fits 328 px (360 − 2×16).
- **Confirm (in place of the row, same position):** a `RetroPanel tone=torch` (no speaker) growing the row; content: line 1 `font-body` 16 `ink-0` "Đổi lấy bài dễ hơn?" / "Đổi lấy bài khó hơn?"; line 2 `font-body` 14 `ink-1` "Tớ viết bài mới cùng chủ đề, mất khoảng nửa phút. Đồng hồ vẫn chạy." plus, only when ≥ 1 item is answered, "Cậu sẽ làm lại từ câu đầu của bài mới."; then two buttons side by side (`size=sm`, each `flex-1`): secondary "Thôi", primary "Đổi bài". Focus moves to "Đổi bài" on open; Esc = "Thôi" and returns focus to the chip that opened it.
- **Swapping:** dialogue box gets `fog` (the existing `ground-2`/60 % overlay, content `inert`); the "Nghe" chip and glosses are inside/above it and stop (see §5). Swap row becomes one line, `aria-live="polite"`: `CompanionSprite crop=face size=32` + "Tớ đang viết bài dễ hơn" / "…khó hơn" + `retro-dots`. After 15 s the line changes once to "Sắp xong rồi, cậu chờ tớ chút nhé". Bottom button disabled. Timer keeps running.
- **Swapped:** fog lifts; items restart at 1; row becomes a single caption `font-body` 14 `ink-1` "Đã đổi sang bài dễ hơn." / "Đã đổi sang bài khó hơn." — no chips (one swap per task).
- **Cap reached / offline:** chips rendered **disabled** (50 %, `aria-disabled`, never removed) with the caption replaced: "Hôm nay đã đổi 3 bài" / "Cần mạng để đổi bài".
- **Error line:** under the chips, `font-body` 14 `text-ember`, `role="alert"`, until the next tap on either chip (copy §7). Chips enabled again.

## 4. States
| State | Row shows | Box | Bottom button |
|---|---|---|---|
| Idle (incomplete task, `difficulty_hint == null`, swaps today < 3, online) | "Bài này" + two chips | normal | normal |
| Confirm | torch panel, "Thôi" / "Đổi bài" | normal, still usable | normal |
| Swapping (`quest.swapping[id]` set) | companion line + dots | `fog`, inert | disabled |
| Swapped (`difficulty_hint != null`) | "Đã đổi sang bài …" | new task from item 1 | per item |
| Cap (swaps today ≥ 3, or a `swap_limit` response this session) | chips disabled, "Hôm nay đã đổi 3 bài" | normal | normal |
| Offline (`online === false`) | chips disabled, "Cần mạng để đổi bài" | normal | normal |
| Error (recoverable) | chips + ember error line | original, un-fogged, answers intact | normal |
| Completed task / review mode / chest / loading / empty / unclassifiable | no row | — | — |
| First-time | same as idle; no hint toast (the caption and the chip words are the explanation) | | |
| Reduced motion | dots static "…"; fog on/off without transition; no sprite idle | | |

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| "Khó quá" / "Dễ quá" | chip drops; confirm panel replaces the row | — | nothing |
| "Thôi" / Esc | drops; row back to idle | focus back to the chip | nothing |
| "Đổi bài" | drops; fog on; row → swapping line; bottom disabled; speech `stop()`; open glossary popover closed | `quest.swap(id, hint)` → `POST /quests/{id}/swap` | store: `swapping[id] = hint` |
| 200 | fog lifts; focus to the eyebrow (`tabindex=-1`) | store replaces `daily.tasks[i]` by id, resets that task's `answers` and item index to 0; content re-classified via `utils/content.ts`; `RetroToast tone=growth` "Bài mới đây. Mình làm từ câu đầu nhé." | task (incl. `difficulty_hint`) in store; timer untouched |
| 409 `exercise_completed` | fog lifts | `quest.load()`; room re-renders in review mode; toast plain "Bài này đã xong rồi, không cần đổi nữa." | — |
| 404 `exercise_not_found` | fog lifts | row hidden; the page's existing ember strip "Nhiệm vụ này không còn trong hôm nay. Về trại để tải lại." | — |
| 429 `swap_limit` | fog lifts | cap state (session flag `swapCapReached`) | — |
| 429 `rate_limited` / 502 / 504 / 400 / network | fog lifts; original content and answers intact | chips + error line (§7) | nothing; the task's swap is not used |
| "‹ Trại" while swapping | drops | `/`; request continues in the store; hub tile shows the old title until it lands, then the new one; re-entry shows swapping or the new task | — |
Timer: never paused, restarted or re-anchored by the swap; `quest.startTimer` is idempotent and is not called by it. `duration_seconds` posted on completion includes the waiting time.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `SwapRow` (`components/learn/SwapRow.vue`) | new | props `hint: 'too_hard'\|'too_easy'\|null` (from task), `swapping: 'too_hard'\|'too_easy'\|null`, `capReached: boolean`, `online: boolean`, `answered: boolean`, `error: string\|null`; emits `swap(hint)`, `dismissError` | owns idle/confirm locally; everything else derived from props; no API calls |
| `quest.swap(id, hint)` (`stores/quest.ts`) | new action | returns the task; throws `ApiError` | sets/clears `swapping[id]`; on 200 splices the task at the same index, clears `answers[id]` and the item index; never touches `timers` |
| `quest.swapsToday` | new getter | — | `daily.tasks.filter(t => t.difficulty_hint).length`; cap = `≥ 3 \|\| swapCapReached` |
| `QuestTask.difficulty_hint` | type addition | `'too_hard'\|'too_easy'\|null` | optional in the type so an old API response still types |
| `pages/learn/[id].vue` | existing | — | mounts `SwapRow` between the content card and the bottom button; maps error codes to copy; passes `fog` to the item's `RetroPanel` while swapping; disables the bottom button while swapping |
| `RetroPanel` (`fog`), `RetroButton`, `CompanionSprite`, `RetroToast` | kit | — | — |

**Kit addition (lands only through the plan):** `RetroButton` prop `size: 'md' | 'sm' = 'md'` — `sm` is 44 px tall, VT323 20, `px-3`, shadow unchanged. Reason: rows of two small actions inside the column (this row, the confirm) need the kit button at 44 px; a new chip component would duplicate `RetroButton`. `SpeakButton` (hear-it) may reuse it for its label/border recipe; not required.

## 7. Copy
Row: "Bài này" · "Khó quá" · "Dễ quá". Confirm: "Đổi lấy bài dễ hơn?" · "Đổi lấy bài khó hơn?" · "Tớ viết bài mới cùng chủ đề, mất khoảng nửa phút. Đồng hồ vẫn chạy." · "Cậu sẽ làm lại từ câu đầu của bài mới." · "Thôi" · "Đổi bài". Swapping: "Tớ đang viết bài dễ hơn" · "Tớ đang viết bài khó hơn" · after 15 s "Sắp xong rồi, cậu chờ tớ chút nhé". Swapped: "Đã đổi sang bài dễ hơn." · "Đã đổi sang bài khó hơn." · toast "Bài mới đây. Mình làm từ câu đầu nhé." Cap: "Hôm nay đã đổi 3 bài". Offline: "Cần mạng để đổi bài". Errors (row line): AI (`ai_bad_output`, `ai_unavailable`, `ai_timeout`), network, `invalid_request` → "Chưa tạo được bài mới, cậu cứ tiếp tục bài này nhé." · `rate_limited` → "Tớ cần nghỉ một phút. Cậu cứ làm bài này, lát thử lại nhé." · `exercise_completed` toast → "Bài này đã xong rồi, không cần đổi nữa." · `exercise_not_found` strip → "Nhiệm vụ này không còn trong hôm nay. Về trại để tải lại." (existing string). `aria-label` of the group: "Độ khó bài này".

## 8. Self-critique
- **Traded away:** a cancel during the 10–30 s wait. The server would still generate and count it; a fake cancel that hides the result is worse. The 15-s line and the running timer are the answer to impatience.
- **Answers are discarded on swap.** Honest (the old questions are gone), and the confirm says so; a learner who swaps at question 9 of 10 loses little time because the timer kept counting.
- **The one-swap rule depends on `difficulty_hint` in the daily shape.** Without it a reload re-offers the chips and the only guard is the daily cap. The plan must add the field or the `aelp.swaps` fallback — not neither.
- **A1 + "Khó quá" / C2 + "Dễ quá"** get a same-level rewrite (server clamps). The copy still says "dễ hơn"; acceptable — a different task at the same level is still a fresh try. Not surfaced in UI.
- **Executor traps:** never call `window.confirm`; do not reset or re-anchor `quest.timers[id]`; replace the task at the same array index (hub order and `sortedTasks` rely on it); reset answers/item index only on 200, never on error; `fog` must make the box `inert` (no answering an item that is about to vanish); the bottom button must be disabled while swapping so `complete()` cannot post the old task mid-swap; the swap promise lives in the store, not the page, or leaving mid-request loses the result; keep the row out of the eyebrow row and the passage panel; the `swap_limit` flag is session-only (the next day's `quest.load()` recomputes from data).
- **Review checks:** stub the endpoint with a 20-s delay → timer keeps ticking, box fogged, bottom disabled, "‹ Trại" and back shows the waiting line; stub 502 → original content and answers still there, chips back with the ember line; after a 200 and a reload the row reads "Đã đổi sang bài …"; a completed task shows no row.

## Acceptance
1. On an incomplete task, every item of the room shows a "Bài này" row with "Khó quá" and "Dễ quá" (44 px, secondary) directly below the dialogue box and above the bottom button — not in the eyebrow row and not inside the passage panel; completed tasks, review mode, the chest, and the loading, empty and unclassifiable states show no row.
2. Tapping a chip replaces the row within 100 ms with an in-page torch panel ("Đổi lấy bài dễ hơn?" / "…khó hơn?", "Thôi", "Đổi bài"); `window.confirm` is never called; "Thôi" or Esc restores the row with no request sent; the "làm lại từ câu đầu" line appears only when at least one item is answered.
3. "Đổi bài" sends `POST /api/v1/quests/{id}/swap` with the chosen `hint`; within 100 ms the dialogue box is fogged and inert, the row shows the companion line with stepping dots, and the bottom button is disabled; after 15 s the line changes to "Sắp xong rồi, cậu chờ tớ chút nhé".
4. The countdown keeps running during and after the swap and is never reset; `quest.timers[id]` is unchanged by the swap, and the `duration_seconds` posted on completion includes the waiting time.
5. On 200 the task is replaced in place in `daily.tasks` (same id, same index), the room restarts at item 1 with the new content, that task's answers are cleared, a growth toast appears, and the row becomes "Đã đổi sang bài dễ hơn." / "…khó hơn." with no chips — also after a reload (from `difficulty_hint`).
6. On `ai_bad_output`, `ai_unavailable`, `ai_timeout`, `invalid_request` or a network error, the original content and its answers stay, the fog lifts, the chips return enabled with "Chưa tạo được bài mới, cậu cứ tiếp tục bài này nhé."; on `rate_limited` the line is "Tớ cần nghỉ một phút. Cậu cứ làm bài này, lát thử lại nhé."
7. On `swap_limit`, or when three of today's tasks have a `difficulty_hint`, the chips on every unswapped task are disabled (not removed) with "Hôm nay đã đổi 3 bài"; offline they are disabled with "Cần mạng để đổi bài".
8. On `exercise_completed` the store reloads and the room shows review mode with "Bài này đã xong rồi, không cần đổi nữa."; on `exercise_not_found` the row disappears and the existing "Nhiệm vụ này không còn trong hôm nay…" strip shows.
9. Leaving with "‹ Trại" during a swap does not lose it: re-entering shows the waiting state until the response lands, and the hub tile shows the new title after it does.
10. Under `prefers-reduced-motion: reduce` the dots are static and the fog toggles without transition; every state remains distinguishable by text.
