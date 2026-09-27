# Design: Catch-up — "Học bù" on any earlier day

**Idea:** `harness/ideas/2026-09-26-run-01/missed-days-are-not-lost-catch-up-on-any-earlier-day-s-tasks.md`
**Inherits:** `harness/designs/frontend-shell.md` (page-state convention), `roadmap-tree.md` (day states, `GET /api/v1/roadmap`, the rule that a star is earned only by `is_target_met`), `retro-roadmap.md` (world map, `DayDetail`), `retro-hub.md` (camp screen, bottom bar, `QuestNode`), `growth-moment.md` (the two seconds after a task).
**Kit:** `harness/UI-KIT.md` v2. No kit additions.
**Spec wireframe:** frontend spec §7.2 (hub) and §7.4 (roadmap). Kept: "🔒 Chưa mở khóa" stays on future days only. Departure: §7.4 says only today is playable; past days become playable through the hub, never from the map itself.
**Wire contract (evaluator's):** `GET /api/v1/quests/daily?day=N` (1 ≤ N ≤ today's `day_number`, else `400 invalid_request`) → the usual body with `day_number: N`, additive `is_today` (false for N < today), and **today's** `date`, `accumulated_seconds`, `is_target_met`. `POST /api/v1/quests/progress` accepts a past day's exercise; its minutes count toward today. **One contract addition this design needs:** `GET /api/v1/roadmap` days gain additive `completed_tasks` (0–3, count of that day's exercises with `is_completed`) so the map can say "Học tiếp" vs done without loading each day. Fallback when the field is absent: §4.1.

**Which screens it lands on.** Whatever hub and roadmap are on `main` at execution. The action sits in the *expanded day panel* in both generations — v1 `RoadmapNode` task panel (where today's "Học ngay →" is) and retro `DayDetail` — and the catch-up view is a mode of `pages/index.vue` whichever look it has. Names below give both (v1 / retro).

## 0. Research
- **Learner's job:** "I missed Tuesday — let me do Tuesday's lessons now, and have it count today."
- **The moment that earns the next minute:** opening the missed day on the map, seeing its three tasks still waiting (not greyed out, not gone), and watching *today's* day bar move when the first one is done. The lesson was not thrown away and the effort is not wasted on a day that is already judged.
- **What today does wrong:** `quests/repo.go` rejects any exercise not on today's `day_number`; roadmap-tree shows a missed day as "○ Bỏ lỡ" with a task list and no way in. A skip costs the plant −30 **and** the lesson.
- **How a lesser design fails:** (1) a catch-up that looks identical to today, so the learner loses track of which day they are on and thinks today's quest is done; (2) rewriting the past — turning ○ into ★ — which contradicts the plant's verdict and the streak; (3) a catch-up that returns the learner to today after each task, so doing a whole missed day is three trips through the map.
- **Open questions, answered:**
  - *Where does the learner start a catch-up?* The expanded past-day panel on `/roadmap` (one tap to expand, one to go). Plus, on the hub, a single quiet row "Học bù ngày N" for the most recent past day with open tasks, shown only when today's target is met (the idea's no-roadmap fallback, and the "finished early, what next" case). Not shown before today is met: today comes first.
  - *Is the catch-up a new page?* No — `/?day=N` on the hub. It keeps the companion, status bar and day bar (today's), and the back button returns to the map naturally. `?day` equal to today, or missing, is the normal hub.
  - *After a catch-up task?* Back to `/?day=N` with the growth moment (the minutes fed today's plant). When all three are done, the view says so and its only button is "Về hôm nay".
  - *Does the past day's star change?* Never. `is_target_met` and `minutes_spent` of day N are unchanged by a catch-up (minutes go to today's date); only `completed_tasks` rises. The status line gains a second caption, the glyph does not.
  - *Future days?* Unchanged: locked, no action, and the API refuses them.

## 1. The signature
**The day banner in torch.** In catch-up mode the hub's "Phòng hôm nay" eyebrow stays above today's day bar, and a `RetroPanel tone=torch speaker="Học bù"` sits between the companion and the path: "Ngày 3 · T5 24/9" and one line "Phút học bù được tính vào hôm nay." Torch is the kit's "reward/today" colour, so the banner reads as a detour on the road, not an error. The path under it is day 3's tasks with their real done state. The learner always sees two facts at once: which day's lessons they are doing (banner) and which day's bar they are filling (today).

## 2. Flow
```
/roadmap ─ tap past node (not met) ─> panel expands ─ "Học bù →" / "Học tiếp →" ─> /?day=N  (catch-up view)
/  (today met) ─ row "Học bù ngày N" ────────────────────────────────────────────────> /?day=N
/?day=N ─ tile / bottom "Học bù nhiệm vụ k →" ─> /learn/:id?day=N ─ complete ─> /?day=N (growth moment)
/?day=N ─ "Về hôm nay" ─> /        /learn/:id?day=N ─ "‹ Ngày N" ─> /?day=N
```
Returning to `/roadmap` after a catch-up: the day's glyph and first status line are unchanged; the second caption shows the new `completed_tasks` (the map revalidates on mount). Returning to `/`: today's day bar, HP and streak reflect the minutes, exactly as after a today task.

## 3. Layout (mobile-first, `max-w-md`; desktop same column)

### 3.1 Roadmap — expanded past-day panel
```
│ ○ Ngày 3 · T5 24/9                   │  glyph unchanged (○ missed / ◐ partial)
│   Asking follow-up questions          │
│   Bỏ lỡ · Đã học bù 1/3 nhiệm vụ      │  2nd caption only when completed_tasks > 0
│  ┌──────────────────────────────────┐ │
│  │ 📖 Từ vựng   Question words  Đã xong│ │  per-task state is NOT known here → titles only (as today)
│  │ 📜 Đọc hiểu  A lunch chat          │ │
│  │ 🗡 Thực hành Ask three questions   │ │
│  │ [ Học tiếp →            ] secondary│ │  RetroButton secondary / AppButton secondary
│  └──────────────────────────────────┘ │
```
Button label by state (§4.1). It is `secondary` — the map's only primary is today's "Học ngay →"/"Vào". Placement: the same slot as today's button, last in the panel. Target-met past days and locked days get no button.

### 3.2 Hub — catch-up view (`/?day=N`)
```
│ [av] Hendrix  ‹ Về hôm nay   [🔥 x5]  │ 1 status bar; "‹ Về hôm nay" replaces nothing, sits after the name (VT323 20, line-lit)
│ ┌ Mầm Non ──────────────────────────┐ │ 2 companion panel, unchanged data; catch-up line (§7)
│ │ [sprite]  HP 70/100  "Học bù …"    │ │
│ └────────────────────────────────────┘ │
│ PHÒNG HÔM NAY               20/30      │ 3 today's DayBar — accumulated_seconds (today) — unchanged
│ [████][████][░░░░]                     │
│ ┌ Học bù ────────────────────────────┐ │ 4 RetroPanel tone=torch: "Ngày 3 · T5 24/9" (VT323 22)
│ │ Ngày 3 · T5 24/9                    │ │   + "Phút học bù được tính vào hôm nay." (body 14 ink-1)
│ │ Phút học bù được tính vào hôm nay.  │ │   + day title when the roadmap store has it (body 17)
│ └────────────────────────────────────┘ │
│ [📖] Question words         Đã xong    │ 5 path of day N: QuestNode ×3, dayQuests.tasks, real is_completed
│ ▶[📜] A lunch chat          Vào        │
│ [🗡] Ask three questions    Khoá       │
├────────────────────────────────────────┤
│ [   HỌC BÙ NHIỆM VỤ 2 →   ]            │ 6 bottom bar primary; "Về hôm nay" when all 3 done
```
- Date in the banner: from the roadmap store's day `date` if loaded, else "Ngày 3" alone (the daily body's `date` is today's — never print it here).
- Tile states: the hub's `rowState()` fed with the *viewed* day's `nextTaskId`; "open" when today's `is_target_met` (same rule as today).
- v1 hub: banner = `AppCard` with a torch left border (`border-l-4 border-streak`), header link in `AppHeader`'s right slot; same content.

### 3.3 Hub — normal view addition (today met only)
Under the path, one `QuestNode`-styled row (v1: text link row) "Học bù ngày N · {k}/3 nhiệm vụ còn lại" → `/?day=N`, N = the latest past day with `is_target_met=false` and `completed_tasks < 3` from the roadmap store. Hidden when the roadmap store has no outline, when no such day exists, or before today is met.

### 3.4 Learning room with `?day=N`
Back link "‹ Ngày N" → `/?day=N`; a torch eyebrow "Học bù · Ngày N" above the task title; everything else identical. Completion navigates to `/?day=N` (replace) instead of `/`.

## 4. States

### 4.1 Past-day action (roadmap panel)
| Day | Condition | Button | 2nd caption |
|---|---|---|---|
| met | `is_target_met` | none | none |
| untouched | `completed_tasks == 0` and `minutes_spent == 0` | "Học bù →" | none |
| started | `completed_tasks` 1–2, or `minutes_spent > 0` with `completed_tasks` 0 | "Học tiếp →" | "Đã học bù {c}/3 nhiệm vụ" when c > 0 |
| all done, not met | `completed_tasks == 3` | none | "Đã xong 3/3 nhiệm vụ" |
| today / locked | as roadmap-tree | unchanged | — |
Fallback if `completed_tasks` is absent: "Học bù →" when `minutes_spent == 0`, else "Học tiếp →", no second caption.

### 4.2 Catch-up view (`/?day=N`)
- **Loading:** status bar, companion and today's day bar from cache as normal; banner renders at once with "Ngày N"; the path shows three `ground-2` tiles; **no bottom button**.
- **Error `400 invalid_request`** (future day, 0, non-integer): banner replaced by an ember panel "Ngày này chưa mở." + primary "Về hôm nay". No retry.
- **Error network/5xx:** ember panel "Không tải được ngày N. Thử lại." with secondary "Thử lại" (refetch day only) and the header "‹ Về hôm nay" still present.
- **Empty (`404 no_active_roadmap`):** same as the hub's empty state; catch-up mode ends.
- **Offline:** a previously loaded day renders from the SW cache (NetworkFirst on the full URL) with the hub's one offline toast; an unvisited day shows "Đang ngoại tuyến — chưa có bài của ngày N." + "Về hôm nay".
- **First-time on the view:** nothing extra; the banner line is the explanation.
- **Done (all 3 tasks of N complete):** banner caption becomes "Đã học bù xong ngày N."; tiles all `done`; bottom primary "Về hôm nay". The companion line switches to the done-catch-up line (§7). No chest, no star — the chest already played in the room.
- **Wilted:** the hub's wilted band stays on top; catch-up tasks remain playable (they count toward today — same as today's path).
- **`?day` = today:** `navigateTo('/', {replace:true})`.
- **Reduced motion:** as the hub; the growth moment shows its final frame.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Store / endpoint |
|---|---|---|---|
| "Học bù →" / "Học tiếp →" (map) | button drop | `/?day=N` | — |
| Hub row "Học bù ngày N" | tile inset | `/?day=N` | — |
| Catch-up tile / bottom button | inset / drop | `/learn/:id?day=N` | — |
| Complete in room | button loading | `POST /quests/progress`, then `/?day=N` + growth moment | `quest.complete()` updates `daily.accumulated_seconds/is_target_met` **and** marks the task done in whichever of `daily`/`dayQuests` holds it |
| "‹ Về hôm nay" / done-state primary | drop | `/`; `quest.clearDay()` | — |
| Retry (day error) | drop | `quest.loadDay(N)` | `GET /quests/daily?day=N` |
Store (`stores/quest.ts`): add `viewingDay: number|null`, `dayQuests: DailyQuests|null`, `dayLoading`, `dayError`, `loadDay(n)`, `clearDay()`, getter `viewSortedTasks`/`viewNextTaskId`. `daily` is never overwritten by a past day; when `loadDay` returns, copy its (today's) `accumulated_seconds`/`is_target_met` into `daily` if present. `taskById` searches `daily` then `dayQuests`; the room, when `?day=N` is set and the task is not found, calls `loadDay(N)` first. Timers stay keyed by task id. Roadmap store: `/roadmap` renders its cache and refetches on every mount (so `completed_tasks` is fresh after a catch-up).

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel` / `AppCard` | kit | `tone=torch`, `speaker="Học bù"` | the banner; `ember` for day errors |
| `QuestNode` / `QuestRow` | kit | as hub | fed from `dayQuests` in catch-up mode |
| `DayBar` / `SegmentedProgress` | kit | today's values only | never bound to `dayQuests` |
| `RetroButton` / `AppButton` | kit | `secondary` on the map, `primary` in bottom bar | — |
| `DayDetail` / `RoadmapNode` panel | from retro-roadmap / roadmap-tree | new prop-driven action slot: `action: {label, to} \| null` | computed by a pure `catchUpAction(day, todayNumber)` in `utils/roadmap.ts` (unit-tested against §4.1) |
| `StateBlock`, `RetroToast` | kit | — | — |
Kit additions: none.

## 7. Copy
Map: "Học bù →" · "Học tiếp →" · "Đã học bù {c}/3 nhiệm vụ" · "Đã xong 3/3 nhiệm vụ".
Hub normal row: "Học bù ngày {N} · {k}/3 nhiệm vụ còn lại".
Catch-up view: speaker "Học bù" · "Ngày {N} · {weekday} {d/m}" (or "Ngày {N}") · "Phút học bù được tính vào hôm nay." · done "Đã học bù xong ngày {N}." · header "‹ Về hôm nay" · bottom "Học bù nhiệm vụ {k} →" · "Về hôm nay".
Companion (catch-up): "Học bù ngày {N} hả? Phút nào cũng tưới cho tớ hôm nay đấy." · done "Ngày {N} xong rồi. Về phòng hôm nay với tớ nhé?" (when today is met: "Ngày {N} xong rồi. Hôm nay tớ đủ nước rồi, cảm ơn cậu.").
Room: eyebrow "Học bù · Ngày {N}" · back "‹ Ngày {N}" · not-found error "Nhiệm vụ này không có trong ngày {N}. Quay lại để tải lại."
Errors: "Ngày này chưa mở." · "Không tải được ngày {N}. Thử lại." · "Đang ngoại tuyến — chưa có bài của ngày {N}."

## 8. Self-critique
- **Traded away:** playing a past day directly on the map. The hub detour costs one screen but keeps the companion and today's bar in view — the whole point is "this feeds today".
- **The contract addition** (`completed_tasks`) is small but real; without it the map cannot tell "Học tiếp" from done and would show "Học bù" on a day already caught up. The fallback in §4.1 keeps the screen honest-enough if the plan drops it; the reviewer should check which one shipped.
- **Executor traps:** (1) printing the daily body's `date` as the catch-up day — it is today's; (2) binding `DayBar` to `dayQuests` — must stay `daily`; (3) overwriting `daily` in `loadDay`; (4) `complete()` only patching `daily.tasks`, so the catch-up tile never ticks; (5) navigating to `/` after a catch-up task, which drops the learner out of the day; (6) letting any catch-up write reach day N's star — the map glyph must be identical before and after; (7) deciding past/future by the client clock instead of `day_number`.
- **Risk:** a learner doing three catch-up days in one sitting can meet today's 30 minutes without touching today's lessons. Acceptable: the spec judges minutes, not which lessons; today's tasks stay open and the hub row only appears once today is met, so the default path is still today first.

## Acceptance
1. On `/roadmap`, an expanded past day with `is_target_met=false` shows "Học bù →" (untouched) or "Học tiếp →" (started) per §4.1; target-met past days, days with 3/3 tasks done, today and future days show no catch-up button.
2. Tapping the button opens `/?day=N`, which requests `GET /api/v1/quests/daily?day=N` and shows a torch "Học bù" banner reading "Ngày N" (with the date from the roadmap outline when loaded, never the body's `date`).
3. In the catch-up view, the path lists day N's three tasks with their real `is_completed` state, while the day bar still shows today's `accumulated_seconds` and today's met state.
4. A "‹ Về hôm nay" link is always visible in the catch-up view (loading, error and done included) and returns to `/` with the normal today path.
5. Opening a catch-up task goes to `/learn/:id?day=N` with the eyebrow "Học bù · Ngày N"; completing it posts progress, returns to `/?day=N`, ticks that tile, moves today's day bar and plays the growth moment once.
6. When all three tasks of day N are done, the banner reads "Đã học bù xong ngày N." and the bottom button is "Về hôm nay".
7. After any catch-up, day N's glyph and first status line on `/roadmap` are unchanged (no ★ added, no streak change); only the second caption "Đã học bù c/3 nhiệm vụ" updates, fetched fresh on the next visit.
8. `/?day=` with a future day, 0 or a non-integer shows "Ngày này chưa mở." and a "Về hôm nay" button; `/?day=<today>` redirects to `/`; network errors show a retry that refetches only that day.
9. The bottom button is absent while the catch-up day loads; offline, a previously loaded day renders from cache with one toast and an unvisited day shows the offline message.
10. On the normal hub, the row "Học bù ngày N · k/3 nhiệm vụ còn lại" appears only after today's target is met and only when such a past day exists; under reduced motion no animation runs in any of these views.
